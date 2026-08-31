package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// testFeatureListJSON envuelve el JSON de una o más features (ya
// serializadas a mano por cada test) en un feature_list.json mínimo pero
// válido, con el mismo rules.valid_status que usa el repo real.
func testFeatureListJSON(featuresJSON string) []byte {
	return []byte(`{
  "rules": {
    "valid_status": ["pending", "spec_ready", "in_progress", "done", "blocked"]
  },
  "features": [` + featuresJSON + `]
}`)
}

func intPtr(i int) *int { return &i }

// TestFeatureSddSinSpecEsFaseSpec cubre status.go:derivePhase() — una
// feature sdd:true sin specs/<name>/spec.md en disco todavía no puede
// avanzar a ninguna fase posterior: phase debe ser "spec".
func TestFeatureSddSinSpecEsFaseSpec(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "pending"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseSpec {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseSpec)
	}

	// nextRecommendedText (status.go) debe apuntar explícitamente a
	// spec_writer para la fase "spec" (spec, US15).
	if !strings.Contains(report.NextRecommended, "spec_writer") {
		t.Errorf("NextRecommended = %q, se esperaba que mencionara %q", report.NextRecommended, "spec_writer")
	}
}

// TestFeatureConSpecYSinTicketsEsFaseTickets cubre el caso "ya hay spec
// aprobada, pero specs/<name>/tickets/ todavía no tiene ningún archivo": la
// fase correspondiente es "tickets" (falta el desglose).
func TestFeatureConSpecYSinTicketsEsFaseTickets(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "spec_ready"}`,
		)},
		// El spec incluye un bloque Given/When/Then (irrelevante para lo que
		// este test cubre — derivePhase/nextRecommendedText) para no caer en
		// no_gwt_coverage (ticket 02, spec_gwt_mechanical_check): este fixture
		// es exactamente "spec existe, sin tickets, status != done", la
		// ventana que ese chequeo vigila, y un blockedReasons no vacío
		// vaciaría nextRecommended, rompiendo la aserción de más abajo sobre
		// ticket_writer sin que este test tenga nada que ver con GWT.
		"specs/april_status_arbiter/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nGiven algo\nWhen otra cosa\nThen resultado\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(2))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseTickets {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseTickets)
	}

	// nextRecommendedText (status.go) debe apuntar explícitamente a
	// ticket_writer para la fase "tickets" (spec, US16).
	if !strings.Contains(report.NextRecommended, "ticket_writer") {
		t.Errorf("NextRecommended = %q, se esperaba que mencionara %q", report.NextRecommended, "ticket_writer")
	}
}

// TestFeatureConTicketsPendientesEsFaseImplementation cubre que, existiendo
// al menos un ticket con Status distinto de done, la fase es
// "implementation" — todavía queda trabajo de agent_developer.
func TestFeatureConTicketsPendientesEsFaseImplementation(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":                &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md":   &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None (can start immediately)\n\n**Status:** in_progress\n")},
		"specs/april_status_arbiter/tickets/02-frontier.md": &fstest.MapFile{Data: []byte("# 02\n\n**Blocked by:** 01\n\n**Status:** pending\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseImplementation {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseImplementation)
	}

	// buildArtifactPaths (status.go) no solo debe traer las claves de
	// artifactPaths — su contenido debe ser el real del fixture: la ruta de
	// spec exacta y exactamente los archivos de ticket existentes.
	wantSpec := "specs/april_status_arbiter/spec.md"
	if report.ArtifactPaths.Spec != wantSpec {
		t.Errorf("ArtifactPaths.Spec = %q, se esperaba %q", report.ArtifactPaths.Spec, wantSpec)
	}
	wantTickets := []string{"01-nucleo.md", "02-frontier.md"}
	if !reflect.DeepEqual(report.ArtifactPaths.Tickets, wantTickets) {
		t.Errorf("ArtifactPaths.Tickets = %v, se esperaba %v", report.ArtifactPaths.Tickets, wantTickets)
	}
}

// TestFeatureConTodosLosTicketsDoneEsFaseReview cubre que, con todos los
// tickets en Status: done pero la feature misma todavía no done, la fase es
// "review" — falta el veredicto/cierre humano, no más implementación.
func TestFeatureConTodosLosTicketsDoneEsFaseReview(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":                &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md":   &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None (can start immediately)\n\n**Status:** done\n")},
		"specs/april_status_arbiter/tickets/02-frontier.md": &fstest.MapFile{Data: []byte("# 02\n\n**Blocked by:** 01\n\n**Status:** done\n")},
	}
	// Este test cubre derivePhase, no la evidencia de tests/review de los
	// tickets 04 y 02 (feature review_verdict_recorded) — se agregan un
	// receipt kind:test y uno kind:review, ambos verdes/vigentes, para que
	// no_test_evidence/no_review_verdict no interfieran con la aserción de
	// nextRecommended.
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(
		ledgerLine(t, ledgerEntry{
			Kind: "test", FeatureID: 2, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-27T20:00:00Z",
		}) + ledgerLine(t, ledgerEntry{
			Kind: "review", FeatureID: 2, TreeHash: currentHash, Timestamp: "2026-08-27T20:05:00Z", Verdict: verdictApproved,
		}),
	)}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseReview {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseReview)
	}

	// nextRecommendedText (status.go) debe apuntar explícitamente a
	// reviewer_agent para la fase "review" (spec, US18).
	if !strings.Contains(report.NextRecommended, "reviewer_agent") {
		t.Errorf("NextRecommended = %q, se esperaba que mencionara %q", report.NextRecommended, "reviewer_agent")
	}
}

// TestFeatureDoneEsFaseClosedSinImportarDisco cubre que status: "done" en
// feature_list.json manda siempre, incluso si no hay ni spec.md ni
// tickets/ en disco — el cierre es la señal final, no se recalcula desde
// specs/.
func TestFeatureDoneEsFaseClosedSinImportarDisco(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 1, "name": "bootstrap_project", "title": "t", "sdd": false, "status": "done"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, intPtr(1))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseClosed {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseClosed)
	}
}

// TestBootstrapProjectPendienteEsFaseGrill cubre que la feature
// bootstrap_project, mientras no esté done, siempre reporta "grill" — la
// distingue de una feature sdd:false cualquiera.
func TestBootstrapProjectPendienteEsFaseGrill(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 1, "name": "bootstrap_project", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseGrill {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseGrill)
	}
}

// TestFeatureSddFalseSinBootstrapEsFaseImplementation cubre que una feature
// sdd:false que no es bootstrap_project, con status pending, no espera
// ninguna fase previa: va directo a "implementation".
func TestFeatureSddFalseSinBootstrapEsFaseImplementation(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 3, "name": "claude_md_routes_by_status", "title": "t", "sdd": false, "status": "pending"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseImplementation {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseImplementation)
	}
}

// TestDosFeaturesInProgressReportaBlockedReasons cubre el estado
// inconsistente de dos features en in_progress a la vez: blockedReasons no
// debe quedar vacío, y por lo tanto nextRecommended debe quedar vacío
// también (nunca se recomienda avanzar con un problema sin resolver).
func TestDosFeaturesInProgressReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "in_progress"},` +
				`{"id": 3, "name": "claude_md_routes_by_status", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con dos features in_progress")
	}
	if report.NextRecommended != "" {
		t.Errorf("nextRecommended = %q, se esperaba vacío con blockedReasons no vacío", report.NextRecommended)
	}
}

// TestStatusInvalidoEnFeatureListReportaBlockedReasons cubre que un status
// fuera de rules.valid_status (corrupción o edición manual errónea) se
// reporta en blockedReasons.
func TestStatusInvalidoEnFeatureListReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "en_curso"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con status fuera de rules.valid_status")
	}
}

// TestSpecRequeridaYFaltanteReportaBlockedReasons cubre que una feature
// sdd:true con status in_progress (que ya debería tener spec aprobada) sin
// specs/<name>/spec.md en disco se reporta en blockedReasons — alguien
// saltó la puerta de aprobación de spec.
func TestSpecRequeridaYFaltanteReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con spec requerida y faltante")
	}
}

// TestFeatureBlockedSeReportaEnBlockedReasons cubre que una feature
// marcada blocked en feature_list.json queda reportada explícitamente en
// blockedReasons con su id/name, sin impedir que se calcule el resto.
func TestFeatureBlockedSeReportaEnBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 7, "name": "review_frozen_candidate", "title": "t", "sdd": true, "status": "blocked"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con una feature blocked")
	}
	found := false
	for _, r := range report.BlockedReasons {
		if strings.Contains(r, "7") && strings.Contains(r, "review_frozen_candidate") {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba que blockedReasons identificara la feature 7 (review_frozen_candidate), se obtuvo %v", report.BlockedReasons)
	}
}

// TestStatusDeTicketFueraDeVocabularioReportaBlockedReasons cubre que un
// ticket con Status fuera de pending/in_progress/done se reporta en
// blockedReasons.
func TestStatusDeTicketFueraDeVocabularioReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":              &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md": &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None (can start immediately)\n\n**Status:** wip\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con un ticket con Status fuera de vocabulario")
	}
}

// TestNextRecommendedVacioCuandoHayBlockedReasons reafirma, sobre un
// escenario distinto (status inválido en vez de doble in_progress), que
// nextRecommended siempre queda vacío mientras blockedReasons no lo esté.
func TestNextRecommendedVacioCuandoHayBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "en_curso"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("precondición del test: se esperaba blockedReasons no vacío")
	}
	if report.NextRecommended != "" {
		t.Errorf("nextRecommended = %q, se esperaba vacío", report.NextRecommended)
	}
}

// TestSinFeaturePendienteNiInProgressReportaClosedSinTarget cubre el
// backlog íntegramente done/blocked: no hay ninguna feature pending ni
// in_progress, así que no hay target. El resultado no es un error: phase
// "closed", frontier vacío, y nextRecommended explica que no hay pendientes.
func TestSinFeaturePendienteNiInProgressReportaClosedSinTarget(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 1, "name": "bootstrap_project", "title": "t", "sdd": false, "status": "done"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS no debería fallar sobre un backlog íntegramente done: %v", err)
	}
	if report.Phase != phaseClosed {
		t.Errorf("Phase = %q, se esperaba %q", report.Phase, phaseClosed)
	}
	if len(report.Frontier) != 0 {
		t.Errorf("Frontier = %v, se esperaba vacío", report.Frontier)
	}
	if report.NextRecommended == "" {
		t.Errorf("se esperaba que nextRecommended explicara que no hay nada pendiente")
	}
	if len(report.BlockedReasons) != 0 {
		t.Errorf("BlockedReasons = %v, se esperaba vacío", report.BlockedReasons)
	}
}

// TestSeleccionaFeatureIdExplicitoAunqueNoSeaLaActiva cubre que pasar un id
// explícito distinto de la feature activa (la in_progress) inspecciona esa
// feature sin cambiar el foco: el reporte debe reflejar la feature pedida,
// no la activa.
func TestSeleccionaFeatureIdExplicitoAunqueNoSeaLaActiva(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "in_progress"},` +
				`{"id": 3, "name": "claude_md_routes_by_status", "title": "t", "sdd": false, "status": "pending"}`,
		)},
	}
	// Este test cubre selectTarget, no la evidencia de tests/review de los
	// tickets 04 y 02 (feature review_verdict_recorded) — se agregan
	// receipts kind:test y kind:review, ambos verdes/vigentes, para la
	// feature 2 (in_progress) para que no_test_evidence/no_review_verdict no
	// interfieran con la aserción de nextRecommended sobre la feature 3
	// pedida explícitamente.
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(
		ledgerLine(t, ledgerEntry{
			Kind: "test", FeatureID: 2, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-27T20:00:00Z",
		}) + ledgerLine(t, ledgerEntry{
			Kind: "review", FeatureID: 2, TreeHash: currentHash, Timestamp: "2026-08-27T20:05:00Z", Verdict: verdictApproved,
		}),
	)}

	report, err := computeStatusFromFS(fsys, intPtr(3))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if report.Phase != phaseImplementation {
		t.Errorf("Phase = %q, se esperaba %q (feature 3, sdd:false pending)", report.Phase, phaseImplementation)
	}
	if !strings.Contains(report.NextRecommended, "3") {
		t.Errorf("nextRecommended = %q, se esperaba que mencionara la feature 3 pedida explícitamente", report.NextRecommended)
	}
}

