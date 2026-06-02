package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
)

const statusTemplate = `id: {ID}
name: {NAME}
created: {CREATED}
created_by: {CREATED_BY}
change_type: feat
issues: []
progress:
  intake: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: {CREATED}
`

const existingStatusYAML = `id: abcd
name: 260310-abcd-old-name
created: "2026-03-10T12:00:00Z"
created_by: test-user
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
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

// setupChangeFixture creates a fab structure with templates and config.
func setupChangeFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")

	// Create directories
	os.MkdirAll(filepath.Join(fabRoot, "changes"), 0755)
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)

	// Create kit directory (simulates cache kit) and set override
	kitDir := filepath.Join(dir, "kit")
	os.MkdirAll(filepath.Join(kitDir, "templates"), 0755)
	os.WriteFile(filepath.Join(kitDir, "templates", "status.yaml"), []byte(statusTemplate), 0644)
	kitpath.SetOverride(kitDir)
	t.Cleanup(func() { kitpath.SetOverride("") })

	// Write minimal config (needed for hooks)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("project:\n  name: test\n"), 0644)

	return fabRoot
}

func TestNew_ValidSlug(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	folder, err := New(fabRoot, "my-feature", "", "")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Verify folder name format: YYMMDD-XXXX-my-feature
	if !strings.HasSuffix(folder, "-my-feature") {
		t.Errorf("folder %q should end with -my-feature", folder)
	}

	parts := strings.SplitN(folder, "-", 3)
	if len(parts) != 3 {
		t.Fatalf("folder %q should have YYMMDD-XXXX-slug format", folder)
	}
	if len(parts[0]) != 6 {
		t.Errorf("date prefix %q should be 6 chars", parts[0])
	}
	if len(parts[1]) != 4 {
		t.Errorf("id %q should be 4 chars", parts[1])
	}

	// Verify directory was created
	changeDir := filepath.Join(fabRoot, "changes", folder)
	if _, err := os.Stat(changeDir); os.IsNotExist(err) {
		t.Error("change directory not created")
	}

	// Verify .status.yaml was initialized
	statusPath := filepath.Join(changeDir, ".status.yaml")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		t.Error(".status.yaml not created")
	}
}

func TestNew_ExplicitID(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	folder, err := New(fabRoot, "my-feature", "ab12", "")
	if err != nil {
		t.Fatalf("New with explicit ID failed: %v", err)
	}

	if !strings.Contains(folder, "-ab12-") {
		t.Errorf("folder %q should contain explicit ID ab12", folder)
	}
}

func TestNew_InvalidSlug(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	_, err := New(fabRoot, "my feature!", "", "")
	if err == nil {
		t.Fatal("expected error for invalid slug")
	}
	if !strings.Contains(err.Error(), "Invalid slug") {
		t.Errorf("error should mention invalid slug, got: %v", err)
	}
}

func TestNew_InvalidSlugLeadingHyphen(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	_, err := New(fabRoot, "-starts-with-hyphen", "", "")
	if err == nil {
		t.Fatal("expected error for slug with leading hyphen")
	}
}

func TestNew_IDCollision(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	// Create existing change with ID "ab12"
	existingFolder := "260310-ab12-existing"
	os.MkdirAll(filepath.Join(fabRoot, "changes", existingFolder), 0755)

	_, err := New(fabRoot, "other-thing", "ab12", "")
	if err == nil {
		t.Fatal("expected error for ID collision")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("error should mention collision, got: %v", err)
	}
}

func TestNew_EmptySlug(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	_, err := New(fabRoot, "", "", "")
	if err == nil {
		t.Fatal("expected error for empty slug")
	}
}

func TestRename(t *testing.T) {
	fabRoot := setupChangeFixture(t)
	folder := "260310-abcd-old-name"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(existingStatusYAML), 0644)

	// Create active symlink
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	os.Symlink("fab/changes/"+folder+"/.status.yaml", symlinkPath)

	newFolder, err := Rename(fabRoot, folder, "new-name")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if newFolder != "260310-abcd-new-name" {
		t.Errorf("newFolder = %q, want 260310-abcd-new-name", newFolder)
	}

	// Verify old dir is gone
	if _, err := os.Stat(changeDir); !os.IsNotExist(err) {
		t.Error("old directory should be removed")
	}

	// Verify new dir exists
	newDir := filepath.Join(fabRoot, "changes", newFolder)
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("new directory should exist")
	}

	// Verify symlink updated
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	expectedTarget := "fab/changes/260310-abcd-new-name/.status.yaml"
	if target != expectedTarget {
		t.Errorf("symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestRename_SameName(t *testing.T) {
	fabRoot := setupChangeFixture(t)
	folder := "260310-abcd-old-name"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(existingStatusYAML), 0644)

	_, err := Rename(fabRoot, folder, "old-name")
	if err == nil {
		t.Fatal("expected error when renaming to same name")
	}
}

func TestSwitch(t *testing.T) {
	fabRoot := setupChangeFixture(t)
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(existingStatusYAML), 0644)

	output, err := Switch(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("Switch failed: %v", err)
	}

	// Verify symlink was created
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	expectedTarget := "fab/changes/" + folder + "/.status.yaml"
	if target != expectedTarget {
		t.Errorf("symlink target = %q, want %q", target, expectedTarget)
	}

	// Verify output contains the change name
	if !strings.Contains(output, folder) {
		t.Errorf("output should contain folder name, got: %s", output)
	}
}

func TestSwitchNone(t *testing.T) {
	fabRoot := setupChangeFixture(t)
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	// Create symlink
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	os.Symlink("fab/changes/"+folder+"/.status.yaml", symlinkPath)

	msg := SwitchNone(fabRoot)
	if !strings.Contains(msg, "No active change") {
		t.Errorf("SwitchNone output = %q, expected 'No active change'", msg)
	}

	// Verify symlink is removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("symlink should be removed after SwitchNone")
	}
}

func TestSwitchNone_AlreadyDeactivated(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	msg := SwitchNone(fabRoot)
	if !strings.Contains(msg, "already deactivated") {
		t.Errorf("SwitchNone output = %q, expected 'already deactivated'", msg)
	}
}

func TestList(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	// Create two changes
	folder1 := "260310-abcd-first-change"
	changeDir1 := filepath.Join(fabRoot, "changes", folder1)
	os.MkdirAll(changeDir1, 0755)
	statusYAML1 := strings.Replace(existingStatusYAML, "abcd", "abcd", 1)
	statusYAML1 = strings.Replace(statusYAML1, "260310-abcd-old-name", folder1, 1)
	os.WriteFile(filepath.Join(changeDir1, ".status.yaml"), []byte(statusYAML1), 0644)

	folder2 := "260310-efgh-second-change"
	changeDir2 := filepath.Join(fabRoot, "changes", folder2)
	os.MkdirAll(changeDir2, 0755)
	statusYAML2 := strings.Replace(existingStatusYAML, "abcd", "efgh", -1)
	statusYAML2 = strings.Replace(statusYAML2, "260310-efgh-old-name", folder2, 1)
	os.WriteFile(filepath.Join(changeDir2, ".status.yaml"), []byte(statusYAML2), 0644)

	results, err := List(fabRoot, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("List returned %d entries, want 2", len(results))
	}

	// Each entry should have format name:display_stage:display_state:score
	// (the :indicative 5th field was dropped in 1.10.0).
	for _, entry := range results {
		parts := strings.Split(entry, ":")
		if len(parts) != 4 {
			t.Errorf("entry %q has %d colon-separated parts, want 4 (name:stage:state:score)", entry, len(parts))
		}
	}
}

func TestList_EmptyChanges(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	results, err := List(fabRoot, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("List should return 0 entries for empty changes, got %d", len(results))
	}
}

func TestListWithOptions_ShowStats(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	// (a) Block present with excluding
	folderA := "260310-aaaa-with-excluding"
	dirA := filepath.Join(fabRoot, "changes", folderA)
	os.MkdirAll(dirA, 0o755)
	yamlA := `id: aaaa
