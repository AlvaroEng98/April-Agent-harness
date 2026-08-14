package main

import (
	"os"
	"strings"
	"testing"
)

// TestPrintUsageMentionsApril cubre main.go:printUsage() — el CLI se invoca
// como `april <command>`, no `harness <command>` (feature rename_cli_to_april,
// ver feature_list.json).
func TestPrintUsageMentionsApril(t *testing.T) {
	out, err := captureStdout(t, func() error {
		printUsage()
		return nil
	})
	if err != nil {
		t.Fatalf("printUsage no debería fallar: %v", err)
	}
	if !strings.Contains(out, "Usage: april <command>") {
		t.Errorf("se esperaba 'Usage: april <command>' en la salida, se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de printUsage, se obtuvo:\n%s", out)
	}
}

// TestVersionCommandPrintsApril cubre main.go:main() rama "version" — tras
// una objeción de un review previo (ver historial de la feature 4 en
// feature_list.json), el comando `version` debía seguir imprimiendo
// "<binario> v<versión>"; este test evita que la regresión vuelva a
// colarse sin fallar el build.
func TestVersionCommandPrintsApril(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"april", "version"}
	defer func() { os.Args = origArgs }()

	out, err := captureStdout(t, func() error {
		main()
		return nil
	})
	if err != nil {
		t.Fatalf("main no debería fallar: %v", err)
	}
	if !strings.HasPrefix(out, "april v") {
		t.Errorf("se esperaba que la salida empezara con 'april v', se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de 'version', se obtuvo:\n%s", out)
	}
}
