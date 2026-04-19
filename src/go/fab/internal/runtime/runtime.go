// Package runtime manages .fab-runtime.yaml — the ephemeral per-worktree
// state file that tracks Claude Code agents and drives fab pane map / fab
// pane send behavior.
//
// Schema (see docs/memory/fab-workflow/runtime-agents.md for the canonical reference):
//
//	_agents:
//	  "<session_id>":
//	    idle_since: <unix-ts>       # present when agent is idle
//	    change: "<folder-name>"      # optional — absent in discussion mode
//	    pid: <int>                   # optional — Claude's PID for GC liveness
//	    tmux_server: "<label>"       # optional — basename of $TMUX socket
//	    tmux_pane: "%15"             # optional — pane ID from $TMUX_PANE
//	    transcript_path: "..."       # optional — from hook payload
//	last_run_gc: <unix-ts>           # throttles GC sweeps to every ~3 min
package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Top-level field and schema key constants. Centralizing these avoids
// scattering magic strings across load/save/GC paths.
const (
	agentsKey     = "_agents"
	lastRunGCKey  = "last_run_gc"
	idleSinceKey  = "idle_since"
	changeKey     = "change"
	pidKey        = "pid"
	tmuxServerKey = "tmux_server"
	tmuxPaneKey   = "tmux_pane"
	transcriptKey = "transcript_path"
)

// AgentEntry is the in-memory representation of a single `_agents[session_id]`
// record. All fields except IdleSince are optional and are omitted from the
// serialized form when empty/nil.
type AgentEntry struct {
	// Change is the change folder name the agent is working on, or empty for
	// discussion-mode agents (no active change).
	Change string `yaml:"change,omitempty"`

	// IdleSince is the Unix timestamp at which the agent became idle. Nil
	// means the agent is active (or the field has been cleared by a
	// user-prompt hook).
	IdleSince *int64 `yaml:"idle_since,omitempty"`

	// PID is Claude's process ID, resolved via proc.ClaudePID(). Nil if
	// resolution failed. Used by GC for kill(pid, 0) liveness checks.
	PID *int `yaml:"pid,omitempty"`

	// TmuxServer is the basename of the $TMUX socket path (the first
	// comma-separated component). Empty when the agent is running outside
	// tmux.
	TmuxServer string `yaml:"tmux_server,omitempty"`

	// TmuxPane is the pane ID from $TMUX_PANE (e.g. "%15"). Empty when the
	// agent is running outside tmux.
	TmuxPane string `yaml:"tmux_pane,omitempty"`

	// TranscriptPath is the on-disk path to Claude's session transcript, as
	// provided in the hook stdin payload. Optional — purely for correlation.
	TranscriptPath string `yaml:"transcript_path,omitempty"`
}

// FilePath returns the absolute path to .fab-runtime.yaml at the repo root.
func FilePath(fabRoot string) string {
	repoRoot := filepath.Dir(fabRoot)
	return filepath.Join(repoRoot, ".fab-runtime.yaml")
}

// LoadFile reads and parses .fab-runtime.yaml, returning an empty map if the
// file doesn't exist.
func LoadFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
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

// SaveFile marshals the map and writes it atomically via temp+rename.
func SaveFile(path string, m map[string]interface{}) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".fab-runtime-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	// Ensure temporary file is cleaned up on error.
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	success = true
	return nil
}

// agentEntryToMap converts an AgentEntry into the map[string]interface{}
// form used in the on-disk YAML. Fields are omitted when empty/nil so the
// serialized record only contains the properties that are actually known.
func agentEntryToMap(e AgentEntry) map[string]interface{} {
	m := make(map[string]interface{})
	if e.Change != "" {
		m[changeKey] = e.Change
	}
	if e.IdleSince != nil {
		m[idleSinceKey] = *e.IdleSince
	}
	if e.PID != nil {
		m[pidKey] = *e.PID
	}
	if e.TmuxServer != "" {
		m[tmuxServerKey] = e.TmuxServer
	}
	if e.TmuxPane != "" {
		m[tmuxPaneKey] = e.TmuxPane
	}
	if e.TranscriptPath != "" {
		m[transcriptKey] = e.TranscriptPath
	}
	return m
}

// agentsMap returns the top-level _agents map from a loaded runtime file,
// creating it in place when absent. The returned map is shared with the
// parent — mutations are visible after subsequent SaveFile calls.
func agentsMap(m map[string]interface{}) map[string]interface{} {
	existing, ok := m[agentsKey].(map[string]interface{})
	if !ok || existing == nil {
		existing = make(map[string]interface{})
		m[agentsKey] = existing
	}
	return existing
}

