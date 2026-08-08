package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// isolateSystemConfig points HOME at an empty temp dir so a test never picks up
// the developer's real ~/.fab-kit/config.yaml. The cascade (added in lpb5) reads
// the system layer at every Load/LoadPath, so tests that assert on the
// project-only result MUST isolate the system layer first. os.UserHomeDir honors
// $HOME on unix, so t.Setenv is the seam. Returns the fake home for tests that
// want to WRITE a system config under it.
func isolateSystemConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range configscope.DottedKeys() {
		t.Setenv(envNameForKey(key), "")
	}
	return home
}

func TestLoad_WithStageHooks(t *testing.T) {
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	content := `branch_prefix: "feature/"
providers:
  claude:
    session_command: "claude --effort high"
project:
  name: test
  linear_workspace: acme
`
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(content), 0o644)
	// The version pin is the plain-text sibling (config.yaml fab_version: is no
	// longer parsed).
	os.WriteFile(filepath.Join(fabRoot, ".fab-version"), []byte("1.2.3\n"), 0o644)

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

// TestLoad_FabVersionFromDotFile pins the 260708-j0qm relocation: Load reads
// fab_version from the plain-text sibling fab/.fab-version FIRST, and the value
// there wins over any (legacy) fab_version: key still in config.yaml.
func TestLoad_FabVersionFromDotFile(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	// A stale fab_version in config.yaml AND a .fab-version sibling: .fab-version wins.
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"),
		[]byte("fab_version: 1.0.0\nproject:\n  name: t\n"), 0o644)
	os.WriteFile(filepath.Join(fabRoot, ".fab-version"), []byte("2.15.0\n"), 0o644)

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetFabVersion(); got != "2.15.0" {
		t.Errorf("GetFabVersion = %q, want %q (.fab-version wins over the config.yaml key)", got, "2.15.0")
	}
}

// TestLoad_FabVersionConfigKeyIgnored pins the sole-source behavior (260719-kq7v):
// with no fab/.fab-version, a stale fab_version: key in config.yaml is an inert
// unknown key — Config.FabVersion is tagged `yaml:"-"`, so GetFabVersion returns ""
// with no error (the config.yaml key is never parsed).
func TestLoad_FabVersionConfigKeyIgnored(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"),
		[]byte("fab_version: 2.14.0\nproject:\n  name: t\n"), 0o644)
	// No .fab-version file.

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetFabVersion(); got != "" {
		t.Errorf("GetFabVersion = %q, want \"\" (config.yaml fab_version: is no longer parsed)", got)
	}
}

// TestLoad_WithProviders: the top-level providers table round-trips both command
// fields and the native capability independently. The accessor is a pure
// pass-through; the built-in merge is internal/agent's job.
func TestLoad_WithProviders(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  claude:
    native: true
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
		t.Errorf("claude.DispatchCommand = %q, want empty (not configured in this fixture)", claude.DispatchCommand)
	}
	if !claude.Native {
		t.Error("claude.Native = false, want true")
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
	if codex.Native {
		t.Error("codex.Native = true, want false when omitted")
	}

	// An unconfigured provider reports no entry.
	if _, ok := cfg.GetProvider("gemini"); ok {
		t.Error("expected no entry for the unconfigured 'gemini' provider")
	}
}

// TestLoad_WithProviderFill (260805-j3cm): the per-provider default-fill fields
// `model`/`effort` parse off a provider entry, independently of the command fields
// (a provider entry MAY carry fill only — the commands then inherit the built-in in
// internal/agent's merge).
func TestLoad_WithProviderFill(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  codex:
    model: gpt-5.3-codex
    effort: high
  gemini:
    model: gemini-2.5-pro
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	codex, ok := cfg.GetProvider("codex")
	if !ok {
		t.Fatal("expected a 'codex' provider entry (fill-only is a valid entry)")
	}
	if codex.Model != "gpt-5.3-codex" || codex.Effort != "high" {
		t.Errorf("codex fill = {%q, %q}, want {gpt-5.3-codex, high}", codex.Model, codex.Effort)
	}
	if codex.SessionCommand != "" || codex.DispatchCommand != "" {
		t.Errorf("codex commands = {%q, %q}, want empty here (the built-in merge is internal/agent's job)", codex.SessionCommand, codex.DispatchCommand)
	}

	// Effort may be omitted independently (gemini has no reasoning-effort knob).
	gemini, ok := cfg.GetProvider("gemini")
	if !ok {
		t.Fatal("expected a 'gemini' provider entry")
	}
	if gemini.Model != "gemini-2.5-pro" || gemini.Effort != "" {
		t.Errorf("gemini fill = {%q, %q}, want {gemini-2.5-pro, \"\"}", gemini.Model, gemini.Effort)
	}
}

