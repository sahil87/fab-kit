package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
	"github.com/spf13/cobra"
)

var version = "dev"

// displayVersion returns version with a "v" prefix when it looks like a real
// release (e.g., "1.9.4" → "v1.9.4"), so `fab-kit --version` matches the
// toolkit-wide standard `<name> version v<X.Y.Z>`. The "dev" sentinel and any
// already-prefixed value pass through unchanged.
func displayVersion(v string) string {
	if v == "dev" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// fabKitCommands lists the commands owned by fab-kit (used by tests), derived
// from the shared internal.LifecycleCommands table — the single source of
// truth also feeding the router's allowlist and help section.
var fabKitCommands = internal.LifecycleCommandSet()

// rootCmd assembles the fab-kit root command with all subcommands registered.
// Extracted from main() so tests can cross-check the registered command set
// (names and Shorts) against internal.LifecycleCommands.
func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "fab-kit",
		Short:         "Fab Kit — workspace lifecycle (init, upgrade-repo, sync)",
		Version:       displayVersion(version),
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		initCmd(),
		upgradeCmd(),
		syncCmd(),
		updateCmd(),
		doctorCmd(),
		migrationsStatusCmd(),
	)

	return root
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize fab in the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Init(version)
		},
	}
}

func upgradeCmd() *cobra.Command {
	var useLatest bool
	cmd := &cobra.Command{
		Use:   "upgrade-repo [version]",
		Short: "Upgrade the repo's kit to the installed binary's version (or --latest / an explicit version)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion := ""
			if len(args) > 0 {
				targetVersion = args[0]
			}
			return internal.Upgrade(version, targetVersion, useLatest)
		},
	}
	cmd.Flags().BoolVar(&useLatest, "latest", false, "Resolve the newest published release from GitHub instead of the installed binary version")
	return cmd
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
			return internal.Sync(version, "", shimOnly, projectOnly)
		},
	}
	cmd.Flags().BoolVar(&shimOnly, "shim", false, "Run shim steps only (prerequisites, version guard, cache, scaffold, direnv)")
	cmd.Flags().BoolVar(&projectOnly, "project", false, "Run project sync scripts only")
	return cmd
}

func updateCmd() *cobra.Command {
	var skipBrewUpdate bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update fab-kit itself via Homebrew",
		RunE: func(cmd *cobra.Command, args []string) error {
			return internal.Update(version, skipBrewUpdate)
		},
	}
	cmd.Flags().BoolVar(&skipBrewUpdate, "skip-brew-update", false,
		"Skip the brew update tap-metadata refresh (still runs brew info + brew upgrade)")
	return cmd
}
