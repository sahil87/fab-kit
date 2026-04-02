package spawn

import (
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultSpawnCommand is the fallback when config.yaml has no agent.spawn_command.
const DefaultSpawnCommand = "claude --dangerously-skip-permissions"

// configFile is a minimal struct covering the agent section of config.yaml.
type configFile struct {
	Agent struct {
		SpawnCommand string `yaml:"spawn_command"`
	} `yaml:"agent"`
}

// Command reads agent.spawn_command from the given config.yaml path.
// Returns the configured command, or DefaultSpawnCommand if the key is
// missing, empty, or the file cannot be read.
func Command(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultSpawnCommand
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DefaultSpawnCommand
	}

	if cfg.Agent.SpawnCommand == "" {
		return DefaultSpawnCommand
	}

	return cfg.Agent.SpawnCommand
}
