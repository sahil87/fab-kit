package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "fab",
		Short: "Fab workflow engine — single binary replacement for kit shell scripts",
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
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
