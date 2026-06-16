package memoryindex

// Freeze-on-write log.md generation tests (change 260616-tayp). These lock the
// intake §8 test matrix rows that concern generation: TC1 (idempotence), TC2
// (append on new change-id), TC3 (no-op on squashed-but-attributable), TC4
// (freeze of unattributable lines), TC6 (--rebuild re-projects), TC10 (loom
// regression — 0 churn), TC12 (seed-merge preserved). TC5 (parseLog round-trip)
// lives in log_test.go; the --check tiers (TC7–TC9, R10) live in loss_test.go.
//
// All git history is SYNTHESIZED with the gitDateRun / writeFile helpers
// (memoryindex_test.go) — no live loom dependency in CI (TC10 requirement).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// logTargetFor returns the LogTarget whose path ends in the given docs/memory-
// relative suffix (e.g. "docs/memory/d/log.md"), or nil. A small reader-helper so
// each test asserts against one folder's log without re-walking the slice inline.
func logTargetFor(targets []LogTarget, suffix string) *LogTarget {
	for i := range targets {
		if strings.HasSuffix(filepath.ToSlash(targets[i].Path), suffix) {
			return &targets[i]
		}
	}
	return nil
}

// writeLog writes content to the folder's log.md (the "freeze" — the committed,
// authoritative on-disk state a later run must read back and preserve).
func writeLog(t *testing.T, target *LogTarget) {
	t.Helper()
	if err := os.WriteFile(target.Path, []byte(target.Content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- TC1: idempotence — second run is a byte-for-byte no-op ----------------

// TestFreeze_TC1_Idempotence runs freeze-on-write twice on the same git state and
// asserts the second run reproduces the on-disk log byte-for-byte (Constitution
// III). The first run bootstraps + freezes; the second reads the frozen log back
// and appends nothing.
func TestFreeze_TC1_Idempotence(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260301-aaaa-first/.status.yaml",
		"id: aaaa\nname: 260301-aaaa-first\nsummary: \"the first change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1 from o/260301-aaaa-first",
		"--date", "2026-03-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatalf("expected d/log.md target, got %+v", first)
	}
	writeLog(t, dLog)

	second, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	dLog2 := logTargetFor(second, "docs/memory/d/log.md")
	if dLog2 == nil {
		t.Fatalf("second run dropped d/log.md target")
	}
	if dLog2.Content != dLog.Content {
		t.Errorf("NOT idempotent: second run differs.\n--- first ---\n%s\n--- second ---\n%s", dLog.Content, dLog2.Content)
	}
}

// --- TC2: append on a new (file, change-id) --------------------------------

// TestFreeze_TC2_AppendOnNewChangeID freezes a log with one attributable entry,
// then a new commit for a NEW (file, change-id) lands. The regen must append
// exactly that one entry and touch no existing line.
func TestFreeze_TC2_AppendOnNewChangeID(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260301-aaaa-first/.status.yaml",
		"id: aaaa\nname: 260301-aaaa-first\nsummary: \"first summary\"\n")
	writeFile(t, repo, "fab/changes/260401-bbbb-second/.status.yaml",
		"id: bbbb\nname: 260401-bbbb-second\nsummary: \"second summary\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1 from o/260301-aaaa-first",
		"--date", "2026-03-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil || !strings.Contains(dLog.Content, "first summary (aaaa)") {
		t.Fatalf("bootstrap should carry the aaaa entry, got %+v", dLog)
	}
	writeLog(t, dLog)
	frozenLineCount := strings.Count(dLog.Content, "\n- ")

	// A second change touches the same file under a new (file, change-id).
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic v2\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-second",
		"--date", "2026-04-01T12:00:00 +0000")

	second, _ := GatherLogs(repo, fabRoot, false)
	dLog2 := logTargetFor(second, "docs/memory/d/log.md")
	if dLog2 == nil {
		t.Fatal("second run dropped the d/log.md target")
	}
	// The new entry is appended exactly once...
	if got := strings.Count(dLog2.Content, "second summary (bbbb)"); got != 1 {
		t.Errorf("expected exactly one bbbb entry appended, got %d:\n%s", got, dLog2.Content)
	}
	// ...the existing aaaa line is preserved verbatim...
	if !strings.Contains(dLog2.Content, "first summary (aaaa)") {
		t.Errorf("the frozen aaaa entry must be preserved, got:\n%s", dLog2.Content)
	}
	// ...and exactly one new bullet was added.
	if got := strings.Count(dLog2.Content, "\n- "); got != frozenLineCount+1 {
		t.Errorf("expected exactly one entry appended (was %d, now %d):\n%s", frozenLineCount, got, dLog2.Content)
	}
}

