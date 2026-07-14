package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/spf13/cobra"
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
// session name, window index, the raw @rk_agent_state pane option value, and the
// tmux window ID.
type paneEntry struct {
	id         string
	tab        string
	cwd        string
	session    string
	index      int
	agentState string // raw @rk_agent_state option ("<state>:<epoch>"), "" when unset
	windowID   string // raw tmux #{window_id} (e.g. "@5"); "" when absent (legacy line)
}

// paneRow holds the resolved data for a single output row.
type paneRow struct {
	session      string
	windowIndex  int
	windowID     string // raw tmux #{window_id} (e.g. "@5"); "" when absent; the JSON window_id source
	pane         string
	tab          string
	worktree     string
	repo         string // absolute main-worktree root for this pane's repo (em dash when unresolved)
	change       string
	stage        string
	displayState string // state half of status.DisplayStage (em dash when unresolved)
	agent        string // Agent-column DISPLAY string (agentColumn output); table-only
	agentOption  string // raw @rk_agent_state option ("<state>:<epoch>", "" when unset); the JSON source of truth
	prURL        string // last entry in .status.yaml prs:, "" when absent/empty/unresolved
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
		return fmt.Errorf("not inside a tmux session")
	}

	// Discover tmux panes
	panes, err := discoverPanes(mode, sessionFlag, server)
	if err != nil {
		return err
	}

	// Resolve each pane to a row. The main worktree root used for relative
	// display-path computation is determined PER DISTINCT REPO (keyed by the
	// pane's git worktree root), so panes from different repos render their
	// worktree paths relative to their own repo's main root — not against a
	// single shared root derived from the first parsable pane.
	var rows []paneRow
	// Cache main-worktree root per pane's git worktree root to avoid re-running
	// `git worktree list` for every pane in the same repo.
	mainRootCache := make(map[string]string)
	// Cache the pane's git worktree root per cwd so each distinct cwd costs
	// exactly one `git rev-parse` per invocation (previously 2 per pane:
	// mainRootForPane and resolvePane each re-ran it).
	wtRootCache := make(map[string]string)

	for _, p := range panes {
		wtRoot := worktreeRootForPane(p.cwd, wtRootCache)
		mainRoot := mainRootForPane(p.cwd, wtRoot, mainRootCache)
		row, ok := resolvePane(p, wtRoot, mainRoot)
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

// worktreeRootForPane returns the pane's git worktree root, "" when cwd is
// not in a git repo, caching by cwd (hits AND misses) so each distinct cwd
// costs at most one `git rev-parse --show-toplevel` per invocation. The ""
// sentinel is load-bearing: resolvePane's non-git branch keys off it.
func worktreeRootForPane(cwd string, cache map[string]string) string {
	if wt, ok := cache[cwd]; ok {
		return wt
	}
	wt, err := pane.GitWorktreeRoot(cwd)
	if err != nil {
		wt = "" // not in a git repo
	}
	cache[cwd] = wt
	return wt
}

// mainRootForPane returns the main-worktree root for the repo that owns cwd,
// caching the result by the pane's (pre-resolved) git worktree root so panes
// sharing a repo reuse a single `git worktree list` lookup. wtRoot is the
// value from worktreeRootForPane — "" (not a git repo) short-circuits to ""
// (the same fallback FindMainWorktreeRoot uses for unresolvable paths),
// letting WorktreeDisplayPath fall back to basename display.
func mainRootForPane(cwd, wtRoot string, cache map[string]string) string {
	if wtRoot == "" {
		return ""
	}
	if mr, ok := cache[wtRoot]; ok {
		return mr
	}
	mr := pane.FindMainWorktreeRoot([]string{cwd})
	cache[wtRoot] = mr
	return mr
}

// sessionMode controls how discoverPanes selects tmux sessions.
type sessionMode int

const (
	sessionDefault sessionMode = iota // current session (tmux list-panes -s)
	sessionNamed                      // specific session by name (-t <name>)
	sessionAll                        // all sessions
)

// tmuxPaneFormat is the format string passed to tmux list-panes -F. It carries
// seven tab-separated fields. #{@rk_agent_state} (field 6) carries the
// agent-state pane option so the Agent column is resolved from the SAME
// list-panes call — zero extra subprocesses (and the tmux_server disambiguation
// problem evaporates, since a pane option lives on exactly one server's pane);
// tmux emits an empty field for it when the option is unset, so it is a
// possibly-empty MIDDLE field. #{window_id} (field 7) is the tmux server-assigned
// window identifier (@N), stable for a window's lifetime and never empty — it is
// deliberately the TRAILING field so the possibly-empty agent-state field can
// never leave the line ending in a tab.
const tmuxPaneFormat = "#{pane_id}\t#{window_name}\t#{pane_current_path}\t#{session_name}\t#{window_index}\t#{@rk_agent_state}\t#{window_id}"

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

// listAllPanesArgs builds the tmux argv for the single server-wide
// `list-panes -a` enumeration used by --all-sessions. When server is
// non-empty, prepends `-L <server>`. Extracted for unit-testability.
func listAllPanesArgs(server string) []string {
	return pane.WithServer(server, "list-panes", "-a", "-F", tmuxPaneFormat)
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

// discoverAllSessions lists every pane on the server in ONE `list-panes -a`
// call — tmuxPaneFormat already carries #{session_name}, so the per-session
// loop (list-sessions + one list-panes per session) was N+1 subprocesses for
// identical rows. The single call also sidesteps `-t <session>` target
// resolution (exact-then-prefix matching) entirely. When server is
// non-empty, the tmux invocation is scoped via `-L <server>`.
func discoverAllSessions(server string) ([]paneEntry, error) {
	out, err := exec.Command("tmux", listAllPanesArgs(server)...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}
	return parsePaneLines(string(out))
}

// parsePaneLines parses tmux list-panes output into paneEntry slices. The
// format string carries seven tab-separated fields. #{@rk_agent_state}
// (field 6) is a possibly-empty MIDDLE field — empty when the option is unset;
// #{window_id} (field 7) is the never-empty TRAILING field. Trimming is
// per-line and newline-only (never TrimSpace), which stays load-bearing for
// legacy SIX-field lines whose empty agent-state field left the line ending in
// a tab. Lines are parsed with graded tolerance: seven fields yield both
// agentState and windowID; a legacy six-field line yields agentState with an
// empty windowID; a legacy five-field line yields neither; lines with fewer
// than five fields are skipped.
func parsePaneLines(output string) ([]paneEntry, error) {
	var panes []paneEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.Trim(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 7)
		if len(parts) < 5 {
			continue
		}
		idx, _ := strconv.Atoi(parts[4])
		agentState := ""
		if len(parts) >= 6 {
			agentState = strings.TrimSpace(parts[5])
		}
		windowID := ""
		if len(parts) == 7 {
			windowID = parts[6]
		}
		panes = append(panes, paneEntry{
			id:         parts[0],
			tab:        parts[1],
			cwd:        parts[2],
			session:    parts[3],
			index:      idx,
			agentState: agentState,
			windowID:   windowID,
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

// resolvePane resolves a pane entry into a table row. Agent state comes from
// the pane's @rk_agent_state option (carried in paneEntry.agentState from the
// list-panes format string) — independent of whether a change is active.
// This is the three-axis model: Change (from .fab-status.yaml), Agent (from
// the pane option), and (not shown here) Process (opt-in via `fab pane process`).
// wtRoot is the pane's pre-resolved git worktree root from
// worktreeRootForPane ("" = not a git repo) — threaded in, like mainRoot,
// so resolvePane never re-spawns `git rev-parse`.
func resolvePane(p paneEntry, wtRoot, mainRoot string) (paneRow, bool) {
	emDash := "\u2014"

	if wtRoot == "" {
		// Non-git pane: the CHANGE axis is em-dash (no fab context), but the
		// AGENT axis still comes from the pane's @rk_agent_state option — a
		// non-git pane can run an instrumented agent. Hardcoding em-dash here
		// would make `pane map` disagree with `pane send`/`capture` (which
		// resolve the option regardless of git/fab context), so resolve it via
		// the same agentColumn helper the git branch below uses.
		return paneRow{
			session:      p.session,
			windowIndex:  p.index,
			windowID:     p.windowID,
			pane:         p.id,
			tab:          p.tab,
			worktree:     filepath.Base(p.cwd) + "/",
			repo:         emDash,
			change:       emDash,
			stage:        emDash,
			displayState: emDash,
			agent:        agentColumn(p.agentState),
			agentOption:  p.agentState,
		}, true
	}

	// repo is the absolute main-worktree root for this pane's repo; em dash
	// when it could not be resolved (e.g. detached / non-standard layout).
	repoRoot := mainRoot
	if repoRoot == "" {
		repoRoot = emDash
	}

	fabDir := filepath.Join(wtRoot, "fab")
	fabDirMissing := false
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		fabDirMissing = true
	}

	wtDisplay := pane.WorktreeDisplayPath(wtRoot, mainRoot)

	changeName := emDash
	stageName := emDash
	stageState := emDash
	prURL := ""
	var folderName string
	if !fabDirMissing {
		changeName, folderName = pane.ReadFabCurrent(wtRoot)
		if folderName != "" {
			statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
			if statusFile, err := sf.Load(statusPath); err == nil {
				stage, state := status.DisplayStage(statusFile)
				stageName = stage
				stageState = state
				if n := len(statusFile.PRs); n > 0 {
					prURL = statusFile.PRs[n-1] // last = most recent
				}
			}
		}
	}

	// Agent resolution runs regardless of fabDir presence — the pane
	// option describes the agent in this pane whether or not a change is
	// active. active/waiting/idle-with-duration, em dash when unknown.
	agentState := agentColumn(p.agentState)

	return paneRow{
		session:      p.session,
		windowIndex:  p.index,
		windowID:     p.windowID,
		pane:         p.id,
		tab:          p.tab,
		worktree:     wtDisplay,
		repo:         repoRoot,
		change:       changeName,
		stage:        stageName,
		displayState: stageState,
		agent:        agentState,
		agentOption:  p.agentState,
		prURL:        prURL,
	}, true
}

// agentColumn renders the raw @rk_agent_state option value into the Agent
// column display string: "active" / "waiting" / "idle (<dur>)" / "—" (em dash
// for the unknown case — absent option, unparseable value, or unknown token).
// Idle carries the epoch-derived duration; active/waiting do not.
func agentColumn(rawOption string) string {
	state, idleDur := pane.AgentDisplayFromOption(rawOption)
	switch {
	case state == "":
		return "—"
	case state == pane.AgentStateIdle:
		return fmt.Sprintf("idle (%s)", idleDur)
	default:
		return state
	}
}

// paneJSON represents a single pane in JSON output.
type paneJSON struct {
	Session           string  `json:"session"`
	WindowIndex       int     `json:"window_index"`
	WindowID          *string `json:"window_id"`
	Pane              string  `json:"pane"`
	Tab               string  `json:"tab"`
	Worktree          string  `json:"worktree"`
	Repo              *string `json:"repo"`
	Change            *string `json:"change"`
	Stage             *string `json:"stage"`
	DisplayState      *string `json:"display_state"`
	AgentState        *string `json:"agent_state"`
	AgentIdleDuration *string `json:"agent_idle_duration"`
	PRURL             *string `json:"pr_url"`
	PRNumber          *int    `json:"pr_number"`
}

// toNullable converts a table display string to a *string for JSON output.
// The unresolved sentinels \u2014 em dash, "(no change)", and the empty string \u2014
// all map to JSON null.
func toNullable(s string) *string {
	if s == "\u2014" || s == "(no change)" || s == "" {
		return nil
	}
	return &s
}

// parsePRNumber extracts the PR number from a GitHub PR URL's trailing
// /pull/<n> segment. Returns (n, true) on success, (0, false) when the URL
// has no parseable /pull/<n> segment (no /pull/, non-numeric segment, or an
// empty URL, or a non-positive number). A trailing path, query string, or
// fragment after the number (e.g. /pull/42/files, /pull/42?w=1,
// /pull/42#issuecomment-1) is tolerated — only the digits immediately after
// the last "/pull/" are parsed. The last "/pull/" wins so a repo or org path
// segment named "pull" cannot shadow the real PR segment.
func parsePRNumber(url string) (int, bool) {
	const marker = "/pull/"
	i := strings.LastIndex(url, marker)
	if i < 0 {
		return 0, false
	}
	seg := url[i+len(marker):]
	// Cut the number off at the first path, query, or fragment delimiter.
	if cut := strings.IndexAny(seg, "/?#"); cut >= 0 {
		seg = seg[:cut]
	}
	n, err := strconv.Atoi(seg)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// agentJSONFields derives the JSON agent_state / agent_idle_duration pair
// DIRECTLY from the raw @rk_agent_state option, NOT from the Agent-column
// display string. This keeps the run-kit-consumed JSON contract independent of
// the human display format: agent_state \u2208 {active, waiting, idle, null} and
// agent_idle_duration is non-null only for idle. Unknown (unparseable / absent
// / unknown token) maps both to null.
func agentJSONFields(rawOption string) (state *string, idleDuration *string) {
	st, dur := pane.AgentDisplayFromOption(rawOption)
	if st == "" {
		return nil, nil
	}
	if dur == "" {
		return &st, nil
	}
	return &st, &dur
}

// printPaneJSON marshals rows to a JSON array and writes to cmd's stdout.
func printPaneJSON(cmd *cobra.Command, rows []paneRow) error {
	out := make([]paneJSON, len(rows))
	for i, r := range rows {
		agentState, idleDur := agentJSONFields(r.agentOption)
		var prNum *int
		if n, ok := parsePRNumber(r.prURL); ok {
			prNum = &n
		}
		out[i] = paneJSON{
			Session:           r.session,
			WindowIndex:       r.windowIndex,
			WindowID:          toNullable(r.windowID),
			Pane:              r.pane,
			Tab:               r.tab,
			Worktree:          r.worktree,
			Repo:              toNullable(r.repo),
			Change:            toNullable(r.change),
			Stage:             toNullable(r.stage),
			DisplayState:      toNullable(r.displayState),
			AgentState:        agentState,
			AgentIdleDuration: idleDur,
			PRURL:             toNullable(r.prURL),
			PRNumber:          prNum,
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
