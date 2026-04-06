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
)

func operatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Launch operator in a dedicated tmux tab (singleton)",
		RunE:  runOperator,
	}
	cmd.AddCommand(operatorTickStartCmd(), operatorTimeCmd())
	return cmd
}

func runOperator(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	errW := cmd.ErrOrStderr()

	// Must be inside tmux
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(errW, "Error: not inside a tmux session.")
		os.Exit(1)
	}

	tabName := "operator"

	// Singleton: switch to existing tab if it exists
	if err := exec.Command("tmux", "select-window", "-t", tabName).Run(); err == nil {
		fmt.Fprintf(w, "Switched to existing %s tab.\n", tabName)
		return nil
	}

	// Resolve repo root
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return fmt.Errorf("cannot determine repo root: %w", err)
	}

	// Read spawn command from config
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}
	configPath := filepath.Join(fabRoot, "project", "config.yaml")
	spawnCmd := spawn.Command(configPath)

	// Create new tab running the operator skill
	shellCmd := fmt.Sprintf("%s '/fab-operator'", spawnCmd)
	if err := exec.Command("tmux", "new-window", "-c", repoRoot, "-n", tabName, shellCmd).Run(); err != nil {
		return fmt.Errorf("tmux new-window failed: %w", err)
	}

	fmt.Fprintf(w, "Launched %s.\n", tabName)
	return nil
}

// gitRepoRoot returns the git repo root for the current directory.
func gitRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
