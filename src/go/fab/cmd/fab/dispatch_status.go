package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

func dispatchStatusCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "status <change> <stage>",
		Short: "Report the dispatch state: running / done / failed / failed (no-result) / orphaned",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchStatus(cmd, args[0], args[1], jsonFlag)
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

// dispatchStatusJSON is the --json output shape for `fab dispatch status`.
//
// Mode discriminates the two observation models so a consumer knows which state
// subset to expect: a `pane` dispatch can only ever report running / done /
// orphaned (no exit-code channel), while a `headless` one can report all five.
// The mode-specific identity fields are omitempty, so a headless object is
// byte-identical to the pre-pane-mode shape apart from the added `mode` key —
// this surface's documented contract is additive evolution with no
// schema_version.
type dispatchStatusJSON struct {
	Change string `json:"change"`
	Stage  string `json:"stage"`
	State  string `json:"state"`
	Mode   string `json:"mode"`
	PID    int    `json:"pid,omitempty"`
	PGID   int    `json:"pgid,omitempty"`
	Pane   string `json:"pane,omitempty"`
	Window string `json:"window,omitempty"`
	Exit   *int   `json:"exit,omitempty"`
}

func runDispatchStatus(cmd *cobra.Command, changeArg, stage string, jsonFlag bool) error {
	dir, id, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	rec, err := dispatch.Load(dir, stage)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no dispatch for %s/%s (run `fab dispatch start` first)", changeArg, stage)
		}
		return err
	}

	out := dispatchStatusJSON{
		Change: id,
		Stage:  stage,
		Mode:   string(rec.Mode()),
	}

	var state dispatch.State
	if rec.IsPane() {
		// Pane mode: result-file presence + pane liveness, no exit file. An
		// unobservable pane (killed, or its whole tmux server gone) reads as not
		// alive, so a resultless pane dispatch degrades to `orphaned` rather than
		// erroring out of status.
		state = dispatch.DerivePaneState(
			dispatch.ResultPresent(dir, stage),
			dispatch.PaneAlive(rec.Pane, rec.Server),
		)
		out.Pane = rec.Pane
		out.Window = rec.Window
	} else {
		exitPresent, exitCode, err := dispatch.ReadExit(dir, stage)
		if err != nil {
			return err
		}
		state = dispatch.DeriveState(exitPresent, exitCode,
			dispatch.ResultPresent(dir, stage), dispatch.Alive(rec.PID))
		out.PID = rec.PID
		out.PGID = rec.PGID
		if exitPresent {
			out.Exit = &exitCode
		}
	}
	out.State = string(state)

	if jsonFlag {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(state))
	return nil
}
