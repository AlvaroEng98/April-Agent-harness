package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// ---- Ticket 01: recordReview / runReviewRecord — comando completo ----

// reviewTestDir prepara un directorio temporal como cwd del proceso de test
// (chdirTemp) con .claude/ ya creado, mismo requisito que verifyTestDir en
// verify_test.go, para que appendToLedger tenga dónde escribir su archivo
// temporal.
func reviewTestDir(t *testing.T) string {
	t.Helper()
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	return dir
}

// TestRecordReviewVerdictoValidoAnexaEntradaConTreeHashYTimestamp cubre que
// un verdict válido queda anexado al ledger real en disco con kind
// "review", featureId correcto, verdict correcto, treeHash no vacío y
// timestamp parseable (RFC3339).
func TestRecordReviewVerdictoValidoAnexaEntradaConTreeHashYTimestamp(t *testing.T) {
	dir := reviewTestDir(t)

	entry, err := recordReview(6, verdictApproved)
	if err != nil {
		t.Fatalf("recordReview falló: %v", err)
	}
	if entry.Kind != "review" {
		t.Errorf("entry.Kind = %q, se esperaba %q", entry.Kind, "review")
	}
	if entry.FeatureID != 6 {
		t.Errorf("entry.FeatureID = %d, se esperaba 6", entry.FeatureID)
	}
	if entry.Verdict != verdictApproved {
		t.Errorf("entry.Verdict = %q, se esperaba %q", entry.Verdict, verdictApproved)
	}
	if entry.TreeHash == "" {
		t.Errorf("entry.TreeHash está vacío")
	}
	if _, err := time.Parse(time.RFC3339, entry.Timestamp); err != nil {
		t.Errorf("entry.Timestamp = %q no es parseable como RFC3339: %v", entry.Timestamp, err)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 {
		t.Fatalf("se esperaba 1 entrada en el ledger, hubo %d", len(entries))
	}
	if entries[0].Kind != "review" || entries[0].FeatureID != 6 || entries[0].Verdict != verdictApproved || entries[0].TreeHash == "" {
		t.Errorf("entrada en disco = %+v, no coincide con lo esperado", entries[0])
	}
}

// TestRecordReviewChangesRequestedSeRegistraConExitoNoEsError cubre US19/
// US20: registrar CHANGES_REQUESTED no es un error del comando, queda
// anexado al ledger igual que cualquier otro verdict válido.
func TestRecordReviewChangesRequestedSeRegistraConExitoNoEsError(t *testing.T) {
	dir := reviewTestDir(t)

	entry, err := recordReview(6, verdictChangesRequested)
	if err != nil {
		t.Fatalf("recordReview no debería fallar para CHANGES_REQUESTED: %v", err)
	}
	if entry.Verdict != verdictChangesRequested {
		t.Errorf("entry.Verdict = %q, se esperaba %q", entry.Verdict, verdictChangesRequested)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 || entries[0].Verdict != verdictChangesRequested {
		t.Fatalf("se esperaba 1 entrada con Verdict == CHANGES_REQUESTED en el ledger, hubo: %+v", entries)
	}
}

// TestRecordReviewVerdictoFueraDeVocabularioNoEscribeLedger cubre que un
// verdict fuera del vocabulario exacto (typo/sinónimo/minúsculas) es un
// error, y NO escribe ninguna entrada al ledger.
func TestRecordReviewVerdictoFueraDeVocabularioNoEscribeLedger(t *testing.T) {
	dir := reviewTestDir(t)

	_, err := recordReview(6, "aprobado")
	if err == nil {
		t.Fatalf("recordReview no devolvió error para un verdict fuera de vocabulario")
	}

	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado tras un verdict fuera de vocabulario (stat err = %v)", statErr)
	}
}

// TestRecordReviewDosCorridasProducenDosEntradas cubre US7: dos corridas
// sucesivas sobre el mismo featureId (CHANGES_REQUESTED seguido de
// APPROVED, simulando una ronda real de revisión) producen dos líneas en
// el ledger, ambas parseables, en orden, ninguna sobrescribe a la otra.
func TestRecordReviewDosCorridasProducenDosEntradas(t *testing.T) {
	dir := reviewTestDir(t)

	if _, err := recordReview(6, verdictChangesRequested); err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}
	if _, err := recordReview(6, verdictApproved); err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 2 {
		t.Fatalf("se esperaban 2 entradas en el ledger, hubo %d: %+v", len(entries), entries)
	}
	if entries[0].Verdict != verdictChangesRequested {
		t.Errorf("primera entrada Verdict = %q, se esperaba %q", entries[0].Verdict, verdictChangesRequested)
	}
	if entries[1].Verdict != verdictApproved {
		t.Errorf("segunda entrada Verdict = %q, se esperaba %q", entries[1].Verdict, verdictApproved)
	}
}

