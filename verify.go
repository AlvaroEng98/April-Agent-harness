package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// verifyLedgerPath es la ruta exacta (relativa a la raíz del árbol de
// trabajo) del ledger que escribe `april verify record` — se excluye del
// cálculo de hashTree para que el propio acto de anexarle una entrada no
// invalide el treeHash que esa misma entrada acaba de registrar (spec,
// "Cálculo del hash del árbol — hashTree").
const verifyLedgerPath = ".claude/verify-ledger.jsonl"

// hashTree calcula un hash agregado y determinístico del contenido de
// todo el árbol bajo fsys: sha256 del contenido de cada archivo, arma
// pares "ruta-relativa:hash", los ordena por ruta (para que el resultado
// no dependa del orden en que el filesystem devuelva las entradas) y
// calcula el sha256 del agregado completo.
//
// Excluye deliberadamente (ver specs/verify_record_ledger/spec.md,
// "Cálculo del hash del árbol", y specs/tree_hash_respects_gitignore/spec.md):
//   - cualquier ruta bajo .git/ (prefijo);
//   - las dos exclusiones fijas de fixedTreeExclusions (el propio ledger,
//     progress/), incondicionales, no dependan de estar en .gitignore;
//   - cualquier ruta que matchee un patrón de .gitignore (ticket 12/01),
//     cargado una sola vez antes de recorrer el árbol.
//
// Building-block puro, sin efectos secundarios ni CLI todavía — extrae a
// producción el mismo algoritmo que antes solo vivía como test helper
// (hashDirTree en status_test.go). Precedente exacto: hashContent en
// scaffold.go.
func hashTree(fsys fs.FS) (string, error) {
	patterns, err := loadGitignorePatterns(fsys)
	if err != nil {
		return "", fmt.Errorf("leyendo .gitignore: %w", err)
	}

	type entry struct {
		rel  string
		hash string
	}
	var entries []entry

	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isExcludedFromTreeHash(p, patterns) {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		entries = append(entries, entry{rel: p, hash: hashContent(data)})
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })

	var agg strings.Builder
	for _, e := range entries {
		agg.WriteString(e.rel)
		agg.WriteString(":")
		agg.WriteString(e.hash)
		agg.WriteString("\n")
	}
	sum := sha256.Sum256([]byte(agg.String()))
	return hex.EncodeToString(sum[:]), nil
}

// isExcludedFromTreeHash decide si una ruta relativa (con separador "/",
// tal como la devuelve fs.WalkDir sobre un fs.FS) queda fuera del cálculo
// de hashTree. Primero, incondicionales: .git/ y las dos exclusiones fijas
// de fixedTreeExclusions (review.go) — no dependen de estar en .gitignore
// (spec, US8/US23). Solo si ninguna de esas aplica, consulta
// gitignoreMatches contra los patrones ya cargados por loadGitignorePatterns
// (ticket 01, tree_hash_respects_gitignore).
func isExcludedFromTreeHash(rel string, patterns []gitignorePattern) bool {
	if rel == fixedTreeExclusions[0] {
		return true
	}
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	if rel == fixedTreeExclusions[1] || strings.HasPrefix(rel, fixedTreeExclusions[1]+"/") {
		return true
	}
	return gitignoreMatches(rel, patterns)
}

// ---- Ticket 01 (tree_hash_respects_gitignore, feature 12): parser de
// .gitignore en Go puro. hashTree/isExcludedFromTreeHash lo consumen
// (ticket 03). Mismo patrón de dos capas (función pura sobre string +
// wrapper de I/O sobre fs.FS) que parseSensitiveAreas/readSensitiveAreas
// en review.go.

// gitignorePattern es un patrón ya interpretado de una línea de
// .gitignore, listo para matchear contra una ruta relativa vía
// gitignoreMatches.
type gitignorePattern struct {
	anchored bool   // "/" al inicio o en medio: se compara desde la raíz
	dirOnly  bool   // terminaba en "/": excluye también todo lo que cuelgue debajo
	glob     string // patrón sin "/" inicial ni final, listo para path.Match
}

