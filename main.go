package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
	case "status":
		cmdStatus()
	case "doctor":
		cmdDoctor()
	case "feature":
		cmdFeature()
	case "verify":
		cmdVerify()
	case "review":
		cmdReview()
	case "update":
		cmdUpdate()
	case "version", "--version", "-v":
		fmt.Printf("april v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: april <command>

Commands:
  init [directory]                                   Scaffold project structure
  status [id] [--json]                                Show computed phase/nextRecommended/blockedReasons
  doctor [--json]                                     Chequeo read-only de salud del entorno (manifiesto, agentes, status, deuda vs baseline)
  doctor --freeze-baseline                            Congela el baseline de deuda actual (única escritura de doctor; falla si ya existe uno)
  feature set-status <id> <estado> [--verdict <valor>] Única vía válida de escritura de feature_list.json
  verify record --feature <id> -- <comando>           Corre <comando>, anexa evidencia a .claude/verify-ledger.jsonl
  review start --feature <id> [--json]                Ejecuta git write-tree, imprime subject_hash; con --json, agrega touchedPaths/sensitiveAreasTouched/extraReviewRequired
  review record --feature <id> --verdict <valor> [--subject-hash <hash>]  Registra el veredicto de reviewer_agent en .claude/verify-ledger.jsonl; con --subject-hash, rechaza si el árbol cambió
  update [version]                                    Update april to the latest (or given) release
  version                                             Print version
  help                                                Show this help`)
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
