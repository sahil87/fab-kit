package archive

import (
	"errors"
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

	// Empty description now succeeds: the fixture has no intake.md, so the
	// description falls back to the humanized slug.
	result, err := Archive(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("expected empty description to succeed via slug fallback, got %v", err)
	}
	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, _ := os.ReadFile(indexFile)
	if !strings.Contains(string(data), "my change") {
		t.Errorf("index.md should contain humanized slug 'my change', got:\n%s", string(data))
	}
	_ = result
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
		Move:    "moved",
		Index:   "created",
		Pointer: "cleared",
		Backlog: "marked",
	}
	output := FormatArchiveYAML(r)

	for _, want := range []string{"action: archive", "name: 260310-abcd-my-change", "move: moved", "index: created", "pointer: cleared", "backlog: marked"} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatArchiveYAML missing %q", want)
		}
	}
	if strings.Contains(output, "clean:") {
		t.Error("FormatArchiveYAML should not contain clean: field")
	}
}

func TestArchive_DerivesFromIntakeTitle(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	// Add an intake.md to the source folder so the description is derived.
	intakeBody := "# Intake: Add OAuth support to the login flow\n"
	os.WriteFile(filepath.Join(fabRoot, "changes", folder, "intake.md"), []byte(intakeBody), 0o644)

	_, err := Archive(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, _ := os.ReadFile(indexFile)
	if !strings.Contains(string(data), "Add OAuth support to the login flow") {
		t.Errorf("index.md should contain the derived intake title, got:\n%s", string(data))
	}
}

func TestArchive_SlugFallback(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// No intake.md → description falls back to humanized slug.
	_, err := Archive(fabRoot, "abcd", "")
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	data, _ := os.ReadFile(indexFile)
	if !strings.Contains(string(data), "my change") {
		t.Errorf("index.md should contain humanized slug 'my change', got:\n%s", string(data))
	}
}

func TestArchiveWithBacklog_MarksDone(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	backlogBody := "# Backlog\n\n- [ ] [abcd] 2026-03-10: make my change\n"
	os.WriteFile(filepath.Join(fabRoot, "backlog.md"), []byte(backlogBody), 0o644)

	result, err := ArchiveWithBacklog(fabRoot, "abcd", "desc")
	if err != nil {
		t.Fatalf("ArchiveWithBacklog failed: %v", err)
	}
	if result.Backlog != "marked" {
		t.Errorf("Backlog = %q, want %q", result.Backlog, "marked")
	}

	data, _ := os.ReadFile(filepath.Join(fabRoot, "backlog.md"))
	if !strings.Contains(string(data), "- [x] [abcd]") {
		t.Errorf("backlog line should be flipped to [x], got:\n%s", string(data))
	}
}

func TestArchiveWithBacklog_NoBacklogFile(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	// No backlog.md exists.

	result, err := ArchiveWithBacklog(fabRoot, "abcd", "desc")
	if err != nil {
		t.Fatalf("ArchiveWithBacklog should succeed with no backlog file: %v", err)
	}
	if result.Backlog != "not_found" {
		t.Errorf("Backlog = %q, want %q", result.Backlog, "not_found")
	}
}

func TestArchiveWithBacklog_NotFromBacklog(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// Backlog has no matching ID.
	backlogBody := "# Backlog\n\n- [ ] [zzzz] 2026-03-10: unrelated item\n"
	os.WriteFile(filepath.Join(fabRoot, "backlog.md"), []byte(backlogBody), 0o644)

	result, err := ArchiveWithBacklog(fabRoot, "abcd", "desc")
	if err != nil {
		t.Fatalf("ArchiveWithBacklog failed: %v", err)
	}
	if result.Backlog != "not_found" {
		t.Errorf("Backlog = %q, want %q", result.Backlog, "not_found")
	}
}

func TestArchiveWithBacklog_MarkErrorPropagates(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// Make backlog.md unreadable by creating it as a directory, so MarkDone's
	// os.ReadFile fails with a non-IsNotExist error (distinct from the silent
	// missing-file no-op). The archive move still succeeds first.
	if err := os.Mkdir(filepath.Join(fabRoot, "backlog.md"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := ArchiveWithBacklog(fabRoot, "abcd", "desc")
	if err == nil {
		t.Fatal("expected an error when the backlog mark fails, got nil")
	}
	// The archive move succeeded, so the result must still be returned.
	if result == nil {
		t.Fatal("result should be non-nil — the archive move succeeded before the backlog failure")
	}
	if result.Move != "moved" {
		t.Errorf("Move = %q, want %q (archive should have completed)", result.Move, "moved")
	}
	// The folder must actually be in the archive despite the backlog failure.
	if _, statErr := os.Stat(filepath.Join(fabRoot, "changes", "archive", "2026", "03", "260310-abcd-my-change")); statErr != nil {
		t.Errorf("archived folder should exist despite backlog failure: %v", statErr)
	}
}

func TestArchive_ErrAlreadyArchivedOnReArchive(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	// First archive moves the folder.
	if _, err := Archive(fabRoot, "abcd", "desc"); err != nil {
		t.Fatalf("first archive failed: %v", err)
	}

	// Recreate the source folder so resolution succeeds, then re-archive.
	changeDir := filepath.Join(fabRoot, "changes", folder)
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(testStatusYAML), 0o644)

	_, err := Archive(fabRoot, folder, "desc")
	if !errors.Is(err, ErrAlreadyArchived) {
		t.Errorf("expected ErrAlreadyArchived, got %v", err)
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

// --- Re-archive soft skip for genuinely archived changes (k4ge) ---

func TestArchive_GenuinelyArchivedSoftSkips(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// First archive moves the folder out of fab/changes/ entirely.
	if _, err := Archive(fabRoot, "abcd", "desc"); err != nil {
		t.Fatalf("first archive failed: %v", err)
	}

	// Re-archive WITHOUT recreating the source folder — the documented
	// soft-skip case the binary previously failed with "No change matches".
	_, err := Archive(fabRoot, "abcd", "desc")
	if !errors.Is(err, ErrAlreadyArchived) {
		t.Errorf("expected ErrAlreadyArchived for genuinely archived change, got %v", err)
	}
}

func TestArchive_UnknownChangePropagatesResolveError(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	_, err := Archive(fabRoot, "zzzz-no-such-change", "desc")
	if err == nil {
		t.Fatal("expected an error for an unknown change")
	}
	if errors.Is(err, ErrAlreadyArchived) {
		t.Errorf("unknown change must not soft-skip, got %v", err)
	}
}

func TestIsArchived(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	if IsArchived(fabRoot, "abcd") {
		t.Error("change should not be archived before Archive runs")
	}

	if _, err := Archive(fabRoot, "abcd", "desc"); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	if !IsArchived(fabRoot, "abcd") {
		t.Error("change should be detected as archived after Archive")
	}
	if IsArchived(fabRoot, "zzzz-no-such-change") {
		t.Error("unknown name must not be detected as archived")
	}
}

// --- Index truncation-safety and honest error reporting (hv7t) ---

func TestRemoveFromIndex_PreservesEntriesAfterOversizedLine(t *testing.T) {
	// The old scanner aborted on a >64KB line; with found=true the rewrite
	// then silently deleted every entry after the abort point. The rewrite
	// must always derive from the complete file.
	dir := t.TempDir()
	indexFile := filepath.Join(dir, "index.md")
	long := "- **260101-zzzz-long** — " + strings.Repeat("x", 70*1024)
	content := "# Archive Index\n\n" +
		"- **260310-abcd-my-change** — target entry\n" +
		long + "\n" +
		"- **260311-wxyz-survivor** — must survive\n"
	os.WriteFile(indexFile, []byte(content), 0o644)

	status, err := removeFromIndex(indexFile, "260310-abcd-my-change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "removed" {
		t.Errorf("status = %q, want removed", status)
	}

	data, _ := os.ReadFile(indexFile)
	out := string(data)
	if strings.Contains(out, "**260310-abcd-my-change**") {
		t.Error("target entry should be removed")
	}
	if !strings.Contains(out, "**260311-wxyz-survivor**") {
		t.Error("entry after the oversized line must survive the rewrite")
	}
	if !strings.Contains(out, "**260101-zzzz-long**") {
		t.Error("the oversized entry itself must survive the rewrite")
	}
}

func TestRemoveFromIndex_MissingFileIsBenign(t *testing.T) {
	status, err := removeFromIndex(filepath.Join(t.TempDir(), "index.md"), "260310-abcd-my-change")
	if err != nil {
		t.Fatalf("missing index must stay a benign not_found, got error: %v", err)
	}
	if status != "not_found" {
		t.Errorf("status = %q, want not_found", status)
	}
}

func TestRemoveFromIndex_ReadFailureReturnsError(t *testing.T) {
	// A directory at the index path makes os.ReadFile fail with a
	// non-IsNotExist error — previously reported as "not_found".
	dir := t.TempDir()
	indexFile := filepath.Join(dir, "index.md")
	os.Mkdir(indexFile, 0o755)

	status, err := removeFromIndex(indexFile, "260310-abcd-my-change")
	if err == nil {
		t.Fatal("expected error for unreadable index, got nil")
	}
	if status != "failed" {
		t.Errorf("status = %q, want failed", status)
	}
}

func TestRestore_IndexFailureSurfacedHonestly(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	if _, err := Archive(fabRoot, "abcd", "desc"); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	// Replace the index with a directory: removeFromIndex's read fails.
	indexFile := filepath.Join(fabRoot, "changes", "archive", "index.md")
	os.Remove(indexFile)
	os.Mkdir(indexFile, 0o755)

	result, err := Restore(fabRoot, folder, false)
	if err == nil {
		t.Fatal("expected error when the index update fails, got nil")
	}
	if result == nil {
		t.Fatal("result must be non-nil — the restore move succeeded before the index failure")
	}
	if result.Move != "restored" {
		t.Errorf("Move = %q, want restored", result.Move)
	}
	if result.Index != "failed" {
		t.Errorf("Index = %q, want failed", result.Index)
	}
}

func TestArchive_IndexFailureSurfacedHonestly(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	// A directory at the index path makes updateIndex's read fail with a
	// non-IsNotExist error — previously swallowed with an unconditional
	// "updated" return.
	archiveDir := filepath.Join(fabRoot, "changes", "archive")
	os.MkdirAll(archiveDir, 0o755)
	os.Mkdir(filepath.Join(archiveDir, "index.md"), 0o755)

	result, err := Archive(fabRoot, "abcd", "desc")
	if err == nil {
		t.Fatal("expected error when the index update fails, got nil")
	}
	if result == nil {
		t.Fatal("result must be non-nil — the move succeeded before the index failure")
	}
	if result.Move != "moved" {
		t.Errorf("Move = %q, want moved (the move completed)", result.Move)
	}
	if result.Index != "failed" {
		t.Errorf("Index = %q, want failed", result.Index)
	}

	// The folder must actually be in the archive despite the index failure.
	if _, statErr := os.Stat(filepath.Join(archiveDir, "2026", "03", "260310-abcd-my-change")); statErr != nil {
		t.Errorf("archived folder should exist despite index failure: %v", statErr)
	}
}

func TestArchiveWithBacklog_IndexFailureStillMarksBacklog(t *testing.T) {
	fabRoot := setupArchiveFixture(t)

	backlogBody := "# Backlog\n\n- [ ] [abcd] 2026-03-10: make my change\n"
	os.WriteFile(filepath.Join(fabRoot, "backlog.md"), []byte(backlogBody), 0o644)

	archiveDir := filepath.Join(fabRoot, "changes", "archive")
	os.MkdirAll(archiveDir, 0o755)
	os.Mkdir(filepath.Join(archiveDir, "index.md"), 0o755)

	result, err := ArchiveWithBacklog(fabRoot, "abcd", "desc")
	if err == nil {
		t.Fatal("expected the index failure to propagate, got nil")
	}
	if result == nil {
		t.Fatal("result must be non-nil for a partial archive")
	}
	// The move is irreversible and a re-run soft-skips, so the backlog mark
	// must still have happened.
	if result.Backlog != "marked" {
		t.Errorf("Backlog = %q, want marked despite the index failure", result.Backlog)
	}
	data, _ := os.ReadFile(filepath.Join(fabRoot, "backlog.md"))
	if !strings.Contains(string(data), "- [x] [abcd]") {
		t.Errorf("backlog line should be flipped to [x], got:\n%s", string(data))
	}
}

func TestUpdateIndex_BackfillDerivesFromWrittenContent(t *testing.T) {
	// backfillIndex receives the content updateIndex just wrote — verify an
	// archive op both inserts the new entry and backfills a pre-index folder
	// in one pass.
	fabRoot := setupArchiveFixture(t)
	archiveDir := filepath.Join(fabRoot, "changes", "archive")

	// A pre-index archived folder with no index entry.
	preIndexed := "260201-old1-pre-index-change"
	os.MkdirAll(filepath.Join(archiveDir, "2026", "02", preIndexed), 0o755)

	if _, err := Archive(fabRoot, "abcd", "fresh entry"); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(archiveDir, "index.md"))
	out := string(data)
	if !strings.Contains(out, "**260310-abcd-my-change** — fresh entry") {
		t.Errorf("new entry missing from index:\n%s", out)
	}
	if !strings.Contains(out, "**"+preIndexed+"** — (no description — pre-index archive)") {
		t.Errorf("pre-index folder not backfilled:\n%s", out)
	}
}

// --- Restore --switch surfaces activation failure (k4ge) ---

func TestRestore_WithSwitchActivationFailure(t *testing.T) {
	fabRoot := setupArchiveFixture(t)
	folder := "260310-abcd-my-change"

	if _, err := Archive(fabRoot, "abcd", "desc"); err != nil {
		t.Fatalf("archive failed: %v", err)
	}

	// Block symlink creation: a non-empty directory at the .fab-status.yaml
	// path makes change.Switch's os.Remove + os.Symlink fail.
	repoRoot := filepath.Dir(fabRoot)
	pointerPath := filepath.Join(repoRoot, ".fab-status.yaml")
	os.Remove(pointerPath) // drop the fixture's symlink first
	if err := os.MkdirAll(filepath.Join(pointerPath, "block"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	result, err := Restore(fabRoot, folder, true)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if result.Move != "restored" {
		t.Errorf("Move = %q, want restored", result.Move)
	}
	if result.Pointer != "failed" {
		t.Errorf("Pointer = %q, want failed (activation failure must be surfaced)", result.Pointer)
	}
}
