package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func paneKillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kill <pane>",
		Short: "Kill a tmux pane",
		Long: "Kill a tmux pane by id. This is the record-free generic kill — the pane\n" +
			"family's missing member: operator removal paths and probe cleanups no longer\n" +
			"need raw `tmux kill-pane`, which bypasses the family's validated exit-code\n" +
			"contract. `fab dispatch kill` (record-keyed, ungated recovery) is unaffected\n" +
			"and remains the pipeline's kill.\n\n" +
			"Exit codes: 0 killed; 2 pane missing; 3 other tmux failure.",
		Example: `  fab pane kill %12
  fab pane kill %3 --server work`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneKill(cmd, args[0])
		},
	}
}

// runPaneKill validates the pane and kills it via the shared KillPane helper.
//
// Validation runs FIRST so a missing pane is the family's exit 2 rather than
// KillPane's idempotent no-op: at the command surface "you asked me to kill a
// pane that is not there" is a branchable answer (2), while KillPane's internal
// already-gone tolerance exists for the dispatch/reap callers whose pane may
// have died between the state derivation and the kill.
func runPaneKill(cmd *cobra.Command, paneID string) error {
	server, _ := cmd.Flags().GetString("server")

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	if err := pane.KillPane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "killed %s\n", paneID)
	if server != "" {
		fmt.Fprintf(out, "server: %s\n", server)
	}
	return nil
}
