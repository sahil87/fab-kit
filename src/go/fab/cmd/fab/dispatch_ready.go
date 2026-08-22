package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func dispatchReadyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ready <change> <stage>",
		Short: "Probe whether an opened pane worker can accept typed input (ready / booting / parked)",
		Long: "Answer one question about a pane opened by `fab dispatch open`: can it accept\n" +
			"typed input right now?\n\n" +
			"The probe is purely MECHANICAL — it types a sentinel literally, checks whether\n" +
			"the sentinel echoed, clears it with C-u, and looks at whether the screen is\n" +
			"still moving. It carries no table of known dialogs, presses no other key, and\n" +
			"answers nothing: dialog text is a version treadmill, and a half-matched pattern\n" +
			"pressing Enter into an unknown screen is worse than stalling.\n\n" +
			"  ready    the sentinel echoed — hand the worker its prompt with `fab dispatch deliver`\n" +
			"  booting  no echo, but the screen is blank or still changing — wait and re-probe\n" +
			"  parked   no echo on a stable screen — a dialog, survey, login wall, or wedged\n" +
			"           process is holding the input; the snippet below shows what\n\n" +
			"Deciding what a parked screen wants is the orchestrator's judgment, which is\n" +
			"why every non-ready report carries the pane, its socket, and a capture snippet.\n" +
			"All three answers exit 0 — the report string is the sole discriminator, as with\n" +
			"`fab dispatch wait`. Re-running the probe is always safe.",
		Example: `  fab dispatch ready b91h apply`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchReady(cmd, args[0], args[1])
		},
	}
}

// runDispatchReady loads the record, insists on a live pane dispatch that may be
// typed into, and reports the gate's classification.
//
// The mid-stage guard is shared with `deliver` because the probe is a SENDER too:
// it types the sentinel and presses C-u, so against a delivered worker still
// executing its stage it would be exactly the input injection the contract's
// carve-out stops at successful delivery.
//
// Non-zero exit is reserved for REAL errors — no dispatch record, a headless
// record, a dead pane, a mid-stage worker, or a tmux failure — never for a
// classification, mirroring `wait`'s rule that an observed answer is a success
// however inconvenient the answer is.
func runDispatchReady(cmd *cobra.Command, changeArg, stage string) error {
	rec, dir, _, err := loadPaneDispatch(changeArg, stage, "ready")
	if err != nil {
		return err
	}
	if err := refuseMidStageDelivery(dir, changeArg, stage, rec); err != nil {
		return err
	}

	state, snippet, err := pane.NewGate(rec.Server).Probe(rec.Pane)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, state)
	if state == pane.ReadyReady {
		return nil
	}
	// The pane and socket are printed for the judgment rounds that follow a
	// non-ready answer: answering a wall means `tmux [-L <server>] send-keys -t
	// <pane> …` (the socket also rides `status --json` as `server`).
	fmt.Fprintf(out, "pane: %s\n", rec.Pane)
	if rec.Server != "" {
		fmt.Fprintf(out, "server: %s\n", rec.Server)
	}
	// No header over an empty snippet: a pane that has drawn nothing yet is the
	// ordinary `booting` case, and a header with nothing under it reads as a
	// truncated report rather than as "there is nothing on the screen".
	if snippet != "" {
		fmt.Fprintf(out, "--- last %d lines ---\n", pane.SnippetLines)
		fmt.Fprint(out, snippet)
	}
	return nil
}

// loadPaneDispatch resolves a change/stage to a PANE dispatch record with a live
// pane, or an actionable error. It returns the record, its dispatch dir, and the
// resolved 4-char change ID — the id because the family's report lines name
// `<id>/<stage>` whatever spelling of the change the caller typed. Shared by
// `ready` and `deliver`, the two verbs that only make sense against a pane.
//
// The two refusals are distinct on purpose: a headless record means the caller
// wanted a different command entirely, while a dead pane means the worker this
// verb would have talked to is gone — which `status` already reports as
// `orphaned` and `restart` already knows how to recover. The liveness read is
// identity-checked (paneWorkerAlive): a restart-aliased pane is an impostor, so
// the refusal fires for it too — no sentinel or pointer is ever typed into an
// unrelated pane.
func loadPaneDispatch(changeArg, stage, verb string) (rec *dispatch.Dispatch, dir, id string, err error) {
	dir, id, err = resolveDispatchDir(changeArg)
	if err != nil {
		return nil, "", "", err
	}
	rec, err = loadDispatchRecord(dir, changeArg, stage)
	if err != nil {
		return nil, "", "", err
	}
	if !rec.IsPane() {
		return nil, "", "", fmt.Errorf("%s/%s is a headless dispatch; `fab dispatch %s` applies only to pane workers (a headless worker is handed its prompt on stdin by `fab dispatch start`)",
			changeArg, stage, verb)
	}
	if !paneWorkerAlive(rec) {
		return nil, "", "", fmt.Errorf("pane %s for %s/%s is gone; run `fab dispatch restart %s %s` to open a fresh worker",
			rec.Pane, changeArg, stage, changeArg, stage)
	}
	return rec, dir, id, nil
}
