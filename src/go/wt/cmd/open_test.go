package main

import (
	"strings"
	"testing"
)

func TestOpen_ErrorNonexistentWorktree(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "open", "--app", "code", "nonexistent-wt")
	if r.ExitCode == 0 {
		t.Error("expected failure for nonexistent worktree")
	}
	assertContains(t, r.Stderr, "not found")
}

func TestOpen_ErrorFromMainRepoWithoutTarget(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "open", "--app", "code")
	if r.ExitCode == 0 {
		t.Error("expected failure from main repo without target")
	}
	assertContains(t, r.Stderr, "No worktree specified")
}

func TestOpen_ErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	r := runWt(t, dir, nil, "open")
	if r.ExitCode == 0 {
		t.Error("expected failure outside git repo")
	}
	assertContains(t, r.Stderr, "Not a git repository")
}

func TestOpen_ErrorUnknownApp(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "app-err")

	r := runWt(t, repo, nil, "open", "--app", "nonexistent-app", wtPath)
	if r.ExitCode == 0 {
		t.Error("expected failure for unknown app")
	}
	assertContains(t, r.Stderr, "Unknown app")
}

func TestOpen_AppDefault(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "default-test")

	// Clear environment to control detection path
	env := []string{
		"TERM_PROGRAM=",
		"TMUX=",
		"BYOBU_BACKEND=",
		"BYOBU_TTY=",
		"BYOBU_SESSION=",
		"BYOBU_CONFIG_DIR=",
		"HOME=" + t.TempDir(),
	}

	r := runWt(t, repo, env, "open", "--app", "default", wtPath)
	// Installed apps vary across environments (e.g., macOS always has Finder).
	// Accept either outcome, but verify the "default" keyword was recognized:
	// - exit 0: some default app resolved and opened
	// - non-zero: no default detected — should show our error, not "Unknown app"
	if r.ExitCode != 0 {
		assertContains(t, r.Stderr, "No default app detected")
	}
	// "default" must never be treated as a literal app name
	if strings.Contains(r.Stderr, "Unknown app: default") {
		t.Errorf("'default' was treated as a literal app name instead of the keyword: %s", r.Stderr)
	}
	if strings.Contains(r.Stderr, "panic") {
		t.Errorf("command panicked: %s", r.Stderr)
	}
}

// NOTE: Testing actual app opening (code, cursor, etc.) requires mock binaries
// on PATH that log their invocations. We test the error paths here; the
// open-by-name success path is tested via the worktree resolution logic
// (which is shared with other commands).
