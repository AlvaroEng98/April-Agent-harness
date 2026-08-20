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

// TestPlanScaffoldIsPure cubre main.go:planScaffold() de forma aislada, sin
// pasar por applyPlan: verifica que planScaffold toma decisiones correctas
// (el modo de init.sh en el plan es 0755, feature_list.json queda con modo
// 0644 y su contenido viene del template embebido) sin escribir ni un solo
// archivo en disco (feature scaffold_decision_io_seam, acceptance A2/A7).
func TestPlanScaffoldIsPure(t *testing.T) {
	// dest no existe todavía (nested dentro de un tempdir vacío): ejercita la
	// rama createTargetDir=true de planScaffold sin que planScaffold llegue a
	// crearlo (eso le corresponde a applyPlan).
	dest := filepath.Join(t.TempDir(), "nested", "dest")

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("se esperaba que %s no existiera antes de planificar, err=%v", dest, err)
	}

	plan, err := planScaffold(dest)
	if err != nil {
		t.Fatalf("planScaffold falló: %v", err)
	}

	if !plan.createTargetDir {
		t.Errorf("se esperaba createTargetDir=true para un directorio inexistente al llamar os.ReadDir por primera vez")
	}

	var initSh, featureList *scaffoldFileWrite
	for i := range plan.files {
		switch plan.files[i].relPath {
		case "init.sh":
			initSh = &plan.files[i]
		case "feature_list.json":
			featureList = &plan.files[i]
		}
	}
	if initSh == nil {
		t.Fatalf("el plan no incluye init.sh")
	}
	if initSh.mode != 0755 {
		t.Errorf("se esperaba modo 0755 para init.sh en el plan, se obtuvo %o", initSh.mode)
	}
	if featureList == nil {
		t.Fatalf("el plan no incluye feature_list.json")
	}
	if featureList.mode != 0644 {
		t.Errorf("se esperaba modo 0644 para feature_list.json en el plan, se obtuvo %o", featureList.mode)
	}
	if featureList.isUpdate {
		t.Errorf("feature_list.json no debería marcarse como merge/update en un destino vacío")
	}

	wantDirs := []string{
		filepath.Join(dest, "specs"),
	}
	for i, want := range wantDirs {
		if i >= len(plan.emptyDirs) || plan.emptyDirs[i] != want {
			t.Errorf("se esperaba emptyDirs[%d]=%q, se obtuvo plan.emptyDirs=%v", i, want, plan.emptyDirs)
			break
		}
	}

	// planScaffold no debe haber creado nada en disco: ni siquiera dest
	// (createTargetDir queda como decisión en el plan, la ejecuta applyPlan).
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("planScaffold no debe escribir en disco: %s existe tras llamarlo, err=%v", dest, err)
	}
}

// captureStdout redirige temporalmente os.Stdout mientras corre fn y
// devuelve todo lo impreso. scaffoldInit imprime mensajes informativos por
// stdout que algunos tests necesitan inspeccionar (p.ej. el aviso de
// "harness existente"). También la usan los tests de dispatch en
// main_test.go.
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

// TestCmdInitGitignoreEsMinimo cubre que el .gitignore que escribe
// scaffoldInit en un destino vacío viene de templates/.gitignore (el lienzo
// mínimo para el proyecto scaffoldeado: specs/, tests/, session-handoff.md),
// no del .gitignore de la raíz de este repo (que trae reglas propias del
// desarrollo del harness — OS, IDE, build de Go — irrelevantes para el
// destino). Regresión directa: go:embed excluye por defecto los archivos que
// empiezan con "." dentro de un patrón de directorio salvo que se declare
// "all:templates"; sin ese prefijo, templates/.gitignore no se empotra y el
// destino queda sin .gitignore.
func TestCmdInitGitignoreEsMinimo(t *testing.T) {
	dest := t.TempDir()

	if _, err := captureStdout(t, func() error {
		return scaffoldInit(dest)
	}); err != nil {
		t.Fatalf("scaffoldInit falló: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, ".gitignore"))
	if err != nil {
		t.Fatalf("no se creó .gitignore en el destino: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("templates", ".gitignore"))
	if err != nil {
		t.Fatalf("no se pudo leer templates/.gitignore: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf(".gitignore del destino no coincide con templates/.gitignore\nesperado:\n%s\nobtenido:\n%s", want, got)
	}

	for _, unwanted := range []string{".DS_Store", "__pycache__", "GUIA-INTEGRACION-SKILLS.md"} {
		if strings.Contains(string(got), unwanted) {
			t.Errorf(".gitignore del destino no debería traer reglas del propio harness (%q)", unwanted)
		}
	}
}

// TestCmdInitGitignoreExistenteHaceMerge cubre que, si el destino ya tiene un
// .gitignore propio, scaffoldInit lo conserva y solo añade al final las
// líneas del template que falten (comportamiento de mergeGitignore, ejercido
// aquí a través del flujo completo de scaffoldInit).
func TestCmdInitGitignoreExistenteHaceMerge(t *testing.T) {
	dest := t.TempDir()
	existing := "node_modules/\n.env\n"
	if err := os.WriteFile(filepath.Join(dest, ".gitignore"), []byte(existing), 0644); err != nil {
		t.Fatalf("no se pudo preparar .gitignore preexistente: %v", err)
	}

	if _, err := captureStdout(t, func() error {
		return scaffoldInit(dest)
	}); err != nil {
		t.Fatalf("scaffoldInit falló: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, ".gitignore"))
	if err != nil {
		t.Fatalf("no se pudo leer .gitignore del destino: %v", err)
	}
	if !strings.HasPrefix(string(got), existing) {
		t.Errorf("el .gitignore preexistente debe preservarse al inicio, se obtuvo:\n%s", got)
	}
	for _, want := range []string{"specs/", "tests/", "/session-handoff.md"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("falta la línea %q tras el merge:\n%s", want, got)
		}
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
