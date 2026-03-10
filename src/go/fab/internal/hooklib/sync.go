package hooklib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HookMapping defines a mapping from a hook script to a Claude Code event.
type HookMapping struct {
	Script  string
	Event   string
	Matcher string
}

// DefaultMappings is the mapping table from hook scripts to Claude Code events.
var DefaultMappings = []HookMapping{
	{Script: "on-session-start.sh", Event: "SessionStart", Matcher: ""},
	{Script: "on-stop.sh", Event: "Stop", Matcher: ""},
	{Script: "on-user-prompt.sh", Event: "UserPromptSubmit", Matcher: ""},
	{Script: "on-artifact-write.sh", Event: "PostToolUse", Matcher: "Write"},
	{Script: "on-artifact-write.sh", Event: "PostToolUse", Matcher: "Edit"},
}

// hookEntry represents a single hook entry in settings.local.json.
type hookEntry struct {
	Matcher string     `json:"matcher"`
	Hooks   []hookSpec `json:"hooks"`
}

// hookSpec represents a hook action specification.
type hookSpec struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// SyncResult holds the outcome of a hook sync operation.
type SyncResult struct {
	Status  string // "created", "updated", "ok"
	Message string
}

// Sync discovers hook scripts in hooksDir, maps them to Claude Code events,
// and merges entries into settingsPath. Idempotent.
func Sync(hooksDir, settingsPath string) (*SyncResult, error) {
	// Discover existing hook scripts
	existingScripts := make(map[string]bool)
	entries, err := os.ReadDir(hooksDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading hooks dir: %w", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			existingScripts[e.Name()] = true
		}
	}

	// Build desired hook entries from mappings (only for scripts that exist)
	type desiredEntry struct {
		event   string
		matcher string
		command string
	}
	var desired []desiredEntry
	for _, m := range DefaultMappings {
		if !existingScripts[m.Script] {
			continue
		}
		cmd := "bash fab/.kit/hooks/" + m.Script
		desired = append(desired, desiredEntry{event: m.Event, matcher: m.Matcher, command: cmd})
	}

	// Ensure settings directory exists
	settingsDir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating settings dir: %w", err)
	}

	// Load or initialize settings
	settings := make(map[string]json.RawMessage)
	if data, err := os.ReadFile(settingsPath); err == nil {
		trimmed := bytes.TrimSpace(data)
		if len(trimmed) > 0 {
			if err := json.Unmarshal(trimmed, &settings); err != nil {
				return nil, fmt.Errorf("parsing settings: %w", err)
			}
		}
	}

	// Parse existing hooks section
	existingHooks := make(map[string][]hookEntry)
	if raw, ok := settings["hooks"]; ok {
		if err := json.Unmarshal(raw, &existingHooks); err != nil {
			// If hooks section is malformed, start fresh
			existingHooks = make(map[string][]hookEntry)
		}
	}

	// Count existing entries for change detection
	existingCount := 0
	for _, entries := range existingHooks {
		existingCount += len(entries)
	}

	// Merge desired entries (deduplicate by matcher + command pair)
	added := 0
	for _, d := range desired {
		eventEntries := existingHooks[d.event]
		if !hasDuplicate(eventEntries, d.matcher, d.command) {
			eventEntries = append(eventEntries, hookEntry{
				Matcher: d.matcher,
				Hooks: []hookSpec{
					{Type: "command", Command: d.command},
				},
			})
			existingHooks[d.event] = eventEntries
			added++
		}
	}

	// Serialize hooks back into settings
	hooksJSON, err := json.Marshal(existingHooks)
	if err != nil {
		return nil, fmt.Errorf("marshaling hooks: %w", err)
	}
	settings["hooks"] = hooksJSON

	// Write settings file
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("writing settings: %w", err)
	}

	// Determine result status
	newCount := 0
	for _, entries := range existingHooks {
		newCount += len(entries)
	}

	if added == 0 {
		return &SyncResult{
			Status:  "ok",
			Message: ".claude/settings.local.json hooks: OK",
		}, nil
	}

	if existingCount == 0 {
		return &SyncResult{
			Status:  "created",
			Message: fmt.Sprintf("Created: .claude/settings.local.json hooks (%d hook entries)", newCount),
		}, nil
	}

	return &SyncResult{
		Status:  "updated",
		Message: fmt.Sprintf("Updated: .claude/settings.local.json hooks (added %d hook entries)", added),
	}, nil
}

// hasDuplicate checks if an entry with the same matcher and command already exists.
func hasDuplicate(entries []hookEntry, matcher, command string) bool {
	for _, e := range entries {
		if e.Matcher == matcher {
			for _, h := range e.Hooks {
				if h.Command == command {
					return true
				}
			}
		}
	}
	return false
}
