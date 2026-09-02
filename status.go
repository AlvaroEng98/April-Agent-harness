package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Valores posibles de statusReport.Phase — ver la tabla de derivación en
// specs/april_status_arbiter/spec.md, sección "Derivación de phase".
const (
	phaseGrill          = "grill"
	phaseSpec           = "spec"
	phaseTickets        = "tickets"
	phaseImplementation = "implementation"
	phaseReview         = "review"
	phaseClosed         = "closed"
)

// bootstrapFeatureName identifica, por convención de nombre, la única
// feature prevista para la Fase Grill — no hay otra señal en disco que la
// distinga de una feature sdd:false cualquiera (ver spec, "Derivación de
// phase").
const bootstrapFeatureName = "bootstrap_project"

// gwtOptOutMarker es el marcador de opt-out explícito que una spec puede
// incluir en cualquier parte del archivo para declarar que ninguna de sus
// historias de usuario tiene rama de comportamiento verificable — su sola
// presencia basta, incluso si además hay bloques Given/When/Then reales
// (ver specs/spec_gwt_mechanical_check/spec.md, "Marcador de opt-out").
const gwtOptOutMarker = "<!-- gwt: no aplica -->"

// ErrFeatureNotFound se devuelve cuando se pide el status de un id que no
// existe en feature_list.json — es un error de invocación, no un
// blockedReason (ver spec, "Resolución de id explícito").
var ErrFeatureNotFound = errors.New("feature no encontrada en feature_list.json")

// featureListRules es el subconjunto de feature_list.json.rules que
// computeStatusFromFS necesita: el vocabulario válido de status.
type featureListRules struct {
	ValidStatus []string `json:"valid_status"`
}

// featureEntry es el subconjunto de cada feature de feature_list.json
// relevante para el cálculo de status.
type featureEntry struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Title  string `json:"title"`
	SDD    bool   `json:"sdd"`
	Status string `json:"status"`
}

// featureListFile es el feature_list.json completo, tal como lo necesita
// computeStatusFromFS.
type featureListFile struct {
	Rules    featureListRules `json:"rules"`
	Features []featureEntry   `json:"features"`
}

// ticketInfo es la información de un archivo de ticket
// (specs/<name>/tickets/<NN>-<slug>.md) relevante para el cálculo de
// status: su número de orden, nombre de archivo, el valor de su campo
// Status, y su Blocked by ya interpretado (ver parseBlockedBy).
type ticketInfo struct {
	NN       string
	Filename string
	Status   string

	// BlockedByRaw es el texto crudo del campo "**Blocked by:**", para
	// mensajes de error cuando no se puede interpretar.
	BlockedByRaw string
	// BlockedBy son los NN (dos dígitos) de los tickets que bloquean a
	// este, ya resueltos y validados contra los tickets existentes de la
	// misma feature. Vacío si el ticket no tiene bloqueadores.
	BlockedBy []string
	// BlockedByValid es false si el texto de Blocked by no se pudo
	// interpretar (ni números de dos dígitos ni "none") o si alguno de los
	// números no corresponde a ningún ticket existente de la feature — en
	// ambos casos se reporta en blockedReasons (ver
	// computeBlockedReasons/ticketBlockedByReasons) y el ticket queda
	// excluido de frontier y del grafo de ciclos.
	BlockedByValid bool
}

// statusArtifactPaths son las rutas relevantes según la fase de la feature
// target: featureList siempre está presente; spec solo si sdd:true;
// ticketsDir/tickets solo si hay al menos un archivo de ticket en disco.
type statusArtifactPaths struct {
	FeatureList string   `json:"featureList"`
	Spec        string   `json:"spec,omitempty"`
	TicketsDir  string   `json:"ticketsDir,omitempty"`
	Tickets     []string `json:"tickets,omitempty"`
}

// statusReport es la salida completa de `april status`: los cinco campos
// que calcula computeStatusFromFS, listos para serializar a JSON o
// formatear como texto plano.
type statusReport struct {
	Phase           string              `json:"phase"`
	NextRecommended string              `json:"nextRecommended"`
	BlockedReasons  []string            `json:"blockedReasons"`
	Frontier        []string            `json:"frontier"`
	ArtifactPaths   statusArtifactPaths `json:"artifactPaths"`
}

// statusFieldRe extrae el valor del campo "**Status:**" de un archivo de
// ticket (plantilla de ticket_writer, ver .claude/agents/ticket_writer.md).
var statusFieldRe = regexp.MustCompile(`(?mi)^\*\*Status:\*\*\s*(.+)$`)

// blockedByFieldRe extrae el valor del campo "**Blocked by:**" de un
// archivo de ticket.
var blockedByFieldRe = regexp.MustCompile(`(?mi)^\*\*Blocked by:\*\*\s*(.+)$`)

