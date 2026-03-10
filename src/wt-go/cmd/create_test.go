package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_ExploratoryWorktree(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive")

	// stderr should have the "Created worktree:" message
	assertContains(t, r.Stderr, "Created worktree:")

	// stdout should be exactly one line: the worktree path
	path := strings.TrimSpace(r.Stdout)
	lines := strings.Split(strings.TrimSpace(r.Stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line of stdout, got %d: %q", len(lines), r.Stdout)
	}
	assertDirExists(t, path)
}

func TestCreate_ExploratoryWorktreeRandomName(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive")

	// Name should be adjective-noun format
	path := strings.TrimSpace(r.Stdout)
	name := filepath.Base(path)
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Errorf("expected adjective-noun name, got %q", name)
	}
}

func TestCreate_WorktreeNameFlag(t *testing.T) {
	repo := createTestRepo(t)

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "custom-name")

	assertContains(t, r.Stderr, "custom-name")
	assertWorktreeExists(t, repo, "custom-name")
}

func TestCreate_BranchNameDerivation(t *testing.T) {
	repo := createTestRepo(t)

	// Create a local branch
	gitRun(t, repo, "checkout", "-b", "feature/login")
	gitRun(t, repo, "checkout", "main")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "feature/login")

	// Should derive "login" from "feature/login"
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "login")
}

func TestCreate_ExistingLocalBranch(t *testing.T) {
	repo := createTestRepo(t)

	gitRun(t, repo, "checkout", "-b", "feature/auth")
	gitRun(t, repo, "checkout", "main")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "my-feature", "feature/auth")

	assertContains(t, r.Stderr, "Created worktree: my-feature")
	assertContains(t, r.Stderr, "Branch: feature/auth")
	assertWorktreeExists(t, repo, "my-feature")
}

func TestCreate_RemoteBranch(t *testing.T) {
	repo := createTestRepo(t)

	// Create a branch, push it, then delete locally
	gitRun(t, repo, "checkout", "-b", "remote-feature")
	os.WriteFile(filepath.Join(repo, "remote-file.txt"), []byte("test"), 0644)
	gitRun(t, repo, "add", "remote-file.txt")
	gitRun(t, repo, "commit", "-q", "-m", "remote feature")
	gitRun(t, repo, "push", "-q", "-u", "origin", "remote-feature")
	gitRun(t, repo, "checkout", "main")
	gitRun(t, repo, "branch", "-D", "remote-feature")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "remote-wt", "remote-feature")

	assertContains(t, r.Stderr, "remote-wt")
	assertWorktreeExists(t, repo, "remote-wt")
}

func TestCreate_NewBranch(t *testing.T) {
	repo := createTestRepo(t)

	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "new-branch-wt", "new-feature")

	assertWorktreeExists(t, repo, "new-branch-wt")
	assertBranchExists(t, repo, "new-feature")
}

func TestCreate_NameCollision(t *testing.T) {
	repo := createTestRepo(t)

	createWorktreeViaWt(t, repo, "collision-test")

	r := runWt(t, repo, nil, "create", "--non-interactive", "--worktree-name", "collision-test")
	if r.ExitCode == 0 {
		t.Error("expected failure on name collision")
	}
	assertContains(t, r.Stderr, "already exists")
}

func TestCreate_ReuseExisting(t *testing.T) {
	repo := createTestRepo(t)

	firstPath := createWorktreeViaWt(t, repo, "reuse-test")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--reuse", "--worktree-name", "reuse-test")

	reusePath := strings.TrimSpace(r.Stdout)
	if reusePath != firstPath {
		t.Errorf("--reuse path mismatch: got %q, want %q", reusePath, firstPath)
	}
}

func TestCreate_ReuseCreatesWhenNoCollision(t *testing.T) {
	repo := createTestRepo(t)

	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--reuse", "--worktree-name", "reuse-fresh", "--worktree-init", "false")
	assertWorktreeExists(t, repo, "reuse-fresh")
}

func TestCreate_ReuseRequiresWorktreeName(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "create", "--non-interactive", "--reuse")
	if r.ExitCode == 0 {
		t.Error("expected failure: --reuse without --worktree-name")
	}
	assertContains(t, r.Stderr, "--reuse requires --worktree-name")
}

func TestCreate_ErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	r := runWt(t, dir, nil, "create")
	if r.ExitCode == 0 {
		t.Error("expected failure outside git repo")
	}
	assertContains(t, r.Stderr, "Not a git repository")
}

func TestCreate_InvalidBranchName(t *testing.T) {
	repo := createTestRepo(t)

	r := runWt(t, repo, nil, "create", "--non-interactive", "--worktree-name", "bad-branch", "refs/invalid..name")
	if r.ExitCode == 0 {
		t.Error("expected failure with invalid branch name")
	}
	// No partial worktree directory should be left behind
	assertDirNotExists(t, worktreePath(repo, "bad-branch"))
}

