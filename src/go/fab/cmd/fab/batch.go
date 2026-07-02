package main

import (
	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// defaultTierSpawnCommand composes the worker session command for `fab batch
// new`/`switch`: the default tier's provider session_command (resolved by
// spawn.Command, which reads providers.<default.provider>.session_command over
// fab-kit's built-in claude provider and falls back to spawn.DefaultSpawnCommand)
// with the default tier's {model}/{effort} SUBSTITUTED via internal/spawn. Workers
// spawn WITH a profile (the former placeholder-stripping print path is gone).
// Substitution resolves every placeholder, so no literal {model}/{effort} braces
// reach tmux.
func defaultTierSpawnCommand(configPath string) string {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		cfg = nil // nil-safe accessors below deliver the built-in fallbacks
	}

	// TierDefault is always in defaultTiers (drift-guarded), so ResolveTier only
	// errors on a truly unknown tier — impossible for the constant TierDefault.
	profile, err := agent.ResolveTier(cfg, agent.TierDefault)
	if err != nil {
		profile, _ = agent.DefaultTier(agent.TierDefault)
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
