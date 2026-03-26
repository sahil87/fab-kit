package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the idea binary to a temp location for integration tests.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "idea")
	cmd := exec.Command("go", "build", "-o", bin, "./")
	cmd.Dir = filepath.Join(findModuleRoot(t), "cmd")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	// Walk up from cmd/ to find go.mod
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod")
		}
		dir = parent
	}
}

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	backlogDir := filepath.Join(dir, "fab")
	if err := os.MkdirAll(backlogDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlogDir, "backlog.md"), []byte("# Backlog\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func TestBareShorthand_AddsIdea(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "refactor auth middleware")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bare shorthand failed: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "Added: [") {
		t.Errorf("expected 'Added: [' prefix, got: %s", out)
	}
	if !strings.Contains(string(out), "refactor auth middleware") {
		t.Errorf("expected idea text in output, got: %s", out)
	}
}

func TestBareShorthand_EmptyTextErrors(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !strings.Contains(string(out), "text is required") {
		t.Errorf("expected empty text error, got: %s", out)
	}
}

func TestBareShorthand_MultipleArgsJoined(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "refactor", "auth", "middleware")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("multi-arg shorthand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "refactor auth middleware") {
		t.Errorf("expected joined text in output, got: %s", out)
	}
}

func TestSubcommand_AddStillWorks(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin, "add", "via add subcommand")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("add subcommand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "via add subcommand") {
		t.Errorf("expected idea text in output, got: %s", out)
	}
}

func TestSubcommand_ListStillWorks(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	// Add an idea first
	addCmd := exec.Command(bin, "add", "test idea")
	addCmd.Dir = repo
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("add failed: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "list")
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list subcommand failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "test idea") {
		t.Errorf("expected idea in list output, got: %s", out)
	}
}

func TestNoArgs_ShowsHelp(t *testing.T) {
	bin := buildBinary(t)
	repo := setupGitRepo(t)

	cmd := exec.Command(bin)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("no-args failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "idea [text]") && !strings.Contains(string(out), "Backlog idea management") {
		t.Errorf("expected help output, got: %s", out)
	}
}
