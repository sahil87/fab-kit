package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
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

// stateDir returns the XDG state base dir, spec-compliant and uniform on Linux
// and macOS. It honors XDG_STATE_HOME only when set AND absolute; otherwise it
// falls back to $HOME/.local/state. Deliberately NOT ~/Library/... on macOS —
// terminal users expect ~/.local/state, and the Go stdlib has no UserStateDir().
func stateDir() (string, error) {
	if s := os.Getenv("XDG_STATE_HOME"); s != "" && filepath.IsAbs(s) {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}

// slugify converts a tmux socket path into a filesystem-safe, deterministic,
// collision-free slug. The rule: strip the leading separator, then replace any
// remaining path separators with "-". Distinct socket paths produce distinct
// slugs because the separator-to-dash mapping preserves the path's structure.
func slugify(s string) string {
	s = strings.TrimPrefix(s, string(os.PathSeparator))
	s = strings.TrimPrefix(s, "/")
	s = strings.ReplaceAll(s, string(os.PathSeparator), "-")
	s = strings.ReplaceAll(s, "/", "-")
	if s == "" {
		return "default"
	}
	return s
}

// serverSlug derives a filesystem-safe slug from the tmux socket path. It falls
// back to "default" when tmux cannot be queried (the operator must still
// function if the #{socket_path} query fails).
func serverSlug(server string) string {
	out, err := exec.Command("tmux", pane.WithServer(server, "display-message", "-p", "#{socket_path}")...).Output()
	if err != nil {
		return "default"
	}
	return slugify(strings.TrimSpace(string(out)))
}

// StatePath returns the server-keyed operator state file path:
// <stateDir>/fab/operator/<server-slug>.yaml. The parent directory is created
// with MkdirAll (0o755). The path is keyed by the tmux socket (via serverSlug)
// so cross-repo coordination state has a stable, server-scoped home rather than
// living at a repo-rooted .fab-operator.yaml.
func StatePath(server string) (string, error) {
	base, err := stateDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "fab", "operator")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, serverSlug(server)+".yaml"), nil
}