// twoDigitRe encuentra números de dos dígitos dentro del texto de Blocked
// by — la convención de parseo de la spec (sección "Convención de parseo de
// Blocked by en tickets").
var twoDigitRe = regexp.MustCompile(`\d{2}`)

// ticketNNRe extrae el prefijo numérico de dos dígitos del nombre de
// archivo de un ticket (<NN>-<slug>.md).
var ticketNNRe = regexp.MustCompile(`^(\d{2})-`)

// computeStatus es el wrapper delgado de computeStatusFromFS sobre el
// filesystem real (os.DirFS(".")) — mismo patrón que
// planScaffold/planScaffoldFromFS en scaffold.go. Es el único punto que
// cmdStatus/runStatus invoca.
func computeStatus(targetID *int) (statusReport, error) {
	return computeStatusFromFS(os.DirFS("."), targetID)
}

// computeStatusFromFS calcula el reporte completo de status leyendo fsys:
// feature_list.json, specs/<name>/spec.md y specs/<name>/tickets/*.md.
// Pura — nunca escribe, todo lo que necesita lo recibe vía fsys. targetID
// == nil significa "sin argumento, elegí la feature activa"; ver
// selectTarget.
func computeStatusFromFS(fsys fs.FS, targetID *int) (statusReport, error) {
	flData, err := fs.ReadFile(fsys, "feature_list.json")
	if err != nil {
		return statusReport{}, fmt.Errorf("leyendo feature_list.json: %w", err)
	}

	var fl featureListFile
	if err := json.Unmarshal(flData, &fl); err != nil {
		return statusReport{}, fmt.Errorf("parseando feature_list.json: %w", err)
	}

	validStatus := make(map[string]bool, len(fl.Rules.ValidStatus))
	for _, s := range fl.Rules.ValidStatus {
		validStatus[s] = true
	}

	// Se precargan spec/tickets de cada feature sdd:true una sola vez: los
	// necesitan tanto blockedReasons (alcance global, todo feature_list.json)
	// como la derivación de phase/artifactPaths de la feature target.
	specExistsByFeature := map[string]bool{}
	specSatisfiesGWTByFeature := map[string]bool{}
	ticketsByFeature := map[string][]ticketInfo{}
	for _, f := range fl.Features {
		if !f.SDD {
			continue
		}
		specExistsByFeature[f.Name] = fileExistsFS(fsys, specMdPath(f.Name))
		if specExistsByFeature[f.Name] {
			specContent, err := fs.ReadFile(fsys, specMdPath(f.Name))
			if err != nil {
				return statusReport{}, fmt.Errorf("leyendo %s: %w", specMdPath(f.Name), err)
			}
			specSatisfiesGWTByFeature[f.Name] = specSatisfiesGWT(string(specContent))
		}
		tickets, err := readTickets(fsys, f.Name)
		if err != nil {
			return statusReport{}, err
		}
		ticketsByFeature[f.Name] = tickets
	}

	ledgerEntries, corruptLedgerLines, err := readLedger(fsys)
	if err != nil {
		return statusReport{}, err
	}
	currentTreeHash, err := hashTree(fsys)
	if err != nil {
		return statusReport{}, fmt.Errorf("calculando hashTree del árbol actual: %w", err)
	}

	blockedReasons := computeBlockedReasons(fl, validStatus, specExistsByFeature, specSatisfiesGWTByFeature, ticketsByFeature, ledgerEntries, corruptLedgerLines, currentTreeHash)

	target, err := selectTarget(fl.Features, targetID)
	if err != nil {
		return statusReport{}, err
	}

	report := statusReport{
		BlockedReasons: blockedReasons,
		Frontier:       []string{},
		ArtifactPaths:  statusArtifactPaths{FeatureList: "feature_list.json"},
	}

	if target == nil {
		// Backlog íntegramente done/blocked: no hay target, pero no es un
		// error — ver spec, "Selección de la feature activa".
		report.Phase = phaseClosed
		report.NextRecommended = nextRecommendedText(blockedReasons, phaseClosed, nil)
		return report, nil
	}

	tickets := ticketsByFeature[target.Name]
	phase := derivePhase(*target, specExistsByFeature[target.Name], tickets)

	report.Phase = phase
	report.Frontier = computeFrontier(tickets)
	report.ArtifactPaths = buildArtifactPaths(*target, tickets)
	report.NextRecommended = nextRecommendedText(blockedReasons, phase, target)

	return report, nil
}

