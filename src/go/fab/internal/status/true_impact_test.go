package status

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// setupGitRepo builds a tiny git repo with two commits, an origin/main ref,
// and a fab/project/config.yaml — returning the repo root.
func setupGitRepo(t *testing.T, withExclude bool) (repoRoot, fabRoot string) {
	t.Helper()
	repoRoot = t.TempDir()
	fabRoot = filepath.Join(repoRoot, "fab")

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}

	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	run("git", "config", "commit.gpgsign", "false")

	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	configContent := "stage_hooks: {}\n"
	if withExclude {
		configContent = "true_impact_exclude:\n  - docs/\n"
	}
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(configContent), 0o644)

	os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\n"), 0o644)
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "initial")

	// Simulate origin/main pointing at the same commit (fresh clone).
	run("git", "update-ref", "refs/remotes/origin/main", "HEAD")

	// Diverge — modify root file, add docs/.
	os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\nworld\nfoo\n"), 0o644)
	os.MkdirAll(filepath.Join(repoRoot, "docs"), 0o755)
	os.WriteFile(filepath.Join(repoRoot, "docs", "b.md"), []byte("doc\nlines\n"), 0o644)
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "second")

	return repoRoot, fabRoot
}

// withCwd runs fn with cwd set to dir, restoring on exit. Tests using this
// MUST NOT be run in parallel — they mutate process-wide cwd.
func withCwd(t *testing.T, dir string, fn func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	}()
	fn()
}

func minimalStatusYAML() string {
	return `id: te1t
name: 260305-test-1-sample
created: "2026-03-05T12:00:00Z"
created_by: test
change_type: feat
issues: []
progress:
  intake: done
  apply: active
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: true
  task_count: 1
  acceptance_count: 1
  acceptance_completed: 0
confidence:
  certain: 1
  confident: 0
  tentative: 0
  unresolved: 0
  score: 5.0
stage_metrics: {}
prs: []
last_updated: "2026-03-05T12:00:00Z"
`
}

func TestWriteTrueImpact_ApplyWritesBlock(t *testing.T) {
	repoRoot, fabRoot := setupGitRepo(t, false)

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	withCwd(t, repoRoot, func() {
		if err := WriteTrueImpact(statusFile, statusPath, fabRoot, "apply"); err != nil {
			t.Fatalf("WriteTrueImpact: %v", err)
		}
	})

	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.TrueImpact == nil {
		t.Fatal("expected true_impact block, got nil")
	}
	if reloaded.TrueImpact.ComputedAtStage != "apply" {
		t.Errorf("computed_at_stage = %q, want apply", reloaded.TrueImpact.ComputedAtStage)
	}
	if reloaded.TrueImpact.Added <= 0 {
		t.Errorf("expected non-zero Added, got %d", reloaded.TrueImpact.Added)
	}
	if reloaded.TrueImpact.Excluding != nil {
		t.Errorf("expected no Excluding (empty exclude list), got %+v", reloaded.TrueImpact.Excluding)
	}
}

func TestWriteTrueImpact_HydrateOverwritesBlock(t *testing.T) {
	repoRoot, fabRoot := setupGitRepo(t, false)

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	withCwd(t, repoRoot, func() {
		_ = WriteTrueImpact(statusFile, statusPath, fabRoot, "apply")

		reloaded, _ := sf.Load(statusPath)
		_ = WriteTrueImpact(reloaded, statusPath, fabRoot, "hydrate")
	})

	final, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if final.TrueImpact == nil {
		t.Fatal("expected true_impact block")
	}
	if final.TrueImpact.ComputedAtStage != "hydrate" {
		t.Errorf("computed_at_stage = %q, want hydrate", final.TrueImpact.ComputedAtStage)
	}
}

