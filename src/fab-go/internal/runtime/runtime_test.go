package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func setupFabRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)
	return fabRoot
}

func TestFilePath(t *testing.T) {
	fabRoot := "/tmp/repo/fab"
	got := FilePath(fabRoot)
	want := "/tmp/repo/.fab-runtime.yaml"
	if got != want {
		t.Errorf("FilePath(%q) = %q, want %q", fabRoot, got, want)
	}
}

func TestLoadFile_NonExistent(t *testing.T) {
	m, err := LoadFile("/tmp/nonexistent-runtime.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestLoadFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")
	content := `test-change:
  agent:
    idle_since: 1741193400
`
	os.WriteFile(path, []byte(content), 0o644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	entry, ok := m["test-change"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test-change entry as map")
	}
	agent, ok := entry["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agent entry as map")
	}
	if agent["idle_since"] == nil {
		t.Error("expected idle_since to be set")
	}
}

func TestLoadFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")
	os.WriteFile(path, []byte(""), 0o644)

	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestSaveFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".fab-runtime.yaml")

	m := map[string]interface{}{
		"test-change": map[string]interface{}{
			"agent": map[string]interface{}{
				"idle_since": 1741193400,
			},
		},
	}

	if err := SaveFile(path, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file was written
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var loaded map[string]interface{}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to parse saved file: %v", err)
	}

	entry, ok := loaded["test-change"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test-change entry")
	}
	agent, ok := entry["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agent entry")
	}
	if agent["idle_since"] == nil {
		t.Error("expected idle_since")
	}
}

func TestSetIdle(t *testing.T) {
	fabRoot := setupFabRoot(t)
	folder := "260310-abcd-my-change"

	if err := SetIdle(fabRoot, folder); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rtPath := FilePath(fabRoot)
	m, err := LoadFile(rtPath)
	if err != nil {
		t.Fatalf("failed to load runtime file: %v", err)
	}

	entry, ok := m[folder].(map[string]interface{})
	if !ok {
		t.Fatal("expected folder entry")
	}
	agent, ok := entry["agent"].(map[string]interface{})
	if !ok {
		t.Fatal("expected agent entry")
	}
	if agent["idle_since"] == nil {
		t.Error("expected idle_since to be set")
	}
}

func TestClearIdle_FileExists(t *testing.T) {
	fabRoot := setupFabRoot(t)
	folder := "260310-abcd-my-change"

	// Set idle first
	if err := SetIdle(fabRoot, folder); err != nil {
		t.Fatalf("SetIdle failed: %v", err)
	}

	// Clear idle
	if err := ClearIdle(fabRoot, folder); err != nil {
		t.Fatalf("ClearIdle failed: %v", err)
	}

	rtPath := FilePath(fabRoot)
	m, err := LoadFile(rtPath)
	if err != nil {
		t.Fatalf("failed to load runtime file: %v", err)
	}

	// Folder entry should be gone (empty folder entry is deleted)
	if _, ok := m[folder]; ok {
		t.Error("expected folder entry to be removed after clearing only agent block")
	}
}

func TestClearIdle_FileNotExists(t *testing.T) {
	fabRoot := setupFabRoot(t)
	folder := "260310-abcd-my-change"

	// Should not create the file
	if err := ClearIdle(fabRoot, folder); err != nil {
		t.Fatalf("ClearIdle failed: %v", err)
	}

	rtPath := FilePath(fabRoot)
	if _, err := os.Stat(rtPath); !os.IsNotExist(err) {
		t.Error("expected runtime file to not exist")
	}
}

func TestClearIdle_Idempotent(t *testing.T) {
	fabRoot := setupFabRoot(t)
	folder := "260310-abcd-my-change"

	// Set idle and clear twice
	if err := SetIdle(fabRoot, folder); err != nil {
		t.Fatalf("SetIdle failed: %v", err)
	}
	if err := ClearIdle(fabRoot, folder); err != nil {
		t.Fatalf("first ClearIdle failed: %v", err)
	}
	if err := ClearIdle(fabRoot, folder); err != nil {
		t.Fatalf("second ClearIdle failed: %v", err)
	}
}
