package configupgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// TestFreeze_Idempotent: running the reconciliation twice on the same input yields
// byte-identical output (Constitution III; the memoryindex byte-stability
// discipline). Covers the fence-append, live-preserve, and parking paths in one
// realistic fixture.
func TestFreeze_Idempotent(t *testing.T) {
	fields := fieldsForTest(t)
	src := `project:
    name: fab-kit
    description: FAB Kit

source_paths:
    - src/

# pin review to fable
agent:
    tiers:
        review:
            model: claude-fable-5
            effort: xhigh

legacy_mode: true
`
	first, _ := render(src, fields, "2.15.0")
	second, _ := render(first, fields, "2.15.0")
	if first != second {
		t.Errorf("render is not idempotent.\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestFreeze_ParkedNotDuplicated: a parked block from a prior run is preserved and
// NOT re-parked (appended exactly once — parked blocks are user territory, never
// regenerated). A third run is still byte-stable.
func TestFreeze_ParkedNotDuplicated(t *testing.T) {
	fields := fieldsForTest(t)
	src := "project:\n    name: t\n\nlegacy_mode: true\ncruft_key: 42\n"

	first, _ := render(src, fields, "2.15.0")
	if got := strings.Count(first, "#   legacy_mode: true"); got != 1 {
		t.Fatalf("expected legacy_mode parked exactly once, got %d:\n%s", got, first)
	}
	if got := strings.Count(first, "#   cruft_key: 42"); got != 1 {
		t.Fatalf("expected cruft_key parked exactly once, got %d:\n%s", got, first)
	}
	second, _ := render(first, fields, "2.15.0")
	if second != first {
		t.Errorf("re-run churned a parked block.\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
	if got := strings.Count(second, "#   legacy_mode: true"); got != 1 {
		t.Errorf("legacy_mode must stay parked exactly once across runs, got %d", got)
	}
}

// TestFreeze_VersionBumpRewritesStampOnly: a kit-version bump re-stamps the BEGIN
// anchor but does not duplicate the fence or churn the user's preamble/parkings.
func TestFreeze_VersionBumpRewritesStampOnly(t *testing.T) {
	fields := fieldsForTest(t)
	src := "project:\n    name: t\n\nlegacy_mode: true\n"

	v1, _ := render(src, fields, "2.15.0")
	v2, _ := render(v1, fields, "2.16.0")

	if strings.Count(v2, "# >>> fab reference") != 1 {
		t.Errorf("a version bump must not duplicate the fence:\n%s", v2)
	}
	if !strings.Contains(v2, "(kit 2.16.0)") {
		t.Errorf("BEGIN anchor must re-stamp to the new kit version:\n%s", v2)
	}
	if strings.Contains(v2, "(kit 2.15.0)") {
		t.Errorf("stale kit-version stamp must be replaced, not kept:\n%s", v2)
	}
	// The parked key survives the bump exactly once.
	if got := strings.Count(v2, "#   legacy_mode: true"); got != 1 {
		t.Errorf("parked key must survive a version bump exactly once, got %d", got)
	}
}

// TestFreeze_CarriesRenameOnce: a live field matching a (synthetic) registry row's
// renamed_from is carried to the new key, value verbatim, exactly once; a re-run is
// a no-op (the old key is gone, so nothing to carry). Uses a synthetic field set
// because renamed_from is "" on every shipped row today.
func TestFreeze_CarriesRenameOnce(t *testing.T) {
	// A minimal synthetic registry: one advertise:false field `new_key` renamed
	// from `old_key`, plus one advertise field so a fence still renders.
	fields := []configref.Field{
		{
			Key:         "new_key",
			Description: "renamed target",
			Scope:       configref.ScopeProject,
			Advertise:   false,
			RenamedFrom: "old_key",
		},
		{
			Key:         "branch_prefix",
			Description: "advertise field so a fence renders",
			Scope:       configref.ScopeProject,
			Advertise:   true,
			Segment:     "# branch_prefix\n# branch_prefix: \"\"",
		},
	}
	src := "old_key: keep-this-value\n"

	first, report := render(src, fields, "2.15.0")
	if !strings.Contains(first, "new_key: keep-this-value") {
		t.Errorf("rename must carry the value verbatim under the new key:\n%s", first)
	}
	if strings.Contains(first, "old_key:") {
		t.Errorf("the old key must be gone after the rename carry:\n%s", first)
	}
	if len(report) == 0 || !strings.Contains(strings.Join(report, "\n"), "old_key") {
		t.Errorf("the report must note the rename, got %v", report)
	}
	second, _ := render(first, fields, "2.15.0")
	if second != first {
		t.Errorf("rename carry must be idempotent.\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestRender_RenameSkippedWhenTargetLive (SF-a): when a rename's TARGET key is
// already live, carrying the rename would emit a duplicate top-level key (yaml.v3
// rejects it → LoadPath errors → every fab command bricks). The carry is skipped,
// the old key is left in place (the parker handles it as an unknown key), and the
// report notes the skip. The output must parse (no duplicate key).
func TestRender_RenameSkippedWhenTargetLive(t *testing.T) {
	fields := []configref.Field{
		{
			Key:         "new_key",
			Description: "renamed target",
			Scope:       configref.ScopeProject,
			Advertise:   false,
			RenamedFrom: "old_key",
		},
		{
			Key:         "branch_prefix",
			Description: "advertise field so a fence renders",
			Scope:       configref.ScopeProject,
			Advertise:   true,
			Segment:     "# branch_prefix\n# branch_prefix: \"\"",
		},
	}
	// Both old_key AND new_key are live — the target is already set.
	src := "old_key: legacy-value\nnew_key: user-chosen-value\n"

	out, report := render(src, fields, "2.15.0")

	if err := validateYAML(out); err != nil {
		t.Fatalf("skipping the carry must keep the output parseable (no duplicate key): %v\n%s", err, out)
	}
	if strings.Count(out, "new_key:") != 1 {
		t.Errorf("new_key must appear exactly once (no duplicate top-level key):\n%s", out)
	}
	if !strings.Contains(out, "user-chosen-value") {
		t.Errorf("the user's explicit new_key value must be preserved:\n%s", out)
	}
	// old_key is now an unknown key (its registry row was consumed by the rename it
	// could not carry) and is parked below the fence — never dropped.
	if !strings.Contains(out, "#   old_key: legacy-value") {
		t.Errorf("old_key must be parked (not carried, not dropped):\n%s", out)
	}
	joined := strings.Join(report, "\n")
	if !strings.Contains(joined, "skipped rename") {
		t.Errorf("report must note the skipped rename, got %v", report)
	}
}

// TestRegistryRenames_NestedAndSameTopSkipped (SF-a): a same-top-level rename
// (a.x→a.y) and a nested rename (a.x→b.y) are NOT added to the top-level rename map,
// so carryRenames neither mis-renames a whole block nor logs a spurious carry line.
func TestRegistryRenames_NestedAndSameTopSkipped(t *testing.T) {
	fields := []configref.Field{
		{Key: "agent.new", Description: "same-top rename", Scope: configref.ScopeBoth, RenamedFrom: "agent.old"},
		{Key: "b.y", Description: "nested cross-top rename", Scope: configref.ScopeProject, RenamedFrom: "a.x"},
		{Key: "flat_new", Description: "genuine top-level rename", Scope: configref.ScopeProject, RenamedFrom: "flat_old"},
	}
	_, renames := registryTopLevelKeys(fields)
	if _, ok := renames["agent"]; ok {
		t.Error("a same-top-level rename (agent.old→agent.new) must NOT enter the rename map")
	}
	if _, ok := renames["a"]; ok {
		t.Error("a nested cross-top rename (a.x→b.y) must NOT enter the rename map")
	}
	if got := renames["flat_old"]; got != "flat_new" {
		t.Errorf("a genuine top-level rename must be carried: flat_old→%q, want flat_new", got)
	}
	if len(renames) != 1 {
		t.Errorf("only the genuine top-level rename should be present, got %v", renames)
	}
}

// TestUpgrade_AtomicWriteAndNoOp exercises the file-I/O entry point: a first run
// rewrites the file (Changed=true) and a second run is a no-op (Changed=false,
// byte-identical on disk).
func TestUpgrade_AtomicWriteAndNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("project:\n    name: t\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res1, err := Upgrade(path, "2.15.0")
	if err != nil {
		t.Fatalf("Upgrade #1: %v", err)
	}
	if !res1.Changed {
		t.Error("first Upgrade should report Changed=true (fence appended)")
	}
	after1, _ := os.ReadFile(path)

	res2, err := Upgrade(path, "2.15.0")
	if err != nil {
		t.Fatalf("Upgrade #2: %v", err)
	}
	if res2.Changed {
		t.Error("second Upgrade should be a no-op (Changed=false)")
	}
	after2, _ := os.ReadFile(path)
	if string(after1) != string(after2) {
		t.Errorf("second Upgrade churned the file.\n--- after1 ---\n%s\n--- after2 ---\n%s", after1, after2)
	}
}

// TestUpgrade_MissingFileWritesFence: a missing config.yaml is treated as empty —
// Upgrade writes a fresh fence-only file and reports Changed=true.
func TestUpgrade_MissingFileWritesFence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	res, err := Upgrade(path, "2.15.0")
	if err != nil {
		t.Fatalf("Upgrade on a missing file: %v", err)
	}
	if !res.Changed {
		t.Error("Upgrade on a missing file should report Changed=true")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected a written file: %v", err)
	}
	if !strings.HasPrefix(string(data), "# >>> fab reference (kit 2.15.0) >>> ") {
		t.Errorf("missing-file write should start with the BEGIN anchor:\n%s", data)
	}
}