// derivePhase deriva la fase de una única feature combinando su sdd,
// status y lo que exista en disco, según la tabla de siete casos de la
// spec (sección "Derivación de phase").
func derivePhase(f featureEntry, specExists bool, tickets []ticketInfo) string {
	if f.Status == "done" {
		return phaseClosed
	}
	if f.Name == bootstrapFeatureName {
		return phaseGrill
	}
	if f.SDD {
		if !specExists {
			return phaseSpec
		}
		if len(tickets) == 0 {
			return phaseTickets
		}
		if allTicketsDone(tickets) {
			return phaseReview
		}
		return phaseImplementation
	}
	// sdd:false (y no bootstrap): no hay fase previa que esperar. Cubre
	// pending/in_progress explícitamente y, por extensión, blocked (que ya
	// queda señalado aparte en blockedReasons) — no hay ninguna otra fase a
	// la que pueda caer una feature sdd:false no cerrada.
	return phaseImplementation
}

func allTicketsDone(tickets []ticketInfo) bool {
	for _, t := range tickets {
		if t.Status != "done" {
			return false
		}
	}
	return true
}

// selectTarget resuelve la feature a inspeccionar. Con targetID explícito,
// busca esa feature exacta (error ErrFeatureNotFound si no existe). Sin
// targetID: la única in_progress si existe; si hay más de una (estado
// inconsistente, ya señalado en blockedReasons) la de menor id entre ellas;
// si no hay ninguna in_progress, la pending de menor id; si no hay ni
// pending ni in_progress, no hay target (nil, nil — no es un error).
func selectTarget(features []featureEntry, targetID *int) (*featureEntry, error) {
	if targetID != nil {
		for i := range features {
			if features[i].ID == *targetID {
				f := features[i]
				return &f, nil
			}
		}
		return nil, fmt.Errorf("%w: id %d", ErrFeatureNotFound, *targetID)
	}

	if f := firstByStatusSortedByID(features, "in_progress"); f != nil {
		return f, nil
	}
	if f := firstByStatusSortedByID(features, "pending"); f != nil {
		return f, nil
	}
	return nil, nil
}

// firstByStatusSortedByID devuelve, entre las features con el status dado,
// la de menor id — o nil si no hay ninguna.
func firstByStatusSortedByID(features []featureEntry, status string) *featureEntry {
	var matches []featureEntry
	for _, f := range features {
		if f.Status == status {
			matches = append(matches, f)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].ID < matches[j].ID })
	chosen := matches[0]
	return &chosen
}

