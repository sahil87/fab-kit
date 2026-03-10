package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/fab-go/internal/resolve"
	sf "github.com/wvrdz/fab-kit/src/fab-go/internal/statusfile"
	"github.com/wvrdz/fab-kit/src/fab-go/internal/status"
	"gopkg.in/yaml.v3"
)

func paneMapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pane-map",
		Short: "Show tmux pane-to-worktree mapping with fab pipeline state",
		Args:  cobra.NoArgs,
		RunE:  runPaneMap,
	}
}

// paneEntry holds a single tmux pane's ID, tab (window) name, and current working directory.
type paneEntry struct {
	id  string
	tab string
	cwd string
}

// paneRow holds the resolved data for a single output row.
type paneRow struct {
	pane      string
	tab       string
	worktree  string
	change    string
	stage     string
	agent     string
}

func runPaneMap(cmd *cobra.Command, args []string) error {
	// Tmux session guard
	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: not inside a tmux session")
		os.Exit(1)
	}

	// Discover tmux panes
	panes, err := discoverPanes()
	if err != nil {
		return err
	}

	// Determine main worktree root for relative path computation.
	// We derive mainRoot from the first resolved pane's git root so that
	// the command works even when invoked from outside the repo.
	mainRoot := findMainWorktreeRoot(panes)

	// Resolve each pane to a row
	var rows []paneRow
	// Cache runtime files per worktree root to avoid re-reading
	runtimeCache := make(map[string]interface{}) // wtRoot -> loaded map or error sentinel

	for _, p := range panes {
		row, ok := resolvePane(p, mainRoot, runtimeCache)
		if ok {
			rows = append(rows, row)
		}
	}

	// Output
	if len(rows) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No fab worktrees found in tmux panes.")
		return nil
	}

	printPaneTable(cmd, rows)
	return nil
}

// discoverPanes runs `tmux list-panes -s` (current session only) and parses the output.
func discoverPanes() ([]paneEntry, error) {
	out, err := exec.Command("tmux", "list-panes", "-s", "-F", "#{pane_id} #{window_name} #{pane_current_path}").Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}

	var panes []paneEntry
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			continue
		}
		panes = append(panes, paneEntry{id: parts[0], tab: parts[1], cwd: parts[2]})
	}
	return panes, nil
}

// findMainWorktreeRoot returns the main worktree root by parsing `git worktree list --porcelain`.
// It derives the root from one of the discovered pane CWDs (via `git -C`) so that
// the command works even when invoked from outside the repo.
// Returns empty string if detection fails.
func findMainWorktreeRoot(panes []paneEntry) string {
	for _, p := range panes {
		out, err := exec.Command("git", "-C", p.cwd, "worktree", "list", "--porcelain").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "worktree ") {
				return strings.TrimPrefix(line, "worktree ")
			}
		}
	}
	return ""
}

// resolvePane resolves a pane entry into a table row.
// Returns false if the pane should be excluded (non-git, non-fab).
func resolvePane(p paneEntry, mainRoot string, runtimeCache map[string]interface{}) (paneRow, bool) {
	// Resolve git worktree root
	wtRoot, err := gitWorktreeRoot(p.cwd)
	if err != nil {
		return paneRow{}, false
	}

	// Check for fab/ directory
	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return paneRow{}, false
	}

	// Compute worktree display path
	wtDisplay := worktreeDisplayPath(wtRoot, mainRoot)

	// Read .fab-status.yaml symlink for the active change folder name
	changeName, folderName := readFabCurrent(wtRoot)

	// Read stage from .status.yaml
	stageName := "\u2014" // em dash
	if folderName != "" {
		statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
		if statusFile, err := sf.Load(statusPath); err == nil {
			stage, _ := status.DisplayStage(statusFile)
			stageName = stage
		}
	}

	// Determine agent state from .fab-runtime.yaml
	agentState := resolveAgentState(wtRoot, folderName, runtimeCache)

	return paneRow{
		pane:     p.id,
		tab:      p.tab,
		worktree: wtDisplay,
		change:   changeName,
		stage:    stageName,
		agent:    agentState,
	}, true
}

// gitWorktreeRoot returns the git worktree root for a given path.
func gitWorktreeRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// worktreeDisplayPath computes the display path for a worktree.
// Main worktree shows "(main)", others show path relative to main's parent.
func worktreeDisplayPath(wtRoot, mainRoot string) string {
	if mainRoot != "" && wtRoot == mainRoot {
		return "(main)"
	}
	if mainRoot != "" {
		parent := filepath.Dir(mainRoot)
		rel, err := filepath.Rel(parent, wtRoot)
		if err == nil {
			return rel + "/"
		}
	}
	// Fallback: basename with trailing slash
	return filepath.Base(wtRoot) + "/"
}

