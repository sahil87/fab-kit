package main

import (
	"os"
	"path/filepath"
	"testing"
)

const testBacklog = `# Backlog

- [ ] [90g5] 2026-04-01: Add retry logic to API client
- [x] [done] 2026-03-30: Fix login page styling
- [ ] [jgt6] [DEV-123] 2026-04-01: Implement caching layer
  with Redis support for session storage
- [ ] [ab12] (BUG) 2026-04-02: Fix memory leak in worker pool
`

func writeTestBacklog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")
	if err := os.WriteFile(path, []byte(testBacklog), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParsePendingItems(t *testing.T) {
	path := writeTestBacklog(t)

	items := parsePendingItems(path)
	if len(items) != 3 {
		t.Fatalf("expected 3 pending items, got %d", len(items))
	}

	if items[0].id != "90g5" {
		t.Errorf("items[0].id = %q, want %q", items[0].id, "90g5")
	}
	if items[1].id != "jgt6" {
		t.Errorf("items[1].id = %q, want %q", items[1].id, "jgt6")
	}
	if items[2].id != "ab12" {
		t.Errorf("items[2].id = %q, want %q", items[2].id, "ab12")
	}
}

func TestExtractBacklogContent_SimpleItem(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := extractBacklogContent(path, "90g5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Add retry logic to API client" {
		t.Errorf("content = %q, want %q", content, "Add retry logic to API client")
	}
}

func TestExtractBacklogContent_ContinuationLine(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := extractBacklogContent(path, "jgt6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Implement caching layer with Redis support for session storage"
	if content != expected {
		t.Errorf("content = %q, want %q", content, expected)
	}
}

func TestExtractBacklogContent_NotFound(t *testing.T) {
	path := writeTestBacklog(t)

	_, err := extractBacklogContent(path, "zzzz")
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestExtractBacklogContent_BugPrefix(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := extractBacklogContent(path, "ab12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Fix memory leak in worker pool" {
		t.Errorf("content = %q, want %q", content, "Fix memory leak in worker pool")
	}
}

func TestBatchNewCmd_Structure(t *testing.T) {
	cmd := batchNewCmd()
	if cmd.Use != "new [backlog-id...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "new [backlog-id...]")
	}

	// Verify flags exist
	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("all") == nil {
		t.Error("missing --all flag")
	}
}
