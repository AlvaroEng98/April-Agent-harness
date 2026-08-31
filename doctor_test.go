package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// helper para capturar archivos antes/después y verificar que doctor no modifica nada
func hashTreeSnapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		snapshot[rel] = hashContent(data)
		return nil
	})
	return snapshot
}

func snapshotsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// chdirTemp cambia al directorio temporal y restaura al final
func chdirTempDoctor(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestDoctorSinDriftExitCero(t *testing.T) {
	dest := t.TempDir()
	// Crear manifiesto con un archivo gestionado sin drift
	content := []byte("contenido original")
	if err := os.WriteFile(filepath.Join(dest, "AGENTS.md"), content, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "test.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	// feature_list mínima para que status no falle
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write feature_list: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"AGENTS.md": {Hash: hashContent(content)},
	}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	before := hashTreeSnapshot(t, dest)

	code := runDoctor(nil)
	if code != 0 {
		t.Errorf("se esperaba exit 0 sin drift, se obtuvo %d", code)
	}

	after := hashTreeSnapshot(t, dest)
	if !snapshotsEqual(before, after) {
		t.Errorf("doctor no debe modificar archivos — snapshot cambió")
	}
}

func TestDoctorArchivoModificadoReportaDrift(t *testing.T) {
	dest := t.TempDir()
	original := []byte("original")
	modified := []byte("modificado fuera de flujo")
	if err := os.WriteFile(filepath.Join(dest, "AGENTS.md"), modified, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "a.md"), []byte("# A\n"), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"AGENTS.md": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	before := hashTreeSnapshot(t, dest)

	code := runDoctor(nil)
	if code == 0 {
		t.Errorf("se esperaba exit !=0 por drift modificado")
	}

	// Verificar que reporta archivo + tipo
	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	found := false
	for _, d := range report.Drifts {
		if d.Path == "AGENTS.md" && d.Kind == "modified" {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba drift AGENTS.md modified, se obtuvo %v", report.Drifts)
	}

	after := hashTreeSnapshot(t, dest)
	if !snapshotsEqual(before, after) {
		t.Errorf("doctor no debe modificar archivos")
	}
}

func TestDoctorArchivoBorradoReportaDrift(t *testing.T) {
	dest := t.TempDir()
	content := []byte("original")
	// No crear AGENTS.md en disco — simula borrado
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "a.md"), []byte("# A\n"), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"AGENTS.md": {Hash: hashContent(content)},
	}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	before := hashTreeSnapshot(t, dest)

	code := runDoctor(nil)
	if code == 0 {
		t.Errorf("se esperaba exit !=0 por drift missing")
	}

	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	found := false
	for _, d := range report.Drifts {
		if d.Path == "AGENTS.md" && d.Kind == "missing" {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba drift AGENTS.md missing, se obtuvo %v", report.Drifts)
	}

	after := hashTreeSnapshot(t, dest)
	if !snapshotsEqual(before, after) {
		t.Errorf("doctor no debe modificar archivos")
	}
}

func TestDoctorReportaAgentesEncontrados(t *testing.T) {
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "agent_developer.md"), []byte("# agent_developer\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "planner_agent.md"), []byte("# planner\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	if len(report.Agents) != 2 {
		t.Errorf("se esperaban 2 agentes, se obtuvo %v", report.Agents)
	}
	// Mismo chequeo que init.sh — header "#"
	for _, want := range []string{"agent_developer.md", "planner_agent.md"} {
		found := false
		for _, g := range report.Agents {
			if g == want {
				found = true
			}
		}
		if !found {
			t.Errorf("falta agente %q en %v", want, report.Agents)
		}
	}

	// JSON debe incluir agentes
	code := runDoctor([]string{"--json"})
	if code != 0 {
		t.Errorf("doctor con agentes válidos y sin drift debería exit 0, got %d", code)
	}
}

func TestDoctorNoEscribeArchivos(t *testing.T) {
	dest := t.TempDir()
	content := []byte("contenido")
	if err := os.WriteFile(filepath.Join(dest, "AGENTS.md"), content, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "a.md"), []byte("# A\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"AGENTS.md": {Hash: hashContent(content)},
	}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	before := hashTreeSnapshot(t, dest)

	// Correr doctor varias veces
	for i := 0; i < 3; i++ {
		_ = runDoctor(nil)
		_ = runDoctor([]string{"--json"})
	}

	after := hashTreeSnapshot(t, dest)
	if !snapshotsEqual(before, after) {
		// Detallar diferencias
		for k, v := range before {
			if after[k] != v {
				t.Errorf("archivo %q cambió: antes %s después %s", k, v, after[k])
			}
		}
		for k := range after {
			if _, ok := before[k]; !ok {
				t.Errorf("archivo nuevo %q tras doctor", k)
			}
		}
		t.Errorf("doctor modificó archivos — debe ser read-only")
	}
}

