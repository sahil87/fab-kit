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
	"doctor":  true,
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
		doctorCmd(),
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
	var shimOnly, projectOnly bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync workspace (skills, directories, scaffold)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shimOnly && projectOnly {
				return fmt.Errorf("--shim and --project are mutually exclusive")
			}
			return internal.Sync(version, shimOnly, projectOnly)
		},
	}
	cmd.Flags().BoolVar(&shimOnly, "shim", false, "Run shim steps only (prerequisites, version guard, cache, scaffold, direnv)")
	cmd.Flags().BoolVar(&projectOnly, "project", false, "Run project sync scripts only")
	return cmd
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
