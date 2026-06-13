package spawn

import (
	"strings"

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

// WithProfile appends --model/--effort to the END of spawnCmd (last-wins),
// omitting each flag when its value is empty. Model is appended before effort.
// Appending last is deliberate: the configured spawn_command may already pin a
// --model/--effort, and a trailing occurrence wins on the claude CLI (duplicate
// --effort is accepted without a parse error), so the caller's deliberate tier
// choice overrides whatever the spawn_command defaulted to. An empty value
// mirrors the documented `empty ⇒ omit` convention (_preamble.md § Per-Stage
// Model Resolution): empty model ⇒ omit --model (inherit), empty effort ⇒ omit
// --effort.
func WithProfile(spawnCmd, model, effort string) string {
	var b strings.Builder
	b.WriteString(spawnCmd)
	if model != "" {
		b.WriteString(" --model ")
		b.WriteString(model)
	}
	if effort != "" {
		b.WriteString(" --effort ")
		b.WriteString(effort)
	}
	return b.String()
}
