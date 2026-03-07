package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/fab-go/internal/resolve"
)

func sendKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "send-keys <change> <text>",
		Short: "Send text to a change's tmux pane",
		Args:  cobra.ExactArgs(2),
		RunE:  runSendKeys,
	}
}

func runSendKeys(cmd *cobra.Command, args []string) error {
	changeArg := args[0]
	text := args[1]

	// Validate preconditions
	if err := validateSendKeysInputs(changeArg, os.Getenv("TMUX")); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}

	// Resolve the change argument to a folder name
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		return err
	}

	// Discover all tmux panes and find the one matching this change
	paneID, err := resolveChangePane(folder, cmd)
	if err != nil {
		return err
	}

	// Send keys to the resolved pane
	sendCmd := exec.Command("tmux", buildSendKeysArgs(paneID, text)...)
	sendCmd.Stdout = cmd.OutOrStdout()
	sendCmd.Stderr = cmd.ErrOrStderr()
	if err := sendCmd.Run(); err != nil {
		return fmt.Errorf("Error: failed to send keys to pane %s: %w", paneID, err)
	}

	return nil
}

// resolveChangePane finds the tmux pane associated with a given change folder.
// It reuses the same pane discovery logic as pane-map: tmux list-panes → git root → .fab-status.yaml.
func resolveChangePane(folder string, cmd *cobra.Command) (string, error) {
	panes, err := discoverPanes()
	if err != nil {
		return "", err
	}

	matches, warning := matchPanesByFolder(panes, folder, resolvePaneChange)

	if len(matches) == 0 {
		return "", fmt.Errorf("No tmux pane found for change %q.", folder)
	}

	if warning != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), warning)
	}

	return matches[0], nil
}

// resolvePaneChange resolves a pane entry to its active change folder name.
// Returns empty string if the pane is not in a git repo, has no fab/ directory,
// or has no active change.
func resolvePaneChange(p paneEntry) string {
	wtRoot, err := gitWorktreeRoot(p.cwd)
	if err != nil {
		return ""
	}

	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ""
	}

	_, folderName := readFabCurrent(wtRoot)
	return folderName
}

// matchPanesByFolder is a testable helper that matches pane entries to a change folder.
// Returns (matchedPaneIDs, warningMessage).
func matchPanesByFolder(panes []paneEntry, folder string, resolveFunc func(paneEntry) string) ([]string, string) {
	var matches []string
	for _, p := range panes {
		if resolveFunc(p) == folder {
			matches = append(matches, p.id)
		}
	}

	warning := ""
	if len(matches) > 1 {
		warning = fmt.Sprintf("Warning: multiple panes found for %s, using %s",
			resolve.ExtractID(folder), matches[0])
	}

	return matches, warning
}

// buildSendKeysArgs constructs the tmux send-keys argument list.
func buildSendKeysArgs(paneID, text string) []string {
	return []string{"send-keys", "-t", paneID, text, "Enter"}
}

// validateSendKeysInputs checks preconditions for send-keys.
// It returns an error if the preconditions are not met.
func validateSendKeysInputs(changeArg string, tmuxEnv string) error {
	if tmuxEnv == "" {
		return fmt.Errorf("Error: not inside a tmux session")
	}
	if strings.TrimSpace(changeArg) == "" {
		return fmt.Errorf("change argument must not be empty")
	}
	return nil
}
