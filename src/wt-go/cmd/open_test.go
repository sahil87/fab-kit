package main

import (
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

// NOTE: Testing actual app opening (code, cursor, etc.) requires mock binaries
// on PATH that log their invocations. The bats tests used mock scripts for this.
// We test the error paths here; the open-by-name success path is tested via the
// worktree resolution logic (which is shared with other commands).

// TODO: When wt pr is implemented in Go, port tests from wt-pr.bats here.