// TestCascade_ProviderFillFromSystemLayer (260805-j3cm): `providers` is scope
// `both`, so a machine-wide fill set once in ~/.fab-kit/config.yaml reaches every
// repo — and a project entry per-key deep-merges over it (the fill value survives a
// project entry that only sets a command).
func TestCascade_ProviderFillFromSystemLayer(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
providers:
  codex:
    model: gpt-5.3-codex
    effort: high
`)
	fabRoot := writeProjectConfig(t, `
providers:
  codex:
    dispatch_command: 'codex exec --json'
`)
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	codex, ok := cfg.GetProvider("codex")
	if !ok {
		t.Fatal("expected a merged 'codex' provider entry")
	}
	if codex.Model != "gpt-5.3-codex" || codex.Effort != "high" {
		t.Errorf("codex fill = {%q, %q}, want the system layer's {gpt-5.3-codex, high}", codex.Model, codex.Effort)
	}
	if codex.DispatchCommand != "codex exec --json" {
		t.Errorf("codex dispatch_command = %q, want the project layer's override", codex.DispatchCommand)
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

func TestLoad_WithAgentProfiles(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  claude:
    session_command: "claude --effort high"
agent:
  profiles:
    doing: { provider: claude, model: claude-sonnet-5, effort: medium }
    fast: { effort: low }
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	doing, ok := cfg.GetAgentProfile("doing")
	if !ok {
		t.Fatal("expected a 'doing' role override")
	}
	if doing.Provider != "claude" || doing.Model != "claude-sonnet-5" || doing.Effort != "medium" {
		t.Errorf("doing = %+v, want {claude claude-sonnet-5 medium}", doing)
	}

	// A partial override (only effort set) round-trips with empty provider/model —
	// continuing down the fill precedence is internal/agent's job, not the
	// accessor's.
	fast, ok := cfg.GetAgentProfile("fast")
	if !ok {
		t.Fatal("expected a 'fast' role override")
	}
	if fast.Provider != "" || fast.Model != "" || fast.Effort != "low" {
		t.Errorf("fast = %+v, want {<empty> <empty> low}", fast)
	}

	// An unconfigured role reports no override.
	if _, ok := cfg.GetAgentProfile("review"); ok {
		t.Error("expected no override for the unconfigured 'review' role")
	}

	// providers still parse alongside the profiles block.
	prov, ok := cfg.GetProvider("claude")
	if !ok || prov.SessionCommand != "claude --effort high" {
		t.Errorf("claude provider = %+v, ok=%v, want session_command 'claude --effort high'", prov, ok)
	}
}

// TestLoad_AgentDepthKnobs: the two advertised knobs parse as plain scalars and
// read back through their nil-safe accessors.
func TestLoad_AgentDepthKnobs(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
agent:
  session: claude
  workers: gemini
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got := cfg.GetAgentSession(); got != "claude" {
		t.Errorf("agent.session = %q, want claude", got)
	}
	if got := cfg.GetAgentWorkers(); got != "gemini" {
		t.Errorf("agent.workers = %q, want gemini", got)
	}
}

// TestGetAgentProfile_LegacyTiersAlias: `agent.tiers` is the deprecated spelling
// of `agent.profiles`, consulted PER ROLE — so a half-migrated config (some roles
// moved, some not) resolves every role, and `profiles` wins wherever both carry
// the same role.
func TestGetAgentProfile_LegacyTiersAlias(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
agent:
  profiles:
    doing: { model: from-profiles }
  tiers:
    doing: { model: from-tiers }
    review: { effort: medium }
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// profiles wins where both carry the role.
	doing, ok := cfg.GetAgentProfile("doing")
	if !ok || doing.Model != "from-profiles" {
		t.Errorf("doing.model = %q ok=%v, want from-profiles (profiles beats the legacy tiers alias)", doing.Model, ok)
	}
	// A role only the legacy spelling carries still resolves.
	review, ok := cfg.GetAgentProfile("review")
	if !ok || review.Effort != "medium" {
		t.Errorf("review = %+v ok=%v, want effort medium from the legacy tiers alias", review, ok)
	}
}

// TestLoad_ProviderProfiles: `providers.<name>.profiles.<role>` parses as a
// per-role fill map, and the DEPRECATED flat `model`/`effort` fill still parses
// alongside it (the pre-2.17.0 spelling internal/agent reads below
// profiles.default).
func TestLoad_ProviderProfiles(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	configYAML := `
providers:
  codex:
    session_command: "codex"
    profiles:
      default: { model: codex-default, effort: medium }
      review: { model: codex-review, effort: high }
  gemini:
    model: flat-model
    effort: flat-effort
`
	os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(configYAML), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	codex, ok := cfg.GetProvider("codex")
	if !ok {
		t.Fatal("expected a codex provider entry")
	}
	if got := codex.Profiles["review"]; got.Model != "codex-review" || got.Effort != "high" {
		t.Errorf("codex.profiles.review = %+v, want {codex-review high}", got)
	}
	if got := codex.Profiles["default"]; got.Model != "codex-default" || got.Effort != "medium" {
		t.Errorf("codex.profiles.default = %+v, want {codex-default medium}", got)
	}

	// The deprecated flat fill still parses into its own fields at load time; the
	// ALIAS semantics are applied downstream, where internal/agent.ResolveProvider
	// folds it into this override's profiles.default per field.
	gemini, ok := cfg.GetProvider("gemini")
	if !ok {
		t.Fatal("expected a gemini provider entry")
	}
	if gemini.Model != "flat-model" || gemini.Effort != "flat-effort" {
		t.Errorf("gemini flat fill = {%s %s}, want {flat-model flat-effort}", gemini.Model, gemini.Effort)
	}
}

func TestLoad_NoAgentProfiles(t *testing.T) {
	isolateSystemConfig(t)
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "project")
	os.MkdirAll(projectDir, 0o755)

	// A config with no agent block must load cleanly (yaml ignores unknown keys;
	// widening AgentConfig is free for existing configs).
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
	if _, ok := cfg.GetAgentProfile("doing"); ok {
		t.Error("expected no role override when agent.profiles is absent")
	}
	if cfg.GetAgentSession() != "" || cfg.GetAgentWorkers() != "" {
		t.Error("expected empty depth knobs when the agent block is absent")
	}
}

func TestGetAgentProfile_NilAndEmptyConfig(t *testing.T) {
	var nilCfg *Config
	if _, ok := nilCfg.GetAgentProfile("doing"); ok {
		t.Error("nil-config GetAgentProfile must report no override")
	}
	if nilCfg.GetAgentSession() != "" || nilCfg.GetAgentWorkers() != "" {
		t.Error("nil-config depth-knob accessors must return empty strings")
	}
	empty := &Config{}
	if _, ok := empty.GetAgentProfile("doing"); ok {
		t.Error("empty-config GetAgentProfile must report no override")
	}
	if empty.GetAgentSession() != "" || empty.GetAgentWorkers() != "" {
		t.Error("empty-config depth-knob accessors must return empty strings")
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
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
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

// --- Cascade (lpb5): project > system (~/.fab-kit/config.yaml) > defaults ---

// captureWarnings redirects the loader's warning writer for the duration of the
// test and returns a function yielding what was written. The fail-open scope +
// malformed-file warnings go through warnw (os.Stderr in production).
func captureWarnings(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	prev := warnw
	warnw = &buf
	t.Cleanup(func() { warnw = prev })
	return buf.String
}

// writeSystemConfig writes a ~/.fab-kit/config.yaml under the isolated fake home
// and returns its path.
func writeSystemConfig(t *testing.T, home, content string) string {
	t.Helper()
	dir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeProjectConfig writes a project fab/project/config.yaml under dir and
// returns the fabRoot (dir).
func writeProjectConfig(t *testing.T, content string) string {
	t.Helper()
	fabRoot := t.TempDir()
	projectDir := filepath.Join(fabRoot, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return fabRoot
}

// TestCascade_MapsMergePerKey: agent.profiles merges per-key across the two
// files — a project role field and a system role field compose, project wins on a
// conflicting leaf, and a system-only role survives alongside a project-only one.
func TestCascade_MapsMergePerKey(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
agent:
  profiles:
    doing: { provider: claude, model: system-model, effort: low }
    sysonly: { model: sys-only-model }
`)
	fabRoot := writeProjectConfig(t, `
agent:
  profiles:
    doing: { model: project-model }
    projonly: { model: proj-only-model }
`)

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// doing.model: project wins (project-model); doing.effort inherited from
	// system (low); doing.provider inherited from system (claude).
	doing, ok := cfg.GetAgentProfile("doing")
	if !ok {
		t.Fatal("expected a merged 'doing' role profile")
	}
	if doing.Model != "project-model" {
		t.Errorf("doing.model = %q, want project-model (project wins)", doing.Model)
	}
	if doing.Effort != "low" {
		t.Errorf("doing.effort = %q, want low (inherited from system layer)", doing.Effort)
	}
	if doing.Provider != "claude" {
		t.Errorf("doing.provider = %q, want claude (inherited from system layer)", doing.Provider)
	}

	// A system-only role survives (per-key merge, not whole-map replacement).
	if sysonly, ok := cfg.GetAgentProfile("sysonly"); !ok || sysonly.Model != "sys-only-model" {
		t.Errorf("system-only role lost in merge: %+v ok=%v", sysonly, ok)
	}
	// A project-only role survives alongside it.
	if projonly, ok := cfg.GetAgentProfile("projonly"); !ok || projonly.Model != "proj-only-model" {
		t.Errorf("project-only role lost in merge: %+v ok=%v", projonly, ok)
	}
}

