package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Se embeben el tooling (idéntico al que dogfoodea este repo) y el directorio
// templates/, que contiene el LIENZO LIMPIO de los archivos con estado
// (feature_list.json, progress/, docs/). El estado de trabajo del propio
// harness vive en la raíz (feature_list.json, progress/, docs/) y NO se embebe:
// así cada `harness init` genera un proyecto en limpio, no la bitácora del harness.
//
//go:embed .claude .opencode AGENT.md CLAUDE.md init.sh recap.sh session-handoff.md CHECKPOINTS.md .gitignore opencode.json templates
var templateFS embed.FS



func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "version", "--version", "-v":
		fmt.Printf("harness v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: harness <command>

Commands:
  init [directory]    Scaffold project structure
  version             Print version
  help                Show this help`)
}

func cmdInit() {
	printBanner()

	detected := DetectTools(defaultTools, exec.LookPath)
	chosen := selectClient(detected)
	if len(chosen) == 0 {
		os.Exit(1)
	}

	// construir set de herramientas elegidas por directorio
	want := map[string]bool{}
	for _, t := range chosen {
		want[t.Dir] = true
	}

	target := "."
	args := os.Args[2:]
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			target = a
			break
		}
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	entries, err := os.ReadDir(absTarget)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(absTarget, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		isExistingHarness := false
		for _, e := range entries {
			if e.Name() == "AGENT.md" || e.Name() == "feature_list.json" {
				isExistingHarness = true
				break
			}
		}
		if isExistingHarness {
			fmt.Println("  Existing harness project detected, overwriting agent directories...")
			for _, t := range chosen {
				agentDir := filepath.Join(absTarget, t.Dir, "agents")
				if err := os.RemoveAll(agentDir); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not clean %s: %v\n", agentDir, err)
				}
			}
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

		// archivos de agentes — copiar a cada herramienta seleccionada
		if strings.HasPrefix(path, ".claude/agents/") && !d.IsDir() {
			data, err := templateFS.ReadFile(path)
			if err != nil {
				return err
			}
			return copyAgentToTools(data, d.Name(), absTarget, want)
		}

		// opencode.json es config OpenCode-específica: solo se copia si esa
		// herramienta fue seleccionada.
		if path == "opencode.json" && !want[".opencode"] {
			return nil
		}

		// gating genérico por herramienta: si un archivo/directorio pertenece
		// al árbol de una herramienta no seleccionada, se omite (los dirs se
		// saltan completos con fs.SkipDir para no recorrer su subárbol).
		for _, toolDir := range []string{".claude", ".opencode"} {
			if path == toolDir || strings.HasPrefix(path, toolDir+"/") {
				if !want[toolDir] {
					if d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
				break
			}
		}

		// resto de archivos y directorios
		destPath := filepath.Join(absTarget, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0644)
		switch d.Name() {
		case "init.sh", "recap.sh", "session_start_recap.sh":
			mode = 0755
		}
		if relPath == ".gitignore" {
			if _, err := os.Stat(destPath); err == nil {
				merged, err := mergeGitignore(destPath, data)
				if err != nil {
					return err
				}
				if err := os.WriteFile(destPath, merged, mode); err != nil {
					return err
				}
				fmt.Printf("  Updated %s\n", filepath.ToSlash(relPath))
				return nil
			}
		}
		if err := os.WriteFile(destPath, data, mode); err != nil {
			return err
		}
		fmt.Printf("  Created %s\n", filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	emptyDirs := []string{"src", "tests", "specs"}
	for _, dir := range emptyDirs {
		dirPath := filepath.Join(absTarget, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not create %s/\n", filepath.ToSlash(dirPath))
		} else {
			fmt.Printf("  Created %s/\n", filepath.ToSlash(dirPath))
		}
	}

	fmt.Println()
	fmt.Printf("Done! Harness project scaffolded in %s\n", absTarget)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit feature_list.json with your project info")
	fmt.Println("  2. Run ./init.sh to verify the environment")
	fmt.Println("  3. Read AGENT.md to understand the workflow")
}

// copyAgentToTools copia un archivo de agente a todas las herramientas seleccionadas.
func copyAgentToTools(data []byte, filename, absTarget string, want map[string]bool) error {
	for toolDir := range want {
		gen, ok := generators[toolDir]
		if !ok {
			continue
		}

		sub := gen.GetSubdir()
		dir := filepath.Join(absTarget, toolDir, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		transformed := gen.Transform(data)
		dest := filepath.Join(dir, filename)
		if err := os.WriteFile(dest, transformed, 0644); err != nil {
			return err
		}
		fmt.Printf("  Created %s/%s/%s\n", toolDir, sub, filename)
	}
	return nil
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
