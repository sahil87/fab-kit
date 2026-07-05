package hooklib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSync_FreshSettings verifies the divestment (ioku): with an empty
// DefaultMappings, a fresh sync registers ZERO session-scoped hook entries.
// The three session hooks (SessionStart/Stop/UserPromptSubmit) that wrote
// `.fab-runtime.yaml` `_agents` state are gone (fab is a pure consumer of the
// `@rk_agent_state` pane-option convention); the earlier artifact-write
// PostToolUse rows were already removed. `Sync` itself is retained but no
// longer migrates or registers anything, so with nothing to add it reports the
// OK status.
func TestSync_FreshSettings(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), ".claude", "settings.local.json")

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q (nothing to register after divestment)", result.Status, "ok")
	}

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

	// No session-scoped or PostToolUse registrations remain.
	for _, event := range []string{"SessionStart", "Stop", "UserPromptSubmit", "PostToolUse"} {
		if len(hooks[event]) != 0 {
			t.Errorf("%s entries = %d, want 0 (all fab hooks divested)", event, len(hooks[event]))
		}
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

	// With an empty DefaultMappings there is nothing to add or migrate, so the
	// status is OK (not "created").
	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q", result.Status, "ok")
	}
}

// TestSync_LeavesLegacyScriptUntouched confirms the re-minting hazard is closed
// (ioku cycle 2): the legacy on-*.sh rewrite rows were dropped from
// `oldScriptToSubcommand`, so `Sync` no longer rewrites an old-style on-stop.sh
// entry into `fab hook stop` — that would re-mint exactly one of the three
// entries the 2.13.6-to-2.14.0 migration deletes. `Sync` has no removal path,
// so it leaves the legacy entry as-is; the migration file is what removes it.
func TestSync_LeavesLegacyScriptUntouched(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")
	legacy := "bash fab/.kit/hooks/on-stop.sh"
	oldSettings := `{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "` + legacy + `"}]}]
  }
}`
	os.WriteFile(settingsPath, []byte(oldSettings), 0o644)

	result, err := Sync(settingsPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nothing to add or migrate — OK status, no re-minting.
	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q (no migration after re-minting rows dropped)", result.Status, "ok")
	}

	hooks := readHooks(t, settingsPath)
	if got := hooks["Stop"][0].Hooks[0].Command; got != legacy {
		t.Errorf("command = %q, want %q left untouched (must NOT re-mint fab hook stop)", got, legacy)
	}
}

// TestSync_DoesNotReMintAbsolutePathScripts is the absolute-path counterpart:
// an old-format on-*.sh entry using the "$CLAUDE_PROJECT_DIR" absolute form is
// likewise left untouched rather than rewritten to the inline `fab hook` form.
func TestSync_DoesNotReMintAbsolutePathScripts(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")
	stopLegacy := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-stop.sh`
	startLegacy := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-session-start.sh`
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

	if result.Status != "ok" {
		t.Errorf("Status = %q, want %q (no migration)", result.Status, "ok")
	}
	if strings.Contains(result.Message, "migrated") {
		t.Errorf("Message must NOT mention migration, got %q", result.Message)
	}

	hooks := readHooks(t, settingsPath)
	if got := hooks["Stop"][0].Hooks[0].Command; got != stopLegacy {
		t.Errorf("Stop = %q, want %q untouched (no re-mint)", got, stopLegacy)
	}
	if got := hooks["SessionStart"][0].Hooks[0].Command; got != startLegacy {
		t.Errorf("SessionStart = %q, want %q untouched (no re-mint)", got, startLegacy)
	}
}

// TestSync_PreservesNonFabHooks confirms a custom (non-fab) hook is preserved
// alongside an untouched legacy fab entry — `Sync` neither migrates the legacy
// entry nor disturbs unrelated hooks.
func TestSync_PreservesNonFabHooks(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")
	legacy := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-stop.sh`

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
		t.Errorf("Stop entries = %d, want 2 (legacy left as-is + custom)", len(hooks["Stop"]))
	}

	// Both the untouched legacy fab entry and the custom hook survive.
	foundLegacy, foundCustom := false, false
	for _, entry := range hooks["Stop"] {
		for _, h := range entry.Hooks {
			switch h.Command {
			case legacy:
				foundLegacy = true
			case "echo custom stop hook":
				foundCustom = true
			}
		}
	}
	if !foundLegacy {
		t.Error("legacy fab entry should be left untouched (not re-minted, not dropped)")
	}
	if !foundCustom {
		t.Error("custom non-fab hook was lost")
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
