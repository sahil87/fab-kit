package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Run executes a shell command string in the given working directory.
// The command is run via "sh -c" so it supports pipes, redirects, etc.
// If command is empty, returns nil (no-op).
// The working directory is the repo root (parent of fabRoot).
func Run(fabRoot, command string) error {
	if command == "" {
		return nil
	}

	repoRoot := filepath.Dir(fabRoot)

	// Support script paths: if the command is a path to a file, resolve it
	// relative to the repo root.
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stage hook failed: %s: %w", command, err)
	}
	return nil
}
