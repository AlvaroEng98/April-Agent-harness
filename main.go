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
	case "update":
		cmdUpdate()
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
  update [version]    Update apil to the latest (or given) release
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