// TestRunReviewRecordFaltaFeatureEsErrorDeInvocacion cubre que invocar sin
// --feature es error de invocación explícito, exit distinto de cero, sin
// tocar el ledger.
func TestRunReviewRecordFaltaFeatureEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--verdict", "APPROVED"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de --feature")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordFaltaVerdictEsErrorDeInvocacion cubre que invocar sin
// --verdict es error de invocación explícito, sin tocar el ledger.
func TestRunReviewRecordFaltaVerdictEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "6"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de --verdict")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordFeatureNoNumericaEsErrorDeInvocacion cubre que un
// --feature con un valor no numérico es error de invocación explícito, sin
// tocar el ledger.
func TestRunReviewRecordFeatureNoNumericaEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "no-numerico", "--verdict", "APPROVED"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por --feature no numérico")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordVerdictFueraDeVocabularioEsErrorDeInvocacion cubre que
// un --verdict fuera del vocabulario exacto es error de invocación
// explícito, sin tocar el ledger.
func TestRunReviewRecordVerdictFueraDeVocabularioEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "6", "--verdict", "LGTM"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por --verdict fuera de vocabulario")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordOrdenDeFlagsInvertidoEsErrorDeInvocacion cubre que un
// orden de flags distinto al esperado (--verdict antes que --feature) es
// error de invocación explícito, sin tocar el ledger.
func TestRunReviewRecordOrdenDeFlagsInvertidoEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--verdict", "APPROVED", "--feature", "6"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por orden de flags inválido")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordArgumentosDeMasEsErrorDeInvocacion cubre que
// argumentos extra tras el valor de --verdict son error de invocación
// explícito, sin tocar el ledger.
func TestRunReviewRecordArgumentosDeMasEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "6", "--verdict", "APPROVED", "extra"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por argumentos de más")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordExitCodeCeroIndependienteDelVerdictRegistrado cubre
// US19: el exit code del proceso `review record` es 0 para los tres
// valores válidos, incluyendo CHANGES_REQUESTED — registrar un rechazo no
// es un fallo del comando.
func TestRunReviewRecordExitCodeCeroIndependienteDelVerdictRegistrado(t *testing.T) {
	verdicts := []string{verdictApproved, verdictApprovedWithObjection, verdictChangesRequested}
	for _, v := range verdicts {
		reviewTestDir(t)
		exitCode := runReviewRecord([]string{"--feature", "6", "--verdict", v})
		if exitCode != 0 {
			t.Errorf("verdict %q: exitCode = %d, se esperaba 0", v, exitCode)
		}
	}
}

// ---- Ticket 01: computeSubjectHash — candidato congelado sobre índice temporal de git ----

// gitRepoTestDir prepara un directorio temporal como cwd del proceso de
// test (chdirTemp) y lo inicializa como un repositorio git real (git init
// -q .) — no hace falta git config user.name/user.email porque git
// write-tree no necesita autor, a diferencia de un commit. Primer
// precedente del repo de un test que depende de tener git instalado en el
// entorno donde corre go test (spec, US30).
func gitRepoTestDir(t *testing.T) string {
	t.Helper()
	dir := chdirTemp(t)
	if out, err := exec.Command("git", "init", "-q", ".").CombinedOutput(); err != nil {
		t.Fatalf("no se pudo inicializar el repositorio git de prueba: %v\n%s", err, out)
	}
	return dir
}

// TestComputeSubjectHashDeterministicoMismoArbolMismoHash cubre US5: dos
// corridas sucesivas sin cambiar nada en el árbol dan el mismo subject_hash.
func TestComputeSubjectHashDeterministicoMismoArbolMismoHash(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("no se pudo crear el subdirectorio del fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("mundo"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	first, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}
	if first == "" {
		t.Fatalf("subject_hash vacío")
	}
	second, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}
	if first != second {
		t.Errorf("first = %q, second = %q, se esperaba el mismo hash sin cambios en el árbol", first, second)
	}
}

// TestComputeSubjectHashCambiaSiElArbolCambia cubre US6: modificar un
// archivo no excluido entre dos corridas cambia el subject_hash.
func TestComputeSubjectHashCambiaSiElArbolCambia(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	first, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola cambiado"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	second, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}
	if first == second {
		t.Errorf("el subject_hash no cambió tras modificar un archivo no excluido")
	}
}

// TestComputeSubjectHashExcluyeLedgerYProgress es la regresión directa del
// problema que la feature 5 ya resolvió para hashTree, ahora sobre el
// mecanismo de git: modificar/crear .claude/verify-ledger.jsonl o
// cualquier archivo bajo progress/ entre dos corridas no cambia el hash.
func TestComputeSubjectHashExcluyeLedgerYProgress(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	first, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, verifyLedgerPath), []byte(`{"kind":"test"}`+"\n"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el ledger: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "progress"), 0755); err != nil {
		t.Fatalf("no se pudo crear progress/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "progress", "current.md"), []byte("bitácora"), 0644); err != nil {
		t.Fatalf("no se pudo escribir progress/current.md: %v", err)
	}

	second, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}
	if first != second {
		t.Errorf("first = %q, second = %q, se esperaba el mismo hash tras tocar solo el ledger y progress/", first, second)
	}
}

// ---- Ticket 02 (tree_hash_respects_gitignore, feature 12): confirmación de que computeSubjectHash ya respeta .gitignore ----

// TestComputeSubjectHashYaRespetaGitignoreParaArchivosNoTrackeados confirma
// (no corrige, porque no hay nada que corregir) que computeSubjectHash ya
// respeta .gitignore de forma nativa para archivos untracked, gracias al
// comportamiento propio de `git add -A`: un archivo gitignoreado que nunca
// se agrega al índice real no participa del árbol congelado, con o sin esta
// feature (spec, Solution).
func TestComputeSubjectHashYaRespetaGitignoreParaArchivosNoTrackeados(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("/HarnessInit\n"), 0644); err != nil {
		t.Fatalf("no se pudo escribir .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "HarnessInit"), []byte("contenido A"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture gitignoreado: %v", err)
	}

	first, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "HarnessInit"), []byte("contenido B, distinto"), 0644); err != nil {
		t.Fatalf("no se pudo sobrescribir el fixture gitignoreado: %v", err)
	}

	second, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("segunda corrida falló: %v", err)
	}
	if first != second {
		t.Errorf("first = %q, second = %q, se esperaba el mismo hash: HarnessInit está gitignoreado y nunca fue trackeado, git add -A no debería tocarlo", first, second)
	}
}

// TestComputeSubjectHashFallaSiNoEsRepositorioGit cubre que correr
// computeSubjectHash fuera de un repositorio git devuelve un error que
// envuelve ErrNotGitRepo.
func TestComputeSubjectHashFallaSiNoEsRepositorioGit(t *testing.T) {
	chdirTemp(t)

	_, err := computeSubjectHash()
	if !errors.Is(err, ErrNotGitRepo) {
		t.Fatalf("error = %v, se esperaba que envolviera ErrNotGitRepo", err)
	}
}

