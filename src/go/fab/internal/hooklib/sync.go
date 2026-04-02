package hooklib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HookMapping defines a mapping from a fab hook subcommand to a Claude Code event.
type HookMapping struct {
	Subcommand string // fab hook subcommand name (e.g., "session-start")
	Event      string
	Matcher    string
}

// DefaultMappings is the mapping table from fab hook subcommands to Claude Code events.
var DefaultMappings = []HookMapping{
	{Subcommand: "session-start", Event: "SessionStart", Matcher: ""},
	{Subcommand: "stop", Event: "Stop", Matcher: ""},
	{Subcommand: "user-prompt", Event: "UserPromptSubmit", Matcher: ""},
	{Subcommand: "artifact-write", Event: "PostToolUse", Matcher: "Write"},
	{Subcommand: "artifact-write", Event: "PostToolUse", Matcher: "Edit"},
}

// oldScriptToSubcommand maps old hook script names to new subcommand names for migration.
var oldScriptToSubcommand = map[string]string{
	"on-session-start.sh":  "session-start",
	"on-stop.sh":           "stop",
	"on-user-prompt.sh":    "user-prompt",
	"on-artifact-write.sh": "artifact-write",
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

// Sync registers inline `fab hook <subcommand>` commands in settingsPath. Idempotent.
func Sync(settingsPath string) (*SyncResult, error) {
	// Build desired hook entries from hardcoded mappings
	type desiredEntry struct {
		event   string
		matcher string
		command string
	}
	var desired []desiredEntry
	for _, m := range DefaultMappings {
		cmd := "fab hook " + m.Subcommand
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

	// Migrate old-style commands to inline fab hook commands
	migrated := migrateOldHookCommands(existingHooks)

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

	if added == 0 && migrated == 0 {
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

	var parts []string
	if added > 0 {
		parts = append(parts, fmt.Sprintf("added %d hook entries", added))
	}
	if migrated > 0 {
		parts = append(parts, fmt.Sprintf("migrated %d to inline commands", migrated))
	}
	return &SyncResult{
		Status:  "updated",
		Message: fmt.Sprintf("Updated: .claude/settings.local.json hooks (%s)", strings.Join(parts, ", ")),
	}, nil
}

// migrateOldHookCommands replaces old-style bash script commands with inline fab hook commands.
// Returns the number of commands migrated.
func migrateOldHookCommands(hooks map[string][]hookEntry) int {
	migrated := 0
	for event, eventEntries := range hooks {
		for i, entry := range eventEntries {
			for j, h := range entry.Hooks {
				// Match old-style: bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh
				// or: bash fab/.kit/hooks/on-*.sh
				for scriptName, subcommand := range oldScriptToSubcommand {
					oldAbsolute := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/` + scriptName
					oldRelative := "bash fab/.kit/hooks/" + scriptName
					if h.Command == oldAbsolute || h.Command == oldRelative {
						hooks[event][i].Hooks[j].Command = "fab hook " + subcommand
						migrated++
						break
					}
				}
			}
		}
	}
	return migrated
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
