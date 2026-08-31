package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

// TestWriteManifestThenLoadManifestRoundtrip cubre el par
// writeManifest/loadManifest de forma aislada, sin pasar por planScaffold ni
// applyPlan: lo que se escribe con writeManifest debe leerse de vuelta igual
// con loadManifest, con found=true y corrupt=false (el manifiesto sí existía
// y era válido).
func TestWriteManifestThenLoadManifestRoundtrip(t *testing.T) {
	dest := t.TempDir()

	want := manifest{
		Files: map[string]manifestEntry{
			"AGENTS.md":         {Hash: hashContent([]byte("contenido AGENTS.md"))},
			"feature_list.json": {Hash: hashContent([]byte("contenido feature_list.json"))},
		},
	}

	if err := writeManifest(dest, want); err != nil {
		t.Fatalf("writeManifest falló: %v", err)
	}

	result := loadManifest(dest)
	if !result.found {
		t.Errorf("se esperaba found=true tras escribir el manifiesto")
	}
	if result.corrupt {
		t.Errorf("se esperaba corrupt=false para un manifiesto recién escrito")
	}
	if result.manifest.SchemaVersion != manifestSchemaVersion {
		t.Errorf("SchemaVersion = %d, se esperaba %d", result.manifest.SchemaVersion, manifestSchemaVersion)
	}
	for path, entry := range want.Files {
		got, ok := result.manifest.Files[path]
		if !ok {
			t.Errorf("falta la entrada %q tras el roundtrip", path)
			continue
		}
		if got.Hash != entry.Hash {
			t.Errorf("hash de %q = %q, se esperaba %q", path, got.Hash, entry.Hash)
		}
	}
}

// TestLoadManifestAusenteEsAdopcion cubre que loadManifest sobre un destino
// sin .claude/manifest.json devuelve found=false, corrupt=false y un
// manifiesto vacío (nunca un error fatal): es la señal para entrar en modo
// adopción, no un caso de error.
func TestLoadManifestAusenteEsAdopcion(t *testing.T) {
	dest := t.TempDir()

	result := loadManifest(dest)
	if result.found {
		t.Errorf("se esperaba found=false: no hay manifiesto en %s", dest)
	}
	if result.corrupt {
		t.Errorf("se esperaba corrupt=false: la ausencia de manifiesto no es corrupción")
	}
	if len(result.manifest.Files) != 0 {
		t.Errorf("se esperaba un manifiesto vacío, se obtuvo %v", result.manifest.Files)
	}
}

// TestPlanScaffoldIsPure cubre main.go:planScaffold() de forma aislada, sin
// pasar por applyPlan: verifica que planScaffold toma decisiones correctas
// (el modo de init.sh en el plan es 0755, feature_list.json queda con modo
// 0644 y su contenido viene del template embebido, y se marca como
// actionCreate porque no hay manifiesto previo) sin escribir ni un solo
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
	if plan.manifestFound {
		t.Errorf("se esperaba manifestFound=false: no hay .claude/manifest.json en un destino inexistente")
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
	if featureList.action != actionCreate {
		t.Errorf("action = %v, se esperaba actionCreate en un destino vacío sin manifiesto previo", featureList.action)
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
// embebido, incluidos archivos de la raíz (AGENTS.md), el estado inicial
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
		{"AGENTS.md", "AGENTS.md"},
		{filepath.Join(".claude", "agents", "spec_writer.md"), filepath.Join(".claude", "agents", "spec_writer.md")},
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

// TestArchivoNuevoDelTemplateSeEscribeYSeRegistra cubre la celda "create" de
// la tabla de decisión: un archivo que no estaba en el manifiesto anterior
// (nuevo.txt) se escribe con el contenido de la plantilla y se registra en
// el manifiesto del plan con su hash.
func TestArchivoNuevoDelTemplateSeEscribeYSeRegistra(t *testing.T) {
	dest := t.TempDir()
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"viejo.txt": {Hash: hashContent([]byte("contenido viejo"))},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "viejo.txt"), []byte("contenido viejo"), 0644); err != nil {
		t.Fatalf("no se pudo preparar viejo.txt: %v", err)
	}

	tmplFS := fstest.MapFS{
		"viejo.txt": {Data: []byte("contenido viejo")},
		"nuevo.txt": {Data: []byte("contenido nuevo del template")},
	}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	var nuevo *scaffoldFileWrite
	for i := range plan.files {
		if plan.files[i].relPath == "nuevo.txt" {
			nuevo = &plan.files[i]
		}
	}
	if nuevo == nil {
		t.Fatalf("el plan no incluye nuevo.txt")
	}
	if nuevo.action != actionCreate {
		t.Errorf("action = %v, se esperaba actionCreate", nuevo.action)
	}

	wantHash := hashContent([]byte("contenido nuevo del template"))
	if entry, ok := plan.manifest.Files["nuevo.txt"]; !ok || entry.Hash != wantHash {
		t.Errorf("el manifiesto del plan no registra nuevo.txt con el hash correcto, got=%v", plan.manifest.Files["nuevo.txt"])
	}
}

