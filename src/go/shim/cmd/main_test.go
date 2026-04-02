package main

import (
	"testing"
)

func TestShimCommands(t *testing.T) {
	// Verify shimCommands map contains the expected entries
	expected := []string{"init", "upgrade"}
	for _, cmd := range expected {
		if !shimCommands[cmd] {
			t.Errorf("expected shimCommands to contain %q", cmd)
		}
	}

	// Verify non-shim commands are not in the map
	nonShim := []string{"status", "preflight", "resolve", "log", "change", "score"}
	for _, cmd := range nonShim {
		if shimCommands[cmd] {
			t.Errorf("expected shimCommands to NOT contain %q (should dispatch to fab-go)", cmd)
		}
	}
}

func TestVersion(t *testing.T) {
	// The version variable should be set (defaults to "dev")
	if version == "" {
		t.Error("version should not be empty")
	}
}