// TestIdInexistenteEsErrorDeInvocacion cubre que pedir el status de un id
// que no existe en feature_list.json es un error de invocación (nunca un
// pánico, nunca un JSON con campos vacíos).
func TestIdInexistenteEsErrorDeInvocacion(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "pending"}`,
		)},
	}

	_, err := computeStatusFromFS(fsys, intPtr(999))
	if err == nil {
		t.Fatalf("se esperaba error al pedir el status de un id inexistente (999)")
	}
	if !errors.Is(err, ErrFeatureNotFound) {
		t.Errorf("error = %v, se esperaba que envolviera ErrFeatureNotFound", err)
	}
}

// ---- Integración: runStatus con disco real (t.TempDir) ----

// chdirTemp crea un directorio temporal, cambia el cwd del proceso de test
// a él, y devuelve una función de limpieza que restaura el cwd original —
// necesario porque computeStatus (a diferencia de computeStatusFromFS) lee
// siempre sobre os.DirFS(".").
func chdirTemp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("no se pudo obtener el cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("no se pudo cambiar al directorio temporal: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("no se pudo restaurar el cwd original: %v", err)
		}
	})

	return dir
}

func writeFixtureFile(t *testing.T, dir, relPath string, content []byte) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("no se pudo crear %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, content, 0644); err != nil {
		t.Fatalf("no se pudo escribir %s: %v", full, err)
	}
}

// hashDirTree calcula un hash agregado y determinístico del contenido de
// todo el árbol bajo root. Envuelve a hashTree (verify.go), la función de
// producción que extrae este mismo algoritmo — no duplica la lógica. Se
// usa para probar que april status no escribe nada: mismo hash antes y
// después de correr el comando.
func hashDirTree(t *testing.T, root string) string {
	t.Helper()

	sum, err := hashTree(os.DirFS(root))
	if err != nil {
		t.Fatalf("no se pudo recorrer %s: %v", root, err)
	}
	return sum
}

// writeCleanFixture deja en dir un feature_list.json válido y sin
// inconsistencias (una única feature pending), suficiente para que
// runStatus salga con exit 0.
func writeCleanFixture(t *testing.T, dir string) {
	t.Helper()
	writeFixtureFile(t, dir, "feature_list.json", testFeatureListJSON(
		`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "pending"}`,
	))
}

// TestCmdStatusJsonEsValidoYTieneLosCincoCampos cubre que `april status
// --json` sobre un fixture real en disco imprime un único objeto JSON
// válido con los cinco campos del reporte.
func TestCmdStatusJsonEsValidoYTieneLosCincoCampos(t *testing.T) {
	dir := chdirTemp(t)
	writeCleanFixture(t, dir)

	out, exitCode := runStatusCaptured(t, []string{"--json"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, se esperaba 0 sobre un fixture limpio; salida:\n%s", exitCode, out)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("la salida no es JSON válido: %v\nsalida:\n%s", err, out)
	}

	for _, field := range []string{"phase", "nextRecommended", "blockedReasons", "frontier", "artifactPaths"} {
		if _, ok := decoded[field]; !ok {
			t.Errorf("falta el campo %q en el JSON de salida: %s", field, out)
		}
	}
}

// TestCmdStatusExitCodeReflejaBlockedReasons cubre que el exit code de
// `april status --json` es 0 cuando blockedReasons está vacío y distinto de
// 0 cuando no lo está — sin necesidad de parsear el JSON en Bash.
func TestCmdStatusExitCodeReflejaBlockedReasons(t *testing.T) {
	t.Run("fixture_limpio_sale_0", func(t *testing.T) {
		dir := chdirTemp(t)
		writeCleanFixture(t, dir)

		_, exitCode := runStatusCaptured(t, []string{"--json"})
		if exitCode != 0 {
			t.Errorf("exitCode = %d, se esperaba 0 sobre un fixture limpio", exitCode)
		}
	})

	t.Run("dos_in_progress_sale_distinto_de_0", func(t *testing.T) {
		dir := chdirTemp(t)
		writeFixtureFile(t, dir, "feature_list.json", testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": false, "status": "in_progress"},`+
				`{"id": 3, "name": "claude_md_routes_by_status", "title": "t", "sdd": false, "status": "in_progress"}`,
		))

		_, exitCode := runStatusCaptured(t, []string{"--json"})
		if exitCode == 0 {
			t.Errorf("exitCode = 0, se esperaba distinto de 0 con dos features in_progress")
		}
	})
}