// TestArchivoNoTocadoPorUsuarioSeActualiza cubre la celda "update": el disco
// coincide con el hash del manifiesto anterior (el usuario no tocó el
// archivo), así que se sobreescribe con la plantilla nueva y se actualiza el
// hash registrado.
func TestArchivoNoTocadoPorUsuarioSeActualiza(t *testing.T) {
	dest := t.TempDir()
	if err := os.WriteFile(filepath.Join(dest, "config.txt"), []byte("v1"), 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"config.txt": {Hash: hashContent([]byte("v1"))},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{"config.txt": {Data: []byte("v2")}}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	var fw *scaffoldFileWrite
	for i := range plan.files {
		if plan.files[i].relPath == "config.txt" {
			fw = &plan.files[i]
		}
	}
	if fw == nil {
		t.Fatalf("el plan no incluye config.txt")
	}
	if fw.action != actionUpdate {
		t.Errorf("action = %v, se esperaba actionUpdate", fw.action)
	}
	if string(fw.content) != "v2" {
		t.Errorf("content = %q, se esperaba el contenido de la plantilla nueva %q", fw.content, "v2")
	}

	wantHash := hashContent([]byte("v2"))
	if entry := plan.manifest.Files["config.txt"]; entry.Hash != wantHash {
		t.Errorf("el manifiesto del plan no actualiza el hash de config.txt, got=%v, want=%q", entry, wantHash)
	}
}

// TestArchivoTocadoPorUsuarioYTemplateSinCambios_NoSeToca cubre la celda
// "skip silencioso": el disco no coincide con el manifiesto (el usuario tocó
// el archivo) pero la plantilla tampoco cambió respecto de lo registrado, así
// que se deja tal cual sin avisar.
func TestArchivoTocadoPorUsuarioYTemplateSinCambios_NoSeToca(t *testing.T) {
	dest := t.TempDir()
	original := []byte("original")
	edited := []byte("editado por el usuario")
	destFile := filepath.Join(dest, "config.txt")
	if err := os.WriteFile(destFile, edited, 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"config.txt": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{"config.txt": {Data: original}}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	var fw *scaffoldFileWrite
	for i := range plan.files {
		if plan.files[i].relPath == "config.txt" {
			fw = &plan.files[i]
		}
	}
	if fw == nil {
		t.Fatalf("el plan no incluye config.txt")
	}
	if fw.action != actionSkipUnmodified {
		t.Errorf("action = %v, se esperaba actionSkipUnmodified", fw.action)
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if strings.Contains(out, "config.txt") {
		t.Errorf("no debería avisarse sobre config.txt en un skip silencioso, se obtuvo:\n%s", out)
	}

	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("no se pudo leer config.txt: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("config.txt no debería tocarse, se obtuvo %q", got)
	}
}

// TestArchivoTocadoPorUsuarioYTemplateCambio_Conflicto cubre la celda "skip
// con aviso": el disco no coincide con el manifiesto (usuario tocó) y la
// plantilla también cambió (conflicto real), así que se conserva la versión
// del usuario y se imprime un aviso mencionando el archivo.
func TestArchivoTocadoPorUsuarioYTemplateCambio_Conflicto(t *testing.T) {
	dest := t.TempDir()
	original := []byte("original")
	edited := []byte("editado por el usuario")
	destFile := filepath.Join(dest, "config.txt")
	if err := os.WriteFile(destFile, edited, 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"config.txt": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{"config.txt": {Data: []byte("plantilla nueva")}}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	var fw *scaffoldFileWrite
	for i := range plan.files {
		if plan.files[i].relPath == "config.txt" {
			fw = &plan.files[i]
		}
	}
	if fw == nil {
		t.Fatalf("el plan no incluye config.txt")
	}
	if fw.action != actionSkipConflict {
		t.Errorf("action = %v, se esperaba actionSkipConflict", fw.action)
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if !strings.Contains(out, "config.txt") {
		t.Errorf("se esperaba un aviso mencionando config.txt, se obtuvo:\n%s", out)
	}

	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("no se pudo leer config.txt: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("config.txt no debería sobreescribirse en un conflicto, se obtuvo %q", got)
	}
}

// TestArchivoObsoletoNoModificadoSeBorra cubre el borrado de archivos que ya
// no vienen en la plantilla nueva: si el disco coincide con el hash
// registrado (el usuario no lo tocó), se borra.
func TestArchivoObsoletoNoModificadoSeBorra(t *testing.T) {
	dest := t.TempDir()
	oldPath := filepath.Join(dest, "old.txt")
	content := []byte("contenido antiguo")
	if err := os.WriteFile(oldPath, content, 0644); err != nil {
		t.Fatalf("no se pudo preparar old.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"old.txt": {Hash: hashContent(content)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{} // la plantilla ya no trae old.txt

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	found := false
	for _, del := range plan.filesToDelete {
		if del.relPath == "old.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("se esperaba old.txt en filesToDelete, got=%v", plan.filesToDelete)
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan) }); err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old.txt debería haberse borrado, err=%v", err)
	}
}

// TestArchivoObsoletoModificadoNoSeBorra cubre la misma situación pero con el
// usuario habiendo modificado el archivo: no coincide con el hash
// registrado, así que se conserva en vez de borrarse.
func TestArchivoObsoletoModificadoNoSeBorra(t *testing.T) {
	dest := t.TempDir()
	oldPath := filepath.Join(dest, "old.txt")
	original := []byte("contenido original")
	edited := []byte("contenido editado por el usuario")
	if err := os.WriteFile(oldPath, edited, 0644); err != nil {
		t.Fatalf("no se pudo preparar old.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"old.txt": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	for _, del := range plan.filesToDelete {
		if del.relPath == "old.txt" {
			t.Errorf("old.txt modificado por el usuario no debería borrarse")
		}
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan) }); err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}

	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("old.txt modificado debería conservarse, err=%v", err)
	}
}

// TestPrimeraCorridaSinManifiesto_ModoAdopcion cubre el modo adopción: sin
// .claude/manifest.json previo, no se sobreescribe ni se borra nada que ya
// exista en disco (el usuario venía trabajando sobre un scaffold de una
// versión anterior de april); solo se crean los archivos de plantilla que
// falten por completo y se adopta el hash de lo existente como línea base.
func TestPrimeraCorridaSinManifiesto_ModoAdopcion(t *testing.T) {
	dest := t.TempDir()
	existing := []byte("feature list del usuario, con progreso real")
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), existing, 0644); err != nil {
		t.Fatalf("no se pudo preparar feature_list.json: %v", err)
	}

	tmplFS := fstest.MapFS{
		"feature_list.json": {Data: []byte("feature list nuevo del template")},
		"AGENTS.md":         {Data: []byte("contenido AGENTS.md")},
	}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}
	if plan.manifestFound {
		t.Errorf("se esperaba manifestFound=false: no hay manifiesto previo")
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if !strings.Contains(out, "adopci") {
		t.Errorf("se esperaba un aviso de modo adopción en la salida, se obtuvo:\n%s", out)
	}

	got, err := os.ReadFile(filepath.Join(dest, "feature_list.json"))
	if err != nil {
		t.Fatalf("no se pudo leer feature_list.json: %v", err)
	}
	if string(got) != string(existing) {
		t.Errorf("modo adopción no debe sobreescribir feature_list.json existente, se obtuvo %q", got)
	}

	if _, err := os.Stat(filepath.Join(dest, "AGENTS.md")); err != nil {
		t.Errorf("modo adopción debe crear los archivos de plantilla que falten por completo: %v", err)
	}

	result := loadManifest(dest)
	wantHash := hashContent(existing)
	if entry := result.manifest.Files["feature_list.json"]; entry.Hash != wantHash {
		t.Errorf("modo adopción debe registrar el hash del contenido en disco, got=%v, want=%q", entry, wantHash)
	}
}

