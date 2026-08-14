package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMergeGitignore cubre main.go:mergeGitignore() en sus tres ramas: el
// caso "destino inexistente" se ejerce llamando directamente a la función
// (que hace os.ReadFile(existingPath) y devuelve error si no existe, la
// misma condición que scaffoldInit evita comprobando os.Stat antes de
// invocarla),
// el caso "sin líneas nuevas que agregar" (el .gitignore existente ya
// contiene todas las entradas no-comentario del template) y el caso "con
// líneas del template ausentes" (se anexan al final, preservando el
// contenido original).
func TestMergeGitignore(t *testing.T) {
	t.Run("destino_inexistente", func(t *testing.T) {
		dir := t.TempDir()
		nonExistent := filepath.Join(dir, "no-existe", ".gitignore")

		_, err := mergeGitignore(nonExistent, []byte("node_modules/\n"))
		if err == nil {
			t.Fatalf("se esperaba error al leer un archivo destino inexistente, no hubo error")
		}
	})

	t.Run("sin_lineas_nuevas", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, ".gitignore")
		existingContent := "# comentario\nnode_modules/\n.env\n"
		if err := os.WriteFile(dest, []byte(existingContent), 0644); err != nil {
			t.Fatalf("no se pudo escribir .gitignore existente: %v", err)
		}

		template := "# comentario distinto\nnode_modules/\n.env\n"

		merged, err := mergeGitignore(dest, []byte(template))
		if err != nil {
			t.Fatalf("mergeGitignore falló: %v", err)
		}
		if string(merged) != existingContent {
			t.Errorf("se esperaba el contenido sin cambios %q, se obtuvo %q", existingContent, string(merged))
		}
	})

	t.Run("lineas_ausentes_se_agregan", func(t *testing.T) {
		dir := t.TempDir()
		dest := filepath.Join(dir, ".gitignore")
		existingContent := "node_modules/\n"
		if err := os.WriteFile(dest, []byte(existingContent), 0644); err != nil {
			t.Fatalf("no se pudo escribir .gitignore existente: %v", err)
		}

		template := "node_modules/\n# comentario\n.env\ndist/\n"

		merged, err := mergeGitignore(dest, []byte(template))
		if err != nil {
			t.Fatalf("mergeGitignore falló: %v", err)
		}

		got := string(merged)
		if !strings.HasPrefix(got, existingContent) {
			t.Errorf("el contenido original debe preservarse al inicio, se obtuvo %q", got)
		}
		for _, want := range []string{".env", "dist/"} {
			if !strings.Contains(got, want) {
				t.Errorf("falta la línea ausente %q en el resultado:\n%s", want, got)
			}
		}
		if strings.Contains(got, "# comentario") {
			t.Errorf("las líneas de comentario del template no deben copiarse:\n%s", got)
		}
	})
}

// TestPrintUsageMentionsApil cubre main.go:printUsage() — el CLI se invoca
// como `apil <command>`, no `harness <command>` (feature rename_cli_to_apil).
func TestPrintUsageMentionsApil(t *testing.T) {
	out, err := captureStdout(t, func() error {
		printUsage()
		return nil
	})
	if err != nil {
		t.Fatalf("printUsage no debería fallar: %v", err)
	}
	if !strings.Contains(out, "Usage: apil <command>") {
		t.Errorf("se esperaba 'Usage: apil <command>' en la salida, se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de printUsage, se obtuvo:\n%s", out)
	}
}

// TestVersionCommandPrintsApil cubre main.go:main() rama "version" — tras
// la objeción del reviewer en progress/review_rename_cli_to_apil.md,
// `apil version` debía seguir imprimiendo "harness v<versión>"; este test
// evita que la regresión vuelva a colarse sin fallar el build.
func TestVersionCommandPrintsApil(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"apil", "version"}
	defer func() { os.Args = origArgs }()

	out, err := captureStdout(t, func() error {
		main()
		return nil
	})
	if err != nil {
		t.Fatalf("main no debería fallar: %v", err)
	}
	if !strings.HasPrefix(out, "apil v") {
		t.Errorf("se esperaba que la salida empezara con 'apil v', se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de 'version', se obtuvo:\n%s", out)
	}
}

