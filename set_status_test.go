package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testFeatureListDoc arma un feature_list.json mínimo pero completo, con
// campos extra (description/acceptance) para verificar que set-status no
// los pierde al reescribir el archivo — mismo espíritu que
// testFeatureListJSON en status_test.go, pero con más campos porque acá sí
// se reescribe el documento.
func testFeatureListDoc(featuresJSON string) []byte {
	return []byte(`{
  "rules": {
    "one_feature_at_a_time": true,
    "valid_status": ["pending", "spec_ready", "in_progress", "done", "blocked"]
  },
  "features": [` + featuresJSON + `]
}`)
}

// ---- validTransition: grafo puro, sin I/O ----

// TestValidTransitionGrafoSddTrue cubre el camino feliz completo del grafo
// para una feature sdd:true: pending → spec_ready → in_progress → done.
func TestValidTransitionGrafoSddTrue(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"pending", "spec_ready", true},
		{"spec_ready", "in_progress", true},
		{"in_progress", "done", true},
	}
	for _, c := range cases {
		if got := validTransition(true, c.from, c.to); got != c.want {
			t.Errorf("validTransition(sdd=true, %q, %q) = %v, se esperaba %v", c.from, c.to, got, c.want)
		}
	}
}

// TestValidTransitionSddFalseSaltaSpecReady cubre que una feature sdd:false
// nunca pasa por spec_ready: pending → in_progress es una arista válida
// para ella (no hay spec que aprobar), pero spec_ready no es un estado
// alcanzable ni de salida para sdd:false (decisión documentada en el
// reporte: el grafo colapsa a pending → in_progress → done cuando sdd es
// false).
func TestValidTransitionSddFalseSaltaSpecReady(t *testing.T) {
	if !validTransition(false, "pending", "in_progress") {
		t.Errorf("pending -> in_progress debería ser válido para sdd:false")
	}
	if validTransition(false, "pending", "spec_ready") {
		t.Errorf("pending -> spec_ready no debería ser válido para sdd:false")
	}
	if validTransition(false, "spec_ready", "in_progress") {
		t.Errorf("spec_ready no es un estado de salida válido para sdd:false")
	}
}

// TestValidTransitionRechazaSaltoDirecto cubre el ejemplo explícito de la
// tarea: pending -> done directo, y spec_ready -> done sin pasar por
// in_progress, ambos fuera del grafo.
func TestValidTransitionRechazaSaltoDirecto(t *testing.T) {
	if validTransition(true, "pending", "done") {
		t.Errorf("pending -> done directo debería ser inválido")
	}
	if validTransition(true, "spec_ready", "done") {
		t.Errorf("spec_ready -> done sin pasar por in_progress debería ser inválido")
	}
	if validTransition(false, "pending", "done") {
		t.Errorf("pending -> done directo debería ser inválido (sdd:false también)")
	}
}

// TestValidTransitionBlockedDesdeYHaciaEstadosAbiertos cubre la decisión
// documentada de blocked: alcanzable desde cualquier estado abierto
// (pending, spec_ready si sdd, in_progress), y desde blocked solo de vuelta
// a un estado abierto — nunca directo a done (el cierre siempre pasa por
// in_progress + --verdict).
func TestValidTransitionBlockedDesdeYHaciaEstadosAbiertos(t *testing.T) {
	if !validTransition(true, "pending", "blocked") {
		t.Errorf("pending -> blocked debería ser válido")
	}
	if !validTransition(true, "spec_ready", "blocked") {
		t.Errorf("spec_ready -> blocked debería ser válido")
	}
	if !validTransition(true, "in_progress", "blocked") {
		t.Errorf("in_progress -> blocked debería ser válido")
	}
	if !validTransition(true, "blocked", "in_progress") {
		t.Errorf("blocked -> in_progress debería ser válido")
	}
	if validTransition(true, "blocked", "done") {
		t.Errorf("blocked -> done directo debería ser inválido")
	}
	if validTransition(true, "done", "blocked") {
		t.Errorf("done -> blocked debería ser inválido: done es terminal")
	}
}

// TestValidTransitionRechazaMismoEstado cubre que una transición al mismo
// estado (no-op) no es una arista del grafo.
func TestValidTransitionRechazaMismoEstado(t *testing.T) {
	if validTransition(true, "in_progress", "in_progress") {
		t.Errorf("in_progress -> in_progress (no-op) debería ser inválido")
	}
}

// ---- computeSetStatus: lógica pura sobre bytes, sin tocar disco ----