// TestComputeSubjectHashFallaSiGitNoEstaEnPath cubre US29: con PATH vacío
// (git no disponible), computeSubjectHash devuelve un error que envuelve
// ErrNotGitRepo, sin panic.
func TestComputeSubjectHashFallaSiGitNoEstaEnPath(t *testing.T) {
	gitRepoTestDir(t)
	t.Setenv("PATH", "")

	_, err := computeSubjectHash()
	if !errors.Is(err, ErrNotGitRepo) {
		t.Fatalf("error = %v, se esperaba que envolviera ErrNotGitRepo", err)
	}
}

// TestComputeSubjectHashNoMutaElIndiceReal cubre US9/US27: hacer git add
// manual de un archivo (staging real del usuario) antes de llamar a
// computeSubjectHash no altera el índice real — git diff --cached sigue
// igual antes y después de la llamada.
func TestComputeSubjectHashNoMutaElIndiceReal(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}
	if out, err := exec.Command("git", "add", "a.txt").CombinedOutput(); err != nil {
		t.Fatalf("no se pudo stagear a.txt en el índice real: %v\n%s", err, out)
	}

	before, err := exec.Command("git", "diff", "--cached", "--name-only").CombinedOutput()
	if err != nil {
		t.Fatalf("git diff --cached falló: %v", err)
	}

	if _, err := computeSubjectHash(); err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	after, err := exec.Command("git", "diff", "--cached", "--name-only").CombinedOutput()
	if err != nil {
		t.Fatalf("git diff --cached falló: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("el índice real cambió: antes=%q después=%q", before, after)
	}
}

// TestComputeSubjectHashNoDejaArchivoTemporalHuerfano cubre US28: no queda
// ningún archivo april-subject-index-* huérfano en el directorio temporal
// del sistema tras terminar, tanto en el camino de éxito como forzando un
// error (acá, quitando permiso de escritura a .git/objects DESPUÉS de que
// el índice temporal ya se creó, para que la falla ocurra dentro de git
// add -A/git write-tree, no antes).
func TestComputeSubjectHashNoDejaArchivoTemporalHuerfano(t *testing.T) {
	countIndexFiles := func() int {
		matches, err := filepath.Glob(filepath.Join(os.TempDir(), "april-subject-index-*"))
		if err != nil {
			t.Fatalf("no se pudo listar el directorio temporal del sistema: %v", err)
		}
		return len(matches)
	}

	baseline := countIndexFiles()

	gitRepoTestDir(t)
	if _, err := computeSubjectHash(); err != nil {
		t.Fatalf("computeSubjectHash falló en el camino de éxito: %v", err)
	}
	if got := countIndexFiles(); got != baseline {
		t.Errorf("quedaron %d archivos april-subject-index-* huérfanos tras el camino de éxito (base %d)", got, baseline)
	}

	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}
	objectsDir := filepath.Join(dir, ".git", "objects")
	if err := os.Chmod(objectsDir, 0555); err != nil {
		t.Fatalf("no se pudo quitar permiso de escritura a .git/objects: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(objectsDir, 0755); err != nil {
			t.Fatalf("no se pudo restaurar el permiso de .git/objects: %v", err)
		}
	})

	if _, err := computeSubjectHash(); err == nil {
		t.Fatalf("se esperaba error al no poder escribir objetos git")
	}
	if got := countIndexFiles(); got != baseline {
		t.Errorf("quedaron %d archivos april-subject-index-* huérfanos tras el camino de error (base %d)", got, baseline)
	}
}

// ---- Ticket 03 (review_depth_by_diff_sensitivity, feature 8): matchSensitiveAreas ----

// TestMatchSensitiveAreasPrefijoDeDirectorio cubre que un área sensible
// terminada en "/" hace match de cualquier ruta tocada dentro de ese
// directorio (prefijo), no solo coincidencia exacta.
func TestMatchSensitiveAreasPrefijoDeDirectorio(t *testing.T) {
	touched := []string{".github/workflows/release.yml"}
	sensitive := []string{".github/workflows/"}

	got := matchSensitiveAreas(touched, sensitive)
	if len(got) != 1 || got[0] != ".github/workflows/release.yml" {
		t.Errorf("got = %v, se esperaba [%q]", got, ".github/workflows/release.yml")
	}
}

// TestMatchSensitiveAreasCoincidenciaExactaDeArchivo cubre US10: un área
// sensible sin "/" exige coincidencia exacta de ruta completa —
// scaffold_test.go no hace match contra scaffold.go.
func TestMatchSensitiveAreasCoincidenciaExactaDeArchivo(t *testing.T) {
	got := matchSensitiveAreas([]string{"scaffold.go"}, []string{"scaffold.go"})
	if len(got) != 1 || got[0] != "scaffold.go" {
		t.Errorf("got = %v, se esperaba [%q] para coincidencia exacta", got, "scaffold.go")
	}

	got = matchSensitiveAreas([]string{"scaffold_test.go"}, []string{"scaffold.go"})
	if len(got) != 0 {
		t.Errorf("got = %v, se esperaba vacío (scaffold_test.go no coincide con scaffold.go)", got)
	}
}

// TestMatchSensitiveAreasDevuelveTodasLasCoincidencias cubre US23: dos rutas
// tocadas, ambas sensibles, aparecen ambas en el resultado, no solo la
// primera.
func TestMatchSensitiveAreasDevuelveTodasLasCoincidencias(t *testing.T) {
	touched := []string{"scaffold.go", ".github/workflows/release.yml"}
	sensitive := []string{"scaffold.go", ".github/workflows/"}

	got := matchSensitiveAreas(touched, sensitive)
	if len(got) != 2 {
		t.Fatalf("got = %v, se esperaban 2 coincidencias", got)
	}
	want := map[string]bool{"scaffold.go": true, ".github/workflows/release.yml": true}
	for _, p := range got {
		if !want[p] {
			t.Errorf("ruta inesperada en el resultado: %q", p)
		}
	}
}

// TestMatchSensitiveAreasVacioSinCoincidencias cubre que, sin ninguna ruta
// tocada coincidente, matchSensitiveAreas devuelve []string{} (no nil).
func TestMatchSensitiveAreasVacioSinCoincidencias(t *testing.T) {
	got := matchSensitiveAreas([]string{"otra_cosa.go"}, []string{"scaffold.go", "init.sh", ".github/workflows/"})
	if got == nil {
		t.Fatalf("got = nil, se esperaba []string{} no nil")
	}
	if len(got) != 0 {
		t.Errorf("got = %v, se esperaba vacío", got)
	}
}