// WriteAgent writes (or overwrites) the _agents[sessionID] entry with the
// provided AgentEntry. Any pre-existing entry is replaced in full — callers
// that want field-level preservation should use ClearAgentIdle instead.
// The runtime file is created if it does not exist.
func WriteAgent(fabRoot, sessionID string, entry AgentEntry) error {
	if sessionID == "" {
		return fmt.Errorf("WriteAgent: empty sessionID")
	}

	rtPath := FilePath(fabRoot)
	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	agents := agentsMap(m)
	agents[sessionID] = agentEntryToMap(entry)
	m[agentsKey] = agents

	return SaveFile(rtPath, m)
}

// ClearAgent deletes the _agents[sessionID] entry entirely. No-op when the
// runtime file is absent or the session ID is not present.
func ClearAgent(fabRoot, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	rtPath := FilePath(fabRoot)
	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		return nil
	}

	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	agents, ok := m[agentsKey].(map[string]interface{})
	if !ok {
		return nil
	}

	if _, present := agents[sessionID]; !present {
		return nil
	}

	delete(agents, sessionID)
	m[agentsKey] = agents

	return SaveFile(rtPath, m)
}

// ClearAgentIdle removes only the `idle_since` key from _agents[sessionID],
// preserving all other fields (change, pid, tmux_server, tmux_pane,
// transcript_path). Used by the user-prompt hook to mark the agent active
// again without losing pane-map correlation properties.
//
// No-op when the runtime file or the entry is absent.
func ClearAgentIdle(fabRoot, sessionID string) error {
	if sessionID == "" {
		return nil
	}

	rtPath := FilePath(fabRoot)
	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		return nil
	}

	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	agents, ok := m[agentsKey].(map[string]interface{})
	if !ok {
		return nil
	}

	entry, ok := agents[sessionID].(map[string]interface{})
	if !ok {
		return nil
	}

	if _, present := entry[idleSinceKey]; !present {
		return nil
	}

	delete(entry, idleSinceKey)
	agents[sessionID] = entry
	m[agentsKey] = agents

	return SaveFile(rtPath, m)
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

// GCIfDue sweeps `_agents` for entries whose stored `pid` no longer
// corresponds to a running process, throttled by the top-level `last_run_gc`
// field so sweeps run at most once per interval.
//
// Behavior:
//  1. If the runtime file is absent, return nil (no-op).
//  2. If `now - last_run_gc < interval`, return nil (throttled).
//  3. Otherwise, for each agent entry with a non-nil `pid`, issue
//     kill(pid, 0). If the call returns ESRCH, delete the entry.
//     Entries without a `pid` field are preserved regardless.
//  4. Update `last_run_gc = now()` and save atomically.
//
// The caller is expected to pass a fixed interval (180s is the canonical
// value used by hook handlers).
func GCIfDue(fabRoot string, interval time.Duration) error {
	rtPath := FilePath(fabRoot)
	if _, err := os.Stat(rtPath); os.IsNotExist(err) {
		return nil
	}

	m, err := LoadFile(rtPath)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if lastRun, ok := asInt64(m[lastRunGCKey]); ok {
		if now-lastRun < int64(interval.Seconds()) {
			return nil
		}
	}

	if agents, ok := m[agentsKey].(map[string]interface{}); ok {
		for sessionID, raw := range agents {
			entry, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			pidVal, hasPid := entry[pidKey]
			if !hasPid {
				continue // preserve pid-less entries
			}
			pid64, ok := asInt64(pidVal)
			if !ok {
				continue
			}
			if !pidAlive(int(pid64)) {
				delete(agents, sessionID)
			}
		}
		m[agentsKey] = agents
	}

	m[lastRunGCKey] = now
	return SaveFile(rtPath, m)
}

// pidAlive returns true when kill(pid, 0) succeeds or returns EPERM (the
// process exists but we lack permission to signal it). It returns false for
// any other error, treating ESRCH ("no such process") and any unexpected
// errno conservatively as "not alive". This is the POSIX-standard liveness
// probe and performs no subprocess work.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	// EPERM means the process exists but we can't signal it — still alive.
	if err == syscall.EPERM {
		return true
	}
	// ESRCH and anything else is treated as dead.
	return false
}