// TestDoctorHeredaNoGwtCoverageSinCodigoPropio dispara no_gwt_coverage
// (ticket 02, feature spec_gwt_mechanical_check) vía una feature real en
// disco y verifica que computeDoctor().BlockedReasons/Healthy lo reflejan
// — documenta que doctor.go no tiene ninguna línea de código propia para
// esto, hereda la señal de computeStatus(nil) tal como ya hace hoy con
// no_test_evidence/no_review_verdict.
func TestDoctorHeredaNoGwtCoverageSinCodigoPropio(t *testing.T) {
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[{"id":20,"name":"feature_sin_gwt","title":"t","sdd":true,"status":"pending"}]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, "specs", "feature_sin_gwt"), 0755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "specs", "feature_sin_gwt", "spec.md"), []byte("# spec sin GWT\n\nProsa sin bloques Given/When/Then.\n"), 0644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	chdirTempDoctor(t, dest)

	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	if !anyContains(report.BlockedReasons, "no_gwt_coverage") {
		t.Errorf("report.BlockedReasons = %v, se esperaba una entrada con no_gwt_coverage", report.BlockedReasons)
	}
	if report.Healthy {
		t.Errorf("report.Healthy = true, se esperaba false con no_gwt_coverage presente")
	}
}

func TestDoctorJSONConDrift(t *testing.T) {
	dest := t.TempDir()
	original := []byte("orig")
	if err := os.WriteFile(filepath.Join(dest, "AGENTS.md"), []byte("cambiado"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".claude", "agents"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, ".claude", "agents", "a.md"), []byte("# A\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[]}`), 0644); err != nil {
		t.Fatalf("write fl: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"AGENTS.md": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	chdirTempDoctor(t, dest)
	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	if len(report.Drifts) == 0 {
		t.Fatalf("se esperaba drift")
	}
	if report.Healthy {
		t.Errorf("con drift no debe ser healthy")
	}

	// Verificar JSON serializa drift con path + kind
	data, _ := json.Marshal(report)
	var parsed doctorReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json roundtrip: %v", err)
	}
	if len(parsed.Drifts) == 0 || parsed.Drifts[0].Path == "" || parsed.Drifts[0].Kind == "" {
		t.Errorf("JSON debe contener path y kind, got %s", string(data))
	}
}

// ---------------------------------------------------------------------
// Feature 11 — doctor_debt_ratchet: métrica de deuda (TODOs sin feature
// asociada) + ratchet contra un baseline congelado.
// ---------------------------------------------------------------------

func TestFindUnlinkedTODOsInContentDetectaTODOSinFeatureAsociada(t *testing.T) {
	featureIDs := map[int]bool{7: true, 12: true}
	content := strings.Join([]string{
		`package main`,
		``,
		`// TODO: esto no menciona ninguna feature — debe contar como deuda`,
		`func a() {}`,
		``,
		`// TODO(alguien): ver feature 7 antes de tocar esto — está asociado`,
		`func b() {}`,
		``,
		`x := 1 // TODO #12 falta manejar el error — asociado, aunque no sea al inicio de línea`,
		``,
		`// computeBlockedReasons calcula blockedReasons sobre TODO feature_list.json`,
		`// — "TODO" acá es la palabra en mayúsculas, no un marcador; no debe contar`,
	}, "\n")

	got := findUnlinkedTODOsInContent("pkg/a.go", content, featureIDs)

	if len(got) != 1 {
		t.Fatalf("se esperaba 1 TODO sin feature asociada, se obtuvo %d: %+v", len(got), got)
	}
	if got[0].Path != "pkg/a.go" || got[0].Line != 3 {
		t.Errorf("se esperaba pkg/a.go:3, se obtuvo %+v", got[0])
	}
}

func TestIsTODOLinkedToFeatureReconocePatrones(t *testing.T) {
	featureIDs := map[int]bool{7: true}

	linked := []string{
		"// TODO: ver feature 7",
		"// TODO: ver feature #7",
		"// TODO: ver #7 para más contexto",
	}
	for _, line := range linked {
		if !isTODOLinkedToFeature(line, featureIDs) {
			t.Errorf("se esperaba asociado a feature 7: %q", line)
		}
	}

	unlinked := []string{
		"// TODO: sin referencia a ninguna feature",
		"// TODO: ver feature 999 — no existe en feature_list.json",
	}
	for _, line := range unlinked {
		if isTODOLinkedToFeature(line, featureIDs) {
			t.Errorf("no se esperaba asociado (feature inexistente o sin referencia): %q", line)
		}
	}
}

func TestComputeTODODebtRecorreArbolYExcluyeGit(t *testing.T) {
	featureIDs := map[int]bool{7: true}
	fsys := fstest.MapFS{
		"a.go": &fstest.MapFile{Data: []byte("package main\n// TODO: sin feature\n")},
		"pkg/b.go": &fstest.MapFile{Data: []byte(
			"package pkg\n// TODO: ver feature 7\nfunc f() {}\n",
		)},
		"pkg/c.go": &fstest.MapFile{Data: []byte(
			"package pkg\n// TODO: otro sin feature\n",
		)},
		"README.md":   &fstest.MapFile{Data: []byte("// TODO esto no es .go, se ignora\n")},
		".git/HEAD":   &fstest.MapFile{Data: []byte("// TODO dentro de .git, debe ignorarse aunque tuviera sufijo .go\n")},
		".git/x/y.go": &fstest.MapFile{Data: []byte("// TODO dentro de .git, debe ignorarse\n")},
	}

	got, err := computeTODODebt(fsys, featureIDs)
	if err != nil {
		t.Fatalf("computeTODODebt: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("se esperaban 2 TODOs sin feature asociada, se obtuvo %d: %+v", len(got), got)
	}
	if got[0].Path != "a.go" || got[1].Path != "pkg/c.go" {
		t.Errorf("se esperaba orden a.go, pkg/c.go — se obtuvo %+v", got)
	}
}

// TestComputeTODODebtIgnoraStringLiteralQuePareceUnTODO cubre de forma
// aislada el mecanismo central de esta feature: go/scanner distingue un
// comentario real de un string literal con el mismo contenido textual. Un
// archivo cuyo único "TODO" vive dentro de un string Go (no en un
// comentario) no debe contar como deuda. Si en el futuro se reemplazara
// go/scanner por un regex sobre texto crudo (o el tokenizer cambiara de
// forma que dejara de distinguir COMMENT de STRING), este test debe fallar.
func TestComputeTODODebtIgnoraStringLiteralQuePareceUnTODO(t *testing.T) {
	featureIDs := map[int]bool{}
	fsys := fstest.MapFS{
		"a.go": &fstest.MapFile{Data: []byte(
			"package main\n\nvar s = \"// TODO: no es un comentario\"\n",
		)},
	}

	got, err := computeTODODebt(fsys, featureIDs)
	if err != nil {
		t.Fatalf("computeTODODebt: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("un TODO dentro de un string literal no debe contar como deuda, se obtuvo %+v", got)
	}
}

func TestParseDoctorBaselineContenidoVacioEsAdopcion(t *testing.T) {
	b, err := parseDoctorBaseline(nil)
	if err != nil {
		t.Fatalf("parseDoctorBaseline(nil): %v", err)
	}
	if b.Metrics == nil || len(b.Metrics) != 0 {
		t.Errorf("se esperaba Metrics vacío no-nil, se obtuvo %#v", b.Metrics)
	}
}

func TestParseDoctorBaselineCorruptoDevuelveError(t *testing.T) {
	if _, err := parseDoctorBaseline([]byte("{no es json")); err == nil {
		t.Errorf("se esperaba error con JSON corrupto")
	}
}

func TestWriteDoctorBaselineLuegoLoadDoctorBaselineRoundtrip(t *testing.T) {
	dest := t.TempDir()

	baseline, found, err := loadDoctorBaseline(dest)
	if err != nil {
		t.Fatalf("loadDoctorBaseline sin archivo: %v", err)
	}
	if found {
		t.Errorf("no se esperaba found=true sin archivo previo")
	}
	if len(baseline.Metrics) != 0 {
		t.Errorf("se esperaba Metrics vacío, se obtuvo %#v", baseline.Metrics)
	}

	toWrite := doctorBaseline{Metrics: map[string]int{todoDebtMetricName: 3}}
	if err := writeDoctorBaseline(dest, toWrite); err != nil {
		t.Fatalf("writeDoctorBaseline: %v", err)
	}

	loaded, found, err := loadDoctorBaseline(dest)
	if err != nil {
		t.Fatalf("loadDoctorBaseline tras escribir: %v", err)
	}
	if !found {
		t.Errorf("se esperaba found=true tras escribir baseline")
	}
	if loaded.Metrics[todoDebtMetricName] != 3 {
		t.Errorf("se esperaba %s=3, se obtuvo %#v", todoDebtMetricName, loaded.Metrics)
	}
}

func TestEvaluateDebtRatchetSinBaselineNuncaFalla(t *testing.T) {
	m := evaluateDebtRatchet(todoDebtMetricName, 5, false, 0)
	if m.Exceeded {
		t.Errorf("sin baseline congelado no debe fallar por deuda, se obtuvo %+v", m)
	}
	if m.BaselineFrozen {
		t.Errorf("se esperaba BaselineFrozen=false")
	}
}

func TestEvaluateDebtRatchetMetricaMenorOIgualNoExcede(t *testing.T) {
	for _, current := range []int{2, 3} {
		m := evaluateDebtRatchet(todoDebtMetricName, current, true, 3)
		if m.Exceeded {
			t.Errorf("current=%d baseline=3 no debería exceder, se obtuvo %+v", current, m)
		}
	}
}

func TestEvaluateDebtRatchetMetricaMayorExcedeConDelta(t *testing.T) {
	m := evaluateDebtRatchet(todoDebtMetricName, 5, true, 3)
	if !m.Exceeded {
		t.Errorf("current=5 baseline=3 debería exceder")
	}
	if m.Delta != 2 {
		t.Errorf("se esperaba delta=2, se obtuvo %d", m.Delta)
	}
}

// fixtureBaseDoctor arma un directorio temporal mínimo para que
// computeDoctor/runDoctor no fallen por causas ajenas a la métrica de
// deuda: sin manifiesto (modo adopción), sin agentes (warn, no fail), y
// feature_list.json con todas las features en done (sin blockedReasons).
func fixtureBaseDoctorDir(t *testing.T, dest string, extraGoFiles map[string]string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), []byte(`{"rules":{"valid_status":["pending","spec_ready","in_progress","done","blocked"]},"features":[{"id":1,"name":"f1","sdd":false,"status":"done"}]}`), 0644); err != nil {
		t.Fatalf("write feature_list: %v", err)
	}
	for name, content := range extraGoFiles {
		full := filepath.Join(dest, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func TestDoctorPrimeraCorridaSinBaselineNoFallaPorDeudaYNoEscribeNada(t *testing.T) {
	dest := t.TempDir()
	fixtureBaseDoctorDir(t, dest, map[string]string{
		"a.go": "package main\n// TODO: sin feature, deuda existente\nfunc a() {}\n",
	})

	chdirTempDoctor(t, dest)
	before := hashTreeSnapshot(t, dest)

	code := runDoctor(nil)
	if code != 0 {
		t.Errorf("primera corrida sin baseline no debe fallar por deuda existente, exit %d", code)
	}

	after := hashTreeSnapshot(t, dest)
	if !snapshotsEqual(before, after) {
		t.Errorf("la corrida por defecto de doctor debe seguir siendo read-only — no debe congelar el baseline sola")
	}
	if _, err := os.Stat(filepath.Join(dest, doctorBaselinePath)); !os.IsNotExist(err) {
		t.Errorf("no se esperaba que se creara %s sin --freeze-baseline", doctorBaselinePath)
	}
}

func TestDoctorFreezeBaselineCongelaValorActual(t *testing.T) {
	dest := t.TempDir()
	fixtureBaseDoctorDir(t, dest, map[string]string{
		"a.go": "package main\n// TODO: uno\n// TODO: dos\nfunc a() {}\n",
	})

	chdirTempDoctor(t, dest)

	code := runDoctor([]string{"--freeze-baseline"})
	if code != 0 {
		t.Fatalf("se esperaba exit 0 al congelar baseline, se obtuvo %d", code)
	}

	baseline, found, err := loadDoctorBaseline(dest)
	if err != nil {
		t.Fatalf("loadDoctorBaseline: %v", err)
	}
	if !found {
		t.Fatalf("se esperaba baseline congelado en disco")
	}
	if baseline.Metrics[todoDebtMetricName] != 2 {
		t.Errorf("se esperaba baseline congelado en 2, se obtuvo %d", baseline.Metrics[todoDebtMetricName])
	}
}

func TestDoctorFreezeBaselineNiegaSobreescribirBaselineExistente(t *testing.T) {
	dest := t.TempDir()
	fixtureBaseDoctorDir(t, dest, map[string]string{
		"a.go": "package main\n// TODO: uno\n",
	})
	chdirTempDoctor(t, dest)

	if code := runDoctor([]string{"--freeze-baseline"}); code != 0 {
		t.Fatalf("primer freeze debería salir 0, se obtuvo %d", code)
	}
	before, _, err := loadDoctorBaseline(dest)
	if err != nil {
		t.Fatalf("loadDoctorBaseline: %v", err)
	}

	// Agregamos deuda nueva antes de reintentar el freeze — no debe colarse.
	if err := os.WriteFile(filepath.Join(dest, "b.go"), []byte("package main\n// TODO: nuevo\n"), 0644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	if code := runDoctor([]string{"--freeze-baseline"}); code == 0 {
		t.Errorf("se esperaba exit != 0 al reintentar congelar un baseline ya existente")
	}

	after, _, err := loadDoctorBaseline(dest)
	if err != nil {
		t.Fatalf("loadDoctorBaseline tras segundo intento: %v", err)
	}
	if after.Metrics[todoDebtMetricName] != before.Metrics[todoDebtMetricName] {
		t.Errorf("el baseline no debía cambiar: antes %d, después %d", before.Metrics[todoDebtMetricName], after.Metrics[todoDebtMetricName])
	}
}

func TestDoctorCorridaPosteriorMetricaMenorOIgualNoFallaPorDeuda(t *testing.T) {
	dest := t.TempDir()
	fixtureBaseDoctorDir(t, dest, map[string]string{
		"a.go": "package main\n// TODO: uno\n// TODO: dos\n",
	})
	chdirTempDoctor(t, dest)

	if code := runDoctor([]string{"--freeze-baseline"}); code != 0 {
		t.Fatalf("freeze inicial debería salir 0, se obtuvo %d", code)
	}

	// Misma cantidad de deuda (2) — no debe fallar por esta causa.
	code := runDoctor(nil)
	if code != 0 {
		t.Errorf("métrica == baseline no debe fallar, exit %d", code)
	}

	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	if len(report.DebtMetrics) != 1 || report.DebtMetrics[0].Exceeded {
		t.Errorf("no se esperaba exceeded=true, se obtuvo %+v", report.DebtMetrics)
	}
}

func TestDoctorCorridaPosteriorMetricaMayorFallaExplicitamenteConMetricaYDelta(t *testing.T) {
	dest := t.TempDir()
	fixtureBaseDoctorDir(t, dest, map[string]string{
		"a.go": "package main\n// TODO: uno\n",
	})
	chdirTempDoctor(t, dest)

	if code := runDoctor([]string{"--freeze-baseline"}); code != 0 {
		t.Fatalf("freeze inicial debería salir 0, se obtuvo %d", code)
	}

	// Deuda nueva sin asociar a ninguna feature — la métrica crece de 1 a 2.
	if err := os.WriteFile(filepath.Join(dest, "b.go"), []byte("package main\n// TODO: nuevo sin feature\n"), 0644); err != nil {
		t.Fatalf("write b.go: %v", err)
	}

	code := runDoctor(nil)
	if code == 0 {
		t.Errorf("se esperaba exit != 0: la deuda creció respecto al baseline")
	}

	report, err := computeDoctor()
	if err != nil {
		t.Fatalf("computeDoctor: %v", err)
	}
	if len(report.DebtMetrics) != 1 {
		t.Fatalf("se esperaba 1 métrica de deuda, se obtuvo %+v", report.DebtMetrics)
	}
	m := report.DebtMetrics[0]
	if !m.Exceeded || m.Delta != 1 || m.Name != todoDebtMetricName {
		t.Errorf("se esperaba métrica excedida, delta=1, name=%s — se obtuvo %+v", todoDebtMetricName, m)
	}
	if report.Healthy {
		t.Errorf("con deuda excedida el reporte no debe ser Healthy")
	}

	// La salida en texto plano debe señalar explícitamente métrica y delta.
	out, err := captureStdout(t, func() error {
		runDoctor(nil)
		return nil
	})
	if err != nil {
		t.Fatalf("captureStdout: %v", err)
	}
	if !strings.Contains(out, todoDebtMetricName) {
		t.Errorf("se esperaba que la salida mencionara la métrica %q, salida: %s", todoDebtMetricName, out)
	}
	if !strings.Contains(out, "+1") && !strings.Contains(out, "delta") {
		t.Errorf("se esperaba que la salida señalara el delta, salida: %s", out)
	}
}
