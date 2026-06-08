package main

import (
	"fmt"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// spawnCommandCmd prints a repo's agent.spawn_command. With --repo <path> it
// reads <path>/fab/project/config.yaml directly; without --repo it resolves the
// current repo's fab/ via resolve.FabRoot() (the same source runOperator uses).
// This lets the operator skill fetch a TARGET repo's spawn command rather than
// only its own. Falls back to spawn.DefaultSpawnCommand when the key is
// missing, empty, or the file cannot be read.
func spawnCommandCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spawn-command",
		Short: "Print a repo's configured agent spawn command",
		Args:  cobra.NoArgs,
		RunE:  runSpawnCommand,
	}
	cmd.Flags().String("repo", "", "Repo root to read agent.spawn_command from (default: current repo)")
	return cmd
}

func runSpawnCommand(cmd *cobra.Command, args []string) error {
	repo, _ := cmd.Flags().GetString("repo")

	var configPath string
	if repo != "" {
		configPath = filepath.Join(repo, "fab", "project", "config.yaml")
	} else {
		fabRoot, err := resolve.FabRoot()
		if err != nil {
			return err
		}
		configPath = filepath.Join(fabRoot, "project", "config.yaml")
	}

	fmt.Fprintln(cmd.OutOrStdout(), spawn.Command(configPath))
	return nil
}