// TestCmdStatusNoEscribeNingunArchivo hashea el árbol completo del fixture
// antes y después de correr computeStatus/runStatus: deben ser idénticos —
// april status nunca escribe nada, ni siquiera de forma indirecta.
func TestCmdStatusNoEscribeNingunArchivo(t *testing.T) {
	dir := chdirTemp(t)
	writeCleanFixture(t, dir)
	writeFixtureFile(t, dir, "specs/april_status_arbiter/spec.md", []byte("# spec\n"))
	writeFixtureFile(t, dir, "specs/april_status_arbiter/tickets/01-nucleo.md",
		[]byte("# 01\n\n**Blocked by:** None (can start immediately)\n\n**Status:** in_progress\n"))

	before := hashDirTree(t, dir)

	if _, err := computeStatus(nil); err != nil {
		t.Fatalf("computeStatus falló: %v", err)
	}
	runStatusCaptured(t, []string{"--json"})
	runStatusCaptured(t, nil)

	after := hashDirTree(t, dir)

	if before != after {
		t.Errorf("el árbol cambió tras correr computeStatus/runStatus: before=%s after=%s", before, after)
	}
}

// runStatusCaptured corre runStatus in-process (sin os.Exit, a diferencia
// de cmdStatus) capturando su stdout, para poder inspeccionar tanto la
// salida como el exit code decidido.
func runStatusCaptured(t *testing.T, args []string) (string, int) {
	t.Helper()
	var exitCode int
	out, err := captureStdout(t, func() error {
		exitCode = runStatus(args)
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	return out, exitCode
}

// ---- Ticket 02: frontier y grafo de dependencias de tickets ----

// TestBlockedByReferenciaTicketInexistenteReportaBlockedReasons cubre la
// segunda pasada de readTickets: un Blocked by que sí tiene un número de dos
// dígitos interpretable, pero que no corresponde a ningún archivo de ticket
// existente de la feature, también debe caer en blockedReasons — no se
// asume "sin bloqueadores" en silencio solo porque el texto tenía forma de
// número válido.
func TestBlockedByReferenciaTicketInexistenteReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
		// 99 no corresponde a ningún archivo de ticket existente en esta
		// feature (solo existe 01-nucleo.md).
		"specs/april_status_arbiter/tickets/01-nucleo.md": &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** 99\n\n**Status:** pending\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con un Blocked by que referencia un ticket inexistente (99)")
	}
	found := false
	for _, r := range report.BlockedReasons {
		if strings.Contains(r, "01-nucleo.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba que blockedReasons identificara el ticket 01-nucleo.md, se obtuvo %v", report.BlockedReasons)
	}
	// Tampoco debe colarse en frontier — la referencia inválida no permite
	// confirmar con seguridad que el ticket no tiene bloqueadores pendientes.
	for _, id := range report.Frontier {
		if id == "01-nucleo" {
			t.Errorf("Frontier = %v, un ticket con Blocked by que referencia un ticket inexistente no debería entrar a frontier", report.Frontier)
		}
	}
}

// TestSelectTargetPriorizaInProgressSobrePendingConIdMenor cubre la
// prioridad real de selectTarget sin targetID: entre una feature pending de
// id menor y una in_progress de id mayor, gana la in_progress — la
// prioridad nunca se había ejercido de verdad porque los tests anteriores
// solo usaban fixtures de una sola feature candidata.
func TestSelectTargetPriorizaInProgressSobrePendingConIdMenor(t *testing.T) {
	features := []featureEntry{
		{ID: 2, Name: "april_status_arbiter", SDD: false, Status: "pending"},
		{ID: 5, Name: "otra_feature", SDD: false, Status: "in_progress"},
	}

	target, err := selectTarget(features, nil)
	if err != nil {
		t.Fatalf("selectTarget falló: %v", err)
	}
	if target == nil {
		t.Fatalf("se esperaba un target, se obtuvo nil")
	}
	if target.ID != 5 {
		t.Errorf("target.ID = %d, se esperaba 5 (la in_progress, con prioridad sobre la pending de id menor)", target.ID)
	}
}

