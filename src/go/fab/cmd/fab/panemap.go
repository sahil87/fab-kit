package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
	sf "github.com/wvrdz/fab-kit/src/go/fab/internal/statusfile"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/status"
	"gopkg.in/yaml.v3"
)

func paneMapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "map",
		Short: "Show tmux pane-to-worktree mapping with fab pipeline state",
		Args:  cobra.NoArgs,
		RunE:  runPaneMap,
	}
	cmd.Flags().Bool("json", false, "Output as JSON array")
	cmd.Flags().String("session", "", "Target a specific tmux session by name")
	cmd.Flags().Bool("all-sessions", false, "Query all tmux sessions")
	cmd.MarkFlagsMutuallyExclusive("session", "all-sessions")
	return cmd
}

// paneEntry holds a single tmux pane's ID, tab (window) name, current working directory,
// session name, and window index.
type paneEntry struct {
	id      string
	tab     string
	cwd     string
	session string
	index   int
}

// paneRow holds the resolved data for a single output row.
type paneRow struct {
	session     string
	windowIndex int
	pane        string
	tab         string
	worktree    string
	change      string
	stage       string
	agent       string
}

func runPaneMap(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")
	sessionFlag, _ := cmd.Flags().GetString("session")
	allSessionsFlag, _ := cmd.Flags().GetBool("all-sessions")

	// Determine session targeting mode
	mode := sessionDefault
	if allSessionsFlag {
		mode = sessionAll
	} else if sessionFlag != "" {
		mode = sessionNamed
	}

	// $TMUX guard only when neither --session nor --all-sessions is set
	if mode == sessionDefault && os.Getenv("TMUX") == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: not inside a tmux session")
		os.Exit(1)
	}

	// Discover tmux panes
	panes, err := discoverPanes(mode, sessionFlag)
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
		fmt.Fprintln(cmd.OutOrStdout(), "No tmux panes found.")
		return nil
	}

	if jsonFlag {
		return printPaneJSON(cmd, rows)
	}

	printPaneTable(cmd, rows, allSessionsFlag)
	return nil
}

// sessionMode controls how discoverPanes selects tmux sessions.
type sessionMode int

const (
	sessionDefault  sessionMode = iota // current session (tmux list-panes -s)
	sessionNamed                       // specific session by name (-t <name>)
	sessionAll                         // all sessions
)

// tmuxPaneFormat is the format string passed to tmux list-panes -F.
const tmuxPaneFormat = "#{pane_id}\t#{window_name}\t#{pane_current_path}\t#{session_name}\t#{window_index}"

// discoverPanes runs `tmux list-panes` with session targeting and parses the output.
// Uses tab as the field delimiter so that window names containing spaces are handled correctly.
func discoverPanes(mode sessionMode, sessionName string) ([]paneEntry, error) {
	switch mode {
	case sessionAll:
		return discoverAllSessions()
	case sessionNamed:
		return discoverSessionPanes(sessionName)
	default:
		return discoverSessionPanes("")
	}
}

// discoverSessionPanes lists panes for a single session (or the current session if name is empty).
func discoverSessionPanes(name string) ([]paneEntry, error) {
	args := []string{"list-panes", "-s", "-F", tmuxPaneFormat}
	if name != "" {
		args = append(args, "-t", name)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}
	return parsePaneLines(string(out))
}

// discoverAllSessions enumerates all tmux sessions, then lists panes for each.
func discoverAllSessions() ([]paneEntry, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-sessions: %w", err)
	}
	var all []paneEntry
	for _, sess := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		sess = strings.TrimSpace(sess)
		if sess == "" {
			continue
		}
		panes, err := discoverSessionPanes(sess)
		if err != nil {
			return nil, err
		}
		all = append(all, panes...)
	}
	return all, nil
}

// parsePaneLines parses tmux list-panes output into paneEntry slices.
func parsePaneLines(output string) ([]paneEntry, error) {
	var panes []paneEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) != 5 {
			continue
		}
		idx, _ := strconv.Atoi(parts[4])
		panes = append(panes, paneEntry{
			id:      parts[0],
			tab:     parts[1],
			cwd:     parts[2],
			session: parts[3],
			index:   idx,
		})
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

// resolvePaneChange resolves a pane entry to its active change folder name.
// Returns empty string if the pane is not in a git repo, has no fab/ directory,
// or has no active change.
func resolvePaneChange(p paneEntry) string {
	wtRoot, err := gitWorktreeRoot(p.cwd)
	if err != nil {
		return ""
	}

	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ""
	}

	_, folderName := readFabCurrent(wtRoot)
	return folderName
}

// matchPanesByFolder is a testable helper that matches pane entries to a change folder.
// Returns (matchedPaneIDs, warningMessage).
func matchPanesByFolder(panes []paneEntry, folder string, resolveFunc func(paneEntry) string) ([]string, string) {
	var matches []string
	for _, p := range panes {
		if resolveFunc(p) == folder {
			matches = append(matches, p.id)
		}
	}

	warning := ""
	if len(matches) > 1 {
		warning = fmt.Sprintf("Warning: multiple panes found for %s, using %s",
			resolve.ExtractID(folder), matches[0])
	}

	return matches, warning
}

