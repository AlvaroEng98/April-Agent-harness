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
// (feature_list.json, progress/, docs/). El estado de trabajo del propio
// harness vive en la raíz (feature_list.json, progress/, docs/) y NO se embebe:
// así cada `harness init` genera un proyecto en limpio, no la bitácora del harness.
//
// release-notes.sh y .goreleaser.yaml NO se embeben: son del pipeline de
// release de este repo (Go + goreleaser), no del arnés que se scaffoldea.
// CHANGELOG.md tampoco: el lienzo limpio vive en templates/CHANGELOG.md.
//
//go:embed .claude AGENT.md CLAUDE.md init.sh sync-changelog.sh session-handoff.md CHECKPOINTS.md .gitignore templates
var templateFS embed.FS

const (
	colorCyan  = "\033[1;36m"
	colorDim   = "\033[2m"
	colorReset = "\033[0m"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "version", "--version", "-v":
		fmt.Printf("apil v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: apil <command>

Commands:
  init [directory]    Scaffold project structure
  version             Print version
  help                Show this help`)
}

func printBanner() {
	art := colorCyan +
		"    _    ____  ____  ___ _     \n" +
		"   / \\  |  _ \\|  _ \\|_ _| |   \n" +
		"  / _ \\ | |_) | |_) || || |   \n" +
		" / ___ \\|  __/|  _ < | || |___\n" +
		"/_/   \\_\\_|   |_| \\_\\___|_____|\n" +
		colorReset
	fmt.Println(art)
	fmt.Printf(colorDim+"  Project scaffolding for AI-assisted development  v%s"+colorReset+"\n\n", version)
}

func cmdInit() {
	printBanner()

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

	if err := scaffoldInit(absTarget); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// scaffoldInit contiene la lógica de scaffolding de `harness init` sobre un
// directorio destino ya resuelto a ruta absoluta. Se separa de cmdInit (que
// depende de os.Args y de os.Exit) para poder testearla in-process contra un
// directorio temporal.
func scaffoldInit(absTarget string) error {
	entries, err := os.ReadDir(absTarget)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(absTarget, 0755); err != nil {
			return err
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
			agentDir := filepath.Join(absTarget, ".claude", "agents")
			if err := os.RemoveAll(agentDir); err != nil {
				fmt.Fprintf(os.Stderr, "  Warning: could not clean %s: %v\n", agentDir, err)
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

		// todos los archivos y directorios (incluido el árbol .claude/) se
		// copian tal cual, sin transformación
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
		case "init.sh", "recap.sh", "sync-changelog.sh":
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
		return err
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