// TestSelectTargetEligeMenorIdEntreDosPendientes cubre que, sin ninguna
// feature in_progress, entre dos pending se elige la de menor id.
func TestSelectTargetEligeMenorIdEntreDosPendientes(t *testing.T) {
	features := []featureEntry{
		{ID: 8, Name: "feature_ocho", SDD: false, Status: "pending"},
		{ID: 3, Name: "feature_tres", SDD: false, Status: "pending"},
	}

	target, err := selectTarget(features, nil)
	if err != nil {
		t.Fatalf("selectTarget falló: %v", err)
	}
	if target == nil {
		t.Fatalf("se esperaba un target, se obtuvo nil")
	}
	if target.ID != 3 {
		t.Errorf("target.ID = %d, se esperaba 3 (menor id entre las dos pending)", target.ID)
	}
}

// TestCmdStatusTextoPlanoContieneLosMismosDatos cubre US31: sin --json, la
// salida en texto plano debe reflejar los mismos datos reales del reporte
// (no solo "se llamó a printStatusText" sin verificar contenido). Fixture
// con phase, frontier y blockedReasons no triviales a la vez.
func TestCmdStatusTextoPlanoContieneLosMismosDatos(t *testing.T) {
	dir := chdirTemp(t)
	writeFixtureFile(t, dir, "feature_list.json", testFeatureListJSON(
		`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"},`+
			`{"id": 7, "name": "review_frozen_candidate", "title": "t", "sdd": false, "status": "blocked"}`,
	))
	writeFixtureFile(t, dir, "specs/april_status_arbiter/spec.md", []byte("# spec\n"))
	writeFixtureFile(t, dir, "specs/april_status_arbiter/tickets/01-nucleo.md",
		[]byte("# 01\n\n**Blocked by:** None\n\n**Status:** done\n"))
	writeFixtureFile(t, dir, "specs/april_status_arbiter/tickets/02-frontier.md",
		[]byte("# 02\n\n**Blocked by:** 01\n\n**Status:** pending\n"))

	out, _ := runStatusCaptured(t, nil)

	report, err := computeStatus(nil)
	if err != nil {
		t.Fatalf("computeStatus falló: %v", err)
	}
	// Precondición del test: el fixture debe producir datos no triviales en
	// los tres campos, para que la comparación sea significativa.
	if report.Phase != phaseImplementation {
		t.Fatalf("precondición del test: Phase = %q, se esperaba %q", report.Phase, phaseImplementation)
	}
	if len(report.Frontier) == 0 {
		t.Fatalf("precondición del test: se esperaba Frontier no vacío")
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("precondición del test: se esperaba BlockedReasons no vacío")
	}

	if !strings.Contains(out, report.Phase) {
		t.Errorf("la salida en texto plano no contiene el phase real (%q); salida:\n%s", report.Phase, out)
	}
	for _, ticketID := range report.Frontier {
		if !strings.Contains(out, ticketID) {
			t.Errorf("la salida en texto plano no contiene el ticket de frontier %q; salida:\n%s", ticketID, out)
		}
	}
	for _, reason := range report.BlockedReasons {
		if !strings.Contains(out, reason) {
			t.Errorf("la salida en texto plano no contiene el blockedReason %q; salida:\n%s", reason, out)
		}
	}
}

// TestFrontierListaSoloTicketsConBloqueadoresEnDone cubre el caso central de
// frontier: de tres tickets encadenados (01 sin bloqueadores y done, 02
// bloqueado por 01 y pendiente, 03 bloqueado por 02 y pendiente), solo 02
// tiene todos sus Blocked by en done — 03 sigue bloqueado por 02, que no
// está done.
func TestFrontierListaSoloTicketsConBloqueadoresEnDone(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":                &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md":   &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None\n\n**Status:** done\n")},
		"specs/april_status_arbiter/tickets/02-frontier.md": &fstest.MapFile{Data: []byte("# 02\n\n**Blocked by:** 01\n\n**Status:** pending\n")},
		"specs/april_status_arbiter/tickets/03-cli.md":      &fstest.MapFile{Data: []byte("# 03\n\n**Blocked by:** 02\n\n**Status:** pending\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.Frontier) != 1 || report.Frontier[0] != "02-frontier" {
		t.Errorf("Frontier = %v, se esperaba exactamente [\"02-frontier\"]", report.Frontier)
	}
}

// TestFrontierExcluyeTicketsYaDone cubre que un ticket ya done nunca aparece
// en frontier, aunque todos sus Blocked by también estén done.
func TestFrontierExcluyeTicketsYaDone(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":                &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md":   &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None\n\n**Status:** done\n")},
		"specs/april_status_arbiter/tickets/02-frontier.md": &fstest.MapFile{Data: []byte("# 02\n\n**Blocked by:** 01\n\n**Status:** done\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	for _, id := range report.Frontier {
		if id == "01-nucleo" || id == "02-frontier" {
			t.Errorf("Frontier = %v, ningún ticket done debería aparecer", report.Frontier)
		}
	}
}

// TestBlockedByConTextoNoInterpretableReportaBlockedReasons cubre que un
// Blocked by sin números de dos dígitos y sin "none" (texto libre no
// interpretable) se reporta en blockedReasons en vez de asumirse "sin
// bloqueadores" en silencio.
func TestBlockedByConTextoNoInterpretableReportaBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":              &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/01-nucleo.md": &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** algo raro sin numero\n\n**Status:** pending\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if len(report.BlockedReasons) == 0 {
		t.Fatalf("se esperaba blockedReasons no vacío con un Blocked by no interpretable")
	}
	found := false
	for _, r := range report.BlockedReasons {
		if strings.Contains(r, "01-nucleo.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba que blockedReasons identificara el ticket 01-nucleo.md, se obtuvo %v", report.BlockedReasons)
	}
	// Un Blocked by no interpretable no debe colarse en frontier — no se
	// puede confirmar con seguridad que no tiene bloqueadores pendientes.
	for _, id := range report.Frontier {
		if id == "01-nucleo" {
			t.Errorf("Frontier = %v, un ticket con Blocked by no interpretable no debería entrar a frontier", report.Frontier)
		}
	}
}

// ---- Ticket 04: readLedger — lectura de .claude/verify-ledger.jsonl ----

// TestReadLedgerArchivoInexistenteEntriesVacio cubre que un ledger
// inexistente (nadie corrió `verify record` todavía) no es error: entries
// vacío, corruptLines vacío.
func TestReadLedgerArchivoInexistenteEntriesVacio(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: []byte(`{"features":[]}`)},
	}

	entries, corruptLines, err := readLedger(fsys)
	if err != nil {
		t.Fatalf("readLedger falló sobre un ledger inexistente: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, se esperaba vacío", entries)
	}
	if len(corruptLines) != 0 {
		t.Errorf("corruptLines = %v, se esperaba vacío", corruptLines)
	}
}

