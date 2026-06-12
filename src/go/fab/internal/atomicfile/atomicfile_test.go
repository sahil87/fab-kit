package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile_CreatesWithContentAndPerm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.md")

	if err := WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(data) != "hello\n" {
		t.Errorf("content = %q, want %q", data, "hello\n")
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o644 {
		t.Errorf("perm = %v, want 0644", info.Mode().Perm())
	}
}

func TestWriteFile_OverwritesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "target.md")
	os.WriteFile(path, []byte("old"), 0o644)

	if err := WriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("content = %q, want %q", data, "new")
	}
}

func TestWriteFile_NoTempResidue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "target.md")

	if err := WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestWriteFile_FailureLeavesOriginalUntouched(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied semantics do not apply to root")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "target.md")
	os.WriteFile(path, []byte("original"), 0o644)

	// A read-only directory makes CreateTemp fail before anything touches
	// the target.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if err := WriteFile(path, []byte("replacement"), 0o644); err == nil {
		t.Fatal("expected error in read-only directory, got nil")
	}

	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Errorf("original must be untouched on failure, got %q", data)
	}
}

func TestWriteFile_MissingDirectoryErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-dir", "target.md")
	if err := WriteFile(path, []byte("x"), 0o644); err == nil {
		t.Fatal("expected error for missing parent directory, got nil")
	}
}