// TestWriteTrueImpact_ShipRecomputesAfterCommit reproduces the h65d timing
// bug: in the standard pipeline nothing is committed until /git-pr (ship), so
// apply-finish and hydrate-finish run when HEAD == merge-base and the
// three-dot diff is empty (0/0/0). The ship-finish recompute — run after
// /git-pr commits and pushes the branch — is the authoritative write and must
// supersede the earlier zeros with the real PR-diff counts and
// computed_at_stage: ship.
func TestWriteTrueImpact_ShipRecomputesAfterCommit(t *testing.T) {
	// Fresh repo whose HEAD == origin/main (no divergent commit yet) — the
	// clean-tree state the standard pipeline sits in during apply/hydrate.
	repoRoot := t.TempDir()
	fabRoot := filepath.Join(repoRoot, "fab")

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	run("git", "config", "commit.gpgsign", "false")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("stage_hooks: {}\n"), 0o644)
	os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\n"), 0o644)
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "initial")
	// origin/main == HEAD: the branch has no commits of its own yet.
	run("git", "update-ref", "refs/remotes/origin/main", "HEAD")

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	withCwd(t, repoRoot, func() {
		// apply-finish and hydrate-finish run while HEAD == merge-base: the
		// three-dot diff is empty, so the block records zeros.
		sfp, _ := sf.Load(statusPath)
		_ = WriteTrueImpact(sfp, statusPath, fabRoot, "apply")
		reloaded, _ := sf.Load(statusPath)
		_ = WriteTrueImpact(reloaded, statusPath, fabRoot, "hydrate")

		afterHydrate, _ := sf.Load(statusPath)
		if afterHydrate.TrueImpact == nil {
			t.Fatal("expected a true_impact block after hydrate")
		}
		if afterHydrate.TrueImpact.ComputedAtStage != "hydrate" {
			t.Errorf("computed_at_stage = %q, want hydrate", afterHydrate.TrueImpact.ComputedAtStage)
		}
		if afterHydrate.TrueImpact.Added != 0 || afterHydrate.TrueImpact.Deleted != 0 {
			t.Errorf("expected zeros at apply/hydrate (HEAD == merge-base), got added=%d deleted=%d",
				afterHydrate.TrueImpact.Added, afterHydrate.TrueImpact.Deleted)
		}

		// /git-pr commits + pushes: the branch tip now diverges from
		// origin/main. Simulate that commit.
		os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\nworld\nfoo\nbar\n"), 0o644)
		run("git", "add", "-A")
		run("git", "commit", "-q", "-m", "feature work")

		// ship-finish recompute: measures the real base...HEAD diff.
		afterHydrate2, _ := sf.Load(statusPath)
		if err := WriteTrueImpact(afterHydrate2, statusPath, fabRoot, "ship"); err != nil {
			t.Fatalf("WriteTrueImpact(ship): %v", err)
		}
	})

	final, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if final.TrueImpact == nil {
		t.Fatal("expected true_impact block after ship")
	}
	if final.TrueImpact.ComputedAtStage != "ship" {
		t.Errorf("computed_at_stage = %q, want ship (should supersede hydrate)", final.TrueImpact.ComputedAtStage)
	}
	if final.TrueImpact.Added <= 0 {
		t.Errorf("expected non-zero Added at ship (branch tip exists), got %d", final.TrueImpact.Added)
	}
}

func TestWriteTrueImpact_NoMergeBaseLeavesUntouched(t *testing.T) {
	// Set up a repo WITHOUT origin/main so merge-base resolution fails.
	repoRoot := t.TempDir()
	fabRoot := filepath.Join(repoRoot, "fab")

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	run("git", "config", "commit.gpgsign", "false")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("stage_hooks: {}\n"), 0o644)
	os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hi\n"), 0o644)
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "only commit")

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	withCwd(t, repoRoot, func() {
		if err := WriteTrueImpact(statusFile, statusPath, fabRoot, "apply"); err != nil {
			t.Errorf("WriteTrueImpact should be best-effort, got error: %v", err)
		}
	})

	reloaded, _ := sf.Load(statusPath)
	if reloaded.TrueImpact != nil {
		t.Errorf("expected no true_impact (merge-base failed), got %+v", reloaded.TrueImpact)
	}
}

func TestWriteTrueImpact_WithExcludeEmitsExcluding(t *testing.T) {
	repoRoot, fabRoot := setupGitRepo(t, true)

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	withCwd(t, repoRoot, func() {
		_ = WriteTrueImpact(statusFile, statusPath, fabRoot, "apply")
	})

	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.TrueImpact == nil {
		t.Fatal("expected true_impact block")
	}
	if reloaded.TrueImpact.Excluding == nil {
		t.Fatal("expected Excluding block when true_impact_exclude is non-empty")
	}
	if reloaded.TrueImpact.Excluding.Added > reloaded.TrueImpact.Added {
		t.Errorf("excluding.added (%d) should not exceed raw.added (%d)",
			reloaded.TrueImpact.Excluding.Added, reloaded.TrueImpact.Added)
	}
	// Sanity-check serialized form
	data, _ := os.ReadFile(statusPath)
	if !strings.Contains(string(data), "true_impact:") {
		t.Errorf(".status.yaml missing true_impact: block:\n%s", data)
	}
}

func TestWriteTrueImpact_NonApplyStageIsNoOp(t *testing.T) {
	repoRoot, fabRoot := setupGitRepo(t, false)

	statusPath := filepath.Join(t.TempDir(), ".status.yaml")
	os.WriteFile(statusPath, []byte(minimalStatusYAML()), 0o644)

	statusFile, err := sf.Load(statusPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	withCwd(t, repoRoot, func() {
		_ = WriteTrueImpact(statusFile, statusPath, fabRoot, "review")
	})

	reloaded, _ := sf.Load(statusPath)
	if reloaded.TrueImpact != nil {
		t.Errorf("expected no-op for stage=review, got %+v", reloaded.TrueImpact)
	}
}
