package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEdge_WorktreeDeletedOutsideGit(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "orphaned")

	// Verify registered
	out := gitRun(t, repo, "worktree", "list")
	assertContains(t, out, "orphaned")

	// Delete outside git
	wtPath := worktreePath(repo, "orphaned")
	os.RemoveAll(wtPath)

	// Prune should clean it up
	gitRun(t, repo, "worktree", "prune")

	out = gitRun(t, repo, "worktree", "list")
	assertNotContains(t, out, "orphaned")
}

func TestEdge_ListWorksAfterExternalDeletion(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "ext-del")

	wtPath := worktreePath(repo, "ext-del")
	os.RemoveAll(wtPath)
	gitRun(t, repo, "worktree", "prune")

	r := runWtSuccess(t, repo, nil, "list")
	combined := r.Stdout + r.Stderr
	assertNotContains(t, combined, "ext-del")
}

func TestEdge_DeleteHandlesAlreadyRemovedDirectory(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "already-gone")

	// Remove directory but don't prune
	wtPath := worktreePath(repo, "already-gone")
	os.RemoveAll(wtPath)

	// Should not crash - either succeed or fail gracefully
	r := runWt(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "already-gone")
	if r.ExitCode > 1 {
		t.Errorf("expected exit code 0 or 1, got %d", r.ExitCode)
	}
}

func TestEdge_CreateWithInvalidBranchNameNoPartialState(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "create", "--non-interactive", "--worktree-name", "bad-branch", "refs/invalid..name")
	if r.ExitCode == 0 {
		t.Error("expected failure with invalid branch name")
	}

	// No partial worktree directory should be left behind
	assertDirNotExists(t, worktreePath(repo, "bad-branch"))
}

func TestEdge_NameCollisionClearError(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "collision")

	r := runWt(t, repo, nil, "create", "--non-interactive", "--worktree-name", "collision")
	if r.ExitCode == 0 {
		t.Error("expected failure on name collision")
	}
	assertContains(t, r.Stderr, "already exists")
}

func TestEdge_BranchWithSlashes(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "feature/deep/nested/branch")

	// Worktree name should be derived from last segment
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "branch")
}

func TestEdge_BranchWithSpecialChars(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "-b", "feature/my_special-branch")
	gitRun(t, repo, "checkout", "main")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "feature/my_special-branch")
	_ = r
}

func TestEdge_ListInDetachedHEAD(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "--detach", "HEAD")

	r := runWtSuccess(t, repo, nil, "list")
	// Should succeed and show detached state
	assertContains(t, r.Stdout, "(detached)")
}

func TestEdge_CreateFromDetachedHEAD(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "--detach", "HEAD")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "from-detached", "--worktree-init", "false")
	_ = r
	assertWorktreeExists(t, repo, "from-detached")
}

func TestEdge_MultipleWorktreesUniqueNames(t *testing.T) {
	repo := createTestRepo(t)

	names := make(map[string]bool)
	for i := 0; i < 5; i++ {
		r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-init", "false")
		wtPath := strings.TrimSpace(r.Stdout)
		name := filepath.Base(wtPath)
		if names[name] {
			t.Errorf("duplicate name: %s", name)
		}
		names[name] = true
	}
	if len(names) != 5 {
		t.Errorf("expected 5 unique names, got %d", len(names))
	}
}

func TestEdge_ListShowsAllWorktrees(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "edge-a")
	createWorktreeViaWt(t, repo, "edge-b")
	createWorktreeViaWt(t, repo, "edge-c")

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "edge-a")
	assertContains(t, r.Stdout, "edge-b")
	assertContains(t, r.Stdout, "edge-c")
}

func TestEdge_ListOnRepoWithNoWorktreesShowsMain(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "list")
	assertContains(t, r.Stdout, "(main)")
}

func TestEdge_DeleteAllWithNoWorktrees(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-all")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "No worktrees found")
}

func TestEdge_ReuseWithOrphanedDirectory(t *testing.T) {
	repo := createTestRepo(t)

	// Create orphaned directory manually (not a real git worktree)
	wtDir := worktreesDir(repo)
	os.MkdirAll(filepath.Join(wtDir, "orphaned"), 0755)

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--reuse", "--worktree-name", "orphaned")
	lastLine := strings.TrimSpace(r.Stdout)

	expected := filepath.Join(wtDir, "orphaned")
	if lastLine != expected {
		t.Errorf("expected %q, got %q", expected, lastLine)
	}
}

func TestEdge_GitFsckAfterCreateDelete(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "fsck-test")
	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "fsck-test")

	// git fsck should pass
	cmd := exec.Command("git", "-C", repo, "fsck", "--no-progress")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("git fsck failed after create-delete: %v\n%s", err, out)
	}
}
