package main

import (
	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// defaultRoleSpawnCommand composes the worker session command for `fab batch
// new`/`switch`: the default role's provider interactive_command (resolved by
// spawn.Command, which reads providers.<default-role.provider>.interactive_command over
// fab-kit's built-in claude provider and falls back to spawn.DefaultSpawnCommand)
// with the default role's {model}/{effort} SUBSTITUTED via internal/spawn. Workers
// spawn WITH a profile (the former placeholder-stripping print path is gone).
// Substitution resolves every placeholder, so no literal {model}/{effort} braces
// reach tmux.
//
// `default` is a Tier-1 (session) role, so which provider it lands on is the
// agent.session knob's call — a batch worker is an agent the user talks to.
func defaultRoleSpawnCommand(configPath string) string {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		cfg = nil // nil-safe accessors below deliver the built-in fallbacks
	}

	// RoleDefault is always a known role (drift-guarded), so ResolveRole only
	// errors on a truly unknown role — impossible for the constant RoleDefault.
	profile, err := agent.ResolveRole(cfg, agent.RoleDefault)
	if err != nil {
		profile, _ = agent.DefaultProfile(agent.RoleDefault)
	}

	return spawn.WithProfile(spawn.Command(configPath), profile.Model, profile.Effort)
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
