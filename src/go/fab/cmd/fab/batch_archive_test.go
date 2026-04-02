package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestIsArchivable_HydrateDone(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, ".status.yaml")
	os.WriteFile(statusPath, []byte(`progress:
  intake: done
  spec: done
  tasks: done
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
