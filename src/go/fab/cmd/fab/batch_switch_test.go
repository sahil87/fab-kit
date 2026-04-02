package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAllChangeNames(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "260401-cd34-fix-bug"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	names := allChangeNames(dir)
	if len(names) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(names))
	}

	// Should not include "archive"
	for _, name := range names {
		if name == "archive" {
			t.Error("archive should be excluded")
		}
	}
}

func TestAllChangeNames_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	names := allChangeNames(dir)
	if len(names) != 0 {
		t.Errorf("expected 0 changes, got %d", len(names))
	}
}

func TestGetBranchPrefix(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	os.WriteFile(configPath, []byte(`branch_prefix: "feature/"`), 0o644)
	got := getBranchPrefix(configPath)
	if got != "feature/" {
		t.Errorf("getBranchPrefix() = %q, want %q", got, "feature/")
	}
}

func TestGetBranchPrefix_Empty(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	os.WriteFile(configPath, []byte(`project:
  name: "test"
`), 0o644)
	got := getBranchPrefix(configPath)
	if got != "" {
		t.Errorf("getBranchPrefix() = %q, want empty", got)
	}
}

func TestGetBranchPrefix_MissingFile(t *testing.T) {
	got := getBranchPrefix("/nonexistent/config.yaml")
	if got != "" {
		t.Errorf("getBranchPrefix() = %q, want empty", got)
	}
}

func TestListChanges(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "260401-ab12-add-feature"), 0o755)
	os.MkdirAll(filepath.Join(dir, "archive"), 0o755)

	var buf bytes.Buffer
	listChanges(&buf, dir)

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("260401-ab12-add-feature")) {
		t.Error("expected change name in output")
	}
	if bytes.Contains([]byte(output), []byte("archive")) && !bytes.Contains([]byte(output), []byte("Available changes")) {
		t.Error("archive should not appear in list")
	}
}

func TestBatchSwitchCmd_Structure(t *testing.T) {
	cmd := batchSwitchCmd()
	if cmd.Use != "switch [change...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "switch [change...]")
	}

	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}