// TestReadLedgerIgnoraLineasVacias cubre que líneas en blanco dentro del
// ledger (ej. el salto de línea final) se ignoran sin producir entradas
// vacías ni corruptLines espurios.
func TestReadLedgerIgnoraLineasVacias(t *testing.T) {
	fsys := fstest.MapFS{
		".claude/verify-ledger.jsonl": &fstest.MapFile{Data: []byte(
			`{"kind":"test","featureId":5,"exitCode":0}` + "\n\n",
		)},
	}

	entries, corruptLines, err := readLedger(fsys)
	if err != nil {
		t.Fatalf("readLedger falló: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("se esperaba 1 entrada, hubo %d: %v", len(entries), entries)
	}
	if len(corruptLines) != 0 {
		t.Errorf("corruptLines = %v, se esperaba vacío (líneas vacías no cuentan como corruptas)", corruptLines)
	}
}

// TestReadLedgerLineaCorruptaVaAparteYNoAbortaLaLectura cubre que una línea
// con JSON inválido se reporta en corruptLines (identificando la línea) sin
// impedir que las líneas válidas antes y después se lean con normalidad.
func TestReadLedgerLineaCorruptaVaAparteYNoAbortaLaLectura(t *testing.T) {
	fsys := fstest.MapFS{
		".claude/verify-ledger.jsonl": &fstest.MapFile{Data: []byte(
			`{"kind":"test","featureId":5,"exitCode":0}` + "\n" +
				`esto no es json valido` + "\n" +
				`{"kind":"test","featureId":5,"exitCode":1}` + "\n",
		)},
	}

	entries, corruptLines, err := readLedger(fsys)
	if err != nil {
		t.Fatalf("readLedger falló: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("se esperaban 2 entradas válidas, hubo %d: %v", len(entries), entries)
	}
	if len(corruptLines) != 1 {
		t.Fatalf("se esperaba 1 línea corrupta, hubo %d: %v", len(corruptLines), corruptLines)
	}
	if !strings.Contains(corruptLines[0], "2") {
		t.Errorf("corruptLines[0] = %q, se esperaba que identificara el número de línea (2)", corruptLines[0])
	}
}

// ---- Ticket 04: computeBlockedReasons — no_test_evidence ----

// ledgerLine serializa una ledgerEntry a una línea JSON con salto de línea
// final, para armar contenido de .claude/verify-ledger.jsonl en fixtures.
func ledgerLine(t *testing.T, e ledgerEntry) string {
	t.Helper()
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal(ledgerEntry) falló: %v", err)
	}
	return string(data) + "\n"
}

// TestSinReceiptParaFeatureInProgressReportaNoTestEvidence cubre que una
// feature in_progress sin ningún receipt kind:test en el ledger reporta
// no_test_evidence.
func TestSinReceiptParaFeatureInProgressReportaNoTestEvidence(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, se esperaba una entrada con no_test_evidence", report.BlockedReasons)
	}
}

// TestReceiptConExitDistintoDeCeroReportaNoTestEvidence cubre que un
// receipt kind:test con exitCode != 0 (la corrida se intentó, pero falló)
// también reporta no_test_evidence — no se confunde "se intentó y falló"
// con "está verde".
func TestReceiptConExitDistintoDeCeroReportaNoTestEvidence(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}
	entry := ledgerEntry{Kind: "test", FeatureID: 5, ExitCode: 1, TreeHash: "cualquiera", Timestamp: "2026-08-27T20:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, se esperaba una entrada con no_test_evidence", report.BlockedReasons)
	}
}

// TestReceiptExitosoSobreArbolActualNoReportaNoTestEvidence cubre que un
// receipt kind:test en verde (exitCode == 0) y vigente (mismo treeHash que
// el árbol actual) NO reporta no_test_evidence.
func TestReceiptExitosoSobreArbolActualNoReportaNoTestEvidence(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "test", FeatureID: 5, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-27T20:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_test_evidence con un receipt verde y vigente", report.BlockedReasons)
	}
}

// TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence cubre que
// un receipt con exitCode == 0 pero treeHash desactualizado (el árbol
// cambió después de la corrida registrada) reporta no_test_evidence.
func TestReceiptExitosoConArbolDesactualizadoReportaNoTestEvidence(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}
	// treeHash de una versión "vieja" del árbol, antes de que existiera este
	// feature_list.json actual.
	oldHash, err := hashTree(fstest.MapFS{"status.go": &fstest.MapFile{Data: []byte("package main\n// version vieja\n")}})
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "test", FeatureID: 5, ExitCode: 0, TreeHash: oldHash, Timestamp: "2026-08-27T20:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, se esperaba no_test_evidence con treeHash desactualizado", report.BlockedReasons)
	}
}

// TestEscribirEnProgressNoInvalidaReceiptVigente es la regresión directa
// del problema de auto-invalidación detectado durante la fase de spec:
// agregar contenido a progress/ después de registrar un receipt vigente no
// debe invalidarlo (hashTree ya excluye progress/ desde el ticket 01).
func TestEscribirEnProgressNoInvalidaReceiptVigente(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "test", FeatureID: 5, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-27T20:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	// Después del receipt vigente, un subagente agrega su bitácora
	// obligatoria a progress/current.md.
	fsys["progress/current.md"] = &fstest.MapFile{Data: []byte("## Progress Log\n- entrada nueva del agent_developer\n")}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, escribir en progress/ no debería invalidar un receipt vigente", report.BlockedReasons)
	}
}

// TestNoTestEvidenceSoloAplicaAFeatureInProgress cubre que features
// pending/done nunca reportan no_test_evidence, tengan o no receipt.
func TestNoTestEvidenceSoloAplicaAFeatureInProgress(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 3, "name": "feature_pendiente", "title": "t", "sdd": false, "status": "pending"},` +
				`{"id": 1, "name": "bootstrap_project", "title": "t", "sdd": false, "status": "done"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_test_evidence sin ninguna feature in_progress", report.BlockedReasons)
	}
}