// TestComputeSetStatusTransicionValidaActualizaStatus cubre el camino
// feliz: pending -> in_progress sobre una feature sdd:false, sin conflicto
// de one_feature_at_a_time, produce el JSON actualizado preservando el
// resto de campos (title, description, acceptance).
func TestComputeSetStatusTransicionValidaActualizaStatus(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "description": "D", "sdd": false, "acceptance": ["a", "b"], "status": "pending"}`)

	out, result, err := computeSetStatus(doc, 4, "in_progress", "")
	if err != nil {
		t.Fatalf("computeSetStatus falló: %v", err)
	}
	if result.From != "pending" || result.To != "in_progress" {
		t.Errorf("result = %+v, se esperaba From=pending To=in_progress", result)
	}

	var got featureListFile
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("la salida no es JSON válido: %v\n%s", err, out)
	}
	if len(got.Features) != 1 || got.Features[0].Status != "in_progress" {
		t.Fatalf("status no actualizado en la salida: %+v", got.Features)
	}

	// campos no relacionados con el estado deben sobrevivir intactos
	if !strings.Contains(string(out), `"description": "D"`) {
		t.Errorf("se perdió el campo description al reescribir, salida:\n%s", out)
	}
	if !strings.Contains(string(out), `"a"`) || !strings.Contains(string(out), `"b"`) {
		t.Errorf("se perdió el campo acceptance al reescribir, salida:\n%s", out)
	}
}

// TestComputeSetStatusInProgressFallaSiOtraYaEstaInProgress cubre
// one_feature_at_a_time: no se puede pasar una segunda feature a
// in_progress mientras otra ya lo está.
func TestComputeSetStatusInProgressFallaSiOtraYaEstaInProgress(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 2, "name": "f2", "title": "T", "sdd": false, "status": "in_progress"},
{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)

	_, _, err := computeSetStatus(doc, 4, "in_progress", "")
	if err == nil {
		t.Fatalf("se esperaba error: ya hay otra feature in_progress")
	}
	if !errors.Is(err, ErrConcurrentInProgress) {
		t.Errorf("error = %v, se esperaba que envolviera ErrConcurrentInProgress", err)
	}
}

// TestComputeSetStatusDoneFallaSinVerdict cubre que done sin --verdict
// falla explícitamente y no escribe nada (out debe venir vacío/no usarse).
func TestComputeSetStatusDoneFallaSinVerdict(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "in_progress"}`)

	_, _, err := computeSetStatus(doc, 4, "done", "")
	if err == nil {
		t.Fatalf("se esperaba error: done sin --verdict")
	}
	if !errors.Is(err, ErrMissingVerdict) {
		t.Errorf("error = %v, se esperaba que envolviera ErrMissingVerdict", err)
	}
}

// TestComputeSetStatusDoneFallaConChangesRequested cubre que
// CHANGES_REQUESTED, aunque es un valor del vocabulario de reviewer_agent,
// no habilita el cierre.
func TestComputeSetStatusDoneFallaConChangesRequested(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "in_progress"}`)

	_, _, err := computeSetStatus(doc, 4, "done", "CHANGES_REQUESTED")
	if err == nil {
		t.Fatalf("se esperaba error: CHANGES_REQUESTED no habilita done")
	}
	if !errors.Is(err, ErrMissingVerdict) {
		t.Errorf("error = %v, se esperaba que envolviera ErrMissingVerdict", err)
	}
}

// TestComputeSetStatusDoneConVerdictValidoLoRegistra cubre el camino feliz
// de cierre: in_progress -> done con --verdict APPROVED registra
// reviewVerdict en la feature.
func TestComputeSetStatusDoneConVerdictValidoLoRegistra(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "in_progress"}`)

	out, result, err := computeSetStatus(doc, 4, "done", "APPROVED_WITH_OBJECTION")
	if err != nil {
		t.Fatalf("computeSetStatus falló: %v", err)
	}
	if result.Verdict != "APPROVED_WITH_OBJECTION" {
		t.Errorf("result.Verdict = %q, se esperaba APPROVED_WITH_OBJECTION", result.Verdict)
	}
	if !strings.Contains(string(out), `"reviewVerdict": "APPROVED_WITH_OBJECTION"`) {
		t.Errorf("no se registró reviewVerdict en la salida:\n%s", out)
	}
}

// TestComputeSetStatusRechazaSaltoDirectoPendingDone cubre el ejemplo
// explícito de la tarea a nivel de computeSetStatus (no solo
// validTransition): pending -> done directo se rechaza con
// ErrInvalidTransition y el error identifica origen y destino.
func TestComputeSetStatusRechazaSaltoDirectoPendingDone(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)

	_, _, err := computeSetStatus(doc, 4, "done", "APPROVED")
	if err == nil {
		t.Fatalf("se esperaba error: pending -> done directo")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("error = %v, se esperaba que envolviera ErrInvalidTransition", err)
	}
	if !strings.Contains(err.Error(), "pending") || !strings.Contains(err.Error(), "done") {
		t.Errorf("error = %v, se esperaba que identificara origen (pending) y destino (done)", err)
	}
}

