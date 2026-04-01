package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func paneSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <pane> <text>",
		Short: "Send text to a tmux pane with safety validation",
		Args:  cobra.ExactArgs(2),
		RunE:  runPaneSend,
	}
	cmd.Flags().Bool("force", false, "Skip agent idle check")
	return cmd
}

func runPaneSend(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	text := args[1]
	forceFlag, _ := cmd.Flags().GetBool("force")

	// Validate pane exists (always, even with --force)
	if err := validatePaneExists(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "pane %s not found\n", paneID)
		os.Exit(1)
	}

	// Agent idle check (unless --force)
	if !forceFlag {
		if !isPaneAgentIdle(paneID) {
			fmt.Fprintf(cmd.ErrOrStderr(), "agent in %s is active, use --force to override\n", paneID)
			os.Exit(1)
		}
	}

	// Send keys via tmux
	return exec.Command("tmux", "send-keys", "-t", paneID, text, "Enter").Run()
}

// isPaneAgentIdle checks whether the agent in a pane is idle.
// Returns true if the pane is not in a fab worktree (non-fab panes treated as idle),
// if there is no runtime file, or if the agent has idle_since set.
func isPaneAgentIdle(paneID string) bool {
	paneCWD := getPaneCWD(paneID)
	if paneCWD == "" {
		return true // can't determine CWD, treat as idle
	}

	wtRoot, err := gitWorktreeRoot(paneCWD)
	if err != nil {
		return true // not in a git repo, treat as idle
	}

	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return true // not a fab worktree, treat as idle
	}

	_, folderName := readFabCurrent(wtRoot)
	if folderName == "" {
		return true // no active change, treat as idle
	}

	runtimeCache := make(map[string]interface{})
	agentState := resolveAgentState(wtRoot, folderName, runtimeCache)

	// "active" means not idle; everything else (idle, ?, em-dash) is treated as idle
	return agentState != "active"
}