// TestSegundaCorridaTrasAdopcionYaDiffeaDeVerdad cubre que, tras la corrida
// de adopción, el manifiesto queda establecido y la corrida siguiente ya
// sincroniza de verdad contra la plantilla (deja de ser puro modo
// protección).
func TestSegundaCorridaTrasAdopcionYaDiffeaDeVerdad(t *testing.T) {
	dest := t.TempDir()
	existing := []byte("feature list del usuario, con progreso real")
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), existing, 0644); err != nil {
		t.Fatalf("no se pudo preparar feature_list.json: %v", err)
	}

	tmplFS := fstest.MapFS{
		"feature_list.json": {Data: []byte("feature list nuevo del template")},
	}

	plan1, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (primera corrida) falló: %v", err)
	}
	if _, err := captureStdout(t, func() error { return applyPlan(plan1) }); err != nil {
		t.Fatalf("applyPlan (primera corrida) falló: %v", err)
	}

	plan2, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (segunda corrida) falló: %v", err)
	}
	if !plan2.manifestFound {
		t.Errorf("se esperaba manifestFound=true en la segunda corrida: ya existe el manifiesto de la adopción")
	}

	var fw *scaffoldFileWrite
	for i := range plan2.files {
		if plan2.files[i].relPath == "feature_list.json" {
			fw = &plan2.files[i]
		}
	}
	if fw == nil {
		t.Fatalf("el plan de la segunda corrida no incluye feature_list.json")
	}
	if fw.action != actionUpdate {
		t.Errorf("action = %v, se esperaba actionUpdate: el usuario no tocó el archivo desde la adopción", fw.action)
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan2) }); err != nil {
		t.Fatalf("applyPlan (segunda corrida) falló: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "feature_list.json"))
	if err != nil {
		t.Fatalf("no se pudo leer feature_list.json: %v", err)
	}
	if string(got) != "feature list nuevo del template" {
		t.Errorf("la segunda corrida debería sincronizar de verdad con la plantilla, se obtuvo %q", got)
	}
}

// TestManifiestoCorrupto_TratadoComoAdopcionConAviso cubre que un
// .claude/manifest.json con JSON inválido no aborta el comando: se trata
// igual que la ausencia de manifiesto (modo adopción) y se avisa.
func TestManifiestoCorrupto_TratadoComoAdopcionConAviso(t *testing.T) {
	dest := t.TempDir()
	manifestFile := manifestPath(dest)
	if err := os.MkdirAll(filepath.Dir(manifestFile), 0755); err != nil {
		t.Fatalf("no se pudo preparar .claude/: %v", err)
	}
	if err := os.WriteFile(manifestFile, []byte("{ esto no es json válido"), 0644); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto corrupto: %v", err)
	}

	existing := []byte("contenido del usuario")
	if err := os.WriteFile(filepath.Join(dest, "config.txt"), existing, 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}

	tmplFS := fstest.MapFS{"config.txt": {Data: []byte("plantilla nueva")}}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}
	if !plan.manifestCorrupt {
		t.Errorf("se esperaba manifestCorrupt=true para JSON inválido")
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if !strings.Contains(out, "corrupto") && !strings.Contains(out, "inválido") {
		t.Errorf("se esperaba un aviso de manifiesto corrupto en la salida, se obtuvo:\n%s", out)
	}

	got, err := os.ReadFile(filepath.Join(dest, "config.txt"))
	if err != nil {
		t.Fatalf("no se pudo leer config.txt: %v", err)
	}
	if string(got) != string(existing) {
		t.Errorf("manifiesto corrupto no debe sobreescribir contenido existente, se obtuvo %q", got)
	}
}