// computeBlockedReasons calcula blockedReasons sobre TODO feature_list.json
// y todo specs/, sin importar qué id se pidió (ver spec, "blockedReasons").
// Cubre: más de una feature in_progress; status fuera de
// rules.valid_status; feature sdd:true con status que requiere spec sin
// specs/<name>/spec.md; feature marcada blocked; Status de un ticket fuera
// de pending/in_progress/done; y (ticket 04,
// specs/verify_record_ledger/spec.md) evidencia de tests faltante o
// desactualizada para la feature in_progress (no_test_evidence); (ticket 02,
// specs/review_verdict_recorded/spec.md) veredicto de revisión faltante,
// que no habilita cierre, o desactualizado para la feature in_progress
// (no_review_verdict); (ticket 02, specs/spec_gwt_mechanical_check/spec.md)
// spec aprobada sin ningún bloque Given/When/Then ni marcador de opt-out,
// todavía sin tickets en disco, para una feature que no está done
// (no_gwt_coverage); más cualquier línea corrupta del ledger, reportada
// aparte.
func computeBlockedReasons(fl featureListFile, validStatus map[string]bool, specExistsByFeature map[string]bool, specSatisfiesGWTByFeature map[string]bool, ticketsByFeature map[string][]ticketInfo, ledgerEntries []ledgerEntry, corruptLedgerLines []string, currentTreeHash string) []string {
	reasons := []string{}

	inProgressCount := 0
	var inProgressIDs []int
	for _, f := range fl.Features {
		if f.Status == "in_progress" {
			inProgressCount++
			inProgressIDs = append(inProgressIDs, f.ID)
		}
	}
	if inProgressCount > 1 {
		sort.Ints(inProgressIDs)
		idStrs := make([]string, len(inProgressIDs))
		for i, id := range inProgressIDs {
			idStrs[i] = strconv.Itoa(id)
		}
		reasons = append(reasons, fmt.Sprintf("hay %d features en in_progress a la vez (máximo 1 permitido por one_feature_at_a_time) — ids en in_progress: %s; correr `april feature set-status <id> pending` para bajar alguna a pending", inProgressCount, strings.Join(idStrs, ", ")))
	}

	requiresSpec := map[string]bool{"spec_ready": true, "in_progress": true, "done": true}

	for _, f := range fl.Features {
		if !validStatus[f.Status] {
			reasons = append(reasons, fmt.Sprintf("feature %d (%s) tiene status %q fuera de rules.valid_status", f.ID, f.Name, f.Status))
		}
		if f.SDD && requiresSpec[f.Status] && !specExistsByFeature[f.Name] {
			reasons = append(reasons, fmt.Sprintf("feature %d (%s) está en status %q pero falta %s", f.ID, f.Name, f.Status, specMdPath(f.Name)))
		}
		if f.Status == "blocked" {
			reasons = append(reasons, fmt.Sprintf("feature %d (%s) está marcada blocked", f.ID, f.Name))
		}
		if f.SDD && specExistsByFeature[f.Name] && len(ticketsByFeature[f.Name]) == 0 && f.Status != "done" && !specSatisfiesGWTByFeature[f.Name] {
			reasons = append(reasons, fmt.Sprintf("feature %d (%s) tiene %s sin ningún bloque Given/When/Then ni el marcador %s (no_gwt_coverage) — agregar al menos un bloque Given/When/Then a %s, o el marcador %s si ninguna historia de usuario tiene rama de comportamiento verificable", f.ID, f.Name, specMdPath(f.Name), gwtOptOutMarker, specMdPath(f.Name), gwtOptOutMarker))
		}
		if f.Status == "in_progress" {
			if reason := noTestEvidenceReason(f, ledgerEntries, currentTreeHash); reason != "" {
				reasons = append(reasons, reason)
			}
			if reason := noReviewVerdictReason(f, ledgerEntries, currentTreeHash); reason != "" {
				reasons = append(reasons, reason)
			}
		}
		for _, t := range ticketsByFeature[f.Name] {
			if !isValidTicketStatus(t.Status) {
				reasons = append(reasons, fmt.Sprintf("ticket %s de la feature %s tiene Status %q fuera de pending/in_progress/done", t.Filename, f.Name, t.Status))
			}
		}
		reasons = append(reasons, ticketBlockedByReasons(f.Name, ticketsByFeature[f.Name])...)
		if cyc := detectBlockedByCycle(f.Name, ticketsByFeature[f.Name]); cyc != "" {
			reasons = append(reasons, cyc)
		}
	}

	// Las líneas corruptas del ledger se reportan aparte, sin importar qué
	// feature esté in_progress (ver spec, "Extensión de
	// computeBlockedReasons").
	reasons = append(reasons, corruptLedgerLines...)

	return reasons
}

// lastTestEntryForFeature busca, entre las entradas del ledger, la última
// (por orden de aparición, el archivo ya es cronológico por ser
// append-only) con kind == "test" y featureId == featureID. Una entrada
// kind:review nunca cuenta, aunque sea para la misma feature (ver spec,
// "Extensión de computeBlockedReasons").
func lastTestEntryForFeature(entries []ledgerEntry, featureID int) (ledgerEntry, bool) {
	var last ledgerEntry
	found := false
	for _, e := range entries {
		if e.Kind == "test" && e.FeatureID == featureID {
			last = e
			found = true
		}
	}
	return last, found
}

// noTestEvidenceReason decide, para una única feature in_progress, si falta
// evidencia de tests vigente: sin ningún receipt kind:test, con el último
// en rojo (exitCode != 0), o con treeHash desactualizado respecto al árbol
// actual. Devuelve "" si el último receipt kind:test está en verde y
// vigente. El string devuelto contiene siempre la substring literal
// "no_test_evidence" (contrato de la spec/ticket 04).
func noTestEvidenceReason(f featureEntry, entries []ledgerEntry, currentTreeHash string) string {
	last, found := lastTestEntryForFeature(entries, f.ID)
	if !found {
		return fmt.Sprintf("feature %d (%s) está in_progress pero no tiene ningún receipt kind:test en %s (no_test_evidence) — april verify record --feature %d -- <comando>", f.ID, f.Name, verifyLedgerPath, f.ID)
	}
	if last.ExitCode != 0 {
		return fmt.Sprintf("feature %d (%s) está in_progress pero su último receipt kind:test tiene exitCode %d != 0 (no_test_evidence) — april verify record --feature %d -- <comando>", f.ID, f.Name, last.ExitCode, f.ID)
	}
	if last.TreeHash != currentTreeHash {
		return fmt.Sprintf("feature %d (%s) está in_progress pero el treeHash de su último receipt kind:test (%s) no coincide con el árbol actual (%s) — el código cambió después de la corrida registrada (no_test_evidence) — april verify record --feature %d -- <comando>", f.ID, f.Name, last.TreeHash, currentTreeHash, f.ID)
	}
	return ""
}

