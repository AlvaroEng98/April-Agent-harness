package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// featureListPath es la ruta del archivo de estado del backlog, relativa al
// cwd — mismo archivo que lee status.go, pero `april feature set-status`
// es la única vía de escritura autorizada sobre él (ver
// docs/architecture.md / CLAUDE.md, "única vía de escritura válida").
const featureListPath = "feature_list.json"

// Vocabulario de veredictos que reconoce set-status para done — mismo que
// usa reviewer_agent (ver .claude/agents/reviewer_agent.md). Es el
// mecanismo INTERINO confirmado por el humano el 27/08/2026: hasta que
// exista el ledger real de veredictos (features 5/6, todavía pending), el
// veredicto se pasa a mano por flag en la misma invocación, no se lee de
// ningún registro. CHANGES_REQUESTED se reconoce explícitamente para dar
// un error claro ("no habilita done") en vez de "valor desconocido".
const (
	verdictApproved              = "APPROVED"
	verdictApprovedWithObjection = "APPROVED_WITH_OBJECTION"
	verdictChangesRequested      = "CHANGES_REQUESTED"
)

// ErrInvalidTransition señala una transición fuera del grafo de estados
// válido — ver validTransition para el grafo completo.
var ErrInvalidTransition = errors.New("transición inválida")

// ErrConcurrentInProgress señala que ya existe otra feature in_progress
// (regla one_feature_at_a_time de feature_list.json.rules).
var ErrConcurrentInProgress = errors.New("ya hay otra feature in_progress")

// ErrMissingVerdict señala que done se pidió sin --verdict, o con un valor
// fuera del vocabulario que habilita cierre (APPROVED/APPROVED_WITH_OBJECTION).
var ErrMissingVerdict = errors.New("falta --verdict válido para done")

// rawFeatureListDoc es feature_list.json visto como documento
// parcialmente opaco: rules y cada feature que set-status no toca se
// preservan byte a byte (json.RawMessage). A diferencia de
// featureListFile (status.go, de solo lectura, que puede darse el lujo de
// ignorar campos que no usa), set-status SÍ reescribe el archivo, así que
// nunca decodifica el documento completo a una struct limitada — eso
// perdería silenciosamente campos como "description" o "acceptance" de
// cada feature.
type rawFeatureListDoc struct {
	Rules    json.RawMessage   `json:"rules"`
	Features []json.RawMessage `json:"features"`
}

// featureSummary es el subconjunto de una feature que set-status necesita
// leer de cada entrada para validar la transición: id, status actual y
// sdd (decide si spec_ready es un estado alcanzable para esa feature).
type featureSummary struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	SDD    bool   `json:"sdd"`
	Status string `json:"status"`
}

// setStatusResult resume la transición aplicada, para el mensaje de salida
// de runSetStatus.
type setStatusResult struct {
	FeatureID int
	From      string
	To        string
	Verdict   string
}

// validTransition decide si moverse de from a to es una arista del grafo
// pending → spec_ready → in_progress → done (+ blocked).
//
// Decisión de diseño (documentada acá porque el grafo no distingue sdd en
// la descripción original de la tarea, y hacía falta resolverlo):
// spec_ready solo tiene sentido para una feature sdd:true (es el estado
// que certifica "hay spec aprobada" — ver derivePhase en status.go, que ya
// distingue sdd:true/false del mismo modo). Para sdd:false el grafo
// colapsa a pending → in_progress → done: pedir spec_ready sobre una
// feature sdd:false se rechaza, igual que cualquier otra arista fuera del
// grafo.
//
// blocked es alcanzable desde cualquier estado "abierto" (pending,
// spec_ready si sdd, in_progress) y desde blocked se puede volver a
// cualquiera de esos mismos estados abiertos — pero nunca directo a done:
// el cierre siempre pasa por in_progress + --verdict (ver setStatus), así
// que blocked no puede saltárselo. done es terminal: no tiene aristas de
// salida (ni siquiera a blocked).
func validTransition(sdd bool, from, to string) bool {
	if from == to {
		return false
	}

	switch from {
	case "pending":
		if sdd {
			return to == "spec_ready" || to == "blocked"
		}
		return to == "in_progress" || to == "blocked"
	case "spec_ready":
		if !sdd {
			return false
		}
		return to == "in_progress" || to == "blocked"
	case "in_progress":
		return to == "done" || to == "blocked"
	case "blocked":
		if sdd {
			return to == "pending" || to == "spec_ready" || to == "in_progress"
		}
		return to == "pending" || to == "in_progress"
	case "done":
		return false
	default:
		return false
	}
}

