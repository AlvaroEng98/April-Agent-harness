package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrNotGitRepo indica que el directorio actual no es un repositorio git,
// o que el binario git no está disponible en el PATH — computeSubjectHash
// no tiene fallback silencioso a hashTree ante ninguno de los dos casos
// (spec, US10/US29).
var ErrNotGitRepo = errors.New("no es un repositorio git")

// ErrStaleSubjectHash señala que el subject_hash pasado a `review record
// --subject-hash <hash>` no coincide con el candidato recalculado en el
// momento de registrar el veredicto: el árbol de trabajo cambió entre que
// se calculó ese hash (típicamente con `review start`) y el intento de
// registro. recordReviewWithSubjectHash rechaza el registro sin tocar el
// ledger cuando esto pasa (spec, US3).
var ErrStaleSubjectHash = errors.New("stale subject_hash")

// fixedTreeExclusions son las rutas que tanto isExcludedFromTreeHash
// (hashTree, verify.go) como computeSubjectHash excluyen sin condicionarlo
// a .gitignore — no dependen de estar ahí (pueden estarlo o no; en este
// repo progress/*.md sí está en .gitignore pero progress/current.md sigue
// trackeado, ver specs/tree_hash_respects_gitignore/spec.md, Problem
// Statement). .git/ no entra en esta lista: computeSubjectHash nunca
// necesita excluirlo a mano porque git no se trackea a sí mismo (ver
// specs/review_frozen_candidate/spec.md).
var fixedTreeExclusions = []string{verifyLedgerPath, "progress"}

