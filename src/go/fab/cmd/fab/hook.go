package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/proc"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
	"github.com/sahil87/fab-kit/src/go/fab/internal/score"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
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

// hookArtifactWriteCmd handles the PostToolUse hook for Write/Edit tools.
// Unlike the three session-scoped hooks above, this handler parses a
// different payload shape (tool_input.file_path) — see hooklib.ParsePayload.
// It does not participate in _agents writes; it only manages artifact
// bookkeeping and git staging for status/history files.
func hookArtifactWriteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "artifact-write",
		Short: "Artifact bookkeeping on PostToolUse Write/Edit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := hooklib.ParsePayload(cmd.InOrStdin())
			if err != nil || filePath == "" {
				return nil // swallow
			}

			match, ok := hooklib.MatchArtifactPath(filePath)
			if !ok {
				return nil // not a fab artifact
			}

			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return nil // swallow
			}

			// Verify the change folder resolves (the only fab/changes
			// directory scan on this path — downstream consumers reuse the
			// exact folder name, mz4q F02).
			_, err = resolve.ToFolder(fabRoot, match.ChangeFolder)
			if err != nil {
				return nil // swallow
			}

			// Load → mutate in memory → exactly one Save, all under the
			// cross-process status lock so the bookkeeping cannot revert a
			// concurrent fab status transition from another pane (mz4q
			// F02/F03).
			statusPath := filepath.Join(fabRoot, "changes", match.ChangeFolder, ".status.yaml")
			var contextParts []string
			lockErr := lockfile.WithLock(statusPath, func() error {
				statusFile, err := sf.Load(statusPath)
				if err != nil {
					return err
				}
				parts, dirty := artifactBookkeeping(fabRoot, filePath, match, statusFile)
				contextParts = parts
				if dirty {
					return statusFile.Save(statusPath)
				}
				return nil
			})
			if lockErr != nil {
				return nil // swallow
			}

			// Auto-stage status files so they don't block git operations
			changeDir := filepath.Join(fabRoot, "changes", match.ChangeFolder)
			repoRoot := filepath.Dir(fabRoot)
			_ = exec.Command("git", "-C", repoRoot, "add",
				filepath.Join(changeDir, ".status.yaml"),
				filepath.Join(changeDir, ".history.jsonl"),
			).Run()

			// Output additionalContext JSON
			if len(contextParts) > 0 {
				ctx := "Bookkeeping: " + strings.Join(contextParts, ", ")
				out := map[string]string{"additionalContext": ctx}
				data, err := json.Marshal(out)
				if err == nil {
					fmt.Fprintln(cmd.OutOrStdout(), string(data))
				}
			}

			return nil
		},
	}
}

// artifactBookkeeping performs per-artifact bookkeeping by mutating the
// in-memory StatusFile only. It returns context description parts and whether
// anything was mutated — the caller persists with exactly one Save (mz4q
// F02). The intake branch reuses the already-resolved folder and the single
// intake.md read for scoring (score.ComputeWithStatus).
func artifactBookkeeping(fabRoot, filePath string, match hooklib.ArtifactMatch, statusFile *sf.StatusFile) ([]string, bool) {
	var contextParts []string
	dirty := false

	// Resolve absolute path for reading file content
	repoRoot := filepath.Dir(fabRoot)
	var absPath string
	if filepath.IsAbs(filePath) {
		absPath = filePath
	} else {
		absPath = filepath.Join(repoRoot, filePath)
	}

	switch match.Artifact {
	case "intake.md":
		content, err := os.ReadFile(absPath)
		if err != nil {
			content = []byte{}
		}

		changeType := hooklib.InferChangeType(string(content))
		if err := status.ApplyChangeType(statusFile, changeType); err == nil {
			dirty = true
		}
		contextParts = append(contextParts, "type: "+changeType)

		changeDir := filepath.Join(fabRoot, "changes", match.ChangeFolder)
		result, err := score.ComputeWithStatus(fabRoot, changeDir, content, statusFile)
		if err == nil {
			dirty = true
			contextParts = append(contextParts, fmt.Sprintf("score: %.1f", result.Score))
		}

	case "plan.md":
		content, err := os.ReadFile(absPath)
		if err != nil {
			content = []byte{}
		}

		hasTasks := hooklib.HasSectionHeading(string(content), hooklib.SectionTasks)
		hasAcceptance := hooklib.HasSectionHeading(string(content), hooklib.SectionAcceptance)

		// Always set generated=true if the file exists with at least the ## Tasks heading.
		if hasTasks {
			if err := status.ApplyAcceptance(statusFile, "generated", "true"); err == nil {
				dirty = true
			}
		}

		// Defensive: only update task_count when ## Tasks is present.
		if hasTasks {
			taskCount := hooklib.CountSectionItemsBounded(string(content), hooklib.SectionTasks)
			if err := status.ApplyAcceptance(statusFile, "task_count", fmt.Sprintf("%d", taskCount)); err == nil {
				dirty = true
			}
			contextParts = append(contextParts, fmt.Sprintf("plan tasks: %d", taskCount))
		}

		// Defensive: only update acceptance counts when ## Acceptance is present.
		if hasAcceptance {
			acceptanceCount := hooklib.CountSectionItemsBounded(string(content), hooklib.SectionAcceptance)
			acceptanceCompleted := hooklib.CountCompletedSectionItemsBounded(string(content), hooklib.SectionAcceptance)
			if err := status.ApplyAcceptance(statusFile, "acceptance_count", fmt.Sprintf("%d", acceptanceCount)); err == nil {
				dirty = true
			}
			if err := status.ApplyAcceptance(statusFile, "acceptance_completed", fmt.Sprintf("%d", acceptanceCompleted)); err == nil {
				dirty = true
			}
			contextParts = append(contextParts, fmt.Sprintf("plan acceptance: %d/%d", acceptanceCompleted, acceptanceCount))
		}
	}

	return contextParts, dirty
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