// computeSetStatus decide, puro (sin tocar disco), el resultado de aplicar
// `set-status <id> <estado> [--verdict <valor>]` sobre docData (el
// contenido actual de feature_list.json): el JSON resultante ya listo
// para escribir y un resumen de la transición. No escribe nada — eso lo
// hace setStatus (vía writeFileAtomic). Si la transición es inválida, si
// hay conflicto de one_feature_at_a_time, o si done se pide sin --verdict
// válido, devuelve error sin construir ninguna salida utilizable.
func computeSetStatus(docData []byte, id int, newStatus, verdict string) ([]byte, setStatusResult, error) {
	var doc rawFeatureListDoc
	if err := json.Unmarshal(docData, &doc); err != nil {
		return nil, setStatusResult{}, fmt.Errorf("parseando %s: %w", featureListPath, err)
	}

	var rules featureListRules
	if err := json.Unmarshal(doc.Rules, &rules); err != nil {
		return nil, setStatusResult{}, fmt.Errorf("parseando rules de %s: %w", featureListPath, err)
	}
	validStatus := make(map[string]bool, len(rules.ValidStatus))
	for _, s := range rules.ValidStatus {
		validStatus[s] = true
	}
	if !validStatus[newStatus] {
		return nil, setStatusResult{}, fmt.Errorf("estado destino %q fuera de rules.valid_status", newStatus)
	}

	targetIdx := -1
	var target featureSummary
	inProgressOther := -1
	for i, raw := range doc.Features {
		var f featureSummary
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, setStatusResult{}, fmt.Errorf("parseando feature en posición %d de %s: %w", i, featureListPath, err)
		}
		if f.ID == id {
			targetIdx = i
			target = f
			continue
		}
		if f.Status == "in_progress" {
			inProgressOther = f.ID
		}
	}
	if targetIdx == -1 {
		return nil, setStatusResult{}, fmt.Errorf("%w: id %d", ErrFeatureNotFound, id)
	}

	if !validStatus[target.Status] {
		return nil, setStatusResult{}, fmt.Errorf("estado actual %q de la feature %d fuera de rules.valid_status, no se puede calcular la transición", target.Status, id)
	}

	if !validTransition(target.SDD, target.Status, newStatus) {
		return nil, setStatusResult{}, fmt.Errorf("%w: %s -> %s no es una arista del grafo pending → spec_ready → in_progress → done (+ blocked) para la feature %d", ErrInvalidTransition, target.Status, newStatus, id)
	}

	if newStatus == "in_progress" && inProgressOther != -1 {
		return nil, setStatusResult{}, fmt.Errorf("%w: feature %d ya está in_progress", ErrConcurrentInProgress, inProgressOther)
	}

	if newStatus == "done" {
		switch verdict {
		case verdictApproved, verdictApprovedWithObjection:
			// habilita done
		case verdictChangesRequested:
			return nil, setStatusResult{}, fmt.Errorf("%w: CHANGES_REQUESTED no habilita done", ErrMissingVerdict)
		case "":
			return nil, setStatusResult{}, fmt.Errorf("%w: falta la flag --verdict", ErrMissingVerdict)
		default:
			return nil, setStatusResult{}, fmt.Errorf("%w: valor %q fuera del vocabulario (APPROVED, APPROVED_WITH_OBJECTION)", ErrMissingVerdict, verdict)
		}
	}

	// Construye la feature actualizada como mapa genérico: es el único
	// punto del documento que se decodifica sin conservar bytes originales,
	// porque es justo lo que hay que modificar (status, y reviewVerdict si
	// corresponde). El resto de campos de esta feature (title, description,
	// acceptance, etc.) sobreviven porque vienen del mismo Unmarshal.
	var targetObj map[string]interface{}
	if err := json.Unmarshal(doc.Features[targetIdx], &targetObj); err != nil {
		return nil, setStatusResult{}, fmt.Errorf("parseando feature %d de %s: %w", id, featureListPath, err)
	}
	targetObj["status"] = newStatus
	if newStatus == "done" {
		targetObj["reviewVerdict"] = verdict
	}
	updatedRaw, err := json.Marshal(targetObj)
	if err != nil {
		return nil, setStatusResult{}, fmt.Errorf("serializando feature %d actualizada: %w", id, err)
	}
	doc.Features[targetIdx] = updatedRaw

	compact, err := json.Marshal(doc)
	if err != nil {
		return nil, setStatusResult{}, fmt.Errorf("serializando %s: %w", featureListPath, err)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, compact, "", "  "); err != nil {
		return nil, setStatusResult{}, fmt.Errorf("formateando %s: %w", featureListPath, err)
	}
	pretty.WriteByte('\n')

	result := setStatusResult{FeatureID: id, From: target.Status, To: newStatus}
	if newStatus == "done" {
		result.Verdict = verdict
	}

	return pretty.Bytes(), result, nil
}