name: ` + folderA + `
created: "2026-03-10T12:00:00Z"
created_by: test-user
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
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
true_impact:
  added: 100
  deleted: 20
  net: 80
  excluding:
    added: 60
    deleted: 20
    net: 40
  computed_at: "2026-03-10T12:00:00Z"
  computed_at_stage: apply
last_updated: "2026-03-10T12:00:00Z"
`
	os.WriteFile(filepath.Join(dirA, ".status.yaml"), []byte(yamlA), 0o644)

	// (b) Block present without excluding
	folderB := "260310-bbbb-no-excluding"
	dirB := filepath.Join(fabRoot, "changes", folderB)
	os.MkdirAll(dirB, 0o755)
	yamlB := `id: bbbb
name: ` + folderB + `
created: "2026-03-10T12:00:00Z"
created_by: test-user
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
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
true_impact:
  added: 50
  deleted: 5
  net: 45
  computed_at: "2026-03-10T12:00:00Z"
  computed_at_stage: apply
last_updated: "2026-03-10T12:00:00Z"
`
	os.WriteFile(filepath.Join(dirB, ".status.yaml"), []byte(yamlB), 0o644)

	// (c) Block absent
	folderC := "260310-cccc-no-impact"
	dirC := filepath.Join(fabRoot, "changes", folderC)
	os.MkdirAll(dirC, 0o755)
	yamlC := strings.Replace(existingStatusYAML, "abcd", "cccc", -1)
	yamlC = strings.Replace(yamlC, "260310-cccc-old-name", folderC, 1)
	os.WriteFile(filepath.Join(dirC, ".status.yaml"), []byte(yamlC), 0o644)

	// Default list (no flag) — no impact column.
	plain, err := ListWithOptions(fabRoot, false, false)
	if err != nil {
		t.Fatalf("ListWithOptions: %v", err)
	}
	for _, row := range plain {
		// Plain row is name:display_stage:display_state:score → 3 colons
		// (the :indicative field was dropped in 1.10.0).
		if got := strings.Count(row, ":"); got != 3 {
			t.Errorf("expected 3 colons in plain row, got %d: %s", got, row)
		}
	}

	// With --show-stats — appended impact column → name:stage:state:score:impact.
	stats, err := ListWithOptions(fabRoot, false, true)
	if err != nil {
		t.Fatalf("ListWithOptions: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(stats))
	}

	rowsByPrefix := map[string]string{}
	for _, row := range stats {
		parts := strings.SplitN(row, ":", 5)
		if len(parts) != 5 {
			t.Errorf("expected 5 parts in stats row, got %d: %s", len(parts), row)
			continue
		}
		rowsByPrefix[parts[0]] = parts[4]
	}

	if got := rowsByPrefix[folderA]; got != "+40" {
		t.Errorf("folder A impact column = %q, want +40", got)
	}
	if got := rowsByPrefix[folderB]; got != "+45" {
		t.Errorf("folder B impact column = %q, want +45", got)
	}
	if got := rowsByPrefix[folderC]; got != "—" {
		t.Errorf("folder C impact column = %q, want —", got)
	}
}

// TestListWithOptions_ShowStatsTestSplit verifies the compact impl/tests/total
// split in the --show-stats impact column, including the per-component negative
// clamp when tests over-counts the total.
func TestListWithOptions_ShowStatsTestSplit(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	mkChange := func(id, suffix, trueImpact string) string {
		folder := "260310-" + id + "-" + suffix
		dir := filepath.Join(fabRoot, "changes", folder)
		os.MkdirAll(dir, 0o755)
		y := `id: ` + id + `
