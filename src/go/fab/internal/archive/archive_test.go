package archive

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testStatusYAML = `id: abcd
name: 260310-abcd-my-change
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

// setupArchiveFixture creates a fab structure with an active change and symlink.
func setupArchiveFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(testStatusYAML), 0644)

	// Create active symlink
	symlinkPath := filepath.Join(dir, ".fab-status.yaml")
	os.Symlink("fab/changes/"+folder+"/.status.yaml", symlinkPath)

	return fabRoot
}

func TestArchive(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	result, err := Archive(fabRoot, "abcd", "Completed feature")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	if result.Action != "archive" {
		t.Errorf("Action = %q, want archive", result.Action)
	}
	if result.Name != folder {
		t.Errorf("Name = %q, want %q", result.Name, folder)
	}
	if result.Move != "moved" {
		t.Errorf("Move = %q, want moved", result.Move)
	}

	// Verify folder moved to archive/2026/03/
	archivedDir := filepath.Join(fabRoot, "changes", "archive", "2026", "03", folder)
	if _, err := os.Stat(archivedDir); os.IsNotExist(err) {
		t.Error("change folder not found in archive directory")
	}

	// Verify original is gone
	origDir := filepath.Join(fabRoot, "changes", folder)
	if _, err := os.Stat(origDir); !os.IsNotExist(err) {
		t.Error("original change folder should be removed after archive")
	}

	// Verify index was created/updated
	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatalf("failed to read index.md: %v", err)
	}
	if !strings.Contains(string(data), folder) {
		t.Error("index.md should contain the archived change name")
	}
	if !strings.Contains(string(data), "Completed feature") {
		t.Error("index.md should contain the description")
	}

	// Verify symlink was cleared
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
		t.Error(".fab-status.yaml symlink should be removed after archiving active change")
	}
	if result.Pointer != "cleared" {
		t.Errorf("Pointer = %q, want cleared", result.Pointer)
	}
}

func TestArchive_MissingArgs(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	_, err := Archive(fabRoot, "", "desc")
	if err == nil {
		t.Error("expected error for empty changeArg")
	}

	_, err = Archive(fabRoot, "abcd", "")
	if err == nil {
		t.Error("expected error for empty description")
	}
}

func TestRestore(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	// First archive the change
	_, err := Archive(fabRoot, "abcd", "test archive")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Restore without switch
	result, err := Restore(fabRoot, "abcd", false)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	if result.Action != "restore" {
		t.Errorf("Action = %q, want restore", result.Action)
	}
	if result.Name != folder {
		t.Errorf("Name = %q, want %q", result.Name, folder)
	}
	if result.Move != "restored" {
		t.Errorf("Move = %q, want restored", result.Move)
	}
	if result.Pointer != "skipped" {
		t.Errorf("Pointer = %q, want skipped", result.Pointer)
	}

	// Verify folder is back in changes/
	restoredDir := filepath.Join(fabRoot, "changes", folder)
	if _, err := os.Stat(restoredDir); os.IsNotExist(err) {
		t.Error("change folder not found in changes/ after restore")
	}

	// Verify index entry was removed
	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, err := os.ReadFile(indexFile)
	if err != nil {
		t.Fatalf("failed to read index.md: %v", err)
	}
	if strings.Contains(string(data), "**"+folder+"**") {
		t.Error("index.md should not contain the restored change name")
	}
}

func TestRestore_WithSwitch(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	// Archive first
	_, err := Archive(fabRoot, "abcd", "test archive")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Restore with switch
	result, err := Restore(fabRoot, "abcd", true)
	if err != nil {
		t.Fatalf("Restore with switch failed: %v", err)
	}

	if result.Pointer != "switched" {
		t.Errorf("Pointer = %q, want switched", result.Pointer)
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
}

func TestList(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// Archive the change
	_, err := Archive(fabRoot, "abcd", "test")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Create a second change and archive it too
	folder2 := "260310-efgh-second-change"
	changeDir2 := filepath.Join(fabRoot, "changes", folder2)
	os.MkdirAll(changeDir2, 0755)
	statusYAML2 := strings.Replace(testStatusYAML, "abcd", "efgh", -1)
	statusYAML2 = strings.Replace(statusYAML2, "260310-abcd-my-change", folder2, -1)
	os.WriteFile(filepath.Join(changeDir2, ".status.yaml"), []byte(statusYAML2), 0644)

	_, err = Archive(fabRoot, folder2, "second archive")
	if err != nil {
		t.Fatalf("Archive second failed: %v", err)
	}

	// List archived changes
	list, err := List(fabRoot)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List returned %d entries, want 2", len(list))
	}

	found := map[string]bool{}
	for _, name := range list {
		found[name] = true
	}
	if !found["260310-abcd-my-change"] {
		t.Error("missing 260310-abcd-my-change in list")
	}
	if !found[folder2] {
		t.Error("missing second change in list")
	}
}

func TestList_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "changes"), 0755)
	// No archive directory

	list, err := List(fabRoot)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if list != nil {
		t.Errorf("List should return nil for no archive, got %v", list)
	}
}

func TestFormatArchiveYAML(t *testing.T) {
	r := &ArchiveResult{
		Action:  "archive",
		Name:    "260310-abcd-my-change",
		Clean:   "not_present",
		Move:    "moved",
		Index:   "created",
		Pointer: "cleared",
	}
	output := FormatArchiveYAML(r)

	for _, want := range []string{"action: archive", "name: 260310-abcd-my-change", "move: moved", "pointer: cleared"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatArchiveYAML missing %q", want)
		}
	}
}

func TestFormatRestoreYAML(t *testing.T) {
	r := &RestoreResult{
		Action:  "restore",
		Name:    "260310-abcd-my-change",
		Move:    "restored",
		Index:   "removed",
		Pointer: "skipped",
	}
	output := FormatRestoreYAML(r)

	for _, want := range []string{"action: restore", "name: 260310-abcd-my-change", "move: restored"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatRestoreYAML missing %q", want)
		}
	}
}
