package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/backlog"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

func batchNewCmd() *cobra.Command {
	var listFlag, allFlag bool

	cmd := &cobra.Command{
		Use:   "new [backlog-id...]",
		Short: "Create worktree tabs from backlog items",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchNew(cmd, args, listFlag, allFlag)
		},
	}

	cmd.Flags().BoolVar(&listFlag, "list", false, "Show pending backlog items and their IDs")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Open tabs for all pending backlog items")

	return cmd
}

func runBatchNew(cmd *cobra.Command, args []string, listFlag, allFlag bool) error {
	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()

	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	backlogPath := backlog.Path(fabRoot)

	if _, err := os.Stat(backlogPath); os.IsNotExist(err) {
		return fmt.Errorf("backlog.md not found at %s", backlogPath)
	}

	// No args defaults to --list
	if len(args) == 0 && !allFlag {
		listFlag = true
	}

	if listFlag {
		return listPendingItems(w, backlogPath)
	}

	// Check tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(errW, "Error: not inside a tmux session")
		os.Exit(1)
	}

	// Collect IDs
	var ids []string
	if allFlag {
		items := backlog.ParsePending(backlogPath)
		if len(items) == 0 {
			fmt.Fprintln(errW, "No pending backlog items found.")
			os.Exit(1)
		}
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		fmt.Fprintf(w, "Opening %d tabs for all pending items...\n", len(ids))
	} else {
		ids = args
	}

	// Read spawn command
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	spawnCmd := spawn.Command(configPath)

	// Process each ID
	for _, id := range ids {
		content, err := backlog.ExtractContent(backlogPath, id)
		if err != nil {
			fmt.Fprintf(errW, "Warning: [%s] not found in backlog, skipping\n", id)
			continue
		}
		if content == "" {
			fmt.Fprintf(errW, "Warning: [%s] has empty content, skipping\n", id)
			continue
		}

		// Truncate display
		display := content
		if len(display) > 70 {
			display = display[:70] + "..."
		}
		fmt.Fprintf(w, "  [%s] %s\n", id, display)

		// Create worktree
		wtOut, err := exec.Command("wt", "create", "--non-interactive", "--worktree-name", id).Output()
		if err != nil {
			fmt.Fprintf(errW, "Error: failed to create worktree for [%s], skipping\n", id)
			continue
		}
		wtPath := strings.TrimSpace(string(wtOut))

		// Escape single quotes for shell
		safe := strings.ReplaceAll(content, "'", "'\\''")

		// Open tmux window
		shellCmd := fmt.Sprintf("%s '/fab-new %s'", spawnCmd, safe)
		exec.Command("tmux", "new-window", "-n", "fab-"+id, "-c", wtPath, shellCmd).Run()
	}

	return nil
}

// listPendingItems prints pending backlog items.
func listPendingItems(w interface{ Write([]byte) (int, error) }, backlogPath string) error {
	items := backlog.ParsePending(backlogPath)
	fmt.Fprintln(w, "Pending backlog items:")
	fmt.Fprintln(w)
	for _, item := range items {
		display := item.Desc
		if len(display) > 80 {
			display = display[:80]
		}
		fmt.Fprintf(w, "  %-6s %s\n", "["+item.ID+"]", display)
	}
	return nil
}
