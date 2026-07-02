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
providers:
  claude:
    session_command: "claude --effort high"
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
	prov, ok := cfg.GetProvider("claude")
	if !ok {
		t.Fatal("expected a 'claude' provider entry")
	}
	if prov.SessionCommand != "claude --effort high" {
		t.Errorf("claude.session_command = %q, want %q", prov.SessionCommand, "claude --effort high")
	}
	if got := cfg.GetLinearWorkspace(); got != "acme" {
		t.Errorf("GetLinearWorkspace = %q, want %q", got, "acme")
	}
}

// TestLoad_WithProviders: the top-level providers table round-trips both command
// fields, and a provider with only a session_command yields an empty
// DispatchCommand (the native-dispatch signal). The accessor is a pure
// pass-through; the built-in merge is internal/agent's job.
func TestLoad_WithProviders(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions'
  codex:
    session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	claude, ok := cfg.GetProvider("claude")
	if !ok {
		t.Fatal("expected a 'claude' provider")
	}
	if claude.SessionCommand != "claude --dangerously-skip-permissions" {
		t.Errorf("claude.SessionCommand = %q", claude.SessionCommand)
	}
	if claude.DispatchCommand != "" {
		t.Errorf("claude.DispatchCommand = %q, want empty (native dispatch)", claude.DispatchCommand)
	}

	codex, ok := cfg.GetProvider("codex")
	if !ok {
		t.Fatal("expected a 'codex' provider")
	}
	if codex.SessionCommand != "codex -m {model} -c model_reasoning_effort={effort}" {
		t.Errorf("codex.SessionCommand = %q", codex.SessionCommand)
	}
	if codex.DispatchCommand != "codex exec -m {model} -c model_reasoning_effort={effort}" {
		t.Errorf("codex.DispatchCommand = %q", codex.DispatchCommand)
	}

	// An unconfigured provider reports no entry.
	if _, ok := cfg.GetProvider("gemini"); ok {
		t.Error("expected no entry for the unconfigured 'gemini' provider")
	}
}

func TestGetProvider_NilAndEmptyConfig(t *testing.T) {
	var nilCfg *Config
	if _, ok := nilCfg.GetProvider("claude"); ok {
		t.Error("nil-config GetProvider must report no entry")
	}
	empty := &Config{}
	if _, ok := empty.GetProvider("claude"); ok {
		t.Error("empty-config GetProvider must report no entry")
	}
}

func TestLoad_WithAgentTiers(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  claude:
    session_command: "claude --effort xhigh"
agent:
  tiers:
    doing: { provider: claude, model: claude-sonnet-5, effort: medium }
    fast: { effort: low }
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	doing, ok := cfg.GetAgentTier("doing")
	if !ok {
		t.Fatal("expected a 'doing' tier override")
	}
	if doing.Provider != "claude" || doing.Model != "claude-sonnet-5" || doing.Effort != "medium" {
		t.Errorf("doing = %+v, want {claude claude-sonnet-5 medium}", doing)
	}

	// A partial override (only effort set) round-trips with empty provider/model —
	// the per-field merge over the default tier is internal/agent's job, not the
	// accessor's.
	fast, ok := cfg.GetAgentTier("fast")
	if !ok {
		t.Fatal("expected a 'fast' tier override")
	}
	if fast.Provider != "" || fast.Model != "" || fast.Effort != "low" {
		t.Errorf("fast = %+v, want {<empty> <empty> low}", fast)
	}

	// An unconfigured tier reports no override.
	if _, ok := cfg.GetAgentTier("review"); ok {
		t.Error("expected no override for the unconfigured 'review' tier")
	}

	// providers still parse alongside the tiers block.
	prov, ok := cfg.GetProvider("claude")
	if !ok || prov.SessionCommand != "claude --effort xhigh" {
		t.Errorf("claude provider = %+v, ok=%v, want session_command 'claude --effort xhigh'", prov, ok)
	}
}

func TestLoad_NoAgentTiers(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	// A config with no agent.tiers block must load cleanly (yaml ignores
	// unknown keys; widening AgentConfig is free for existing configs).
	configYAML := `
providers:
  claude:
    session_command: "claude"
project:
  name: "test"
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if _, ok := cfg.GetAgentTier("doing"); ok {
		t.Error("expected no tier override when agent.tiers is absent")
	}
}

func TestGetAgentTier_NilAndEmptyConfig(t *testing.T) {
	var nilCfg *Config
	if _, ok := nilCfg.GetAgentTier("doing"); ok {
		t.Error("nil-config GetAgentTier must report no override")
	}
	empty := &Config{}
	if _, ok := empty.GetAgentTier("doing"); ok {
		t.Error("empty-config GetAgentTier must report no override")
	}
}

func TestAccessors_NilConfig(t *testing.T) {
	var cfg *Config
	if cfg.GetBranchPrefix() != "" || cfg.GetFabVersion() != "" ||
		cfg.GetLinearWorkspace() != "" {
		t.Error("nil-config accessors must all return empty strings")
	}
	if _, ok := cfg.GetProvider("claude"); ok {
		t.Error("nil-config GetProvider must report no entry")
	}
}

func TestAccessors_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	if cfg.GetBranchPrefix() != "" || cfg.GetFabVersion() != "" ||
		cfg.GetLinearWorkspace() != "" {
		t.Error("empty-config accessors must all return empty strings")
	}
	if _, ok := cfg.GetProvider("claude"); ok {
		t.Error("empty-config GetProvider must report no entry")
	}
}

func TestLoadPath_MissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := LoadPath(filepath.Join(t.TempDir(), "nope", "config.yaml"))
	if err != nil {
		t.Fatalf("missing file must not error, got: %v", err)
	}
	if _, ok := cfg.GetProvider("claude"); ok {
		t.Error("missing file must yield empty config (no providers)")
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
	// providers is perfectly fine — but the single Unmarshal fails.
	content := `branch_prefix:
  oops: true
providers:
  claude:
    session_command: "claude"
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
	if _, ok := cfg.GetProvider("claude"); ok {
		t.Error("nil-safe accessor must report no entry")
	}
}
