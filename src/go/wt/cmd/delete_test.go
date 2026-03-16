package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDelete_ByName(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "test-wt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "test-wt")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")
	assertWorktreeNotExists(t, repo, "test-wt")
}

func TestDelete_BranchDeletedByDefault(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "branch-test")

	assertBranchExists(t, repo, "branch-test")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "branch-test")

	assertWorktreeNotExists(t, repo, "branch-test")
	assertBranchNotExists(t, repo, "branch-test")
}

func TestDelete_PreservesBranchWhenFalse(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "keep-branch")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "keep-branch", "--delete-branch", "false")

	assertWorktreeNotExists(t, repo, "keep-branch")
	assertBranchExists(t, repo, "keep-branch")
}

func TestDelete_ErrorNonexistent(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "nonexistent")
	if r.ExitCode == 0 {
		t.Error("expected failure for nonexistent worktree")
	}
	assertContains(t, r.Stderr, "not found")
}

func TestDelete_ErrorNoWorktreeSpecifiedNonInteractive(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "delete", "--non-interactive")
	if r.ExitCode == 0 {
		t.Error("expected failure when no worktree specified in non-interactive mode")
	}
	assertContains(t, r.Stderr, "No worktree specified")
}

func TestDelete_StashFlag(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "stash-test")

	// Create uncommitted changes in the worktree
	os.WriteFile(filepath.Join(wtPath, "dirty-file.txt"), []byte("uncommitted"), 0644)
	gitRun(t, wtPath, "add", "dirty-file.txt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "stash-test", "--stash")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Stashing changes")
	assertContains(t, combined, "Deleted worktree")

	// Verify stash exists
	stashOut := gitRun(t, repo, "stash", "list")
	assertContains(t, stashOut, "wt-delete")
}

func TestDelete_DiscardsInNonInteractive(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "discard-test")

	os.WriteFile(filepath.Join(wtPath, "dirty-file.txt"), []byte("will-be-discarded"), 0644)

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "discard-test")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")
	assertWorktreeNotExists(t, repo, "discard-test")
}

func TestDelete_All(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "wt-all-1")
	createWorktreeViaWt(t, repo, "wt-all-2")
	createWorktreeViaWt(t, repo, "wt-all-3")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-all")

	assertWorktreeNotExists(t, repo, "wt-all-1")
	assertWorktreeNotExists(t, repo, "wt-all-2")
	assertWorktreeNotExists(t, repo, "wt-all-3")
}

func TestDelete_AllNoWorktrees(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-all")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "No worktrees found")
}

func TestDelete_AllCleansBranches(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "all-branch-1")
	createWorktreeViaWt(t, repo, "all-branch-2")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-all", "--delete-branch", "true")

	assertBranchNotExists(t, repo, "all-branch-1")
	assertBranchNotExists(t, repo, "all-branch-2")
}

func TestDelete_DirectoryRemoved(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "dir-check")

	assertDirExists(t, wtPath)

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "dir-check")

	assertDirNotExists(t, wtPath)
}

func TestDelete_NotInListAfterDeletion(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "list-check")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "list-check")

	r := runWtSuccess(t, repo, nil, "list")
	combined := r.Stdout + r.Stderr
	assertNotContains(t, combined, "list-check")
}

func TestDelete_ErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	r := runWt(t, dir, nil, "delete")
	if r.ExitCode == 0 {
		t.Error("expected failure outside git repo")
	}
	assertContains(t, r.Stderr, "Not a git repository")
}

func TestDelete_UnpushedCommitsNonInteractive(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "unpushed-test")

	// Create unpushed commits
	os.WriteFile(filepath.Join(wtPath, "new.txt"), []byte("change"), 0644)
	gitRun(t, wtPath, "add", ".")
	gitRun(t, wtPath, "commit", "-q", "-m", "unpushed")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "unpushed-test")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")
}

func TestDelete_CreateThenDeleteWithAllOptions(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "full-delete-test")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive",
		"--worktree-name", "full-delete-test",
		"--delete-branch", "true",
		"--delete-remote", "true")

	assertWorktreeNotExists(t, repo, "full-delete-test")
	assertBranchNotExists(t, repo, "full-delete-test")
}

func TestDelete_LifecycleStashAndCleanup(t *testing.T) {
	repo := createTestRepo(t)
	wtPath := createWorktreeViaWt(t, repo, "lifecycle-test")

	// Make uncommitted changes
	os.WriteFile(filepath.Join(wtPath, "work.txt"), []byte("important work"), 0644)
	gitRun(t, wtPath, "add", "work.txt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "lifecycle-test", "--stash")
	_ = r

	assertWorktreeNotExists(t, repo, "lifecycle-test")

	stashOut := gitRun(t, repo, "stash", "list")
	if !strings.Contains(stashOut, "lifecycle-test") {
		t.Error("expected stash to contain lifecycle-test reference")
	}
}

// ---------- Multi-Delete Tests ----------

func TestDelete_MultipleByPositionalArgs(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "multi-a")
	createWorktreeViaWt(t, repo, "multi-b")
	createWorktreeViaWt(t, repo, "multi-c")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "multi-a", "multi-b")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")

	assertWorktreeNotExists(t, repo, "multi-a")
	assertWorktreeNotExists(t, repo, "multi-b")
	assertWorktreeExists(t, repo, "multi-c")
}

