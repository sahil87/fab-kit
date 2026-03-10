// Package worktree discovers git worktrees.
package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Info holds the core state for a single worktree.
type Info struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Branch    string `json:"branch"`
	IsMain    bool   `json:"is_main"`
	IsCurrent bool   `json:"is_current"`
}

// List discovers all git worktrees via `git worktree list --porcelain`.
func List() ([]Info, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	currentDir, _ = filepath.EvalSymlinks(currentDir)

	raw, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	entries := parseWorktreeList(string(raw))

	// First entry in `git worktree list` is always the main worktree
	mainPath := ""
	if len(entries) > 0 {
		mainPath = entries[0].path
	}

	var infos []Info
	for _, e := range entries {
		info := Info{
			Path:      e.path,
			Branch:    e.branch,
			IsCurrent: currentDir == e.path || strings.HasPrefix(currentDir, e.path+string(filepath.Separator)),
			IsMain:    e.path == mainPath,
		}

		if info.IsMain {
			info.Name = "(main)"
		} else {
			info.Name = filepath.Base(e.path)
		}

		infos = append(infos, info)
	}

	return infos, nil
}

// FindByName returns the worktree matching the given name.
func FindByName(name string) (*Info, error) {
	all, err := List()
	if err != nil {
		return nil, err
	}
	nameLower := strings.ToLower(name)
	for i := range all {
		if strings.ToLower(all[i].Name) == nameLower {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("Worktree '%s' not found", name)
}

// Current returns the worktree for the current working directory.
func Current() (*Info, error) {
	all, err := List()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].IsCurrent {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("current directory is not a git worktree")
}

// FormatHuman formats a single worktree as a human-readable line.
func FormatHuman(info *Info) string {
	return fmt.Sprintf("%s  %s", info.Name, info.Branch)
}

// FormatAllHuman formats all worktrees as a human-readable table.
func FormatAllHuman(infos []Info) string {
	repoName := ""
	wtDir := ""
	if len(infos) > 0 {
		for _, info := range infos {
			if info.IsMain {
				repoName = filepath.Base(info.Path)
				wtDir = filepath.Dir(info.Path)
				break
			}
		}
		if repoName == "" {
			repoName = filepath.Base(infos[0].Path)
			wtDir = filepath.Dir(infos[0].Path)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Worktrees for: %s\n", repoName)
	fmt.Fprintf(&sb, "Location: %s\n\n", wtDir)

	for _, info := range infos {
		marker := "  "
		if info.IsCurrent {
			marker = "* "
		}
		fmt.Fprintf(&sb, "%s%-14s %s\n", marker, info.Name, info.Branch)
	}

	fmt.Fprintf(&sb, "\nTotal: %d worktree(s)", len(infos))
	return sb.String()
}

// FormatJSON formats a single worktree as JSON.
func FormatJSON(info *Info) (string, error) {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatAllJSON formats all worktrees as a JSON array.
func FormatAllJSON(infos []Info) (string, error) {
	data, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type worktreeEntry struct {
	path   string
	branch string
}

func parseWorktreeList(raw string) []worktreeEntry {
	var entries []worktreeEntry
	var current worktreeEntry

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if current.path != "" {
				entries = append(entries, current)
			}
			current = worktreeEntry{path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}
	if current.path != "" {
		entries = append(entries, current)
	}
	return entries
}
