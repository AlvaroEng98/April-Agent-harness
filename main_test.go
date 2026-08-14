package main

import (
	"os"
	"strings"
	"testing"
)

// TestPrintUsageMentionsApil cubre main.go:printUsage() — el CLI se invoca
// como `apil <command>`, no `harness <command>` (feature rename_cli_to_apil).
func TestPrintUsageMentionsApil(t *testing.T) {
	out, err := captureStdout(t, func() error {
		printUsage()
		return nil
	})
	if err != nil {
		t.Fatalf("printUsage no debería fallar: %v", err)
	}
	if !strings.Contains(out, "Usage: apil <command>") {
		t.Errorf("se esperaba 'Usage: apil <command>' en la salida, se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de printUsage, se obtuvo:\n%s", out)
	}
}

// TestVersionCommandPrintsApil cubre main.go:main() rama "version" — tras
// la objeción del reviewer en progress/review_rename_cli_to_apil.md,
// `apil version` debía seguir imprimiendo "harness v<versión>"; este test
// evita que la regresión vuelva a colarse sin fallar el build.
func TestVersionCommandPrintsApil(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"apil", "version"}
	defer func() { os.Args = origArgs }()

	out, err := captureStdout(t, func() error {
		main()
		return nil
	})
	if err != nil {
		t.Fatalf("main no debería fallar: %v", err)
	}
	if !strings.HasPrefix(out, "apil v") {
		t.Errorf("se esperaba que la salida empezara con 'apil v', se obtuvo:\n%s", out)
	}
	if strings.Contains(out, "harness") {
		t.Errorf("no se esperaba la palabra 'harness' en la salida de 'version', se obtuvo:\n%s", out)
	}
}