name: ` + folder + `
created: "2026-03-10T12:00:00Z"
created_by: test-user
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
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
` + trueImpact + `last_updated: "2026-03-10T12:00:00Z"
`
		os.WriteFile(filepath.Join(dir, ".status.yaml"), []byte(y), 0o644)
		return folder
	}

	// (a) Normal split: total (excluding.net) = 502, tests.net = 400 →
	//     impl = 102 → "102i+400t=502".
	folderSplit := mkChange("spl1", "split", `true_impact:
  added: 612
  deleted: 38
  net: 574
  excluding:
    added: 540
    deleted: 38
    net: 502
  tests:
    added: 400
    deleted: 0
    net: 400
  computed_at: "2026-03-10T12:00:00Z"
  computed_at_stage: apply
`)

	// (b) tests present but no excluding: total falls back to raw net (45),
	//     tests.net = 30 → impl = 15 → "15i+30t=45".
	folderRaw := mkChange("raw1", "raw-split", `true_impact:
  added: 50
  deleted: 5
  net: 45
  tests:
    added: 30
    deleted: 0
    net: 30
  computed_at: "2026-03-10T12:00:00Z"
  computed_at_stage: apply
`)

	// (c) tests over-counts total: total = 100, tests.net = 150 → impl clamps
	//     to 0 → "0i+150t=100" (never negative).
	folderClamp := mkChange("clm1", "clamp", `true_impact:
  added: 100
  deleted: 0
  net: 100
  excluding:
    added: 100
    deleted: 0
    net: 100
  tests:
    added: 150
    deleted: 0
    net: 150
  computed_at: "2026-03-10T12:00:00Z"
  computed_at_stage: apply
`)

	stats, err := ListWithOptions(fabRoot, false, true)
	if err != nil {
		t.Fatalf("ListWithOptions: %v", err)
	}

	got := map[string]string{}
	for _, row := range stats {
		parts := strings.SplitN(row, ":", 5)
		if len(parts) != 5 {
			t.Fatalf("expected 5 parts in stats row, got %d: %s", len(parts), row)
		}
		got[parts[0]] = parts[4]
	}

	if got[folderSplit] != "102i+400t=502" {
		t.Errorf("split impact column = %q, want 102i+400t=502", got[folderSplit])
	}
	if got[folderRaw] != "15i+30t=45" {
		t.Errorf("raw-split impact column = %q, want 15i+30t=45", got[folderRaw])
	}
	if got[folderClamp] != "0i+150t=100" {
		t.Errorf("clamp impact column = %q, want 0i+150t=100 (no negative impl)", got[folderClamp])
	}
}
