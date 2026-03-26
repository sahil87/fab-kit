package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var fileFlag string
var mainFlag bool

func main() {
	root := &cobra.Command{
		Use:   "idea [text]",
		Short: "Backlog idea management (current worktree; use --main for main worktree)",
		Long: `Backlog idea management (current worktree; use --main for main worktree).

Shorthand: "idea <text>" is equivalent to "idea add <text>".`,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			// Delegate to the "add" subcommand to keep behavior consistent.
			add := addCmd()
			add.SetIn(cmd.InOrStdin())
			add.SetOut(cmd.OutOrStdout())
			add.SetErr(cmd.ErrOrStderr())
			// addCmd expects exactly 1 arg; join multiple positional args.
			return add.RunE(add, []string{strings.Join(args, " ")})
		},
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
