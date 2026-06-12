package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/backlog"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
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
		return fmt.Errorf("not inside a tmux session")
	}

	// Collect IDs
	var ids []string
	if allFlag {
		items, err := backlog.ParsePending(backlogPath)
		if err != nil {
			return fmt.Errorf("reading backlog: %w", err)
		}
		if len(items) == 0 {
			return fmt.Errorf("No pending backlog items found.")
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

	// Process each ID. Launch failures (wt create, tmux new-window) are
	// reported per item with a failure count and a non-zero exit when any
	// item failed — never a silent exit 0 leaving an orphaned worktree.
	// Pattern precedent: batch_archive.go archiveLoop. Backlog lookup
	// problems remain warn-and-skip (user-input issues, not launch failures)
	// and don't count toward the launch-attempt total.
	attempted := 0
	failed := 0
	for _, id := range ids {
		content, err := backlog.ExtractContent(backlogPath, id)
		if err != nil {
			// Warn-and-skip per ID; the error is accurate now — "not found
			// in backlog" for a missing ID, the real read error otherwise.
			fmt.Fprintf(errW, "Warning: [%s] %v, skipping\n", id, err)
			continue
		}
		if content == "" {
			fmt.Fprintf(errW, "Warning: [%s] has empty content, skipping\n", id)
			continue
		}

		attempted++

		// Truncate display
		display := content
		if len(display) > 70 {
			display = display[:70] + "..."
		}
		fmt.Fprintf(w, "  [%s] %s\n", id, display)

		// Create worktree
		wtOut, wtStderr, err := pane.RunCmd("wt", "create", "--non-interactive", "--worktree-name", id)
		if err != nil {
			fmt.Fprintf(errW, "  [%s] FAILED: wt create: %v\n", id, pane.StderrError(err, wtStderr))
			failed++
			continue
		}
		wtPath := strings.TrimSpace(wtOut)

		// Escape single quotes for shell
		safe := strings.ReplaceAll(content, "'", "'\\''")

		// Open tmux window. The worktree already exists at this point, so a
		// launch failure names it as the recovery/cleanup hint.
		shellCmd := fmt.Sprintf("%s '/fab-new %s'", spawnCmd, safe)
		if _, stderr, err := pane.RunCmd("tmux", "new-window", "-n", "fab-"+id, "-c", wtPath, shellCmd); err != nil {
			fmt.Fprintf(errW, "  [%s] FAILED: tmux new-window: %v (worktree already created at %s)\n",
				id, pane.StderrError(err, stderr), wtPath)
			failed++
			continue
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d item(s) failed to launch", failed, attempted)
	}
	return nil
}

// listPendingItems prints pending backlog items.
func listPendingItems(w interface{ Write([]byte) (int, error) }, backlogPath string) error {
	items, err := backlog.ParsePending(backlogPath)
	if err != nil {
		return fmt.Errorf("reading backlog: %w", err)
	}
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