// ---- Ticket 03 (review_depth_by_diff_sensitivity, feature 8): runReviewStart extendido con --json ----

// TestRunReviewStartSinJsonMantieneSalidaDeSoloHash reusa el
// fixture/aserciones de TestRunReviewStartImprimeSubjectHashEnStdout
// (feature 7) sobre un repo con commit y un cambio real tocando
// scaffold.go; verifica que, SIN --json, la salida sigue siendo una sola
// línea con el hash, sin ningún rastro de touchedPaths ni
// extraReviewRequired (US12/US13).
func TestRunReviewStartSinJsonMantieneSalidaDeSoloHash(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{"scaffold.go": "package main\n"})
	if err := os.WriteFile(filepath.Join(dir, "scaffold.go"), []byte("package main\n// cambiado\n"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runReviewStart([]string{"--feature", "8"})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
	}

	trimmed := strings.TrimRight(out, "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 1 {
		t.Fatalf("se esperaba una sola línea en stdout, hubo %d: %q", len(lines), out)
	}
	if !subjectHashHexPattern.MatchString(lines[0]) {
		t.Errorf("stdout = %q, se esperaba un hex de SHA-1 de 40 caracteres sin texto decorativo", lines[0])
	}
	if strings.Contains(out, "touchedPaths") || strings.Contains(out, "extraReviewRequired") {
		t.Errorf("stdout = %q, no debería contener rastro de touchedPaths/extraReviewRequired sin --json", out)
	}
}

// TestRunReviewStartJsonReportaTouchedPathsYExtraReviewRequiredTrue cubre el
// caso central de la feature: con docs/conventions.md conteniendo la
// sección real de "Áreas sensibles", un commit baseline y un cambio
// posterior tocando scaffold.go, --json reporta ExtraReviewRequired ==
// true, TouchedPaths y SensitiveAreasTouched conteniendo "scaffold.go", y
// SubjectHash no vacío.
func TestRunReviewStartJsonReportaTouchedPathsYExtraReviewRequiredTrue(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{
		"scaffold.go":         "package main\n",
		"docs/conventions.md": sensitiveAreasSectionFixture,
	})
	if err := os.WriteFile(filepath.Join(dir, "scaffold.go"), []byte("package main\n// cambiado\n"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runReviewStart([]string{"--feature", "8", "--json"})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
	}

	var report reviewStartReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("no se pudo parsear stdout como JSON: %v\nstdout = %q", err, out)
	}
	if !report.ExtraReviewRequired {
		t.Errorf("ExtraReviewRequired = false, se esperaba true")
	}
	if !containsString(report.TouchedPaths, "scaffold.go") {
		t.Errorf("TouchedPaths = %v, se esperaba que contuviera %q", report.TouchedPaths, "scaffold.go")
	}
	if !containsString(report.SensitiveAreasTouched, "scaffold.go") {
		t.Errorf("SensitiveAreasTouched = %v, se esperaba que contuviera %q", report.SensitiveAreasTouched, "scaffold.go")
	}
	if report.SubjectHash == "" {
		t.Errorf("SubjectHash está vacío")
	}
}

// containsString es un helper mínimo de búsqueda lineal para los tests de
// runReviewStart --json.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// TestRunReviewStartJsonExtraReviewRequiredFalseSiNoTocaAreaSensible cubre
// que, con el mismo fixture de docs/conventions.md, un cambio que toca un
// archivo no sensible da ExtraReviewRequired == false y
// SensitiveAreasTouched vacío, mientras TouchedPaths sí contiene ese
// archivo.
func TestRunReviewStartJsonExtraReviewRequiredFalseSiNoTocaAreaSensible(t *testing.T) {
	gitRepoWithCommitTestDir(t, map[string]string{
		"docs/conventions.md": sensitiveAreasSectionFixture,
	})
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("no se pudo obtener el cwd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "otra_cosa.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runReviewStart([]string{"--feature", "8", "--json"})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
	}

	var report reviewStartReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("no se pudo parsear stdout como JSON: %v\nstdout = %q", err, out)
	}
	if report.ExtraReviewRequired {
		t.Errorf("ExtraReviewRequired = true, se esperaba false")
	}
	if len(report.SensitiveAreasTouched) != 0 {
		t.Errorf("SensitiveAreasTouched = %v, se esperaba vacío", report.SensitiveAreasTouched)
	}
	if !containsString(report.TouchedPaths, "otra_cosa.go") {
		t.Errorf("TouchedPaths = %v, se esperaba que contuviera %q", report.TouchedPaths, "otra_cosa.go")
	}
}

// TestRunReviewStartJsonSinSeccionDeAreasSensiblesSiempreFalse cubre US11:
// con docs/conventions.md sin la sección (o ausente), un cambio tocando
// scaffold.go da ExtraReviewRequired == false sin error.
func TestRunReviewStartJsonSinSeccionDeAreasSensiblesSiempreFalse(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{"scaffold.go": "package main\n"})
	if err := os.WriteFile(filepath.Join(dir, "scaffold.go"), []byte("package main\n// cambiado\n"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runReviewStart([]string{"--feature", "8", "--json"})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
	}

	var report reviewStartReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("no se pudo parsear stdout como JSON: %v\nstdout = %q", err, out)
	}
	if report.ExtraReviewRequired {
		t.Errorf("ExtraReviewRequired = true, se esperaba false sin sección de Áreas sensibles")
	}
}

