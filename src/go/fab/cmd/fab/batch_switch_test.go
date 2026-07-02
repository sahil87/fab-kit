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

// batchSwitchFixture creates a fab root with one change folder and a
// fab/project/config.yaml carrying the given agent.spawn_command, chdirs into
// it (via hookTestEnv, TMUX set), and returns the resolvable change name.
func batchSwitchFixture(t *testing.T, spawnCommand string) (root, change string) {
	t.Helper()
	root = t.TempDir()
	change = "260401-ab12-add-feature"
	if err := os.MkdirAll(filepath.Join(root, "fab", "changes", change), 0o755); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent:\n  spawn_command: \"" + spawnCommand + "\"\n"
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	hookTestEnv(t, root, map[string]string{"TMUX": "/tmp/tmux-test/default,123,0"})
	return root, change
}

// stubBatchSwitchTmuxCapture stubs `wt` (echoing a worktree path) and a `tmux`
// that appends its full argument list to a capture file, prepended to $PATH.
// runBatchSwitch invokes both via raw exec.Command (PATH-resolved). Returns the
// capture file path.
func stubBatchSwitchTmuxCapture(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	capture := filepath.Join(t.TempDir(), "tmux-args")
	scripts := map[string]string{
		"wt":   "echo /fake/worktrees/switch",
		"tmux": `printf '%s\n' "$@" >> ` + capture,
	}
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return capture
}

// TestRunBatchSwitch_SpawnCommandPlaceholderStripping verifies that a templated
// agent.spawn_command has its {model}/{effort} placeholders stripped before it
// is interpolated into the tmux new-window shell command (no literal braces
// reach tmux), and that a non-templated command passes through verbatim.
func TestRunBatchSwitch_SpawnCommandPlaceholderStripping(t *testing.T) {
	t.Run("templated spawn_command stripped, no literal braces reach tmux", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "codex -m {model} -c model_reasoning_effort={effort}")
		capture := stubBatchSwitchTmuxCapture(t)

		cmd := batchSwitchCmd()
		var out, errOut bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		if err := runBatchSwitch(cmd, []string{change}, false, false); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, errOut.String())
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		got := string(args)
		if strings.Contains(got, "{model}") || strings.Contains(got, "{effort}") {
			t.Errorf("literal placeholder braces reached tmux:\n%s", got)
		}
		if !strings.Contains(got, "codex '/fab-switch") {
			t.Errorf("composed spawn command not stripped to `codex`:\n%s", got)
		}
	})

	t.Run("non-templated spawn_command passes through verbatim", func(t *testing.T) {
		_, change := batchSwitchFixture(t, "claude --dangerously-skip-permissions")
		capture := stubBatchSwitchTmuxCapture(t)

		cmd := batchSwitchCmd()
		var out, errOut bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		if err := runBatchSwitch(cmd, []string{change}, false, false); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, errOut.String())
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		if !strings.Contains(string(args), "claude --dangerously-skip-permissions '/fab-switch") {
			t.Errorf("non-templated spawn command not passed through verbatim:\n%s", string(args))
		}
	})
}