// resolvePane resolves a pane entry into a table row.
// Returns true for all panes — non-git and non-fab panes show fallback values.
func resolvePane(p paneEntry, mainRoot string, runtimeCache map[string]interface{}) (paneRow, bool) {
	emDash := "\u2014"

	// Resolve git worktree root
	wtRoot, err := gitWorktreeRoot(p.cwd)
	if err != nil {
		// Non-git directory: show basename of CWD
		return paneRow{
			session:     p.session,
			windowIndex: p.index,
			pane:        p.id,
			tab:         p.tab,
			worktree:    filepath.Base(p.cwd) + "/",
			change:      emDash,
			stage:       emDash,
			agent:       emDash,
		}, true
	}

	// Check for fab/ directory
	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		// Git repo without fab/: show worktree path
		return paneRow{
			session:     p.session,
			windowIndex: p.index,
			pane:        p.id,
			tab:         p.tab,
			worktree:    worktreeDisplayPath(wtRoot, mainRoot),
			change:      emDash,
			stage:       emDash,
			agent:       emDash,
		}, true
	}

	// Compute worktree display path
	wtDisplay := worktreeDisplayPath(wtRoot, mainRoot)

	// Read .fab-status.yaml symlink for the active change folder name
	changeName, folderName := readFabCurrent(wtRoot)

	// Read stage from .status.yaml
	stageName := emDash
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
		session:     p.session,
		windowIndex: p.index,
		pane:        p.id,
		tab:         p.tab,
		worktree:    wtDisplay,
		change:      changeName,
		stage:       stageName,
		agent:       agentState,
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

// paneJSON represents a single pane in JSON output.
type paneJSON struct {
	Session           string  `json:"session"`
	WindowIndex       int     `json:"window_index"`
	Pane              string  `json:"pane"`
	Tab               string  `json:"tab"`
	Worktree          string  `json:"worktree"`
	Change            *string `json:"change"`
	Stage             *string `json:"stage"`
	AgentState        *string `json:"agent_state"`
	AgentIdleDuration *string `json:"agent_idle_duration"`
}

// toNullable converts a table display string to a *string for JSON output.
// Em-dash and "(no change)" map to nil (JSON null).
func toNullable(s string) *string {
	if s == "\u2014" || s == "(no change)" {
		return nil
	}
	return &s
}

// splitAgentState splits the combined agent display string into separate
// state and idle duration values for JSON output.
//
// Mapping:
//
//	"active"       → ("active", nil)
//	"idle (2m)"    → ("idle", "2m")
//	"—" (em dash)  → (nil, nil)
//	"?"            → ("unknown", nil)
func splitAgentState(agent string) (state *string, idleDuration *string) {
	switch {
	case agent == "\u2014":
		return nil, nil
	case agent == "?":
		s := "unknown"
		return &s, nil
	case strings.HasPrefix(agent, "idle ("):
		s := "idle"
		dur := strings.TrimSuffix(strings.TrimPrefix(agent, "idle ("), ")")
		return &s, &dur
	default:
		// "active" or any other value
		return &agent, nil
	}
}

// printPaneJSON marshals rows to a JSON array and writes to cmd's stdout.
func printPaneJSON(cmd *cobra.Command, rows []paneRow) error {
	out := make([]paneJSON, len(rows))
	for i, r := range rows {
		agentState, idleDur := splitAgentState(r.agent)
		out[i] = paneJSON{
			Session:           r.session,
			WindowIndex:       r.windowIndex,
			Pane:              r.pane,
			Tab:               r.tab,
			Worktree:          r.worktree,
			Change:            toNullable(r.change),
			Stage:             toNullable(r.stage),
			AgentState:        agentState,
			AgentIdleDuration: idleDur,
		}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// printPaneTable prints the aligned pane map table.
// When showSession is true, a Session column is prepended.
func printPaneTable(cmd *cobra.Command, rows []paneRow, showSession bool) {
	// Build dynamic column list
	type col struct {
		header string
		value  func(r paneRow) string
	}

	var cols []col
	if showSession {
		cols = append(cols, col{"Session", func(r paneRow) string { return r.session }})
	}
	cols = append(cols,
		col{"Pane", func(r paneRow) string { return r.pane }},
		col{"WinIdx", func(r paneRow) string { return strconv.Itoa(r.windowIndex) }},
		col{"Tab", func(r paneRow) string { return r.tab }},
		col{"Worktree", func(r paneRow) string { return r.worktree }},
		col{"Change", func(r paneRow) string { return r.change }},
		col{"Stage", func(r paneRow) string { return r.stage }},
		col{"Agent", func(r paneRow) string { return r.agent }},
	)

	// Compute column widths
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.header)
	}
	for _, r := range rows {
		for i, c := range cols {
			if v := len(c.value(r)); v > widths[i] {
				widths[i] = v
			}
		}
	}

	// Build format string: all columns left-aligned with two-space gap, last column unpadded
	var fmtParts []string
	for i := range cols {
		if i == len(cols)-1 {
			fmtParts = append(fmtParts, "%s")
		} else {
			fmtParts = append(fmtParts, fmt.Sprintf("%%-%ds", widths[i]))
		}
	}
	fmtStr := strings.Join(fmtParts, "  ") + "\n"

	// Print header
	hvals := make([]interface{}, len(cols))
	for i, c := range cols {
		hvals[i] = c.header
	}
	fmt.Fprintf(cmd.OutOrStdout(), fmtStr, hvals...)

	// Print data rows
	for _, r := range rows {
		vals := make([]interface{}, len(cols))
		for i, c := range cols {
			vals[i] = c.value(r)
		}
		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, vals...)
	}
}