// TestComputeSetStatusIdInexistenteEsError cubre pedir set-status de un id
// que no existe en feature_list.json.
func TestComputeSetStatusIdInexistenteEsError(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)

	_, _, err := computeSetStatus(doc, 999, "in_progress", "")
	if err == nil {
		t.Fatalf("se esperaba error: id inexistente")
	}
	if !errors.Is(err, ErrFeatureNotFound) {
		t.Errorf("error = %v, se esperaba que envolviera ErrFeatureNotFound", err)
	}
}

// TestComputeSetStatusEstadoDestinoFueraDeVocabularioEsError cubre pedir un
// estado destino que ni siquiera está en rules.valid_status.
func TestComputeSetStatusEstadoDestinoFueraDeVocabularioEsError(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)

	_, _, err := computeSetStatus(doc, 4, "no-existe", "")
	if err == nil {
		t.Fatalf("se esperaba error: estado destino fuera de rules.valid_status")
	}
}

// TestComputeSetStatusPreservaOtrasFeaturesIntactas cubre que set-status
// sobre una feature no modifica ni reordena las demás entradas del array
// (siguen siendo el mismo JSON, byte a byte, salvo la propia feature
// tocada).
func TestComputeSetStatusPreservaOtrasFeaturesIntactas(t *testing.T) {
	doc := testFeatureListDoc(`{"id": 1, "name": "f1", "title": "T1", "sdd": false, "status": "done"},
{"id": 4, "name": "f4", "title": "T4", "sdd": false, "status": "pending"}`)

	out, _, err := computeSetStatus(doc, 4, "in_progress", "")
	if err != nil {
		t.Fatalf("computeSetStatus falló: %v", err)
	}

	var got featureListFile
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("salida no es JSON válido: %v", err)
	}
	if len(got.Features) != 2 {
		t.Fatalf("se esperaban 2 features, se obtuvieron %d", len(got.Features))
	}
	if got.Features[0].ID != 1 || got.Features[0].Status != "done" || got.Features[0].Name != "f1" {
		t.Errorf("la feature 1 no debería haberse tocado: %+v", got.Features[0])
	}
}

// ---- setStatus / runSetStatus: integración con disco real y CLI ----

// TestRunSetStatusEscrituraAtomicaActualizaArchivo cubre el flujo completo
// vía runSetStatus sobre un feature_list.json real en disco.
func TestRunSetStatusEscrituraAtomicaActualizaArchivo(t *testing.T) {
	dir := chdirTemp(t)
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)
	if err := os.WriteFile(filepath.Join(dir, "feature_list.json"), doc, 0644); err != nil {
		t.Fatalf("no se pudo escribir fixture: %v", err)
	}

	exitCode := runSetStatusCaptured(t, []string{"4", "in_progress"})
	if exitCode != 0 {
		t.Fatalf("exit code = %d, se esperaba 0", exitCode)
	}

	data, err := os.ReadFile(filepath.Join(dir, "feature_list.json"))
	if err != nil {
		t.Fatalf("no se pudo leer feature_list.json tras set-status: %v", err)
	}
	var fl featureListFile
	if err := json.Unmarshal(data, &fl); err != nil {
		t.Fatalf("feature_list.json resultante no es JSON válido: %v", err)
	}
	if fl.Features[0].Status != "in_progress" {
		t.Errorf("status en disco = %q, se esperaba in_progress", fl.Features[0].Status)
	}
}

// TestRunSetStatusTransicionInvalidaNoEscribeNada cubre que una transición
// fuera del grafo deja el archivo intacto (mismo hash antes/después) y
// exit code != 0.
func TestRunSetStatusTransicionInvalidaNoEscribeNada(t *testing.T) {
	dir := chdirTemp(t)
	doc := testFeatureListDoc(`{"id": 4, "name": "f4", "title": "T", "sdd": false, "status": "pending"}`)
	if err := os.WriteFile(filepath.Join(dir, "feature_list.json"), doc, 0644); err != nil {
		t.Fatalf("no se pudo escribir fixture: %v", err)
	}

	before := hashDirTree(t, dir)

	exitCode := runSetStatusCaptured(t, []string{"4", "done"})
	if exitCode == 0 {
		t.Fatalf("se esperaba exit code != 0 para transición inválida")
	}

	after := hashDirTree(t, dir)
	if before != after {
		t.Errorf("el árbol cambió pese a que la transición era inválida: before=%s after=%s", before, after)
	}
}

// runSetStatusCaptured corre runSetStatus in-process (sin os.Exit),
// capturando y descartando su stdout/stderr, devolviendo solo el exit
// code — mismo patrón que runStatusCaptured en status_test.go.
func runSetStatusCaptured(t *testing.T, args []string) int {
	t.Helper()
	var exitCode int
	if _, err := captureStdout(t, func() error {
		exitCode = runSetStatus(args)
		return nil
	}); err != nil {
		t.Fatalf("captureStdout falló: %v", err)
	}
	return exitCode
}
