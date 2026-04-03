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
// and verifying the pane ID appears in the output.
func ValidatePane(paneID string) error {
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id}").Output()
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

// GetPanePID returns the shell PID of a tmux pane.
func GetPanePID(paneID string) (int, error) {
	out, err := exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_pid}").Output()
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
func ResolvePaneContext(paneID string) (*PaneContext, error) {
	// Get pane CWD
	out, err := exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_current_path}").Output()
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

	// Set worktree display (always, regardless of fab presence)
	ctx.WorktreeDisplay = filepath.Base(wtRoot) + "/"

	// Check for fab/ directory
	fabDir := filepath.Join(wtRoot, "fab")
	if _, err := os.Stat(fabDir); os.IsNotExist(err) {
		return ctx, nil
	}

	// Read .fab-status.yaml symlink for the active change
	_, folderName := ReadFabCurrent(wtRoot)
	if folderName != "" {
		ctx.Change = &folderName

		// Read stage from .status.yaml
		statusPath := filepath.Join(fabDir, "changes", folderName, ".status.yaml")
		if statusFile, err := sf.Load(statusPath); err == nil {
			stage, _ := status.DisplayStage(statusFile)
			ctx.Stage = &stage
		}

		// Resolve agent state
		state, idleDur := ResolveAgentState(wtRoot, folderName)
		if state != "" {
			ctx.AgentState = &state
		}
		if idleDur != "" {
			ctx.AgentIdleDuration = &idleDur
		}
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

// ResolveAgentState determines the agent state and idle duration for a change.
// Returns (state, idleDuration). state is "active", "idle", or "unknown".
// idleDuration is non-empty only when state is "idle".
func ResolveAgentState(wtRoot, folderName string) (string, string) {
	if folderName == "" {
		return "", ""
	}

	rtPath := filepath.Join(wtRoot, ".fab-runtime.yaml")
	rtData, err := LoadRuntimeFile(rtPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "unknown", ""
		}
		return "unknown", ""
	}

	folderEntry, ok := rtData[folderName].(map[string]interface{})
	if !ok {
		return "active", ""
	}

	agentBlock, ok := folderEntry["agent"].(map[string]interface{})
	if !ok {
		return "active", ""
	}

	idleSince, ok := agentBlock["idle_since"]
	if !ok {
		return "active", ""
	}

	var ts int64
	switch v := idleSince.(type) {
	case int:
		ts = int64(v)
	case int64:
		ts = v
	case float64:
		ts = int64(v)
	default:
		return "active", ""
	}

	elapsed := time.Now().Unix() - ts
	if elapsed < 0 {
		elapsed = 0
	}

	return "idle", FormatIdleDuration(elapsed)
}

// ResolveAgentStateWithCache is like ResolveAgentState but uses a per-worktree
// cache to avoid re-reading .fab-runtime.yaml for multiple panes in the same worktree.
// Returns the combined display string used by pane map (e.g., "active", "idle (2m)", "?", em dash).
func ResolveAgentStateWithCache(wtRoot, folderName string, cache map[string]interface{}) string {
	if folderName == "" {
		return "\u2014" // em dash for no change
	}

	rtPath := filepath.Join(wtRoot, ".fab-runtime.yaml")

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
		loaded, err := LoadRuntimeFile(rtPath)
		if err != nil {
			if os.IsNotExist(err) {
				cache[wtRoot] = nil
				fileMissing = true
			} else {
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

	return fmt.Sprintf("idle (%s)", FormatIdleDuration(elapsed))
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
