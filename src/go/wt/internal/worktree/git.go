package worktree

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// HasUncommittedChanges returns true if the current directory has staged or unstaged changes.
func HasUncommittedChanges() bool {
	if err := exec.Command("git", "diff", "--quiet").Run(); err != nil {
		return true
	}
	if err := exec.Command("git", "diff", "--cached", "--quiet").Run(); err != nil {
		return true
	}
	return false
}

// HasUntrackedFiles returns true if there are untracked files in the current directory.
func HasUntrackedFiles() bool {
	out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// HasUnpushedCommits returns true if the branch has unpushed commits.
func HasUnpushedCommits(branch string) bool {
	upstream, err := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}").Output()
	if err != nil {
		return false
	}
	up := strings.TrimSpace(string(upstream))
	out, err := exec.Command("git", "log", up+".."+branch, "--oneline").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// GetUnpushedCount returns the number of unpushed commits for the branch.
func GetUnpushedCount(branch string) int {
	upstream, err := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}").Output()
	if err != nil {
		return 0
	}
	up := strings.TrimSpace(string(upstream))
	out, err := exec.Command("git", "rev-list", "--count", up+".."+branch).Output()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return n
}

// BranchExistsLocally returns true if the branch exists as a local ref.
func BranchExistsLocally(branch string) bool {
	return exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

// BranchExistsRemotely returns true if the branch exists on origin.
func BranchExistsRemotely(branch string) bool {
	out, err := exec.Command("git", "ls-remote", "--heads", "origin", branch).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), branch)
}

// FetchRemoteBranch fetches a branch from origin into a local branch.
func FetchRemoteBranch(branch string) error {
	cmd := exec.Command("git", "fetch", "origin", branch+":"+branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch origin %s failed: %s", branch, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeleteLocalBranch deletes a local branch. If force is true, uses -D instead of -d.
func DeleteLocalBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	cmd := exec.Command("git", "branch", flag, branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch %s %s: %s", flag, branch, strings.TrimSpace(string(out)))
	}
	return nil
}

// DeleteRemoteBranch deletes a branch on origin.
func DeleteRemoteBranch(branch string) error {
	cmd := exec.Command("git", "push", "origin", "--delete", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push origin --delete %s: %s", branch, strings.TrimSpace(string(out)))
	}
	return nil
}

// GetUnpushedCommitLines returns the oneline log of unpushed commits for display.
func GetUnpushedCommitLines(branch string, limit int) []string {
	upstream, err := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}").Output()
	if err != nil {
		return nil
	}
	up := strings.TrimSpace(string(upstream))
	out, err := exec.Command("git", "log", up+".."+branch, "--oneline").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	if limit > 0 && len(lines) > limit {
		return lines[:limit]
	}
	return lines
}
