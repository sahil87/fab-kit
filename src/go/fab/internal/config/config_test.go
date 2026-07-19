package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
// fields, and a provider with only a session_command yields an empty
// DispatchCommand (the native-dispatch signal). The accessor is a pure
// pass-through; the built-in merge is internal/agent's job.
func TestLoad_WithProviders(t *testing.T) {
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
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
	isolateSystemConfig(t)
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

// TestCascade_MapsMergePerKey: agent.tiers merges per-key across the two files —
// a project tier field and a system tier field compose, project wins on a
// conflicting leaf, and a system-only tier survives alongside a project-only one.
func TestCascade_MapsMergePerKey(t *testing.T) {
	home := isolateSystemConfig(t)
	writeSystemConfig(t, home, `
agent:
  tiers:
    doing: { provider: claude, model: system-model, effort: low }
    sysonly: { model: sys-only-model }
`)
	fabRoot := writeProjectConfig(t, `
agent:
  tiers:
    doing: { model: project-model }
    projonly: { model: proj-only-model }
`)

	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// doing.model: project wins (project-model); doing.effort inherited from
	// system (low); doing.provider inherited from system (claude).
	doing, ok := cfg.GetAgentTier("doing")
	if !ok {
		t.Fatal("expected a merged 'doing' tier")
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

	// A system-only tier survives (per-key merge, not whole-map replacement).
	if sysonly, ok := cfg.GetAgentTier("sysonly"); !ok || sysonly.Model != "sys-only-model" {
		t.Errorf("system-only tier lost in merge: %+v ok=%v", sysonly, ok)
	}
	// A project-only tier survives alongside it.
	if projonly, ok := cfg.GetAgentTier("projonly"); !ok || projonly.Model != "proj-only-model" {
		t.Errorf("project-only tier lost in merge: %+v ok=%v", projonly, ok)
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

// --- Scope enforcement (lpb5, decision 6) ---

// TestScope_PruneProjectScopedFromSystem: a project-scoped field placed in the
// system file is pruned (not applied) and a `fab: warning:` names it; a
// both-scoped field (agent.tiers) is honored; an unknown key is ignored silently.
func TestScope_PruneProjectScopedFromSystem(t *testing.T) {
	home := isolateSystemConfig(t)
	warnings := captureWarnings(t)
	writeSystemConfig(t, home, `
source_paths:
  - system-only-src/
agent:
  tiers:
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

	// agent.tiers (scope both) from the system file is honored end-to-end.
	cfg, err := Load(fabRoot)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if doing, ok := cfg.GetAgentTier("doing"); !ok || doing.Effort != "high" {
		t.Errorf("both-scoped agent.tiers must be honored from the system layer: %+v ok=%v", doing, ok)
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
// through the pruner and asserts each is dropped with a warning, while the two
// both-scoped keys survive. fab_version is not a config key (it lives in
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
		"stage_hooks":         map[string]any{"apply": map[string]any{"pre": "x"}},
		"branch_prefix":       "p/",
		"fab_version":         "1.0.0", // not a config key — an inert unknown key
		"agent":               map[string]any{"tiers": map[string]any{}},
		"providers":           map[string]any{"claude": map[string]any{}},
	}
	pruneProjectScoped(m, "/fake/system.yaml")

	for _, gone := range []string{"project", "source_paths", "test_paths", "true_impact_exclude", "checklist", "stage_hooks", "branch_prefix"} {
		if _, ok := m[gone]; ok {
			t.Errorf("project-scoped key %q must be pruned from the system layer", gone)
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
	if c := strings.Count(warnings(), "fab: warning:"); c != 7 {
		t.Errorf("expected 7 pruning warnings (one per project-scoped key), got %d", c)
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
