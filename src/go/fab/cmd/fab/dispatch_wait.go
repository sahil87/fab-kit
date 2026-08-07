package main

import (
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

// dispatchWaitCmd is `status`'s BLOCKING sibling: it re-derives the same state on
// an internal tick and returns as soon as that state leaves `running`.
//
// It exists to convert the CLI-adapter dispatch path from long-poll back to push.
// An orchestrator that polls `fab dispatch status` every 30s burns a full
// inference turn per poll — a 90-minute stage costs ~180 idle turns of pure spin,
// each one also growing the transcript. Run as a BACKGROUND command, one blocking
// `wait` collapses all of those into a single wake-up when something actually
// happens (the harness re-invokes the orchestrator when the command exits). Run in
// the foreground on a harness with no such seam, `--timeout 300` still trades ten
// turns for one.
//
// The skill wiring is `_preamble.md` § CLI-Adapter Dispatch step 2; `--timeout`
// there carries the peek-on-suspicion cadence (300s = the former 10 polls × 30s),
// not a poll interval.
func dispatchWaitCmd() *cobra.Command {
	var (
		jsonFlag    bool
		timeoutSecs int
	)
	cmd := &cobra.Command{
		Use:   "wait <change> <stage>",
		Short: "Block until the dispatch leaves `running`, then report its state",
		Long: "Block until the dispatch's state leaves `running`, then print it exactly as\n" +
			"`fab dispatch status` does. State is re-derived on an internal ~2s tick through\n" +
			"the same loader and derivations `status` uses, so the two can never disagree.\n\n" +
			"An already-terminal dispatch returns immediately (idempotent — safe to re-arm).\n" +
			"--timeout <secs> bounds the block: on expiry the still-current state (`running`)\n" +
			"is printed and the exit code is 0, so the STATE STRING is the timeout\n" +
			"discriminator. Only real errors (no dispatch record, unresolvable change) exit\n" +
			"non-zero. Without --timeout the wait is unbounded.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchWait(cmd, args[0], args[1], timeoutSecs, jsonFlag)
		},
	}
	cmd.Flags().IntVar(&timeoutSecs, "timeout", 0,
		"Upper bound in seconds; on expiry print the current state and exit 0 (0 = wait indefinitely)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func runDispatchWait(cmd *cobra.Command, changeArg, stage string, timeoutSecs int, jsonFlag bool) error {
	dir, id, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}
	// The record is loaded ONCE: {stage}.yaml is written by start/restart and is
	// immutable for the lifetime of an attempt, so only the derived SIGNALS (exit
	// file, result file, pid/pane liveness) need re-reading per tick.
	rec, err := loadDispatchRecord(dir, changeArg, stage)
	if err != nil {
		return err
	}

	// The observation carried out of the loop, so the printed object is the one
	// the terminal (or timed-out) observation produced — identity keys included.
	var out dispatchStatusJSON
	observe := func() (dispatch.State, error) {
		observed, obsErr := observeDispatch(dir, id, stage, rec)
		if obsErr != nil {
			return "", obsErr
		}
		out = observed
		return dispatch.State(observed.State), nil
	}

	if _, err := dispatch.Wait(cmd.Context(), observe,
		dispatch.TickInterval, time.Duration(timeoutSecs)*time.Second); err != nil {
		return err
	}

	return renderDispatchState(cmd, out, jsonFlag)
}