// lastReviewEntryForFeature busca, entre las entradas del ledger, la última
// (por orden de aparición, el archivo ya es cronológico por ser
// append-only) con kind == "review" y featureId == featureID. Una entrada
// kind:test nunca cuenta, aunque sea para la misma feature — misma
// separación estricta que lastTestEntryForFeature en el otro sentido (ver
// spec, "Extensión de status.go").
func lastReviewEntryForFeature(entries []ledgerEntry, featureID int) (ledgerEntry, bool) {
	var last ledgerEntry
	found := false
	for _, e := range entries {
		if e.Kind == "review" && e.FeatureID == featureID {
			last = e
			found = true
		}
	}
	return last, found
}

// noReviewVerdictReason decide, para una única feature in_progress, si falta
// un veredicto de revisión vigente que habilite el cierre: sin ninguna
// entrada kind:review, con el último verdict fuera de
// APPROVED/APPROVED_WITH_OBJECTION (incluye CHANGES_REQUESTED), o con
// treeHash desactualizado respecto al árbol actual. Devuelve "" si la
// última entrada kind:review tiene un verdict que habilita cierre y su
// treeHash coincide con el árbol actual. El string devuelto contiene
// siempre la substring literal "no_review_verdict" (contrato de la
// spec/ticket 02 de review_verdict_recorded).
func noReviewVerdictReason(f featureEntry, entries []ledgerEntry, currentTreeHash string) string {
	last, found := lastReviewEntryForFeature(entries, f.ID)
	if !found {
		return fmt.Sprintf("feature %d (%s) está in_progress pero no tiene ningún receipt kind:review en %s (no_review_verdict) — april review record --feature %d --verdict <valor> (valores válidos: APPROVED, APPROVED_WITH_OBJECTION, CHANGES_REQUESTED)", f.ID, f.Name, verifyLedgerPath, f.ID)
	}
	if last.Verdict != verdictApproved && last.Verdict != verdictApprovedWithObjection {
		return fmt.Sprintf("feature %d (%s) está in_progress pero su último receipt kind:review tiene verdict %q, que no habilita cierre (no_review_verdict) — april review record --feature %d --verdict <valor> (valores que habilitan cierre: APPROVED, APPROVED_WITH_OBJECTION)", f.ID, f.Name, last.Verdict, f.ID)
	}
	if last.TreeHash != currentTreeHash {
		return fmt.Sprintf("feature %d (%s) está in_progress pero el treeHash de su último receipt kind:review (%s) no coincide con el árbol actual (%s) — el código cambió después del veredicto registrado (no_review_verdict) — april review record --feature %d --verdict <valor>", f.ID, f.Name, last.TreeHash, currentTreeHash, f.ID)
	}
	return ""
}

// specSatisfiesGWT decide si el contenido de un spec.md satisface el
// requisito de cobertura Given/When/Then (ver
// specs/spec_gwt_mechanical_check/spec.md, "Detección del bloque
// Given/When/Then"): o bien el archivo completo contiene el marcador de
// opt-out gwtOptOutMarker en cualquier parte (basta por sí solo, incluso
// con GWT real presente — la redundancia no se arbitra), o bien tiene, en
// cualquier parte del archivo, al menos una línea que empieza (ignorando
// espacios en blanco iniciales) literalmente con "Given", al menos una con
// "When" y al menos una con "Then" — sensible a mayúsculas/minúsculas, sin
// exigir adyacencia ni orden relativo entre ellas.
func specSatisfiesGWT(content string) bool {
	if strings.Contains(content, gwtOptOutMarker) {
		return true
	}

	var hasGiven, hasWhen, hasThen bool
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		switch {
		case strings.HasPrefix(trimmed, "Given"):
			hasGiven = true
		case strings.HasPrefix(trimmed, "When"):
			hasWhen = true
		case strings.HasPrefix(trimmed, "Then"):
			hasThen = true
		}
	}
	return hasGiven && hasWhen && hasThen
}

func isValidTicketStatus(s string) bool {
	switch s {
	case "pending", "in_progress", "done":
		return true
	}
	return false
}

// buildArtifactPaths arma las rutas relevantes para la feature target:
// featureList siempre; spec si sdd:true; ticketsDir/tickets si hay al
// menos un archivo de ticket en disco.
func buildArtifactPaths(f featureEntry, tickets []ticketInfo) statusArtifactPaths {
	ap := statusArtifactPaths{FeatureList: "feature_list.json"}
	if f.SDD {
		ap.Spec = specMdPath(f.Name)
	}
	if len(tickets) > 0 {
		ap.TicketsDir = ticketsDirPath(f.Name)
		for _, t := range tickets {
			ap.Tickets = append(ap.Tickets, t.Filename)
		}
	}
	return ap
}

