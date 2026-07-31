package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupRecapDir arma un directorio temporal con una copia ejecutable de
// recap.sh y los fixtures indicados, y devuelve su ruta. Como recap.sh se
// auto-localiza vía BASH_SOURCE, ejecutarlo desde esta copia hace que sus
// rutas relativas resuelvan contra el directorio temporal.
func setupRecapDir(t *testing.T, history, featureList, current string) string {
	t.Helper()

	dir := t.TempDir()

	src, err := os.ReadFile("recap.sh")
	if err != nil {
		t.Fatalf("no se pudo leer recap.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "recap.sh"), src, 0755); err != nil {
		t.Fatalf("no se pudo escribir recap.sh temporal: %v", err)
	}

	if history != "" {
		progressDir := filepath.Join(dir, "progress")
		if err := os.MkdirAll(progressDir, 0755); err != nil {
			t.Fatalf("no se pudo crear progress/: %v", err)
		}
		if err := os.WriteFile(filepath.Join(progressDir, "history.md"), []byte(history), 0644); err != nil {
			t.Fatalf("no se pudo escribir history.md: %v", err)
		}
	}

	if featureList != "" {
		if err := os.WriteFile(filepath.Join(dir, "feature_list.json"), []byte(featureList), 0644); err != nil {
			t.Fatalf("no se pudo escribir feature_list.json: %v", err)
		}
	}

	if current != "" {
		progressDir := filepath.Join(dir, "progress")
		if err := os.MkdirAll(progressDir, 0755); err != nil {
			t.Fatalf("no se pudo crear progress/: %v", err)
		}
		if err := os.WriteFile(filepath.Join(progressDir, "current.md"), []byte(current), 0644); err != nil {
			t.Fatalf("no se pudo escribir current.md: %v", err)
		}
	}

	return dir
}

// runRecap ejecuta la copia de recap.sh del directorio dado y devuelve stdout.
func runRecap(t *testing.T, dir string) string {
	t.Helper()

	out, err := exec.Command("bash", filepath.Join(dir, "recap.sh")).Output()
	if err != nil {
		t.Fatalf("recap.sh falló: %v", err)
	}
	return string(out)
}

// TestRecapAllPieces verifica las 3 líneas del recap con datos completos:
// última sesión (R2), feature actual no-done (R3) y sesión activa (R5).
func TestRecapAllPieces(t *testing.T) {
	history := "# Session History\n\n## 2026-07-16 — Feature: centralize_config (ID: 1)\n\ncontenido\n"
	featureList := `{"features":[{"id":1,"title":"Centralize Config","status":"done"},{"id":2,"title":"Recap Hook","status":"in_progress"}]}`
	current := "# Current Session\n\n- Name: auto_recap_hook\n- Status: in_progress\n"

	dir := setupRecapDir(t, history, featureList, current)
	out := runRecap(t, dir)

	wants := []string{
		"Última sesión: 2026-07-16 — Feature: centralize_config (ID: 1)",
		"Feature actual: Recap Hook (in_progress)",
		"Sesión activa: auto_recap_hook (in_progress)",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("falta línea esperada %q en salida:\n%s", w, out)
		}
	}
}

// TestRecapAllFeaturesDone cubre el caso borde R4: todas las features done.
func TestRecapAllFeaturesDone(t *testing.T) {
	featureList := `{"features":[{"id":1,"title":"A","status":"done"},{"id":2,"title":"B","status":"done"}]}`
	current := "- Name: algo\n- Status: in_progress\n"

	dir := setupRecapDir(t, "", featureList, current)
	out := runRecap(t, dir)

	want := "Feature actual: Todas las features completadas"
	if !strings.Contains(out, want) {
		t.Errorf("falta %q en salida:\n%s", want, out)
	}
}

// TestRecapNoActiveSession cubre el caso borde R6: sin sesión activa
// (línea "- Name:" ausente).
func TestRecapNoActiveSession(t *testing.T) {
	featureList := `{"features":[{"id":1,"title":"A","status":"in_progress"}]}`
	current := "# Current Session\n\nSin sesión activa por ahora.\n"

	dir := setupRecapDir(t, "", featureList, current)
	out := runRecap(t, dir)

	want := "No hay sesión activa"
	if !strings.Contains(out, want) {
		t.Errorf("falta %q en salida:\n%s", want, out)
	}
	if strings.Contains(out, "Sesión activa:") {
		t.Errorf("no debería haber línea 'Sesión activa:' en salida:\n%s", out)
	}
}
