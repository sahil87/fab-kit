package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/shim/internal"
	"github.com/spf13/cobra"
)

var version = "dev"

// shimCommands are commands handled directly by the shim.
var shimCommands = map[string]bool{
	"init":    true,
	"upgrade": true,
}

func main() {
	// Check if the first arg is a shim-handled command.
	// If not, bypass cobra entirely and dispatch to fab-go.
	if len(os.Args) > 1 {
		arg := os.Args[1]

		// Handle --version / -v directly
		if arg == "--version" || arg == "-v" {
			fmt.Printf("fab %s (shim)\n", version)
			return
		}

		// Non-shim commands: dispatch directly to fab-go
		if !shimCommands[arg] && arg != "--help" && arg != "-h" && arg != "help" {
			if err := internal.Dispatch(os.Args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
				os.Exit(1)
			}
			return
		}
	} else {
		// No args: dispatch to fab-go (shows fab-go help)
		if err := internal.Dispatch(nil); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		return
	}

	// Shim commands: use cobra
	root := &cobra.Command{
		Use:           "fab",
		Short:         "Fab CLI shim — version-aware dispatch to the fab-go backend",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		initCmd(),
		upgradeCmd(),
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
