package main

import (
	"bytes"
	"encoding/json"

	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func TestPaneCaptureCmd(t *testing.T) {
	t.Run("json and raw are mutually exclusive", func(t *testing.T) {
		cmd := paneCaptureCmd()
		cmd.SetArgs([]string{"%5", "--json", "--raw"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for mutually exclusive flags, got nil")
		}
	})

	t.Run("requires pane argument", func(t *testing.T) {
		cmd := paneCaptureCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing argument, got nil")
		}
	})

	t.Run("default line count is 50", func(t *testing.T) {
		cmd := paneCaptureCmd()
		lines, _ := cmd.Flags().GetInt("lines")
		if lines != 50 {
			t.Errorf("default lines = %d, want 50", lines)
		}
	})
}

func TestCaptureJSONStructure(t *testing.T) {
	t.Run("json output has correct fields", func(t *testing.T) {
		change := "260306-r3m7-test"
		stage := "apply"
		agentState := "idle"
		idleDur := "5m"
		out := captureJSON{
			Pane:              "%5",
			Lines:             50,
			Content:           "test content",
			Worktree:          "repo.worktrees/alpha/",
			Change:            &change,
			Stage:             &stage,
			AgentState:        &agentState,
			AgentIdleDuration: &idleDur,
		}

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(out); err != nil {
			t.Fatal(err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatal(err)
		}

		expectedFields := []string{"pane", "lines", "content", "worktree", "change", "stage", "agent_state", "agent_idle_duration"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("JSON output missing field %q", field)
			}
		}
	})

	t.Run("null fields for non-fab pane", func(t *testing.T) {
		out := captureJSON{
			Pane:     "%5",
			Lines:    50,
			Content:  "some output",
			Worktree: "tmp/",
		}

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		if err := enc.Encode(out); err != nil {
			t.Fatal(err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatal(err)
		}

		if result["change"] != nil {
			t.Errorf("change should be null, got %v", result["change"])
		}
		if result["stage"] != nil {
			t.Errorf("stage should be null, got %v", result["stage"])
		}
		if result["agent_state"] != nil {
			t.Errorf("agent_state should be null, got %v", result["agent_state"])
		}
	})
}

func TestPrintCaptureHeader(t *testing.T) {
	t.Run("header with full context", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		change := "260306-r3m7-test"
		stage := "apply"
		agentState := "idle"
		idleDur := "5m"

		ctx := &pane.PaneContext{
			Pane:              "%5",
			WorktreeDisplay:   "repo.worktrees/alpha/",
			Change:            &change,
			Stage:             &stage,
			AgentState:        &agentState,
			AgentIdleDuration: &idleDur,
		}

		printCaptureHeader(cmd, "%5", ctx)

		output := buf.String()
		if !strings.Contains(output, "pane %5") {
			t.Errorf("header missing pane ID: %q", output)
		}
		if !strings.Contains(output, "change: 260306-r3m7-test") {
			t.Errorf("header missing change: %q", output)
		}
		if !strings.Contains(output, "stage: apply") {
			t.Errorf("header missing stage: %q", output)
		}
		if !strings.Contains(output, "agent: idle (5m)") {
			t.Errorf("header missing agent state: %q", output)
		}
	})

	t.Run("header with no fab context", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		ctx := &pane.PaneContext{
			Pane:            "%5",
			WorktreeDisplay: "tmp/",
		}

		printCaptureHeader(cmd, "%5", ctx)

		output := buf.String()
		if !strings.Contains(output, "pane %5") {
			t.Errorf("header missing pane ID: %q", output)
		}
		if strings.Contains(output, "change:") {
			t.Errorf("header should not contain change: %q", output)
		}
	})
}

// Verify the capture command uses the correct line count flag name
func TestCaptureLineFlagShorthand(t *testing.T) {
	cmd := paneCaptureCmd()
	flag := cmd.Flags().Lookup("lines")
	if flag == nil {
		t.Fatal("expected 'lines' flag to exist")
	}
	if flag.Shorthand != "l" {
		t.Errorf("expected shorthand 'l', got %q", flag.Shorthand)
	}
	if flag.DefValue != "50" {
		t.Errorf("expected default 50, got %q", flag.DefValue)
	}

}
