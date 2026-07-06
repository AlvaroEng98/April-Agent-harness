package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	colorCyan  = "\033[1;36m"
	colorDim   = "\033[2m"
	colorReset = "\033[0m"
)

var knownTools = []struct {
	name   string
	binary string
	dir    string // carpeta a crear en el proyecto
}{
	{"Claude Code", "claude", ".claude"},
	{"OpenCode", "opencode", ".opencode"},
}

type detectedTool struct {
	name   string
	binary string
	dir    string
	found  bool
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

func detectTools() []detectedTool {
	result := make([]detectedTool, len(knownTools))
	for i, t := range knownTools {
		_, err := exec.LookPath(t.binary)
		result[i] = detectedTool{t.name, t.binary, t.dir, err == nil}
	}
	return result
}

// selectClient muestra las herramientas detectadas y retorna los dirs elegidos.
// Retorna nil si el usuario cancela o no hay herramientas disponibles.
func selectClient(detected []detectedTool) []detectedTool {
	fmt.Println("  Herramientas AI detectadas:\n")
	for _, d := range detected {
		if d.found {
			fmt.Printf("    \033[32m✓\033[0m  %s\n", d.name)
		} else {
			fmt.Printf("    \033[90m✗  %s\033[0m\n", d.name)
		}
	}
	fmt.Println()

	var available []detectedTool
	for _, d := range detected {
		if d.found {
			available = append(available, d)
		}
	}

	if len(available) == 0 {
		fmt.Fprintln(os.Stderr, "  \033[31mNo se detectó ninguna herramienta compatible.\033[0m")
		fmt.Fprintln(os.Stderr, "  Instala claude, cursor, opencode o pi y vuelve a intentar.")
		return nil
	}

	if len(available) == 1 {
		fmt.Printf("  \033[1;36m→ Estructura para: %s (auto-detectado)\033[0m\n\n", available[0].name)
		return available
	}

	fmt.Println("  ¿Para qué herramientas crear la estructura de carpetas?\n")
	return selectInteractive(available)
}

type selectionState struct {
	tool     detectedTool
	selected bool
}

func selectInteractive(available []detectedTool) []detectedTool {
	states := make([]selectionState, len(available))
	for i, t := range available {
		states[i] = selectionState{tool: t}
	}
	cursor := 0

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return selectNumbered(available)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	printList := func() {
		for i, s := range states {
			checkbox := "[ ]"
			if s.selected {
				checkbox = "[x]"
			}
			if i == cursor {
				fmt.Printf("  \033[1;36m> %s %s\033[0m\r\n", checkbox, s.tool.name)
			} else {
				fmt.Printf("    %s %s\r\n", checkbox, s.tool.name)
			}
		}
		fmt.Print("\r\n")
		fmt.Print("  \033[2m↑↓ navegar  Espacio marcar  Enter confirmar  Esc cancelar\033[0m")
	}

	moveUp := func() {
		for i := 0; i < len(states)+1; i++ {
			fmt.Print("\033[2K\033[1A")
		}
		fmt.Print("\033[2K\r")
	}

	printList()

	buf := make([]byte, 3)
	for {
		n, _ := os.Stdin.Read(buf)
		if n == 0 {
			continue
		}

		switch {
		case buf[0] == 13: // Enter — confirmar
			term.Restore(int(os.Stdin.Fd()), oldState)
			var result []detectedTool
			for _, s := range states {
				if s.selected {
					result = append(result, s.tool)
				}
			}
			if len(result) == 0 {
				fmt.Print("\r\n\r\n  No se seleccionó ninguna herramienta.\r\n\r\n")
				return nil
			}
			fmt.Print("\r\n\r\n")
			return result

		case buf[0] == 32: // Espacio — toggle
			states[cursor].selected = !states[cursor].selected
			moveUp()
			printList()

		case buf[0] == 3: // Ctrl+C
			term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print("\r\n")
			os.Exit(0)

		case buf[0] == 27 && n == 1: // Esc
			term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print("\r\n\r\n  Cancelado.\r\n")
			return nil

		case n >= 3 && buf[0] == 27 && buf[1] == '[':
			switch buf[2] {
			case 'A': // flecha arriba
				if cursor > 0 {
					cursor--
					moveUp()
					printList()
				}
			case 'B': // flecha abajo
				if cursor < len(states)-1 {
					cursor++
					moveUp()
					printList()
				}
			}
		}
	}
}

// selectNumbered es el fallback si el terminal no soporta raw mode.
func selectNumbered(available []detectedTool) []detectedTool {
	for i, d := range available {
		fmt.Printf("  [%d] %s\n", i+1, d.name)
	}
	fmt.Printf("\n  Selecciona (ej: 1,3) (Enter para cancelar): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("\n  Cancelado.")
		return nil
	}

	var result []detectedTool
	seen := map[int]bool{}
	for _, part := range strings.Split(input, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > len(available) || seen[n] {
			continue
		}
		seen[n] = true
		result = append(result, available[n-1])
	}
	if len(result) == 0 {
		fmt.Fprintln(os.Stderr, "\n  \033[31mSelección inválida.\033[0m")
	}
	return result
}
