package main

import (
	"encoding/json"
	"testing"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"single line", "hello\n", 1},
		{"multiple lines", "line1\nline2\nline3\n", 3},
		{"no trailing newline", "line1\nline2", 2},
		{"blank lines", "line1\n\nline3\n", 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := countLines(tc.input)
			if result != tc.expected {
				t.Errorf("countLines(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

func TestResolveAgentStateForJSON(t *testing.T) {
	tests := []struct {
		name     string
		agent    string
		expected string
	}{
		{"em dash returns empty", "\u2014", ""},
		{"question mark returns unknown", "?", "unknown"},
		{"idle with duration returns idle", "idle (5m)", "idle"},
		{"active returns active", "active", "active"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveAgentStateForJSON(tc.agent)
			if result != tc.expected {
				t.Errorf("resolveAgentStateForJSON(%q) = %q, want %q", tc.agent, result, tc.expected)
			}
		})
	}
}

func TestPaneCaptureJSONSchema(t *testing.T) {
	t.Run("JSON fields are correct", func(t *testing.T) {
		result := paneCaptureJSON{
			Pane:       "%3",
			Lines:      20,
			Content:    "hello world\n",
			Change:     strPtr("260331-r3m7-add-retry-logic"),
			Stage:      strPtr("apply"),
			AgentState: strPtr("idle"),
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}

		for _, field := range []string{"pane", "lines", "content", "change", "stage", "agent_state"} {
			if _, ok := m[field]; !ok {
				t.Errorf("JSON output missing field %q", field)
			}
		}
	})

	t.Run("null fields for no fab context", func(t *testing.T) {
		result := paneCaptureJSON{
			Pane:       "%5",
			Lines:      10,
			Content:    "some output\n",
			Change:     nil,
			Stage:      nil,
			AgentState: nil,
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}

		for _, field := range []string{"change", "stage", "agent_state"} {
			if m[field] != nil {
				t.Errorf("field %q should be null, got %v", field, m[field])
			}
		}
	})
}