// nextRecommendedText describe la única acción legal, o "" si
// blockedReasons no está vacío — nunca hay recomendación de avanzar
// mientras haya un problema sin resolver (ver spec, "nextRecommended").
func nextRecommendedText(blockedReasons []string, phase string, target *featureEntry) string {
	if len(blockedReasons) > 0 {
		return ""
	}

	switch phase {
	case phaseSpec:
		return fmt.Sprintf("lanzar spec_writer para la feature %d (%s)", target.ID, target.Name)
	case phaseTickets:
		return fmt.Sprintf("lanzar ticket_writer para la feature %d (%s)", target.ID, target.Name)
	case phaseImplementation:
		return fmt.Sprintf("implementar la frontera de tickets pendientes de la feature %d (%s)", target.ID, target.Name)
	case phaseReview:
		return fmt.Sprintf("lanzar reviewer_agent para la feature %d (%s)", target.ID, target.Name)
	case phaseGrill:
		return "continuar la Fase Grill de bootstrap_project"
	case phaseClosed:
		if target != nil {
			return fmt.Sprintf("nada — la feature %d (%s) ya está cerrada", target.ID, target.Name)
		}
		return "nada — no hay features pendientes"
	default:
		return ""
	}
}

// specMdPath es la ruta esperada del spec de una feature, relativa a la
// raíz del repo/proyecto (rutas de fs.FS, siempre con "/").
func specMdPath(featureName string) string {
	return path.Join("specs", featureName, "spec.md")
}

// ticketsDirRaw es la ruta del directorio de tickets de una feature, sin
// slash final (para fs.ReadDir).
func ticketsDirRaw(featureName string) string {
	return path.Join("specs", featureName, "tickets")
}

// ticketsDirPath es la ruta del directorio de tickets con slash final, tal
// como la espera artifactPaths.ticketsDir.
func ticketsDirPath(featureName string) string {
	return ticketsDirRaw(featureName) + "/"
}

// fileExistsFS reporta si p existe en fsys.
func fileExistsFS(fsys fs.FS, p string) bool {
	_, err := fs.Stat(fsys, p)
	return err == nil
}

// readTickets lee specs/<name>/tickets/*.md de fsys y extrae de cada uno su
// número de orden (NN) y su campo Status. Si el directorio no existe
// todavía (fase "tickets": spec aprobada pero sin desglose), no es un
// error: devuelve una lista vacía.
func readTickets(fsys fs.FS, featureName string) ([]ticketInfo, error) {
	dirPath := ticketsDirRaw(featureName)

	entries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("leyendo %s: %w", dirPath, err)
	}

	var tickets []ticketInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		content, err := fs.ReadFile(fsys, path.Join(dirPath, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("leyendo %s/%s: %w", dirPath, e.Name(), err)
		}
		nn := ""
		if m := ticketNNRe.FindStringSubmatch(e.Name()); m != nil {
			nn = m[1]
		}
		raw, blockedBy, ok := parseBlockedBy(string(content))
		tickets = append(tickets, ticketInfo{
			NN:             nn,
			Filename:       e.Name(),
			Status:         parseTicketStatus(string(content)),
			BlockedByRaw:   raw,
			BlockedBy:      blockedBy,
			BlockedByValid: ok,
		})
	}

	// Segunda pasada: valida los NN referenciados en Blocked by contra los
	// tickets que de verdad existen en esta feature — un número de dos
	// dígitos que no corresponde a ningún archivo también cae en "no
	// interpretable" (ver spec, "Convención de parseo de Blocked by").
	nnSet := map[string]bool{}
	for _, t := range tickets {
		if t.NN != "" {
			nnSet[t.NN] = true
		}
	}
	for i := range tickets {
		if !tickets[i].BlockedByValid {
			continue
		}
		for _, nn := range tickets[i].BlockedBy {
			if !nnSet[nn] {
				tickets[i].BlockedByValid = false
				break
			}
		}
	}

	return tickets, nil
}

// parseBlockedBy interpreta el campo "**Blocked by:**" del contenido de un
// ticket según la convención de la spec: busca todos los números de dos
// dígitos en el texto y los toma como los NN que bloquean a este ticket; si
// no hay números pero el texto (sin distinguir mayúsculas/minúsculas)
// contiene "none", el ticket no tiene bloqueadores. Cualquier otro caso —
// campo ausente, texto sin número y sin "none" — se reporta como no válido
// (ok == false) para que el llamador lo reporte en blockedReasons en vez de
// asumir "sin bloqueadores" en silencio.
func parseBlockedBy(content string) (raw string, blockedBy []string, ok bool) {
	m := blockedByFieldRe.FindStringSubmatch(content)
	if m != nil {
		raw = strings.TrimSpace(m[1])
	}

	nums := twoDigitRe.FindAllString(raw, -1)
	if len(nums) > 0 {
		seen := map[string]bool{}
		for _, n := range nums {
			if !seen[n] {
				seen[n] = true
				blockedBy = append(blockedBy, n)
			}
		}
		return raw, blockedBy, true
	}

	if strings.Contains(strings.ToLower(raw), "none") {
		return raw, []string{}, true
	}

	return raw, nil, false
}