// TestFeatureListJsonProtegidoSinCasoEspecial es la regresión end-to-end del
// pedido original: feature_list.json editado por el usuario sobrevive a una
// segunda corrida sin que el código tenga ningún `if relPath ==
// "feature_list.json"` — la protección sale sola de la regla general.
func TestFeatureListJsonProtegidoSinCasoEspecial(t *testing.T) {
	dest := t.TempDir()
	tmplFS := fstest.MapFS{"feature_list.json": {Data: []byte(`{"features":[]}`)}}

	plan1, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (primera corrida) falló: %v", err)
	}
	if _, err := captureStdout(t, func() error { return applyPlan(plan1) }); err != nil {
		t.Fatalf("applyPlan (primera corrida) falló: %v", err)
	}

	edited := []byte(`{"features":[{"id":1,"name":"algo real"}]}`)
	if err := os.WriteFile(filepath.Join(dest, "feature_list.json"), edited, 0644); err != nil {
		t.Fatalf("no se pudo editar feature_list.json: %v", err)
	}

	plan2, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (segunda corrida) falló: %v", err)
	}
	if _, err := captureStdout(t, func() error { return applyPlan(plan2) }); err != nil {
		t.Fatalf("applyPlan (segunda corrida) falló: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "feature_list.json"))
	if err != nil {
		t.Fatalf("no se pudo leer feature_list.json: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("feature_list.json editado por el usuario no debería perderse en una segunda corrida, se obtuvo %q", got)
	}
}

// TestProgressHistoryMdProtegidoEnConflictoRealSinCasoEspecial cubre el mismo
// mecanismo genérico que TestFeatureListJsonProtegidoSinCasoEspecial pero con
// una ruta anidada (progress/history.md, distinta de un archivo plano en la
// raíz) y forzando el caso de conflicto real: el usuario edita el archivo
// semilla a mano (simulando trabajo real de progreso) Y la plantilla nueva
// también cambia ese mismo archivo en la segunda corrida. El resultado
// esperado es que la edición del usuario sobreviva intacta, salga el aviso de
// conflicto mencionando el archivo, y el manifiesto conserve el hash de la
// versión del usuario (no el de la plantilla nueva) — todo sin que
// scaffold.go tenga ningún `if relPath == "progress/history.md"`.
func TestProgressHistoryMdProtegidoEnConflictoRealSinCasoEspecial(t *testing.T) {
	dest := t.TempDir()
	tmplFS1 := fstest.MapFS{"progress/history.md": {Data: []byte("# Historial\n\n(vacío)\n")}}

	plan1, err := planScaffoldFromFS(dest, tmplFS1)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (primera corrida) falló: %v", err)
	}
	if _, err := captureStdout(t, func() error { return applyPlan(plan1) }); err != nil {
		t.Fatalf("applyPlan (primera corrida) falló: %v", err)
	}

	edited := []byte("# Historial\n\n## 2026-08-25\n\nSe implementó scaffold_manifest_sync.\n")
	historyPath := filepath.Join(dest, "progress", "history.md")
	if err := os.WriteFile(historyPath, edited, 0644); err != nil {
		t.Fatalf("no se pudo editar progress/history.md: %v", err)
	}

	// La plantilla también cambia en la segunda corrida: esto fuerza el caso
	// de conflicto real (usuario tocó Y plantilla cambió), a diferencia de
	// dejar la plantilla igual (que caería en el skip silencioso por
	// "usuario tocó, plantilla no cambió").
	tmplFS2 := fstest.MapFS{"progress/history.md": {Data: []byte("# Historial\n\n(plantilla actualizada)\n")}}

	plan2, err := planScaffoldFromFS(dest, tmplFS2)
	if err != nil {
		t.Fatalf("planScaffoldFromFS (segunda corrida) falló: %v", err)
	}
	stdout, err := captureStdout(t, func() error { return applyPlan(plan2) })
	if err != nil {
		t.Fatalf("applyPlan (segunda corrida) falló: %v", err)
	}

	got, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("no se pudo leer progress/history.md: %v", err)
	}
	if string(got) != string(edited) {
		t.Errorf("progress/history.md editado por el usuario no debería perderse en una segunda corrida con conflicto real, se obtuvo %q", got)
	}

	if !strings.Contains(stdout, "progress/history.md") {
		t.Errorf("se esperaba un aviso de conflicto mencionando progress/history.md, salida obtenida:\n%s", stdout)
	}

	result := loadManifest(dest)
	entry, ok := result.manifest.Files["progress/history.md"]
	if !ok {
		t.Fatalf("progress/history.md debería seguir registrado en el manifiesto tras el conflicto")
	}
	if entry.Hash != hashContent(edited) {
		t.Errorf("el manifiesto debería conservar el hash de la versión editada por el usuario, no el de la plantilla nueva")
	}
}

