package log

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupLogFixture creates a minimal fab/ structure with a change directory.
// Returns (fabRoot, changeFolder).
func setupLogFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	// Create .fab-status.yaml symlink so resolve works
	symlinkPath := filepath.Join(dir, ".fab-status.yaml")
	os.Symlink("fab/changes/"+folder+"/.status.yaml", symlinkPath)

	return fabRoot, folder
}

// readLastLine reads the last non-empty line from a file.
func readLastLine(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines[len(lines)-1]
}

// parseJSON parses a JSON string into a map.
func parseJSON(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("failed to parse JSON %q: %v", s, err)
	}
	return m
}

func TestCommand(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := Command(fabRoot, "fab-continue", folder, "")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if m["event"] != "command" {
		t.Errorf("event = %v, want command", m["event"])
	}
	if m["cmd"] != "fab-continue" {
		t.Errorf("cmd = %v, want fab-continue", m["cmd"])
	}
	if _, ok := m["ts"]; !ok {
		t.Error("missing ts field")
	}
}

func TestCommand_OptionalFieldsOmitted(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := Command(fabRoot, "fab-status", folder, "")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if _, ok := m["args"]; ok {
		t.Error("args field should be omitted when empty")
	}
}

func TestCommand_WithArgs(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := Command(fabRoot, "fab-new", folder, "--slug test-change")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if m["args"] != "--slug test-change" {
		t.Errorf("args = %v, want '--slug test-change'", m["args"])
	}
}

func TestTransition(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := Transition(fabRoot, folder, "spec", "finish", "", "", "fab-ff")
	if err != nil {
		t.Fatalf("Transition failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if m["event"] != "stage-transition" {
		t.Errorf("event = %v, want stage-transition", m["event"])
	}
	if m["stage"] != "spec" {
		t.Errorf("stage = %v, want spec", m["stage"])
	}
	if m["action"] != "finish" {
		t.Errorf("action = %v, want finish", m["action"])
	}
	if m["driver"] != "fab-ff" {
		t.Errorf("driver = %v, want fab-ff", m["driver"])
	}
	// from and reason should be omitted (empty strings)
	if _, ok := m["from"]; ok {
		t.Error("from field should be omitted when empty")
	}
	if _, ok := m["reason"]; ok {
		t.Error("reason field should be omitted when empty")
	}
}

func TestReview(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := Review(fabRoot, folder, "passed", "")
	if err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if m["event"] != "review" {
		t.Errorf("event = %v, want review", m["event"])
	}
	if m["result"] != "passed" {
		t.Errorf("result = %v, want passed", m["result"])
	}
	if _, ok := m["rework"]; ok {
		t.Error("rework field should be omitted when empty")
	}
}

func TestConfidenceLog(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	err := ConfidenceLog(fabRoot, folder, 4.2, "+0.3", "spec")
	if err != nil {
		t.Fatalf("ConfidenceLog failed: %v", err)
	}

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	if m["event"] != "confidence" {
		t.Errorf("event = %v, want confidence", m["event"])
	}
	if m["score"] != 4.2 {
		t.Errorf("score = %v, want 4.2", m["score"])
	}
	if m["delta"] != "+0.3" {
		t.Errorf("delta = %v, want +0.3", m["delta"])
	}
	if m["trigger"] != "spec" {
		t.Errorf("trigger = %v, want spec", m["trigger"])
	}
}

func TestAppendOnly(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	// Write two events
	Command(fabRoot, "fab-status", folder, "")
	Command(fabRoot, "fab-continue", folder, "")

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	data, err := os.ReadFile(historyFile)
	if err != nil {
		t.Fatalf("failed to read history: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Verify first entry preserved
	m1 := parseJSON(t, lines[0])
	if m1["cmd"] != "fab-status" {
		t.Errorf("first entry cmd = %v, want fab-status", m1["cmd"])
	}

	m2 := parseJSON(t, lines[1])
	if m2["cmd"] != "fab-continue" {
		t.Errorf("second entry cmd = %v, want fab-continue", m2["cmd"])
	}
}

func TestTimestampFormat(t *testing.T) {
	fabRoot, folder := setupLogFixture(t)

	Command(fabRoot, "fab-status", folder, "")

	historyFile := filepath.Join(fabRoot, "changes", folder, ".history.jsonl")
	line := readLastLine(t, historyFile)
	m := parseJSON(t, line)

	ts, ok := m["ts"].(string)
	if !ok {
		t.Fatal("ts is not a string")
	}
	// RFC3339 contains a T separator and timezone offset
	if !strings.Contains(ts, "T") {
		t.Errorf("timestamp %q does not look like ISO 8601", ts)
	}
}
