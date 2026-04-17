package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
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
	server, _ := cmd.Flags().GetString("server")

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
	panes, err := discoverPanes(mode, sessionFlag, server)
	if err != nil {
		return err
	}

	// Determine main worktree root for relative path computation.
	cwds := make([]string, len(panes))
	for i, p := range panes {
		cwds[i] = p.cwd
	}
	mainRoot := pane.FindMainWorktreeRoot(cwds)

	// Resolve each pane to a row
	var rows []paneRow
	// Cache runtime files per worktree root to avoid re-reading
	runtimeCache := make(map[string]interface{})

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
// When server is non-empty, every tmux invocation is scoped via `-L <server>`.
func discoverPanes(mode sessionMode, sessionName, server string) ([]paneEntry, error) {
	switch mode {
	case sessionAll:
		return discoverAllSessions(server)
	case sessionNamed:
		return discoverSessionPanes(sessionName, server)
	default:
		return discoverSessionPanes("", server)
	}
}

// listPanesArgs builds the tmux argv for `list-panes -s ...`. When name is
// non-empty, adds `-t <name>`. When server is non-empty, prepends `-L <server>`.
// Extracted for unit-testability of argv construction.
func listPanesArgs(name, server string) []string {
	args := []string{"list-panes", "-s", "-F", tmuxPaneFormat}
	if name != "" {
		args = append(args, "-t", name)
	}
	return pane.WithServer(server, args...)
}

// listSessionsArgs builds the tmux argv for `list-sessions -F ...`. When server
// is non-empty, prepends `-L <server>`. Extracted for unit-testability.
func listSessionsArgs(server string) []string {
	return pane.WithServer(server, "list-sessions", "-F", "#{session_name}")
}

// discoverSessionPanes lists panes for a single session (or the current session if name is empty).
// When server is non-empty, the tmux invocation is scoped via `-L <server>`.
func discoverSessionPanes(name, server string) ([]paneEntry, error) {
	out, err := exec.Command("tmux", listPanesArgs(name, server)...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}
	return parsePaneLines(string(out))
}

// discoverAllSessions enumerates all tmux sessions, then lists panes for each.
// When server is non-empty, every tmux invocation is scoped via `-L <server>`.
func discoverAllSessions(server string) ([]paneEntry, error) {
	out, err := exec.Command("tmux", listSessionsArgs(server)...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-sessions: %w", err)
	}
	var all []paneEntry
	for _, sess := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		sess = strings.TrimSpace(sess)
		if sess == "" {
			continue
		}
		panes, err := discoverSessionPanes(sess, server)
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

// resolvePaneChange resolves a pane entry to its active change folder name.
func resolvePaneChange(p paneEntry) string {
	wtRoot, err := pane.GitWorktreeRoot(p.cwd)
	if err != nil {
		return ""
	}

	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ""
	}

	_, folderName := pane.ReadFabCurrent(wtRoot)
	return folderName
}

// matchPanesByFolder is a testable helper that matches pane entries to a change folder.
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
func resolvePane(p paneEntry, mainRoot string, runtimeCache map[string]interface{}) (paneRow, bool) {
	emDash := "\u2014"

	wtRoot, err := pane.GitWorktreeRoot(p.cwd)
	if err != nil {
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

	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return paneRow{
			session:     p.session,
			windowIndex: p.index,
			pane:        p.id,
			tab:         p.tab,
			worktree:    pane.WorktreeDisplayPath(wtRoot, mainRoot),
			change:      emDash,
			stage:       emDash,
			agent:       emDash,
		}, true
	}

	wtDisplay := pane.WorktreeDisplayPath(wtRoot, mainRoot)
	changeName, folderName := pane.ReadFabCurrent(wtRoot)

	stageName := emDash
	if folderName != "" {
		statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
		if statusFile, err := sf.Load(statusPath); err == nil {
			stage, _ := status.DisplayStage(statusFile)
			stageName = stage
		}
	}

	agentState := pane.ResolveAgentStateWithCache(wtRoot, folderName, runtimeCache)

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
func toNullable(s string) *string {
	if s == "\u2014" || s == "(no change)" {
		return nil
	}
	return &s
}

// splitAgentState splits the combined agent display string into separate
// state and idle duration values for JSON output.
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
func printPaneTable(cmd *cobra.Command, rows []paneRow, showSession bool) {
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

	var fmtParts []string
	for i := range cols {
		if i == len(cols)-1 {
			fmtParts = append(fmtParts, "%s")
		} else {
			fmtParts = append(fmtParts, fmt.Sprintf("%%-%ds", widths[i]))
		}
	}
	fmtStr := strings.Join(fmtParts, "  ") + "\n"

	hvals := make([]interface{}, len(cols))
	for i, c := range cols {
		hvals[i] = c.header
	}
	fmt.Fprintf(cmd.OutOrStdout(), fmtStr, hvals...)

	for _, r := range rows {
		vals := make([]interface{}, len(cols))
		for i, c := range cols {
			vals[i] = c.value(r)
		}
		fmt.Fprintf(cmd.OutOrStdout(), fmtStr, vals...)
	}
}
