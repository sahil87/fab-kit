package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StageHook holds pre/post shell commands for a pipeline stage.
type StageHook struct {
	Pre  string `yaml:"pre"`
	Post string `yaml:"post"`
}

// ProviderConfig models one entry of the top-level `providers:` table: a named
// invocation grammar for an agent harness. Provider names are opaque, user-chosen
// strings (fab never infers a provider from a model string).
//
//   - SessionCommand opens an interactive agent SESSION (the relocated
//     agent.spawn_command semantics — consumed by fab operator / fab batch /
//     fab agent).
//   - DispatchCommand runs ONE headless stage task via fab dispatch (the relocated
//     per-tier spawn_command semantics). ABSENT DispatchCommand = native
//     Agent-tool dispatch (the default). There is NO fallback between the two
//     fields: absence of DispatchCommand signals native dispatch, never "use
//     SessionCommand".
//
// The two fields are deliberately NOT merged into one command: session and
// dispatch are different invocations of the same binary (claude interactive `-n`
// vs headless `-p`; codex TUI vs `codex exec`), and no single template expresses
// both. Both strings pass through verbatim — fab applies NO validation against any
// provider's accepted set (provider neutrality, Constitution Principle I). The
// {model}/{effort} placeholders are substituted at resolve time via internal/spawn.
type ProviderConfig struct {
	SessionCommand  string `yaml:"session_command"`
	DispatchCommand string `yaml:"dispatch_command"`
}

// TierProfile is a named `{provider, model, effort}` agent profile. Every field
// MAY be empty: an empty Provider/Model/Effort inherits from the project's
// `default` tier, then from fab-kit's built-in (per-field merge, performed by
// internal/agent). An empty Model additionally signals "inherit the
// session/orchestrator model" once resolution bottoms out.
//
// Provider names the entry in the top-level `providers:` table whose command
// grammar this tier's stages use. The command itself lives on the provider, NOT
// the tier — inheriting {provider, model, effort} is safe precisely because the
// dangerous cross-semantics command inheritance can no longer happen.
//
// All strings are pass-through — fab applies NO validation (provider neutrality,
// Constitution Principle I). See internal/agent for resolution.
type TierProfile struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
}

// AgentConfig models the `agent:` section of config.yaml.
//
// Tiers is the sole per-stage-model override surface: a map of role-tier name
// (default/operator/doing/review/fast) → profile. The stage→tier mapping itself
// is fab-owned and NOT user-overridable (no stage_tiers, no per-stage escape
// hatch); users override only what each tier *means*. An omitted tier — or an
// omitted field within a tier — falls back to the project's `default` tier and
// then fab-kit's built-in default (per-field merge, performed by internal/agent).
// yaml.v3 ignores unknown keys, so adding Tiers is free for existing configs (the
// same property that made stage_hooks free).
type AgentConfig struct {
	Tiers map[string]TierProfile `yaml:"tiers"`
}

// ProjectConfig models the `project:` section of config.yaml.
type ProjectConfig struct {
	LinearWorkspace string `yaml:"linear_workspace"`
}

// Config holds the parsed project config relevant to the fab binary. It is
// the single owner of fab/project/config.yaml parsing — every key the fab
// module consumes is modeled here and read through a nil-safe accessor, so no
// satellite one-off parser re-reads the file (260612-ye8r). yaml.v3 ignores
// unknown keys, so widening this struct is free for existing configs.
//
// Known coupled-failure caveat: a yaml type error on ANY modeled key fails
// the single Unmarshal, sending every accessor to its documented fallback
// (default spawn command, empty branch prefix, empty workspace, silent
// staleness skip). The documented per-caller fallbacks make this safe for
// malformed configs — a deliberate, recorded semantic for the consolidation.
type Config struct {
	StageHooks        map[string]StageHook      `yaml:"stage_hooks"`
	TrueImpactExclude []string                  `yaml:"true_impact_exclude"`
	TestPaths         []string                  `yaml:"test_paths"`
	BranchPrefix      string                    `yaml:"branch_prefix"`
	FabVersion        string                    `yaml:"fab_version"`
	Providers         map[string]ProviderConfig `yaml:"providers"`
	Agent             AgentConfig               `yaml:"agent"`
	Project           ProjectConfig             `yaml:"project"`
}

// Load reads fab/project/config.yaml from fabRoot and returns the parsed config.
// Returns an empty config if the file doesn't exist.
func Load(fabRoot string) (*Config, error) {
	return LoadPath(filepath.Join(fabRoot, "project", "config.yaml"))
}

// LoadPath reads a config.yaml at an explicit path. Callers that build the
// path themselves (e.g. `fab spawn-command --repo <path>` reading a target
// repo's config) use this directly; everyone else goes through Load.
// Returns an empty config (no error) if the file doesn't exist.
func LoadPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.StageHooks == nil {
		cfg.StageHooks = make(map[string]StageHook)
	}

	return &cfg, nil
}

// GetStageHook returns the hook config for a stage, or an empty hook if none configured.
func (c *Config) GetStageHook(stage string) StageHook {
	if c == nil || c.StageHooks == nil {
		return StageHook{}
	}
	return c.StageHooks[stage]
}

// GetBranchPrefix returns branch_prefix, or "" when unset (nil-safe).
func (c *Config) GetBranchPrefix() string {
	if c == nil {
		return ""
	}
	return c.BranchPrefix
}

// GetFabVersion returns fab_version, or "" when unset (nil-safe).
func (c *Config) GetFabVersion() string {
	if c == nil {
		return ""
	}
	return c.FabVersion
}

// GetProvider returns the configured ProviderConfig for a provider name and
// whether one was set. Nil-safe: a nil *Config, an absent providers block, or an
// unconfigured name all report (zero, false). The bool lets a caller distinguish
// "no provider entry" from "entry present but with empty fields" — the distinction
// internal/agent relies on for per-field merge over fab-kit's built-in provider
// table.
func (c *Config) GetProvider(name string) (ProviderConfig, bool) {
	if c == nil || c.Providers == nil {
		return ProviderConfig{}, false
	}
	p, ok := c.Providers[name]
	return p, ok
}

// GetAgentTier returns the configured override profile for a tier name and
// whether one was set. Nil-safe: a nil *Config, an absent agent.tiers block, or
// an unconfigured tier all report (zero, false). The bool lets a caller
// distinguish "no override" from "override present but with empty fields" — the
// distinction internal/agent.Resolve relies on for per-field merge over the
// fab-kit default.
func (c *Config) GetAgentTier(tier string) (TierProfile, bool) {
	if c == nil || c.Agent.Tiers == nil {
		return TierProfile{}, false
	}
	p, ok := c.Agent.Tiers[tier]
	return p, ok
}

// GetLinearWorkspace returns project.linear_workspace, or "" when unset (nil-safe).
func (c *Config) GetLinearWorkspace() string {
	if c == nil {
		return ""
	}
	return c.Project.LinearWorkspace
}
