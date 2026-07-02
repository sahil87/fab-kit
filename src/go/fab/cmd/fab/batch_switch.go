package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func batchSwitchCmd() *cobra.Command {
	var listFlag, allFlag bool

	cmd := &cobra.Command{
		Use:   "switch [change...]",
		Short: "Open tmux tabs in worktrees for one or more changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchSwitch(cmd, args, listFlag, allFlag)
		},
	}

	cmd.Flags().BoolVar(&listFlag, "list", false, "Show available changes")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Open tabs for all changes")

	return cmd
}

func runBatchSwitch(cmd *cobra.Command, args []string, listFlag, allFlag bool) error {
	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()

	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	changesDir := filepath.Join(fabRoot, "changes")
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return fmt.Errorf("changes directory not found at %s", changesDir)
	}

	// No args defaults to --list
	if len(args) == 0 && !allFlag {
		listFlag = true
	}

	if listFlag {
		return listChanges(w, changesDir)
	}

	// Check tmux
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not inside a tmux session")
	}

	// Collect change names
	var changes []string
	if allFlag {
		changes = allChangeNames(changesDir)
		if len(changes) == 0 {
			return fmt.Errorf("No changes found.")
		}
		fmt.Fprintf(w, "Opening %d tabs for all changes...\n", len(changes))
	} else {
		changes = args
	}

	// Compose the worker spawn command from the default tier's provider
	// session_command with the default tier's {model}/{effort} profile SUBSTITUTED
	// (workers finally spawn WITH a profile). Substitution resolves all
	// placeholders so no literal braces reach the tmux new-window shell command.
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	spawnCmd := defaultTierSpawnCommand(configPath)
	cfg, _ := config.Load(fabRoot)
	branchPrefix := cfg.GetBranchPrefix()

	// Process each change
	for _, change := range changes {
		// Resolve in-process — the canonical resolver, same as batch archive.
		// No `fab change resolve` subprocess (PATH dependency, shim round-trip,
		// stderr-detail loss); the warning surfaces the specific error.
		match, err := resolve.ToFolder(fabRoot, change)
		if err != nil {
			fmt.Fprintf(errW, "Warning: could not resolve '%s' (%v), skipping\n", change, err)
			continue
		}

		fmt.Fprintf(w, "  %s\n", match)

		// Construct branch name
		branchName := branchPrefix + match

		// Create worktree
		wtOut, err := exec.Command("wt", "create", "--non-interactive", "--reuse", "--worktree-name", match, branchName).Output()
		if err != nil {
			fmt.Fprintf(errW, "Error: failed to create worktree for '%s', skipping\n", match)
			continue
		}
		wtPath := strings.TrimSpace(string(wtOut))

		// Escape single quotes for shell
		safe := strings.ReplaceAll(match, "'", "'\\''")

		// Open tmux window
		shellCmd := fmt.Sprintf("%s '/fab-switch %s'", spawnCmd, safe)
		exec.Command("tmux", "new-window", "-n", match, "-c", wtPath, shellCmd).Run()
	}

	return nil
}

// listChanges prints available changes (excluding archive).
func listChanges(w interface{ Write([]byte) (int, error) }, changesDir string) error {
	fmt.Fprintln(w, "Available changes:")
	fmt.Fprintln(w)
	names := allChangeNames(changesDir)
	for _, name := range names {
		fmt.Fprintf(w, "  %s\n", name)
	}
	return nil
}

// allChangeNames returns all non-archive change folder names.
func allChangeNames(changesDir string) []string {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "archive" {
			continue
		}
		names = append(names, name)
	}
	return names
}
