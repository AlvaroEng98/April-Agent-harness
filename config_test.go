package main

import (
	"testing"
)

func TestVersionNotEmpty(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}