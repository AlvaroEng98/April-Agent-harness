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

//go:embed .claude docs AGENT.md CLAUDE.md feature_list.json init.sh session-handoff.md progress CHECKPOINTS.md .gitignore
var templateFS embed.FS

var version = "0.1.0"

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

	detected := detectTools()
	chosen := selectClient(detected)
	if len(chosen) == 0 {
		os.Exit(1)
	}

	// construir set de herramientas elegidas por directorio
	want := map[string]bool{}
	for _, t := range chosen {
		want[t.dir] = true
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
				agentDir := filepath.Join(absTarget, t.dir, "agents")
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

		// archivos de agentes — copiar a cada herramienta seleccionada
		if strings.HasPrefix(path, ".claude/agents/") && !d.IsDir() {
			data, err := templateFS.ReadFile(path)
			if err != nil {
				return err
			}
			return copyAgentToTools(data, d.Name(), absTarget, want)
		}

		// directorio .claude/agents — solo crear si claude está seleccionado
		if path == ".claude" || path == ".claude/agents" {
			if want[".claude"] {
				return os.MkdirAll(filepath.Join(absTarget, path), 0755)
			}
			return nil
		}

		// resto de archivos y directorios
		destPath := filepath.Join(absTarget, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		data, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0644)
		if d.Name() == "init.sh" {
			mode = 0755
		}
		if err := os.WriteFile(destPath, data, mode); err != nil {
			return err
		}
		fmt.Printf("  Created %s\n", filepath.ToSlash(path))
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
