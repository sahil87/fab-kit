package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execLogCommand runs `fab log command <args...>` via the cobra command and
// returns (execute error, stderr).
func execLogCommand(t *testing.T, args ...string) (error, string) {
	t.Helper()
	cmd := logCommandCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	return cmd.Execute(), errOut.String()
}

// TestLogCommand_NoFabRootExitsZeroWithWarning: `fab log command` owns its
// best-effort contract (F28) — outside a fab repo it warns on stderr and
// still exits 0, so an unguarded skill call can never STOP a pipeline.
func TestLogCommand_NoFabRootExitsZeroWithWarning(t *testing.T) {
	hookTestEnv(t, t.TempDir(), map[string]string{})

	err, stderr := execLogCommand(t, "fab-test")
	if err != nil {
		t.Fatalf("log command must exit 0 without a fab root, got: %v", err)
	}
	if !strings.Contains(stderr, "Warning: fab log command:") {
		t.Errorf("expected stderr warning, got: %q", stderr)
	}
}

// TestLogCommand_BadExplicitChangeExitsZeroWithWarning: an explicit change
// arg that fails to resolve (previously exit 1) is now warn-and-exit-0.
func TestLogCommand_BadExplicitChangeExitsZeroWithWarning(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	hookTestEnv(t, root, map[string]string{})

	err, stderr := execLogCommand(t, "fab-test", "zzzz-no-such-change")
	if err != nil {
		t.Fatalf("log command must exit 0 on a bad change arg, got: %v", err)
	}
	if !strings.Contains(stderr, "Warning: fab log command:") {
		t.Errorf("expected stderr warning, got: %q", stderr)
	}
}

// TestLogCommand_SuccessAppendsEntry: the happy path still appends to
// .history.jsonl with no warning.
func TestLogCommand_SuccessAppendsEntry(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "fab", "changes", "260401-ab12-add-feature")
	os.MkdirAll(changeDir, 0o755)
	hookTestEnv(t, root, map[string]string{})

	err, stderr := execLogCommand(t, "fab-test", "ab12")
	if err != nil {
		t.Fatalf("log command failed: %v", err)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %q", stderr)
	}

	data, readErr := os.ReadFile(filepath.Join(changeDir, ".history.jsonl"))
	if readErr != nil {
		t.Fatalf("history file not written: %v", readErr)
	}
	if !strings.Contains(string(data), `"cmd":"fab-test"`) {
		t.Errorf("history entry missing, got: %s", data)
	}
}

// TestLogCommand_UnwritableHistoryExitsZeroWithWarning: an unwritable
// .history.jsonl (previously exit 1) warns and exits 0.
func TestLogCommand_UnwritableHistoryExitsZeroWithWarning(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "fab", "changes", "260401-ab12-add-feature")
	os.MkdirAll(changeDir, 0o755)
	// Make the history path unopenable by creating it as a directory.
	os.MkdirAll(filepath.Join(changeDir, ".history.jsonl"), 0o755)
	hookTestEnv(t, root, map[string]string{})

	err, stderr := execLogCommand(t, "fab-test", "ab12")
	if err != nil {
		t.Fatalf("log command must exit 0 on unwritable history, got: %v", err)
	}
	if !strings.Contains(stderr, "Warning: fab log command:") {
		t.Errorf("expected stderr warning, got: %q", stderr)
	}
}
