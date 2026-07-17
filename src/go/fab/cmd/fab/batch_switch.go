package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
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

		// Create worktree. Route per wt's 2af2 contract: the positional is
		// new-branch-only (exits 2 on an existing local/remote branch), and
		// --checkout <branch> is the explicit opt-in for an existing branch.
		// --reuse's name-collision short-circuit ignores branch selectors, so
		// it is retained on both forms.
		wtArgs := []string{"create", "--non-interactive", "--reuse", "--worktree-name", match}
		if branchExists(branchName) {
			wtArgs = append(wtArgs, "--checkout", branchName)
		} else {
			wtArgs = append(wtArgs, branchName)
		}
		wtOut, wtStderr, err := pane.RunCmd("wt", wtArgs...)
		if err != nil {
			fmt.Fprintf(errW, "Error: failed to create worktree for '%s' (%v), skipping\n", match, pane.StderrError(err, wtStderr))
			continue
		}
		wtPath := strings.TrimSpace(wtOut)

		// Escape single quotes for shell
		safe := strings.ReplaceAll(match, "'", "'\\''")

		// Open tmux window
		shellCmd := fmt.Sprintf("%s '/fab-switch %s'", spawnCmd, safe)
		exec.Command("tmux", "new-window", "-n", match, "-c", wtPath, shellCmd).Run()
	}

	return nil
}

// branchExists reports whether branch exists locally or on origin, mirroring
// wt's own BranchExistsLocally / BranchExistsRemotely checks
// (internal/worktree/git.go in the wt repo) so fab's routing never disagrees
// with wt's positional validation (positional = new-branch-only under the 2af2
// contract). Local is checked first (no network); the origin ls-remote runs
// only when the branch is not local. A failed/offline ls-remote degrades to
// not-remote → positional → wt itself re-checks and errors visibly.
func branchExists(branch string) bool {
	if exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil {
		return true
	}
	out, err := exec.Command("git", "ls-remote", "--heads", "origin", branch).Output()
	return err == nil && strings.TrimSpace(string(out)) != ""
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
