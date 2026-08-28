package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// TestHashTreeExcluyeGitProgressYElPropioLedger cubre verify.go:hashTree —
// modificar solo archivos bajo .git/, progress/ o el propio
// .claude/verify-ledger.jsonl no debe cambiar el hash resultante (spec,
// "Cálculo del hash del árbol").
func TestHashTreeExcluyeGitProgressYElPropioLedger(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json":           &fstest.MapFile{Data: []byte(`{"features":[]}`)},
		".git/HEAD":                   &fstest.MapFile{Data: []byte("ref: refs/heads/main\n")},
		".git/objects/abc":            &fstest.MapFile{Data: []byte("dato git")},
		"progress/current.md":         &fstest.MapFile{Data: []byte("bitacora inicial")},
		".claude/verify-ledger.jsonl": &fstest.MapFile{Data: []byte(`{"kind":"test"}` + "\n")},
	}

	before, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	// Modifica solo los archivos excluidos.
	fsys[".git/HEAD"] = &fstest.MapFile{Data: []byte("ref: refs/heads/otra\n")}
	fsys[".git/objects/abc"] = &fstest.MapFile{Data: []byte("dato git modificado")}
	fsys["progress/current.md"] = &fstest.MapFile{Data: []byte("bitacora modificada, distinto contenido")}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(`{"kind":"test"}` + "\n" + `{"kind":"test"}` + "\n")}

	after, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	if before != after {
		t.Errorf("hashTree cambió tras modificar solo archivos excluidos: before=%s after=%s", before, after)
	}
}

// TestHashTreeCambiaSiUnArchivoNoExcluidoCambia cubre verify.go:hashTree —
// modificar un archivo fuera de las tres exclusiones sí debe cambiar el
// hash resultante.
func TestHashTreeCambiaSiUnArchivoNoExcluidoCambia(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: []byte(`{"features":[]}`)},
		"status.go":         &fstest.MapFile{Data: []byte("package main\n")},
	}

	before, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	fsys["status.go"] = &fstest.MapFile{Data: []byte("package main\n// cambio\n")}

	after, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	if before == after {
		t.Errorf("hashTree no cambió tras modificar un archivo no excluido (status.go)")
	}
}

// TestHashTreeEsDeterministicoSinImportarOrden cubre verify.go:hashTree —
// dos fstest.MapFS con el mismo contenido, construidos insertando las
// mismas claves en distinto orden (el orden de iteración de un mapa Go no
// es estable), deben producir el mismo hash.
func TestHashTreeEsDeterministicoSinImportarOrden(t *testing.T) {
	a := fstest.MapFS{}
	a["feature_list.json"] = &fstest.MapFile{Data: []byte(`{"features":[]}`)}
	a["docs/architecture.md"] = &fstest.MapFile{Data: []byte("# arquitectura\n")}
	a["status.go"] = &fstest.MapFile{Data: []byte("package main\n")}

	b := fstest.MapFS{}
	b["status.go"] = &fstest.MapFile{Data: []byte("package main\n")}
	b["feature_list.json"] = &fstest.MapFile{Data: []byte(`{"features":[]}`)}
	b["docs/architecture.md"] = &fstest.MapFile{Data: []byte("# arquitectura\n")}

	hashA, err := hashTree(a)
	if err != nil {
		t.Fatalf("hashTree falló sobre a: %v", err)
	}
	hashB, err := hashTree(b)
	if err != nil {
		t.Fatalf("hashTree falló sobre b: %v", err)
	}

	if hashA != hashB {
		t.Errorf("hashTree no es determinístico: hashA=%s hashB=%s", hashA, hashB)
	}
}

// ---- Ticket 03 (tree_hash_respects_gitignore, feature 12): hashTree
// respeta .gitignore — casos nuevos sobre fstest.MapFS, mismo patrón que
// los tres tests de arriba (que no se editan, US10 de la spec).

// TestHashTreeExcluyeArchivoGitignoreadoAunSinListaFija reproduce el bug
// real que origina esta feature (US6/US12 de la spec): un archivo
// gitignoreado (no en la lista fija de tres exclusiones) que se
// "regenera" con contenido distinto — como hace `go build ./...` con
// /HarnessInit — no debe cambiar el treeHash.
func TestHashTreeExcluyeArchivoGitignoreadoAunSinListaFija(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: []byte(`{"features":[]}`)},
		".gitignore":        &fstest.MapFile{Data: []byte("/HarnessInit\n")},
		"HarnessInit":       &fstest.MapFile{Data: []byte("contenido A")},
	}

	before, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	// Simula un rebuild real: mismo archivo, contenido distinto.
	fsys["HarnessInit"] = &fstest.MapFile{Data: []byte("contenido B, distinto (rebuild)")}

	after, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	if before != after {
		t.Errorf("hashTree cambió tras regenerar un archivo gitignoreado: before=%s after=%s", before, after)
	}
}

// TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore cubre
// US8/US23: las tres exclusiones fijas (.git/, el ledger, progress/) se
// preservan incondicionales, aunque el .gitignore presente no las
// mencione en absoluto.
func TestHashTreeSigueExcluyendoLasTresFijasAunNoEstenEnGitignore(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json":           &fstest.MapFile{Data: []byte(`{"features":[]}`)},
		".gitignore":                  &fstest.MapFile{Data: []byte("*.pyc\n")},
		".git/HEAD":                   &fstest.MapFile{Data: []byte("ref: refs/heads/main\n")},
		"progress/current.md":         &fstest.MapFile{Data: []byte("bitacora inicial")},
		".claude/verify-ledger.jsonl": &fstest.MapFile{Data: []byte(`{"kind":"test"}` + "\n")},
	}

	before, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	fsys[".git/HEAD"] = &fstest.MapFile{Data: []byte("ref: refs/heads/otra\n")}
	fsys["progress/current.md"] = &fstest.MapFile{Data: []byte("bitacora modificada, distinto contenido")}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(`{"kind":"test"}` + "\n" + `{"kind":"test"}` + "\n")}

	after, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	if before != after {
		t.Errorf("hashTree cambió tras modificar solo las tres exclusiones fijas, pese a que .gitignore no las menciona: before=%s after=%s", before, after)
	}
}

// TestHashTreeArchivoNoGitignoreadoSigueCambiandoElHashConGitignorePresente
// cubre US7: con un .gitignore real presente en el árbol, modificar un
// archivo que no matchea ningún patrón sigue cambiando el hash — la
// corrección no vuelve a hashTree ciego a cambios reales de código.
func TestHashTreeArchivoNoGitignoreadoSigueCambiandoElHashConGitignorePresente(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: []byte(`{"features":[]}`)},
		".gitignore":        &fstest.MapFile{Data: []byte("/HarnessInit\n*.pyc\n")},
		"status.go":         &fstest.MapFile{Data: []byte("package main\n")},
	}

	before, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	fsys["status.go"] = &fstest.MapFile{Data: []byte("package main\n// cambio\n")}

	after, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}

	if before == after {
		t.Errorf("hashTree no cambió tras modificar un archivo no gitignoreado (status.go), con .gitignore presente")
	}
}

// ---- ledgerEntry: esquema y serialización ----

// TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec cubre que ledgerEntry
// serializa a JSON con exactamente los ocho campos y nombres del esquema
// de la spec (sección "Esquema del ledger"), en una sola línea sin
// pretty-print, y que el resultado deserializa de vuelta sin pérdida.
func TestLedgerEntrySerializaAlEsquemaExactoDeLaSpec(t *testing.T) {
	entry := ledgerEntry{
		Kind:      "test",
		FeatureID: 5,
		Command:   []string{"go", "test", "./..."},
		ExitCode:  0,
		TreeHash:  "3f8a",
		Timestamp: "2026-08-27T18:04:05Z",
		Stdout:    "ok\n",
		Stderr:    "",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal(entry) falló: %v", err)
	}

	if strings.Contains(string(data), "\n") {
		t.Errorf("la serialización de ledgerEntry no debe contener saltos de línea (una sola línea, sin pretty-print): %q", data)
	}

	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		t.Fatalf("no se pudo deserializar la salida de json.Marshal(entry): %v", err)
	}
	wantKeys := []string{"kind", "featureId", "command", "exitCode", "treeHash", "timestamp", "stdout", "stderr"}
	for _, k := range wantKeys {
		if _, ok := generic[k]; !ok {
			t.Errorf("falta la clave %q en el JSON serializado: %s", k, data)
		}
	}
	if len(generic) != len(wantKeys) {
		t.Errorf("el JSON serializado tiene %d claves, se esperaban exactamente %d (%v): %s", len(generic), len(wantKeys), wantKeys, data)
	}

	var roundTrip ledgerEntry
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("no se pudo deserializar de vuelta a ledgerEntry: %v", err)
	}
	if roundTrip.Kind != entry.Kind || roundTrip.FeatureID != entry.FeatureID ||
		roundTrip.ExitCode != entry.ExitCode || roundTrip.TreeHash != entry.TreeHash ||
		roundTrip.Timestamp != entry.Timestamp || roundTrip.Stdout != entry.Stdout ||
		roundTrip.Stderr != entry.Stderr || len(roundTrip.Command) != len(entry.Command) {
		t.Errorf("round-trip de ledgerEntry no preservó los datos: got %+v, want %+v", roundTrip, entry)
	}
	for i := range entry.Command {
		if roundTrip.Command[i] != entry.Command[i] {
			t.Errorf("round-trip de command[%d] = %q, se esperaba %q", i, roundTrip.Command[i], entry.Command[i])
		}
	}
}

