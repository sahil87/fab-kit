package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
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
// Delivered is pane-only and a POINTER so `false` is reportable: a pane dispatch
// is opened and delivered to in two steps, and "opened but not yet delivered" is
// exactly the case a consumer needs to see. A plain bool with omitempty would
// erase it. It is bookkeeping, never a state — `state` is derived without it.
// Server is pane-only and omitempty: a socket-scoped pane dispatch carries it so
// a consumer can assemble a socket-scoped `fab pane capture -L <server> <pane>`
// from --json alone; a default-socket or headless dispatch omits the key
// (additive evolution — the repo/window_id/pr_url precedent).
type dispatchStatusJSON struct {
	Change    string `json:"change"`
	Stage     string `json:"stage"`
	State     string `json:"state"`
	Mode      string `json:"mode"`
	PID       int    `json:"pid,omitempty"`
	PGID      int    `json:"pgid,omitempty"`
	Pane      string `json:"pane,omitempty"`
	Window    string `json:"window,omitempty"`
	Server    string `json:"server,omitempty"`
	Delivered *bool  `json:"delivered,omitempty"`
	Exit      *int   `json:"exit,omitempty"`
}

func runDispatchStatus(cmd *cobra.Command, changeArg, stage string, jsonFlag bool) error {
	dir, id, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}
	rec, err := loadDispatchRecord(dir, changeArg, stage)
	if err != nil {
		return err
	}

	out, err := observeDispatch(dir, id, stage, rec)
	if err != nil {
		return err
	}
	return renderDispatchState(cmd, out, jsonFlag)
}

// loadDispatchRecord loads {stage}.yaml, translating "no such file" into the
// family's actionable no-dispatch error. Shared by `status` and `wait` so their
// error surfaces are identical by construction (the wait contract: only REAL
// errors are non-zero, and a missing record is the same real error for both).
func loadDispatchRecord(dir, changeArg, stage string) (*dispatch.Dispatch, error) {
	rec, err := dispatch.Load(dir, stage)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no dispatch for %s/%s (run `fab dispatch start` for a headless worker, or `fab dispatch open` for a pane worker, first)", changeArg, stage)
		}
		return nil, err
	}
	return rec, nil
}

// observeDispatch derives the dispatch's current state and assembles the output
// object, branching on the record's DERIVED mode.
//
// This is the ONE derivation in the command layer: `fab dispatch wait` calls it
// on every tick, so `wait` and `status` cannot report different states for the
// same on-disk signals. Keeping it a single function (rather than duplicating the
// branch in the wait command) is what makes that guarantee structural.
func observeDispatch(dir, id, stage string, rec *dispatch.Dispatch) (dispatchStatusJSON, error) {
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
			pane.PaneAlive(rec.Pane, rec.Server),
		)
		out.Pane = rec.Pane
		out.Window = rec.Window
		out.Server = rec.Server
		delivered := rec.Delivered
		out.Delivered = &delivered
	} else {
		exitPresent, exitCode, err := dispatch.ReadExit(dir, stage)
		if err != nil {
			return out, err
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
	return out, nil
}

// renderDispatchState writes the observed state — the bare state string, or the
// indented JSON object under --json. Shared by `status` and `wait` so the JSON
// surface stays single-sourced (one struct, one encoder configuration).
func renderDispatchState(cmd *cobra.Command, out dispatchStatusJSON, jsonFlag bool) error {
	if jsonFlag {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Fprintln(cmd.OutOrStdout(), out.State)
	return nil
}