// TestCascade_ScalarReplaceProjectWins: a scalar set in both layers takes the
// project value (providers.claude.session_command is `both`-scoped, so it is a
// valid system-file override to exercise end-to-end). A system-only provider
// entry survives alongside it (per-key map merge). The list-replace rule is
// exercised separately in TestCascade_ListReplace (lists are project-scoped, so
// the generic rule is asserted on the merge helper directly).
func TestCascade_ScalarReplaceProjectWins(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
providers:
  claude:
    session_command: system-session
  codex:
    dispatch_command: codex exec
`)
	fabRoot := writeProjectConfig(t, `
providers:
  claude:
    session_command: project-session
`)

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	claude, ok := cfg.GetProvider("claude")
	if !ok || claude.SessionCommand != "project-session" {
		t.Errorf("claude.session_command = %q ok=%v, want project-session (scalar: project wins)", claude.SessionCommand, ok)
	}
	// System-only provider survives (per-key map merge).
	if codex, ok := cfg.GetProvider("codex"); !ok || codex.DispatchCommand != "codex exec" {
		t.Errorf("system-only codex provider lost: %+v ok=%v", codex, ok)
	}
}

// TestCascade_ListReplace: a list present in BOTH layers is replaced wholesale by
// the higher layer (never concatenated). test_paths is project-scoped, so to
// exercise the generic list-replace merge rule we build the layers directly via
// deepMerge (the rule is field-agnostic — it operates on decoded YAML values,
// before scope pruning or unmarshal).
func TestCascade_ListReplace(t *testing.T) {
	base := map[string]any{"xs": []any{"a", "b", "c"}}
	over := map[string]any{"xs": []any{"z"}}
	merged := deepMerge(base, over)
	got, _ := merged["xs"].([]any)
	if len(got) != 1 || got[0] != "z" {
		t.Errorf("list merge = %v, want [z] (lists replace, never concatenate)", merged["xs"])
	}
}

// TestCascade_AbsentSystemFile: with no ~/.fab-kit/config.yaml, Load returns a
// result byte-identical to the pre-cascade single-file parse (no error, no
// warning), and the project values are intact.
func TestCascade_AbsentSystemFile(t *testing.T) {
	isolateSystemConfig(t) // empty fake home ⇒ no system file
	warnings := captureWarnings(t)
	fabRoot := writeProjectConfig(t, `
branch_prefix: "feature/"
providers:
  claude:
    session_command: only-project
`)
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GetBranchPrefix() != "feature/" {
		t.Errorf("branch_prefix = %q, want feature/", cfg.GetBranchPrefix())
	}
	if claude, ok := cfg.GetProvider("claude"); !ok || claude.SessionCommand != "only-project" {
		t.Errorf("claude.session_command = %+v ok=%v, want only-project", claude, ok)
	}
	if w := warnings(); w != "" {
		t.Errorf("absent system file must emit no warning, got %q", w)
	}
}

// TestCascade_MalformedSystemFileFailsOpen: a malformed system file warns on
// stderr and is SKIPPED — the project-over-defaults result is returned with no
// error. Fail-open: a broken personal file must not brick the repo.
func TestCascade_MalformedSystemFileFailsOpen(t *testing.T) {
	home := isolateSystemConfig(t)
	warnings := captureWarnings(t)
	writeSystemConfig(t, home, "this: is: not: valid: yaml: [[[\n")
	fabRoot := writeProjectConfig(t, `
providers:
  claude:
    session_command: project-wins
`)
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("malformed system file must be fail-open (no error), got: %v", err)
	}
	if claude, ok := cfg.GetProvider("claude"); !ok || claude.SessionCommand != "project-wins" {
		t.Errorf("project layer must survive a skipped system layer: %+v ok=%v", claude, ok)
	}
	if w := warnings(); !strings.Contains(w, "fab: warning:") || !strings.Contains(w, "malformed system config") {
		t.Errorf("expected a fail-open malformed-system warning, got %q", w)
	}
}

// TestCascade_MalformedProjectFileStillErrors: a malformed PROJECT file keeps
// today's error behavior — the parse error is returned (only the system layer is
// fail-open).
func TestCascade_MalformedProjectFileStillErrors(t *testing.T) {
	isolateSystemConfig(t)
	// A type error on a modeled key surfaces at the final unmarshal into Config.
	fabRoot := writeProjectConfig(t, "branch_prefix:\n  oops: true\n")
	if _, err := Load(fabRoot); err == nil {
		t.Fatal("a malformed project file must still return an error (not fail-open)")
	}
}

// TestCascade_ProjectAbsentSystemPresent: with no project file but a system file
// present, the system layer alone forms the effective config (the system layer is
// user-global and applies even where there is no project config).
func TestCascade_ProjectAbsentSystemPresent(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
providers:
  claude:
    session_command: from-system
`)
	fabRoot := t.TempDir() // no project/config.yaml written
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if claude, ok := cfg.GetProvider("claude"); !ok || claude.SessionCommand != "from-system" {
		t.Errorf("system layer must apply with no project file: %+v ok=%v", claude, ok)
	}
}