// --- TC3: no-op on a squashed-but-attributable commit ----------------------

// TestFreeze_TC3_NoOpOnSquashedAttributable freezes a log with change aaaa's
// entry, then rewrites history so aaaa's work collapses into a single squashed
// commit that STILL carries the aaaa token. The (file, change-id) pair is already
// present → no append, byte-stable.
func TestFreeze_TC3_NoOpOnSquashedAttributable(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260301-aaaa-first/.status.yaml",
		"id: aaaa\nname: 260301-aaaa-first\nsummary: \"aaaa summary\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1 from o/260301-aaaa-first",
		"--date", "2026-03-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("expected a d/log.md target at bootstrap")
	}
	writeLog(t, dLog)

	// "Squash": a new commit touches the same file, still carrying the aaaa token
	// in its subject (a squash-merge that preserved the bare change-id). The
	// (topic, aaaa) pair is already frozen → must be a no-op.
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic squashed\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "feat: squashed work (aaaa)",
		"--date", "2026-05-01T12:00:00 +0000")

	second, _ := GatherLogs(repo, fabRoot, false)
	dLog2 := logTargetFor(second, "docs/memory/d/log.md")
	if dLog2 == nil {
		t.Fatal("second run dropped the d/log.md target")
	}
	if dLog2.Content != dLog.Content {
		t.Errorf("squashed-but-attributable commit must be a no-op (aaaa already present).\n--- frozen ---\n%s\n--- regen ---\n%s", dLog.Content, dLog2.Content)
	}
}

// --- TC4: freeze of unattributable lines -----------------------------------

// TestFreeze_TC4_FreezeUnattributable bootstraps a log with an UNATTRIBUTABLE
// entry (a direct-main commit with no registry token), freezes it, then a re-run
// projects a different (squash-reworded) unattributable subject for the same file.
// The frozen line must be unchanged AND the new unattributable commit must NOT be
// projected (the §3 rule — R3).
func TestFreeze_TC4_FreezeUnattributable(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	// No registry entry → the commit is unattributable.
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "docs: tweak topic (#99)",
		"--date", "2026-03-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	// Bootstrap: the unattributable commit IS projected (frozen on first write).
	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("bootstrap should project the unattributable commit into d/log.md")
	}
	if !strings.Contains(dLog.Content, "docs: tweak topic (#99)") {
		t.Fatalf("bootstrap must project the unattributable subject, got:\n%s", dLog.Content)
	}
	writeLog(t, dLog)

	// A NEW unattributable commit (a squash that reworded the subject) lands.
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic reworded\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "docs: REWORDED topic (#100)",
		"--date", "2026-04-01T12:00:00 +0000")

	second, _ := GatherLogs(repo, fabRoot, false)
	dLog2 := logTargetFor(second, "docs/memory/d/log.md")
	if dLog2 == nil {
		t.Fatal("second run dropped the d/log.md target")
	}
	// Frozen line unchanged.
	if !strings.Contains(dLog2.Content, "docs: tweak topic (#99)") {
		t.Errorf("frozen unattributable line must be preserved verbatim, got:\n%s", dLog2.Content)
	}
	// New unattributable commit NOT projected.
	if strings.Contains(dLog2.Content, "REWORDED") {
		t.Errorf("a NEW unattributable commit must NOT be projected after first write, got:\n%s", dLog2.Content)
	}
	// Byte-stable overall (nothing changed).
	if dLog2.Content != dLog.Content {
		t.Errorf("freeze-of-unattributable should be byte-stable.\n--- frozen ---\n%s\n--- regen ---\n%s", dLog.Content, dLog2.Content)
	}
}

// --- TC6: --rebuild re-projects (destructive) ------------------------------

