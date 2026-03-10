package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration_CreateListDeleteLifecycle(t *testing.T) {
	repo := createTestRepo(t)

	// Create
	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "lifecycle-test", "--worktree-init", "false")
	assertContains(t, r.Stderr, "Created worktree: lifecycle-test")

	// List
	r = runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "lifecycle-test")

	// Delete
	r = runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "lifecycle-test")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")

	// Verify gone from list
	r = runWtSuccess(t, repo, nil, "list")
	assertNotContains(t, r.Stdout, "lifecycle-test")
}

func TestIntegration_CreateMultipleDeleteAll(t *testing.T) {
	repo := createTestRepo(t)

	createWorktreeViaWt(t, repo, "multi-1")
	createWorktreeViaWt(t, repo, "multi-2")
	createWorktreeViaWt(t, repo, "multi-3")

	// Verify all exist
	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "multi-1")
	assertContains(t, r.Stdout, "multi-2")
	assertContains(t, r.Stdout, "multi-3")

	// Delete all
	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-all")

	// Verify all gone
	r = runWtSuccess(t, repo, nil, "list")
	assertNotContains(t, r.Stdout, "multi-1")
	assertNotContains(t, r.Stdout, "multi-2")
	assertNotContains(t, r.Stdout, "multi-3")

	// Verify branches cleaned up
	assertBranchNotExists(t, repo, "multi-1")
	assertBranchNotExists(t, repo, "multi-2")
	assertBranchNotExists(t, repo, "multi-3")
}

func TestIntegration_NonInteractiveAutomation(t *testing.T) {
	repo := createTestRepo(t)

	// Full lifecycle using only --non-interactive flags
	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "auto-test", "--worktree-init", "false")

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "auto-test")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "auto-test", "--delete-branch", "true", "--delete-remote", "true")

	r = runWtSuccess(t, repo, nil, "list")
	assertNotContains(t, r.Stdout, "auto-test")
}

func TestIntegration_CreatedWorktreeHasCorrectBranch(t *testing.T) {
	repo := createTestRepo(t)

	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "branch-verify", "--worktree-init", "false")
	assertBranchExists(t, repo, "branch-verify")

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "branch-verify")
}

func TestIntegration_BranchDeletePreservesOthers(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "-b", "feature/keep-me")
	os.WriteFile(filepath.Join(repo, "keep.txt"), []byte("keep"), 0644)
	gitRun(t, repo, "add", "keep.txt")
	gitRun(t, repo, "commit", "-q", "-m", "keep")
	gitRun(t, repo, "checkout", "main")

	gitRun(t, repo, "checkout", "-b", "feature/delete-me")
	gitRun(t, repo, "checkout", "main")

	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "del-branch", "feature/delete-me")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "del-branch", "--delete-branch", "true")

	// delete-me should be gone
	assertBranchNotExists(t, repo, "feature/delete-me")

	// keep-me should still exist
	assertBranchExists(t, repo, "feature/keep-me")
}

func TestIntegration_RapidCreateDeleteCycle(t *testing.T) {
	repo := createTestRepo(t)

	for i := 1; i <= 3; i++ {
		name := "cycle-" + strings.Repeat("x", i) // cycle-x, cycle-xx, cycle-xxx for uniqueness
		createWorktreeViaWt(t, repo, name)
		runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", name)
	}

	// All should be gone
	r := runWtSuccess(t, repo, nil, "list")
	assertNotContains(t, r.Stdout, "cycle-")
}

func TestIntegration_GitStateCleanAfterCreateDelete(t *testing.T) {
	repo := createTestRepo(t)

	createWorktreeViaWt(t, repo, "integrity-test")
	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "integrity-test")

	assertGitStateClean(t, repo)
}

func TestIntegration_MainRepoUnaffectedByWorktreeOps(t *testing.T) {
	repo := createTestRepo(t)

	initialCommit := gitRun(t, repo, "rev-parse", "HEAD")

	// Create, modify, and delete a worktree
	wtPath := createWorktreeViaWt(t, repo, "no-affect")
	os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("new"), 0644)
	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-q", "-m", "worktree commit")
	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "no-affect")

	currentCommit := gitRun(t, repo, "rev-parse", "HEAD")
	if initialCommit != currentCommit {
		t.Errorf("main repo HEAD changed: %s -> %s", initialCommit, currentCommit)
	}
}

func TestIntegration_WorktreeCommitIndependent(t *testing.T) {
	repo := createTestRepo(t)

	wtPath := createWorktreeViaWt(t, repo, "independent")

	// Commit in worktree
	os.WriteFile(filepath.Join(wtPath, "wt-file.txt"), []byte("wt content"), 0644)
	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-q", "-m", "wt change")

	// File should not exist in main repo
	if _, err := os.Stat(filepath.Join(repo, "wt-file.txt")); err == nil {
		t.Error("wt-file.txt should not exist in main repo")
	}

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "independent")
}
