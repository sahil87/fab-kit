package main

import (
	"github.com/spf13/cobra"
)

// dispatchOpenCmd is PANE MODE'S ENTRY: it spawns the interactive worker's tmux
// pane and stops there, delivering no prompt.
//
// It exists as its own verb because handing a pane worker its prompt is a
// separate, VERIFIED step that Go cannot complete unattended. A freshly spawned
// agent TUI may be booting, or parked behind a trust dialog, a survey, or a login
// wall, and deciding what a given screen wants is judgment — so the sequence is
// `open` → `fab dispatch ready` (a mechanical echo probe the orchestrator loops
// over, answering walls itself) → `fab dispatch deliver` (the verified send-keys
// choreography). The previous single-shot `start --pane` embedded a pointer to
// the prompt file as a positional spawn argument, which is fire-and-forget: a CLI
// that silently drops it leaves a worker at an empty prompt while the dispatch
// reads `running`.
//
// Everything else is `start`'s pane path verbatim — the same resolution, the same
// two placement shapes, the same record-keyed column stacking, the same
// `fab-{id}-{stage}` identity, the same refuse-if-running check, the same
// stale-file clearing, and the same prompt persistence to {stage}-prompt.md,
// which `deliver` later points the worker at. Only the pointer argument is gone.
//
// Pane mode here is EXPLICIT, never a ladder result: `open` opens a pane or it
// errors. A missing prerequisite (unreachable tmux, no interactive_command on the
// resolved provider) is a hard error with nothing launched and nothing persisted,
// rather than a silent descent to headless — which is the opposite of what this
// verb's caller asked for.
func dispatchOpenCmd() *cobra.Command {
	// Declared before the command literal so RunE can close over it; addLaunchFlags
	// binds it to the registered flags before any RunE can run.
	var f *launchFlags
	cmd := &cobra.Command{
		Use:   "open <change> <stage>",
		Short: "Spawn a pane-mode stage worker WITHOUT delivering its prompt (pane mode's entry)",
		Long: "Open an interactive stage worker in a tmux pane and persist the stage prompt\n" +
			"from stdin, delivering nothing.\n\n" +
			"The worker's prompt is handed over afterwards by `fab dispatch deliver`, behind\n" +
			"the `fab dispatch ready` readiness gate — a freshly spawned agent TUI may still\n" +
			"be booting or parked behind a first-run wall, and answering one is the\n" +
			"orchestrator's judgment, not the binary's. The composed interactive_command is\n" +
			"launched verbatim, so prompt delivery no longer depends on whether a provider's\n" +
			"CLI accepts a positional prompt.\n\n" +
			"Pane mode is explicit here: unreachable tmux or a provider with no\n" +
			"interactive_command is a hard error, not a descent to headless.",
		Example: `  # Open the worker, gate it, then deliver its prompt
  fab dispatch open b91h apply < prompt.md
  fab dispatch ready b91h apply
  fab dispatch deliver b91h apply

  # Target a specific tmux socket (works from outside tmux)
  fab dispatch open b91h review --server work < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchLaunch(cmd, args[0], args[1], f, promptFromStdin)
		},
	}
	f = addLaunchFlags(cmd, paneForced)
	return cmd
}