// parseGitignore interpreta el contenido ya leído de un .gitignore, línea
// por línea, en una lista de gitignorePattern. Ignora líneas vacías y
// comentarios (#). Las líneas de negación (!patrón) no están soportadas
// (ver spec, Out of Scope) — se ignoran sin producir ni error ni un
// patrón incorrecto, en vez de fallar silenciosamente con un resultado
// erróneo. Si una línea termina en "/", marca dirOnly y recorta esa
// barra. Si empieza con "/", o si (después de recortar esa barra inicial)
// todavía contiene "/", el patrón queda anchored (regla real de git:
// cualquier "/" que no sea el último ancla el patrón a la raíz). El
// resto, sin la barra inicial, queda como glob listo para path.Match.
func parseGitignore(content string) []gitignorePattern {
	var patterns []gitignorePattern

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			// Negación no soportada (spec, Out of Scope) — se ignora
			// deliberadamente en vez de producir un patrón incorrecto.
			continue
		}

		var p gitignorePattern

		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}

		anchored := strings.HasPrefix(line, "/")
		if anchored {
			line = strings.TrimPrefix(line, "/")
		}
		if strings.Contains(line, "/") {
			anchored = true
		}
		p.anchored = anchored
		p.glob = line

		patterns = append(patterns, p)
	}

	return patterns
}

// gitignoreMatches decide si rel (ruta relativa de un archivo, siempre
// con "/" como separador) matchea alguno de los patrones.
func gitignoreMatches(rel string, patterns []gitignorePattern) bool {
	for _, p := range patterns {
		if gitignorePatternMatches(rel, p) {
			return true
		}
	}
	return false
}

// gitignorePatternMatches aplica un único gitignorePattern contra rel.
func gitignorePatternMatches(rel string, p gitignorePattern) bool {
	if p.anchored {
		if ok, _ := path.Match(p.glob, rel); ok {
			return true
		}
		return p.dirOnly && strings.HasPrefix(rel, p.glob+"/")
	}
	// No anclado: p.glob es un único componente (nunca contiene "/", por
	// construcción de parseGitignore) — matchea el nombre de CUALQUIER
	// segmento de rel (archivo final o directorio intermedio), igual que
	// git trata un patrón sin "/" como "**/patrón" implícito.
	segments := strings.Split(rel, "/")
	for i, seg := range segments {
		ok, _ := path.Match(p.glob, seg)
		if !ok {
			continue
		}
		isLastSegment := i == len(segments)-1
		if !p.dirOnly || !isLastSegment {
			return true
		}
	}
	return false
}

// loadGitignorePatterns es el wrapper de I/O de parseGitignore: lee
// .gitignore de la raíz de fsys. Si no existe, no es un error — devuelve
// nil, nil (cero patrones extra), mismo criterio que readSensitiveAreas
// en review.go para docs/conventions.md ausente. Cualquier otro error de
// lectura se propaga envuelto.
func loadGitignorePatterns(fsys fs.FS) ([]gitignorePattern, error) {
	data, err := fs.ReadFile(fsys, ".gitignore")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("leyendo .gitignore: %w", err)
	}
	return parseGitignore(string(data)), nil
}

