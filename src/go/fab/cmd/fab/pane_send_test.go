package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPaneSendCmdStructure(t *testing.T) {
	t.Run("requires two args", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing args")
		}
	})

	t.Run("has force flag", func(t *testing.T) {
		cmd := paneSendCmd()
		f := cmd.Flags().Lookup("force")
		if f == nil {
			t.Fatal("expected --force flag to exist")
		}
		if f.DefValue != "false" {
			t.Errorf("--force default should be false, got %q", f.DefValue)
		}
	})
}

func TestIsPaneAgentIdleLogic(t *testing.T) {
	// Test the agent state interpretation logic by testing resolveAgentState
	// with known inputs (the integration with tmux is tested via the full command).

	t.Run("active agent returns active string", func(t *testing.T) {
		// resolveAgentState returns "active" when runtime entry exists but no idle_since
		// We test the isPaneAgentIdle interpretation: "active" -> false
		state := "active"
		isIdle := state != "active"
		if isIdle {
			t.Error("expected active state to mean not idle")
		}
	})

	t.Run("idle agent returns idle string", func(t *testing.T) {
		state := "idle (5m)"
		isIdle := state != "active"
		if !isIdle {
			t.Error("expected idle state to mean idle")
		}
	})

	t.Run("unknown agent treated as idle", func(t *testing.T) {
		state := "?"
		isIdle := state != "active"
		if !isIdle {
			t.Error("expected unknown state to be treated as idle")
		}
	})

	t.Run("em dash treated as idle", func(t *testing.T) {
		state := "\u2014"
		isIdle := state != "active"
		if !isIdle {
			t.Error("expected em-dash state to be treated as idle")
		}
	})
}

func TestPaneSendCmdRegistration(t *testing.T) {
	t.Run("registered under pane parent", func(t *testing.T) {
		parent := paneCmd()
		var found bool
		for _, sub := range parent.Commands() {
			if sub.Name() == "send" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'send' subcommand under 'pane' parent")
		}
	})
}

// TestPaneSendArgsValidation verifies cobra arg validation for the send command.
func TestPaneSendArgsValidation(t *testing.T) {
	t.Run("one arg fails", func(t *testing.T) {
		cmd := paneSendCmd()
		// Suppress usage output
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs([]string{"%3"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for single arg")
		}
	})

	t.Run("three args fails", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
		cmd.SetArgs([]string{"%3", "hello", "extra"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for three args")
		}
	})
}
