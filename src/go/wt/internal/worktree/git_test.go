package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupGitRepo creates a temporary git repository for testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}

	// Create initial commit
	testFile := filepath.Join(dir, "README.md")
	os.WriteFile(testFile, []byte("# Test\n"), 0644)
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("initial commit: %s: %v", out, err)
	}

	return dir
}

func TestHasUncommittedChanges_Clean(t *testing.T) {
	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if HasUncommittedChanges() {
		t.Error("HasUncommittedChanges() = true, want false for clean repo")
	}
}

func TestHasUncommittedChanges_Modified(t *testing.T) {
	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Modify a tracked file
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0644)

	if !HasUncommittedChanges() {
		t.Error("HasUncommittedChanges() = false, want true for modified repo")
	}
}

func TestHasUntrackedFiles_Clean(t *testing.T) {
	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if HasUntrackedFiles() {
		t.Error("HasUntrackedFiles() = true, want false for clean repo")
	}
}

func TestHasUntrackedFiles_WithNew(t *testing.T) {
	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create an untracked file
	os.WriteFile(filepath.Join(dir, "new-file.txt"), []byte("new\n"), 0644)

	if !HasUntrackedFiles() {
		t.Error("HasUntrackedFiles() = false, want true with untracked file")
	}
}

func TestBranchExistsLocally(t *testing.T) {
	dir := setupGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if !BranchExistsLocally("main") {
		t.Error("BranchExistsLocally('main') = false, want true")
	}

	if BranchExistsLocally("nonexistent-branch") {
		t.Error("BranchExistsLocally('nonexistent-branch') = true, want false")
	}
}