// captureStdout redirige temporalmente os.Stdout mientras corre fn y
// devuelve todo lo impreso. scaffoldInit imprime mensajes informativos por
// stdout que algunos tests necesitan inspeccionar (p.ej. el aviso de
// "harness existente").
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("no se pudo crear pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	fnErr := fn()

	os.Stdout = origStdout
	w.Close()

	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, readErr := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}
	r.Close()

	return string(buf), fnErr
}

// TestCmdInitScaffoldsEmptyDir cubre la rama "feliz" de scaffoldInit (la
// lógica de scaffolding de cmdInit, extraída para poder testearla
// in-process): un directorio vacío recibe el scaffold completo del template
// embebido, incluidos archivos de la raíz (AGENT.md), el estado inicial
// (feature_list.json) y el árbol .claude/agents/.
func TestCmdInitScaffoldsEmptyDir(t *testing.T) {
	dest := t.TempDir()

	if _, err := captureStdout(t, func() error {
		return scaffoldInit(dest)
	}); err != nil {
		t.Fatalf("scaffoldInit falló: %v", err)
	}

	cases := []struct {
		relPath string
		srcPath string
	}{
		{"AGENT.md", "AGENT.md"},
		{filepath.Join(".claude", "agents", "orquestador.md"), filepath.Join(".claude", "agents", "orquestador.md")},
	}
	for _, c := range cases {
		gotPath := filepath.Join(dest, c.relPath)
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatalf("no se creó %s: %v", c.relPath, err)
		}
		want, err := os.ReadFile(c.srcPath)
		if err != nil {
			t.Fatalf("no se pudo leer el fixture fuente %s: %v", c.srcPath, err)
		}
		if string(got) != string(want) {
			t.Errorf("%s no coincide con el contenido embebido esperado", c.relPath)
		}
	}

	// feature_list.json viene de templates/feature_list.json (el lienzo
	// limpio), no de la raíz del repo (que tiene el estado real de trabajo).
	featureListPath := filepath.Join(dest, "feature_list.json")
	got, err := os.ReadFile(featureListPath)
	if err != nil {
		t.Fatalf("no se creó feature_list.json: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("templates", "feature_list.json"))
	if err != nil {
		t.Fatalf("no se pudo leer templates/feature_list.json: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("feature_list.json no coincide con el template embebido esperado")
	}
}

// TestCmdInitExistingHarnessRegeneratesAgents cubre la rama
// "isExistingHarness" de scaffoldInit (la lógica de cmdInit): cuando el
// directorio destino ya contiene AGENT.md o feature_list.json,
// .claude/agents/ se borra por completo antes de volver a escribirse desde
// el template embebido.
func TestCmdInitExistingHarnessRegeneratesAgents(t *testing.T) {
	dest := t.TempDir()

	// Simula un harness preexistente con un agente "viejo" que no forma
	// parte del template actual.
	if err := os.WriteFile(filepath.Join(dest, "AGENT.md"), []byte("# AGENT.md viejo\n"), 0644); err != nil {
		t.Fatalf("no se pudo preparar AGENT.md preexistente: %v", err)
	}
	oldAgentsDir := filepath.Join(dest, ".claude", "agents")
	if err := os.MkdirAll(oldAgentsDir, 0755); err != nil {
		t.Fatalf("no se pudo preparar .claude/agents/ preexistente: %v", err)
	}
	oldAgentPath := filepath.Join(oldAgentsDir, "agente_obsoleto.md")
	if err := os.WriteFile(oldAgentPath, []byte("# agente obsoleto\n"), 0644); err != nil {
		t.Fatalf("no se pudo preparar agente obsoleto: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return scaffoldInit(dest)
	})
	if err != nil {
		t.Fatalf("scaffoldInit falló: %v", err)
	}
	if !strings.Contains(out, "Existing harness project detected") {
		t.Errorf("se esperaba el aviso de harness existente en la salida, se obtuvo:\n%s", out)
	}

	if _, err := os.Stat(oldAgentPath); !os.IsNotExist(err) {
		t.Errorf("agente_obsoleto.md debería haberse borrado al regenerar .claude/agents/, err=%v", err)
	}

	regenerated := filepath.Join(dest, ".claude", "agents", "orquestador.md")
	if _, err := os.Stat(regenerated); err != nil {
		t.Errorf(".claude/agents/orquestador.md debería haberse regenerado: %v", err)
	}
}
