package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit_RunsScriptWhenExists(t *testing.T) {
	repo := createTestRepo(t)
	createInitScript(t, repo)

	r := runWt(t, repo, []string{"WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh"}, "init")
	assertExitCode(t, r, 0)
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Running worktree init")
	assertContains(t, combined, "Test init script executed")
	assertContains(t, combined, "Worktree init complete")

	assertFileExists(t, filepath.Join(repo, ".init-script-ran"))
}

func TestInit_GuidanceWhenNoScript(t *testing.T) {
	repo := createTestRepo(t)

	// With WORKTREE_INIT_SCRIPT pointing to a non-existent file, shows guidance
	r := runWt(t, repo, []string{"WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh"}, "init")
	assertExitCode(t, r, 0)
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "No init script found")
	assertContains(t, combined, "To add an init script")
	assertContains(t, combined, "mkdir -p")
	assertContains(t, combined, "touch")
}

func TestInit_SkipsWhenCommandNotOnPath(t *testing.T) {
	repo := createTestRepo(t)

	// Use an explicitly nonexistent command (fab-kit may be on PATH in dev envs)
	r := runWt(t, repo, []string{"WORKTREE_INIT_SCRIPT=__nonexistent_cmd__ sync"}, "init")
	assertExitCode(t, r, 0)
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "not found on PATH")
	assertContains(t, combined, "skipping init")
}

func TestInit_ErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	r := runWt(t, dir, nil, "init")
	if r.ExitCode == 0 {
		t.Errorf("expected non-zero exit code outside git repo")
	}
	assertContains(t, r.Stderr, "Not a git repository")
}

func TestInit_Idempotent(t *testing.T) {
	repo := createTestRepo(t)
	createInitScript(t, repo)

	env := []string{"WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh"}
	r1 := runWt(t, repo, env, "init")
	assertExitCode(t, r1, 0)

	r2 := runWt(t, repo, env, "init")
	assertExitCode(t, r2, 0)
}

func TestInit_RunsFromWorktree(t *testing.T) {
	repo := createTestRepo(t)
	createInitScript(t, repo)

	// Commit the init script so worktrees see it
	gitRun(t, repo, "add", "scripts/worktree-init.sh")
	gitRun(t, repo, "commit", "-q", "-m", "Add init script")

	wtPath := createWorktreeViaWt(t, repo, "init-wt-test")

	r := runWt(t, wtPath, []string{"WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh"}, "init")
	assertExitCode(t, r, 0)
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Worktree init complete")
}

func TestInit_RunsInRepoRoot(t *testing.T) {
	repo := createTestRepo(t)

	// Create a script that writes pwd to a file
	scriptDir := filepath.Join(repo, "scripts")
	os.MkdirAll(scriptDir, 0755)
	script := filepath.Join(scriptDir, "worktree-init.sh")
	content := `#!/usr/bin/env bash
pwd > current-dir.txt
`
	os.WriteFile(script, []byte(content), 0755)

	r := runWt(t, repo, []string{"WORKTREE_INIT_SCRIPT=scripts/worktree-init.sh"}, "init")
	assertExitCode(t, r, 0)

	dirContent, err := os.ReadFile(filepath.Join(repo, "current-dir.txt"))
	if err != nil {
		t.Fatalf("current-dir.txt not created: %v", err)
	}
	// The script should run in the repo root
	assertContains(t, string(dirContent), filepath.Base(repo))
}
