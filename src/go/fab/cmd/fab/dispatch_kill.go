package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

func dispatchKillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "kill <change> <stage>",
		Short: "Kill the dispatch — the process group (headless) or the tmux pane (pane dispatch); idempotent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchKill(cmd, args[0], args[1])
		},
	}
}

// runDispatchKill terminates a dispatch by the mechanism its recorded mode
// implies — the detached worker's process group (headless) or the tmux pane
// (pane mode). Both paths are idempotent: an already-dead target is a benign
// no-op with a clear report, and a missing record is a clear error.
//
// Neither path writes a marker: with no result file present, a killed dispatch
// derives `orphaned` from the dead pid / dead pane on the next status read.
func runDispatchKill(cmd *cobra.Command, changeArg, stage string) error {
	dir, _, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	rec, err := dispatch.Load(dir, stage)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dispatch for %s/%s", changeArg, stage)
		}
		return err
	}

	if rec.IsPane() {
		// Killing the tmux pane takes the interactive worker down with it — the
		// pane's process group is tmux's to reap, so there is no separate
		// signalling. An already-gone pane is the benign no-op case.
		if !dispatch.PaneAlive(rec.Pane, rec.Server) {
			fmt.Fprintf(cmd.OutOrStdout(), "dispatch %s/%s already dead (pane %s); nothing to kill\n", changeArg, stage, rec.Pane)
			return nil
		}
		if err := dispatch.KillPane(rec.Pane, rec.Server); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "killed %s/%s (pane %s)\n", changeArg, stage, rec.Pane)
		return nil
	}

	// Idempotent: if the process group is already gone, report the benign no-op
	// rather than erroring. Alive() gates the "already dead" report; KillGroup
	// itself treats ESRCH as benign (a race between the probe and the signal).
	if !dispatch.Alive(rec.PID) {
		fmt.Fprintf(cmd.OutOrStdout(), "dispatch %s/%s already dead (pid %d); nothing to kill\n", changeArg, stage, rec.PID)
		return nil
	}

	if err := dispatch.KillGroup(rec.PGID); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "killed %s/%s (pgid %d)\n", changeArg, stage, rec.PGID)
	return nil
}
