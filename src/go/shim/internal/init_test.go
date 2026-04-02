package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetFabVersion_NewFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "fab", "project", "config.yaml")

	if err := setFabVersion(path, "0.43.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Fatal("file is empty")
	}

	// Verify the version can be read back
	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("cannot read back fab_version: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0, got %s", v)
	}
}

func TestSetFabVersion_ExistingFile(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.yaml")

	// Write existing content
	existing := "project:\n  name: test-project\nfab_version: \"0.42.0\"\n"
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := setFabVersion(path, "0.43.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify version updated
	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("cannot read back fab_version: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0, got %s", v)
	}
}

func TestCopyDir(t *testing.T) {
	// Create source structure
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "VERSION"), []byte("0.43.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skills", "test.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy to destination
	dst := filepath.Join(t.TempDir(), "kit")
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify files
	data, err := os.ReadFile(filepath.Join(dst, "VERSION"))
	if err != nil {
		t.Fatalf("VERSION not found: %v", err)
	}
	if string(data) != "0.43.0\n" {
		t.Errorf("unexpected VERSION content: %s", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dst, "skills", "test.md"))
	if err != nil {
		t.Fatalf("skills/test.md not found: %v", err)
	}
	if string(data) != "# Test\n" {
		t.Errorf("unexpected skill content: %s", string(data))
	}
}