// ---- appendLedgerEntry: función pura de append ----

// TestAppendLedgerEsAppendOnlyNoPisaEntradasPrevias cubre que dos llamadas
// sucesivas a la función pura de append sobre el mismo contenido inicial
// producen un resultado con ambas líneas, en orden, con la primera intacta
// (nunca se pisa) — ver ticket 02, checklist.
func TestAppendLedgerEsAppendOnlyNoPisaEntradasPrevias(t *testing.T) {
	first := ledgerEntry{Kind: "test", FeatureID: 5, Command: []string{"go", "test", "./..."}, ExitCode: 0, TreeHash: "hash-uno", Timestamp: "2026-08-27T18:00:00Z", Stdout: "ok\n"}
	second := ledgerEntry{Kind: "test", FeatureID: 5, Command: []string{"go", "vet", "./..."}, ExitCode: 1, TreeHash: "hash-dos", Timestamp: "2026-08-27T18:05:00Z", Stderr: "falló\n"}

	afterFirst, err := appendLedgerEntry(nil, first)
	if err != nil {
		t.Fatalf("appendLedgerEntry(nil, first) falló: %v", err)
	}

	afterSecond, err := appendLedgerEntry(afterFirst, second)
	if err != nil {
		t.Fatalf("appendLedgerEntry(afterFirst, second) falló: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(afterSecond), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("se esperaban 2 líneas tras dos appends sucesivos, hubo %d: %q", len(lines), afterSecond)
	}

	if lines[0] != strings.TrimRight(string(afterFirst), "\n") {
		t.Errorf("la primera línea cambió tras el segundo append: antes=%q después=%q", afterFirst, lines[0])
	}

	var gotFirst, gotSecond ledgerEntry
	if err := json.Unmarshal([]byte(lines[0]), &gotFirst); err != nil {
		t.Fatalf("la primera línea no es JSON válido: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &gotSecond); err != nil {
		t.Fatalf("la segunda línea no es JSON válido: %v", err)
	}
	if gotFirst.TreeHash != first.TreeHash {
		t.Errorf("la primera línea tiene treeHash=%q, se esperaba %q", gotFirst.TreeHash, first.TreeHash)
	}
	if gotSecond.TreeHash != second.TreeHash {
		t.Errorf("la segunda línea tiene treeHash=%q, se esperaba %q", gotSecond.TreeHash, second.TreeHash)
	}
}

// TestAppendLedgerSobreArchivoInexistenteEmpiezaLimpio cubre que un
// contenido inicial vacío (equivalente a "archivo inexistente") produce un
// resultado con una sola línea válida.
func TestAppendLedgerSobreArchivoInexistenteEmpiezaLimpio(t *testing.T) {
	entry := ledgerEntry{Kind: "test", FeatureID: 7, Command: []string{"go", "build", "./..."}, ExitCode: 0, TreeHash: "hash-unico", Timestamp: "2026-08-27T19:00:00Z"}

	out, err := appendLedgerEntry([]byte{}, entry)
	if err != nil {
		t.Fatalf("appendLedgerEntry([]byte{}, entry) falló: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 sola línea sobre contenido inicial vacío, hubo %d: %q", len(lines), out)
	}

	var got ledgerEntry
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("la única línea no es JSON válido: %v", err)
	}
	if got.FeatureID != entry.FeatureID {
		t.Errorf("featureId = %d, se esperaba %d", got.FeatureID, entry.FeatureID)
	}
}

// ---- appendToLedger: wrapper de I/O sobre disco real ----

// TestAppendToLedgerArchivoInexistenteEmpiezaLimpio cubre que el wrapper de
// I/O trata un ledger inexistente en disco como contenido vacío, no como
// error, y que el archivo resultante queda con una sola línea válida.
func TestAppendToLedgerArchivoInexistenteEmpiezaLimpio(t *testing.T) {
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	entry := ledgerEntry{Kind: "test", FeatureID: 5, Command: []string{"go", "test", "./..."}, ExitCode: 0, TreeHash: "hash-real", Timestamp: "2026-08-27T20:00:00Z"}

	if err := appendToLedger(entry); err != nil {
		t.Fatalf("appendToLedger sobre ledger inexistente falló: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, verifyLedgerPath))
	if err != nil {
		t.Fatalf("no se pudo leer el ledger recién escrito: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("se esperaba 1 línea en el ledger recién creado, hubo %d: %q", len(lines), data)
	}
	var got ledgerEntry
	if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
		t.Fatalf("la línea escrita no es JSON válido: %v", err)
	}
	if got.TreeHash != entry.TreeHash {
		t.Errorf("treeHash en disco = %q, se esperaba %q", got.TreeHash, entry.TreeHash)
	}
}

// TestAppendToLedgerDosLlamadasProducenDosLineasSinPisarNiDejarArchivosTemp
// cubre que el wrapper escribe con writeFileAtomic (reusada de
// set_status.go): dos llamadas sucesivas producen dos líneas (ninguna se
// pisa) y no dejan archivos temporales (`.tmp-*`) huérfanos en `.claude/`
// tras cada escritura.
func TestAppendToLedgerDosLlamadasProducenDosLineasSinPisarNiDejarArchivosTemp(t *testing.T) {
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}

	first := ledgerEntry{Kind: "test", FeatureID: 5, Command: []string{"go", "test", "./..."}, ExitCode: 0, TreeHash: "hash-1", Timestamp: "2026-08-27T20:00:00Z"}
	second := ledgerEntry{Kind: "test", FeatureID: 5, Command: []string{"go", "vet", "./..."}, ExitCode: 0, TreeHash: "hash-2", Timestamp: "2026-08-27T20:05:00Z"}

	if err := appendToLedger(first); err != nil {
		t.Fatalf("primer appendToLedger falló: %v", err)
	}
	if err := appendToLedger(second); err != nil {
		t.Fatalf("segundo appendToLedger falló: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, verifyLedgerPath))
	if err != nil {
		t.Fatalf("no se pudo leer el ledger: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("se esperaban 2 líneas tras dos appendToLedger, hubo %d: %q", len(lines), data)
	}

	entries, err := os.ReadDir(filepath.Join(dir, ".claude"))
	if err != nil {
		t.Fatalf("no se pudo listar .claude/: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("quedó un archivo temporal huérfano en .claude/: %s", e.Name())
		}
	}
}

// ---- Ticket 03: recordVerify / runVerifyRecord — comando completo ----

// verifyTestDir prepara un directorio temporal como cwd del proceso de test
// (chdirTemp) con .claude/ ya creado, para que appendToLedger tenga dónde
// escribir su archivo temporal — mismo requisito que ya usan los tests de
// appendToLedger de arriba.
func verifyTestDir(t *testing.T) string {
	t.Helper()
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	return dir
}

// readLedgerLines lee el ledger real en dir y devuelve sus líneas no
// vacías ya parseadas.
func readLedgerLines(t *testing.T, dir string) []ledgerEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, verifyLedgerPath))
	if err != nil {
		t.Fatalf("no se pudo leer el ledger: %v", err)
	}
	var entries []ledgerEntry
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var e ledgerEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("línea del ledger no es JSON válido: %v (%q)", err, line)
		}
		entries = append(entries, e)
	}
	return entries
}

