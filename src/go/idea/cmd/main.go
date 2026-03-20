package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var fileFlag string
var mainFlag bool

func main() {
	root := &cobra.Command{
		Use:   "idea",
		Short: "Backlog idea management (current worktree; use --main for main worktree)",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&fileFlag, "file", "", "Override backlog file path (relative to git root)")
	root.PersistentFlags().BoolVar(&mainFlag, "main", false, "Operate on the main worktree's backlog instead of the current worktree")

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
