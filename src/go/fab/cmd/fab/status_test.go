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

func TestStatusSummaryCmds_RegisteredWithExpectedUse(t *testing.T) {
	if !strings.HasPrefix(statusSetSummaryCmd().Use, "set-summary ") {
		t.Errorf("statusSetSummaryCmd Use = %q, want prefix \"set-summary \"", statusSetSummaryCmd().Use)
	}
	if !strings.HasPrefix(statusGetSummaryCmd().Use, "get-summary ") {
		t.Errorf("statusGetSummaryCmd Use = %q, want prefix \"get-summary \"", statusGetSummaryCmd().Use)
	}
}

func TestStatusCmd_RegistersSummaryVerbs(t *testing.T) {
	root := statusCmd()
	hasSetSummary := false
	hasGetSummary := false
	for _, sub := range root.Commands() {
		switch {
		case strings.HasPrefix(sub.Use, "set-summary"):
			hasSetSummary = true
		case strings.HasPrefix(sub.Use, "get-summary"):
			hasGetSummary = true
		}
	}
	if !hasSetSummary {
		t.Error("statusCmd missing set-summary subcommand")
	}
	if !hasGetSummary {
		t.Error("statusCmd missing get-summary subcommand")
	}
}

func TestStatusBaseBranchCmds_RegisteredWithExpectedUse(t *testing.T) {
	if !strings.HasPrefix(statusSetBaseBranchCmd().Use, "set-base-branch ") {
		t.Errorf("statusSetBaseBranchCmd Use = %q, want prefix \"set-base-branch \"", statusSetBaseBranchCmd().Use)
	}
	if !strings.HasPrefix(statusGetBaseBranchCmd().Use, "get-base-branch ") {
		t.Errorf("statusGetBaseBranchCmd Use = %q, want prefix \"get-base-branch \"", statusGetBaseBranchCmd().Use)
	}
}

func TestStatusCmd_RegistersBaseBranchVerbs(t *testing.T) {
	root := statusCmd()
	hasSetBaseBranch := false
	hasGetBaseBranch := false
	for _, sub := range root.Commands() {
		switch {
		case strings.HasPrefix(sub.Use, "set-base-branch"):
			hasSetBaseBranch = true
		case strings.HasPrefix(sub.Use, "get-base-branch"):
			hasGetBaseBranch = true
		}
	}
	if !hasSetBaseBranch {
		t.Error("statusCmd missing set-base-branch subcommand")
	}
	if !hasGetBaseBranch {
		t.Error("statusCmd missing get-base-branch subcommand")
	}
}