// TestGitignoreNuncaEntraAlManifiesto cubre que .gitignore queda fuera del
// manifiesto por completo, tanto en el plan como en lo que queda escrito en
// disco tras applyPlan: sigue con su lógica de merge propia, no con el diff
// genérico.
func TestGitignoreNuncaEntraAlManifiesto(t *testing.T) {
	dest := t.TempDir()
	tmplFS := fstest.MapFS{".gitignore": {Data: []byte("node_modules/\n")}}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}
	if _, ok := plan.manifest.Files[".gitignore"]; ok {
		t.Errorf(".gitignore no debería entrar al manifiesto del plan")
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan) }); err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}

	result := loadManifest(dest)
	if _, ok := result.manifest.Files[".gitignore"]; ok {
		t.Errorf(".gitignore no debería quedar registrado en .claude/manifest.json tras applyPlan")
	}
}

// TestManifestJsonEmbebidoNuncaSePropaga cubre la mitigación de dogfooding:
// si por error .claude/manifest.json de este propio repo quedara embebido en
// el fs.FS de plantilla (porque alguien corrió april init sobre este mismo
// repo), planScaffoldFromFS lo salta explícitamente y nunca lo escribe en el
// destino ni lo entra al manifiesto nuevo.
func TestManifestJsonEmbebidoNuncaSePropaga(t *testing.T) {
	dest := t.TempDir()
	tmplFS := fstest.MapFS{
		".claude/manifest.json": {Data: []byte(`{"schemaVersion":1,"files":{"algo":{"hash":"x"}}}`)},
		"AGENTS.md":             {Data: []byte("contenido AGENTS.md")},
	}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	for _, fw := range plan.files {
		if fw.relPath == ".claude/manifest.json" {
			t.Errorf(".claude/manifest.json embebido en la plantilla no debería propagarse al plan")
		}
	}
	if _, ok := plan.manifest.Files[".claude/manifest.json"]; ok {
		t.Errorf(".claude/manifest.json no debería entrar al manifiesto nuevo")
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan) }); err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}

	// applyPlan sí escribe SU PROPIO .claude/manifest.json al final (el real,
	// resultado de este plan) — lo que no debe pasar es que el contenido
	// embebido falso ("algo") se haya colado dentro de él.
	result := loadManifest(dest)
	if _, ok := result.manifest.Files["algo"]; ok {
		t.Errorf("el contenido de .claude/manifest.json embebido en la plantilla no debería colarse en el manifiesto real del destino")
	}
}

