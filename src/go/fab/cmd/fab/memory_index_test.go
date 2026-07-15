package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
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
	if cmd.Flags().Lookup("rebuild") == nil {
		t.Error("memoryIndexCmd missing --rebuild flag")
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

// TestMemoryIndexCmd_CheckMalformed_BlocksOnByteCleanTree pins the loom case:
// a tree whose committed indexes are byte-identical to their regenerated form
// (index drift tier 0) but whose SOURCE frontmatter is malformed must still make
// --check exit non-zero — the malformed detection runs independent of drift.
// (Uses the non-JSON path so the error is capturable; the JSON path os.Exit(1)s.)
func TestMemoryIndexCmd_CheckMalformed_BlocksOnByteCleanTree(t *testing.T) {
	repo := setupFabRepo(t)
	// Regenerate so every index is byte-stable (drift tier 0).
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	// Corrupt a topic file's frontmatter (the loom glued-fence, verbatim shape).
	// This does NOT change the rendered row (the corrupted value already renders
	// verbatim), so the index stays byte-clean — only the source is malformed.
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "login.md"),
		"---\ndescription: \"Login flow\"---")
	// Re-regenerate so the committed index reflects the (unchanged) rendered row,
	// guaranteeing drift tier 0 — the malformed check must fail on its own.
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen over corrupt source failed: %v", err)
	}

	err, _, stderr := runMemoryIndex(t, "--check")
	if err == nil {
		t.Error("malformed frontmatter on a byte-clean tree must make --check fail (exit ≥ 1)")
	}
	if !strings.Contains(stderr, "malformed frontmatter") ||
		!strings.Contains(stderr, "auth/login.md") {
		t.Errorf("--check stderr must enumerate the offending file, got:\n%s", stderr)
	}
}

// TestMemoryIndexCmd_CheckMalformed_BenignDriftErrorNamesMalformed pins that when
// a malformed floor CO-OCCURS with benign drift (tier 1), the RETURNED error text
// (not just the stderr enumeration) names the corruption — so a caller surfacing
// only the error is not misled into treating it as mere staleness.
func TestMemoryIndexCmd_CheckMalformed_BenignDriftErrorNamesMalformed(t *testing.T) {
	repo := setupFabRepo(t)
	// Clean baseline.
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	// Corrupt login.md's frontmatter (glued fence → malformed, renders verbatim so
	// it does NOT drift the index on its own — the loom shape).
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "login.md"),
		"---\ndescription: \"Login flow\"---")
	// Separately change the auth domain description (valid) WITHOUT regenerating, so
	// the committed root row is stale → benign drift (tier 1) layered on the floor.
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "index.md"),
		"---\ndescription: \"Auth domain (revised)\"\n---\n# Auth Documentation\n")

	err, _, stderr := runMemoryIndex(t, "--check")
	if err == nil {
		t.Fatal("benign drift + malformed --check must return an error")
	}
	if !strings.Contains(err.Error(), "malformed frontmatter") {
		t.Errorf("returned error must name malformed frontmatter, got: %v", err)
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Errorf("returned error should still name the drift, got: %v", err)
	}
	// stderr enumeration of the offending file is retained.
	if !strings.Contains(stderr, "auth/login.md") {
		t.Errorf("stderr must still enumerate the malformed file, got:\n%s", stderr)
	}
}

// TestMemoryIndexCmd_CheckMalformed_JSONHasMalformedArray confirms the additive
// `malformed` array is present and parseable, and that the existing
// tier/drift/losses keys are unchanged, on a CLEAN tree (which returns nil, so
// no os.Exit — the malformed-populated JSON path exits and cannot be captured).
func TestMemoryIndexCmd_CheckMalformed_JSONHasMalformedArray(t *testing.T) {
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
		Malformed []struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		} `json:"malformed"`
	}
	if jerr := json.Unmarshal([]byte(out), &report); jerr != nil {
		t.Fatalf("--json stdout must be a parseable object, got %q (err %v)", out, jerr)
	}
	if report.Tier != 0 || report.Drift {
		t.Errorf("clean tree → {tier:0, drift:false}, got tier=%d drift=%v", report.Tier, report.Drift)
	}
	// `malformed` must be present as an empty array (not null / absent).
	if report.Malformed == nil {
		t.Error("--json report must carry a non-null `malformed` array")
	}
	if len(report.Malformed) != 0 {
		t.Errorf("clean tree → empty malformed array, got %+v", report.Malformed)
	}
	// The `malformed` key must literally appear in the JSON (non-null contract).
	if !strings.Contains(out, "\"malformed\"") {
		t.Errorf("--json must include the `malformed` key, got:\n%s", out)
	}
}

