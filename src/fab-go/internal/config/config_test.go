package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithStageHooks(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
project:
  name: "test"
stage_hooks:
  review:
    pre: "cargo test"
    post: "cargo clippy -- -D warnings"
  apply:
    pre: "./scripts/pre-apply.sh"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.StageHooks) != 2 {
		t.Fatalf("expected 2 stage hooks, got %d", len(cfg.StageHooks))
	}

	review := cfg.GetStageHook("review")
	if review.Pre != "cargo test" {
		t.Errorf("review.pre = %q, want %q", review.Pre, "cargo test")
	}
	if review.Post != "cargo clippy -- -D warnings" {
		t.Errorf("review.post = %q, want %q", review.Post, "cargo clippy -- -D warnings")
	}

	apply := cfg.GetStageHook("apply")
	if apply.Pre != "./scripts/pre-apply.sh" {
		t.Errorf("apply.pre = %q, want %q", apply.Pre, "./scripts/pre-apply.sh")
	}
	if apply.Post != "" {
		t.Errorf("apply.post = %q, want empty", apply.Post)
	}
}

func TestLoad_NoStageHooks(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
project:
  name: "test"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	hook := cfg.GetStageHook("review")
	if hook.Pre != "" || hook.Post != "" {
		t.Errorf("expected empty hook, got pre=%q post=%q", hook.Pre, hook.Post)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}

	if len(cfg.StageHooks) != 0 {
		t.Errorf("expected empty stage hooks, got %d", len(cfg.StageHooks))
	}
}

func TestGetStageHook_NilConfig(t *testing.T) {
	var cfg *Config
	hook := cfg.GetStageHook("review")
	if hook.Pre != "" || hook.Post != "" {
		t.Errorf("expected empty hook from nil config")
	}
}
