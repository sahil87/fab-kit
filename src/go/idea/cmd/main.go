package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/idea/internal/idea"
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
			text := strings.TrimSpace(strings.Join(args, " "))
			if text == "" {
				return fmt.Errorf("idea text cannot be empty")
			}
			path, err := resolveFile()
			if err != nil {
				return err
			}
			i, err := idea.Add(path, text, "", "")
			if err != nil {
				return err
			}
			fmt.Printf("Added: [%s] %s: %s\n", i.ID, i.Date, i.Text)
			return nil
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
