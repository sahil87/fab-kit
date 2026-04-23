package pane

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	"gopkg.in/yaml.v3"
)

// Runtime schema keys — kept in sync with the canonical definitions in the
// runtime package. Duplicating the constants here avoids a circular import
// (runtime → pane would be circular).
const (
	agentsKey    = "_agents"
	idleSinceKey = "idle_since"
	tmuxPaneKey  = "tmux_pane"
	tmuxSrvKey   = "tmux_server"
)

// WithServer prepends "-L <server>" to a tmux argument list when server is
// non-empty, and returns args unchanged otherwise. Callers use this to build
// the argv for `exec.Command("tmux", ...)` so the --server/-L CLI flag is
// plumbed through to every tmux invocation. The input args slice is never
// mutated; a new slice is allocated when a prefix is added.
//
// Exported (rather than unexported as originally drafted) so that the
// `cmd/fab` package — which owns the pane subcommand wiring — shares the
// single canonical argv builder instead of duplicating the logic.
func WithServer(server string, args ...string) []string {
	if server == "" {
		return args
	}
	return append([]string{"-L", server}, args...)
}

// PaneContext holds resolved fab context for a single tmux pane.
type PaneContext struct {
	Pane              string
	CWD               string
	WorktreeRoot      string
	WorktreeDisplay   string
	Change            *string // nil if no active change
	Stage             *string // nil if no stage
	AgentState        *string // nil if not applicable
	AgentIdleDuration *string // nil if not idle
}

// ValidatePane checks that a tmux pane exists by running `tmux list-panes -a`
// and verifying the pane ID appears in the output. If server is non-empty, the
// tmux invocation is scoped to that server via `-L <server>`.
func ValidatePane(paneID, server string) error {
	out, err := exec.Command("tmux", WithServer(server, "list-panes", "-a", "-F", "#{pane_id}")...).Output()
	if err != nil {
		return fmt.Errorf("tmux list-panes: %w", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == paneID {
			return nil
		}
	}
	return fmt.Errorf("pane %s not found", paneID)
}

// ReadWindowName returns the current window name for a tmux pane via
// `tmux display-message -p -t <pane> '#W'`. If server is non-empty, the tmux
// invocation is scoped to that server via `-L <server>`. The returned name is
// trimmed of trailing newline/whitespace.
func ReadWindowName(paneID, server string) (string, error) {
	out, err := exec.Command("tmux", WithServer(server, "display-message", "-p", "-t", paneID, "#W")...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux display-message: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetPanePID returns the shell PID of a tmux pane. If server is non-empty, the
// tmux invocation is scoped to that server via `-L <server>`.
func GetPanePID(paneID, server string) (int, error) {
	out, err := exec.Command("tmux", WithServer(server, "display-message", "-t", paneID, "-p", "#{pane_pid}")...).Output()
	if err != nil {
		return 0, fmt.Errorf("tmux display-message: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parsing pane PID: %w", err)
	}
	return pid, nil
}

// ResolvePaneContext resolves the fab context for a given tmux pane.
// mainRoot is the main worktree root used for computing relative display paths.
// Pass "" if unknown — WorktreeDisplay will fall back to filepath.Base.
// If server is non-empty, the tmux invocation is scoped to that server via
// `-L <server>`; file reads, git-worktree detection, and runtime-file lookups
// are independent of the tmux server.
//
// Agent-state resolution is independent of whether a change is active — a
// pane with a running Claude in "discussion mode" (no change) will still
// populate AgentState if a matching `_agents` entry exists.
func ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error) {
	// Get pane CWD
	out, err := exec.Command("tmux", WithServer(server, "display-message", "-t", paneID, "-p", "#{pane_current_path}")...).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux display-message: %w", err)
	}
	cwd := strings.TrimSpace(string(out))

	ctx := &PaneContext{
		Pane: paneID,
		CWD:  cwd,
	}

	// Resolve git worktree root
	wtRoot, err := GitWorktreeRoot(cwd)
	if err != nil {
		// Not in a git repo
		ctx.WorktreeRoot = cwd
		ctx.WorktreeDisplay = filepath.Base(cwd) + "/"
		return ctx, nil
	}
	ctx.WorktreeRoot = wtRoot

	// Set worktree display using the same logic as pane map
	ctx.WorktreeDisplay = WorktreeDisplayPath(wtRoot, mainRoot)

	// Check for fab/ directory
	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ctx, nil
	}

	// Read .fab-status.yaml symlink for the active change (independent axis).
	_, folderName := ReadFabCurrent(wtRoot)
	if folderName != "" {
		ctx.Change = &folderName

		// Read stage from .status.yaml
		statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
		if statusFile, err := sf.Load(statusPath); err == nil {
			stage, _ := status.DisplayStage(statusFile)
			ctx.Stage = &stage
		}
	}

	// Agent resolution — independent of whether a change is active. Runs
	// regardless of folderName so discussion-mode panes get populated.
	state, idleDur := ResolveAgentState(wtRoot, paneID, server)
	if state != "" {
		ctx.AgentState = &state
	}
	if idleDur != "" {
		ctx.AgentIdleDuration = &idleDur
	}

	return ctx, nil
}

