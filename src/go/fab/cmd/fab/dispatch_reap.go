package main

import (
	"fmt"
	"io"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func dispatchReapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reap <change> <stage>",
		Short: "Reclaim a DONE pane worker's tmux pane; a reported no-op in every other case",
		Long: "Reclaim the tmux pane of a pane-mode stage worker that has finished.\n\n" +
			"A pane worker never exits on completion — it writes {stage}-result.yaml and sits\n" +
			"at its prompt — so across a pipeline the carved worker column fills with dead\n" +
			"panes. `reap` kills such a pane, and ONLY such a pane: it fires when the record\n" +
			"is pane-mode AND the derived state is `done` AND dispatch.reap_done is true.\n" +
			"Every other case is a no-op that reports its reason and exits 0.\n\n" +
			"Reap is NOT kill. It never terminates a running, orphaned, or failed dispatch,\n" +
			"and it removes no .fab-dispatch/ state — the record and result file stay, so the\n" +
			"dispatch still reads `done` afterwards. State cleanup remains archive-time\n" +
			"deletion plus explicit `fab dispatch clean`.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchReap(cmd, args[0], args[1])
		},
	}
}

// runDispatchReap applies the three-condition reap guard and, when it passes, kills
// the worker's tmux pane on the socket the record itself persisted (so a
// `--server`-started dispatch is reaped correctly with no flag).
//
// The WHOLE guard lives here rather than at the skill call site: the knob resolves
// through the four-tier config cascade (env > system ~/.fab-kit/config.yaml > project >
// defaults), which only Go can read — a skill parsing fab/project/config.yaml
// directly would miss the system layer, which is exactly where a machine-wide
// `both`-scope preference lives. The skill wiring is therefore unconditional and
// dumb, and every branch below is reachable from that one call.
//
// Only REAL errors exit non-zero (no dispatch record, unresolvable change), sharing
// `status`/`wait`'s message surface via loadDispatchRecord. Every no-op — headless
// record, non-`done` state, disabled knob, already-gone pane — is a one-line report
// and exit 0, mirroring `kill`'s idempotent no-op contract.
func runDispatchReap(cmd *cobra.Command, changeArg, stage string) error {
	dir, _, err := resolveDispatchDir(changeArg)
	if err != nil {
		return err
	}

	rec, err := loadDispatchRecord(dir, changeArg, stage)
	if err != nil {
		return err
	}

	// State is only meaningful for a pane record, and DecideReap short-circuits on
	// the mode before it looks at either later argument — so a headless record needs
	// neither its exit file read nor a pane-liveness probe.
	var state dispatch.State
	if rec.IsPane() {
		state = dispatch.DerivePaneState(
			dispatch.ResultPresent(dir, stage),
			dispatch.PaneAlive(rec.Pane, rec.Server),
		)
	}

	// The knob is the guard's THIRD condition, so it can only change the outcome once
	// the first two already hold — and resolving it eagerly would let an unreadable
	// config turn a headless or not-done no-op into a non-zero exit, breaking the
	// contract below. Resolve it only where it can matter; DecideReap short-circuits
	// on mode and then state, so the placeholder is never consulted in those cases.
	reapDone := config.DefaultDispatchReapDone
	if rec.IsPane() && state == dispatch.StateDone {
		reapDone = dispatchReapEnabled(cmd.ErrOrStderr())
	}

	out := cmd.OutOrStdout()
	switch verdict := dispatch.DecideReap(rec.IsPane(), state, reapDone); verdict {
	case dispatch.ReapSkipHeadless:
		fmt.Fprintf(out, "%s/%s is a headless dispatch; nothing to reap (the worker process already exited)\n", changeArg, stage)
		return nil
	case dispatch.ReapSkipNotDone:
		fmt.Fprintf(out, "%s/%s is %s; nothing to reap (reap fires only on done — use `fab dispatch kill` to terminate a live dispatch)\n", changeArg, stage, state)
		return nil
	case dispatch.ReapSkipDisabled:
		fmt.Fprintf(out, "dispatch.reap_done is false; keeping pane %s for %s/%s\n", rec.Pane, changeArg, stage)
		return nil
	}

	// Idempotent, exactly like `kill`: a pane killed by hand (or lost with its tmux
	// server) is a benign already-gone report rather than an error. The liveness probe
	// gates the report; KillPane itself also treats a missing pane as a no-op, so a
	// race between the two is harmless.
	if !dispatch.PaneAlive(rec.Pane, rec.Server) {
		fmt.Fprintf(out, "pane %s for %s/%s is already gone; nothing to reap\n", rec.Pane, changeArg, stage)
		return nil
	}
	if err := dispatch.KillPane(rec.Pane, rec.Server); err != nil {
		return err
	}
	fmt.Fprintf(out, "reaped pane %s for %s/%s\n", rec.Pane, changeArg, stage)
	return nil
}

// dispatchReapEnabled resolves dispatch.reap_done through the four-tier config
// cascade. It re-walks to the fab root rather than threading one out of
// resolveDispatchDir: the walk is a cheap upward directory search, and leaving that
// shared helper's signature alone keeps its three other call sites untouched.
//
// It NEVER fails. Reap's exit contract reserves non-zero for exactly two real errors
// — no dispatch record, unresolvable change — and the skill wiring calls reap
// unconditionally after every `done`, so an unreadable config must not turn pane
// hygiene into a pipeline failure. It warns and falls back to the knob's built-in
// default, which is what an absent key resolves to anyway.
func dispatchReapEnabled(warn io.Writer) bool {
	fabRoot, err := resolve.FabRoot()
	if err == nil {
		var cfg *config.Config
		if cfg, err = config.Load(fabRoot); err == nil {
			return cfg.GetDispatchReapDone()
		}
	}
	fmt.Fprintf(warn, "warning: could not resolve dispatch.reap_done (%v); using default %t\n",
		err, config.DefaultDispatchReapDone)
	return config.DefaultDispatchReapDone
}
