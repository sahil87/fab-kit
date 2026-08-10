package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func dispatchDeliverCmd() *cobra.Command {
	var promptFile string
	cmd := &cobra.Command{
		Use:   "deliver <change> <stage>",
		Short: "Hand an opened pane worker its prompt, verifying every step",
		Long: "Type a one-line pointer to the stage prompt into a pane worker and VERIFY that\n" +
			"it landed: readiness probe, clear the input line, type the pointer, check that\n" +
			"it echoed, press Enter, confirm the worker reacted. Any failed check costs one\n" +
			"attempt; there is exactly one retry, and a second failure reports the pane's\n" +
			"screen instead of retrying again.\n\n" +
			"Verification is the point. The pointer used to ride the launch command as a\n" +
			"positional argument, which is fire-and-forget: a CLI that silently drops it\n" +
			"leaves the worker at an empty prompt while the dispatch reads `running`.\n\n" +
			"`--prompt-file` points the worker at a different prompt — the pane-arm RESUME:\n" +
			"an orchestrator writes a continuation prompt (triaged findings plus the rework\n" +
			"action) and delivers it into the still-live apply pane instead of paying a cold\n" +
			"start. A verified delivery clears the previous attempt's result file, so the\n" +
			"dispatch reads `running` again; a delivery that never verifies puts it back,\n" +
			"leaving the dispatch exactly as it was found.",
		Example: `  # Initial delivery, after the readiness gate reports ready
  fab dispatch deliver b91h apply

  # Rework-cycle continuation into the same live pane
  fab dispatch deliver b91h apply --prompt-file .fab-dispatch/b91h/apply-continuation.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchDeliver(cmd, args[0], args[1], promptFile)
		},
	}
	cmd.Flags().StringVar(&promptFile, "prompt-file", "",
		"Deliver a pointer to this prompt file instead of the stage's own {stage}-prompt.md (the rework-cycle continuation path)")
	return cmd
}

// runDispatchDeliver applies the delivery guards, runs the verified send-keys
// choreography, and — only then — supersedes the previous attempt's completion
// signal and flips the record's delivery marker.
//
// Both writes happen only AFTER verification succeeds: a failed delivery must
// leave `delivered` unset so the caller can tell "the worker never got its
// prompt" from "the worker got it and failed at the work" — the exact distinction
// the old spawn-time argument could not express — and must leave the previous
// result in place so the dispatch does not end up in a state every recovery verb
// refuses.
func runDispatchDeliver(cmd *cobra.Command, changeArg, stage, promptFile string) error {
	rec, dir, id, err := loadPaneDispatch(changeArg, stage, "deliver")
	if err != nil {
		return err
	}
	if err := refuseMidStageDelivery(dir, changeArg, stage, rec); err != nil {
		return err
	}

	pointerPath, err := deliveryPointerPath(dir, stage, promptFile)
	if err != nil {
		return err
	}

	// Take the PREVIOUS attempt's completion signals out of the way before typing
	// anything: on a continuation the stale {stage}-result.yaml is what makes the
	// dispatch read `done`, so leaving it in place would let the orchestrator's
	// next `wait` return immediately on the last cycle's result. They are STASHED
	// rather than deleted, because only a verified delivery has actually
	// superseded them — see restoreStash below.
	stash, err := stashSignals(dispatch.ResultPath(dir, stage), dispatch.ExitPath(dir, stage))
	if err != nil {
		// A stash that failed part-way has already removed the entries it got
		// through, and nothing has been typed yet — so put them back before
		// returning, exactly as the failed-delivery path does.
		restoreStash(cmd, stash)
		return err
	}

	warnings, snippet, err := pane.NewGate(rec.Server).
		Deliver(rec.Pane, pane.PointerPrompt(pointerPath))
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", w)
	}
	if err != nil {
		// A delivery that never verified leaves the dispatch exactly as it found
		// it. Without this, a failed CONTINUATION would strand the record at
		// `delivered: true` with no result — derived state `running` — which both
		// recovery paths refuse: `deliver` via the mid-stage guard below and `open`
		// via its already-running check. The fresh-dispatch fallback the pane-arm
		// resume rests on (resume is an optimization, never correctness-bearing)
		// would then need an undocumented `fab dispatch kill` to become executable
		// at all.
		restoreStash(cmd, stash)
		if snippet != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "--- pane %s, last %d lines ---\n%s",
				rec.Pane, pane.SnippetLines, snippet)
		}
		return err
	}

	rec.Delivered = true
	if err := dispatch.Save(dir, stage, rec); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "delivered %s/%s (pane %s, prompt %s)\n",
		id, stage, rec.Pane, pointerPath)
	return nil
}

// stashedSignal is one completion signal held in memory across a delivery
// attempt, so a delivery that fails can put it back.
type stashedSignal struct {
	path string
	data []byte
	mode os.FileMode
}

// stashSignals removes the named completion signals and remembers their bytes.
// An absent file — the initial-delivery case — is simply nothing to stash and
// nothing to restore.
//
// A failure part-way through returns the signals stashed SO FAR alongside the
// error, because those files are already gone from disk: dropping them would
// lose the previous cycle's result for good and strand the dispatch at
// `delivered: true` with no result — the unrecoverable state this whole
// stash/restore machinery exists to prevent. Every caller therefore restores on
// the error path too.
func stashSignals(paths ...string) ([]stashedSignal, error) {
	var stash []stashedSignal
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return stash, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return stash, err
		}
		if err := os.Remove(p); err != nil {
			return stash, err
		}
		stash = append(stash, stashedSignal{path: p, data: data, mode: info.Mode().Perm()})
	}
	return stash, nil
}

// restoreStash writes stashed completion signals back, restoring the state the
// dispatch was in before an unverified delivery disturbed it. A restore that
// itself fails is a warning rather than the returned error: the caller is
// already reporting why the delivery did not happen, and that cause is the more
// actionable of the two.
func restoreStash(cmd *cobra.Command, stash []stashedSignal) {
	for _, s := range stash {
		if err := os.WriteFile(s.path, s.data, s.mode); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not restore %s after the failed delivery: %v\n",
				s.path, err)
		}
	}
}

// refuseMidStageDelivery is the code-level expression of the contract's
// no-input-injection rule (docs/specs/harness-adapters.md § 3): between `open`
// and a successful `deliver` the pane holds no stage context and may be typed
// into, but a DELIVERED worker executing its stage never may.
//
// The permitted cases are therefore exactly two: an undelivered pane (the initial
// delivery), and a delivered pane whose result is present — `done`, meaning the
// worker finished and is sitting at its prompt, which is the sanctioned
// continuation case the rework loop uses.
//
// It guards BOTH mechanical senders — `deliver` and `ready`, which types the
// sentinel — so no `fab dispatch` verb can reach a mid-stage worker's keyboard.
func refuseMidStageDelivery(dir, changeArg, stage string, rec *dispatch.Dispatch) error {
	if !rec.Delivered {
		return nil
	}
	if dispatch.ResultPresent(dir, stage) {
		return nil
	}
	return fmt.Errorf("%s/%s already has its prompt and is still running (pane %s); the pipeline never types into a worker mid-stage — wait for its result, or `fab dispatch kill %s %s` first",
		changeArg, stage, rec.Pane, changeArg, stage)
}

// deliveryPointerPath resolves what the delivered pointer names, REPO-RELATIVE
// because the worker pane's cwd is the repo root.
//
// With no flag it is the stage's own persisted prompt (written by `open`). With
// --prompt-file it is the caller's continuation prompt, whose existence is
// checked here: a pointer at a file that is not there would type cleanly, verify
// cleanly, and leave the worker reading nothing — the silent failure this whole
// choreography exists to prevent.
func deliveryPointerPath(dir, stage, promptFile string) (string, error) {
	path := dispatch.PromptPath(dir, stage)
	if promptFile != "" {
		path = repoAnchored(dir, promptFile)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no prompt at %s — nothing to deliver; open the worker with `fab dispatch open` (prompt on stdin), or point --prompt-file at a file that exists", path)
		}
		return "", err
	}
	return repoRelative(path), nil
}

// repoAnchored resolves a caller-supplied --prompt-file the way the flag is
// documented and exampled — REPO-RELATIVE, like the pointer that gets typed.
// Left raw, os.Stat below would read it against the CALLER's cwd instead, and
// `fab` runs from anywhere inside the repo (resolve.FabRoot walks upward), so a
// continuation delivered from a subdirectory would fail the existence check on a
// file that is right there.
//
// The anchor is derived from the dispatch dir rather than a second
// resolve.FabRoot() walk: dir is always `<repo root>/.fab-dispatch/<id>` (see
// dispatch.DirFor), so the root is exactly two levels up and cannot fail to
// resolve — whereas a FabRoot error would fall back to the cwd-relative reading
// this function exists to remove. An absolute path is already unambiguous and
// passes through untouched.
func repoAnchored(dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(filepath.Dir(dir)), path)
}

// repoRelative renders path relative to the repository root when it can, so the
// typed pointer stays short and portable across worktrees. An unresolvable root
// (or a path outside it) falls back to the path as given, which still reads
// correctly from the pane's cwd when absolute.
//
// The escape test is `..`, the repo's own idiom (internal/memoryindex's
// gitRelPath): a leading-dot test would reject every `.fab-dispatch/...` path —
// which is to say every pointer this function is ever asked to render.
func repoRelative(path string) string {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(filepath.Dir(fabRoot), abs)
	if err != nil || rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}
