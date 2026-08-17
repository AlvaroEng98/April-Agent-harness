package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Se embeben el tooling (idéntico al que dogfoodea este repo) y el directorio
// templates/, que contiene el LIENZO LIMPIO de los archivos con estado
// (feature_list.json, progress/, docs/, .gitignore). El estado de trabajo del
// propio harness vive en la raíz (feature_list.json, progress/, docs/,
// .gitignore) y NO se embebe: así cada `harness init` genera un proyecto en
// limpio, no la bitácora ni las reglas de ignore del harness. El .gitignore
// de la raíz de este repo tiene reglas propias del desarrollo del harness
// (OS, IDE, build de Go, notas de sesión) que no aplican al proyecto
// scaffoldeado; por eso el template vive aparte, en templates/.gitignore, con
// solo las reglas que sí aplican al destino.
//
// release-notes.sh, sync-changelog.sh y .goreleaser.yaml NO se embeben: son
// del pipeline de release de este repo (Go + goreleaser), no del arnés que se
// scaffoldea. CHANGELOG.md tampoco tiene plantilla: el proyecto scaffoldeado
// no arranca con changelog propio.
//
//go:embed .claude AGENT.md CLAUDE.md init.sh session-handoff.md CHECKPOINTS.md all:templates
var templateFS embed.FS

// scaffoldFileWrite describe un único archivo a escribir en el destino: su
// ruta relativa (para el mensaje de progreso), su ruta absoluta de destino,
// el contenido ya resuelto (incluido el merge de .gitignore si aplica), el
// modo con el que se escribe y si el mensaje de progreso es "Created" o
// "Updated" (caso .gitignore fusionado).
type scaffoldFileWrite struct {
	relPath  string
	destPath string
	content  []byte
	mode     fs.FileMode
	isUpdate bool
}

// scaffoldPlan es el resultado, ya decidido, de qué hacer al scaffoldear
// absTarget: qué limpiar, qué escribir y con qué modo, y qué directorios
// vacíos crear. No contiene ninguna llamada de I/O de escritura en sí mismo:
// lo produce planScaffold (puro) y lo consume applyPlan (ejecutor).
type scaffoldPlan struct {
	absTarget         string
	createTargetDir   bool
	isExistingHarness bool
	agentDirToClean   string
	dirsToCreate      []string
	files             []scaffoldFileWrite
	emptyDirs         []string
}

// planScaffold decide todo lo que hay que hacer para scaffoldear absTarget
// sin tocar el disco (salvo lecturas): detecta si absTarget existe y si es
// una instalación de harness existente, recorre el embed.FS para decidir qué
// archivos y directorios corresponden, resuelve el merge de .gitignore si el
// destino ya lo tiene, y arma la lista de directorios vacíos a crear. No
// llama os.WriteFile, os.RemoveAll ni os.MkdirAll.
func planScaffold(absTarget string) (scaffoldPlan, error) {
	plan := scaffoldPlan{absTarget: absTarget}

	entries, err := os.ReadDir(absTarget)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return scaffoldPlan{}, err
		}
		plan.createTargetDir = true
	} else {
		for _, e := range entries {
			if e.Name() == "AGENT.md" || e.Name() == "feature_list.json" {
				plan.isExistingHarness = true
				break
			}
		}
		if plan.isExistingHarness {
			plan.agentDirToClean = filepath.Join(absTarget, ".claude", "agents")
		}
	}

	err = fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		// templates/ contiene el lienzo limpio de los archivos con estado.
		// Se copia a la raíz del destino stripping el prefijo "templates/".
		// El directorio "templates" en sí no se crea en el destino.
		relPath := path
		if path == "templates" {
			return nil
		}
		if strings.HasPrefix(path, "templates/") {
			relPath = strings.TrimPrefix(path, "templates/")
		}

		// todos los archivos y directorios (incluido el árbol .claude/) se
		// copian tal cual, sin transformación
		destPath := filepath.Join(absTarget, relPath)
		if d.IsDir() {
			plan.dirsToCreate = append(plan.dirsToCreate, destPath)
			return nil
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0644)
		switch d.Name() {
		case "init.sh", "recap.sh":
			mode = 0755
		}
		fw := scaffoldFileWrite{
			relPath:  filepath.ToSlash(relPath),
			destPath: destPath,
			mode:     mode,
		}
		if relPath == ".gitignore" {
			if _, err := os.Stat(destPath); err == nil {
				merged, err := mergeGitignore(destPath, data)
				if err != nil {
					return err
				}
				fw.content = merged
				fw.isUpdate = true
				plan.files = append(plan.files, fw)
				return nil
			}
		}
		fw.content = data
		plan.files = append(plan.files, fw)
		return nil
	})
	if err != nil {
		return scaffoldPlan{}, err
	}

	plan.emptyDirs = []string{
		filepath.Join(absTarget, "specs"),
	}

	return plan, nil
}

