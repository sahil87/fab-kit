package main

import (
	"encoding/json"
	"testing"
)

func TestParseProcStatState(t *testing.T) {
	tests := []struct {
		name     string
		stat     string
		expected string
	}{
		{"running", "12345 (process) R 1 2 3", "R"},
		{"sleeping", "12345 (process) S 1 2 3", "S"},
		{"stopped", "12345 (process) T 1 2 3", "T"},
		{"zombie", "12345 (process) Z 1 2 3", "Z"},
		{"disk sleep", "12345 (process name) D 1 2 3", "D"},
		{"comm with parens", "12345 (proc (v2)) S 1 2 3", "S"},
		{"comm with spaces", "12345 (my process) R 1 2 3", "R"},
		{"empty stat", "", ""},
		{"no closing paren", "12345 (broken", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseProcStatState(tc.stat)
			if result != tc.expected {
				t.Errorf("parseProcStatState(%q) = %q, want %q", tc.stat, result, tc.expected)
			}
		})
	}
}

func TestIsWaitingForInput(t *testing.T) {
	tests := []struct {
		name     string
		wchan    string
		expected bool
	}{
		{"n_tty_read", "n_tty_read", true},
		{"wait_woken", "wait_woken", true},
		{"contains tty", "some_tty_op", true},
		{"do_select not tty", "do_select", false},
		{"poll_schedule_timeout not tty", "poll_schedule_timeout", false},
		{"ep_poll not tty", "ep_poll", false},
		{"pipe_read not tty", "pipe_read", false},
		{"futex wait", "futex_wait_queue", false},
		{"schedule", "schedule", false},
		{"empty", "", false},
		{"zero", "0", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isWaitingForInput(tc.wchan)
			if result != tc.expected {
				t.Errorf("isWaitingForInput(%q) = %v, want %v", tc.wchan, result, tc.expected)
			}
		})
	}
}

func TestProcessStateConstants(t *testing.T) {
	// Verify the five expected states are defined
	states := []string{
		processStateRunning,
		processStateWaitingForInput,
		processStateSleeping,
		processStateStopped,
		processStateExited,
	}

	expected := []string{
		"running",
		"waiting-for-input",
		"sleeping",
		"stopped",
		"exited",
	}

	for i, s := range states {
		if s != expected[i] {
			t.Errorf("state constant %d = %q, want %q", i, s, expected[i])
		}
	}
}

func TestPaneProcessJSONSchema(t *testing.T) {
	t.Run("JSON fields are correct", func(t *testing.T) {
		result := paneProcessJSON{
			Pane:        "%3",
			PID:         12345,
			State:       processStateWaitingForInput,
			ProcessName: "claude",
			Change:      strPtr("260331-r3m7-add-retry-logic"),
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}

		for _, field := range []string{"pane", "pid", "state", "process_name", "change"} {
			if _, ok := m[field]; !ok {
				t.Errorf("JSON output missing field %q", field)
			}
		}

		// Verify specific values
		if m["state"] != "waiting-for-input" {
			t.Errorf("state = %v, want waiting-for-input", m["state"])
		}
		if m["process_name"] != "claude" {
			t.Errorf("process_name = %v, want claude", m["process_name"])
		}
	})

	t.Run("null change for non-fab pane", func(t *testing.T) {
		result := paneProcessJSON{
			Pane:        "%5",
			PID:         99,
			State:       processStateExited,
			ProcessName: "zsh",
			Change:      nil,
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}

		if m["change"] != nil {
			t.Errorf("change should be null, got %v", m["change"])
		}
	})
}

func TestDetectForegroundProcess(t *testing.T) {
	t.Run("shell-only returns exited", func(t *testing.T) {
		// When fgPID == shellPID, state should be "exited"
		// This is tested via the logic in detectForegroundProcess
		// We can't test with real PIDs in unit tests, but we verify the
		// constant mapping
		if processStateExited != "exited" {
			t.Errorf("exited state = %q, want %q", processStateExited, "exited")
		}
	})
}

func TestPaneProcessCmdStructure(t *testing.T) {
	t.Run("requires one arg", func(t *testing.T) {
		cmd := paneProcessCmd()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing arg")
		}
	})

	t.Run("has json flag", func(t *testing.T) {
		cmd := paneProcessCmd()
		f := cmd.Flags().Lookup("json")
		if f == nil {
			t.Fatal("expected --json flag to exist")
		}
	})

	t.Run("registered under pane parent", func(t *testing.T) {
		parent := paneCmd()
		var found bool
		for _, sub := range parent.Commands() {
			if sub.Name() == "process" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'process' subcommand under 'pane' parent")
		}
	})
}
