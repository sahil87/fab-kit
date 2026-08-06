package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/spf13/cobra"
)

// dispatchRestartCmd relaunches a NON-RUNNING dispatch from the prompt `start`
// already persisted — the pipeline's bounded recovery verb for a worker that died
// or was killed (post-retry provider exhaustion, OOM, a closed tmux window).
//
// It is `start` with ONE difference: the prompt comes from
// .fab-dispatch/{id}/{stage}-prompt.md instead of stdin, because the orchestrator
// that needs to restart may have lost the multi-thousand-token block prompt to
// compaction. Everything else is `start`'s — the same flags with the same
// exclusions, the same mode-selection ladder (mode re-derived from the CURRENT
// environment, so restarting an orphaned pane dispatch after a tmux server death
// correctly soft-falls-back to headless), the same refuse-if-running check, and the
// same output/record shape. A restart is a fresh attempt under the existing
// last-attempt-only semantics, so it introduces no new state string, no attempt
// history, and no `restarted:` marker.
func dispatchRestartCmd() *cobra.Command {
	// The flag surface is shared with `start` via addLaunchFlags (dispatch_start.go)
	// so the two subcommands cannot drift on flags, the --pane/--timeout guard, or
	// the mode ladder — the same single-sourcing runDispatchLaunch gives the tail.
	// Declared before the command literal so RunE can close over it.
	var f *launchFlags
	cmd := &cobra.Command{
		Use:   "restart <change> <stage>",
		Short: "Relaunch a non-running dispatch from its persisted prompt — same flags and mode ladder as start",
		Long: "Relaunch a stage worker from the prompt `fab dispatch start` persisted at\n" +
			".fab-dispatch/{id}/{stage}-prompt.md, so the caller does not need the block prompt\n" +
			"in context. Refuses a genuinely running dispatch (kill it first); overwrites a\n" +
			"completed/failed/orphaned one. The launch mode is re-derived from the CURRENT\n" +
			"environment by the same ladder `start` uses — the prior attempt's mode is not\n" +
			"inherited — so a restart after a tmux server death lands headless.",
		Example: `  # Recover an orphaned dispatch (mode auto-resolves from the environment)
  fab dispatch restart b91h apply

  # Force the relaunched worker headless even inside tmux
  fab dispatch restart b91h apply --headless

  # Relaunch into a watchable tmux window
  fab dispatch restart b91h review --pane`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, reason, err := f.resolveMode(cmd)
			if err != nil {
				return err
			}
			return runDispatchLaunch(cmd, args[0], args[1], f.timeout, mode, reason, f.server, promptFromStateDir)
		},
	}
	f = addLaunchFlags(cmd)
	return cmd
}

// promptFromStateDir is `restart`'s prompt source: the prompt `start` already
// persisted. It asks the shared launch path NOT to re-persist (persist=false) —
// the file IS the input, so rewriting it with its own bytes is a no-op that only
// risks corruption on a partial write.
//
// An absent file means there is nothing to relaunch, which is a clear error rather
// than an empty-prompt launch: the shared path has not written any state yet at
// this point (the refusal check ran, the record was not touched), so nothing is
// left behind.
func promptFromStateDir(_ *cobra.Command, dir, stage string) ([]byte, bool, error) {
	path := dispatch.PromptPath(dir, stage)
	prompt, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, fmt.Errorf("no persisted prompt at %s — nothing to relaunch; run `fab dispatch start` with the prompt on stdin", path)
		}
		return nil, false, fmt.Errorf("read persisted prompt: %w", err)
	}
	return prompt, false, nil
}