// TestFreeze_TC6_Rebuild freezes a log carrying a line that current git can no
// longer reach (squash-stale), then runs --rebuild and asserts the re-projection
// drops the unreachable line (destructive, as designed — R6).
func TestFreeze_TC6_Rebuild(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260401-bbbb-live/.status.yaml",
		"id: bbbb\nname: 260401-bbbb-live\nsummary: \"the live change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-live",
		"--date", "2026-04-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	// Bootstrap the live entry, then hand-craft a frozen log that ALSO carries a
	// squash-stale line (a change cccc git can no longer reach).
	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("expected a d/log.md target at bootstrap")
	}
	frozen := dLog.Content +
		"\n## 2026-01-01\n- **Update** [topic](/d/topic.md) — squash-stale history (cccc)\n"
	if err := os.WriteFile(dLog.Path, []byte(frozen), 0o644); err != nil {
		t.Fatal(err)
	}

	// A plain regen PRESERVES the stale line (freeze-on-write).
	plain, _ := GatherLogs(repo, fabRoot, false)
	plainLog := logTargetFor(plain, "docs/memory/d/log.md")
	if plainLog == nil || !strings.Contains(plainLog.Content, "squash-stale history (cccc)") {
		t.Fatalf("plain regen must preserve the frozen stale line, got:\n%v", plainLog)
	}

	// --rebuild DISCARDS the frozen state and re-projects from current git only,
	// dropping the now-unreachable cccc line.
	rebuilt, _ := GatherLogs(repo, fabRoot, true)
	rebuiltLog := logTargetFor(rebuilt, "docs/memory/d/log.md")
	if rebuiltLog == nil {
		t.Fatal("--rebuild dropped the d/log.md target")
	}
	if strings.Contains(rebuiltLog.Content, "squash-stale history (cccc)") {
		t.Errorf("--rebuild must drop the unreachable frozen line, got:\n%s", rebuiltLog.Content)
	}
	// The live entry is still re-projected.
	if !strings.Contains(rebuiltLog.Content, "the live change (bbbb)") {
		t.Errorf("--rebuild must re-project the live entry, got:\n%s", rebuiltLog.Content)
	}
}

// --- TC12: seed-merge preserved at bootstrap / --rebuild -------------------

// TestFreeze_TC12_SeedMergePreserved confirms freeze-on-write did not regress the
// log.seed.md seed-merge: a folder with a seed at bootstrap (and under --rebuild)
// still merges the seed entries beneath the git projection (no mergeSeedEntries
// regression — R4).
func TestFreeze_TC12_SeedMergePreserved(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260601-aaaa-recent/.status.yaml",
		"id: aaaa\nname: 260601-aaaa-recent\nsummary: \"the recent change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	writeFile(t, repo, "docs/memory/d/log.seed.md",
		"## 2026-02-09\n- **Creation** [topic](/d/topic.md) — initial pre-FKF creation (h3v7)\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1 from o/260601-aaaa-recent",
		"--date", "2026-06-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	for _, rebuild := range []bool{false, true} {
		targets, err := GatherLogs(repo, fabRoot, rebuild)
		if err != nil {
			t.Fatal(err)
		}
		dLog := logTargetFor(targets, "docs/memory/d/log.md")
		if dLog == nil {
			t.Fatalf("rebuild=%v: expected a d/log.md target", rebuild)
		}
		if !strings.Contains(dLog.Content, "the recent change (aaaa)") {
			t.Errorf("rebuild=%v: git-projected entry missing:\n%s", rebuild, dLog.Content)
		}
		if !strings.Contains(dLog.Content, "initial pre-FKF creation (h3v7)") {
			t.Errorf("rebuild=%v: seed entry must merge beneath the projection:\n%s", rebuild, dLog.Content)
		}
	}
}

// --- TC10: loom regression fixture — 0 churn across the folder set ---------

