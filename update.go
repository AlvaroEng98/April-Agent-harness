package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// installScriptURL apunta siempre a main: install.sh resuelve la última
// release por sí mismo (o la que indique VERSION), así que no hay que
// versionar esta URL.
const installScriptURL = "https://raw.githubusercontent.com/AlvaroEng98/April-Agent-harness/main/install.sh"

func cmdUpdate() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	binDir := filepath.Dir(exe)

	version := ""
	if len(os.Args) > 2 {
		version = os.Args[2]
	}

	fmt.Printf("Actualizando apil en %s...\n", binDir)

	name, args, env := buildUpdateCmd(binDir, version)
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// buildUpdateCmd decide qué comando correr para actualizar apil: siempre
// "sh -c" contra installScriptURL, con el entorno que arma updateEnv. No
// ejecuta nada — cmdUpdate es quien crea el *exec.Cmd y lo corre.
func buildUpdateCmd(binDir, version string) (name string, args []string, env []string) {
	return "sh", []string{"-c", "curl -fsSL " + installScriptURL + " | sh"}, updateEnv(os.Environ(), binDir, version)
}

// updateEnv arma el entorno del subproceso que corre install.sh: BIN_DIR
// siempre apunta al directorio del binario en ejecución (no ~/.local/bin a
// secas, por si el usuario instaló en otra ruta con BIN_DIR original). VERSION
// solo se agrega si el usuario la pidió explícita; si no, install.sh resuelve
// la última release por su cuenta.
func updateEnv(baseEnv []string, binDir, version string) []string {
	env := append(append([]string{}, baseEnv...), "BIN_DIR="+binDir)
	if version != "" {
		env = append(env, "VERSION="+version)
	}
	return env
}