// TestMemoryIndexCmd_CheckOverLength_DoesNotBlock pins the asymmetry: an
// over-length `description:` on an otherwise-clean, malformed-free tree is
// advisory only — --check exits 0 and the length warning prints to stderr.
func TestMemoryIndexCmd_CheckOverLength_DoesNotBlock(t *testing.T) {
	repo := setupFabRepo(t)
	// Replace login.md's description with a 600-char one-liner (over the 500 cap).
	long := strings.Repeat("x", 600)
	mustWrite(t, filepath.Join(repo, "docs", "memory", "auth", "login.md"),
		"---\ndescription: \""+long+"\"\n---\n# Login\n")
	// Regenerate so the (long-but-valid) description is reflected → drift tier 0.
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	err, _, stderr := runMemoryIndex(t, "--check")
	if err != nil {
		t.Errorf("over-length (advisory) alone must NOT fail --check, got: %v", err)
	}
	if !strings.Contains(stderr, "description:") || !strings.Contains(stderr, "soft cap") {
		t.Errorf("--check stderr should carry the advisory length warning, got:\n%s", stderr)
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

// TestMemoryIndexCmd_RebuildProbeInHelp (TC11) pins the contract the 2.5.5→2.6.0
// re-baseline migration's pre-check relies on: the migration probes
// `fab memory-index --help` for the `--rebuild` flag and ABORTS (rewriting nothing)
// when it is absent — i.e. against an old binary that predates the flag. This test
// confirms the probe is reliable on a 2.6.0 binary: the rendered help text exposes
// `--rebuild`, so the probe succeeds here and would fail (→ abort) on any binary
// lacking the flag. Spawning a real old binary in CI is infeasible, so the probe's
// positive case + the migration's documented abort path are the TC11 realization.
func TestMemoryIndexCmd_RebuildProbeInHelp(t *testing.T) {
	cmd := memoryIndexCmd()
	help := cmd.UsageString()
	if !strings.Contains(help, "--rebuild") {
		t.Errorf("the migration pre-check probes `fab memory-index --help` for --rebuild; "+
			"it must appear in the usage text, got:\n%s", help)
	}
	// And the flag must be a real registered bool (an old binary errors on it).
	if f := cmd.Flags().Lookup("rebuild"); f == nil || f.Value.Type() != "bool" {
		t.Errorf("--rebuild must be a registered bool flag, got %+v", f)
	}
}

// gitRun runs a git command in dir with a deterministic identity, skipping the
// test when git is unavailable (mirrors the package-level gitDateRun helper).
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git %v failed (git unavailable?): %v\n%s", args, err, out)
	}
}

// setupGitFabRepo builds a git-backed fab repo with one attributable memory
// commit, chdirs in, and returns the repo root. Used by the freeze-on-write
// end-to-end --check / --rebuild cmd tests (log.md targets need real git history).
func setupGitFabRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitRun(t, repo, "init")
	mustWrite(t, filepath.Join(repo, "fab", "changes", "260401-bbbb-live", ".status.yaml"),
		"id: bbbb\nname: 260401-bbbb-live\nsummary: \"the live change\"\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "d", "topic.md"),
		"---\ndescription: \"a topic\"\n---\n# Topic\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "d", "index.md"),
		"---\ndescription: \"d domain\"\n---\n# D Documentation\n")
	mustWrite(t, filepath.Join(repo, "docs", "memory", "index.md"), "# Memory Index\n")
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-live",
		"--date", "2026-04-01T12:00:00 +0000")

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

// TestMemoryIndexCmd_FreezeOnWrite_CheckSupersetPasses (R7, end-to-end) confirms
// the cmd's --check exits 0 (returns nil) when the committed log.md is a valid
// SUPERSET of the freeze-on-write merge — a frozen squash-stale line the live
// history no longer shows must NOT make --check fail.
func TestMemoryIndexCmd_FreezeOnWrite_CheckSupersetPasses(t *testing.T) {
	repo := setupGitFabRepo(t)
	// Generate the baseline (writes index + log).
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("baseline regen failed: %v", err)
	}
	// Append a frozen squash-stale line to the committed log (a superset).
	logPath := filepath.Join(repo, "docs", "memory", "d", "log.md")
	existing, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	superset := string(existing) +
		"\n## 2026-01-01\n- **Update** [topic](/d/topic.md) — squash-stale frozen history (cccc)\n"
	if err := os.WriteFile(logPath, []byte(superset), 0o644); err != nil {
		t.Fatal(err)
	}
	// --check must still pass (the merge reproduces the superset byte-for-byte).
	if err, _, _ := runMemoryIndex(t, "--check"); err != nil {
		t.Errorf("R7: --check on a valid superset log.md should pass (exit 0), got: %v", err)
	}
	// A plain regen preserves the frozen line (freeze-on-write, not a rewrite).
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("regen failed: %v", err)
	}
	after, _ := os.ReadFile(logPath)
	if !strings.Contains(string(after), "squash-stale frozen history (cccc)") {
		t.Errorf("R1: a plain regen must preserve the frozen line, got:\n%s", after)
	}
}

// TestMemoryIndexCmd_Rebuild_DropsStaleLine (R6, end-to-end) confirms the cmd's
// --rebuild discards the frozen state and re-projects from current git, dropping a
// now-unreachable frozen line.
func TestMemoryIndexCmd_Rebuild_DropsStaleLine(t *testing.T) {
	repo := setupGitFabRepo(t)
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatalf("baseline regen failed: %v", err)
	}
	logPath := filepath.Join(repo, "docs", "memory", "d", "log.md")
	existing, _ := os.ReadFile(logPath)
	stale := string(existing) +
		"\n## 2026-01-01\n- **Update** [topic](/d/topic.md) — squash-stale frozen history (cccc)\n"
	if err := os.WriteFile(logPath, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	// --rebuild re-projects, dropping the unreachable cccc line.
	if err, _, _ := runMemoryIndex(t, "--rebuild"); err != nil {
		t.Fatalf("--rebuild failed: %v", err)
	}
	after, _ := os.ReadFile(logPath)
	if strings.Contains(string(after), "squash-stale frozen history (cccc)") {
		t.Errorf("R6: --rebuild must drop the unreachable frozen line, got:\n%s", after)
	}
	if !strings.Contains(string(after), "the live change (bbbb)") {
		t.Errorf("R6: --rebuild must re-project the live entry, got:\n%s", after)
	}
}
