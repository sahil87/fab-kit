package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const statusTemplate = `id: {ID}
name: {NAME}
created: {CREATED}
created_by: {CREATED_BY}
change_type: feat
issues: []
progress:
  intake: pending
  spec: pending
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
checklist:
  generated: false
  path: checklist.md
  completed: 0
  total: 0
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
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
checklist:
  generated: false
  path: checklist.md
  completed: 0
  total: 0
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
	os.MkdirAll(filepath.Join(fabRoot, ".kit", "templates"), 0755)
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)

	// Write status template
	os.WriteFile(filepath.Join(fabRoot, ".kit", "templates", "status.yaml"), []byte(statusTemplate), 0644)

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

func TestSwitchBlank(t *testing.T) {
	fabRoot := setupChangeFixture(t)
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)

	// Create symlink
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	os.Symlink("fab/changes/"+folder+"/.status.yaml", symlinkPath)

	msg := SwitchBlank(fabRoot)
	if !strings.Contains(msg, "No active change") {
		t.Errorf("SwitchBlank output = %q, expected 'No active change'", msg)
	}

	// Verify symlink is removed
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error("symlink should be removed after SwitchBlank")
	}
}

func TestSwitchBlank_AlreadyBlank(t *testing.T) {
	fabRoot := setupChangeFixture(t)

	msg := SwitchBlank(fabRoot)
	if !strings.Contains(msg, "already blank") {
		t.Errorf("SwitchBlank output = %q, expected 'already blank'", msg)
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

	// Each entry should have format name:stage:state:score:indicative
	for _, entry := range results {
		parts := strings.Split(entry, ":")
		if len(parts) < 5 {
			t.Errorf("entry %q has fewer than 5 colon-separated parts (expected name:stage:state:score:indicative)", entry)
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