// TestLineaCorruptaDeLedgerSeReportaEnBlockedReasons cubre que una línea
// con JSON inválido en el ledger se reporta explícitamente en
// blockedReasons, sin romper el resto del cálculo ni hacer fallar el
// comando.
func TestLineaCorruptaDeLedgerSeReportaEnBlockedReasons(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 3, "name": "feature_pendiente", "title": "t", "sdd": false, "status": "pending"}`,
		)},
		".claude/verify-ledger.jsonl": &fstest.MapFile{Data: []byte("esto no es json valido\n")},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS no debería fallar por una línea corrupta del ledger: %v", err)
	}
	found := false
	for _, r := range report.BlockedReasons {
		if strings.Contains(r, "verify-ledger.jsonl") {
			found = true
		}
	}
	if !found {
		t.Errorf("BlockedReasons = %v, se esperaba una entrada identificando la línea corrupta del ledger", report.BlockedReasons)
	}
}

// TestKindReviewEnLedgerNoSeConfundeConKindTest cubre que una entrada
// kind:review para la misma feature no cuenta como evidencia de test — solo
// kind:test importa para no_test_evidence.
func TestKindReviewEnLedgerNoSeConfundeConKindTest(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 5, "name": "verify_record_ledger", "title": "t", "sdd": false, "status": "in_progress"}`,
		)},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	// Una entrada kind:review, en verde y con el treeHash actual — pero no es
	// kind:test, así que no debe evitar no_test_evidence.
	entry := ledgerEntry{Kind: "review", FeatureID: 5, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-27T20:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(5))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, se esperaba no_test_evidence: una entrada kind:review no cuenta como evidencia de test", report.BlockedReasons)
	}
}

// ---- Ticket 02 (feature review_verdict_recorded): no_review_verdict ----

// TestSinEntradaReviewParaFeatureInProgressReportaNoReviewVerdict cubre que
// una feature in_progress sin ninguna entrada kind:review en el ledger
// reporta no_review_verdict, aunque tenga una entrada kind:test en verde y
// vigente (para aislar que el gap es específicamente de revisión).
func TestSinEntradaReviewParaFeatureInProgressReportaNoReviewVerdict(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "test", FeatureID: 6, ExitCode: 0, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z"}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, se esperaba una entrada con no_review_verdict", report.BlockedReasons)
	}
}

// TestReviewChangesRequestedConHashVigenteReportaNoReviewVerdict cubre que
// una última entrada kind:review con verdict CHANGES_REQUESTED no habilita
// cierre, aunque su treeHash coincida con el árbol actual.
func TestReviewChangesRequestedConHashVigenteReportaNoReviewVerdict(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictChangesRequested}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, se esperaba no_review_verdict con verdict CHANGES_REQUESTED", report.BlockedReasons)
	}
}

// TestReviewApprovedConHashVigenteNoReportaNoReviewVerdict cubre que una
// última entrada kind:review con verdict APPROVED y treeHash vigente
// habilita el cierre: no_review_verdict no debe aparecer.
func TestReviewApprovedConHashVigenteNoReportaNoReviewVerdict(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictApproved}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_review_verdict con verdict APPROVED y hash vigente", report.BlockedReasons)
	}
}

// TestReviewApprovedWithObjectionConHashVigenteNoReportaNoReviewVerdict
// cubre que APPROVED_WITH_OBJECTION también habilita cierre igual que
// APPROVED, con treeHash vigente.
func TestReviewApprovedWithObjectionConHashVigenteNoReportaNoReviewVerdict(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictApprovedWithObjection}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_review_verdict con verdict APPROVED_WITH_OBJECTION y hash vigente", report.BlockedReasons)
	}
}

// TestReviewApprovedConHashDesactualizadoReportaNoReviewVerdict cubre que un
// verdict APPROVED con treeHash de una versión vieja del árbol (el código
// cambió después del veredicto) sigue reportando no_review_verdict.
func TestReviewApprovedConHashDesactualizadoReportaNoReviewVerdict(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	oldHash, err := hashTree(fstest.MapFS{"status.go": &fstest.MapFile{Data: []byte("package main\n// version vieja\n")}})
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: oldHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictApproved}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, se esperaba no_review_verdict con treeHash desactualizado", report.BlockedReasons)
	}
}

// TestUltimaEntradaReviewResuelveBloqueoAunqueLaPrimeraSeaChangesRequested
// cubre que, con dos entradas kind:review para la misma feature (primero
// CHANGES_REQUESTED con hash viejo, después APPROVED con hash actual), se
// evalúa la última — no_review_verdict debe estar ausente.
func TestUltimaEntradaReviewResuelveBloqueoAunqueLaPrimeraSeaChangesRequested(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	oldHash, err := hashTree(fstest.MapFS{"status.go": &fstest.MapFile{Data: []byte("package main\n// version vieja\n")}})
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	first := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: oldHash, Timestamp: "2026-08-28T09:00:00Z", Verdict: verdictChangesRequested}
	second := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictApproved}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, first) + ledgerLine(t, second))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_review_verdict: la última entrada kind:review es APPROVED con hash vigente", report.BlockedReasons)
	}
}

// TestKindReviewNuncaCuentaComoEvidenciaDeTest cubre la separación estricta
// en el otro sentido: una entrada kind:review APPROVED vigente, sin ninguna
// entrada kind:test, sigue reportando no_test_evidence.
func TestKindReviewNuncaCuentaComoEvidenciaDeTest(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 6, "name": "review_verdict_recorded", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/review_verdict_recorded/spec.md": &fstest.MapFile{Data: []byte("# spec\n")},
	}
	currentHash, err := hashTree(fsys)
	if err != nil {
		t.Fatalf("hashTree falló: %v", err)
	}
	entry := ledgerEntry{Kind: "review", FeatureID: 6, TreeHash: currentHash, Timestamp: "2026-08-28T10:00:00Z", Verdict: verdictApproved}
	fsys[".claude/verify-ledger.jsonl"] = &fstest.MapFile{Data: []byte(ledgerLine(t, entry))}

	report, err := computeStatusFromFS(fsys, intPtr(6))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_test_evidence") {
		t.Errorf("BlockedReasons = %v, se esperaba no_test_evidence: una entrada kind:review nunca cuenta como evidencia de test", report.BlockedReasons)
	}
}

