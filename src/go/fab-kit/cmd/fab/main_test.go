package main

import (
	"testing"
)

func TestFabKitArgs(t *testing.T) {
	// Verify the allowlist contains exactly the fab-kit commands
	expected := []string{"init", "upgrade", "sync", "update"}
	for _, cmd := range expected {
		if !fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to contain %q", cmd)
		}
	}

	// Verify workflow commands are not in the allowlist (they go to fab-go)
	workflow := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range workflow {
		if fabKitArgs[cmd] {
			t.Errorf("expected fabKitArgs to NOT contain %q (belongs to fab-go)", cmd)
		}
	}
}

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}
