package hooklib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSync_FreshSettings(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "created" {
		t.Errorf("Status = %q, want %q", result.Status, "created")
	}

	// Verify the file was created with hook entries
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	var hooks map[string][]hookEntry
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("failed to parse hooks: %v", err)
	}

	// Should have SessionStart, Stop, UserPromptSubmit, PostToolUse
	if len(hooks["SessionStart"]) != 1 {
		t.Errorf("SessionStart entries = %d, want 1", len(hooks["SessionStart"]))
	}
	if len(hooks["Stop"]) != 1 {
		t.Errorf("Stop entries = %d, want 1", len(hooks["Stop"]))
	}
	if len(hooks["UserPromptSubmit"]) != 1 {
		t.Errorf("UserPromptSubmit entries = %d, want 1", len(hooks["UserPromptSubmit"]))
	}
	if len(hooks["PostToolUse"]) != 2 {
		t.Errorf("PostToolUse entries = %d, want 2 (Write + Edit)", len(hooks["PostToolUse"]))
	}
}

func TestSync_Deduplication(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	// First sync
	_, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Second sync should be OK (no changes)
	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q", result.Status, "ok")
	}
}

func TestSync_PreserveNonHookSettings(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Write initial settings with non-hook data
	initial := `{"model":"claude-opus-4-6","permissions":{"allow":["Read"]}}`
	os.WriteFile(settingsPath, []byte(initial), 0o644)

	_, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify non-hook settings preserved
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var model string
	json.Unmarshal(settings["model"], &model)
	if model != "claude-opus-4-6" {
		t.Errorf("model = %q, want %q", model, "claude-opus-4-6")
	}

	if settings["permissions"] == nil {
		t.Error("permissions should be preserved")
	}
}

func TestSync_EmptySettings(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")
	os.WriteFile(settingsPath, []byte("{}"), 0o644)

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "created" {
		t.Errorf("Status = %q, want %q", result.Status, "created")
	}
}

func TestSync_UsesInlineCommand(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	_, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := readHooks(t, settingsPath)
	cmd := hooks["Stop"][0].Hooks[0].Command
	want := "fab hook stop"
	if cmd != want {
		t.Errorf("command = %q, want %q", cmd, want)
	}
}

func TestSync_MigratesOldAbsolutePaths(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Write settings with old-format (absolute path) hooks
	oldSettings := `{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR\"/fab/.kit/hooks/on-stop.sh"}]}],
    "SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR\"/fab/.kit/hooks/on-session-start.sh"}]}]
  }
}`
	os.WriteFile(settingsPath, []byte(oldSettings), 0o644)

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "updated" {
		t.Errorf("Status = %q, want %q", result.Status, "updated")
	}
	if !strings.Contains(result.Message, "migrated") {
		t.Errorf("Message should mention migration, got %q", result.Message)
	}

	// Verify commands were migrated to inline format
	hooks := readHooks(t, settingsPath)
	if hooks["Stop"][0].Hooks[0].Command != "fab hook stop" {
		t.Errorf("Stop not migrated, got: %q", hooks["Stop"][0].Hooks[0].Command)
	}
	if hooks["SessionStart"][0].Hooks[0].Command != "fab hook session-start" {
		t.Errorf("SessionStart not migrated, got: %q", hooks["SessionStart"][0].Hooks[0].Command)
	}

	// No duplicate entries — migration should update in place, not add new ones
	if len(hooks["Stop"]) != 1 {
		t.Errorf("Stop entries = %d, want 1 (no duplicates after migration)", len(hooks["Stop"]))
	}
}

func TestSync_MigratesOldRelativePaths(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Write settings with old-format (relative path) hooks
	oldSettings := `{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "bash fab/.kit/hooks/on-stop.sh"}]}]
  }
}`
	os.WriteFile(settingsPath, []byte(oldSettings), 0o644)

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "updated" {
		t.Errorf("Status = %q, want %q", result.Status, "updated")
	}

	hooks := readHooks(t, settingsPath)
	if hooks["Stop"][0].Hooks[0].Command != "fab hook stop" {
		t.Errorf("Stop not migrated from relative path, got: %q", hooks["Stop"][0].Hooks[0].Command)
	}
}

func TestSync_MigratePreservesNonFabHooks(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Mix of fab hooks (old format) and non-fab hooks
	oldSettings := `{
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR\"/fab/.kit/hooks/on-stop.sh"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": "echo custom stop hook"}]}
    ]
  }
}`
	os.WriteFile(settingsPath, []byte(oldSettings), 0o644)

	_, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := readHooks(t, settingsPath)
	if len(hooks["Stop"]) != 2 {
		t.Errorf("Stop entries = %d, want 2 (migrated fab + custom)", len(hooks["Stop"]))
	}

	// Find the custom hook — should be untouched
	found := false
	for _, entry := range hooks["Stop"] {
		for _, h := range entry.Hooks {
			if h.Command == "echo custom stop hook" {
				found = true
			}
		}
	}
	if !found {
		t.Error("custom non-fab hook was lost during migration")
	}
}

// readHooks is a test helper that reads and parses hooks from the settings file.
func readHooks(t *testing.T, settingsPath string) map[string][]hookEntry {
	t.Helper()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}
	var hooks map[string][]hookEntry
	if err := json.Unmarshal(settings["hooks"], &hooks); err != nil {
		t.Fatalf("failed to parse hooks: %v", err)
	}
	return hooks
}
