package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
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
	server, _ := cmd.Flags().GetString("server")

	// Validate line count
	if lines < 1 {
		return fmt.Errorf("--lines must be >= 1")
	}

	// Validate pane exists. Exit codes follow the pane-family scheme shared
	// with window-name: 2 = pane missing, 3 = other tmux failure — so an
	// operator script can branch on cause uniformly across the family. The
	// in-handler os.Exit stays because non-1 codes are genuinely needed here.
	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	// Capture pane content
	content, err := capturePaneContent(server, paneID, lines)
	if err != nil {
		return fmt.Errorf("capture-pane: %w", err)
	}

	// Raw mode: just output the captured text
	if rawFlag {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Resolve fab context
	ctx, err := pane.ResolvePaneContext(paneID, "", server)
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

// capturePaneArgs returns the tmux capture-pane argument list for the given pane and line count.
// It uses -S -N to capture the last N lines from the pane's scrollback buffer.
// When server is non-empty, the argv is prepended with `-L <server>`.
func capturePaneArgs(server, paneID string, lines int) []string {
	return pane.WithServer(server, "capture-pane", "-t", paneID, "-p", "-S", fmt.Sprintf("-%d", lines))
}

// capturePaneContent runs tmux capture-pane and returns the captured text
// (raw — never trimmed, so --raw output stays byte-identical to tmux's). On
// failure the error names the pane and carries tmux's stderr diagnostic.
// When server is non-empty, the tmux invocation is scoped via `-L <server>`.
func capturePaneContent(server, paneID string, lines int) (string, error) {
	out, stderr, err := pane.RunCmd("tmux", capturePaneArgs(server, paneID, lines)...)
	if err != nil {
		return "", pane.StderrError(fmt.Errorf("pane %s: %w", paneID, err), stderr)
	}
	return out, nil
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