// TestFreeze_TC10_LoomRegression is the canonical loom regression (memory
// loom-runkit-memory-shape-evidence), built from SYNTHESIZED git history (no live
// loom dependency). It reproduces the exact failure mode: a migration landed as
// two commits (Part 2a / Part 2b) that are later squash-merged into a single
// unattributable commit (#1721), making the pre-squash commits unreachable. The
// frozen log was bootstrapped BEFORE the squash; after the squash a freeze-on-write
// regen must produce ZERO churn — the merged log is byte-identical to the frozen
// input, 0 entries appended, 0 destroyed.
func TestFreeze_TC10_LoomRegression(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	// The pre-squash history: two unattributable migration commits touching the
	// same memory file (Part 2a then Part 2b — no registry change-id, as on loom).
	writeFile(t, repo, "docs/memory/wd-web-canvas/canvas.md", "# Canvas\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Part 2a: migrate canvas memory",
		"--date", "2026-05-10T12:00:00 +0000")
	writeFile(t, repo, "docs/memory/wd-web-canvas/canvas.md", "# Canvas v2\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Part 2b: finish canvas migration",
		"--date", "2026-05-11T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	// Bootstrap the frozen log from the PRE-squash history and commit it (the
	// state every contributor pulls).
	pre, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	frozen := logTargetFor(pre, "docs/memory/wd-web-canvas/log.md")
	if frozen == nil {
		t.Fatalf("bootstrap should project the migration commits into the log, got %+v", pre)
	}
	// Both pre-squash subjects were projected as unattributable lines.
	if !strings.Contains(frozen.Content, "Part 2a") || !strings.Contains(frozen.Content, "Part 2b") {
		t.Fatalf("frozen log should carry both pre-squash lines, got:\n%s", frozen.Content)
	}
	writeLog(t, frozen)

	// THE SQUASH: rewrite history so Part 2a/2b collapse into a single
	// unattributable commit (#1721) — git reset --soft to the root's parent is
	// impossible (root has no parent), so reset to an orphan baseline instead:
	// create a fresh root commit carrying the final tree, with the squashed
	// subject. This makes the pre-squash commits unreachable (git log shows only
	// #1721), reproducing the loom failure exactly.
	gitDateRun(t, repo, "checkout", "--orphan", "squashed")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1721 from loom/migration",
		"--date", "2026-05-12T12:00:00 +0000")

	// Freeze-on-write regen AFTER the squash: must be ZERO churn.
	post, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	postLog := logTargetFor(post, "docs/memory/wd-web-canvas/log.md")
	if postLog == nil {
		t.Fatal("post-squash regen dropped the log target")
	}
	if postLog.Content != frozen.Content {
		t.Errorf("LOOM REGRESSION: post-squash regen churned the log (expected 0 churn).\n--- frozen ---\n%s\n--- post-squash ---\n%s", frozen.Content, postLog.Content)
	}
	// Explicitly: the squash commit #1721 was NOT appended as a new line.
	if strings.Contains(postLog.Content, "#1721") {
		t.Errorf("the squashed unattributable #1721 must NOT be projected (frozen-not-reprojected), got:\n%s", postLog.Content)
	}
}

// --- --check redesign: superset PASS / missing FAIL / hand-edit FAIL -------
//
// `fab memory-index --check` byte-compares the on-disk log.md (Existing) against
// the freeze-on-write merge GatherLogs(repo, fab, false) produces (Rendered).
// These tests exercise that exact seam — the content relationship the cmd's
// Classify byte-compare reads — plus assert Classify keeps a log.md drift in the
// benign tier (R10), never destructive-loss. classifyLog is the cmd's per-log
// CheckTarget assembly, lifted here so the verdict is asserted directly.

// classifyLog runs Classify over a single log.md target exactly as the cmd does
// (IsLog true, the on-disk bytes vs the merge content) and returns the report.
func classifyLog(existing, rendered string, exists map[string]bool) LossReport {
	return Classify([]CheckTarget{
		{Path: "docs/memory/d/log.md", Existing: existing, Rendered: rendered, LinkBase: "d", IsLog: true},
	}, setExists(exists))
}

// TestFreeze_TC7_CheckSupersetPasses pins R7: a committed log that is a valid
// SUPERSET of the freeze-on-write merge (it carries a frozen line the live history
// no longer shows) is CLEAN — the merge reproduces the on-disk bytes exactly. This
// is the case byte-equality false-fails today.
func TestFreeze_TC7_CheckSupersetPasses(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260401-bbbb-live/.status.yaml",
		"id: bbbb\nname: 260401-bbbb-live\nsummary: \"live change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-live",
		"--date", "2026-04-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("expected a d/log.md target")
	}
	// Commit a SUPERSET: the projected live line + a frozen squash-stale line.
	superset := dLog.Content +
		"\n## 2026-01-01\n- **Update** [topic](/d/topic.md) — squash-stale frozen history (cccc)\n"
	if err := os.WriteFile(dLog.Path, []byte(superset), 0o644); err != nil {
		t.Fatal(err)
	}

	// --check (freeze-on-write merge): the merge re-renders existing ∪ projection.
	check, _ := GatherLogs(repo, fabRoot, false)
	checkLog := logTargetFor(check, "docs/memory/d/log.md")
	if checkLog == nil {
		t.Fatal("--check dropped the d/log.md target")
	}
	onDisk, _ := os.ReadFile(dLog.Path)
	if checkLog.Content != string(onDisk) {
		t.Errorf("R7: a valid superset must reproduce on-disk bytes (clean), got drift.\n--- on disk ---\n%s\n--- merge ---\n%s", onDisk, checkLog.Content)
	}
	report := classifyLog(string(onDisk), checkLog.Content, map[string]bool{"d/topic.md": true})
	if report.Tier != TierClean {
		t.Errorf("R7: superset log.md → TierClean, got %d (losses %v)", report.Tier, report.Losses)
	}
}

