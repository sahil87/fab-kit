package hooklib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupHooksDir(t *testing.T, scripts ...string) string {
	t.Helper()
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	os.MkdirAll(hooksDir, 0o755)
	for _, s := range scripts {
		os.WriteFile(filepath.Join(hooksDir, s), []byte("#!/bin/bash\nexit 0\n"), 0o755)
	}
	return hooksDir
}

func TestSync_FreshSettings(t *testing.T) {
	hooksDir := setupHooksDir(t,
		"on-session-start.sh",
		"on-stop.sh",
		"on-user-prompt.sh",
		"on-artifact-write.sh",
	)

	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	result, err := Sync(hooksDir, settingsPath)
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
	hooksDir := setupHooksDir(t,
		"on-session-start.sh",
		"on-stop.sh",
		"on-user-prompt.sh",
		"on-artifact-write.sh",
	)

	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	// First sync
	_, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Second sync should be OK (no changes)
	result, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q", result.Status, "ok")
	}
}

func TestSync_MergeNewEntries(t *testing.T) {
	hooksDir := setupHooksDir(t,
		"on-session-start.sh",
		"on-stop.sh",
	)

	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	// First sync with only two scripts
	_, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Add more scripts
	os.WriteFile(filepath.Join(hooksDir, "on-user-prompt.sh"), []byte("#!/bin/bash\nexit 0\n"), 0o755)
	os.WriteFile(filepath.Join(hooksDir, "on-artifact-write.sh"), []byte("#!/bin/bash\nexit 0\n"), 0o755)

	// Second sync should add new entries
	result, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if result.Status != "updated" {
		t.Errorf("Status = %q, want %q", result.Status, "updated")
	}
}

func TestSync_MissingScript(t *testing.T) {
	// Only create session-start and stop, NOT user-prompt or artifact-write
	hooksDir := setupHooksDir(t,
		"on-session-start.sh",
		"on-stop.sh",
	)

	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	result, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "created" {
		t.Errorf("Status = %q, want %q", result.Status, "created")
	}

	// Verify only 2 entries (no UserPromptSubmit, no PostToolUse)
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)
	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if _, ok := hooks["UserPromptSubmit"]; ok {
		t.Error("UserPromptSubmit should not be present (script missing)")
	}
	if _, ok := hooks["PostToolUse"]; ok {
		t.Error("PostToolUse should not be present (script missing)")
	}
}

func TestSync_PreserveNonHookSettings(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Write initial settings with non-hook data
	initial := `{"model":"claude-opus-4-6","permissions":{"allow":["Read"]}}`
	os.WriteFile(settingsPath, []byte(initial), 0o644)

	hooksDir := setupHooksDir(t, "on-session-start.sh")

	_, err := Sync(hooksDir, settingsPath)
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

	hooksDir := setupHooksDir(t, "on-session-start.sh")

	result, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "created" {
		t.Errorf("Status = %q, want %q", result.Status, "created")
	}
}

func TestSync_EmptyHooksDir(t *testing.T) {
	hooksDir := setupHooksDir(t) // no scripts
	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	result, err := Sync(hooksDir, settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No entries added, but file should be written
	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q", result.Status, "ok")
	}
}
