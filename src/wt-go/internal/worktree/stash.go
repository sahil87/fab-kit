package worktree

import (
	"os/exec"
	"strings"
)

// StashCreate creates a stash commit using the hash-based approach (git stash create)
// for concurrency safety. It adds all files, creates the stash, stores it in the reflog,
// then resets and cleans. Returns the stash hash (empty string if no changes).
func StashCreate(msg string) (string, error) {
	// Stage all files
	if err := exec.Command("git", "add", "-A").Run(); err != nil {
		// Not fatal — might just have nothing to add
	}

	// Create a stash commit (does not modify ref)
	out, err := exec.Command("git", "stash", "create", msg).Output()
	if err != nil {
		return "", nil // No changes to stash
	}

	hash := strings.TrimSpace(string(out))
	if hash == "" {
		return "", nil // No changes
	}

	// Store the stash in the reflog for recovery
	exec.Command("git", "stash", "store", hash, "-m", msg).Run()

	// Reset and clean
	exec.Command("git", "reset", "--hard", "HEAD").Run()
	exec.Command("git", "clean", "-fd").Run()

	return hash, nil
}

// StashApply applies a stash by hash. No-op if hash is empty.
func StashApply(hash string) error {
	if hash == "" {
		return nil
	}
	cmd := exec.Command("git", "stash", "apply", hash)
	if out, err := cmd.CombinedOutput(); err != nil {
		return &StashApplyError{Hash: hash, Output: strings.TrimSpace(string(out))}
	}
	return nil
}

// StashApplyError represents a failed stash apply.
type StashApplyError struct {
	Hash   string
	Output string
}

func (e *StashApplyError) Error() string {
	return "git stash apply " + e.Hash + ": " + e.Output
}
