package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
	"github.com/sahil87/fab-kit/src/go/fab/internal/proc"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
	"github.com/spf13/cobra"
)

// gcInterval is the throttle window for the inline GC sweep folded into each
// hook handler's runtime mutation (runtime.UpdateAgent). Matches the
// 180-second value documented in the spec.
const gcInterval = 180 * time.Second

// envTmux is the environment variable name for the tmux socket path.
// envTmuxPane is the environment variable name for the current pane ID.
const (
	envTmux     = "TMUX"
	envTmuxPane = "TMUX_PANE"
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

// parseTmuxServer extracts the basename of the $TMUX socket path. A $TMUX
// value looks like "/tmp/tmux-1001/fabKit,8671,0" — we take the first
// comma-separated component and return its basename ("fabKit"). Returns
// empty when $TMUX is unset or malformed.
func parseTmuxServer(tmuxEnv string) string {
	if tmuxEnv == "" {
		return ""
	}
	first := tmuxEnv
	if idx := strings.Index(first, ","); idx >= 0 {
		first = first[:idx]
	}
	if first == "" {
		return ""
	}
	return filepath.Base(first)
}

// resolveClaudePID returns a pointer to Claude's PID resolved via the
// platform-split grandparent walker, or nil on failure. Nil is preserved in
// the serialized entry so GC does not attempt liveness checks on an absent
// field.
func resolveClaudePID() *int {
	pid, err := proc.ClaudePID()
	if err != nil || pid <= 0 {
		return nil
	}
	return &pid
}

// resolveActiveChangeFolder returns the folder name of the active change, or
// empty string if none is active. Swallows all errors — discussion-mode
// agents MUST NOT fail the hook just because no change is set.
func resolveActiveChangeFolder(fabRoot string) string {
	folder, err := resolve.ToFolder(fabRoot, "")
	if err != nil {
		return ""
	}
	return folder
}

// buildAgentEntry assembles an AgentEntry from the current hook invocation
// context. Only IdleSince is set by the caller (Stop uses now; others pass
// nil). All other fields are pulled from the environment and the grandparent
// walker — missing fields are omitted from the written record.
func buildAgentEntry(fabRoot string, idleSince *int64, transcriptPath string) runtime.AgentEntry {
	return runtime.AgentEntry{
		Change:         resolveActiveChangeFolder(fabRoot),
		IdleSince:      idleSince,
		PID:            resolveClaudePID(),
		TmuxServer:     parseTmuxServer(os.Getenv(envTmux)),
		TmuxPane:       os.Getenv(envTmuxPane),
		TranscriptPath: transcriptPath,
	}
}

func hookSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Delete agent entry on session start",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := hooklib.ParseSessionPayload(cmd.InOrStdin())
			if err != nil || payload.SessionID == "" {
				return nil // swallow
			}

			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return nil // swallow
			}

			// Single merged call: entry clear + inline GC in one load/save.
			_ = runtime.ClearAgent(fabRoot, payload.SessionID, gcInterval)
			return nil
		},
	}
}

func hookStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Record agent idle entry on stop",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := hooklib.ParseSessionPayload(cmd.InOrStdin())
			if err != nil || payload.SessionID == "" {
				return nil // swallow
			}

			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return nil // swallow
			}

			now := time.Now().Unix()
			entry := buildAgentEntry(fabRoot, &now, payload.TranscriptPath)
			// Single merged call: entry write + inline GC in one load/save.
			_ = runtime.WriteAgent(fabRoot, payload.SessionID, entry, gcInterval)
			return nil
		},
	}
}

func hookUserPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user-prompt",
		Short: "Clear idle_since on user prompt, preserving other entry fields",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := hooklib.ParseSessionPayload(cmd.InOrStdin())
			if err != nil || payload.SessionID == "" {
				return nil // swallow
			}

			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return nil // swallow
			}

			// Single merged call: idle clear + inline GC in one load/save.
			_ = runtime.ClearAgentIdle(fabRoot, payload.SessionID, gcInterval)
			return nil
		},
	}
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
	return &cobra.Command{
		Use:    "artifact-write",
		Short:  "Deprecated no-op — bookkeeping moved to `fab status refresh`; hook registration removed by the 2.11.0 migration, shim kept for one release",
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No-op: consume nothing, emit nothing, exit 0.
			return nil
		},
	}
}

func hookSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Register hook commands into .claude/settings.local.json",
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