// applyPlan ejecuta un scaffoldPlan ya decidido: solo hace las llamadas
// os.RemoveAll/os.WriteFile/os.MkdirAll y los mensajes de progreso por
// consola. No toma ninguna decisión de contenido.
func applyPlan(plan scaffoldPlan) error {
	if plan.createTargetDir {
		if err := os.MkdirAll(plan.absTarget, 0755); err != nil {
			return err
		}
	} else if plan.isExistingHarness {
		fmt.Println("  Existing harness project detected, overwriting agent directories...")
		if err := os.RemoveAll(plan.agentDirToClean); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not clean %s: %v\n", plan.agentDirToClean, err)
		}
	}

	for _, dir := range plan.dirsToCreate {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	created, updated := 0, 0
	for _, fw := range plan.files {
		if err := os.WriteFile(fw.destPath, fw.content, fw.mode); err != nil {
			return err
		}
		if fw.isUpdate {
			updated++
		} else {
			created++
		}
	}

	for _, dirPath := range plan.emptyDirs {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create %s/\n", filepath.ToSlash(dirPath))
		} else {
			created++
		}
	}

	switch {
	case updated > 0 && created > 0:
		fmt.Printf("  %d files created, %d updated\n", created, updated)
	case updated > 0:
		fmt.Printf("  %d files updated\n", updated)
	default:
		fmt.Printf("  %d files created\n", created)
	}

	fmt.Println()
	fmt.Printf("Done! Harness project scaffolded in %s\n", plan.absTarget)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit feature_list.json with your project info")
	fmt.Println("  2. Run ./init.sh to verify the environment")
	fmt.Println("  3. Read AGENT.md to understand the workflow")
	return nil
}

// scaffoldInit contiene la lógica de scaffolding de `harness init` sobre un
// directorio destino ya resuelto a ruta absoluta. Se separa de cmdInit (que
// depende de os.Args y de os.Exit) para poder testearla in-process contra un
// directorio temporal. Es pura composición: decide con planScaffold y
// ejecuta con applyPlan.
func scaffoldInit(absTarget string) error {
	plan, err := planScaffold(absTarget)
	if err != nil {
		return err
	}
	return applyPlan(plan)
}

// mergeGitignore agrega al archivo existente las entradas del template que falten.
func mergeGitignore(existingPath string, templateData []byte) ([]byte, error) {
	existing, err := os.ReadFile(existingPath)
	if err != nil {
		return nil, err
	}

	existingLines := strings.Split(string(existing), "\n")
	existingSet := make(map[string]bool, len(existingLines))
	for _, l := range existingLines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			existingSet[trimmed] = true
		}
	}

	templateLines := strings.Split(string(templateData), "\n")
	var missing []string
	for _, l := range templateLines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !existingSet[trimmed] {
			missing = append(missing, l)
		}
	}

	if len(missing) == 0 {
		return existing, nil
	}

	result := string(existing)
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result += "\n"
	}
	result += "\n"
	for _, l := range missing {
		result += l + "\n"
	}

	return []byte(result), nil
}
