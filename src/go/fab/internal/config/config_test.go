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

func TestLoad_WidenedKeys(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	content := `fab_version: 1.2.3
branch_prefix: "feature/"
agent:
  spawn_command: "claude --effort high"
project:
  name: test
  linear_workspace: acme
`
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(content), 0o644)

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetFabVersion(); got != "1.2.3" {
		t.Errorf("GetFabVersion = %q, want %q", got, "1.2.3")
	}
	if got := cfg.GetBranchPrefix(); got != "feature/" {
		t.Errorf("GetBranchPrefix = %q, want %q", got, "feature/")
	}
	if got := cfg.GetSpawnCommand(); got != "claude --effort high" {
		t.Errorf("GetSpawnCommand = %q, want %q", got, "claude --effort high")
	}
	if got := cfg.GetLinearWorkspace(); got != "acme" {
		t.Errorf("GetLinearWorkspace = %q, want %q", got, "acme")
	}
}

func TestAccessors_NilConfig(t *testing.T) {
	var cfg *Config
	if cfg.GetBranchPrefix() != "" || cfg.GetFabVersion() != "" ||
		cfg.GetSpawnCommand() != "" || cfg.GetLinearWorkspace() != "" {
		t.Error("nil-config accessors must all return empty strings")
	}
}

func TestAccessors_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	if cfg.GetBranchPrefix() != "" || cfg.GetFabVersion() != "" ||
		cfg.GetSpawnCommand() != "" || cfg.GetLinearWorkspace() != "" {
		t.Error("empty-config accessors must all return empty strings")
	}
}

func TestLoadPath_MissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := LoadPath(filepath.Join(t.TempDir(), "nope", "config.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error, got: %v", err)
	}
	if cfg.GetSpawnCommand() != "" {
		t.Error("missing file must yield empty config")
	}
}

// TestLoadPath_MalformedCoupledFailure records the deliberate coupled-failure
// semantic of the consolidated parser (260612-ye8r): a yaml type error on ANY
// modeled key fails the single Unmarshal, so every accessor falls back. The
// nil-safe accessors make this safe for callers that ignore the Load error.
func TestLoadPath_MalformedCoupledFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// branch_prefix has a type error (mapping where a scalar is expected);
	// agent.spawn_command is perfectly fine — but the single Unmarshal fails.
	content := `branch_prefix:
  oops: true
agent:
  spawn_command: "claude"
`
	os.WriteFile(path, []byte(content), 0o644)

	cfg, err := LoadPath(path)
	if err == nil {
		t.Fatal("expected a parse error for the malformed key")
	}
	if cfg != nil {
		t.Fatal("malformed config must return nil *Config")
	}
	// Nil-safe accessors deliver the documented fallbacks.
	if cfg.GetSpawnCommand() != "" {
		t.Error("nil-safe accessor must return the empty fallback")
	}
}