// readFabCurrent reads .fab-status.yaml symlink and returns (displayName, folderName).
// displayName is what shows in the Change column; folderName is the actual folder.
func readFabCurrent(wtRoot string) (string, string) {
	symlinkPath := filepath.Join(wtRoot, ".fab-status.yaml")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "(no change)", ""
	}

	// target is "fab/changes/{name}/.status.yaml"
	folderName := resolve.ExtractFolderFromSymlink(target)
	if folderName == "" {
		return "(no change)", ""
	}

	return folderName, folderName
}

// resolveAgentState determines the agent state string for a pane row.
func resolveAgentState(wtRoot, folderName string, cache map[string]interface{}) string {
	if folderName == "" {
		return "\u2014" // em dash for no change
	}

	rtPath := filepath.Join(wtRoot, ".fab-runtime.yaml")

	// Check cache
	var rtData map[string]interface{}
	var fileMissing bool

	if cached, ok := cache[wtRoot]; ok {
		switch v := cached.(type) {
		case map[string]interface{}:
			rtData = v
		case nil:
			fileMissing = true
		}
	} else {
		// Load and cache
		loaded, err := loadPaneMapRuntimeFile(rtPath)
		if err != nil {
			if os.IsNotExist(err) {
				// Cache absence of runtime file so we don't re-check repeatedly.
				cache[wtRoot] = nil
				fileMissing = true
			} else {
				// Do not cache nil for non-NotExist errors; they may be transient
				// or indicate a corrupted .fab-runtime.yaml. Surface a warning.
				fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", rtPath, err)
				return "?"
			}
		} else {
			cache[wtRoot] = loaded
			rtData = loaded
		}
	}

	if fileMissing {
		return "?"
	}

	// Look up change entry
	folderEntry, ok := rtData[folderName].(map[string]interface{})
	if !ok {
		return "active"
	}

	agentBlock, ok := folderEntry["agent"].(map[string]interface{})
	if !ok {
		return "active"
	}

	idleSince, ok := agentBlock["idle_since"]
	if !ok {
		return "active"
	}

	// Parse idle_since as Unix timestamp
	var ts int64
	switch v := idleSince.(type) {
	case int:
		ts = int64(v)
	case int64:
		ts = v
	case float64:
		ts = int64(v)
	default:
		return "active"
	}

	elapsed := time.Now().Unix() - ts
	if elapsed < 0 {
		elapsed = 0
	}

	return fmt.Sprintf("idle (%s)", formatIdleDuration(elapsed))
}

// loadPaneMapRuntimeFile reads and parses .fab-runtime.yaml.
// Returns an error if the file doesn't exist (distinct from loadRuntimeFile in runtime.go
// which returns an empty map for missing files — we need to distinguish missing vs empty).
func loadPaneMapRuntimeFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // includes os.IsNotExist
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if m == nil {
		m = make(map[string]interface{})
	}
	return m, nil
}

// formatIdleDuration formats elapsed seconds into a human-readable duration.
// Uses floor division: <60s -> Ns, 60s-3599s -> Nm, >=3600s -> Nh.
func formatIdleDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%dh", seconds/3600)
}

// printPaneTable prints the aligned pane map table.
func printPaneTable(cmd *cobra.Command, rows []paneRow) {
	// Compute column widths
	headers := [6]string{"Pane", "Tab", "Worktree", "Change", "Stage", "Agent"}
	widths := [6]int{len(headers[0]), len(headers[1]), len(headers[2]), len(headers[3]), len(headers[4]), len(headers[5])}

	for _, r := range rows {
		cols := [6]string{r.pane, r.tab, r.worktree, r.change, r.stage, r.agent}
		for i, c := range cols {
			if len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}

	// Print header
	fmtStr := fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%-%ds  %%-%ds  %%s\n", widths[0], widths[1], widths[2], widths[3], widths[4])
	fmt.Fprintf(cmd.OutOrStdout(), fmtStr, headers[0], headers[1], headers[2], headers[3], headers[4], headers[5])

	// Print data rows
	for _, r := range rows {
		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, r.pane, r.tab, r.worktree, r.change, r.stage, r.agent)
	}
}
