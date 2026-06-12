package backlog

import (
	"os"
	"path/filepath"
	"strings"
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

func TestPath(t *testing.T) {
	got := Path("/tmp/fab")
	want := filepath.Join("/tmp/fab", "backlog.md")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestParsePending(t *testing.T) {
	path := writeTestBacklog(t)

	items, err := ParsePending(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 pending items, got %d", len(items))
	}

	if items[0].ID != "90g5" {
		t.Errorf("items[0].ID = %q, want %q", items[0].ID, "90g5")
	}
	if items[1].ID != "jgt6" {
		t.Errorf("items[1].ID = %q, want %q", items[1].ID, "jgt6")
	}
	if items[2].ID != "ab12" {
		t.Errorf("items[2].ID = %q, want %q", items[2].ID, "ab12")
	}
}

func TestParsePending_MissingFileReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backlog.md") // not written

	items, err := ParsePending(path)
	if err == nil {
		t.Fatal("expected error for missing backlog, got nil (previously swallowed)")
	}
	if items != nil {
		t.Errorf("items = %v, want nil on error", items)
	}
}

func TestParsePending_ItemsBelowOversizedLineSurvive(t *testing.T) {
	// The old scanner aborted on a >64KB line, silently dropping every
	// pending item below it from batch new --list/--all.
	long := "- [ ] [big1] 2026-04-01: " + strings.Repeat("x", 70*1024)
	body := "# Backlog\n\n" +
		long + "\n" +
		"- [ ] [aft1] 2026-04-02: item after the long line\n"
	path := filepath.Join(t.TempDir(), "backlog.md")
	os.WriteFile(path, []byte(body), 0o644)

	items, err := ParsePending(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items (oversized + after), got %d", len(items))
	}
	if items[1].ID != "aft1" {
		t.Errorf("items[1].ID = %q, want %q (item below long line must survive)", items[1].ID, "aft1")
	}
}

func TestExtractContent_IDAfterOversizedLineFound(t *testing.T) {
	long := "- [ ] [big1] 2026-04-01: " + strings.Repeat("x", 70*1024)
	body := "# Backlog\n\n" +
		long + "\n" +
		"- [ ] [aft1] 2026-04-02: item after the long line\n"
	path := filepath.Join(t.TempDir(), "backlog.md")
	os.WriteFile(path, []byte(body), 0o644)

	content, err := ExtractContent(path, "aft1")
	if err != nil {
		t.Fatalf("ID after the long line must be found, got: %v", err)
	}
	if content != "item after the long line" {
		t.Errorf("content = %q, want %q", content, "item after the long line")
	}
}

func TestExtractContent_ReadErrorIsNotMisreportedAsNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backlog.md") // not written

	_, err := ExtractContent(path, "90g5")
	if err == nil {
		t.Fatal("expected error for missing backlog, got nil")
	}
	if strings.Contains(err.Error(), "not found in backlog") {
		t.Errorf("read failure must not be reported as a missing ID, got %q", err.Error())
	}
}

func TestExtractContent_SimpleItem(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := ExtractContent(path, "90g5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Add retry logic to API client" {
		t.Errorf("content = %q, want %q", content, "Add retry logic to API client")
	}
}

func TestExtractContent_ContinuationLine(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := ExtractContent(path, "jgt6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Implement caching layer with Redis support for session storage"
	if content != expected {
		t.Errorf("content = %q, want %q", content, expected)
	}
}

func TestExtractContent_NotFound(t *testing.T) {
	path := writeTestBacklog(t)

	_, err := ExtractContent(path, "zzzz")
	if err == nil {
		t.Error("expected error for missing ID")
	}
}

func TestExtractContent_BugPrefix(t *testing.T) {
	path := writeTestBacklog(t)

	content, err := ExtractContent(path, "ab12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "Fix memory leak in worker pool" {
		t.Errorf("content = %q, want %q", content, "Fix memory leak in worker pool")
	}
}

func TestMarkDone_FlipsUnchecked(t *testing.T) {
	path := writeTestBacklog(t)

	status, err := MarkDone(path, "90g5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "marked" {
		t.Errorf("status = %q, want %q", status, "marked")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] [90g5] 2026-04-01: Add retry logic to API client") {
		t.Error("expected [90g5] line to be flipped to [x]")
	}
}

func TestMarkDone_AlreadyChecked(t *testing.T) {
	path := writeTestBacklog(t)
	before, _ := os.ReadFile(path)
	beforeInfo, _ := os.Stat(path)

	status, err := MarkDone(path, "done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "already" {
		t.Errorf("status = %q, want %q", status, "already")
	}

	// File must not be rewritten.
	after, _ := os.ReadFile(path)
	afterInfo, _ := os.Stat(path)
	if string(after) != string(before) {
		t.Error("file content should be unchanged for already-done item")
	}
	if afterInfo.ModTime() != beforeInfo.ModTime() {
		t.Error("file should not be rewritten for already-done item")
	}
}

func TestMarkDone_MissingID(t *testing.T) {
	path := writeTestBacklog(t)

	status, err := MarkDone(path, "zzzz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "not_found" {
		t.Errorf("status = %q, want %q", status, "not_found")
	}
}

func TestMarkDone_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md") // not written

	status, err := MarkDone(path, "90g5")
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if status != "not_found" {
		t.Errorf("status = %q, want %q", status, "not_found")
	}
}

func TestMarkDone_ContinuationUntouched(t *testing.T) {
	path := writeTestBacklog(t)

	if _, err := MarkDone(path, "jgt6"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	out := string(data)
	if !strings.Contains(out, "- [x] [jgt6] [DEV-123] 2026-04-01: Implement caching layer") {
		t.Error("expected [jgt6] item line flipped to [x]")
	}
	// The continuation line must be preserved verbatim.
	if !strings.Contains(out, "\n  with Redis support for session storage\n") {
		t.Error("continuation line should be untouched")
	}
}

func TestMarkDone_OnlyMatchingItemFlipped(t *testing.T) {
	path := writeTestBacklog(t)

	if _, err := MarkDone(path, "ab12"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	out := string(data)
	// Target flipped.
	if !strings.Contains(out, "- [x] [ab12] (BUG) 2026-04-02: Fix memory leak in worker pool") {
		t.Error("expected [ab12] line flipped")
	}
	// Other unchecked items remain unchecked.
	if !strings.Contains(out, "- [ ] [90g5]") {
		t.Error("[90g5] should remain unchecked")
	}
	if !strings.Contains(out, "- [ ] [jgt6]") {
		t.Error("[jgt6] should remain unchecked")
	}
}