// TestRecordVerifyComandoExitosoRegistraExitCeroYTreeHash cubre que un
// comando exitoso (sh -c "exit 0") queda registrado en el ledger real en
// disco con exitCode == 0, featureId correcto y treeHash no vacío.
func TestRecordVerifyComandoExitosoRegistraExitCeroYTreeHash(t *testing.T) {
	dir := verifyTestDir(t)

	entry, exitCode, err := recordVerify(5, []string{"sh", "-c", "exit 0"})
	if err != nil {
		t.Fatalf("recordVerify falló: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode devuelto = %d, se esperaba 0", exitCode)
	}
	if entry.ExitCode != 0 {
		t.Errorf("entry.ExitCode = %d, se esperaba 0", entry.ExitCode)
	}
	if entry.FeatureID != 5 {
		t.Errorf("entry.FeatureID = %d, se esperaba 5", entry.FeatureID)
	}
	if entry.TreeHash == "" {
		t.Errorf("entry.TreeHash está vacío")
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 {
		t.Fatalf("se esperaba 1 entrada en el ledger, hubo %d", len(entries))
	}
	if entries[0].ExitCode != 0 || entries[0].FeatureID != 5 || entries[0].TreeHash == "" {
		t.Errorf("entrada en disco = %+v, no coincide con lo esperado", entries[0])
	}
}

// TestRecordVerifyComandoFallidoRegistraExitCodeReal cubre que un comando
// fallido (sh -c "exit 3") queda registrado con exitCode == 3 real, no un
// booleano genérico de "falló".
func TestRecordVerifyComandoFallidoRegistraExitCodeReal(t *testing.T) {
	dir := verifyTestDir(t)

	entry, exitCode, err := recordVerify(5, []string{"sh", "-c", "exit 3"})
	if err != nil {
		t.Fatalf("recordVerify no debería devolver error para un comando que arrancó y falló: %v", err)
	}
	if exitCode != 3 {
		t.Errorf("exitCode devuelto = %d, se esperaba 3", exitCode)
	}
	if entry.ExitCode != 3 {
		t.Errorf("entry.ExitCode = %d, se esperaba 3", entry.ExitCode)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 || entries[0].ExitCode != 3 {
		t.Fatalf("se esperaba 1 entrada con ExitCode == 3 en el ledger, hubo: %+v", entries)
	}
}

// TestRecordVerifyCapturaStdoutYStderrPorSeparado cubre que stdout y
// stderr del comando quedan capturados por separado en la entrada del
// ledger, con el contenido exacto emitido por el comando.
func TestRecordVerifyCapturaStdoutYStderrPorSeparado(t *testing.T) {
	verifyTestDir(t)

	entry, _, err := recordVerify(9, []string{"sh", "-c", "echo hola-stdout; echo hola-stderr >&2"})
	if err != nil {
		t.Fatalf("recordVerify falló: %v", err)
	}
	if entry.Stdout != "hola-stdout\n" {
		t.Errorf("entry.Stdout = %q, se esperaba %q", entry.Stdout, "hola-stdout\n")
	}
	if entry.Stderr != "hola-stderr\n" {
		t.Errorf("entry.Stderr = %q, se esperaba %q", entry.Stderr, "hola-stderr\n")
	}
}

// TestRecordVerifyComandoInexistenteNoEscribeLedger cubre que un comando
// cuyo binario no existe/no arranca devuelve error de invocación y NO
// escribe ninguna entrada al ledger.
func TestRecordVerifyComandoInexistenteNoEscribeLedger(t *testing.T) {
	dir := verifyTestDir(t)

	_, _, err := recordVerify(5, []string{"april-comando-binario-inexistente-xyz"})
	if err == nil {
		t.Fatalf("recordVerify no devolvió error para un binario inexistente")
	}

	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería haberse creado tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRecordVerifyDosCorridasProducenDosEntradas cubre que dos corridas
// sucesivas sobre el mismo featureId producen dos líneas en el ledger,
// ambas parseables, ninguna sobrescribe a la otra.
func TestRecordVerifyDosCorridasProducenDosEntradas(t *testing.T) {
	dir := verifyTestDir(t)

	if _, _, err := recordVerify(5, []string{"sh", "-c", "exit 0"}); err != nil {
		t.Fatalf("primera corrida falló: %v", err)
	}
	if _, _, err := recordVerify(5, []string{"sh", "-c", "exit 1"}); err != nil {
		t.Fatalf("segunda corrida (que falla, pero arranca) devolvió error inesperado: %v", err)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 2 {
		t.Fatalf("se esperaban 2 entradas en el ledger, hubo %d: %+v", len(entries), entries)
	}
	if entries[0].ExitCode != 0 {
		t.Errorf("primera entrada ExitCode = %d, se esperaba 0", entries[0].ExitCode)
	}
	if entries[1].ExitCode != 1 {
		t.Errorf("segunda entrada ExitCode = %d, se esperaba 1", entries[1].ExitCode)
	}
}

// TestRunVerifyRecordExitCodeReflejaComandoExitoso cubre que el exit code
// del propio proceso `april verify record` es el mismo que el del comando
// corrido: comando exitoso -> exit 0 del proceso.
func TestRunVerifyRecordExitCodeReflejaComandoExitoso(t *testing.T) {
	dir := verifyTestDir(t)

	exitCode := runVerifyRecord([]string{"--feature", "5", "--", "sh", "-c", "exit 0"})
	if exitCode != 0 {
		t.Errorf("exitCode = %d, se esperaba 0", exitCode)
	}

	entries := readLedgerLines(t, dir)
	if len(entries) != 1 || entries[0].FeatureID != 5 {
		t.Fatalf("se esperaba 1 entrada con featureId == 5, hubo: %+v", entries)
	}
}

// TestRunVerifyRecordExitCodeReflejaComandoFallido cubre que un comando con
// exit 3 hace que el propio proceso `verify record` también salga con
// exit 3.
func TestRunVerifyRecordExitCodeReflejaComandoFallido(t *testing.T) {
	verifyTestDir(t)

	exitCode := runVerifyRecord([]string{"--feature", "5", "--", "sh", "-c", "exit 3"})
	if exitCode != 3 {
		t.Errorf("exitCode = %d, se esperaba 3", exitCode)
	}
}

// TestRunVerifyRecordFaltaFeatureEsErrorDeInvocacion cubre que invocar sin
// --feature es error de invocación explícito, exit distinto de cero, sin
// tocar el ledger.
func TestRunVerifyRecordFaltaFeatureEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runVerifyRecord([]string{"--", "sh", "-c", "exit 0"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de --feature")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunVerifyRecordFaltaDobleGuionEsErrorDeInvocacion cubre que invocar
// sin el separador -- es error de invocación explícito, sin tocar el
// ledger.
func TestRunVerifyRecordFaltaDobleGuionEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runVerifyRecord([]string{"--feature", "5", "sh", "-c", "exit 0"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta del separador --")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunVerifyRecordSinComandoTrasDobleGuionEsErrorDeInvocacion cubre que
// invocar con -- pero sin ningún comando después es error de invocación
// explícito, sin tocar el ledger.
func TestRunVerifyRecordSinComandoTrasDobleGuionEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runVerifyRecord([]string{"--feature", "5", "--"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por falta de comando tras --")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// TestRunVerifyRecordFeatureNoNumericaEsErrorDeInvocacion cubre que un
// --feature con un valor no numérico es error de invocación explícito, sin
// tocar el ledger.
func TestRunVerifyRecordFeatureNoNumericaEsErrorDeInvocacion(t *testing.T) {
	dir := chdirTemp(t)

	exitCode := runVerifyRecord([]string{"--feature", "no-numerico", "--", "sh", "-c", "exit 0"})
	if exitCode == 0 {
		t.Errorf("exitCode = 0, se esperaba distinto de 0 por --feature no numérico")
	}
	if _, statErr := os.Stat(filepath.Join(dir, verifyLedgerPath)); !os.IsNotExist(statErr) {
		t.Errorf("el ledger no debería existir tras un error de invocación (stat err = %v)", statErr)
	}
}

// ---- Ticket 01 (tree_hash_respects_gitignore, feature 12): parser de
// .gitignore en Go puro — parseGitignore/gitignoreMatches/
// loadGitignorePatterns. Building block puro: todavía sin llamador real
// (hashTree se conecta en el ticket 03) — se verifica por sí solo, sobre
// literales de string y fstest.MapFS.

// TestParseGitignoreReconocePatronesBasicos cubre parseGitignore sobre un
// literal con una línea de cada clase real del .gitignore de este repo:
// verifica campo por campo (anchored/dirOnly/glob) contra lo esperado a
// mano.
func TestParseGitignoreReconocePatronesBasicos(t *testing.T) {
	content := "/HarnessInit\n" +
		"*.exe\n" +
		".vscode/\n" +
		"progress/*.md\n" +
		"harness-backend\n" +
		"\n" +
		"# comentario, se ignora\n"

	got := parseGitignore(content)

	want := []gitignorePattern{
		{anchored: true, dirOnly: false, glob: "HarnessInit"},
		{anchored: false, dirOnly: false, glob: "*.exe"},
		{anchored: false, dirOnly: true, glob: ".vscode"},
		{anchored: true, dirOnly: false, glob: "progress/*.md"},
		{anchored: false, dirOnly: false, glob: "harness-backend"},
	}

	if len(got) != len(want) {
		t.Fatalf("parseGitignore devolvió %d patrones, se esperaban %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("patrón[%d] = %+v, se esperaba %+v", i, got[i], want[i])
		}
	}
}

// TestParseGitignoreIgnoraNegacionSinFallar cubre que una línea de
// negación (!patrón, no soportada) no produce ningún patrón, ni error.
func TestParseGitignoreIgnoraNegacionSinFallar(t *testing.T) {
	got := parseGitignore("!importante.txt\n")
	if len(got) != 0 {
		t.Errorf("parseGitignore sobre una línea de negación devolvió %d patrones, se esperaban 0: %+v", len(got), got)
	}
}

// TestGitignoreMatchesPatronAncladoSoloEnRaiz cubre que un patrón anclado
// (/HarnessInit) matchea solo en la raíz, no en subdirectorios.
func TestGitignoreMatchesPatronAncladoSoloEnRaiz(t *testing.T) {
	patterns := parseGitignore("/HarnessInit\n")

	if !gitignoreMatches("HarnessInit", patterns) {
		t.Errorf("se esperaba que /HarnessInit matcheara %q", "HarnessInit")
	}
	if gitignoreMatches("sub/HarnessInit", patterns) {
		t.Errorf("no se esperaba que /HarnessInit matcheara %q (anclaje a raíz)", "sub/HarnessInit")
	}
}

// TestGitignoreMatchesPatronSinAnclaCualquierProfundidad cubre que un
// patrón sin ancla (*.pyc) matchea a cualquier profundidad del árbol.
func TestGitignoreMatchesPatronSinAnclaCualquierProfundidad(t *testing.T) {
	patterns := parseGitignore("*.pyc\n")

	if !gitignoreMatches("x.pyc", patterns) {
		t.Errorf("se esperaba que *.pyc matcheara %q", "x.pyc")
	}
	if !gitignoreMatches("sub/dir/x.pyc", patterns) {
		t.Errorf("se esperaba que *.pyc matcheara %q (cualquier profundidad)", "sub/dir/x.pyc")
	}
}

// TestGitignoreMatchesDirOnlyExcluyeContenidoCompleto cubre que un patrón
// terminado en / (.vscode/) matchea todo el contenido de ese directorio,
// no solo un nombre exacto.
func TestGitignoreMatchesDirOnlyExcluyeContenidoCompleto(t *testing.T) {
	patterns := parseGitignore(".vscode/\n")

	if !gitignoreMatches("sub/.vscode/settings.json", patterns) {
		t.Errorf("se esperaba que .vscode/ matcheara %q", "sub/.vscode/settings.json")
	}
}

// TestGitignoreMatchesPatronConSlashInternoQuedaAnclado cubre que un
// patrón con "/" intermedio (progress/*.md) queda anclado a la raíz.
func TestGitignoreMatchesPatronConSlashInternoQuedaAnclado(t *testing.T) {
	patterns := parseGitignore("progress/*.md\n")

	if !gitignoreMatches("progress/current.md", patterns) {
		t.Errorf("se esperaba que progress/*.md matcheara %q", "progress/current.md")
	}
	if gitignoreMatches("otro/progress/current.md", patterns) {
		t.Errorf("no se esperaba que progress/*.md matcheara %q (queda anclado a la raíz)", "otro/progress/current.md")
	}
}

// TestLoadGitignorePatternsSinArchivoDevuelveNil cubre que
// loadGitignorePatterns sobre un fstest.MapFS sin .gitignore devuelve
// nil, nil (cero patrones, cero error).
func TestLoadGitignorePatternsSinArchivoDevuelveNil(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: []byte(`{"features":[]}`)},
	}

	patterns, err := loadGitignorePatterns(fsys)
	if err != nil {
		t.Fatalf("loadGitignorePatterns falló: %v", err)
	}
	if patterns != nil {
		t.Errorf("patterns = %+v, se esperaba nil", patterns)
	}
}

// TestLoadGitignorePatternsConArchivoDevuelveMismosPatronesQueParseGitignore
// cubre que loadGitignorePatterns sobre un fstest.MapFS con .gitignore
// sintético devuelve los mismos patrones que parseGitignore sobre ese
// mismo contenido.
func TestLoadGitignorePatternsConArchivoDevuelveMismosPatronesQueParseGitignore(t *testing.T) {
	content := "/HarnessInit\n*.pyc\n.vscode/\n"
	fsys := fstest.MapFS{
		".gitignore": &fstest.MapFile{Data: []byte(content)},
	}

	got, err := loadGitignorePatterns(fsys)
	if err != nil {
		t.Fatalf("loadGitignorePatterns falló: %v", err)
	}

	want := parseGitignore(content)

	if len(got) != len(want) {
		t.Fatalf("loadGitignorePatterns devolvió %d patrones, se esperaban %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("patrón[%d] = %+v, se esperaba %+v", i, got[i], want[i])
		}
	}
}

// ---- Ticket 03 (tree_hash_respects_gitignore, feature 12): integración
// extremo a extremo sobre disco real — recordVerify/computeStatus, no
// hashTree en aislamiento (spec, "Testing Decisions"). Mismo patrón que
// verifyTestDir/chdirTemp ya usado arriba en este archivo.

// gitignoreIntegrationFeatureID es el id de la única feature del fixture
// que usan los tres tests de integración de abajo.
const gitignoreIntegrationFeatureID = 42

// gitignoreIntegrationFixtureDir prepara un directorio temporal como cwd
// del proceso de test (chdirTemp, con .claude/ ya creado para el ledger) y
// escribe un feature_list.json mínimo con una única feature in_progress,
// sdd:false, más un .gitignore con /build-artifact — el fixture que
// comparten los tres tests de integración de esta feature.
func gitignoreIntegrationFixtureDir(t *testing.T) string {
	t.Helper()
	dir := chdirTemp(t)
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0755); err != nil {
		t.Fatalf("no se pudo crear .claude/: %v", err)
	}
	writeFixtureFile(t, dir, "feature_list.json", testFeatureListJSON(
		`{"id": 42, "name": "gitignore_integration_fixture", "title": "t", "sdd": false, "status": "in_progress"}`,
	))
	writeFixtureFile(t, dir, ".gitignore", []byte("/build-artifact\n"))
	return dir
}

// blockedReasonsContain devuelve true si alguna entrada de reasons contiene
// substr.
func blockedReasonsContain(reasons []string, substr string) bool {
	for _, r := range reasons {
		if strings.Contains(r, substr) {
			return true
		}
	}
	return false
}

// TestRecordVerifyLuegoRegenerarArtefactoGitignoreadoNoProduceNoTestEvidence
// cubre US1/US3/US13 de la spec: recordVerify sobre un árbol con un
// artefacto gitignoreado, seguido de "regenerar" ese artefacto (simulando
// un go build ./... corrido después del record), no debe reaparecer
// no_test_evidence en blockedReasons.
func TestRecordVerifyLuegoRegenerarArtefactoGitignoreadoNoProduceNoTestEvidence(t *testing.T) {
	dir := gitignoreIntegrationFixtureDir(t)
	writeFixtureFile(t, dir, "build-artifact", []byte("contenido A"))

	if _, _, err := recordVerify(gitignoreIntegrationFeatureID, []string{"sh", "-c", "exit 0"}); err != nil {
		t.Fatalf("recordVerify falló: %v", err)
	}

	// Simula un go build ./... corrido después del record.
	writeFixtureFile(t, dir, "build-artifact", []byte("contenido B, distinto (rebuild posterior)"))

	report, err := computeStatus(nil)
	if err != nil {
		t.Fatalf("computeStatus falló: %v", err)
	}
	if blockedReasonsContain(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("blockedReasons contiene no_test_evidence tras regenerar un artefacto gitignoreado después del record: %v", report.BlockedReasons)
	}
}

// TestRegenerarArtefactoGitignoreadoLuegoRecordVerifyNoProduceNoTestEvidence
// cubre el orden inverso (US3/US14): regenerar el artefacto gitignoreado
// antes de recordVerify también debe dejar blockedReasons limpio.
func TestRegenerarArtefactoGitignoreadoLuegoRecordVerifyNoProduceNoTestEvidence(t *testing.T) {
	dir := gitignoreIntegrationFixtureDir(t)
	// El artefacto ya existe con contenido B antes de correr recordVerify —
	// simula un go build ./... corrido antes del record.
	writeFixtureFile(t, dir, "build-artifact", []byte("contenido B"))

	if _, _, err := recordVerify(gitignoreIntegrationFeatureID, []string{"sh", "-c", "exit 0"}); err != nil {
		t.Fatalf("recordVerify falló: %v", err)
	}

	report, err := computeStatus(nil)
	if err != nil {
		t.Fatalf("computeStatus falló: %v", err)
	}
	if blockedReasonsContain(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("blockedReasons contiene no_test_evidence con el orden inverso (rebuild antes del record): %v", report.BlockedReasons)
	}
}

// TestModificarArchivoNoGitignoreadoLuegoRecordVerifySigueDetectandoNoTestEvidence
// es el control (US7, reafirmado a nivel de status.go): modificar un
// archivo de código NO gitignoreado después de recordVerify sí debe hacer
// reaparecer no_test_evidence — la corrección no vuelve ciego el mecanismo
// a cambios reales.
func TestModificarArchivoNoGitignoreadoLuegoRecordVerifySigueDetectandoNoTestEvidence(t *testing.T) {
	dir := gitignoreIntegrationFixtureDir(t)
	writeFixtureFile(t, dir, "app.go", []byte("package main\n"))

	if _, _, err := recordVerify(gitignoreIntegrationFeatureID, []string{"sh", "-c", "exit 0"}); err != nil {
		t.Fatalf("recordVerify falló: %v", err)
	}

	// Modifica un archivo de código real, NO gitignoreado.
	writeFixtureFile(t, dir, "app.go", []byte("package main\n// cambio real de código\n"))

	report, err := computeStatus(nil)
	if err != nil {
		t.Fatalf("computeStatus falló: %v", err)
	}
	if !blockedReasonsContain(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("blockedReasons NO contiene no_test_evidence tras modificar un archivo de código no gitignoreado: %v", report.BlockedReasons)
	}
}
