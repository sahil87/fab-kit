package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStatusSetChecklistRemovedCmd_ReturnsPointerError(t *testing.T) {
	cmd := statusSetChecklistRemovedCmd()
	cmd.SetArgs([]string{"some-change", "completed", "5"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected non-nil error from set-checklist removed stub")
	}

	msg := err.Error()
	if !strings.Contains(msg, "\"set-checklist\" is now \"set-acceptance\"") {
		t.Errorf("error should contain pointer to set-acceptance, got: %v", err)
	}
	if !strings.Contains(msg, "fab status set-acceptance") {
		t.Errorf("error should suggest fab status set-acceptance, got: %v", err)
	}
}

func TestStatusSetAcceptanceCmd_RegisteredWithExpectedUse(t *testing.T) {
	cmd := statusSetAcceptanceCmd()
	if !strings.HasPrefix(cmd.Use, "set-acceptance ") {
		t.Errorf("statusSetAcceptanceCmd Use = %q, want prefix \"set-acceptance \"", cmd.Use)
	}
}

func TestStatusCmd_RegistersBothChecklistRemovedAndSetAcceptance(t *testing.T) {
	root := statusCmd()
	hasSetAcceptance := false
	hasSetChecklistRemoved := false
	for _, sub := range root.Commands() {
		switch {
		case strings.HasPrefix(sub.Use, "set-acceptance"):
			hasSetAcceptance = true
		case strings.HasPrefix(sub.Use, "set-checklist"):
			hasSetChecklistRemoved = true
		}
	}
	if !hasSetAcceptance {
		t.Error("statusCmd missing set-acceptance subcommand")
	}
	if !hasSetChecklistRemoved {
		t.Error("statusCmd missing set-checklist removed-error subcommand")
	}
}
