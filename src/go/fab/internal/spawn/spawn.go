package spawn

import (
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// DefaultSpawnCommand is the fallback when config.yaml has no agent.spawn_command.
const DefaultSpawnCommand = "claude --dangerously-skip-permissions"

// Command reads agent.spawn_command from the given config.yaml path via the
// shared internal/config loader (the single config.yaml parser). Returns the
// configured command, or DefaultSpawnCommand if the key is missing, empty, or
// the file cannot be read/parsed. The path-based signature is kept because
// `fab spawn-command --repo <path>` builds the path from an arbitrary repo
// root.
func Command(configPath string) string {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		return DefaultSpawnCommand
	}

	if cmd := cfg.GetSpawnCommand(); cmd != "" {
		return cmd
	}
	return DefaultSpawnCommand
}