// TestRunReviewStartJsonEsJsonValidoConCamposNuncaNulos cubre US14: sobre un
// caso sin cambios, touchedPaths y sensitiveAreasTouched deserializan como
// arreglo vacío, nunca como null.
func TestRunReviewStartJsonEsJsonValidoConCamposNuncaNulos(t *testing.T) {
	gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	out, err := captureStdout(t, func() error {
		exitCode := runReviewStart([]string{"--feature", "8", "--json"})
		if exitCode != 0 {
			t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("no se pudo parsear stdout como JSON: %v\nstdout = %q", err, out)
	}
	if string(raw["touchedPaths"]) != "[]" {
		t.Errorf("touchedPaths crudo = %s, se esperaba []", raw["touchedPaths"])
	}
	if string(raw["sensitiveAreasTouched"]) != "[]" {
		t.Errorf("sensitiveAreasTouched crudo = %s, se esperaba []", raw["sensitiveAreasTouched"])
	}
}

// TestRunReviewStartJsonFueraDeRepositorioGitFallaExplicito cubre US15: sin
// git init, --feature <id> --json falla con exit≠0 y stderr menciona
// explícitamente que no es un repositorio git.
func TestRunReviewStartJsonFueraDeRepositorioGitFallaExplicito(t *testing.T) {
	chdirTemp(t)

	var exitCode int
	stderr := captureStderr(t, func() {
		exitCode = runReviewStart([]string{"--feature", "8", "--json"})
	})

	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 fuera de un repositorio git")
	}
	if !strings.Contains(stderr, "no es un repositorio git") {
		t.Errorf("stderr = %q, se esperaba que mencionara explícitamente que no es un repositorio git", stderr)
	}
}

// TestRunReviewStartArgumentosDeMasConJsonEsErrorDeInvocacion cubre que
// ["--feature", "<id>", "--json", "extra"] es error de invocación explícito.
func TestRunReviewStartArgumentosDeMasConJsonEsErrorDeInvocacion(t *testing.T) {
	chdirTemp(t)

	exitCode := runReviewStart([]string{"--feature", "8", "--json", "extra"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por argumentos de más tras --json")
	}
}

// TestRunReviewStartFlagDesconocidoTrasFeatureEsErrorDeInvocacion cubre que
// ["--feature", "<id>", "--jason"] (typo) es error de invocación explícito.
func TestRunReviewStartFlagDesconocidoTrasFeatureEsErrorDeInvocacion(t *testing.T) {
	chdirTemp(t)

	exitCode := runReviewStart([]string{"--feature", "8", "--jason"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por flag desconocido (typo de --json)")
	}
}

// ---- Ticket 02: runReviewStart — comando `april review start --feature <id>` ----

var subjectHashHexPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// TestRunReviewStartImprimeSubjectHashEnStdout cubre US14/US15: en un
// repositorio git, exit 0 y una sola línea no vacía en stdout con el
// subject_hash (hex de SHA-1, 40 caracteres), sin texto decorativo alrededor.
func TestRunReviewStartImprimeSubjectHashEnStdout(t *testing.T) {
	gitRepoTestDir(t)

	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runReviewStart([]string{"--feature", "7"})
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0", exitCode)
	}

	trimmed := strings.TrimRight(out, "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) != 1 {
		t.Fatalf("se esperaba una sola línea en stdout, hubo %d: %q", len(lines), out)
	}
	if !subjectHashHexPattern.MatchString(lines[0]) {
		t.Errorf("stdout = %q, se esperaba un hex de SHA-1 de 40 caracteres sin texto decorativo", lines[0])
	}
}

// TestRunReviewStartNoEscribeNadaAlLedger cubre que review start es una
// consulta pura: no crea .claude/verify-ledger.jsonl.
func TestRunReviewStartNoEscribeNadaAlLedger(t *testing.T) {
	dir := gitRepoTestDir(t)

	if _, err := captureStdout(t, func() error {
		runReviewStart([]string{"--feature", "7"})
		return nil
	}); err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado por review start (stat err = %v)", statErr)
	}
}

// TestRunReviewStartFaltaFeatureEsErrorDeInvocacion cubre que invocar sin
// --feature es error de invocación explícito, exit≠0, sin necesitar un
// repositorio git para fallar (nunca llega a computeSubjectHash).
func TestRunReviewStartFaltaFeatureEsErrorDeInvocacion(t *testing.T) {
	chdirTemp(t)

	exitCode := runReviewStart([]string{})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de --feature")
	}
}

// TestRunReviewStartFaltaValorDeFeatureEsErrorDeInvocacion cubre que
// --feature sin valor es error de invocación explícito.
func TestRunReviewStartFaltaValorDeFeatureEsErrorDeInvocacion(t *testing.T) {
	chdirTemp(t)

	exitCode := runReviewStart([]string{"--feature"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de valor de --feature")
	}
}

// TestRunReviewStartFeatureNoNumericaEsErrorDeInvocacion cubre que un
// --feature con valor no numérico es error de invocación explícito.
func TestRunReviewStartFeatureNoNumericaEsErrorDeInvocacion(t *testing.T) {
	chdirTemp(t)

	exitCode := runReviewStart([]string{"--feature", "no-numerico"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por --feature no numérico")
	}
}

// TestRunReviewStartFueraDeRepositorioGitFallaExplicito cubre US10: fuera
// de un repositorio git, con --feature válido, exit≠0 y stderr menciona
// explícitamente que no es un repositorio git.
func TestRunReviewStartFueraDeRepositorioGitFallaExplicito(t *testing.T) {
	chdirTemp(t)

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("no se pudo crear el pipe: %v", err)
	}
	os.Stderr = w

	exitCode := runReviewStart([]string{"--feature", "7"})

	w.Close()
	os.Stderr = origStderr
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	stderr := buf.String()

	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 fuera de un repositorio git")
	}
	if !strings.Contains(stderr, "no es un repositorio git") {
		t.Errorf("stderr = %q, se esperaba que mencionara explícitamente que no es un repositorio git", stderr)
	}
}

// ---- Ticket 03: recordReviewWithSubjectHash / runReviewRecord --subject-hash ----

// captureStderr redirige os.Stderr durante fn y devuelve lo capturado, mismo
// patrón que ya usa TestRunReviewStartFueraDeRepositorioGitFallaExplicito,
// extraído a helper para no repetirlo en los tests de este ticket.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("no se pudo crear el pipe: %v", err)
	}
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = origStderr
	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	return buf.String()
}

