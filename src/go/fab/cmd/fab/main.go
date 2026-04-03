package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:     "fab",
		Short:   "Fab workflow engine — single binary replacement for kit shell scripts",
		Version: version,
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
		runtimeCmd(),
		hookCmd(),
		paneCmd(),
		fabHelpCmd(),
		operatorCmd(),
		batchCmd(),
		kitPathCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
