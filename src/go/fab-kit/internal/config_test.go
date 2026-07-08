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

// TestReadFabVersion_FromDotFile: fab/.fab-version is authoritative and wins over
// a (legacy) config.yaml fab_version: key (260708-j0qm relocation).
func TestReadFabVersion_FromDotFile(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	// Stale config.yaml key + a .fab-version sibling — the sibling wins.
	if err := os.WriteFile(configPath, []byte("fab_version: \"1.0.0\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "fab", ".fab-version"), []byte("2.15.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := readFabVersion(repoRoot, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "2.15.0" {
		t.Errorf("expected 2.15.0 from fab/.fab-version, got %s", v)
	}
}

// TestReadFabVersion_FallbackToConfig: with no fab/.fab-version (a not-yet-migrated
// repo), readFabVersion falls back to the config.yaml fab_version: key.
func TestReadFabVersion_FallbackToConfig(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("fab_version: \"0.43.0\"\nproject:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := readFabVersion(repoRoot, configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0 (config.yaml fallback), got %s", v)
	}
}

// TestReadFabVersion_MissingBothSources: neither fab/.fab-version nor a config.yaml
// key ⇒ a real error (the router needs a pinned version).
func TestReadFabVersion_MissingBothSources(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := readFabVersion(repoRoot, configPath)
	if err == nil {
		t.Fatal("expected error when neither .fab-version nor a config.yaml key is present")
	}
}

func TestReadFabVersion_InvalidYAML(t *testing.T) {
	repoRoot := t.TempDir()
	configDir := filepath.Join(repoRoot, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	// No .fab-version, so it falls through to parsing config.yaml, which is invalid.
	if err := os.WriteFile(configPath, []byte("not: valid: yaml: ["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := readFabVersion(repoRoot, configPath)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// TestExitNotManaged pins the distinct "not a fab-managed repo" exit code that
// external callers (wt, hop, operators) branch on. Changing it is a documented-
// contract break — the constant, not the literal 3, is what docs and consumers
// reference (mirrors the fab binary's TestTmuxExitCode / TestPaneValidationExitCode
// exit-scheme pins).
func TestExitNotManaged(t *testing.T) {
	if ExitNotManaged != 3 {
		t.Errorf("ExitNotManaged = %d, want 3 (documented not-a-fab-managed-repo exit code)", ExitNotManaged)
	}
}

// TestRequireManagedRepo_Managed: inside a managed repo, RequireManagedRepo
// returns the resolved config unchanged and never exits.
func TestRequireManagedRepo_Managed(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("fab_version: \"1.2.3\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	cfg, err := RequireManagedRepo()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config in a managed repo")
	}
	if cfg.FabVersion != "1.2.3" {
		t.Errorf("expected FabVersion 1.2.3, got %s", cfg.FabVersion)
	}
}

// TestRequireManagedRepo_RealError: a genuine ResolveConfig failure (config
// present but fab_version missing) is propagated as an error for the caller to
// return — it collapses to exit 1 in main(), NOT the ExitNotManaged path. This
// is the R2 guarantee that real failures stay exit 1. (The (nil, nil) unmanaged
// case calls os.Exit and is therefore the untested thin wrapper, mirroring the
// os.Exit branches in the fab binary's memory_index.go / doctor.go.)
func TestRequireManagedRepo_RealError(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	cfg, err := RequireManagedRepo()
	if err == nil {
		t.Fatal("expected a real error for a config missing fab_version (must stay exit-1 path, not ExitNotManaged)")
	}
	if cfg != nil {
		t.Errorf("expected nil config on error, got %+v", cfg)
	}
}
