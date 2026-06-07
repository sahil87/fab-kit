package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryIndexCmd_RegisteredWithExpectedUse(t *testing.T) {
	cmd := memoryIndexCmd()
	if cmd.Use != "memory-index" {
		t.Errorf("memoryIndexCmd Use = %q, want \"memory-index\"", cmd.Use)
	}
	if cmd.Flags().Lookup("check") == nil {
		t.Error("memoryIndexCmd missing --check flag")
	}
}

// setupFabRepo creates a minimal fab/ + docs/memory/ tree in a temp dir and
// chdirs into it so resolve.FabRoot() resolves. Returns the repo root.
func setupFabRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	mustMkdir(t, filepath.Join(repo, "fab"))
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "login.md"),
		"---\ndescription: \"Login flow\"\n---\n# Login\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "index.md"),
		"---\ndescription: \"Auth domain\"\n---\n# Auth Documentation\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "index.md"), "# Memory Index\n")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	return repo
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	mustMkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryIndexCmd_RegeneratesAndIsIdempotent(t *testing.T) {
	repo := setupFabRepo(t)

	// First run: writes the root + domain index.
	cmd := memoryIndexCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs(nil)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	rootIdx, err := os.ReadFile(filepath.Join(repo, "docs", "memory", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rootIdx), "| [auth](auth/index.md) | Auth domain |") {
		t.Errorf("root index missing generated auth domain row:\n%s", rootIdx)
	}
	if strings.Contains(string(rootIdx), "Memory Files") {
		t.Error("root index must be domains-only (no 'Memory Files' column)")
	}

	domIdx, err := os.ReadFile(filepath.Join(repo, "docs", "memory", "auth", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(domIdx), "| [login](login.md) | Login flow |") {
		t.Errorf("domain index missing generated login row:\n%s", domIdx)
	}

	// Second run: byte-stable → --check passes (exit zero).
	check := memoryIndexCmd()
	check.SilenceUsage = true
	check.SetArgs([]string{"--check"})
	check.SetOut(&bytes.Buffer{})
	check.SetErr(&bytes.Buffer{})
	if err := check.Execute(); err != nil {
		t.Errorf("--check should pass after a fresh regen (idempotent), got: %v", err)
	}
}

func TestMemoryIndexCmd_CheckDetectsDrift(t *testing.T) {
	setupFabRepo(t)
	// Root index is stale (just "# Memory Index\n") and has never been
	// regenerated → --check must fail.
	cmd := memoryIndexCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--check"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil {
		t.Error("--check should fail when indexes are out of date")
	}
}