// TestVerifyLedgerEmbebidoNuncaSePropaga cubre la mitigación de dogfooding:
// si por error .claude/verify-ledger.jsonl de este propio repo quedara
// embebido en el fs.FS de plantilla (porque alguien corrió april init sobre
// este mismo repo), planScaffoldFromFS lo salta explícitamente y nunca lo
// escribe en el destino ni lo entra al manifiesto nuevo.
func TestVerifyLedgerEmbebidoNuncaSePropaga(t *testing.T) {
	dest := t.TempDir()
	tmplFS := fstest.MapFS{
		".claude/verify-ledger.jsonl": {Data: []byte(`{"featureId":"1","treeHash":"x"}` + "\n")},
		"AGENTS.md":                   {Data: []byte("contenido AGENTS.md")},
	}

	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	for _, fw := range plan.files {
		if fw.relPath == verifyLedgerPath {
			t.Errorf(".claude/verify-ledger.jsonl embebido en la plantilla no debería propagarse al plan")
		}
	}
	if _, ok := plan.manifest.Files[verifyLedgerPath]; ok {
		t.Errorf(".claude/verify-ledger.jsonl no debería entrar al manifiesto nuevo")
	}

	if _, err := captureStdout(t, func() error { return applyPlan(plan) }); err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dest, ".claude", "verify-ledger.jsonl")); err == nil {
		t.Errorf(".claude/verify-ledger.jsonl embebido en la plantilla no debería escribirse en el destino")
	} else if !os.IsNotExist(err) {
		t.Fatalf("error inesperado al comprobar .claude/verify-ledger.jsonl en el destino: %v", err)
	}
}

// TestInitShInvocaAprilStatusSinHeredocPython es un guardarraíl barato (no
// reemplaza la revisión humana del diff de init.sh, área sensible — ver
// docs/conventions.md) contra que alguien reintroduzca el heredoc
// `python3 - <<'PY' ... PY` de validación: lee el init.sh real del repo (no
// el embebido, el archivo en disco que ejecuta el humano/agente) y verifica
// por contenido que la delegación a `april status` sigue ahí (feature
// april_status_arbiter, ticket 03).
func TestInitShInvocaAprilStatusSinHeredocPython(t *testing.T) {
	content, err := os.ReadFile("init.sh")
	if err != nil {
		t.Fatalf("no se pudo leer init.sh: %v", err)
	}
	text := string(content)

	if strings.Contains(text, "<<'PY'") {
		t.Errorf("init.sh todavía contiene el heredoc python3 - <<'PY' de validación")
	}
	if !strings.Contains(text, "status") {
		t.Errorf("init.sh ya no invoca el comando 'status'")
	}
}

// TestBackupCandidatesPure cubre backupCandidates() de forma aislada, sin
// tocar disco: opera solo sobre el struct scaffoldPlan ya decidido. Los
// archivos con acción actionCreate/actionUpdate (applyPlan les hace
// os.WriteFile) y los de filesToDelete (os.Remove) deben aparecer; los que
// applyPlan no toca (actionSkipUnmodified, actionSkipConflict, actionAdopt)
// no deben aparecer.
func TestBackupCandidatesPure(t *testing.T) {
	plan := scaffoldPlan{
		files: []scaffoldFileWrite{
			{relPath: "create.txt", action: actionCreate},
			{relPath: "update.txt", action: actionUpdate},
			{relPath: "skip-unmodified.txt", action: actionSkipUnmodified},
			{relPath: "skip-conflict.txt", action: actionSkipConflict},
			{relPath: "adopt.txt", action: actionAdopt},
		},
		filesToDelete: []scaffoldFileDelete{
			{relPath: "borrar.txt"},
		},
	}

	got := backupCandidates(plan)

	want := map[string]bool{"create.txt": true, "update.txt": true, "borrar.txt": true}
	if len(got) != len(want) {
		t.Fatalf("backupCandidates = %v, se esperaban exactamente %v", got, want)
	}
	for _, rel := range got {
		if !want[rel] {
			t.Errorf("backupCandidates incluyó %q sin que applyPlan vaya a tocarlo", rel)
		}
	}
}

// TestBackupBeforeApplySinArchivosExistentesNoCreaBackup cubre el caso
// "scaffold inicial sobre directorio vacío": todos los archivos son
// actionCreate y ninguno existe todavía en disco, así que no hay nada que
// perder — backupBeforeApply no debe crear ningún directorio de backup.
func TestBackupBeforeApplySinArchivosExistentesNoCreaBackup(t *testing.T) {
	dest := t.TempDir()

	plan := scaffoldPlan{
		absTarget: dest,
		files: []scaffoldFileWrite{
			{relPath: "nuevo.txt", action: actionCreate},
		},
	}

	backupDir, err := backupBeforeApply(plan)
	if err != nil {
		t.Fatalf("backupBeforeApply falló: %v", err)
	}
	if backupDir != "" {
		t.Errorf("se esperaba backupDir vacío (nada que respaldar), se obtuvo %q", backupDir)
	}
	if _, err := os.Stat(filepath.Join(dest, ".claude", "backups")); !os.IsNotExist(err) {
		t.Errorf(".claude/backups no debería crearse cuando no hay archivos existentes que respaldar")
	}
}

