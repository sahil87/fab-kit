package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllChangeNames(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "260401-cd34-fix-bug"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	names := allChangeNames(dir)
	if len(names) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(names))
	}

	// Should not include "archive"
	for _, name := range names {
		if name == "archive" {
			t.Error("archive should be excluded")
		}
	}
}

func TestAllChangeNames_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := allChangeNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 changes, got %d", len(names))
	}
}

// (getBranchPrefix was retired in 260612-ye8r — branch_prefix now comes from
// the shared internal/config accessor, tested in internal/config.)

// TestRunBatchSwitch_NoTmuxReturnsError verifies the $TMUX guard returns an
// error through RunE (previously os.Exit(1)) — stderr becomes
// `ERROR: not inside a tmux session` via main.go's single formatter.
func TestRunBatchSwitch_NoTmuxReturnsError(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes"), 0o755)
	hookTestEnv(t, root, map[string]string{"TMUX": ""})

	cmd := batchSwitchCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	err := runBatchSwitch(cmd, []string{"whatever"}, false, false)
	if err == nil {
		t.Fatal("expected error outside tmux")
	}
	if !strings.Contains(err.Error(), "not inside a tmux session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestRunBatchSwitch_UnresolvableWarnsAndSkips verifies in-process resolution
// (resolve.ToFolder, no `fab change resolve` subprocess): an unresolvable
// name warns with the resolver's SPECIFIC error and the loop continues
// (exit 0).
func TestRunBatchSwitch_UnresolvableWarnsAndSkips(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "fab", "changes", "260401-ab12-add-feature"), 0o755)
	hookTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})

	cmd := batchSwitchCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := runBatchSwitch(cmd, []string{"zzzz-no-such-change"}, false, false); err != nil {
		t.Fatalf("warn-and-skip path must not error, got: %v", err)
	}
	stderr := errOut.String()
	if !strings.Contains(stderr, "could not resolve 'zzzz-no-such-change'") {
		t.Errorf("missing warn-and-skip warning, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "No change matches") {
		t.Errorf("warning must surface the resolver's specific error, got:\n%s", stderr)
	}
}

func TestListChanges(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	var buf bytes.Buffer
	listChanges(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("260401-ab12-add-feature")) {
		t.Error("expected change name in output")
	}
	if bytes.Contains([]byte(output), []byte("archive")) && !bytes.Contains([]byte(output), []byte("Available changes")) {
		t.Error("archive should not appear in list")
	}
}

func TestBatchSwitchCmd_Structure(t *testing.T) {
	cmd := batchSwitchCmd()
	if cmd.Use != "switch [change...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "switch [change...]")
	}

	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}