func TestCreate_BranchesOffCurrentBranch(t *testing.T) {
	repo := createTestRepo(t)

	// Create a feature branch with a unique commit
	gitRun(t, repo, "checkout", "-b", "feature/has-marker")
	os.WriteFile(filepath.Join(repo, "marker.txt"), []byte("marker"), 0644)
	gitRun(t, repo, "add", "marker.txt")
	gitRun(t, repo, "commit", "-q", "-m", "Add marker")
	featureCommit := gitRun(t, repo, "rev-parse", "HEAD")

	// Stay on feature branch and create exploratory worktree
	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "from-feature", "--worktree-init", "false")

	wtPath := strings.TrimSpace(r.Stdout)

	// The worktree should have the marker file (branched off feature, not main)
	assertFileExists(t, filepath.Join(wtPath, "marker.txt"))

	// The worktree's HEAD should match the feature commit
	wtCommit := gitRun(t, wtPath, "rev-parse", "HEAD")
	if wtCommit != featureCommit {
		t.Errorf("worktree HEAD %s != feature commit %s", wtCommit, featureCommit)
	}
}

func TestCreate_ExploratoryFromMainStillWorks(t *testing.T) {
	repo := createTestRepo(t)

	mainCommit := gitRun(t, repo, "rev-parse", "HEAD")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "from-main", "--worktree-init", "false")

	wtPath := strings.TrimSpace(r.Stdout)
	wtCommit := gitRun(t, wtPath, "rev-parse", "HEAD")
	if wtCommit != mainCommit {
		t.Errorf("worktree HEAD %s != main commit %s", wtCommit, mainCommit)
	}
}

func TestCreate_ExistingBranchUnaffectedByCurrentBranch(t *testing.T) {
	repo := createTestRepo(t)

	// Create branch-a with unique content
	gitRun(t, repo, "checkout", "-b", "branch-a")
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("branch-a content"), 0644)
	gitRun(t, repo, "add", "a.txt")
	gitRun(t, repo, "commit", "-q", "-m", "Add a.txt")
	gitRun(t, repo, "checkout", "main")

	// Create branch-b with different content
	gitRun(t, repo, "checkout", "-b", "branch-b")
	os.WriteFile(filepath.Join(repo, "b.txt"), []byte("branch-b content"), 0644)
	gitRun(t, repo, "add", "b.txt")
	gitRun(t, repo, "commit", "-q", "-m", "Add b.txt")

	// While on branch-b, check out branch-a into a worktree
	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "checkout-a", "branch-a")

	wtPath := strings.TrimSpace(r.Stdout)
	assertFileExists(t, filepath.Join(wtPath, "a.txt"))
	if _, err := os.Stat(filepath.Join(wtPath, "b.txt")); err == nil {
		t.Error("worktree should not have b.txt (it's from branch-b)")
	}
}

func TestCreate_CorrectDirectoryStructure(t *testing.T) {
	repo := createTestRepo(t)

	runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "test-structure", "--worktree-init", "false")

	expected := worktreePath(repo, "test-structure")
	assertDirExists(t, expected)
}

func TestCreate_PorcelainStdoutOnlyPath(t *testing.T) {
	repo := createTestRepo(t)

	cmd := exec.Command(wtBinary, "create", "--non-interactive", "--worktree-name", "porcelain-test", "--worktree-init", "false")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	stdout, err := cmd.Output()
	if err != nil {
		// Ignore exit error if any - we want to check stdout
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("wt create failed (exit %d): %s", exitErr.ExitCode(), exitErr.Stderr)
		}
	}

	// stdout should be exactly one line: the worktree path
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line of stdout, got %d: %q", len(lines), string(stdout))
	}
	assertDirExists(t, strings.TrimSpace(string(stdout)))
}

func TestCreate_InitScriptRuns(t *testing.T) {
	repo := createTestRepo(t)
	createInitScript(t, repo)

	// Commit init script so worktrees see it
	gitRun(t, repo, "add", "fab/.kit/worktree-init.sh")
	gitRun(t, repo, "commit", "-q", "-m", "Add init script")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "init-run-test")

	wtPath := strings.TrimSpace(r.Stdout)
	assertFileExists(t, filepath.Join(wtPath, ".init-script-ran"))
}

func TestCreate_InitScriptSkippedWhenFalse(t *testing.T) {
	repo := createTestRepo(t)
	createInitScript(t, repo)
	gitRun(t, repo, "add", "fab/.kit/worktree-init.sh")
	gitRun(t, repo, "commit", "-q", "-m", "Add init script")

	r := runWtSuccess(t, repo, nil, "create", "--non-interactive", "--worktree-name", "no-init-test", "--worktree-init", "false")

	wtPath := strings.TrimSpace(r.Stdout)
	if _, err := os.Stat(filepath.Join(wtPath, ".init-script-ran")); err == nil {
		t.Error("init script should not have run with --worktree-init false")
	}
}

func TestCreate_ImmediatelyListable(t *testing.T) {
	repo := createTestRepo(t)

	createWorktreeViaWt(t, repo, "immediate-list")

	r := runWtSuccess(t, repo, nil, "list")
	combined := r.Stdout + r.Stderr
	assertContains(t, combined, "immediate-list")
}
