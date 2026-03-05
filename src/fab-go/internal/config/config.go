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

// Config holds the parsed project config relevant to the fab binary.
type Config struct {
	StageHooks map[string]StageHook `yaml:"stage_hooks"`
}

// Load reads fab/project/config.yaml from fabRoot and returns the parsed config.
// Returns an empty config (no hooks) if the file doesn't exist or has no stage_hooks.
func Load(fabRoot string) (*Config, error) {
	path := filepath.Join(fabRoot, "project", "config.yaml")
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
