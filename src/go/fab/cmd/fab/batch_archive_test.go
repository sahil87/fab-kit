package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsArchivable_HydrateDone(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  intake: done
  apply: done
  review: done
  hydrate: done
`), 0o644)

	if !isArchivable(statusPath) {
		t.Error("expected archivable for hydrate: done")
	}
}

func TestIsArchivable_HydrateSkipped(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  hydrate: skipped
`), 0o644)

	if !isArchivable(statusPath) {
		t.Error("expected archivable for hydrate: skipped")
	}
}

func TestIsArchivable_HydratePending(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  hydrate: pending
`), 0o644)

	if isArchivable(statusPath) {
		t.Error("expected not archivable for hydrate: pending")
	}
}

func TestIsArchivable_MissingFile(t *testing.T) {
	if isArchivable("/nonexistent/.status.yaml") {
		t.Error("expected not archivable for missing file")
	}
}

func TestAllArchivableNames(t *testing.T) {
	dir := t.TempDir()

	// Archivable change
	changeDir := filepath.Join(dir, "260401-ab12-done-change")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("  hydrate: done\n"), 0o644)

	// Non-archivable change
	pendingDir := filepath.Join(dir, "260401-cd34-pending-change")
	os.MkdirAll(pendingDir, 0o755)
	os.WriteFile(filepath.Join(pendingDir, ".status.yaml"), []byte("  hydrate: pending\n"), 0o644)

	// Archive directory (should be excluded)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	names := allArchivableNames(dir)
	if len(names) != 1 {
		t.Fatalf("expected 1 archivable, got %d", len(names))
	}
	if names[0] != "260401-ab12-done-change" {
		t.Errorf("name = %q, want %q", names[0], "260401-ab12-done-change")
	}
}

func TestAllArchivableNames_NoEligible(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "260401-ab12-pending")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("  hydrate: active\n"), 0o644)

	names := allArchivableNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 archivable, got %d", len(names))
	}
}

func TestListArchivable(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "260401-ab12-done")
	os.MkdirAll(changeDir, 0o755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("  hydrate: done\n"), 0o644)

	var buf bytes.Buffer
	listArchivable(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("260401-ab12-done")) {
		t.Error("expected change name in output")
	}
}

func TestListArchivable_None(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	listArchivable(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("(none)")) {
		t.Error("expected (none) in output")
	}
}

func TestArchiveLoop(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	changesDir := filepath.Join(fabRoot, "changes")

	// Helper to create an archivable change folder (intake + hydrate: done).
	makeChange := func(folder, title string) {
		cd := filepath.Join(changesDir, folder)
		os.MkdirAll(cd, 0o755)
		os.WriteFile(filepath.Join(cd, ".status.yaml"), []byte("progress:\n  hydrate: done\n"), 0o644)
		os.WriteFile(filepath.Join(cd, "intake.md"), []byte("# Intake: "+title+"\n"), 0o644)
	}

	makeChange("260401-aa11-first-change", "First change")
	makeChange("260401-bb22-second-change", "Second change")

	// A third change that is already archived (destination exists) → skipped.
	thirdFolder := "260401-cc33-third-change"
	makeChange(thirdFolder, "Third change")
	archivedDest := filepath.Join(changesDir, "archive", "2026", "04")
	os.MkdirAll(filepath.Join(archivedDest, thirdFolder), 0o755)

	var out, errOut bytes.Buffer
	resolved := []string{
		"260401-aa11-first-change",
		"260401-bb22-second-change",
		thirdFolder,
	}
	archived, skipped, failed := archiveLoop(&out, &errOut, fabRoot, resolved)

	if archived != 2 {
		t.Errorf("archived = %d, want 2", archived)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1 (already-archived third change)", skipped)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}

	// Both archivable changes moved under archive/2026/04/.
	for _, f := range []string{"260401-aa11-first-change", "260401-bb22-second-change"} {
		moved := filepath.Join(archivedDest, f)
		if _, err := os.Stat(moved); os.IsNotExist(err) {
			t.Errorf("%s not found in archive/2026/04/", f)
		}
		orig := filepath.Join(changesDir, f)
		if _, err := os.Stat(orig); !os.IsNotExist(err) {
			t.Errorf("%s should be removed from changes/ after archive", f)
		}
	}

	// Footer reflects the counts.
	if !strings.Contains(out.String(), "Archived 2, skipped 1, failed 0.") {
		t.Errorf("footer missing or wrong, got:\n%s", out.String())
	}
	// Already-archived third change is reported as skipped, not failed.
	if !strings.Contains(out.String(), thirdFolder+" — already archived, skipping") {
		t.Errorf("expected third change reported as skipped, got:\n%s", out.String())
	}
}

func TestBatchArchiveCmd_Structure(t *testing.T) {
	cmd := batchArchiveCmd()
	if cmd.Use != "archive [change...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "archive [change...]")
	}

	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}
