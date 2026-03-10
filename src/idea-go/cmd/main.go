package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var fileFlag string

func main() {
	root := &cobra.Command{
		Use:   "idea",
		Short: "Backlog idea management — CRUD for fab/backlog.md",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&fileFlag, "file", "", "Override backlog file path (relative to git root)")

	root.AddCommand(
		addCmd(),
		listCmd(),
		showCmd(),
		doneCmd(),
		reopenCmd(),
		editCmd(),
		rmCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
