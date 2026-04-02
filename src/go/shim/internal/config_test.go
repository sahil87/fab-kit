package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigFrom_Found(t *testing.T) {
	// Create a temp directory tree: root/fab/project/config.yaml
	root := t.TempDir()
	configDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("fab_version: \"1.2.3\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Start search from a subdirectory
	subDir := filepath.Join(root, "src", "go", "shim")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	result, err := resolveConfigFrom(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FabVersion != "1.2.3" {
		t.Errorf("expected FabVersion 1.2.3, got %s", result.FabVersion)
	}
	if result.RepoRoot != root {
		t.Errorf("expected RepoRoot %s, got %s", root, result.RepoRoot)
	}
}

func TestResolveConfigFrom_NotFound(t *testing.T) {
	// Create a temp directory with no config
	dir := t.TempDir()

	result, err := resolveConfigFrom(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when no config found, got %+v", result)
	}
}

func TestResolveConfigFrom_MissingFabVersion(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Config exists but no fab_version field
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := resolveConfigFrom(root)
	if err == nil {
		t.Fatal("expected error for missing fab_version")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

func TestReadFabVersion_Valid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("fab_version: \"0.43.0\"\nproject:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0, got %s", v)
	}
}

func TestReadFabVersion_MissingField(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := readFabVersion(path)
	if err == nil {
		t.Fatal("expected error for missing fab_version field")
	}
}

func TestReadFabVersion_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: ["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := readFabVersion(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