// ticketIdentifier es el identificador de ticket usado en frontier: el
// nombre de archivo (NN-slug) sin la extensión .md.
func ticketIdentifier(filename string) string {
	return strings.TrimSuffix(filename, ".md")
}

// computeFrontier calcula, para los tickets de la feature target, los
// tomables en paralelo ahora mismo: Status distinto de "done", Blocked by
// interpretable, y todos sus bloqueadores en Status: done. Un ticket cuyo
// Blocked by no se pudo interpretar queda excluido — no se puede confirmar
// con seguridad que no tiene bloqueadores pendientes, y ya se reporta aparte
// en blockedReasons.
func computeFrontier(tickets []ticketInfo) []string {
	statusByNN := map[string]string{}
	for _, t := range tickets {
		if t.NN != "" {
			statusByNN[t.NN] = t.Status
		}
	}

	frontier := []string{}
	for _, t := range tickets {
		if t.Status == "done" || !isValidTicketStatus(t.Status) {
			continue
		}
		if !t.BlockedByValid {
			continue
		}
		allDone := true
		for _, nn := range t.BlockedBy {
			if statusByNN[nn] != "done" {
				allDone = false
				break
			}
		}
		if allDone {
			frontier = append(frontier, ticketIdentifier(t.Filename))
		}
	}
	return frontier
}

// ticketBlockedByReasons reporta, para los tickets de una feature, cualquier
// Blocked by no interpretable (ver parseBlockedBy y la segunda pasada de
// readTickets, que también invalida referencias a NN inexistentes).
func ticketBlockedByReasons(featureName string, tickets []ticketInfo) []string {
	var reasons []string
	for _, t := range tickets {
		if !t.BlockedByValid {
			reasons = append(reasons, fmt.Sprintf("ticket %s de la feature %s tiene Blocked by no interpretable (%q): ni números de ticket de dos dígitos ni \"none\", o referencia un ticket inexistente — el formato esperado es números de ticket de dos dígitos separados por coma, ej. \"01, 02\", o la palabra \"none\" si no tiene bloqueadores; editar el campo **Blocked by:** de %s", t.Filename, featureName, t.BlockedByRaw, t.Filename))
		}
	}
	return reasons
}

