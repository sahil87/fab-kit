package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func paneCaptureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture <pane>",
		Short: "Capture terminal content from a tmux pane with fab context enrichment",
		Args:  cobra.ExactArgs(1),
		RunE:  runPaneCapture,
	}
	cmd.Flags().IntP("lines", "l", 50, "Number of lines to capture")
	cmd.Flags().Bool("json", false, "Output as JSON with metadata")
	cmd.Flags().Bool("raw", false, "Output raw captured text only")
	cmd.MarkFlagsMutuallyExclusive("json", "raw")
	return cmd
}

// captureJSON is the JSON output structure for pane capture.
type captureJSON struct {
	Pane              string  `json:"pane"`
	Lines             int     `json:"lines"`
	Content           string  `json:"content"`
	Worktree          string  `json:"worktree"`
	Change            *string `json:"change"`
	Stage             *string `json:"stage"`
	AgentState        *string `json:"agent_state"`
	AgentIdleDuration *string `json:"agent_idle_duration"`
}

func runPaneCapture(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	lines, _ := cmd.Flags().GetInt("lines")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	rawFlag, _ := cmd.Flags().GetBool("raw")

	// Validate pane exists
	if err := pane.ValidatePane(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(1)
	}

	// Capture pane content
	content, err := capturePaneContent(paneID, lines)
	if err != nil {
		return fmt.Errorf("capture-pane: %w", err)
	}

	// Raw mode: just output the captured text
	if rawFlag {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Resolve fab context
	ctx, err := pane.ResolvePaneContext(paneID, "")
	if err != nil {
		return fmt.Errorf("resolve context: %w", err)
	}

	if jsonFlag {
		out := captureJSON{
			Pane:              paneID,
			Lines:             lines,
			Content:           content,
			Worktree:          ctx.WorktreeDisplay,
			Change:            ctx.Change,
			Stage:             ctx.Stage,
			AgentState:        ctx.AgentState,
			AgentIdleDuration: ctx.AgentIdleDuration,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Default: human-readable output with header
	printCaptureHeader(cmd, paneID, ctx)
	fmt.Fprint(cmd.OutOrStdout(), content)
	return nil
}

// capturePaneContent runs tmux capture-pane and returns the captured text.
func capturePaneContent(paneID string, lines int) (string, error) {
	out, err := exec.Command("tmux", "capture-pane", "-t", paneID, "-p", "-l", fmt.Sprintf("%d", lines)).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// printCaptureHeader prints the human-readable header block.
func printCaptureHeader(cmd *cobra.Command, paneID string, ctx *pane.PaneContext) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "--- pane %s ---\n", paneID)

	var parts []string
	if ctx.WorktreeDisplay != "" {
		parts = append(parts, fmt.Sprintf("worktree: %s", ctx.WorktreeDisplay))
	}
	if ctx.Change != nil {
		parts = append(parts, fmt.Sprintf("change: %s", *ctx.Change))
	}
	if ctx.Stage != nil {
		parts = append(parts, fmt.Sprintf("stage: %s", *ctx.Stage))
	}
	if ctx.AgentState != nil {
		state := *ctx.AgentState
		if ctx.AgentIdleDuration != nil {
			state += " (" + *ctx.AgentIdleDuration + ")"
		}
		parts = append(parts, fmt.Sprintf("agent: %s", state))
	}

	if len(parts) > 0 {
		fmt.Fprintln(w, strings.Join(parts, " | "))
	}
	fmt.Fprintln(w, "---")
}