// --- Environment override layer: env > project > system > defaults ---

func TestParseYAMLValue(t *testing.T) {
	t.Run("scalar", func(t *testing.T) {
		got, err := ParseYAMLValue("codex")
		if err != nil || got != "codex" {
			t.Fatalf("ParseYAMLValue scalar = %#v, %v; want codex, nil", got, err)
		}
	})
	t.Run("flow map", func(t *testing.T) {
		got, err := ParseYAMLValue("{review: {provider: codex}}")
		if err != nil {
			t.Fatalf("ParseYAMLValue flow map: %v", err)
		}
		outer, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("flow map type = %T, want map[string]any", got)
		}
		review, _ := outer["review"].(map[string]any)
		if review["provider"] != "codex" {
			t.Errorf("flow map = %#v, want review.provider=codex", got)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if _, err := ParseYAMLValue("{not-closed"); err == nil {
			t.Fatal("invalid YAML must return an error")
		}
	})
}

func TestEnvNameForKey(t *testing.T) {
	for key, want := range map[string]string{
		"agent.workers":         "FAB_AGENT_WORKERS",
		"dispatch.column_width": "FAB_DISPATCH_COLUMN_WIDTH",
	} {
		if got := envNameForKey(key); got != want {
			t.Errorf("envNameForKey(%q) = %q, want %q", key, got, want)
		}
	}
}

