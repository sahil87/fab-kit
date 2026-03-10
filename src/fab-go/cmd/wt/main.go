package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	wt "github.com/wvrdz/fab-kit/src/fab-go/internal/worktree"
)

func main() {
	root := &cobra.Command{
		Use:   "wt",
		Short: "Git worktree management — create, list, open, delete worktrees",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		createCmd(),
		listCmd(),
		openCmd(),
		deleteCmd(),
		initCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(wt.ExitGeneralError)
	}
}
