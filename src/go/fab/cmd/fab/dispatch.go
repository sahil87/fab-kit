package main

import (
	"fmt"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// dispatchCmd is the parent of the stage-worker process-manager command family:
// `fab dispatch <start|open|ready|deliver|restart|status|wait|logs|kill|reap|clean>
// [args...]`. It is the CLI adapter for cross-harness stage dispatch, in two launch
// modes — a detached headless process or an interactive tmux pane.
//
// The two modes have SEPARATE ENTRY VERBS, because they hand a worker its prompt
// in fundamentally different ways. `start` launches headless, piping the prompt on
// stdin: one step, no ambiguity. Pane mode takes three — `open` (spawn the pane,
// deliver nothing), `ready` (a mechanical echo probe the orchestrator loops over,
// answering any first-run wall itself), `deliver` (type the prompt pointer and
// verify it landed) — because a freshly spawned agent TUI may be booting or parked
// behind a trust dialog, and answering one is judgment the binary cannot do.
//
// `restart` is the recovery verb: it relaunches a non-running dispatch from the
// persisted prompt over `start`'s launch path, re-deriving the mode from the
// current environment; a pane landing performs the `open` step and hands the gate
// back. `wait` is `status`'s blocking sibling: it re-derives the same state on an
// internal tick so an orchestrator can be woken by a state change instead of
// polling for one. `reap` is the hygiene verb: it reclaims a DONE pane worker's
// tmux pane and is a reported no-op for every other dispatch (distinct from `kill`, the
// recovery verb, which is valid in any state). The family is parallel to, and
// independent of, `fab pane` / `fab operator` (which stay the operator's interactive
// path). See docs/specs/harness-adapters.md for the cross-adapter contract.
func dispatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Process manager for CLI-dispatched pipeline stages (headless or tmux-pane worker)",
		Long: "Process manager for CLI-dispatched stage workers:\n" +
			"start/open/ready/deliver/restart/status/wait/logs/kill/reap/clean.\n\n" +
			"The two launch modes have separate entries. `start` launches a detached\n" +
			"HEADLESS worker with the prompt on stdin. PANE mode takes three steps —\n" +
			"`open` spawns the pane without a prompt, `ready` probes whether it can accept\n" +
			"typed input (ready / booting / parked, with a capture snippet), and `deliver`\n" +
			"types the prompt pointer and verifies it landed. The gate exists because a\n" +
			"fresh agent TUI may be booting or parked behind a first-run wall.\n\n" +
			"`restart` relaunches a non-running dispatch from the persisted prompt,\n" +
			"re-deriving the mode from the current environment. `status` is the one-shot\n" +
			"probe; `wait` blocks until the state leaves `running` so a poll loop becomes a\n" +
			"single wake-up. `reap` reclaims a done pane worker's pane (never a live one,\n" +
			"and no state files). Tracks the worker under .fab-dispatch/{id}/ and exposes a\n" +
			"byte-stable poll surface. The headless launch is POSIX-only (v1).",
	}

	cmd.AddCommand(
		dispatchStartCmd(),
		dispatchOpenCmd(),
		dispatchReadyCmd(),
		dispatchDeliverCmd(),
		dispatchRestartCmd(),
		dispatchStatusCmd(),
		dispatchWaitCmd(),
		dispatchLogsCmd(),
		dispatchKillCmd(),
		dispatchReapCmd(),
		dispatchCleanCmd(),
	)

	return cmd
}

// resolveDispatchDir resolves <change> to its 4-char ID and returns the
// absolute .fab-dispatch/{id}/ directory (DirFor joins onto the absolute
// repoRoot) plus the resolved ID. Shared
// by status/logs/kill/reap (clean has its own multi-dir resolution; start/restart
// resolve inline in the shared launch path, which needs the repo root too). fabRoot
// is found via resolve.FabRoot; the repo root is its parent (the same
// derivation internal/archive uses for the .fab-status.yaml pointer).
func resolveDispatchDir(changeArg string) (dir, id string, err error) {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return "", "", err
	}
	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		return "", "", err
	}
	id = resolve.ExtractID(folder)
	if id == "" {
		return "", "", fmt.Errorf("could not extract change ID from %q", folder)
	}
	repoRoot := filepath.Dir(fabRoot)
	return dispatch.DirFor(repoRoot, id), id, nil
}
