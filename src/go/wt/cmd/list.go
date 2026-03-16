package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	wt "github.com/wvrdz/fab-kit/src/go/wt/internal/worktree"
)

// ansiPattern matches ANSI escape sequences for display width calculation.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// listEntry holds enriched worktree info for the list command.
type listEntry struct {
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	Path      string `json:"path"`
	IsMain    bool   `json:"is_main"`
	IsCurrent bool   `json:"is_current"`
	Dirty     bool   `json:"dirty"`
	Unpushed  int    `json:"unpushed"`
}

func listCmd() *cobra.Command {
	var (
		pathName string
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all git worktrees",
		Long: `List all git worktrees for the current repository.

The current worktree is marked with a green asterisk (*).
Dirty worktrees show * and unpushed commits show ↑N.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check mutual exclusivity
			if pathName != "" && jsonOut {
				wt.ExitWithError(wt.ExitInvalidArgs,
					"--path and --json are mutually exclusive",
					"Use one output mode at a time",
					"Run 'wt list --help' for usage information")
			}

			if err := wt.ValidateGitRepo(); err != nil {
				wt.ExitWithError(wt.ExitGitError,
					"Not a git repository",
					"This command requires a git repository",
					"Navigate to a git repository and try again")
			}

			ctx, err := wt.GetRepoContext()
			if err != nil {
				wt.ExitWithError(wt.ExitGeneralError, "Cannot get repo context", err.Error(), "")
			}

			// Path lookup mode
			if pathName != "" {
				return handlePathLookup(pathName, ctx)
			}

			entries, err := getEnrichedEntries(ctx)
			if err != nil {
				wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
			}

			if jsonOut {
				return handleJSONOutput(entries)
			}

			return handleFormattedOutput(entries, ctx)
		},
	}

	cmd.Flags().StringVar(&pathName, "path", "", "Output just the absolute path for a named worktree")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output worktree data as a JSON array")

	return cmd
}

func handlePathLookup(name string, ctx *wt.RepoContext) error {
	entries, err := listWorktreeEntries()
	if err != nil {
		wt.ExitWithError(wt.ExitGitError, "Cannot list worktrees", err.Error(), "")
	}

	for _, e := range entries {
		entryName := filepath.Base(e.path)
		if e.path == ctx.RepoRoot {
			entryName = "main"
		}
		if strings.EqualFold(entryName, name) {
			fmt.Println(e.path)
			return nil
		}
	}

	fmt.Fprintf(os.Stderr, "Worktree '%s' not found. Use 'wt list' to see available worktrees.\n", name)
	os.Exit(wt.ExitGeneralError)
	return nil
}

func handleJSONOutput(entries []listEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON encoding: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// displayWidth returns the visible width of s, excluding ANSI escape sequences.
// Uses RuneCountInString to correctly count multi-byte characters (e.g. "↑").
func displayWidth(s string) int {
	return utf8.RuneCountInString(ansiPattern.ReplaceAllString(s, ""))
}

// relativePath computes a short relative path for display.
// Main worktree: "{repoName}/"
// Other worktrees: "{repoName}.worktrees/{wtName}/"
func relativePath(entryPath string, ctx *wt.RepoContext) string {
	parent := filepath.Dir(ctx.WorktreesDir)
	rel, err := filepath.Rel(parent, entryPath)
	if err != nil {
		return entryPath
	}
	return rel + "/"
}

func handleFormattedOutput(entries []listEntry, ctx *wt.RepoContext) error {
	fmt.Printf("Worktrees for: %s%s%s\n", wt.ColorBold, ctx.RepoName, wt.ColorReset)
	fmt.Printf("Location: %s\n\n", ctx.WorktreesDir)

	// Headers
	headers := [4]string{"Name", "Branch", "Status", "Path"}

	// Pre-compute display values for each entry
	type displayRow struct {
		name   string
		branch string
		status string
		path   string
	}
	rows := make([]displayRow, len(entries))
	for i, e := range entries {
		name := e.Name
		if e.IsMain {
			name = wt.ColorBold + "(main)" + wt.ColorReset
		}

		// Build status indicators
		var dirtyMarker, unpushedMarker string
		if e.Dirty {
			dirtyMarker = wt.ColorYellow + "*" + wt.ColorReset
		}
		if e.Unpushed > 0 {
			unpushedMarker = wt.ColorYellow + "↑" + strconv.Itoa(e.Unpushed) + wt.ColorReset
		}

		var status string
		if dirtyMarker != "" && unpushedMarker != "" {
			status = dirtyMarker + " " + unpushedMarker
		} else if dirtyMarker != "" {
			status = dirtyMarker
		} else if unpushedMarker != "" {
			status = unpushedMarker
		}

		rows[i] = displayRow{
			name:   name,
			branch: e.Branch,
			status: status,
			path:   relativePath(e.Path, ctx),
		}
	}

	// Compute dynamic column widths (minimum = header label length)
	colWidths := [4]int{len(headers[0]), len(headers[1]), len(headers[2]), len(headers[3])}
	for _, r := range rows {
		if w := displayWidth(r.name); w > colWidths[0] {
			colWidths[0] = w
		}
		if w := displayWidth(r.branch); w > colWidths[1] {
			colWidths[1] = w
		}
		if w := displayWidth(r.status); w > colWidths[2] {
			colWidths[2] = w
		}
		if w := displayWidth(r.path); w > colWidths[3] {
			colWidths[3] = w
		}
	}

	// Print header row (2-char prefix for alignment with marker column)
	fmt.Printf("  %-*s  %-*s  %-*s  %s\n",
		colWidths[0], headers[0],
		colWidths[1], headers[1],
		colWidths[2], headers[2],
		headers[3])

	// Print separator row (dashes match header label length)
	fmt.Printf("  %s  %s  %s  %s\n",
		strings.Repeat("-", len(headers[0])),
		strings.Repeat("-", len(headers[1])),
		strings.Repeat("-", len(headers[2])),
		strings.Repeat("-", len(headers[3])))

	// Print data rows
	for i, r := range rows {
		marker := "  "
		if entries[i].IsCurrent {
			marker = wt.ColorGreen + "*" + wt.ColorReset + " "
		}

		// Pad name and status accounting for ANSI escape codes
		namePad := colWidths[0] - displayWidth(r.name)
		statusPad := colWidths[2] - displayWidth(r.status)

		fmt.Printf("%s%s%s  %-*s  %s%s  %s\n",
			marker,
			r.name, strings.Repeat(" ", namePad),
			colWidths[1], r.branch,
			r.status, strings.Repeat(" ", statusPad),
			r.path)
	}

	fmt.Printf("\nTotal: %d worktree(s)\n", len(entries))
	return nil
}

type rawEntry struct {
	path   string
	branch string
}

func listWorktreeEntries() ([]rawEntry, error) {
	out, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var entries []rawEntry
	var current rawEntry

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			if current.path != "" {
				entries = append(entries, current)
			}
			current = rawEntry{path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "detached" {
			current.branch = "(detached)"
		}
	}
	if current.path != "" {
		entries = append(entries, current)
	}
	return entries, nil
}

func getEnrichedEntries(ctx *wt.RepoContext) ([]listEntry, error) {
	raw, err := listWorktreeEntries()
	if err != nil {
		return nil, err
	}

	currentDir, _ := os.Getwd()
	currentDir, _ = filepath.EvalSymlinks(currentDir)

	var mainPath string
	if len(raw) > 0 {
		mainPath = raw[0].path
	}

	var entries []listEntry
	for _, r := range raw {
		e := listEntry{
			Path:   r.path,
			Branch: r.branch,
			IsMain: r.path == mainPath,
		}

		if e.IsMain {
			e.Name = "main"
		} else {
			e.Name = filepath.Base(r.path)
		}

		// Check if current
		resolvedPath, _ := filepath.EvalSymlinks(r.path)
		if resolvedPath == currentDir || strings.HasPrefix(currentDir, resolvedPath+string(filepath.Separator)) {
			e.IsCurrent = true
		}

		// Get status by cd-ing into the worktree
		if _, statErr := os.Stat(r.path); statErr == nil {
			e.Dirty = checkDirty(r.path)
			if r.branch != "(detached)" {
				e.Unpushed = getUnpushedInDir(r.path, r.branch)
			}
		}

		entries = append(entries, e)
	}

	return entries, nil
}

func checkDirty(wtPath string) bool {
	// Check uncommitted changes
	cmd := exec.Command("git", "diff", "--quiet")
	cmd.Dir = wtPath
	if err := cmd.Run(); err != nil {
		return true
	}

	// Check staged changes
	cmd = exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = wtPath
	if err := cmd.Run(); err != nil {
		return true
	}

	// Check untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = wtPath
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func getUnpushedInDir(wtPath, branch string) int {
	upstream := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	upstream.Dir = wtPath
	upOut, err := upstream.Output()
	if err != nil {
		return 0
	}
	up := strings.TrimSpace(string(upOut))

	countCmd := exec.Command("git", "rev-list", "--count", up+".."+branch)
	countCmd.Dir = wtPath
	out, err := countCmd.Output()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return n
}