// FindMainWorktreeRoot returns the main worktree root by parsing
// `git worktree list --porcelain`. Derives the root from one of the
// provided pane CWDs so the command works even outside the repo.
func FindMainWorktreeRoot(cwds []string) string {
	for _, cwd := range cwds {
		out, err := exec.Command("git", "-C", cwd, "worktree", "list", "--porcelain").Output()
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

// GitWorktreeRoot returns the git worktree root for a given path.
func GitWorktreeRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// WorktreeDisplayPath computes the display path for a worktree.
// Main worktree shows "(main)", others show path relative to main's parent.
func WorktreeDisplayPath(wtRoot, mainRoot string) string {
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
	return filepath.Base(wtRoot) + "/"
}

// ReadFabCurrent reads .fab-status.yaml symlink and returns (displayName, folderName).
func ReadFabCurrent(wtRoot string) (string, string) {
	symlinkPath := filepath.Join(wtRoot, ".fab-status.yaml")
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return "(no change)", ""
	}
	folderName := resolve.ExtractFolderFromSymlink(target)
	if folderName == "" {
		return "(no change)", ""
	}
	return folderName, folderName
}

// findAgentByPane scans the given `_agents` map for an entry matching
// paneID (exact `tmux_pane` equality) AND tmux_server (if the entry has a
// non-empty tmux_server, it must equal server; an entry with empty
// tmux_server matches any server). Returns the matching entry map and true
// on hit; nil, false otherwise.
//
// When multiple entries match, preference goes to: (1) an active entry (no
// `idle_since`), then (2) the most recently idle entry (largest idle_since).
// This gives "active agent here" priority over stale idle entries for the
// same pane — matching the pane-map semantics in the spec.
func findAgentByPane(rtData map[string]interface{}, paneID, server string) (map[string]interface{}, bool) {
	agents, ok := rtData[agentsKey].(map[string]interface{})
	if !ok {
		return nil, false
	}

	var activeMatch map[string]interface{}
	var idleMatch map[string]interface{}
	var idleMatchTs int64

	for _, raw := range agents {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		entryPane, _ := entry[tmuxPaneKey].(string)
		if entryPane != paneID {
			continue
		}
		entrySrv, _ := entry[tmuxSrvKey].(string)
		if entrySrv != "" && server != "" && entrySrv != server {
			continue
		}
		// Entry matches — determine if active or idle.
		if _, hasIdle := entry[idleSinceKey]; !hasIdle {
			// Active entry wins immediately.
			if activeMatch == nil {
				activeMatch = entry
			}
			continue
		}
		// Idle entry — keep the most recent idle_since for disambiguation.
		ts, _ := asInt64(entry[idleSinceKey])
		if idleMatch == nil || ts > idleMatchTs {
			idleMatch = entry
			idleMatchTs = ts
		}
	}

	if activeMatch != nil {
		return activeMatch, true
	}
	if idleMatch != nil {
		return idleMatch, true
	}
	return nil, false
}

// ResolveAgentState determines the agent state and idle duration for a
// pane by matching `_agents[*].tmux_pane` to paneID. Returns (state,
// idleDuration). state is "active" or "idle". idleDuration is non-empty
// only when state is "idle". Returns ("", "") when no entry matches.
//
// The pane-keyed resolution is independent of the active change — a pane
// in discussion mode still returns "idle"/"active" when an entry matches,
// enabling pane-map visibility for pre-intake work.
func ResolveAgentState(wtRoot, paneID, server string) (string, string) {
	if paneID == "" {
		return "", ""
	}

	rtPath := filepath.Join(wtRoot, ".fab-runtime.yaml")
	rtData, err := LoadRuntimeFile(rtPath)
	if err != nil {
		// Missing file or parse error — no match, no state.
		return "", ""
	}

	entry, ok := findAgentByPane(rtData, paneID, server)
	if !ok {
		return "", ""
	}

	idleVal, hasIdle := entry[idleSinceKey]
	if !hasIdle {
		return "active", ""
	}
	ts, ok := asInt64(idleVal)
	if !ok {
		return "active", ""
	}
	elapsed := time.Now().Unix() - ts
	if elapsed < 0 {
		elapsed = 0
	}
	return "idle", FormatIdleDuration(elapsed)
}

// ResolveAgentStateWithCache is the pane-map variant of ResolveAgentState
// that uses a per-worktree cache to avoid re-reading .fab-runtime.yaml for
// multiple panes in the same worktree. Returns the combined display string
// used by pane map (e.g., "active", "idle (2m)", em dash for no match).
func ResolveAgentStateWithCache(wtRoot, paneID, server string, cache map[string]interface{}) string {
	emDash := "\u2014"
	if paneID == "" {
		return emDash
	}

	rtData, ok := loadRuntimeForCache(wtRoot, cache)
	if !ok {
		return emDash
	}

	entry, ok := findAgentByPane(rtData, paneID, server)
	if !ok {
		return emDash
	}

	idleVal, hasIdle := entry[idleSinceKey]
	if !hasIdle {
		return "active"
	}
	ts, ok := asInt64(idleVal)
	if !ok {
		return "active"
	}
	elapsed := time.Now().Unix() - ts
	if elapsed < 0 {
		elapsed = 0
	}
	return fmt.Sprintf("idle (%s)", FormatIdleDuration(elapsed))
}

// loadRuntimeForCache returns the parsed runtime map for wtRoot, using and
// populating the shared cache. Returns (data, true) on a successful read,
// (nil, false) when the file is missing or unreadable — the cache records
// nil in either case to avoid retries.
func loadRuntimeForCache(wtRoot string, cache map[string]interface{}) (map[string]interface{}, bool) {
	if cached, present := cache[wtRoot]; present {
		if m, ok := cached.(map[string]interface{}); ok {
			return m, true
		}
		return nil, false
	}

	rtPath := filepath.Join(wtRoot, ".fab-runtime.yaml")
	loaded, err := LoadRuntimeFile(rtPath)
	if err != nil {
		cache[wtRoot] = nil
		return nil, false
	}
	cache[wtRoot] = loaded
	return loaded, true
}

// LoadRuntimeFile reads and parses .fab-runtime.yaml.
// Returns an error if the file doesn't exist.
func LoadRuntimeFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
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

// FormatIdleDuration formats elapsed seconds into a human-readable duration.
// Uses floor division: <60s -> Ns, 60s-3599s -> Nm, >=3600s -> Nh.
func FormatIdleDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%dh", seconds/3600)
}

// asInt64 coerces a YAML-decoded numeric value to int64. Returns ok=false
// when the value is missing or of an unexpected type.
func asInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}
