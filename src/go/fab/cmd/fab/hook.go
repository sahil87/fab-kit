package main

import (
	"fmt"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func hookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Claude Code hook subcommands",
	}

	cmd.AddCommand(
		hookSessionStartCmd(),
		hookStopCmd(),
		hookUserPromptCmd(),
		hookArtifactWriteCmd(),
		hookSyncCmd(),
	)

	return cmd
}

// noOpHookShim builds a session-scoped hook handler that consumes nothing,
// emits nothing, and exits 0. The three session-scoped hooks
// (stop / user-prompt / session-start) used to WRITE `.fab-runtime.yaml`
// `_agents` state, but fab no longer PRODUCES agent lifecycle state — it is a
// pure consumer of the `@rk_agent_state` tmux pane-option convention written
// by run-kit's `rk agent-setup` (see internal/pane and the pane readers). The
// `_agents` write pipeline (WriteAgent/ClearAgent/ClearAgentIdle, the inline
// GC sweep, the grandparent PID walker, and the internal/runtime +
// internal/proc packages) was deleted wholesale.
//
// These handlers survive for ONE release as silent no-op shims for
// un-migrated projects whose `.claude/settings.local.json` still registers
// them as SessionStart/Stop/UserPromptSubmit entries. The silence matters: an
// *unregistered* `fab hook <x>` subcommand exits 0 but prints cobra help to
// stdout, which a still-registered hook entry would feed to Claude Code as
// noisy non-JSON additionalContext. The shim emits nothing, avoiding that
// until the 2.13.6-to-2.14.0 migration removes the settings entries and
// `hooklib.Sync` stops registering them. This mirrors the `artifact-write`
// removal precedent (y022, 2.10.1-to-2.11.0). Full subcommand removal is a
// follow-up.
func noOpHookShim(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Short:  short,
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No-op: consume nothing, emit nothing, exit 0.
			return nil
		},
	}
}

func hookSessionStartCmd() *cobra.Command {
	return noOpHookShim("session-start", "Deprecated no-op — agent-state production divested to run-kit's @rk_agent_state convention; shim kept for one release")
}

func hookStopCmd() *cobra.Command {
	return noOpHookShim("stop", "Deprecated no-op — agent-state production divested to run-kit's @rk_agent_state convention; shim kept for one release")
}

func hookUserPromptCmd() *cobra.Command {
	return noOpHookShim("user-prompt", "Deprecated no-op — agent-state production divested to run-kit's @rk_agent_state convention; shim kept for one release")
}

// hookArtifactWriteCmd is a one-release no-op shim retained for un-migrated
// projects whose .claude/settings.local.json still registers `fab hook
// artifact-write` as a PostToolUse (Write/Edit) entry. Artifact bookkeeping is
// no longer hook-owned: the change_type/confidence/plan-count recompute moved
// to the pull-based `fab status refresh` (internal/refresh), self-healed at the
// transition seams (fab status advance/finish, fab preflight). The registration
// is dropped from DefaultMappings and removed from existing settings by the
// 2.10.1-to-2.11.0 migration.
//
// The shim writes NOTHING to stdout and exits 0. This matters: an *unregistered*
// `fab hook <x>` subcommand exits 0 but prints cobra help text to stdout, which
// a still-registered PostToolUse entry feeds to Claude Code as additionalContext
// (invalid JSON — noisy). The silent shim avoids that until the migration
// removes the settings entry. It carries no bookkeeping, no payload parse, and
// no git staging (status/history are staged by /git-pr at ship, not here).
func hookArtifactWriteCmd() *cobra.Command {
	return noOpHookShim("artifact-write", "Deprecated no-op — bookkeeping moved to `fab status refresh`; hook registration removed by the 2.11.0 migration, shim kept for one release")
}

func hookSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Deprecated no-op — registers nothing and rewrites nothing; retained one release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// All hook subcommands exit 0 so they never block agent flows.
			// Unlike the session-scoped handlers (which swallow errors
			// silently), sync is setup-facing: failures are surfaced on
			// stderr — but still exit 0.
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "hook sync: %v\n", err)
				return nil
			}

			repoRoot := filepath.Dir(fabRoot)
			settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")

			result, err := hooklib.Sync(settingsPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "hook sync: %v\n", err)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), result.Message)
			return nil
		},
	}
}