// TestFreeze_TC8_CheckMissingEntryFails pins R8: a committed log MISSING a
// projected attributable (file, change-id) entry drifts (non-clean) — the merge
// appends the missing entry, so the merge content differs from on-disk. Benign
// tier (R10), and the appended entry names the gap.
func TestFreeze_TC8_CheckMissingEntryFails(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260401-bbbb-live/.status.yaml",
		"id: bbbb\nname: 260401-bbbb-live\nsummary: \"live change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-live",
		"--date", "2026-04-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("expected a d/log.md target")
	}
	// Commit a log that LACKS the projected (topic, bbbb) entry (a header-only
	// log — someone forgot to regenerate-and-commit).
	missing := "# Log — D\n" + logHeaderComment + "\n"
	if err := os.WriteFile(dLog.Path, []byte(missing), 0o644); err != nil {
		t.Fatal(err)
	}

	check, _ := GatherLogs(repo, fabRoot, false)
	checkLog := logTargetFor(check, "docs/memory/d/log.md")
	if checkLog == nil {
		t.Fatal("--check dropped the d/log.md target")
	}
	onDisk, _ := os.ReadFile(dLog.Path)
	if checkLog.Content == string(onDisk) {
		t.Errorf("R8: a missing attributable entry must drift, got clean:\n%s", checkLog.Content)
	}
	// The merge names the gap (the bbbb entry it would append).
	if !strings.Contains(checkLog.Content, "live change (bbbb)") {
		t.Errorf("R8: the merge must surface the missing (topic, bbbb) entry, got:\n%s", checkLog.Content)
	}
	report := classifyLog(string(onDisk), checkLog.Content, map[string]bool{"d/topic.md": true})
	if report.Tier != TierBenignDrift {
		t.Errorf("R8/R10: missing-entry log.md drift → TierBenignDrift, got %d", report.Tier)
	}
}

// TestFreeze_TC9_CheckHandEditFails pins R9: a frozen line hand-edited in a
// RENDER-UNSTABLE way (broken §6.2 grammar parseLog cannot round-trip) drifts —
// the merge cannot reproduce the malformed bytes. Benign tier (R10).
func TestFreeze_TC9_CheckHandEditFails(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260401-bbbb-live/.status.yaml",
		"id: bbbb\nname: 260401-bbbb-live\nsummary: \"live change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #2 from o/260401-bbbb-live",
		"--date", "2026-04-01T12:00:00 +0000")
	fabRoot := filepath.Join(repo, "fab")

	first, _ := GatherLogs(repo, fabRoot, false)
	dLog := logTargetFor(first, "docs/memory/d/log.md")
	if dLog == nil {
		t.Fatal("expected a d/log.md target")
	}
	// Hand-edit: break the entry grammar (drop the ` — ` separator the renderer
	// always emits). parseLog skips the malformed bullet, so the merge re-renders
	// WITHOUT it (and re-appends the live projection) → differs from on-disk.
	handEdited := "# Log — D\n" + logHeaderComment + "\n" +
		"\n## 2026-04-01\n- **Update** [topic](/d/topic.md) hand-mangled no separator (bbbb)\n"
	if err := os.WriteFile(dLog.Path, []byte(handEdited), 0o644); err != nil {
		t.Fatal(err)
	}

	check, _ := GatherLogs(repo, fabRoot, false)
	checkLog := logTargetFor(check, "docs/memory/d/log.md")
	if checkLog == nil {
		t.Fatal("--check dropped the d/log.md target")
	}
	onDisk, _ := os.ReadFile(dLog.Path)
	if checkLog.Content == string(onDisk) {
		t.Errorf("R9: a render-unstable hand-edit must drift, got clean:\n%s", checkLog.Content)
	}
	report := classifyLog(string(onDisk), checkLog.Content, map[string]bool{"d/topic.md": true})
	if report.Tier != TierBenignDrift {
		t.Errorf("R9/R10: hand-edited log.md drift → TierBenignDrift, got %d (losses %v)", report.Tier, report.Losses)
	}
}