// TestNoReviewVerdictSoloAplicaAFeatureInProgress cubre que features
// pending/spec_ready/blocked/done nunca reportan no_review_verdict, tengan o
// no entradas kind:review.
func TestNoReviewVerdictSoloAplicaAFeatureInProgress(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 3, "name": "feature_pendiente", "title": "t", "sdd": false, "status": "pending"},` +
				`{"id": 1, "name": "bootstrap_project", "title": "t", "sdd": false, "status": "done"}`,
		)},
	}

	report, err := computeStatusFromFS(fsys, nil)
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_review_verdict") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_review_verdict sin ninguna feature in_progress", report.BlockedReasons)
	}
}

// ---- Ticket 01 (feature spec_gwt_mechanical_check): test de caracterización ----

// featuresSddDoneAlImplementarTicket01 es el conjunto exacto de features
// sdd:true con status:"done" en feature_list.json, verificado explícitamente
// contra el archivo real al momento de escribir este test (31/08/2026): ids
// 2 (april_status_arbiter), 5 (verify_record_ledger), 6
// (review_verdict_recorded), 7 (review_frozen_candidate), 8
// (review_depth_by_diff_sensitivity) y 12 (tree_hash_respects_gitignore).
// Ninguna otra feature sdd:true tiene status:"done" en ese momento (13 es
// sdd:false; 14 es la propia feature de este ticket, in_progress; 15 está
// pending). Ver docs/conventions.md, "Cambios a la lógica de derivación de
// fase" — este test es la red de seguridad obligatoria antes de que el
// ticket 02 de esta misma feature toque computeBlockedReasons.
var featuresSddDoneAlImplementarTicket01 = []struct {
	id   int
	name string
}{
	{2, "april_status_arbiter"},
	{5, "verify_record_ledger"},
	{6, "review_verdict_recorded"},
	{7, "review_frozen_candidate"},
	{8, "review_depth_by_diff_sensitivity"},
	{12, "tree_hash_respects_gitignore"},
}

// readRealFileForCaracterizacion lee un archivo del árbol real de este mismo
// repo (nunca un fixture) — se usa únicamente para poblar fielmente el
// contenido de spec.md/tickets de las features sdd:true ya done dentro del
// fstest.MapFS aislado que arma buildIsolatedDoneFeatureFixture, sin acoplar
// el test a texto tipeado a mano que pueda desincronizarse del archivo real.
func readRealFileForCaracterizacion(t *testing.T, p string) []byte {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("no se pudo leer %s del árbol real: %v", p, err)
	}
	return data
}

// buildIsolatedDoneFeatureFixture arma, para una única feature sdd:true ya
// done, un fstest.MapFS que replica fielmente su spec.md y sus tickets
// reales (leídos del árbol real de este repo vía
// readRealFileForCaracterizacion), pero con un feature_list.json aislado que
// contiene ÚNICAMENTE esa feature — nunca el backlog completo real.
// blockedReasons (status.go, computeBlockedReasons) es una señal GLOBAL que
// recorre TODO feature_list.json: si este test usara el feature_list.json
// real completo (vía os.DirFS(".")), el resultado quedaría contaminado por
// el estado transitorio de cualquier otra feature en curso — en particular,
// la propia feature 14 (spec_gwt_mechanical_check, in_progress mientras se
// escribe este test) reporta hoy no_test_evidence/no_review_verdict, y ese
// ruido desaparecerá en cuanto se registre evidencia para la feature 14 sin
// que eso tenga nada que ver con derivePhase/computeBlockedReasons de las
// seis features ya done. Aislar la feature es lo que hace de este test una
// caracterización estable de esas tres funciones para las seis features
// done, no un espejo frágil del estado momentáneo del resto del backlog.
func buildIsolatedDoneFeatureFixture(t *testing.T, id int, name string) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			fmt.Sprintf(`{"id": %d, "name": %q, "title": "t", "sdd": true, "status": "done"}`, id, name),
		)},
	}

	specPath := specMdPath(name)
	fsys[specPath] = &fstest.MapFile{Data: readRealFileForCaracterizacion(t, specPath)}

	ticketsDir := ticketsDirRaw(name)
	entries, err := os.ReadDir(ticketsDir)
	if err != nil {
		t.Fatalf("no se pudo listar %s del árbol real: %v", ticketsDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := path.Join(ticketsDir, e.Name())
		fsys[p] = &fstest.MapFile{Data: readRealFileForCaracterizacion(t, p)}
	}

	return fsys
}

// TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons es el test
// de caracterización obligatorio de docs/conventions.md ("Cambios a la
// lógica de derivación de fase") que MUST existir antes de que el ticket 02
// de esta misma feature (spec_gwt_mechanical_check) toque
// computeBlockedReasons: fija, como literales hardcodeados (nunca
// recalculados llamando a derivePhase/computeBlockedReasons/
// nextRecommendedText), el phase/blockedReasons/nextRecommended actual de
// cada una de las seis features sdd:true con status:"done" que existen HOY
// en feature_list.json (featuresSddDoneAlImplementarTicket01). Las seis dan
// hoy exactamente el mismo patrón — phase "closed", blockedReasons vacío,
// nextRecommended "nada — ... ya está cerrada" — porque derivePhase corta a
// "closed" en cuanto status == "done" sin mirar el disco, y ninguna de las
// seis cae en la ventana "spec existe, sin tickets, status != done" que va a
// tocar el ticket 02 (las seis ya tienen spec.md y al menos un ticket en
// disco). El ticket 02 debe demostrar que este mismo test sigue pasando
// exactamente igual después de su cambio.
func TestCaracterizacionFeaturesSddDoneAntesDeComputeBlockedReasons(t *testing.T) {
	want := map[int]struct {
		phase           string
		blockedReasons  []string
		nextRecommended string
	}{
		2:  {phaseClosed, []string{}, "nada — la feature 2 (april_status_arbiter) ya está cerrada"},
		5:  {phaseClosed, []string{}, "nada — la feature 5 (verify_record_ledger) ya está cerrada"},
		6:  {phaseClosed, []string{}, "nada — la feature 6 (review_verdict_recorded) ya está cerrada"},
		7:  {phaseClosed, []string{}, "nada — la feature 7 (review_frozen_candidate) ya está cerrada"},
		8:  {phaseClosed, []string{}, "nada — la feature 8 (review_depth_by_diff_sensitivity) ya está cerrada"},
		12: {phaseClosed, []string{}, "nada — la feature 12 (tree_hash_respects_gitignore) ya está cerrada"},
	}

	for _, f := range featuresSddDoneAlImplementarTicket01 {
		t.Run(f.name, func(t *testing.T) {
			fsys := buildIsolatedDoneFeatureFixture(t, f.id, f.name)

			report, err := computeStatusFromFS(fsys, intPtr(f.id))
			if err != nil {
				t.Fatalf("computeStatusFromFS falló para la feature %d (%s): %v", f.id, f.name, err)
			}

			w := want[f.id]
			if report.Phase != w.phase {
				t.Errorf("Phase = %q, se esperaba el literal %q (feature %d, %s)", report.Phase, w.phase, f.id, f.name)
			}
			if !reflect.DeepEqual(report.BlockedReasons, w.blockedReasons) {
				t.Errorf("BlockedReasons = %v, se esperaba el literal %v (feature %d, %s)", report.BlockedReasons, w.blockedReasons, f.id, f.name)
			}
			if report.NextRecommended != w.nextRecommended {
				t.Errorf("NextRecommended = %q, se esperaba el literal %q (feature %d, %s)", report.NextRecommended, w.nextRecommended, f.id, f.name)
			}
		})
	}
}

// ---- Ticket 02 (feature spec_gwt_mechanical_check): no_gwt_coverage ----

// TestSpecSinGWTNiMarcadorNiTicketsReportaNoGwtCoverage cubre US1: una spec
// existente, sin ningún bloque Given/When/Then, sin el marcador de
// opt-out, sin tickets en disco, y con status distinto de done, dispara
// no_gwt_coverage identificando la feature.
func TestSpecSinGWTNiMarcadorNiTicketsReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "pending"}`,
		)},
		"specs/feature_sin_gwt/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nProsa sin bloques Given/When/Then.\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	found := false
	for _, r := range report.BlockedReasons {
		if strings.Contains(r, "no_gwt_coverage") && strings.Contains(r, "20") && strings.Contains(r, "feature_sin_gwt") {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba que blockedReasons tuviera una entrada con no_gwt_coverage identificando id 20 y nombre feature_sin_gwt, se obtuvo %v", report.BlockedReasons)
	}
}