// detectBlockedByCycle busca un ciclo en el grafo de Blocked by de los
// tickets de una feature con DFS + pila de recursión — acotado por
// construcción al número de tickets leídos (cada nodo se visita una sola
// vez vía el mapa de colores), nunca recursión sin límite. Solo considera
// aristas ya validadas (BlockedByValid == true); las no interpretables ya se
// reportan aparte y no participan del grafo. Devuelve "" si no hay ciclo, o
// un mensaje con la feature y la cadena de tickets que lo forma.
func detectBlockedByCycle(featureName string, tickets []ticketInfo) string {
	adj := map[string][]string{}
	for _, t := range tickets {
		if t.NN == "" || !t.BlockedByValid {
			continue
		}
		adj[t.NN] = append(adj[t.NN], t.BlockedBy...)
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var path []string
	var cycle []string

	var dfs func(nn string)
	dfs = func(nn string) {
		if cycle != nil {
			return
		}
		color[nn] = gray
		path = append(path, nn)
		for _, next := range adj[nn] {
			if cycle != nil {
				return
			}
			switch color[next] {
			case white:
				dfs(next)
			case gray:
				idx := indexOfString(path, next)
				cycle = append(append([]string{}, path[idx:]...), next)
			}
		}
		if cycle == nil {
			path = path[:len(path)-1]
			color[nn] = black
		}
	}

	for _, t := range tickets {
		if t.NN == "" {
			continue
		}
		if color[t.NN] == white {
			dfs(t.NN)
		}
		if cycle != nil {
			break
		}
	}

	if cycle == nil {
		return ""
	}

	// Resuelve el primer NN de la cadena detectada a su Filename real, para
	// que la receta nombre al menos un archivo concreto a editar (ver spec,
	// "Recetas a agregar, caso por caso" — sin estructura ni archivo nuevo,
	// el mapeo se arma aquí mismo con el tickets []ticketInfo ya recibido).
	filenameByNN := map[string]string{}
	for _, t := range tickets {
		if t.NN != "" {
			filenameByNN[t.NN] = t.Filename
		}
	}
	firstFilename := filenameByNN[cycle[0]]

	return fmt.Sprintf("ciclo detectado en Blocked by de tickets de %s: %s — editar el campo **Blocked by:** de %s (o de otro ticket de la cadena) para quitar o corregir la referencia que cierra el ciclo", featureName, strings.Join(cycle, " → "), firstFilename)
}

// indexOfString devuelve el índice de v en s, o -1 si no está.
func indexOfString(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// parseTicketStatus extrae el valor del campo "**Status:**" del contenido
// de un archivo de ticket. Devuelve "" si el campo no está presente (esto
// mismo lo vuelve inválido frente a isValidTicketStatus, y por lo tanto se
// reporta en blockedReasons en vez de asumirse en silencio).
func parseTicketStatus(content string) string {
	m := statusFieldRe.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// readLedger lee el ledger de evidencia .claude/verify-ledger.jsonl (ver
// verify.go, verifyLedgerPath) línea por línea, ignorando líneas vacías.
// Intenta json.Unmarshal cada línea no vacía: si falla, la línea (con su
// número, 1-indexado) va a corruptLines en vez de abortar la lectura
// completa — mismo espíritu que parseBlockedBy/ticketBlockedByReasons:
// nunca fallar en silencio, nunca tirar todo el cálculo por una entrada
// mala. Archivo inexistente (nadie corrió `verify record` todavía) no es
// error: entries vacío, corruptLines vacío.
func readLedger(fsys fs.FS) (entries []ledgerEntry, corruptLines []string, err error) {
	data, err := fs.ReadFile(fsys, verifyLedgerPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("leyendo %s: %w", verifyLedgerPath, err)
	}

	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e ledgerEntry
		if uerr := json.Unmarshal([]byte(line), &e); uerr != nil {
			corruptLines = append(corruptLines, fmt.Sprintf("línea %d de %s no es JSON válido: %v", i+1, verifyLedgerPath, uerr))
			continue
		}
		entries = append(entries, e)
	}

	return entries, corruptLines, nil
}

// runStatus contiene toda la lógica de `april status`: parseo de args
// (targetID posicional opcional, --json en cualquier posición), cálculo vía
// computeStatus, formateo de salida (JSON o texto plano) y decisión del
// exit code. Se separa de cmdStatus (que sí llama a os.Exit) para poder
// testearla in-process sin terminar el proceso de test — mismo espíritu que
// la separación scaffoldInit/cmdInit en scaffold.go.
func runStatus(args []string) int {
	jsonOutput := false
	var targetID *int

	for _, a := range args {
		if a == "--json" {
			jsonOutput = true
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		id, err := strconv.Atoi(a)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: id inválido %q (debe ser el id numérico de una feature)\n", a)
			return 1
		}
		targetID = &id
	}

	report, err := computeStatus(targetID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if jsonOutput {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(string(data))
	} else {
		printStatusText(report)
	}

	if len(report.BlockedReasons) > 0 {
		return 1
	}
	return 0
}

// cmdStatus es el entry point del CLI para `april status [id] [--json]`:
// arma los argumentos desde os.Args y termina el proceso con el exit code
// que decide runStatus.
func cmdStatus() {
	os.Exit(runStatus(os.Args[2:]))
}

// printStatusText imprime el reporte en texto plano legible (sin --json),
// con los mismos cinco campos que la salida JSON.
func printStatusText(r statusReport) {
	fmt.Printf("Phase: %s\n", r.Phase)

	if r.NextRecommended != "" {
		fmt.Printf("Next recommended: %s\n", r.NextRecommended)
	}

	if len(r.BlockedReasons) > 0 {
		fmt.Println("Blocked reasons:")
		for _, reason := range r.BlockedReasons {
			fmt.Printf("  - %s\n", reason)
		}
	}

	if len(r.Frontier) > 0 {
		fmt.Println("Frontier:")
		for _, t := range r.Frontier {
			fmt.Printf("  - %s\n", t)
		}
	}

	fmt.Println("Artifact paths:")
	fmt.Printf("  featureList: %s\n", r.ArtifactPaths.FeatureList)
	if r.ArtifactPaths.Spec != "" {
		fmt.Printf("  spec: %s\n", r.ArtifactPaths.Spec)
	}
	if r.ArtifactPaths.TicketsDir != "" {
		fmt.Printf("  ticketsDir: %s\n", r.ArtifactPaths.TicketsDir)
		for _, t := range r.ArtifactPaths.Tickets {
			fmt.Printf("    - %s\n", t)
		}
	}
}