// TestRecordReviewWithSubjectHashVigenteAdmiteNormalmente cubre US4: un
// subject_hash que coincide con el candidato recalculado en el momento se
// registra normalmente, con SubjectHash == hash, TreeHash no vacío y el
// Verdict correcto.
func TestRecordReviewWithSubjectHashVigenteAdmiteNormalmente(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	hash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	entry, err := recordReviewWithSubjectHash(6, verdictApproved, hash)
	if err != nil {
		t.Fatalf("recordReviewWithSubjectHash falló: %v", err)
	}
	if entry.SubjectHash != hash {
		t.Errorf("entry.SubjectHash = %q, se esperaba %q", entry.SubjectHash, hash)
	}
	if entry.TreeHash == "" {
		t.Errorf("entry.TreeHash está vacío")
	}
	if entry.Verdict != verdictApproved {
		t.Errorf("entry.Verdict = %q, se esperaba %q", entry.Verdict, verdictApproved)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 || entries[0].SubjectHash != hash {
		t.Fatalf("se esperaba 1 entrada en el ledger con SubjectHash == %q, hubo: %+v", hash, entries)
	}
}

// TestRecordReviewWithSubjectHashStaleEsRechazado cubre US3: un subject_hash
// desactualizado (el árbol cambió desde que se calculó) rechaza el registro
// con un error que envuelve ErrStaleSubjectHash y contiene la substring
// literal "stale subject_hash", sin tocar el ledger.
func TestRecordReviewWithSubjectHashStaleEsRechazado(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	staleHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola cambiado"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	_, err = recordReviewWithSubjectHash(6, verdictApproved, staleHash)
	if !errors.Is(err, ErrStaleSubjectHash) {
		t.Fatalf("error = %v, se esperaba que envolviera ErrStaleSubjectHash", err)
	}
	if !strings.Contains(err.Error(), "stale subject_hash") {
		t.Errorf("error = %q, se esperaba que contuviera la substring literal %q", err.Error(), "stale subject_hash")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado tras un subject_hash stale (stat err = %v)", statErr)
	}
}

// TestRecordReviewWithSubjectHashVerdictFueraDeVocabularioNoEscribeLedger
// cubre US19: un verdict fuera de vocabulario se rechaza igual que en
// recordReview, sin tocar el ledger, incluso con un subjectHash cualquiera.
func TestRecordReviewWithSubjectHashVerdictFueraDeVocabularioNoEscribeLedger(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	_, err := recordReviewWithSubjectHash(6, "aprobado", "cualquier-hash")
	if err == nil {
		t.Fatalf("recordReviewWithSubjectHash no devolvió error para un verdict fuera de vocabulario")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado tras un verdict fuera de vocabulario (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordConSubjectHashVigenteAdmiteNormalmente cubre la
// integración completa vía runReviewRecord con --subject-hash vigente.
func TestRunReviewRecordConSubjectHashVigenteAdmiteNormalmente(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	hash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	exitCode := runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED", "--subject-hash", hash})
	if exitCode != 0 {
		t.Errorf("exitCode = %d, se esperaba 0", exitCode)
	}
}

// TestRunReviewRecordConSubjectHashStaleRechazaConExitDistintoDeCero cubre
// que un --subject-hash desactualizado rechaza el registro vía
// runReviewRecord, exit≠0, sin tocar el ledger.
func TestRunReviewRecordConSubjectHashStaleRechazaConExitDistintoDeCero(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	staleHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola cambiado"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	exitCode := runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED", "--subject-hash", staleHash})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por subject_hash stale")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado tras un subject_hash stale (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordSubjectHashSinValorEsErrorDeInvocacion cubre que
// --subject-hash como último argumento sin valor es error de invocación
// explícito, exit≠0, sin tocar el ledger.
func TestRunReviewRecordSubjectHashSinValorEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED", "--subject-hash"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por --subject-hash sin valor")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordArgumentosDeMasTrasSubjectHashEsErrorDeInvocacion cubre
// que argumentos extra después del valor de --subject-hash son error de
// invocación explícito, exit≠0, sin tocar el ledger.
func TestRunReviewRecordArgumentosDeMasTrasSubjectHashEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED", "--subject-hash", "abc123", "extra"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por argumentos de más tras --subject-hash <hash>")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunReviewRecordFueraDeRepositorioGitConSubjectHashFallaExplicito cubre
// US11: fuera de un repositorio git, con --subject-hash presente, el
// comando falla explícito, exit≠0, stderr menciona que no es un
// repositorio git.
func TestRunReviewRecordFueraDeRepositorioGitConSubjectHashFallaExplicito(t *testing.T) {
	chdirTemp(t)

	var exitCode int
	stderr := captureStderr(t, func() {
		exitCode = runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED", "--subject-hash", "abc123"})
	})

	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 fuera de un repositorio git")
	}
	if !strings.Contains(stderr, "no es un repositorio git") {
		t.Errorf("stderr = %q, se esperaba que mencionara explícitamente que no es un repositorio git", stderr)
	}
}

