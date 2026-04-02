package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"gopkg.in/yaml.v3"
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
		fmt.Fprintln(errW, "Error: not inside a tmux session")
		os.Exit(1)
	}

	// Collect change names
	var changes []string
	if allFlag {
		changes = allChangeNames(changesDir)
		if len(changes) == 0 {
			fmt.Fprintln(errW, "No changes found.")
			os.Exit(1)
		}
		fmt.Fprintf(w, "Opening %d tabs for all changes...\n", len(changes))
	} else {
		changes = args
	}

	// Read spawn command and branch prefix
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	spawnCmd := spawn.Command(configPath)
	branchPrefix := getBranchPrefix(configPath)

	// Process each change
	for _, change := range changes {
		// Resolve via fab change resolve
		out, err := exec.Command("fab", "change", "resolve", change).Output()
		if err != nil {
			fmt.Fprintf(errW, "Warning: could not resolve '%s', skipping\n", change)
			continue
		}
		match := strings.TrimSpace(string(out))

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

// getBranchPrefix reads branch_prefix from config.yaml.
func getBranchPrefix(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var cfg struct {
		BranchPrefix string `yaml:"branch_prefix"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.BranchPrefix
}
