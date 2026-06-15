package main

import (
	"bytes"
	"encoding/json"
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
	if cmd.Flags().Lookup("json") == nil {
		t.Error("memoryIndexCmd missing --json flag")
	}
}

// runMemoryIndex executes a fresh memory-index cmd with the given args and
// returns the RunE error plus captured stdout/stderr.
func runMemoryIndex(t *testing.T, args ...string) (error, string, string) {
	t.Helper()
	cmd := memoryIndexCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	err := cmd.Execute()
	return err, out.String(), errBuf.String()
}

func TestMemoryIndexCmd_CheckTier0_CleanReturnsNil(t *testing.T) {
	setupFabRepo(t)
	// Regenerate so the tree is byte-stable, then --check must be clean (nil).
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	if err, _, _ := runMemoryIndex(t, "--check"); err != nil {
		t.Errorf("clean tree --check should return nil (exit 0), got: %v", err)
	}
}

func TestMemoryIndexCmd_CheckTier1_BenignDriftReturnsError(t *testing.T) {
	repo := setupFabRepo(t)
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	// Improve login.md's description: regen would render the new text. The file
	// still has frontmatter and is on disk → benign drift (tier 1), not loss.
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "login.md"),
		"---\ndescription: \"Improved login flow\"\n---\n# Login\n")
	err, _, _ := runMemoryIndex(t, "--check")
	if err == nil {
		t.Error("benign drift --check should return an error (exit 1)")
	}
}

func TestMemoryIndexCmd_CheckJSON_CleanEmitsTier0(t *testing.T) {
	setupFabRepo(t)
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	err, out, _ := runMemoryIndex(t, "--check", "--json")
	if err != nil {
		t.Errorf("clean --check --json should return nil, got: %v", err)
	}
	var report struct {
		Tier   int  `json:"tier"`
		Drift  bool `json:"drift"`
		Losses []struct {
			Category string `json:"category"`
		} `json:"losses"`
	}
	if jerr := json.Unmarshal([]byte(out), &report); jerr != nil {
		t.Fatalf("--json stdout must be a parseable object, got %q (err %v)", out, jerr)
	}
	if report.Tier != 0 || report.Drift {
		t.Errorf("clean tree → {tier:0, drift:false}, got tier=%d drift=%v", report.Tier, report.Drift)
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

func TestMemoryIndexCmd_GeneratesSubDomainIndex(t *testing.T) {
	repo := t.TempDir()
	mustMkdir(t, filepath.Join(repo, "fab"))
	// Domain with a sub-domain holding a depth-3 topic file.
	mustWrite(t, filepath.Join(repo, "docs", "memory", "fab-workflow", "context-loading.md"),
		"---\ndescription: \"Loading\"\n---\n# Context Loading\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "fab-workflow", "index.md"),
		"---\ndescription: \"Fab workflow\"\n---\n# Fab Workflow Documentation\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "fab-workflow", "runtime", "runtime-agents.md"),
		"---\ndescription: \"Agents\"\n---\n# Runtime Agents\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "fab-workflow", "runtime", "index.md"),
		"---\ndescription: \"Runtime tier\"\n---\n# Runtime Documentation\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "index.md"), "# Memory Index\n")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	cmd := memoryIndexCmd()
	cmd.SilenceUsage = true
	cmd.SetArgs(nil)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// The sub-domain index.md is generated with its topic row.
	subIdx, err := os.ReadFile(filepath.Join(repo, "docs", "memory", "fab-workflow", "runtime", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(subIdx), "| [runtime-agents](runtime-agents.md) | Agents |") {
		t.Errorf("sub-domain index missing generated topic row:\n%s", subIdx)
	}

	// The parent domain index references the sub-domain.
	domIdx, err := os.ReadFile(filepath.Join(repo, "docs", "memory", "fab-workflow", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(domIdx), "## Sub-Domains") ||
		!strings.Contains(string(domIdx), "| [runtime](runtime/index.md) | Runtime tier |") {
		t.Errorf("parent domain index missing sub-domain reference:\n%s", domIdx)
	}

	// Idempotent: a second --check run is clean.
	check := memoryIndexCmd()
	check.SilenceUsage = true
	check.SetArgs([]string{"--check"})
	check.SetOut(&bytes.Buffer{})
	check.SetErr(&bytes.Buffer{})
	if err := check.Execute(); err != nil {
		t.Errorf("--check should pass after a nested-tree regen (idempotent), got: %v", err)
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