// TestRunReviewRecordSinSubjectHashFuncionaFueraDeUnRepositorioGit cubre
// US12/US26: el camino sin --subject-hash sigue funcionando fuera de un
// repositorio git, exactamente como antes de esta feature.
func TestRunReviewRecordSinSubjectHashFuncionaFueraDeUnRepositorioGit(t *testing.T) {
	chdirTemp(t)
	if err := os.MkdirAll(".claude", 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	exitCode := runReviewRecord([]string{"--feature", "7", "--verdict", "APPROVED"})
	if exitCode != 0 {
		t.Errorf("exitCode = %d, se esperaba 0 fuera de un repositorio git sin --subject-hash", exitCode)
	}
}

// ---- Ticket 01 (review_depth_by_diff_sensitivity, feature 8): parseSensitiveAreas / readSensitiveAreas ----

// sensitiveAreasSectionFixture es el contenido literal real de la sección
// "## Áreas sensibles" de docs/conventions.md de este repo (confirmada con
// el humano el 26/08/2026): las tres rutas scaffold.go, init.sh,
// .github/workflows/, en ese orden.
const sensitiveAreasSectionFixture = "## Áreas sensibles\n\n" +
	"Precondición de la feature `review_depth_by_diff_sensitivity`\n" +
	"(`feature_list.json` id 8, `ROADMAP.md` E5): rutas cuyo blast radius exige\n" +
	"revisión más profunda por parte de `reviewer_agent`, confirmadas con el\n" +
	"humano el 26/08/2026:\n\n" +
	"- `scaffold.go` — el motor que aplica cambios sobre el filesystem del\n" +
	"  usuario; un bug aquí puede borrar o sobrescribir trabajo real.\n" +
	"- `init.sh` — es lo que valida que el entorno es confiable antes de que se\n" +
	"  confíe en él; un bug aquí deja pasar un entorno roto sin avisar.\n" +
	"- `.github/workflows/` — dispara releases automáticas; un bug aquí puede\n" +
	"  publicar una versión rota o filtrar algo indebido en CI.\n\n" +
	"Cualquier diff que toque una de estas rutas exige el paso adicional de\n" +
	"revisión que defina la feature 8 al implementarse. Fuera de estas tres,\n" +
	"no se exige profundidad extra por defecto.\n"

// TestParseSensitiveAreasExtraeRutasDeLaSeccion cubre que parseSensitiveAreas
// sobre el contenido literal real de la sección de docs/conventions.md de
// este repo devuelve exactamente las tres rutas esperadas, en ese orden.
func TestParseSensitiveAreasExtraeRutasDeLaSeccion(t *testing.T) {
	got := parseSensitiveAreas(sensitiveAreasSectionFixture)
	want := []string{"scaffold.go", "init.sh", ".github/workflows/"}
	if len(got) != len(want) {
		t.Fatalf("got = %v, want = %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q (got completo = %v)", i, got[i], want[i], got)
		}
	}
}

// TestParseSensitiveAreasVacioSiSeccionAusente cubre que un contenido con
// otras secciones pero sin "## Áreas sensibles" devuelve []string{} (no
// nil, sin error — la firma no devuelve error).
func TestParseSensitiveAreasVacioSiSeccionAusente(t *testing.T) {
	content := "## Otra sección\n\n- `algo.go` — nota.\n"
	got := parseSensitiveAreas(content)
	if got == nil {
		t.Fatalf("got = nil, se esperaba []string{} no nil")
	}
	if len(got) != 0 {
		t.Errorf("got = %v, se esperaba vacío", got)
	}
}

// TestParseSensitiveAreasSeDetieneEnElSiguienteEncabezado cubre que los
// ítems de una sección posterior al siguiente "## " no se cuelan en el
// resultado.
func TestParseSensitiveAreasSeDetieneEnElSiguienteEncabezado(t *testing.T) {
	content := "## Áreas sensibles\n\n" +
		"- `scaffold.go` — motor de aplicación.\n\n" +
		"## Otra sección posterior\n\n" +
		"- `no_deberia_aparecer.go` — no pertenece a Áreas sensibles.\n"

	got := parseSensitiveAreas(content)
	want := []string{"scaffold.go"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got = %v, want = %v", got, want)
	}
}

// TestParseSensitiveAreasIgnoraTextoSinBackticks cubre que un ítem de lista
// sin ruta entre backticks (ej. una nota aclaratoria) no aporta una
// entrada vacía o basura al resultado.
func TestParseSensitiveAreasIgnoraTextoSinBackticks(t *testing.T) {
	content := "## Áreas sensibles\n\n" +
		"- `scaffold.go` — motor de aplicación.\n" +
		"- Nota aclaratoria sin ruta entre backticks.\n" +
		"- `init.sh` — validación de entorno.\n"

	got := parseSensitiveAreas(content)
	want := []string{"scaffold.go", "init.sh"}
	if len(got) != len(want) {
		t.Fatalf("got = %v, want = %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q (got completo = %v)", i, got[i], want[i], got)
		}
	}
}

// TestReadSensitiveAreasLeeDocsConventions cubre que readSensitiveAreas
// sobre un fstest.MapFS con docs/conventions.md sintético conteniendo la
// sección devuelve la lista esperada.
func TestReadSensitiveAreasLeeDocsConventions(t *testing.T) {
	fsys := fstest.MapFS{
		"docs/conventions.md": &fstest.MapFile{Data: []byte(sensitiveAreasSectionFixture)},
	}

	got, err := readSensitiveAreas(fsys)
	if err != nil {
		t.Fatalf("readSensitiveAreas falló: %v", err)
	}
	want := []string{"scaffold.go", "init.sh", ".github/workflows/"}
	if len(got) != len(want) {
		t.Fatalf("got = %v, want = %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q (got completo = %v)", i, got[i], want[i], got)
		}
	}
}

// TestReadSensitiveAreasArchivoAusenteNoEsError cubre que readSensitiveAreas
// sobre un fstest.MapFS sin esa ruta devuelve err == nil y lista vacía (no
// falla por archivo ausente).
func TestReadSensitiveAreasArchivoAusenteNoEsError(t *testing.T) {
	fsys := fstest.MapFS{}

	got, err := readSensitiveAreas(fsys)
	if err != nil {
		t.Fatalf("readSensitiveAreas devolvió error para archivo ausente: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got = %v, se esperaba vacío", got)
	}
}

// ---- Ticket 02 (review_depth_by_diff_sensitivity, feature 8): computeTouchedPaths — diff de árbol contra el candidato congelado ----

// gitRepoWithCommitTestDir extiende gitRepoTestDir (feature 7): además de
// inicializar el repositorio, escribe los archivos indicados y hace un
// commit real con autor fijo inline (git -c user.email=... -c user.name=...
// commit, sin tocar git config --global del sistema donde corre el test) —
// primer precedente del repo de un test que necesita un commit real, no
// solo git init (spec, US21).
func gitRepoWithCommitTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := gitRepoTestDir(t)

	for path, content := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("no se pudo crear el directorio para %q: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("no se pudo escribir %q: %v", path, err)
		}
	}

	if out, err := exec.Command("git", "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add -A falló: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-c", "user.email=test@april.dev", "-c", "user.name=test", "commit", "-q", "-m", "baseline").CombinedOutput(); err != nil {
		t.Fatalf("git commit falló: %v\n%s", err, out)
	}

	return dir
}

