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

// TierProfile is a named `{model, effort}` agent profile. Either field MAY be
// empty: an empty Model signals "inherit the session/orchestrator model" and
// an empty Effort omits the effort entirely. Both strings are pass-through —
// fab applies NO validation against any provider's accepted set (provider
// neutrality, Constitution Principle I). See internal/agent for resolution.
type TierProfile struct {
	Model  string `yaml:"model"`
	Effort string `yaml:"effort"`
}

// AgentConfig models the `agent:` section of config.yaml.
//
// Tiers is the sole per-stage-model override surface: a map of tier name
// (thinking/doing/ship) → profile. The stage→tier mapping itself is fab-owned
// and NOT user-overridable (no stage_tiers, no per-stage escape hatch); users
// override only what each tier *means*. An omitted tier — or an omitted field
// within a tier — falls back to fab-kit's built-in default (per-field merge,
// performed by internal/agent). yaml.v3 ignores unknown keys, so adding Tiers
// is free for existing configs (the same property that made stage_hooks free).
type AgentConfig struct {
	SpawnCommand string                 `yaml:"spawn_command"`
	Tiers        map[string]TierProfile `yaml:"tiers"`
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
	StageHooks        map[string]StageHook `yaml:"stage_hooks"`
	TrueImpactExclude []string             `yaml:"true_impact_exclude"`
	TestPaths         []string             `yaml:"test_paths"`
	BranchPrefix      string               `yaml:"branch_prefix"`
	FabVersion        string               `yaml:"fab_version"`
	Agent             AgentConfig          `yaml:"agent"`
	Project           ProjectConfig        `yaml:"project"`
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

// GetSpawnCommand returns agent.spawn_command, or "" when unset (nil-safe).
// The default-command fallback lives with the spawn package's contract
// (spawn.DefaultSpawnCommand), not here.
func (c *Config) GetSpawnCommand() string {
	if c == nil {
		return ""
	}
	return c.Agent.SpawnCommand
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
