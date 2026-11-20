package main

import (
	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// defaultRoleSpawnCommand composes the worker session command for `fab batch
// new`/`switch`: the default role's CLI interactive_command (falling back
// to spawn.DefaultSpawnCommand when absent)
// with the default role's {model}/{effort} SUBSTITUTED via internal/spawn. Workers
// spawn WITH a profile (the former placeholder-stripping print path is gone).
// Substitution resolves every placeholder, so no literal {model}/{effort} braces
// reach tmux.
//
// `default` is a Tier-1 (session) role, so which CLI it lands on is the
// agent.session knob's call — a batch worker is an agent the user talks to.
// The returned CLI identifies the command actually launched, including fallback.
func defaultRoleSpawnCommand(configPath string) (command, cli string) {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		cfg = nil // nil-safe accessors below deliver the built-in fallbacks
	}

	// RoleDefault is always a known role (drift-guarded), so ResolveRole only
	// errors on a truly unknown role — impossible for the constant RoleDefault.
	command, profile, err := roleSessionCommand(cfg, agent.RoleDefault)
	if err != nil {
		profile, _ = agent.DefaultProfile(agent.RoleDefault)
	}

	cli = profile.CLI
	if command == "" {
		command = spawn.WithProfile(spawn.DefaultSpawnCommand, profile.Model, profile.Effort)
		cli = "claude"
	}
	return command, cli
}

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Multi-target batch operations",
	}

	cmd.AddCommand(
		batchNewCmd(),
		batchSwitchCmd(),
		batchArchiveCmd(),
	)

	return cmd
}
