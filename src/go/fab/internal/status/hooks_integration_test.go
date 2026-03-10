package status

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunStageHook_PreHookBlocks(t *testing.T) {
	dir := t.TempDir()
	fabRoot := dir
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
stage_hooks:
  review:
    pre: "false"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	err := runStageHook(fabRoot, "review", "pre")
	if err == nil {
		t.Error("pre hook with 'false' should return error")
	}
}

func TestRunStageHook_PostHookSucceeds(t *testing.T) {
	dir := t.TempDir()
	fabRoot := dir
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
stage_hooks:
  apply:
    post: "true"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	err := runStageHook(fabRoot, "apply", "post")
	if err != nil {
		t.Errorf("post hook with 'true' should succeed: %v", err)
	}
}

func TestRunStageHook_NoHookConfigured(t *testing.T) {
	dir := t.TempDir()
	fabRoot := dir
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
project:
  name: "test"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	err := runStageHook(fabRoot, "review", "pre")
	if err != nil {
		t.Errorf("no hook configured should return nil: %v", err)
	}
}

func TestRunStageHook_MissingConfigFile(t *testing.T) {
	dir := t.TempDir()

	err := runStageHook(dir, "review", "pre")
	if err != nil {
		t.Errorf("missing config should return nil: %v", err)
	}
}
