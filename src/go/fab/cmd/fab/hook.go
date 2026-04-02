package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/hooklib"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
	"github.com/sahil87/fab-kit/src/go/fab/internal/score"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
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

func hookSessionStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "session-start",
		Short: "Clear agent idle state on session start",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clearIdleForActiveChange()
			return nil
		},
	}
}

func hookStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Set agent idle timestamp on stop",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return nil // swallow
			}

			folder, err := resolve.ToFolder(fabRoot, "")
			if err != nil {
				return nil // swallow
			}

			_ = runtime.SetIdle(fabRoot, folder)
			return nil
		},
	}
}

func hookUserPromptCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user-prompt",
		Short: "Clear agent idle state on user prompt",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clearIdleForActiveChange()
			return nil
		},
	}
}

func hookArtifactWriteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "artifact-write",
		Short: "Artifact bookkeeping on PostToolUse Write/Edit",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath, err := hooklib.ParsePayload(os.Stdin)
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

			// Verify the change folder resolves
			_, err = resolve.ToFolder(fabRoot, match.ChangeFolder)
			if err != nil {
				return nil // swallow
			}

			// Load status file
			statusPath := filepath.Join(fabRoot, "changes", match.ChangeFolder, ".status.yaml")
			statusFile, err := sf.Load(statusPath)
			if err != nil {
				return nil // swallow
			}

			contextParts := artifactBookkeeping(fabRoot, filePath, match, statusFile, statusPath)

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

// artifactBookkeeping performs per-artifact bookkeeping and returns context description parts.
func artifactBookkeeping(fabRoot, filePath string, match hooklib.ArtifactMatch, statusFile *sf.StatusFile, statusPath string) []string {
	var contextParts []string

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
		_ = status.SetChangeType(statusFile, statusPath, changeType)
		contextParts = append(contextParts, "type: "+changeType)

		result, err := score.Compute(fabRoot, match.ChangeFolder, "intake")
		if err == nil {
			contextParts = append(contextParts, fmt.Sprintf("score: %.1f", result.Score))
		}

	case "spec.md":
		result, err := score.Compute(fabRoot, match.ChangeFolder, "spec")
		if err == nil {
			contextParts = append(contextParts, fmt.Sprintf("score: %.1f", result.Score))
		}

	case "tasks.md":
		content, err := os.ReadFile(absPath)
		if err != nil {
			content = []byte{}
		}

		count := hooklib.CountUncheckedTasks(string(content))
		_ = status.SetChecklist(statusFile, statusPath, "total", fmt.Sprintf("%d", count))
		contextParts = append(contextParts, fmt.Sprintf("tasks total: %d", count))

	case "checklist.md":
		_ = status.SetChecklist(statusFile, statusPath, "generated", "true")

		content, err := os.ReadFile(absPath)
		if err != nil {
			content = []byte{}
		}

		count := hooklib.CountChecklistItems(string(content))
		_ = status.SetChecklist(statusFile, statusPath, "total", fmt.Sprintf("%d", count))
		_ = status.SetChecklist(statusFile, statusPath, "completed", "0")
		contextParts = append(contextParts, fmt.Sprintf("checklist generated, total: %d", count))
	}

	return contextParts
}

func hookSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Register hook commands into .claude/settings.local.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			repoRoot := filepath.Dir(fabRoot)
			settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")

			result, err := hooklib.Sync(settingsPath)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), result.Message)
			return nil
		},
	}
}

// clearIdleForActiveChange resolves the active change and clears its idle state.
// Swallows all errors.
func clearIdleForActiveChange() {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return
	}

	folder, err := resolve.ToFolder(fabRoot, "")
	if err != nil {
		return
	}

	_ = runtime.ClearIdle(fabRoot, folder)
}
