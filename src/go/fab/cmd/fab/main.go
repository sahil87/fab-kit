package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

// newRootCmd assembles the fab-go root command with all subcommands
// registered. Extracted from main() so tests can walk the live command tree
// (e.g. the router-allowlist collision test sources top-level names from the
// help-dump tree of this root).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "fab",
		Short:         "Fab workflow engine — single binary replacement for kit shell scripts",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		resolveCmd(),
		logCmd(),
		statusCmd(),
		preflightCmd(),
		changeCmd(),
		scoreCmd(),
		hookCmd(),
		paneCmd(),
		fabHelpCmd(),
		operatorCmd(),
		spawnCommandCmd(),
		batchCmd(),
		kitPathCmd(),
		impactCmd(),
		prMetaCmd(),
		memoryIndexCmd(),
		shellInitCmd(),
		helpDumpCmd(),
	)

	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
