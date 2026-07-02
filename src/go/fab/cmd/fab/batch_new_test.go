package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/backlog"
)

const testBacklog = `# Backlog

- [ ] [90g5] 2026-04-01: Add retry logic to API client
- [x] [done] 2026-03-30: Fix login page styling
- [ ] [jgt6] [DEV-123] 2026-04-01: Implement caching layer
  with Redis support for session storage
- [ ] [ab12] (BUG) 2026-04-02: Fix memory leak in worker pool
`

func writeTestBacklog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")
	if err := os.WriteFile(path, []byte(testBacklog), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParsePendingItems(t *testing.T) {
	path := writeTestBacklog(t)

	items, err := backlog.ParsePending(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 pending items, got %d", len(items))
	}

	if items[0].ID != "90g5" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "90g5")
	}
	if items[1].ID != "jgt6" {
		t.Errorf("items[1].ID = %q, want %q", items[1].ID, "jgt6")
	}
	if items[2].ID != "ab12" {
		t.Errorf("items[2].ID = %q, want %q", items[2].ID, "ab12")
	}
}

func TestExtractBacklogContent_SimpleItem(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := backlog.ExtractContent(path, "90g5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Add retry logic to API client" {
		t.Errorf("content = %q, want %q", content, "Add retry logic to API client")
	}
}

func TestExtractBacklogContent_ContinuationLine(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := backlog.ExtractContent(path, "jgt6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Implement caching layer with Redis support for session storage"
	if content != expected {
		t.Errorf("content = %q, want %q", content, expected)
	}
}

func TestExtractBacklogContent_NotFound(t *testing.T) {
	path := writeTestBacklog(t)

	_, err := backlog.ExtractContent(path, "zzzz")
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestExtractBacklogContent_BugPrefix(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := backlog.ExtractContent(path, "ab12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Fix memory leak in worker pool" {
		t.Errorf("content = %q, want %q", content, "Fix memory leak in worker pool")
	}
}

func TestListPendingItems_MissingBacklogReturnsError(t *testing.T) {
	var buf strings.Builder
	err := listPendingItems(&buf, filepath.Join(t.TempDir(), "backlog.md"))
	if err == nil {
		t.Fatal("expected error for unreadable backlog, got nil (previously listed nothing)")
	}
}

func TestBatchNewCmd_Structure(t *testing.T) {
	cmd := batchNewCmd()
	if cmd.Use != "new [backlog-id...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "new [backlog-id...]")
	}

	// Verify flags exist
	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}

// chdirBatchNewFixture creates a temp fab root (fab/backlog.md with the given
// content) and chdirs into it so resolve.FabRoot() resolves to the fixture.
// Restores the previous working directory on cleanup.
func chdirBatchNewFixture(t *testing.T, backlogContent string) string {
	t.Helper()
	dir := t.TempDir()
	fabDir := filepath.Join(dir, "fab")
	if err := os.MkdirAll(fabDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "backlog.md"), []byte(backlogContent), 0o644); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})
	return dir
}