// computeSubjectHash calcula el subject_hash: el SHA-1 de árbol que
// devuelve `git write-tree` sobre un índice temporal aislado (vía
// GIT_INDEX_FILE), reflejando el contenido actual del árbol de trabajo
// (staged + unstaged + untracked no ignorado) salvo las mismas dos rutas
// que hashTree ya excluye por el mismo motivo de auto-invalidación
// (verifyLedgerPath, progress/). Nunca toca el índice real (.git/index)
// del usuario. Building-block puro sobre subprocesos git reales, todavía
// sin CLI que lo invoque (llega en los tickets 02/03).
func computeSubjectHash() (string, error) {
	if _, _, err := runGit(nil, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", fmt.Errorf("%w: %v", ErrNotGitRepo, err)
	}

	f, err := os.CreateTemp("", "april-subject-index-*")
	if err != nil {
		return "", fmt.Errorf("creando archivo de índice temporal: %w", err)
	}
	indexPath := f.Name()
	f.Close()
	os.Remove(indexPath)
	defer os.Remove(indexPath)

	env := append(os.Environ(), "GIT_INDEX_FILE="+indexPath)

	if _, _, err := runGit(env, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add -A sobre el índice temporal: %w", err)
	}

	rmArgs := append([]string{"rm", "--cached", "-r", "--ignore-unmatch", "--"}, fixedTreeExclusions...)
	if _, _, err := runGit(env, rmArgs...); err != nil {
		return "", fmt.Errorf("git rm --cached sobre el índice temporal: %w", err)
	}

	stdout, _, err := runGit(env, "write-tree")
	if err != nil {
		return "", fmt.Errorf("git write-tree sobre el índice temporal: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

// runGit corre git con args, con env opcional (nil deja que exec.Command
// use el entorno del proceso actual sin modificar). Devuelve stdout/stderr
// capturados por separado; err no nil tanto si el binario no arrancó
// (ausente del PATH) como si terminó con exit code distinto de cero — a
// diferencia de recordVerify, acá ambos casos son la misma clase de falla
// para el llamador (computeSubjectHash no distingue entre ellos, salvo en
// el mensaje).
func runGit(env []string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command("git", args...)
	if env != nil {
		cmd.Env = env
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout, stderr = outBuf.String(), errBuf.String()
	if runErr != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = runErr.Error()
		}
		return stdout, stderr, errors.New(detail)
	}
	return stdout, stderr, nil
}

// isValidVerdict decide si v es uno de los tres valores reconocidos por
// review record (verdictApproved, verdictApprovedWithObjection,
// verdictChangesRequested — reusados tal cual de set_status.go). Extraído
// de recordReview (ticket 03) para que recordReviewWithSubjectHash use
// exactamente el mismo vocabulario, sin una segunda lista definida en otro
// lugar (spec, US18).
func isValidVerdict(v string) bool {
	switch v {
	case verdictApproved, verdictApprovedWithObjection, verdictChangesRequested:
		return true
	default:
		return false
	}
}

// recordReview es la función pura de orquestación de `april review record
// --feature <id> --verdict <valor>`, análoga a recordVerify pero sin
// subproceso: el veredicto ya fue decidido por reviewer_agent antes de
// invocar el comando, esta función solo lo persiste.
//
// Valida que verdict sea uno de los tres valores reconocidos vía
// isValidVerdict. Un valor fuera de ese vocabulario es un error de
// invocación explícito, sin tocar el ledger. CHANGES_REQUESTED se acepta y
// registra normalmente — no es un error, registrar un rechazo es la función
// del comando (US19/US20 de la spec).
//
// Con un verdict válido, calcula hashTree(os.DirFS(".")), arma la
// ledgerEntry con Kind "review" y hace el append atómico vía
// appendToLedger (reusada tal cual de verify.go). No corre ningún
// exec.Command.
func recordReview(featureID int, verdict string) (entry ledgerEntry, err error) {
	if !isValidVerdict(verdict) {
		return ledgerEntry{}, fmt.Errorf("--verdict inválido %q (debe ser uno de %s, %s, %s)", verdict, verdictApproved, verdictApprovedWithObjection, verdictChangesRequested)
	}

	treeHash, err := hashTree(os.DirFS("."))
	if err != nil {
		return ledgerEntry{}, fmt.Errorf("calculando hashTree: %w", err)
	}

	entry = ledgerEntry{
		Kind:      "review",
		FeatureID: featureID,
		TreeHash:  treeHash,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Verdict:   verdict,
	}

	if err := appendToLedger(entry); err != nil {
		return ledgerEntry{}, fmt.Errorf("anexando al ledger: %w", err)
	}

	return entry, nil
}

// recordReviewWithSubjectHash es la variante de recordReview que además
// valida el candidato congelado de git (spec, "recordReviewWithSubjectHash"
// de Implementation Decisions): valida verdict con isValidVerdict (mismo
// vocabulario, mismo comportamiento de error sin tocar el ledger),
// recalcula el candidato actual con computeSubjectHash() y lo compara
// contra subjectHash. Si no coincide, rechaza el registro con un error que
// envuelve ErrStaleSubjectHash (ambos hashes en el mensaje), sin tocar el
// ledger. Si coincide, calcula también hashTree(os.DirFS(".")) — igual que
// recordReview, para no perder esa señal (US20) —, arma la ledgerEntry con
// Kind "review", TreeHash, SubjectHash y Verdict, y hace el append atómico
// vía appendToLedger (reusada tal cual). Si computeSubjectHash falla (no es
// un repositorio git, git no disponible), el error se propaga tal cual, sin
// fallback (US11).
func recordReviewWithSubjectHash(featureID int, verdict, subjectHash string) (entry ledgerEntry, err error) {
	if !isValidVerdict(verdict) {
		return ledgerEntry{}, fmt.Errorf("--verdict inválido %q (debe ser uno de %s, %s, %s)", verdict, verdictApproved, verdictApprovedWithObjection, verdictChangesRequested)
	}

	current, err := computeSubjectHash()
	if err != nil {
		return ledgerEntry{}, err
	}
	if current != subjectHash {
		return ledgerEntry{}, fmt.Errorf("%w: el árbol cambió desde que se calculó el candidato (recibido %q, actual %q)", ErrStaleSubjectHash, subjectHash, current)
	}

	treeHash, err := hashTree(os.DirFS("."))
	if err != nil {
		return ledgerEntry{}, fmt.Errorf("calculando hashTree: %w", err)
	}

	entry = ledgerEntry{
		Kind:        "review",
		FeatureID:   featureID,
		TreeHash:    treeHash,
		SubjectHash: subjectHash,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Verdict:     verdict,
	}

	if err := appendToLedger(entry); err != nil {
		return ledgerEntry{}, fmt.Errorf("anexando al ledger: %w", err)
	}

	return entry, nil
}

// runReviewRecord contiene la lógica de `april review record --feature
// <id> --verdict <valor> [--subject-hash <hash>]`: parseo simple y
// posicional, mismo estilo estricto que runVerifyRecord/runSetStatus. Los
// primeros cuatro argumentos (--feature <id> --verdict <valor>) se parsean
// exactamente igual que en la feature 6 — cero cambios ahí (US13: los 11
// tests existentes siguen en verde sin editarlos). Lo que sigue después
// (args[4:], ticket 03) se interpreta así:
//   - vacío → recordReview(featureID, verdict), sin cambios;
//   - exactamente ["--subject-hash", "<hash>"] →
//     recordReviewWithSubjectHash(featureID, verdict, hash);
//   - cualquier otra cosa (flag desconocido, --subject-hash sin valor,
//     argumentos de más) → error de invocación explícito, exit≠0, sin
//     invocar ninguna de las dos funciones de registro (US16).
//
// El exit code de éxito es siempre 0 — no hay comando externo cuyo exit
// code reflejar, a diferencia de verify record.
func runReviewRecord(args []string) int {
	const usage = "Error: uso: april review record --feature <id> --verdict <valor> [--subject-hash <hash>]"

	if len(args) < 1 || args[0] != "--feature" {
		fmt.Fprintln(os.Stderr, usage)
		return 1
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: --feature requiere un valor")
		return 1
	}
	featureID, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --feature inválido %q (debe ser el id numérico de una feature)\n", args[1])
		return 1
	}
	if len(args) < 3 || args[2] != "--verdict" {
		fmt.Fprintln(os.Stderr, "Error: falta --verdict entre --feature <id> y el valor del veredicto")
		return 1
	}
	if len(args) < 4 {
		fmt.Fprintln(os.Stderr, "Error: --verdict requiere un valor")
		return 1
	}
	verdict := args[3]
	rest := args[4:]

	switch {
	case len(rest) == 0:
		if _, err := recordReview(featureID, verdict); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case len(rest) == 2 && rest[0] == "--subject-hash":
		subjectHash := rest[1]
		if _, err := recordReviewWithSubjectHash(featureID, verdict, subjectHash); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(os.Stderr, "Error: argumentos de más después de --verdict <valor> (solo se acepta, opcionalmente, --subject-hash <hash>)")
		return 1
	}
}

// runReviewStart contiene la lógica de `april review start --feature <id>
// [--json]`: parsea --feature <id> con el mismo estilo estricto que
// runReviewRecord/runVerifyRecord/runSetStatus (falta el flag, falta el
// valor, o id no numérico son errores de invocación explícitos, exit≠0,
// sin llamar a computeSubjectHash). El id no participa del cálculo del
// hash — se pide solo por consistencia con el resto de la familia
// review/verify (spec, US17). Lo que sigue (args[2:], ticket 03 de la
// feature 8) se interpreta así:
//   - vacío → comportamiento IDÉNTICO a la feature 7: llama solo a
//     computeSubjectHash(), imprime SOLO el subject_hash en stdout (sin
//     texto decorativo) en éxito, o el error en stderr; nunca llama a
//     computeTouchedPaths ni a readSensitiveAreas (US12/US13);
//   - exactamente ["--json"] → arma un reviewStartReport (subjectHash,
//     touchedPaths, sensitiveAreasTouched, extraReviewRequired) y lo
//     imprime como un único objeto JSON;
//   - cualquier otra cosa (typo del flag, argumentos de más, --json en
//     otra posición) → error de invocación explícito, exit 1, sin llamar
//     a ninguna función de cálculo (US16/US17).
//
// Es siempre una consulta pura: no escribe nada al ledger, y
// extraReviewRequired nunca cambia el exit code (US18).
func runReviewStart(args []string) int {
	const usage = "Error: uso: april review start --feature <id> [--json]"

	if len(args) < 1 || args[0] != "--feature" {
		fmt.Fprintln(os.Stderr, usage)
		return 1
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: --feature requiere un valor")
		return 1
	}
	if _, err := strconv.Atoi(args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: --feature inválido %q (debe ser el id numérico de una feature)\n", args[1])
		return 1
	}
	rest := args[2:]

	switch {
	case len(rest) == 0:
		subjectHash, err := computeSubjectHash()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(subjectHash)
		return 0
	case len(rest) == 1 && rest[0] == "--json":
		subjectHash, err := computeSubjectHash()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		touched, err := computeTouchedPaths(subjectHash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		sensitive, err := readSensitiveAreas(os.DirFS("."))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		matched := matchSensitiveAreas(touched, sensitive)

		report := reviewStartReport{
			SubjectHash:           subjectHash,
			TouchedPaths:          touched,
			SensitiveAreasTouched: matched,
			ExtraReviewRequired:   len(matched) > 0,
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Println(string(data))
		return 0
	default:
		fmt.Fprintln(os.Stderr, "Error: argumentos de más o desconocidos después de --feature <id> (solo se acepta, opcionalmente, --json)")
		return 1
	}
}

// cmdReview es el entry point del CLI para `april review <subcomando>
// ...`. Subcomandos válidos: record y start; cualquier otro es un error
// explícito.
func cmdReview() {
	const usage = "Error: uso: april review record --feature <id> --verdict <valor>\n       april review start --feature <id> [--json]"

	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	switch args[0] {
	case "record":
		os.Exit(runReviewRecord(args[1:]))
	case "start":
		os.Exit(runReviewStart(args[1:]))
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
}

// ---- Ticket 01 (review_depth_by_diff_sensitivity, feature 8): Áreas
// sensibles — parseo desde docs/conventions.md ----
//
// Building block puro: todavía ningún subcomando de `april` invoca estas
// funciones (eso llega en el ticket 03). Mismo patrón de dos capas que ya
// usa status.go (parseTicketStatus/parseBlockedBy puras sobre string +
// readTickets como wrapper de I/O sobre fs.FS): parseSensitiveAreas es la
// función pura testeable con literales de string, readSensitiveAreas es el
// wrapper de I/O testeable con fstest.MapFS.

// sensitiveAreasHeadingRe encuentra el encabezado "## Áreas sensibles" de
// docs/conventions.md.
var sensitiveAreasHeadingRe = regexp.MustCompile(`(?m)^## Áreas sensibles\s*$`)

// nextHeadingRe encuentra el siguiente encabezado de nivel 2, el límite
// donde termina la sección "Áreas sensibles".
var nextHeadingRe = regexp.MustCompile(`(?m)^## `)

// sensitiveAreaItemRe extrae la ruta entre backticks al inicio de un ítem
// de lista markdown ("- `ruta` — descripción"). Un ítem de lista sin ruta
// entre backticks simplemente no matchea, y se ignora silenciosamente.
var sensitiveAreaItemRe = regexp.MustCompile("(?m)^- `([^`]+)`")

// parseSensitiveAreas extrae, del contenido ya leído de docs/conventions.md,
// las rutas declaradas en la sección "## Áreas sensibles": encuentra el
// encabezado, recorta hasta el siguiente "## " (o el fin del archivo), y
// devuelve la ruta entre backticks de cada ítem de lista markdown en ese
// rango, en el orden en que aparecen. Si la sección no existe, devuelve
// []string{} — nunca nil, nunca error (la firma no lo permite).
func parseSensitiveAreas(content string) []string {
	loc := sensitiveAreasHeadingRe.FindStringIndex(content)
	if loc == nil {
		return []string{}
	}

	rest := content[loc[1]:]
	if end := nextHeadingRe.FindStringIndex(rest); end != nil {
		rest = rest[:end[0]]
	}

	matches := sensitiveAreaItemRe.FindAllStringSubmatch(rest, -1)
	areas := make([]string, 0, len(matches))
	for _, m := range matches {
		areas = append(areas, m[1])
	}
	return areas
}

// readSensitiveAreas es el wrapper de I/O de parseSensitiveAreas: lee
// docs/conventions.md de fsys y delega en parseSensitiveAreas. Si el
// archivo no existe (fs.ErrNotExist), no es un error — devuelve lista
// vacía (proyecto sin esa sección definida todavía). Cualquier otro error
// de lectura se propaga envuelto con contexto.
func readSensitiveAreas(fsys fs.FS) ([]string, error) {
	data, err := fs.ReadFile(fsys, "docs/conventions.md")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("leyendo docs/conventions.md: %w", err)
	}
	return parseSensitiveAreas(string(data)), nil
}

// ---- Ticket 02 (review_depth_by_diff_sensitivity, feature 8): computeTouchedPaths — diff de árbol contra el candidato congelado ----
//
// Building block puro: todavía ningún subcomando de `april` invoca estas
// funciones (eso llega en el ticket 03). Reusa runGit, el mismo seam que ya
// usa computeSubjectHash — sin un segundo mecanismo de subprocess/parseo de
// diff (spec, US24).

// gitEmptyTreeHash es el SHA-1 fijo y bien conocido del árbol vacío de git:
// no es un valor calculado por April, es una constante de git en sí (el
// árbol vacío siempre tiene este hash, sin importar el repositorio). Se usa
// como base de diff cuando el repositorio todavía no tiene ningún commit
// (HEAD no resuelve).
const gitEmptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// baseTreeForDiff calcula el árbol base contra el que se compara el
// candidato congelado en computeTouchedPaths: el árbol de HEAD si el
// repositorio ya tiene al menos un commit, o gitEmptyTreeHash si todavía no
// (HEAD no resuelve). computeSubjectHash ya validó antes que sí es un
// repositorio git válido, así que la única causa realista de que
// `git rev-parse --verify -q HEAD^{tree}` falle acá es un HEAD sin resolver
// — el caso normal de "primer commit todavía no existe" (spec, US7), no una
// falla que deba propagarse. Cualquier otro tipo de fallo de git en este
// punto sería indistinguible de ese caso por diseño (mismo criterio simple
// que ya aceptó la feature 7 para ErrNotGitRepo).
func baseTreeForDiff() (string, error) {
	stdout, _, err := runGit(nil, "rev-parse", "--verify", "-q", "HEAD^{tree}")
	if err != nil {
		return gitEmptyTreeHash, nil
	}
	return strings.TrimSpace(stdout), nil
}

// computeTouchedPaths calcula qué rutas cambiaron entre baseTreeForDiff()
// (el estado anterior del repositorio) y subjectHash (el árbol congelado
// que ya produjo computeSubjectHash, feature 7), corriendo
// `git diff --name-only <base> <subjectHash>` — comparación pura de árbol
// contra árbol, sin tocar índice ni working tree. Filtra las mismas dos
// rutas que computeSubjectHash ya excluye del árbol congelado
// (verifyLedgerPath, cualquier ruta con prefijo "progress/"), para que un
// proyecto donde esas rutas sí están commiteadas en HEAD no reporte un
// falso "cambio" solo por el propio mecanismo de exclusión del candidato
// (spec, US8). Devuelve la lista filtrada, normalizada a []string{} (nunca
// nil) cuando está vacía. Un fallo de git diff que no sea el ya cubierto en
// baseTreeForDiff (ej. corrupción de objetos) se propaga envuelto con
// contexto, sin fallback silencioso.
func computeTouchedPaths(subjectHash string) ([]string, error) {
	base, err := baseTreeForDiff()
	if err != nil {
		return nil, err
	}

	stdout, _, err := runGit(nil, "diff", "--name-only", base, subjectHash)
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only sobre el árbol congelado: %w", err)
	}

	touched := []string{}
	for _, line := range strings.Split(stdout, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if path == verifyLedgerPath || strings.HasPrefix(path, "progress/") {
			continue
		}
		touched = append(touched, path)
	}
	return touched, nil
}

// ---- Ticket 03 (review_depth_by_diff_sensitivity, feature 8): matchSensitiveAreas / reviewStartReport — ensamblaje final ----

// matchSensitiveAreas cruza touched (las rutas tocadas por el diff, spec
// computeTouchedPaths) contra sensitive (las áreas sensibles declaradas en
// docs/conventions.md, spec readSensitiveAreas) y devuelve todas las rutas
// de touched que coinciden con alguna de sensitive (US23: no solo la
// primera). Un área sensible terminada en "/" (ej. ".github/workflows/") se
// interpreta como prefijo de directorio; cualquier otra (ej. "scaffold.go")
// exige coincidencia exacta de ruta completa, sin substring ni prefijo
// parcial de nombre de archivo (US10). Devuelve []string{} (nunca nil)
// cuando ninguna ruta coincide.
func matchSensitiveAreas(touched, sensitive []string) []string {
	matched := []string{}
	for _, t := range touched {
		for _, s := range sensitive {
			if strings.HasSuffix(s, "/") {
				if strings.HasPrefix(t, s) {
					matched = append(matched, t)
					break
				}
			} else if t == s {
				matched = append(matched, t)
				break
			}
		}
	}
	return matched
}

// reviewStartReport es la salida de `review start --feature <id> --json`:
// subjectHash (igual que siempre), touchedPaths (computeTouchedPaths),
// sensitiveAreasTouched (matchSensitiveAreas) y extraReviewRequired
// (true si y solo si sensitiveAreasTouched no está vacío). Nombres de
// campo en camelCase, consistentes con statusReport/ledgerEntry — no se
// persiste en el ledger (spec, US19).
type reviewStartReport struct {
	SubjectHash           string   `json:"subjectHash"`
	TouchedPaths          []string `json:"touchedPaths"`
	SensitiveAreasTouched []string `json:"sensitiveAreasTouched"`
	ExtraReviewRequired   bool     `json:"extraReviewRequired"`
}