// TestBackupBeforeApplyCopiaArchivosExistentes cubre que backupBeforeApply
// copia, con copia fiel de contenido, todo archivo que YA existe en disco y
// que el plan marca para sobreescribir (actionUpdate) o borrar
// (filesToDelete), antes de que applyPlan los toque.
func TestBackupBeforeApplyCopiaArchivosExistentes(t *testing.T) {
	dest := t.TempDir()

	if err := os.WriteFile(filepath.Join(dest, "config.txt"), []byte("contenido viejo"), 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "old.txt"), []byte("a borrar"), 0644); err != nil {
		t.Fatalf("no se pudo preparar old.txt: %v", err)
	}

	plan := scaffoldPlan{
		absTarget: dest,
		files: []scaffoldFileWrite{
			{relPath: "config.txt", action: actionUpdate},
		},
		filesToDelete: []scaffoldFileDelete{
			{relPath: "old.txt", destPath: filepath.Join(dest, "old.txt")},
		},
	}

	backupDir, err := backupBeforeApply(plan)
	if err != nil {
		t.Fatalf("backupBeforeApply falló: %v", err)
	}
	if backupDir == "" {
		t.Fatalf("se esperaba un backupDir no vacío")
	}
	if !strings.Contains(filepath.ToSlash(backupDir), ".claude/backups/") {
		t.Errorf("backupDir = %q, se esperaba que quedara bajo .claude/backups/", backupDir)
	}

	for _, rel := range []string{"config.txt", "old.txt"} {
		got, err := os.ReadFile(filepath.Join(backupDir, rel))
		if err != nil {
			t.Fatalf("no se encontró copia de %s en el backup: %v", rel, err)
		}
		want, err := os.ReadFile(filepath.Join(dest, rel))
		if err != nil {
			t.Fatalf("no se pudo leer el original %s: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Errorf("backup de %s = %q, se esperaba copia fiel %q", rel, got, want)
		}
	}
}

