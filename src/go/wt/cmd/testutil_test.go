package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// wtBinary is the path to the compiled wt binary, set by TestMain.
var wtBinary string

func TestMain(m *testing.M) {
	// Build the wt binary into a temp directory
	tmpDir, err := os.MkdirTemp("", "wt-test-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create temp dir: %v\n", err)
		os.Exit(1)
	}

	wtBinary = filepath.Join(tmpDir, "wt")
	buildCmd := exec.Command("go", "build", "-o", wtBinary, ".")
	buildCmd.Dir = filepath.Join(mustGetModuleRoot(), "cmd")
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot build wt binary: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func mustGetModuleRoot() string {
	// We know we're in src/go/wt/cmd/ during tests
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Walk up to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("cannot find go.mod")
		}
		dir = parent
	}
}

// ---------- Test Repository Management ----------

// createTestRepo creates a temporary git repo with an initial commit on "main"
// and a bare remote "origin". Returns the resolved path to the repo.
// Cleanup is handled automatically by t.TempDir().
func createTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Resolve symlinks (macOS /tmp -> /private/tmp)
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = resolved
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test User",
			"GIT_COMMITTER_NAME=Test User",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("git", "init", "-q")
	run("git", "config", "user.name", "Test User")
	run("git", "config", "user.email", "test@example.com")

	// Initial commit
	readme := filepath.Join(resolved, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatalf("WriteFile README: %v", err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-q", "-m", "Initial commit")

	// Ensure branch is named "main"
	branchOut, _ := exec.Command("git", "-C", resolved, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if strings.TrimSpace(string(branchOut)) != "main" {
		run("git", "branch", "-m", "main")
	}

	// Set up bare remote inside temp dir
	remoteDir := filepath.Join(resolved, "remote.git")
	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("MkdirAll remote: %v", err)
	}
	remoteInit := exec.Command("git", "init", "-q", "--bare")
	remoteInit.Dir = remoteDir
	if out, err := remoteInit.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	run("git", "remote", "add", "origin", remoteDir)

	return resolved
}

// ---------- wt binary helpers ----------

// wtResult holds the result of running the wt binary.
type wtResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// runWt runs the wt binary with the given args, with cwd set to dir.
// Environment variables can be passed as "KEY=VALUE" strings via env.
func runWt(t *testing.T, dir string, env []string, args ...string) wtResult {
	t.Helper()
	cmd := exec.Command(wtBinary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		// Always set NO_COLOR to simplify output matching
		"NO_COLOR=1",
		// Set WORKTREE_INIT_SCRIPT to a nonexistent command so init is
		// silently skipped. Empty string would fall through to the default
		// "fab sync" which fails in non-fab-managed test repos.
		"WORKTREE_INIT_SCRIPT=__wt_test_noinit__ noop",
	)
	// Append test-provided env vars last so they can override defaults above
	cmd.Env = append(cmd.Env, env...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return wtResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// runWtSuccess runs wt and fails the test if the command does not succeed.
func runWtSuccess(t *testing.T, dir string, env []string, args ...string) wtResult {
	t.Helper()
	r := runWt(t, dir, env, args...)
	if r.ExitCode != 0 {
		t.Fatalf("wt %s failed (exit %d):\nstdout: %s\nstderr: %s",
			strings.Join(args, " "), r.ExitCode, r.Stdout, r.Stderr)
	}
	return r
}

// ---------- Assertion Helpers ----------

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q, got:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", substr, s)
	}
}

func assertExitCode(t *testing.T, r wtResult, code int) {
	t.Helper()
	if r.ExitCode != code {
		t.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
			code, r.ExitCode, r.Stdout, r.Stderr)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		t.Errorf("expected directory to exist: %s", path)
	}
}

func assertDirNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected directory NOT to exist: %s", path)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %s", path)
	}
}

// ---------- Worktree Helpers ----------

// worktreesDir returns the .worktrees directory for a given repo path.
func worktreesDir(repoPath string) string {
	return filepath.Join(filepath.Dir(repoPath), filepath.Base(repoPath)+".worktrees")
}

// worktreePath returns the expected path for a named worktree.
func worktreePath(repoPath, name string) string {
	return filepath.Join(worktreesDir(repoPath), name)
}

// assertWorktreeExists checks that a named worktree directory exists and is registered with git.
func assertWorktreeExists(t *testing.T, repoPath, name string) {
	t.Helper()
	wtPath := worktreePath(repoPath, name)
	assertDirExists(t, wtPath)

	out, err := exec.Command("git", "-C", repoPath, "worktree", "list").CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	if !strings.Contains(string(out), wtPath) {
		t.Errorf("worktree %q not registered in git worktree list:\n%s", name, out)
	}
}

// assertWorktreeNotExists checks that a named worktree does not exist.
func assertWorktreeNotExists(t *testing.T, repoPath, name string) {
	t.Helper()
	wtPath := worktreePath(repoPath, name)
	assertDirNotExists(t, wtPath)
}

// assertBranchExists checks that a local branch exists.
func assertBranchExists(t *testing.T, repoPath, branch string) {
	t.Helper()
	err := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	if err != nil {
		t.Errorf("expected branch %q to exist", branch)
	}
}

// assertBranchNotExists checks that a local branch does not exist.
func assertBranchNotExists(t *testing.T, repoPath, branch string) {
	t.Helper()
	err := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run()
	if err == nil {
		t.Errorf("expected branch %q NOT to exist", branch)
	}
}

// assertGitStateClean runs git fsck and fails if there are issues.
func assertGitStateClean(t *testing.T, repoPath string) {
	t.Helper()
	cmd := exec.Command("git", "-C", repoPath, "fsck", "--no-progress")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("git fsck failed: %v\n%s", err, out)
	}
}

// ---------- Git Helpers ----------

// gitRun runs a git command in the given directory.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s (in %s): %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// createWorktreeViaWt creates a worktree using the wt binary and returns the path.
func createWorktreeViaWt(t *testing.T, repoPath, name string) string {
	t.Helper()
	r := runWtSuccess(t, repoPath, nil, "create", "--non-interactive", "--worktree-name", name, "--worktree-init", "false")
	path := strings.TrimSpace(r.Stdout)
	if path == "" {
		t.Fatalf("wt create returned empty path; stderr: %s", r.Stderr)
	}
	return path
}

// createInitScript creates a test init script in the repo at a test path.
func createInitScript(t *testing.T, repoPath string) {
	t.Helper()
	scriptDir := filepath.Join(repoPath, "scripts")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		t.Fatalf("MkdirAll init script dir: %v", err)
	}
	script := filepath.Join(scriptDir, "worktree-init.sh")
	content := `#!/usr/bin/env bash
echo "Test init script executed"
touch .init-script-ran
`
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatalf("WriteFile init script: %v", err)
	}
}

// pushMainToRemote pushes main to the mock remote.
func pushMainToRemote(t *testing.T, repoPath string) {
	t.Helper()
	gitRun(t, repoPath, "push", "-q", "-u", "origin", "main")
}

// parseJSONList parses JSON output from wt list --json.
func parseJSONList(t *testing.T, jsonStr string) []map[string]interface{} {
	t.Helper()
	var entries []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, jsonStr)
	}
	return entries
}
