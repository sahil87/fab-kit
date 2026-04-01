package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/status"
	sf "github.com/wvrdz/fab-kit/src/go/fab/internal/statusfile"
)

func paneCaptureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capture <pane>",
		Short: "Capture visible content of a tmux pane",
		Args:  cobra.ExactArgs(1),
		RunE:  runPaneCapture,
	}
	cmd.Flags().IntP("lines", "l", 0, "Number of lines to capture (default: all visible)")
	cmd.Flags().Bool("json", false, "Output as JSON object with fab context")
	return cmd
}

// paneCaptureJSON represents JSON output for pane capture.
type paneCaptureJSON struct {
	Pane       string  `json:"pane"`
	Lines      int     `json:"lines"`
	Content    string  `json:"content"`
	Change     *string `json:"change"`
	Stage      *string `json:"stage"`
	AgentState *string `json:"agent_state"`
}

func runPaneCapture(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	linesFlag, _ := cmd.Flags().GetInt("lines")
	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Validate pane exists
	if err := validatePaneExists(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "pane %s not found\n", paneID)
		os.Exit(1)
	}

	// Build tmux capture-pane command
	tmuxArgs := []string{"capture-pane", "-t", paneID, "-p"}
	if linesFlag > 0 {
		tmuxArgs = append(tmuxArgs, "-l", fmt.Sprintf("%d", linesFlag))
	}

	out, err := exec.Command("tmux", tmuxArgs...).Output()
	if err != nil {
		return fmt.Errorf("tmux capture-pane: %w", err)
	}

	content := string(out)

	if !jsonFlag {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// JSON mode: enrich with fab context
	lineCount := countLines(content)
	if linesFlag > 0 {
		lineCount = linesFlag
	}

	var change, stage, agentState *string

	// Resolve fab context from pane CWD
	paneCWD := getPaneCWD(paneID)
	if paneCWD != "" {
		wtRoot, err := gitWorktreeRoot(paneCWD)
		if err == nil {
			fabDir := filepath.Join(wtRoot, "fab")
			if _, statErr := os.Stat(fabDir); statErr == nil {
				_, folderName := readFabCurrent(wtRoot)
				if folderName != "" {
					change = &folderName

					// Read stage
					s := resolvePaneStage(wtRoot, folderName)
					if s != "" {
						stage = &s
					}

					// Read agent state
					runtimeCache := make(map[string]interface{})
					a := resolveAgentState(wtRoot, folderName, runtimeCache)
					as := resolveAgentStateForJSON(a)
					if as != "" {
						agentState = &as
					}
				}
			}
		}
	}

	result := paneCaptureJSON{
		Pane:       paneID,
		Lines:      lineCount,
		Content:    content,
		Change:     change,
		Stage:      stage,
		AgentState: agentState,
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// validatePaneExists checks whether a tmux pane exists by querying tmux.
func validatePaneExists(paneID string) error {
	return exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_id}").Run()
}

// getPaneCWD returns the current working directory of a tmux pane.
func getPaneCWD(paneID string) string {
	out, err := exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_current_path}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// resolvePaneStage reads the current pipeline stage for a change in a worktree.
func resolvePaneStage(wtRoot, folderName string) string {
	statusPath := filepath.Join(wtRoot, "fab", "changes", folderName, ".status.yaml")
	statusFile, err := sf.Load(statusPath)
	if err != nil {
		return ""
	}
	stage, _ := status.DisplayStage(statusFile)
	return stage
}

// resolveAgentStateForJSON converts the display agent state to a JSON-friendly value.
// Returns empty string for em-dash values (which become null in JSON).
func resolveAgentStateForJSON(agent string) string {
	switch {
	case agent == "\u2014":
		return ""
	case agent == "?":
		return "unknown"
	case strings.HasPrefix(agent, "idle"):
		return "idle"
	default:
		return agent
	}
}

// countLines counts the number of lines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return len(lines)
}