// TestBackupBeforeApplyCubreActionCreateConArchivoYaExistente cubre el caso
// en que planScaffoldFromFS marca un archivo como actionCreate (no estaba en
// el manifiesto previo, hadPrev=false) pero ese mismo relPath YA existe en
// disco con contenido distinto al del template (p.ej. el usuario lo creó a
// mano, o quedó de otro origen, sin que el manifiesto lo rastreara). La rama
// !hadPrev de planScaffoldFromFS no chequea diskExists antes de decidir
// actionCreate, así que backupBeforeApply debe respaldar igual el contenido
// que había en disco antes de que applyPlan lo sobreescriba.
func TestBackupBeforeApplyCubreActionCreateConArchivoYaExistente(t *testing.T) {
	dest := t.TempDir()

	original := []byte("contenido manual del usuario")
	if err := os.WriteFile(filepath.Join(dest, "nuevo.txt"), original, 0644); err != nil {
		t.Fatalf("no se pudo preparar nuevo.txt: %v", err)
	}
	// Manifiesto previo presente (no corrupto) pero sin rastro de nuevo.txt:
	// adopting=false, hadPrev=false para ese archivo.
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{"nuevo.txt": {Data: []byte("contenido del template")}}
	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	// Confirma la premisa del caso: el plan lo marca actionCreate pese a que
	// nuevo.txt ya existe en disco.
	if len(plan.files) != 1 || plan.files[0].action != actionCreate {
		t.Fatalf("se esperaba un único archivo con actionCreate, plan.files=%+v", plan.files)
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if !strings.Contains(out, ".claude") || !strings.Contains(out, "backup") {
		t.Errorf("se esperaba un aviso mencionando la ubicación del backup, se obtuvo:\n%s", out)
	}

	entries, err := os.ReadDir(filepath.Join(dest, ".claude", "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("se esperaba exactamente un directorio de backup, err=%v entries=%v", err, entries)
	}
	backupDir := filepath.Join(dest, ".claude", "backups", entries[0].Name())

	got, err := os.ReadFile(filepath.Join(backupDir, "nuevo.txt"))
	if err != nil {
		t.Fatalf("no se encontró copia de nuevo.txt en el backup: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("el backup de nuevo.txt = %q, se esperaba el contenido previo %q", got, original)
	}

	// applyPlan ya sobreescribió el destino con la plantilla nueva.
	gotDest, err := os.ReadFile(filepath.Join(dest, "nuevo.txt"))
	if err != nil {
		t.Fatalf("no se pudo leer nuevo.txt del destino: %v", err)
	}
	if string(gotDest) != "contenido del template" {
		t.Errorf("nuevo.txt del destino = %q, se esperaba el contenido del template tras aplicar", gotDest)
	}
}

// TestApplyPlanGeneraBackupAntesDeAplicar cubre el flujo completo a través
// de applyPlan: sobre un directorio ya scaffoldeado (config.txt gestionado
// por el manifiesto, actionUpdate), correr applyPlan debe dejar un backup
// localizable con el contenido ANTERIOR a la sobreescritura, y avisar por
// stdout dónde quedó.
func TestApplyPlanGeneraBackupAntesDeAplicar(t *testing.T) {
	dest := t.TempDir()
	original := []byte("v1")
	if err := os.WriteFile(filepath.Join(dest, "config.txt"), original, 0644); err != nil {
		t.Fatalf("no se pudo preparar config.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"config.txt": {Hash: hashContent(original)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	tmplFS := fstest.MapFS{"config.txt": {Data: []byte("v2")}}
	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	out, err := captureStdout(t, func() error { return applyPlan(plan) })
	if err != nil {
		t.Fatalf("applyPlan falló: %v", err)
	}
	if !strings.Contains(out, ".claude") || !strings.Contains(out, "backup") {
		t.Errorf("se esperaba un aviso mencionando la ubicación del backup, se obtuvo:\n%s", out)
	}

	entries, err := os.ReadDir(filepath.Join(dest, ".claude", "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("se esperaba exactamente un directorio de backup, err=%v entries=%v", err, entries)
	}
	backupDir := filepath.Join(dest, ".claude", "backups", entries[0].Name())

	got, err := os.ReadFile(filepath.Join(backupDir, "config.txt"))
	if err != nil {
		t.Fatalf("no se encontró config.txt en el backup: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("el backup de config.txt = %q, se esperaba el contenido previo %q", got, original)
	}

	// applyPlan ya sobreescribió el destino con la plantilla nueva.
	gotDest, err := os.ReadFile(filepath.Join(dest, "config.txt"))
	if err != nil {
		t.Fatalf("no se pudo leer config.txt del destino: %v", err)
	}
	if string(gotDest) != "v2" {
		t.Errorf("config.txt del destino = %q, se esperaba v2 tras aplicar el plan", gotDest)
	}
}

// TestBackupIntegroSiApplyPlanFallaAMitadDeCamino cubre la acceptance de que,
// si applyPlan falla a mitad de camino, el backup ya escrito antes de
// empezar queda íntegro y localizable. Se fuerza el fallo dejando uno de los
// archivos de destino sin permiso de escritura: applyPlan alcanza a
// respaldar y a escribir el primer archivo (orden alfabético, fileA antes
// que fileB) pero falla en el segundo — el backup de AMBOS debe seguir
// intacto pese al fallo a mitad de camino. El rollback (restaurar desde el
// backup) es manual: applyPlan no lo hace automáticamente.
func TestBackupIntegroSiApplyPlanFallaAMitadDeCamino(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("corriendo como root: los bits de permiso no bloquean la escritura")
	}

	dest := t.TempDir()
	contentA := []byte("A-v1")
	contentB := []byte("B-v1")
	pathA := filepath.Join(dest, "fileA.txt")
	pathB := filepath.Join(dest, "fileB.txt")
	if err := os.WriteFile(pathA, contentA, 0644); err != nil {
		t.Fatalf("no se pudo preparar fileA.txt: %v", err)
	}
	if err := os.WriteFile(pathB, contentB, 0644); err != nil {
		t.Fatalf("no se pudo preparar fileB.txt: %v", err)
	}
	if err := writeManifest(dest, manifest{Files: map[string]manifestEntry{
		"fileA.txt": {Hash: hashContent(contentA)},
		"fileB.txt": {Hash: hashContent(contentB)},
	}}); err != nil {
		t.Fatalf("no se pudo preparar el manifiesto previo: %v", err)
	}

	// fileB.txt queda sin permiso de escritura: applyPlan fallará al
	// intentar sobreescribirlo con la plantilla nueva.
	if err := os.Chmod(pathB, 0444); err != nil {
		t.Fatalf("no se pudo quitar permiso de escritura a fileB.txt: %v", err)
	}
	t.Cleanup(func() { os.Chmod(pathB, 0644) })

	tmplFS := fstest.MapFS{
		"fileA.txt": {Data: []byte("A-v2")},
		"fileB.txt": {Data: []byte("B-v2")},
	}
	plan, err := planScaffoldFromFS(dest, tmplFS)
	if err != nil {
		t.Fatalf("planScaffoldFromFS falló: %v", err)
	}

	_, applyErr := captureStdout(t, func() error { return applyPlan(plan) })
	if applyErr == nil {
		t.Fatalf("se esperaba que applyPlan fallara al escribir fileB.txt sin permiso de escritura")
	}

	entries, err := os.ReadDir(filepath.Join(dest, ".claude", "backups"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("se esperaba exactamente un directorio de backup pese al fallo, err=%v entries=%v", err, entries)
	}
	backupDir := filepath.Join(dest, ".claude", "backups", entries[0].Name())

	for rel, want := range map[string][]byte{"fileA.txt": contentA, "fileB.txt": contentB} {
		got, err := os.ReadFile(filepath.Join(backupDir, rel))
		if err != nil {
			t.Fatalf("el backup de %s no está íntegro tras el fallo a mitad de camino: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Errorf("backup de %s = %q, se esperaba el contenido previo %q (íntegro pese al fallo)", rel, got, want)
		}
	}
}