// TestBaseTreeForDiffConCommitDevuelveArbolDeHead cubre que, en un
// repositorio con al menos un commit, baseTreeForDiff devuelve el árbol de
// HEAD (no gitEmptyTreeHash).
func TestBaseTreeForDiffConCommitDevuelveArbolDeHead(t *testing.T) {
	gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	got, err := baseTreeForDiff()
	if err != nil {
		t.Fatalf("baseTreeForDiff falló: %v", err)
	}

	want, err := exec.Command("git", "rev-parse", "HEAD^{tree}").Output()
	if err != nil {
		t.Fatalf("git rev-parse falló: %v", err)
	}
	if got != strings.TrimSpace(string(want)) {
		t.Errorf("got = %q, want = %q", got, strings.TrimSpace(string(want)))
	}
	if got == gitEmptyTreeHash {
		t.Errorf("baseTreeForDiff devolvió gitEmptyTreeHash en un repo con al menos un commit")
	}
}

// TestBaseTreeForDiffSinCommitsDevuelveArbolVacio cubre US7: en un
// repositorio sin ningún commit todavía, baseTreeForDiff devuelve
// gitEmptyTreeHash sin error.
func TestBaseTreeForDiffSinCommitsDevuelveArbolVacio(t *testing.T) {
	gitRepoTestDir(t)

	got, err := baseTreeForDiff()
	if err != nil {
		t.Fatalf("baseTreeForDiff falló: %v", err)
	}
	if got != gitEmptyTreeHash {
		t.Errorf("got = %q, want = %q (gitEmptyTreeHash)", got, gitEmptyTreeHash)
	}
}

// TestComputeTouchedPathsDetectaArchivoModificado cubre que, con un
// baseline commiteado y un archivo modificado después, computeTouchedPaths
// devuelve exactamente esa ruta modificada.
func TestComputeTouchedPathsDetectaArchivoModificado(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola cambiado"), 0644); err != nil {
		t.Fatalf("no se pudo modificar el fixture: %v", err)
	}

	subjectHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	got, err := computeTouchedPaths(subjectHash)
	if err != nil {
		t.Fatalf("computeTouchedPaths falló: %v", err)
	}
	if len(got) != 1 || got[0] != "a.txt" {
		t.Errorf("got = %v, se esperaba exactamente [%q]", got, "a.txt")
	}
}

// TestComputeTouchedPathsVacioSinCambios cubre que, con un baseline
// commiteado sin ningún cambio posterior, computeTouchedPaths devuelve
// lista vacía.
func TestComputeTouchedPathsVacioSinCambios(t *testing.T) {
	gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	subjectHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	got, err := computeTouchedPaths(subjectHash)
	if err != nil {
		t.Fatalf("computeTouchedPaths falló: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got = %v, se esperaba vacío", got)
	}
}

// TestComputeTouchedPathsSinCommitsPrevios cubre US7: sin ningún commit
// previo (gitRepoTestDir puro), con archivos recién escritos,
// computeTouchedPaths devuelve todos esos archivos (diff contra el árbol
// vacío), sin error.
func TestComputeTouchedPathsSinCommitsPrevios(t *testing.T) {
	dir := gitRepoTestDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hola"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("mundo"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	subjectHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	got, err := computeTouchedPaths(subjectHash)
	if err != nil {
		t.Fatalf("computeTouchedPaths falló: %v", err)
	}

	want := map[string]bool{"a.txt": true, "b.txt": true}
	if len(got) != len(want) {
		t.Fatalf("got = %v, se esperaban exactamente %v", got, want)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("ruta inesperada en el resultado: %q", p)
		}
	}
}

// TestComputeTouchedPathsExcluyeLedgerYProgress cubre US8: con un baseline
// commiteado sin ledger ni progress/, si después del commit aparecen
// .claude/verify-ledger.jsonl y progress/current.md en el árbol de trabajo,
// ninguna de las dos rutas aparece en el resultado de computeTouchedPaths.
func TestComputeTouchedPathsExcluyeLedgerYProgress(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, verifyLedgerPath), []byte(`{"kind":"test"}`+"\n"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el ledger: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "progress"), 0755); err != nil {
		t.Fatalf("no se pudo crear progress/: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "progress", "current.md"), []byte("bitácora"), 0644); err != nil {
		t.Fatalf("no se pudo escribir progress/current.md: %v", err)
	}

	subjectHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	got, err := computeTouchedPaths(subjectHash)
	if err != nil {
		t.Fatalf("computeTouchedPaths falló: %v", err)
	}
	for _, p := range got {
		if p == verifyLedgerPath || strings.HasPrefix(p, "progress/") {
			t.Errorf("%q no debería aparecer en touchedPaths", p)
		}
	}
}

// TestComputeTouchedPathsArchivoNuevoNoTrackeadoCuenta cubre que un archivo
// nuevo nunca trackeado (sin git add manual) sí aparece en el resultado.
func TestComputeTouchedPathsArchivoNuevoNoTrackeadoCuenta(t *testing.T) {
	dir := gitRepoWithCommitTestDir(t, map[string]string{"a.txt": "hola"})

	if err := os.WriteFile(filepath.Join(dir, "nuevo.txt"), []byte("contenido nuevo"), 0644); err != nil {
		t.Fatalf("no se pudo escribir el fixture: %v", err)
	}

	subjectHash, err := computeSubjectHash()
	if err != nil {
		t.Fatalf("computeSubjectHash falló: %v", err)
	}

	got, err := computeTouchedPaths(subjectHash)
	if err != nil {
		t.Fatalf("computeTouchedPaths falló: %v", err)
	}
	found := false
	for _, p := range got {
		if p == "nuevo.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("got = %v, se esperaba que incluyera %q", got, "nuevo.txt")
	}
}
