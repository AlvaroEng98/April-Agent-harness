package main

import "testing"

func TestUpdateEnv(t *testing.T) {
	base := []string{"HOME=/home/x", "PATH=/usr/bin"}

	t.Run("sin version explicita", func(t *testing.T) {
		env := updateEnv(base, "/home/x/.local/bin", "")
		assertContains(t, env, "BIN_DIR=/home/x/.local/bin")
		assertNotContainsPrefix(t, env, "VERSION=")
	})

	t.Run("con version explicita", func(t *testing.T) {
		env := updateEnv(base, "/opt/april", "0.3.4")
		assertContains(t, env, "BIN_DIR=/opt/april")
		assertContains(t, env, "VERSION=0.3.4")
	})

	t.Run("no muta el entorno base", func(t *testing.T) {
		before := append([]string{}, base...)
		updateEnv(base, "/tmp", "1.0.0")
		if len(base) != len(before) {
			t.Fatalf("updateEnv mutó el slice base: %v", base)
		}
	})
}

func assertContains(t *testing.T, env []string, want string) {
	t.Helper()
	for _, e := range env {
		if e == want {
			return
		}
	}
	t.Fatalf("esperaba %q en %v", want, env)
}

func assertNotContainsPrefix(t *testing.T, env []string, prefix string) {
	t.Helper()
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			t.Fatalf("no esperaba entrada con prefijo %q, encontrado %q", prefix, e)
		}
	}
}

// TestBuildUpdateCmd cubre update.go:buildUpdateCmd() — la especificación
// completa del comando (name, args, env) que antes solo vivía inline en
// cmdUpdate y nunca pasaba por un test.
func TestBuildUpdateCmd(t *testing.T) {
	t.Run("sin version explicita", func(t *testing.T) {
		name, args, env := buildUpdateCmd("/home/x/.local/bin", "")

		if name != "sh" {
			t.Errorf("se esperaba name=\"sh\", se obtuvo %q", name)
		}
		wantArgs := []string{"-c", "curl -fsSL " + installScriptURL + " | sh"}
		if len(args) != len(wantArgs) || args[0] != wantArgs[0] || args[1] != wantArgs[1] {
			t.Errorf("se esperaba args=%v, se obtuvo %v", wantArgs, args)
		}
		assertContains(t, env, "BIN_DIR=/home/x/.local/bin")
		assertNotContainsPrefix(t, env, "VERSION=")
	})

	t.Run("con version explicita", func(t *testing.T) {
		_, _, env := buildUpdateCmd("/opt/april", "0.3.4")

		assertContains(t, env, "BIN_DIR=/opt/april")
		assertContains(t, env, "VERSION=0.3.4")
	})
}