func TestDelete_MultipleFailFastOnInvalidName(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "valid-one")

	r := runWt(t, repo, nil, "delete", "--non-interactive", "valid-one", "typo-name")
	if r.ExitCode == 0 {
		t.Error("expected failure when one name is invalid")
	}
	assertContains(t, r.Stderr, "Worktree 'typo-name' not found")
	// valid-one must NOT have been deleted (fail-fast)
	assertWorktreeExists(t, repo, "valid-one")
}

func TestDelete_MultipleDeduplication(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "dedup-wt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "dedup-wt", "dedup-wt")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")

	assertWorktreeNotExists(t, repo, "dedup-wt")
}

func TestDelete_MultipleBranchCleanup(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "bc-alpha")
	createWorktreeViaWt(t, repo, "bc-bravo")

	assertBranchExists(t, repo, "bc-alpha")
	assertBranchExists(t, repo, "bc-bravo")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-branch", "true", "bc-alpha", "bc-bravo")

	assertWorktreeNotExists(t, repo, "bc-alpha")
	assertWorktreeNotExists(t, repo, "bc-bravo")
	assertBranchNotExists(t, repo, "bc-alpha")
	assertBranchNotExists(t, repo, "bc-bravo")
}

func TestDelete_MixPositionalAndFlagError(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "mix-alpha")
	createWorktreeViaWt(t, repo, "mix-bravo")

	r := runWt(t, repo, nil, "delete", "--non-interactive", "mix-alpha", "--worktree-name", "mix-bravo")
	if r.ExitCode == 0 {
		t.Error("expected failure when mixing positional args and --worktree-name")
	}
	assertContains(t, r.Stderr, "Cannot mix positional arguments and --worktree-name")
}

func TestDelete_SinglePositionalArg(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "single-pos")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "single-pos")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")

	assertWorktreeNotExists(t, repo, "single-pos")
}

func TestDelete_DeprecatedFlagStillWorks(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "deprecated-wt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--worktree-name", "deprecated-wt")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Deleted worktree")
	assertContains(t, r.Stderr, "deprecated")

	assertWorktreeNotExists(t, repo, "deprecated-wt")
}

func TestDelete_MultipleAllNamesInvalid(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "delete", "--non-interactive", "foo", "bar")
	if r.ExitCode == 0 {
		t.Error("expected failure when all names are invalid")
	}
	assertContains(t, r.Stderr, "Worktree 'foo' not found")
	assertContains(t, r.Stderr, "Worktree 'bar' not found")
}

func TestDelete_MultipleBranchPreservation(t *testing.T) {
	repo := createTestRepo(t)
	createWorktreeViaWt(t, repo, "bp-alpha")
	createWorktreeViaWt(t, repo, "bp-bravo")

	assertBranchExists(t, repo, "bp-alpha")
	assertBranchExists(t, repo, "bp-bravo")

	runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--delete-branch", "false", "bp-alpha", "bp-bravo")

	assertWorktreeNotExists(t, repo, "bp-alpha")
	assertWorktreeNotExists(t, repo, "bp-bravo")
	// Branches should still exist
	assertBranchExists(t, repo, "bp-alpha")
	assertBranchExists(t, repo, "bp-bravo")
}

func TestDelete_MultipleWithStash(t *testing.T) {
	repo := createTestRepo(t)
	wtPathA := createWorktreeViaWt(t, repo, "stash-alpha")
	wtPathB := createWorktreeViaWt(t, repo, "stash-bravo")

	// Create uncommitted changes in both worktrees
	os.WriteFile(filepath.Join(wtPathA, "dirty-a.txt"), []byte("alpha changes"), 0644)
	gitRun(t, wtPathA, "add", "dirty-a.txt")
	os.WriteFile(filepath.Join(wtPathB, "dirty-b.txt"), []byte("bravo changes"), 0644)
	gitRun(t, wtPathB, "add", "dirty-b.txt")

	r := runWtSuccess(t, repo, nil, "delete", "--non-interactive", "--stash", "stash-alpha", "stash-bravo")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "Stashing changes")

	assertWorktreeNotExists(t, repo, "stash-alpha")
	assertWorktreeNotExists(t, repo, "stash-bravo")

	// Verify stashes exist
	stashOut := gitRun(t, repo, "stash", "list")
	assertContains(t, stashOut, "stash-alpha")
	assertContains(t, stashOut, "stash-bravo")
}
