package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
	"github.com/spf13/cobra"
)

var version = "dev"

// fabKitCommands lists the commands owned by fab-kit (used by tests).
var fabKitCommands = map[string]bool{
	"init":    true,
	"upgrade": true,
	"sync":    true,
	"update":  true,
}

func main() {
	root := &cobra.Command{
		Use:           "fab-kit",
		Short:         "Fab Kit — workspace lifecycle (init, upgrade, sync)",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		initCmd(),
		upgradeCmd(),
		syncCmd(),
		updateCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize fab in the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Init()
		},
	}
}

func upgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [version]",
		Short: "Upgrade fab/.kit/ to a specific or latest version",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion := ""
			if len(args) > 0 {
				targetVersion = args[0]
			}
			return internal.Upgrade(targetVersion)
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync workspace (skills, directories, scaffold)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Sync()
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update fab-kit itself via Homebrew",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Update(version)
		},
	}
}
