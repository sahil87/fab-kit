package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RepoContext holds the main repository context for worktree operations.
type RepoContext struct {
	RepoRoot     string // Absolute path to main repo root
	RepoName     string // Basename of the repo root
	WorktreesDir string // Path to <repo>.worktrees/ directory
}

// GetRepoContext derives the repo context from the current git repo.
// It always resolves to the main repo root, even when run from a worktree.
func GetRepoContext() (*RepoContext, error) {
	out, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}

	gitCommonDir := strings.TrimSpace(string(out))

	// Convert to absolute, symlink-resolved path
	absPath, err := filepath.Abs(gitCommonDir)
	if err != nil {
		return nil, fmt.Errorf("resolving git common dir: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return nil, fmt.Errorf("resolving symlinks: %w", err)
	}

	// Derive main repo root by stripping /.git suffix
	repoRoot := strings.TrimSuffix(resolved, string(filepath.Separator)+".git")
	repoName := filepath.Base(repoRoot)
	worktreesDir := filepath.Join(filepath.Dir(repoRoot), repoName+".worktrees")

	return &RepoContext{
		RepoRoot:     repoRoot,
		RepoName:     repoName,
		WorktreesDir: worktreesDir,
	}, nil
}

// IsWorktree returns true if the current directory is inside a worktree (not the main repo).
func IsWorktree() bool {
	gitDir, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return false
	}
	commonDir, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(gitDir)) != strings.TrimSpace(string(commonDir))
}

// DefaultBranch detects the default branch of the repo.
// Checks refs/remotes/origin/HEAD first, then refs/heads/main, refs/heads/master, finally HEAD.
func DefaultBranch() string {
	// Try origin/HEAD first
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		if branch != "" {
			return branch
		}
	}

	// Fallback: check main/master existence
	if err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/main").Run(); err == nil {
		return "main"
	}
	if err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/master").Run(); err == nil {
		return "master"
	}

	// Fallback: current HEAD
	out, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		branch := strings.TrimSpace(string(out))
		if branch != "" {
			return branch
		}
	}

	return "main"
}

// invalidBranchPattern matches git ref characters that are invalid.
var invalidBranchPattern = regexp.MustCompile(`[\s~^:?*\[]`)

// ValidateBranchName validates a branch name against git ref naming rules.
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if invalidBranchPattern.MatchString(name) {
		return fmt.Errorf("branch name contains invalid characters")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..'")
	}
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock'")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("branch name cannot start with '.'")
	}
	if strings.Contains(name, "/.") {
		return fmt.Errorf("branch name cannot contain '/.'")
	}
	return nil
}

// DeriveWorktreeName extracts a worktree name from a branch name.
// Takes the last segment after slashes and replaces non-alphanumeric characters
// (except - and _) with -.
func DeriveWorktreeName(branch string) string {
	// Get last segment after slashes
	parts := strings.Split(branch, "/")
	name := parts[len(parts)-1]

	// Replace non-alphanumeric chars except - and _
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result.WriteRune(c)
		} else {
			result.WriteRune('-')
		}
	}
	return result.String()
}

// ValidateGitRepo checks if the current directory is inside a git repository.
func ValidateGitRepo() error {
	err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run()
	if err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

// CurrentWorktreeTopLevel returns the top-level path of the current worktree.
func CurrentWorktreeTopLevel() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not in a git worktree")
	}
	path := strings.TrimSpace(string(out))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil
	}
	return resolved, nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("cannot determine current branch")
	}
	return strings.TrimSpace(string(out)), nil
}

// InitScriptPath returns the path to the init script, respecting WORKTREE_INIT_SCRIPT env var.
func InitScriptPath() string {
	if v := os.Getenv("WORKTREE_INIT_SCRIPT"); v != "" {
		return v
	}
	return "fab/.kit/worktree-init.sh"
}