// stubBatchNewBinaries writes fake `wt` and `tmux` executables into a temp
// dir prepended to $PATH, so runBatchNew's launch loop can be exercised
// in-process without a tmux server. tmuxScript/wtScript are POSIX sh bodies.
func stubBatchNewBinaries(t *testing.T, wtScript, tmuxScript string) {
	t.Helper()
	bin := t.TempDir()
	for name, body := range map[string]string{"wt": wtScript, "tmux": tmuxScript} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// runBatchNewCmd executes `fab batch new <args...>` in-process and returns
// (stdout, stderr, err).
func runBatchNewCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := batchNewCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// TestRunBatchNew_NoTmux verifies the $TMUX guard returns an error through
// RunE (previously os.Exit(1)) — stderr becomes `ERROR: not inside a tmux
// session` via main.go's central handler.
func TestRunBatchNew_NoTmux(t *testing.T) {
	chdirBatchNewFixture(t, testBacklog)
	t.Setenv("TMUX", "")

	_, _, err := runBatchNewCmd(t, "90g5")
	if err == nil {
		t.Fatal("expected error when $TMUX is unset, got nil")
	}
	if err.Error() != "not inside a tmux session" {
		t.Errorf("error = %q, want %q", err.Error(), "not inside a tmux session")
	}
}

// TestRunBatchNew_NoPendingItems verifies the --all empty-backlog guard
// returns an error through RunE (previously os.Exit(1)). The error string is
// the intake-pinned deliberate output change: `ERROR: No pending backlog
// items found.` after the central handler's prefix.
func TestRunBatchNew_NoPendingItems(t *testing.T) {
	chdirBatchNewFixture(t, "# Backlog\n\n- [x] [done] 2026-03-30: Fix login page styling\n")
	t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")

	_, _, err := runBatchNewCmd(t, "--all")
	if err == nil {
		t.Fatal("expected error for empty pending backlog, got nil")
	}
	if err.Error() != "No pending backlog items found." {
		t.Errorf("error = %q, want %q", err.Error(), "No pending backlog items found.")
	}
}

// TestRunBatchNew_LaunchFailures exercises the launch loop with PATH-stubbed
// wt/tmux binaries: a tmux new-window failure must produce a per-item FAILED
// line naming the already-created worktree path, and a non-nil error (→
// non-zero exit) — never the silent exit 0 that orphaned worktrees before.
func TestRunBatchNew_LaunchFailures(t *testing.T) {
	t.Run("tmux failure: FAILED line + worktree path + non-nil error", func(t *testing.T) {
		chdirBatchNewFixture(t, testBacklog)
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		stubBatchNewBinaries(t,
			"echo /fake/worktrees/90g5",
			"echo 'boom: create window failed' 1>&2; exit 1")

		_, stderr, err := runBatchNewCmd(t, "90g5")
		if err == nil {
			t.Fatal("expected non-nil error when a launch fails, got nil (silent exit 0 regression)")
		}
		if want := "1 of 1 item(s) failed to launch"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
		for _, fragment := range []string{"[90g5] FAILED: tmux new-window:", "boom: create window failed", "worktree already created at /fake/worktrees/90g5"} {
			if !strings.Contains(stderr, fragment) {
				t.Errorf("stderr missing %q:\n%s", fragment, stderr)
			}
		}
	})

	t.Run("wt create failure: FAILED line with wt stderr + non-nil error", func(t *testing.T) {
		chdirBatchNewFixture(t, testBacklog)
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		stubBatchNewBinaries(t,
			"echo 'worktree 90g5 already exists' 1>&2; exit 1",
			"exit 0")

		_, stderr, err := runBatchNewCmd(t, "90g5")
		if err == nil {
			t.Fatal("expected non-nil error when wt create fails, got nil")
		}
		for _, fragment := range []string{"[90g5] FAILED: wt create:", "worktree 90g5 already exists"} {
			if !strings.Contains(stderr, fragment) {
				t.Errorf("stderr missing %q:\n%s", fragment, stderr)
			}
		}
	})

	t.Run("all launches succeed: nil error", func(t *testing.T) {
		chdirBatchNewFixture(t, testBacklog)
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		stubBatchNewBinaries(t,
			"echo /fake/worktrees/90g5",
			"exit 0")

		out, _, err := runBatchNewCmd(t, "90g5")
		if err != nil {
			t.Fatalf("expected nil error for successful launch, got %v", err)
		}
		if !strings.Contains(out, "[90g5]") {
			t.Errorf("stdout missing launched item line:\n%s", out)
		}
	})

	t.Run("skipped items excluded from the failure denominator", func(t *testing.T) {
		chdirBatchNewFixture(t, testBacklog)
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		stubBatchNewBinaries(t,
			"echo /fake/worktrees/90g5",
			"echo 'boom: create window failed' 1>&2; exit 1")

		// zzzz is warn-and-skip (never launch-attempted); 90g5 fails launch.
		// The summary denominator counts attempts, not requested IDs.
		_, _, err := runBatchNewCmd(t, "zzzz", "90g5")
		if err == nil {
			t.Fatal("expected non-nil error when a launch fails, got nil")
		}
		if want := "1 of 1 item(s) failed to launch"; err.Error() != want {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	})

	t.Run("unknown backlog id stays warn-and-skip (no failure exit)", func(t *testing.T) {
		chdirBatchNewFixture(t, testBacklog)
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		stubBatchNewBinaries(t, "echo /fake/wt", "exit 0")

		_, stderr, err := runBatchNewCmd(t, "zzzz")
		if err != nil {
			t.Fatalf("backlog-lookup misses are skips, not failures; got error %v", err)
		}
		if !strings.Contains(stderr, "Warning: [zzzz] not found in backlog, skipping") {
			t.Errorf("stderr missing skip warning:\n%s", stderr)
		}
	})
}

// writeBatchNewConfig writes fab/project/config.yaml under an existing fixture
// root (from chdirBatchNewFixture) with the given agent.spawn_command, so
// spawn.Command reads it instead of falling back to DefaultSpawnCommand.
func writeBatchNewConfig(t *testing.T, root, spawnCommand string) {
	t.Helper()
	projectDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "agent:\n  spawn_command: \"" + spawnCommand + "\"\n"
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// stubBatchNewTmuxCapture stubs `wt` (echoing a worktree path) and a `tmux`
// that appends its full argument list to a capture file, so a test can inspect
// the composed `tmux new-window` shell command (the last argument). Returns the
// capture file path.
func stubBatchNewTmuxCapture(t *testing.T) string {
	t.Helper()
	capture := filepath.Join(t.TempDir(), "tmux-args")
	stubBatchNewBinaries(t,
		"echo /fake/worktrees/wt",
		`printf '%s\n' "$@" >> `+capture+`; exit 0`)
	return capture
}

// TestRunBatchNew_SpawnCommandPlaceholderStripping verifies that a templated
// agent.spawn_command has its {model}/{effort} placeholders stripped before it
// is interpolated into the tmux new-window shell command (no literal braces
// reach tmux), and that a non-templated command passes through verbatim.
func TestRunBatchNew_SpawnCommandPlaceholderStripping(t *testing.T) {
	t.Run("templated spawn_command stripped, no literal braces reach tmux", func(t *testing.T) {
		root := chdirBatchNewFixture(t, testBacklog)
		writeBatchNewConfig(t, root, "codex -m {model} -c model_reasoning_effort={effort}")
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		capture := stubBatchNewTmuxCapture(t)

		if _, stderr, err := runBatchNewCmd(t, "90g5"); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		got := string(args)
		if strings.Contains(got, "{model}") || strings.Contains(got, "{effort}") {
			t.Errorf("literal placeholder braces reached tmux:\n%s", got)
		}
		if !strings.Contains(got, "codex '/fab-new") {
			t.Errorf("composed spawn command not stripped to `codex`:\n%s", got)
		}
	})

	t.Run("non-templated spawn_command passes through verbatim", func(t *testing.T) {
		root := chdirBatchNewFixture(t, testBacklog)
		writeBatchNewConfig(t, root, "claude --dangerously-skip-permissions")
		t.Setenv("TMUX", "/tmp/tmux-fake/default,123,0")
		capture := stubBatchNewTmuxCapture(t)

		if _, stderr, err := runBatchNewCmd(t, "90g5"); err != nil {
			t.Fatalf("expected nil error, got %v\nstderr: %s", err, stderr)
		}

		args, err := os.ReadFile(capture)
		if err != nil {
			t.Fatalf("reading tmux capture: %v", err)
		}
		if !strings.Contains(string(args), "claude --dangerously-skip-permissions '/fab-new") {
			t.Errorf("non-templated spawn command not passed through verbatim:\n%s", string(args))
		}
	})
}