// ledgerEntry es una entrada del ledger de evidencia
// .claude/verify-ledger.jsonl (JSON Lines: una línea, un objeto JSON, sin
// pretty-print — ver specs/verify_record_ledger/spec.md, "Esquema del
// ledger"). kind queda como string libre desde ya: esta feature solo
// escribe "test", reservado para que la feature 6 agregue "review" al
// mismo archivo sin migrar el formato.
type ledgerEntry struct {
	Kind      string   `json:"kind"`
	FeatureID int      `json:"featureId"`
	Command   []string `json:"command"`
	ExitCode  int      `json:"exitCode"`
	TreeHash  string   `json:"treeHash"`
	Timestamp string   `json:"timestamp"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	Verdict   string   `json:"verdict,omitempty"` // nuevo — solo kind:"review" (feature 6)
	// SubjectHash — nuevo (feature 7, ticket 03): solo kind:"review" cuando
	// se registró con `review record --subject-hash <hash>`. omitempty para
	// que ninguna entrada existente (kind:"test", o kind:"review" sin el
	// flag nuevo) cambie de serialización.
	SubjectHash string `json:"subjectHash,omitempty"`
}

// appendLedgerEntry es la función pura de append: dado el contenido actual
// del ledger (bytes, posiblemente vacío si el archivo no existe todavía) y
// una ledgerEntry nueva, devuelve el contenido completo resultante con la
// línea nueva anexada al final, sin alterar ni reordenar ninguna línea
// previa. No toca disco — eso lo hace appendToLedger.
func appendLedgerEntry(current []byte, entry ledgerEntry) ([]byte, error) {
	line, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("serializando entrada del ledger: %w", err)
	}

	out := make([]byte, 0, len(current)+len(line)+1)
	out = append(out, current...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	out = append(out, line...)
	out = append(out, '\n')
	return out, nil
}

// appendToLedger es el wrapper de I/O de appendLedgerEntry sobre el ledger
// real en disco (verifyLedgerPath, relativo al cwd): lee el contenido
// actual con os.ReadFile (archivo inexistente se trata como contenido
// vacío, no como error — mismo criterio de "adopción" que loadManifest en
// scaffold.go), calcula el contenido resultante vía appendLedgerEntry, y lo
// escribe completo con writeFileAtomic (reusada tal cual de
// set_status.go, sin duplicar el patrón temp-then-rename).
func appendToLedger(entry ledgerEntry) error {
	current, err := os.ReadFile(verifyLedgerPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("leyendo %s: %w", verifyLedgerPath, err)
		}
		current = nil
	}

	out, err := appendLedgerEntry(current, entry)
	if err != nil {
		return err
	}

	if err := writeFileAtomic(verifyLedgerPath, out, 0644); err != nil {
		return err
	}
	return nil
}

// recordVerify corre cmdArgs[0] con cmdArgs[1:]... como argumentos, argv a
// argv y directo (sin sh -c implícito — ver spec, US24): quien necesite
// pipes/redirects/&& los pide explícito con "-- sh -c \"...\"". Mientras
// corre, su stdout/stderr se reflejan en vivo en la terminal de quien
// invocó (io.MultiWriter hacia os.Stdout/os.Stderr) y a la vez quedan
// capturados por separado en dos buffers para la entrada del ledger.
//
// Distingue dos clases de resultado (spec, "Orquestación — recordVerify"):
//   - el comando ni siquiera arrancó (binario inexistente, permiso
//     denegado — el error de cmd.Run() no es *exec.ExitError): devuelve
//     err != nil, no escribe ninguna entrada al ledger.
//   - el comando arrancó y terminó con exit code != 0 (*exec.ExitError):
//     es una corrida válida, se registra normalmente con ese exit code
//     real, err == nil.
//
// Tras la corrida (si arrancó), calcula hashTree(os.DirFS(".")), arma la
// ledgerEntry y hace el append atómico vía appendToLedger. exitCode
// devuelto es el del comando corrido, para que runVerifyRecord lo use
// como exit code del propio proceso `april verify record`.
func recordVerify(featureID int, cmdArgs []string) (entry ledgerEntry, exitCode int, err error) {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			// El comando ni siquiera arrancó: error de invocación, no se
			// escribe nada al ledger (US11 de la spec).
			return ledgerEntry{}, 0, fmt.Errorf("no se pudo ejecutar %q: %w", strings.Join(cmdArgs, " "), runErr)
		}
		exitCode = exitErr.ExitCode()
	}

	treeHash, err := hashTree(os.DirFS("."))
	if err != nil {
		return ledgerEntry{}, 0, fmt.Errorf("calculando hashTree tras correr el comando: %w", err)
	}

	entry = ledgerEntry{
		Kind:      "test",
		FeatureID: featureID,
		Command:   append([]string{}, cmdArgs...),
		ExitCode:  exitCode,
		TreeHash:  treeHash,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
	}

	if err := appendToLedger(entry); err != nil {
		return ledgerEntry{}, 0, fmt.Errorf("anexando al ledger: %w", err)
	}

	return entry, exitCode, nil
}

// runVerifyRecord contiene la lógica de `april verify record --feature <id>
// -- <comando>`: parseo simple y posicional (mismo estilo que
// runSetStatus/runStatus), invocación de recordVerify, y decisión del exit
// code del propio proceso. Cualquier falta en el parseo (--feature
// ausente/no numérico, separador -- ausente, o sin comando tras --) es un
// error de invocación explícito en stderr, exit distinto de cero, sin
// tocar el ledger — recordVerify nunca se llega a invocar en esos casos.
func runVerifyRecord(args []string) int {
	const usage = "Error: uso: april verify record --feature <id> -- <comando>"

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
	if len(args) < 3 || args[2] != "--" {
		fmt.Fprintln(os.Stderr, "Error: falta el separador -- entre --feature <id> y el comando a ejecutar")
		return 1
	}
	cmdArgs := args[3:]
	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: falta el comando a ejecutar después de --")
		return 1
	}

	_, exitCode, err := recordVerify(featureID, cmdArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return exitCode
}

// cmdVerify es el entry point del CLI para `april verify <subcomando>
// ...`. Hoy el único subcomando es record; cualquier otro es un error
// explícito.
func cmdVerify() {
	args := os.Args[2:]
	if len(args) == 0 || args[0] != "record" {
		fmt.Fprintln(os.Stderr, "Error: uso: april verify record --feature <id> -- <comando>")
		os.Exit(1)
	}
	os.Exit(runVerifyRecord(args[1:]))
}
