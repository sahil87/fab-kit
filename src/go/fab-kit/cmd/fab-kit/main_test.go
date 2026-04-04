package main

import (
	"testing"
)

func TestFabKitCommands(t *testing.T) {
	// Verify fabKitCommands map contains the expected entries
	expected := []string{"init", "upgrade-repo", "sync"}
	for _, cmd := range expected {
		if !fabKitCommands[cmd] {
			t.Errorf("expected fabKitCommands to contain %q", cmd)
		}
	}

	// Verify workflow commands are not in the map (they belong to fab-go)
	workflow := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range workflow {
		if fabKitCommands[cmd] {
			t.Errorf("expected fabKitCommands to NOT contain %q (belongs to fab-go)", cmd)
		}
	}
}

func TestVersion(t *testing.T) {
	// The version variable should be set (defaults to "dev")
	if version == "" {
		t.Error("version should not be empty")
	}
}