// writeFileAtomic escribe data en path de forma atómica: primero a un
// archivo temporal en el mismo directorio (mismo filesystem, para que
// os.Rename sea atómico) y luego renombra sobre el destino final. Es el
// único punto de escritura de feature_list.json — a diferencia de
// scaffold.go/applyPlan, que escribe plantillas con os.WriteFile directo
// porque no son el estado crítico de progreso del proyecto (ver
// docs/conventions.md: "escritura atómica obligatoria para estado
// crítico").
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("creando archivo temporal para escritura atómica de %s: %w", path, err)
	}
	tmpPath := tmp.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("escribiendo archivo temporal para %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cerrando archivo temporal para %s: %w", path, err)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("ajustando permisos del archivo temporal para %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renombrando archivo temporal sobre %s: %w", path, err)
	}
	success = true
	return nil
}

// setStatus valida y aplica `april feature set-status <id> <estado>
// [--verdict <valor>]` sobre el feature_list.json real (cwd), con
// escritura atómica. No toca el archivo si la transición es inválida —
// computeSetStatus falla antes de que setStatus llegue a escribir nada.
func setStatus(id int, newStatus, verdict string) (setStatusResult, error) {
	data, err := os.ReadFile(featureListPath)
	if err != nil {
		return setStatusResult{}, fmt.Errorf("leyendo %s: %w", featureListPath, err)
	}

	out, result, err := computeSetStatus(data, id, newStatus, verdict)
	if err != nil {
		return setStatusResult{}, err
	}

	if err := writeFileAtomic(featureListPath, out, 0644); err != nil {
		return setStatusResult{}, err
	}

	return result, nil
}

// runSetStatus contiene la lógica de `april feature set-status <id>
// <estado> [--verdict <valor>]`: parseo de args, invocación de setStatus,
// mensajes de error explícitos y decisión de exit code. Separado de
// cmdFeature (que sí llama a os.Exit) para poder testearlo in-process, sin
// terminar el proceso de test — mismo patrón que runStatus/cmdStatus.
func runSetStatus(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: uso: april feature set-status <id> <estado> [--verdict <valor>]")
		return 1
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: id inválido %q (debe ser el id numérico de una feature)\n", args[0])
		return 1
	}
	newStatus := args[1]

	verdict := ""
	for i := 2; i < len(args); i++ {
		if args[i] == "--verdict" {
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --verdict requiere un valor")
				return 1
			}
			verdict = args[i+1]
			i++
		}
	}

	result, err := setStatus(id, newStatus, verdict)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Printf("feature %d: %s -> %s\n", result.FeatureID, result.From, result.To)
	if result.Verdict != "" {
		fmt.Printf("  reviewVerdict: %s\n", result.Verdict)
	}
	return 0
}

// cmdFeature es el entry point del CLI para `april feature <subcomando>
// ...`. Hoy el único subcomando es set-status.
func cmdFeature() {
	args := os.Args[2:]
	if len(args) == 0 || args[0] != "set-status" {
		fmt.Fprintln(os.Stderr, "Error: uso: april feature set-status <id> <estado> [--verdict <valor>]")
		os.Exit(1)
	}
	os.Exit(runSetStatus(args[1:]))
}