// TestEnvCascade_PrecedenceAndMapMerge covers the motivating path end-to-end:
// a scalar env override beats project/system, while a map-valued env row merges
// per key with non-conflicting leaves from both file layers.
func TestEnvCascade_PrecedenceAndMapMerge(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
agent:
  workers: gemini
  profiles:
    review: { model: system-review, effort: low }
    hydrate: { effort: medium }
`)
	fabRoot := writeProjectConfig(t, `
agent:
  workers: claude
  profiles:
    review: { model: project-review }
    doing: { effort: high }
`)
	t.Setenv("FAB_AGENT_WORKERS", "codex")
	t.Setenv("FAB_AGENT_PROFILES", "{review: {provider: codex}}")

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetAgentWorkers(); got != "codex" {
		t.Errorf("agent.workers = %q, want codex (env wins)", got)
	}
	review, ok := cfg.GetAgentProfile("review")
	if !ok {
		t.Fatal("merged review profile missing")
	}
	if review.Provider != "codex" || review.Model != "project-review" || review.Effort != "low" {
		t.Errorf("review profile = %+v, want env provider + project model + system effort", review)
	}
	if doing, ok := cfg.GetAgentProfile("doing"); !ok || doing.Effort != "high" {
		t.Errorf("project-only doing profile lost under env map merge: %+v ok=%v", doing, ok)
	}
	if hydrate, ok := cfg.GetAgentProfile("hydrate"); !ok || hydrate.Effort != "medium" {
		t.Errorf("system-only hydrate profile lost under env map merge: %+v ok=%v", hydrate, ok)
	}
}

func TestEnvCascade_ProjectScopedWarnedAndUnknownIgnored(t *testing.T) {
	isolateSystemConfig(t)
	warnings := captureWarnings(t)
	fabRoot := writeProjectConfig(t, "source_paths:\n  - project-src/\n")
	t.Setenv("FAB_SOURCE_PATHS", "[private-src/]")
	t.Setenv("FAB_SOMETHING_UNKNOWN", "true")

	layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if len(layers.Env) != 0 {
		t.Errorf("project-scoped env var must not enter env layer: %#v", layers.Env)
	}
	if got := layers.Effective["source_paths"]; fmt.Sprint(got) != "[project-src/]" {
		t.Errorf("effective source_paths = %v, want project value", got)
	}
	w := warnings()
	if !strings.Contains(w, "project-scoped environment override $FAB_SOURCE_PATHS") {
		t.Errorf("scope warning must name FAB_SOURCE_PATHS, got %q", w)
	}
	if strings.Contains(w, "FAB_SOMETHING_UNKNOWN") {
		t.Errorf("unknown FAB_* variables are not scanned and must not warn, got %q", w)
	}
}

func TestEnvCascade_MalformedFailsOpenAndEmptyIsUnset(t *testing.T) {
	isolateSystemConfig(t)
	warnings := captureWarnings(t)
	fabRoot := writeProjectConfig(t, "agent:\n  workers: claude\n")

	t.Setenv("FAB_AGENT_WORKERS", "{not-closed")
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("malformed env value must fail open, got: %v", err)
	}
	if got := cfg.GetAgentWorkers(); got != "claude" {
		t.Errorf("malformed env value must leave project value effective, got %q", got)
	}
	if w := warnings(); !strings.Contains(w, "malformed environment override $FAB_AGENT_WORKERS") {
		t.Errorf("malformed env warning missing variable name: %q", w)
	}

	// Empty is the shell-level unset convention: no warning and no null override.
	t.Setenv("FAB_AGENT_WORKERS", "")
	warnings = captureWarnings(t)
	cfg, err = Load(fabRoot)
	if err != nil {
		t.Fatalf("empty env value: %v", err)
	}
	if got := cfg.GetAgentWorkers(); got != "claude" {
		t.Errorf("empty env value must behave as unset, got %q", got)
	}
	if w := warnings(); w != "" {
		t.Errorf("empty env value must not warn, got %q", w)
	}
}

// TestEnvCascade_BlockStyleCollectionResolves pins the env layer's parity with
// the file layers: a both-scoped variable carrying a BLOCK-style YAML collection
// resolves like any other override. The shared value parser is deliberately
// style-agnostic — `fab config set` refuses collections through its own gates,
// not by narrowing the parser the env layer shares with it.
func TestEnvCascade_BlockStyleCollectionResolves(t *testing.T) {
	isolateSystemConfig(t)
	warnings := captureWarnings(t)
	fabRoot := writeProjectConfig(t, "agent:\n  workers: claude\n")
	t.Setenv("FAB_PROVIDERS", "custom:\n  session_command: tool")

	layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if got := layers.EnvOrigins["providers"]; got != "FAB_PROVIDERS" {
		t.Errorf("providers origin = %q, want FAB_PROVIDERS", got)
	}

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if custom, ok := cfg.GetProvider("custom"); !ok || custom.SessionCommand != "tool" {
		t.Errorf("block-style env collection did not resolve: %+v ok=%v", custom, ok)
	}
	if w := warnings(); w != "" {
		t.Errorf("block-style env collection must not warn, got %q", w)
	}
}

// TestEnvCascade_UnusableScalarWarnsAndSkips: a value that reaches the parser but
// cannot become a config value is skipped with a warning naming the variable —
// never silently. Whitespace-only text is distinct from a bare empty value (the
// shell's unset convention, which is silent), and a bare date resolves to the
// unsupported !!timestamp tag rather than to a string.
func TestEnvCascade_UnusableScalarWarnsAndSkips(t *testing.T) {
	for _, tt := range []struct{ name, envValue string }{
		{"whitespace only", "   "},
		{"timestamp-tagged scalar", "2026-01-01"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			isolateSystemConfig(t)
			warnings := captureWarnings(t)
			fabRoot := writeProjectConfig(t, "agent:\n  workers: claude\n")
			t.Setenv("FAB_AGENT_WORKERS", tt.envValue)

			layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
			if err != nil {
				t.Fatalf("LoadLayers must fail open: %v", err)
			}
			if layers.Env != nil || layers.EnvOrigins != nil {
				t.Fatalf("unusable value must be skipped, got Env=%#v origins=%#v", layers.Env, layers.EnvOrigins)
			}

			cfg, err := Load(fabRoot)
			if err != nil {
				t.Fatalf("config loading must not error: %v", err)
			}
			if got := cfg.GetAgentWorkers(); got != "claude" {
				t.Errorf("unusable env value must leave project value effective, got %q", got)
			}
			if w := warnings(); !strings.Contains(w, "malformed environment override $FAB_AGENT_WORKERS") {
				t.Errorf("warning must name FAB_AGENT_WORKERS, got %q", w)
			}
		})
	}
}

func TestEnvCascade_TypeIncompatibleFailsOpen(t *testing.T) {
	tests := []struct {
		name        string
		envName     string
		envValue    string
		projectYAML string
		assertLower func(*testing.T, *Config)
	}{
		{
			name:        "boolean field rejects string",
			envName:     "FAB_DISPATCH_WATCHABLE",
			envValue:    "not-a-bool",
			projectYAML: "dispatch:\n  watchable: true\n",
			assertLower: func(t *testing.T, cfg *Config) {
				t.Helper()
				if !cfg.GetDispatchWatchable() {
					t.Error("invalid env bool must leave project watchable=true effective")
				}
			},
		},
		{
			name:        "scalar field rejects map",
			envName:     "FAB_AGENT_WORKERS",
			envValue:    "{bogus: value}",
			projectYAML: "agent:\n  workers: claude\n",
			assertLower: func(t *testing.T, cfg *Config) {
				t.Helper()
				if got := cfg.GetAgentWorkers(); got != "claude" {
					t.Errorf("invalid env map must leave project workers effective, got %q", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateSystemConfig(t)
			warnings := captureWarnings(t)
			fabRoot := writeProjectConfig(t, tt.projectYAML)
			t.Setenv(tt.envName, tt.envValue)

			layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
			if err != nil {
				t.Fatalf("LoadLayers must fail open: %v", err)
			}
			if layers.Env != nil || layers.EnvOrigins != nil {
				t.Fatalf("incompatible override must be skipped, got Env=%#v origins=%#v", layers.Env, layers.EnvOrigins)
			}

			cfg, err := Load(fabRoot)
			if err != nil {
				t.Fatalf("config loading must not error: %v", err)
			}
			tt.assertLower(t, cfg)
			if w := warnings(); !strings.Contains(w, "malformed environment override $"+tt.envName) {
				t.Errorf("warning must name %s, got %q", tt.envName, w)
			}
		})
	}
}

func TestEnvCascade_NoVariablesPreservesFileMerge(t *testing.T) {
	home := isolateSystemConfig(t)
	warnings := captureWarnings(t)
	writeSystemConfig(t, home, "agent:\n  workers: gemini\n")
	fabRoot := writeProjectConfig(t, "agent:\n  session: codex\n")

	layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if layers.Env != nil || layers.EnvOrigins != nil {
		t.Errorf("no variables set must yield nil env layer/provenance, got Env=%#v origins=%#v", layers.Env, layers.EnvOrigins)
	}
	agentMap, _ := layers.Effective["agent"].(map[string]any)
	if agentMap["session"] != "codex" || agentMap["workers"] != "gemini" {
		t.Errorf("no-env file merge changed: %#v", layers.Effective)
	}
	if w := warnings(); w != "" {
		t.Errorf("no env variables must emit no warning, got %q", w)
	}
}

func TestEnvCascade_ExplicitNullRemainsPresent(t *testing.T) {
	isolateSystemConfig(t)
	fabRoot := writeProjectConfig(t, "agent:\n  workers: project-worker\n")
	t.Setenv("FAB_AGENT_WORKERS", "null")

	layers, err := LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	envAgent, ok := layers.Env["agent"].(map[string]any)
	if !ok {
		t.Fatalf("environment agent layer missing: %#v", layers.Env)
	}
	if value, present := envAgent["workers"]; !present || value != nil {
		t.Fatalf("environment workers = %#v, present=%v; want explicit nil with presence", value, present)
	}
	effectiveAgent, ok := layers.Effective["agent"].(map[string]any)
	if !ok {
		t.Fatalf("effective agent layer missing: %#v", layers.Effective)
	}
	if value, present := effectiveAgent["workers"]; !present || value != nil {
		t.Fatalf("effective workers = %#v, present=%v; want explicit nil override", value, present)
	}
	if got := layers.EnvOrigins["agent.workers"]; got != "FAB_AGENT_WORKERS" {
		t.Fatalf("environment origin = %q, want FAB_AGENT_WORKERS", got)
	}
}

// --- Scope enforcement (lpb5, decision 6) ---

// TestScope_PruneProjectScopedFromSystem: a project-scoped field placed in the
// system file is pruned (not applied) and a `fab: warning:` names it; a
// both-scoped field (agent.profiles) is honored; an unknown key is ignored
// silently.
func TestScope_PruneProjectScopedFromSystem(t *testing.T) {
	home := isolateSystemConfig(t)
	warnings := captureWarnings(t)
	writeSystemConfig(t, home, `
source_paths:
  - system-only-src/
agent:
  profiles:
    doing: { effort: high }
totally_unknown_key: 42
`)
	fabRoot := writeProjectConfig(t, `
source_paths:
  - project-src/
`)
	projectPath := filepath.Join(fabRoot, "project", "config.yaml")

	// source_paths is skill-consumed (not modeled in Config), so assert on the
	// resolved LAYERS: the system layer must no longer carry source_paths after
	// pruning, and the effective source_paths must be the project's.
	layers, err := LoadLayers(projectPath)
	if err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if _, ok := layers.System["source_paths"]; ok {
		t.Error("source_paths (scope project) must be pruned out of the system layer")
	}
	effSrc, _ := layers.Effective["source_paths"].([]any)
	if len(effSrc) != 1 || effSrc[0] != "project-src/" {
		t.Errorf("effective source_paths = %v, want [project-src/] (project wins; system layer pruned)", layers.Effective["source_paths"])
	}

	// agent.profiles (scope both) from the system file is honored end-to-end.
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if doing, ok := cfg.GetAgentProfile("doing"); !ok || doing.Effort != "high" {
		t.Errorf("both-scoped agent.profiles must be honored from the system layer: %+v ok=%v", doing, ok)
	}

	w := warnings()
	wantWarn := `fab: warning: ignoring project-scoped field "source_paths"`
	if !strings.Contains(w, wantWarn) {
		t.Errorf("expected scope-pruning warning %q, got %q", wantWarn, w)
	}
	// The unknown key must NOT produce a warning (ignored silently).
	if strings.Contains(w, "totally_unknown_key") {
		t.Errorf("unknown system key must be ignored silently, but a warning mentioned it: %q", w)
	}
}

// TestScope_PruneAllProjectScopedFields walks every project-scoped top-level key
// through the pruner and asserts each is dropped with a NAMED warning, while the
// two both-scoped keys survive. The enumeration must track configscope.keyScopes:
// a project-scoped key added there without being added here is silently
// untested (consolidate, 4v91). fab_version is not a config key (it lives in
// fab/.fab-version, 260708-j0qm), so a stale system-file fab_version: is an inert
// unknown key — left in place SILENTLY like any other unknown key (nothing
// unmarshals it, so it can never bleed into a repo's resolved version).
func TestScope_PruneAllProjectScopedFields(t *testing.T) {
	warnings := captureWarnings(t)
	m := map[string]any{
		"project":             map[string]any{"name": "x"},
		"source_paths":        []any{"a"},
		"test_paths":          []any{"b"},
		"true_impact_exclude": []any{"c"},
		"checklist":           map[string]any{"extra_categories": []any{"d"}},
		"consolidate":         map[string]any{"detectors": []any{"jscpd {paths}"}},
		"stage_hooks":         map[string]any{"apply": map[string]any{"pre": "x"}},
		"branch_prefix":       "p/",
		"fab_version":         "1.0.0", // not a config key — an inert unknown key
		"agent":               map[string]any{"tiers": map[string]any{}},
		"providers":           map[string]any{"claude": map[string]any{}},
	}
	pruneProjectScoped(m, "/fake/system.yaml")

	for _, gone := range []string{"project", "source_paths", "test_paths", "true_impact_exclude", "checklist", "consolidate", "stage_hooks", "branch_prefix"} {
		if _, ok := m[gone]; ok {
			t.Errorf("project-scoped key %q must be pruned from the system layer", gone)
		}
		wantWarn := fmt.Sprintf("fab: warning: ignoring project-scoped field %q", gone)
		if !strings.Contains(warnings(), wantWarn) {
			t.Errorf("expected pruning warning %q, got %q", wantWarn, warnings())
		}
	}
	for _, kept := range []string{"agent", "providers"} {
		if _, ok := m[kept]; !ok {
			t.Errorf("both-scoped key %q must survive in the system layer", kept)
		}
	}
	// fab_version is an unknown key (not scoped): left in place silently, no warning.
	// It cannot bleed into the resolved version because Config.FabVersion is
	// tagged `yaml:"-"` and nothing unmarshals it.
	if _, ok := m["fab_version"]; !ok {
		t.Error("an unknown system-file key (fab_version) must be left in place, like any unrecognized key")
	}
	if strings.Contains(warnings(), "fab_version") {
		t.Errorf("an unknown key must be ignored silently (no warning), got %q", warnings())
	}
	if c := strings.Count(warnings(), "fab: warning:"); c != 8 {
		t.Errorf("expected 8 pruning warnings (one per project-scoped key), got %d", c)
	}
}

// TestScope_SystemFabVersionDoesNotBleedIntoResolvedConfig is the end-to-end guard
// that a fab_version in the system file never becomes the repo's Config.FabVersion:
// fab_version is not a config key (Config.FabVersion is `yaml:"-"`), so it is an
// inert unknown key that nothing unmarshals — the resolved version comes only from
// fab/.fab-version.
func TestScope_SystemFabVersionDoesNotBleedIntoResolvedConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sysDir := filepath.Join(home, ".fab-kit")
	if err := os.MkdirAll(sysDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sysDir, "config.yaml"), []byte("fab_version: 9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A project file with no fab_version and no fab/.fab-version sibling.
	fabRoot := filepath.Join(t.TempDir(), "fab")
	if err := os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("project:\n  name: t\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetFabVersion(); got != "" {
		t.Errorf("a system-file fab_version must not bleed into the resolved version, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// dispatch.mode — the preferred descent-ladder rung (scope `both`).
// ---------------------------------------------------------------------------

// TestLoad_DispatchMode: valid values parse verbatim; absent resolves to native.
func TestLoad_DispatchMode(t *testing.T) {
	isolateSystemConfig(t)
	for _, mode := range []string{"pane", "native", "headless"} {
		fabRoot := writeProjectConfig(t, "dispatch:\n  mode: "+mode+"\n")
		cfg, err := Load(fabRoot)
		if err != nil {
			t.Fatalf("Load(%s): %v", mode, err)
		}
		if got := cfg.GetDispatchMode(); got != mode {
			t.Errorf("dispatch.mode %q resolved %q", mode, got)
		}
	}

	bareRoot := writeProjectConfig(t, "project:\n  name: t\n")
	bare, err := Load(bareRoot)
	if err != nil {
		t.Fatalf("Load (bare): %v", err)
	}
	if got := bare.GetDispatchMode(); got != DefaultDispatchMode {
		t.Errorf("absent dispatch.mode = %q, want %q", got, DefaultDispatchMode)
	}
}

// TestCascade_DispatchModeFromSystemLayer: `dispatch` is scope `both`, so a
// machine-wide preference set once in ~/.fab-kit/config.yaml reaches a repo whose
// project config never mentions it (the requirement the scope exists for — a
// project-scoped key would be PRUNED from the system layer with a warning).
func TestCascade_DispatchModeFromSystemLayer(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, "dispatch:\n  mode: pane\n")
	fabRoot := writeProjectConfig(t, "project:\n  name: t\n")

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetDispatchMode(); got != "pane" {
		t.Errorf("system-layer dispatch.mode = %q, want pane", got)
	}
}

// TestCascade_DispatchModeProjectWins: the project layer beats the system layer.
func TestCascade_DispatchModeProjectWins(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, "dispatch:\n  mode: pane\n")
	root := writeProjectConfig(t, "dispatch:\n  mode: headless\n")
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetDispatchMode(); got != "headless" {
		t.Errorf("project dispatch.mode = %q, want headless over system pane", got)
	}
}

// TestGetDispatchMode_NilEmptyAndInvalid: the accessor is nil-safe and invalid
// values warn and fail open to the default.
func TestGetDispatchMode_NilEmptyAndInvalid(t *testing.T) {
	var nilCfg *Config
	if got := nilCfg.GetDispatchMode(); got != DefaultDispatchMode {
		t.Errorf("nil-config mode = %q, want %q", got, DefaultDispatchMode)
	}
	if got := (&Config{}).GetDispatchMode(); got != DefaultDispatchMode {
		t.Errorf("empty-config mode = %q, want %q", got, DefaultDispatchMode)
	}

	old := warnw
	var warnings bytes.Buffer
	warnw = &warnings
	t.Cleanup(func() { warnw = old })
	cfg := &Config{Dispatch: DispatchConfig{Mode: "sideways"}}
	if got := cfg.GetDispatchMode(); got != DefaultDispatchMode {
		t.Errorf("invalid mode = %q, want %q", got, DefaultDispatchMode)
	}
	want := "fab: warning: invalid dispatch.mode \"sideways\"; using \"native\"\n"
	if got := warnings.String(); got != want {
		t.Errorf("warning = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// dispatch.column_width — the pane-worker column width (scope `both`).
// ---------------------------------------------------------------------------

// TestLoad_DispatchColumnWidth: an in-range value parses and is reported verbatim;
// every out-of-range value — including an ABSENT key, which yaml cannot distinguish
// from an explicit 0 — resolves to DefaultDispatchColumnWidth. 0 and 100 are the
// degenerate widths the fallback exists for (0 gives the worker nothing, 100 leaves
// the dispatching agent nothing).
func TestLoad_DispatchColumnWidth(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"in-range value parses", "dispatch:\n  column_width: 20\n", 20},
		{"absent key ⇒ default", "project:\n  name: t\n", DefaultDispatchColumnWidth},
		{"absent width beside a live mode ⇒ default", "dispatch:\n  mode: pane\n", DefaultDispatchColumnWidth},
		{"explicit 0 reads as unset ⇒ default", "dispatch:\n  column_width: 0\n", DefaultDispatchColumnWidth},
		{"negative ⇒ default", "dispatch:\n  column_width: -10\n", DefaultDispatchColumnWidth},
		{"100 leaves the dispatcher nothing ⇒ default", "dispatch:\n  column_width: 100\n", DefaultDispatchColumnWidth},
		{"over 100 ⇒ default", "dispatch:\n  column_width: 250\n", DefaultDispatchColumnWidth},
		{"1 is in range", "dispatch:\n  column_width: 1\n", 1},
		{"99 is in range", "dispatch:\n  column_width: 99\n", 99},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateSystemConfig(t)
			fabRoot := writeProjectConfig(t, tt.body)
			cfg, err := Load(fabRoot)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.GetDispatchColumnWidth(); got != tt.want {
				t.Errorf("GetDispatchColumnWidth() = %d, want %d (config %q)", got, tt.want, tt.body)
			}
		})
	}
}

// TestCascade_DispatchColumnWidthFromSystemLayer: `dispatch` is scope `both`, so a
// personal column width set once in ~/.fab-kit/config.yaml reaches every repo — the
// whole point of the scope (a project-scoped key would be PRUNED from the system
// layer with a warning). The project layer still wins where it sets one.
func TestCascade_DispatchColumnWidthFromSystemLayer(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, "dispatch:\n  column_width: 25\n")
	fabRoot := writeProjectConfig(t, "project:\n  name: t\n")

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.GetDispatchColumnWidth(); got != 25 {
		t.Errorf("system-layer column_width = %d, want 25 (scope `both`, not pruned)", got)
	}

	home2 := isolateSystemConfig(t)
	writeSystemConfig(t, home2, "dispatch:\n  column_width: 25\n")
	projRoot := writeProjectConfig(t, "dispatch:\n  column_width: 40\n")
	cfg2, err := Load(projRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg2.GetDispatchColumnWidth(); got != 40 {
		t.Errorf("project column_width = %d, want the project's 40 to beat the system's 25", got)
	}
}

// TestGetDispatchColumnWidth_NilAndEmptyConfig: the accessor is nil-safe and reports
// the built-in default for a zero Config — the value every caller falls back to when
// config could not be loaded at all.
func TestGetDispatchColumnWidth_NilAndEmptyConfig(t *testing.T) {
	var nilCfg *Config
	if got := nilCfg.GetDispatchColumnWidth(); got != DefaultDispatchColumnWidth {
		t.Errorf("nil-config GetDispatchColumnWidth() = %d, want %d", got, DefaultDispatchColumnWidth)
	}
	if got := (&Config{}).GetDispatchColumnWidth(); got != DefaultDispatchColumnWidth {
		t.Errorf("empty-config GetDispatchColumnWidth() = %d, want %d", got, DefaultDispatchColumnWidth)
	}
}

// ---------------------------------------------------------------------------
// dispatch.reap_done — done-worker pane reaping (scope `both`, default TRUE).
// ---------------------------------------------------------------------------

// TestLoad_DispatchReapDone: the default-TRUE knob is the one dispatch field whose
// Go zero value means the OPPOSITE of its default, which is why it is modeled as a
// *bool. The table pins the property that motivates the pointer: an absent key and
// an explicit `false` must NOT resolve alike.
func TestLoad_DispatchReapDone(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"absent dispatch block ⇒ default true", "project:\n  name: t\n", true},
		{"absent key beside a live sibling ⇒ default true", "dispatch:\n  mode: pane\n", true},
		{"explicit true parses", "dispatch:\n  reap_done: true\n", true},
		{"explicit false parses (NOT collapsed into absent)", "dispatch:\n  reap_done: false\n", false},
		{"explicit false beside the siblings", "dispatch:\n  mode: pane\n  column_width: 20\n  reap_done: false\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateSystemConfig(t)
			fabRoot := writeProjectConfig(t, tt.body)
			cfg, err := Load(fabRoot)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cfg.GetDispatchReapDone(); got != tt.want {
				t.Errorf("GetDispatchReapDone() = %v, want %v (config %q)", got, tt.want, tt.body)
			}
		})
	}
}

// TestCascade_DispatchReapDoneFromSystemLayer: `dispatch` is scope `both`, so the
// opt-OUT set once in ~/.fab-kit/config.yaml reaches a repo whose project config
// never mentions it — the whole point of the scope for a preference this personal
// (a project-scoped key would be PRUNED from the system layer with a warning). The
// project layer still wins where it sets one, in both directions.
func TestCascade_DispatchReapDoneFromSystemLayer(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, "dispatch:\n  reap_done: false\n")
	fabRoot := writeProjectConfig(t, "project:\n  name: t\n")

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.GetDispatchReapDone() {
		t.Error("a system-layer dispatch.reap_done: false must be honored (scope `both`, not pruned)")
	}

	home2 := isolateSystemConfig(t)
	writeSystemConfig(t, home2, "dispatch:\n  reap_done: false\n")
	onRoot := writeProjectConfig(t, "dispatch:\n  reap_done: true\n")
	cfg2, err := Load(onRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg2.GetDispatchReapDone() {
		t.Error("a project `true` must beat a system `false`")
	}

	home3 := isolateSystemConfig(t)
	writeSystemConfig(t, home3, "dispatch:\n  reap_done: true\n")
	offRoot := writeProjectConfig(t, "dispatch:\n  reap_done: false\n")
	cfg3, err := Load(offRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg3.GetDispatchReapDone() {
		t.Error("a project `false` must beat a system `true`")
	}
}

// TestGetDispatchReapDone_NilAndEmptyConfig: the accessor is nil-safe in BOTH senses
// — a nil *Config and a present Config whose ReapDone pointer is nil — and reports
// the built-in default for each. This is the case a plain bool would have gotten
// wrong (it would report false, the opposite of the default).
func TestGetDispatchReapDone_NilAndEmptyConfig(t *testing.T) {
	var nilCfg *Config
	if got := nilCfg.GetDispatchReapDone(); got != DefaultDispatchReapDone {
		t.Errorf("nil-config GetDispatchReapDone() = %v, want %v", got, DefaultDispatchReapDone)
	}
	if got := (&Config{}).GetDispatchReapDone(); got != DefaultDispatchReapDone {
		t.Errorf("empty-config GetDispatchReapDone() = %v, want %v", got, DefaultDispatchReapDone)
	}
	no := false
	if (&Config{Dispatch: DispatchConfig{ReapDone: &no}}).GetDispatchReapDone() {
		t.Error("an explicitly-false ReapDone pointer must report false")
	}
}
