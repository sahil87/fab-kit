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
		Short: "Kill the dispatch's process group (idempotent)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchKill(cmd, args[0], args[1])
		},
	}
}

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