// TestSpecConGWTRealNoReportaNoGwtCoverage cubre US2: una spec con al menos
// un bloque Given/When/Then real no dispara no_gwt_coverage.
func TestSpecConGWTRealNoReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "pending"}`,
		)},
		"specs/feature_sin_gwt/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nGiven algo\nWhen otra cosa\nThen resultado\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_gwt_coverage con un bloque Given/When/Then real", report.BlockedReasons)
	}
}

// TestSpecConMarcadorOptOutNoReportaNoGwtCoverage cubre US3: el marcador
// explícito <!-- gwt: no aplica --> basta por sí solo, sin necesidad de
// ningún bloque Given/When/Then.
func TestSpecConMarcadorOptOutNoReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "pending"}`,
		)},
		"specs/feature_sin_gwt/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nProsa sin GWT.\n\n<!-- gwt: no aplica -->\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_gwt_coverage con el marcador de opt-out presente", report.BlockedReasons)
	}
}

// TestSpecSinGWTConTicketsNoReportaNoGwtCoverage cubre US4: en cuanto la
// feature ya tiene al menos un archivo de ticket en disco, el chequeo deja
// de aplicar — ya pasó la puerta spec→tickets.
func TestSpecSinGWTConTicketsNoReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/feature_sin_gwt/spec.md":              &fstest.MapFile{Data: []byte("# spec\n\nProsa sin GWT.\n")},
		"specs/feature_sin_gwt/tickets/01-nucleo.md": &fstest.MapFile{Data: []byte("# 01\n\n**Blocked by:** None\n\n**Status:** pending\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_gwt_coverage con al menos un ticket ya en disco", report.BlockedReasons)
	}
}

// TestSpecSinGWTConStatusDoneNoReportaNoGwtCoverage cubre US5: una feature
// ya done no vuelve a evaluarse retroactivamente.
func TestSpecSinGWTConStatusDoneNoReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "done"}`,
		)},
		"specs/feature_sin_gwt/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nProsa sin GWT.\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_gwt_coverage con status done", report.BlockedReasons)
	}
}

// TestSpecConGWTYMarcadorSimultaneoNoReportaNoGwtCoverage cubre US19: la
// redundancia (marcador de opt-out + bloques Given/When/Then reales a la
// vez) no se arbitra — la sola presencia del marcador basta.
func TestSpecConGWTYMarcadorSimultaneoNoReportaNoGwtCoverage(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "pending"}`,
		)},
		"specs/feature_sin_gwt/spec.md": &fstest.MapFile{Data: []byte("# spec\n\nGiven algo\nWhen otra cosa\nThen resultado\n\n<!-- gwt: no aplica -->\n")},
	}

	report, err := computeStatusFromFS(fsys, intPtr(20))
	if err != nil {
		t.Fatalf("computeStatusFromFS falló: %v", err)
	}
	if anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("BlockedReasons = %v, no se esperaba no_gwt_coverage con GWT real y marcador de opt-out simultáneos", report.BlockedReasons)
	}
}

// TestStatusYDoctorNoEscribenArchivosConNoGwtCoverage — mismo patrón que
// TestDoctorNoEscribeArchivos (feature 9, doctor_readonly_check): disparar
// no_gwt_coverage vía una feature real en disco no hace que
// `april status`/`april doctor` escriban, borren ni modifiquen ningún
// archivo, corridos varias veces.
func TestStatusYDoctorNoEscribenArchivosConNoGwtCoverage(t *testing.T) {
	dir := chdirTemp(t)
	writeFixtureFile(t, dir, "feature_list.json", testFeatureListJSON(
		`{"id": 20, "name": "feature_sin_gwt", "title": "t", "sdd": true, "status": "pending"}`,
	))
	writeFixtureFile(t, dir, "specs/feature_sin_gwt/spec.md", []byte("# spec\n\nProsa sin bloques Given/When/Then.\n"))

	before := hashTreeSnapshot(t, dir)

	for i := 0; i < 3; i++ {
		runStatusCaptured(t, []string{"--json"})
		runStatusCaptured(t, nil)
		_ = runDoctor(nil)
		_ = runDoctor([]string{"--json"})
	}

	after := hashTreeSnapshot(t, dir)
	if !snapshotsEqual(before, after) {
		t.Errorf("status/doctor modificaron el árbol al disparar no_gwt_coverage — deben ser read-only")
	}
}

// anyContains reporta si algún elemento de ss contiene substr.
func anyContains(ss []string, substr string) bool {
	for _, s := range ss {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// TestCicloEnBlockedByDeTicketsSeDetectaYNoCuelga arma un ciclo directo
// (02 bloqueado por 03, 03 bloqueado por 02) y corre el cálculo con un
// timeout explícito: el comando debe terminar y reportar el ciclo en
// blockedReasons, nunca colgarse.
func TestCicloEnBlockedByDeTicketsSeDetectaYNoCuelga(t *testing.T) {
	fsys := fstest.MapFS{
		"feature_list.json": &fstest.MapFile{Data: testFeatureListJSON(
			`{"id": 2, "name": "april_status_arbiter", "title": "t", "sdd": true, "status": "in_progress"}`,
		)},
		"specs/april_status_arbiter/spec.md":                &fstest.MapFile{Data: []byte("# spec\n")},
		"specs/april_status_arbiter/tickets/02-frontier.md": &fstest.MapFile{Data: []byte("# 02\n\n**Blocked by:** 03\n\n**Status:** pending\n")},
		"specs/april_status_arbiter/tickets/03-cli.md":      &fstest.MapFile{Data: []byte("# 03\n\n**Blocked by:** 02\n\n**Status:** pending\n")},
	}

	type result struct {
		report statusReport
		err    error
	}
	done := make(chan result, 1)
	go func() {
		report, err := computeStatusFromFS(fsys, nil)
		done <- result{report: report, err: err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("computeStatusFromFS falló: %v", r.err)
		}
		found := false
		for _, reason := range r.report.BlockedReasons {
			if strings.Contains(reason, "ciclo detectado") && strings.Contains(reason, "april_status_arbiter") {
				found = true
			}
		}
		if !found {
			t.Errorf("se esperaba un blockedReason de ciclo detectado, se obtuvo %v", r.report.BlockedReasons)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("computeStatusFromFS no terminó en 5s — posible recursión sin límite frente a un ciclo en Blocked by")
	}
}
