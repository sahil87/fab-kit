package lines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadFileLines_Basic(t *testing.T) {
	path := writeFile(t, "alpha\nbeta\ngamma\n")

	got, err := ReadFileLines(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"alpha", "beta", "gamma", ""}
	if len(got) != len(want) {
		t.Fatalf("got %d lines %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadFileLines_CRLF(t *testing.T) {
	path := writeFile(t, "alpha\r\nbeta\r\ngamma")

	got, err := ReadFileLines(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// bufio.ScanLines strips the trailing \r — the helper must match.
	want := []string{"alpha", "beta", "gamma"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q (trailing \\r must be trimmed)", i, got[i], want[i])
		}
	}
}

func TestReadFileLines_OversizedLine(t *testing.T) {
	// A single line beyond bufio.MaxScanTokenSize (64KB) aborted the old
	// scanner sites mid-file. The helper must return every line.
	long := strings.Repeat("x", 70*1024)
	path := writeFile(t, "before\n"+long+"\nafter\n")

	got, err := ReadFileLines(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d lines, want 4 (no truncation)", len(got))
	}
	if got[0] != "before" || got[1] != long || got[2] != "after" {
		t.Error("lines around the oversized line must survive intact")
	}
}

func TestReadFileLines_MissingFile(t *testing.T) {
	_, err := ReadFileLines(filepath.Join(t.TempDir(), "absent.md"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("error should satisfy os.IsNotExist, got %v", err)
	}
}

func TestReadFileLines_EmptyFile(t *testing.T) {
	path := writeFile(t, "")

	got, err := ReadFileLines(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// strings.Split("", "\n") yields one empty line — callers treat it as a
	// non-matching line, same net effect as the scanner's zero iterations.
	if len(got) != 1 || got[0] != "" {
		t.Errorf("got %v, want [\"\"]", got)
	}
}

func TestSplit_CRLFAndTrailingNewline(t *testing.T) {
	got := Split("a\r\nb\nc\n")
	want := []string{"a", "b", "c", ""}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplit_InteriorCarriageReturnPreserved(t *testing.T) {
	// Only a trailing \r is trimmed (bufio.ScanLines parity) — an interior
	// \r stays.
	got := Split("a\rb\nc")
	if got[0] != "a\rb" {
		t.Errorf("interior \\r must be preserved, got %q", got[0])
	}
}
