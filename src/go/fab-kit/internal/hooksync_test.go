package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncHooks_CreateNew(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "Created:") {
		t.Errorf("expected Created message, got: %s", msg)
	}

	// Verify settings file has hooks
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if len(hooks["SessionStart"]) != 1 {
		t.Errorf("expected 1 SessionStart hook, got %d", len(hooks["SessionStart"]))
	}
	if len(hooks["Stop"]) != 1 {
		t.Errorf("expected 1 Stop hook, got %d", len(hooks["Stop"]))
	}
	if len(hooks["UserPromptSubmit"]) != 1 {
		t.Errorf("expected 1 UserPromptSubmit hook, got %d", len(hooks["UserPromptSubmit"]))
	}
	if len(hooks["PostToolUse"]) != 2 {
		t.Errorf("expected 2 PostToolUse hooks (Write + Edit), got %d", len(hooks["PostToolUse"]))
	}

	// Verify inline command format
	cmd := hooks["Stop"][0].Hooks[0].Command
	if cmd != "fab hook stop" {
		t.Errorf("expected inline command, got: %s", cmd)
	}
}

func TestSyncHooks_Idempotent(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Run twice
	syncHooks(settingsPath)
	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("second syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "OK") {
		t.Errorf("expected OK on second run, got: %s", msg)
	}

	// Verify no duplicates
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if len(hooks["SessionStart"]) != 1 {
		t.Errorf("expected 1 SessionStart hook (no duplicates), got %d", len(hooks["SessionStart"]))
	}
}

func TestSyncHooks_MigratesOldAbsolutePaths(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Create settings with old-format absolute path
	oldSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-session-start.sh`,
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(oldSettings, "", "  ")
	os.WriteFile(settingsPath, data, 0644)

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "migrated") {
		t.Errorf("expected migration message, got: %s", msg)
	}

	// Verify path was migrated to inline command
	data, _ = os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)
	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	cmd := hooks["SessionStart"][0].Hooks[0].Command
	if cmd != "fab hook session-start" {
		t.Errorf("expected inline command, got: %s", cmd)
	}
}

func TestSyncHooks_MigratesOldRelativePaths(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Create settings with old-format relative path
	oldSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "bash fab/.kit/hooks/on-session-start.sh",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(oldSettings, "", "  ")
	os.WriteFile(settingsPath, data, 0644)

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "migrated") {
		t.Errorf("expected migration message, got: %s", msg)
	}
}

func TestSyncHooks_ArtifactWriteDoubleMapping(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "Created:") {
		t.Errorf("expected Created message, got: %s", msg)
	}

	// Verify both Write and Edit matchers exist
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if len(hooks["PostToolUse"]) != 2 {
		t.Errorf("expected 2 PostToolUse hooks (Write + Edit), got %d", len(hooks["PostToolUse"]))
	}
}
