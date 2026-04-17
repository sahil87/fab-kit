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

func TestCapturePaneArgs(t *testing.T) {
	t.Run("default 50 lines uses -S -50", func(t *testing.T) {
		args := capturePaneArgs("", "%5", 50)
		expected := []string{"capture-pane", "-t", "%5", "-p", "-S", "-50"}
		if len(args) != len(expected) {
			t.Fatalf("args length = %d, want %d: got %v", len(args), len(expected), args)
		}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})

	t.Run("custom 20 lines uses -S -20", func(t *testing.T) {
		args := capturePaneArgs("", "%5", 20)
		expected := []string{"capture-pane", "-t", "%5", "-p", "-S", "-20"}
		if len(args) != len(expected) {
			t.Fatalf("args length = %d, want %d: got %v", len(args), len(expected), args)
		}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})

	t.Run("lines=1 is the minimum valid value", func(t *testing.T) {
		args := capturePaneArgs("", "%5", 1)
		expected := []string{"capture-pane", "-t", "%5", "-p", "-S", "-1"}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})

	t.Run("non-empty server prepends -L", func(t *testing.T) {
		args := capturePaneArgs("runKit", "%5", 50)
		expected := []string{"-L", "runKit", "capture-pane", "-t", "%5", "-p", "-S", "-50"}
		if len(args) != len(expected) {
			t.Fatalf("args length = %d, want %d: got %v", len(args), len(expected), args)
		}
		for i, v := range expected {
			if args[i] != v {
				t.Errorf("args[%d] = %q, want %q", i, args[i], v)
			}
		}
	})

	t.Run("empty server returns same argv as before change", func(t *testing.T) {
		args := capturePaneArgs("", "%5", 50)
		for _, el := range args {
			if el == "-L" {
				t.Errorf("did not expect -L in argv for empty server, got %v", args)
			}
		}
	})
}

func TestPaneCaptureServerFlag(t *testing.T) {
	t.Run("--server flag inherited from pane parent", func(t *testing.T) {
		parent := paneCmd()
		var sub *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "capture <pane>" {
				sub = c
				break
			}
		}
		if sub == nil {
			t.Fatal("paneCmd did not register a capture subcommand")
		}
		flag := sub.Flags().Lookup("server")
		if flag == nil {
			flag = sub.InheritedFlags().Lookup("server")
		}
		if flag == nil {
			t.Fatal("expected --server flag to be visible on pane capture subcommand")
		}
		if flag.Shorthand != "L" {
			t.Errorf("expected shorthand \"L\", got %q", flag.Shorthand)
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
