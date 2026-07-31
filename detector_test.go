package main

import (
	"fmt"
	"testing"
)

func TestDetectTools_AllFound(t *testing.T) {
	tools := []Tool{
		{"Claude", "claude", ".claude"},
		{"OpenCode", "opencode", ".opencode"},
	}
	mockLookPath := func(bin string) (string, error) {
		return "/usr/bin/" + bin, nil
	}

	result := DetectTools(tools, mockLookPath)

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}
	for _, r := range result {
		if !r.Found {
			t.Errorf("expected %s to be found", r.Name)
		}
	}
}

func TestDetectTools_NoneFound(t *testing.T) {
	tools := []Tool{
		{"Claude", "claude", ".claude"},
	}
	mockLookPath := func(bin string) (string, error) {
		return "", fmt.Errorf("not found: %s", bin)
	}

	result := DetectTools(tools, mockLookPath)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Found {
		t.Error("expected claude to not be found")
	}
}

func TestDetectTools_Mixed(t *testing.T) {
	tools := []Tool{
		{"Claude", "claude", ".claude"},
		{"OpenCode", "opencode", ".opencode"},
	}
	mockLookPath := func(bin string) (string, error) {
		if bin == "claude" {
			return "/usr/bin/claude", nil
		}
		return "", fmt.Errorf("not found")
	}

	result := DetectTools(tools, mockLookPath)

	if !result[0].Found {
		t.Error("expected claude to be found")
	}
	if result[1].Found {
		t.Error("expected opencode to not be found")
	}
}

func TestDetectTools_EmptyList(t *testing.T) {
	mockLookPath := func(bin string) (string, error) {
		return "", fmt.Errorf("not found")
	}

	result := DetectTools(nil, mockLookPath)

	if len(result) != 0 {
		t.Fatalf("expected 0 results, got %d", len(result))
	}
}

func TestAvailableTools(t *testing.T) {
	detected := []DetectedTool{
		{Tool: Tool{Name: "A", Binary: "a", Dir: ".a"}, Found: true},
		{Tool: Tool{Name: "B", Binary: "b", Dir: ".b"}, Found: false},
		{Tool: Tool{Name: "C", Binary: "c", Dir: ".c"}, Found: true},
	}

	avail := availableTools(detected)

	if len(avail) != 2 {
		t.Fatalf("expected 2 available, got %d", len(avail))
	}
	if avail[0].Name != "A" || avail[1].Name != "C" {
		t.Errorf("expected [A, C], got [%s, %s]", avail[0].Name, avail[1].Name)
	}
}

func TestDetectTools_PreservesToolMetadata(t *testing.T) {
	tools := []Tool{
		{"Claude Code", "claude", ".claude"},
	}
	mockLookPath := func(bin string) (string, error) {
		return "/usr/bin/claude", nil
	}

	result := DetectTools(tools, mockLookPath)

	if result[0].Name != "Claude Code" {
		t.Errorf("expected name 'Claude Code', got '%s'", result[0].Name)
	}
	if result[0].Binary != "claude" {
		t.Errorf("expected binary 'claude', got '%s'", result[0].Binary)
	}
	if result[0].Dir != ".claude" {
		t.Errorf("expected dir '.claude', got '%s'", result[0].Dir)
	}
}
