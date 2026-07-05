package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyncHooks_RegistersNoSessionHooks verifies the divestment (ioku): with
// an empty defaultHookMappings, a fresh sync registers ZERO session-scoped
// hook entries (fab is a pure consumer of the `@rk_agent_state` pane-option
// convention). `syncHooks` is retained but no longer migrates or registers
// anything, so with nothing to add it reports OK.
func TestSyncHooks_RegistersNoSessionHooks(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "OK") {
		t.Errorf("expected OK message (nothing to register after divestment), got: %s", msg)
	}

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	for _, event := range []string{"SessionStart", "Stop", "UserPromptSubmit", "PostToolUse"} {
		if len(hooks[event]) != 0 {
			t.Errorf("expected 0 %s hooks (all fab hooks divested), got %d", event, len(hooks[event]))
		}
	}
}

func TestSyncHooks_Idempotent(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	// Run twice — both are OK now that nothing is registered.
	syncHooks(settingsPath)
	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("second syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "OK") {
		t.Errorf("expected OK on second run, got: %s", msg)
	}
}

// TestSyncHooks_DoesNotReMintAbsolutePathScripts confirms the re-minting hazard
// is closed (ioku cycle 2): the legacy on-*.sh rewrite rows were dropped from
// oldScriptToSubcommand, so an old-format absolute-path on-session-start.sh
// entry is left untouched rather than rewritten to `fab hook session-start` —
// which would re-mint one of the entries the 2.13.6-to-2.14.0 migration deletes.
func TestSyncHooks_DoesNotReMintAbsolutePathScripts(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	legacy := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-session-start.sh`
	// Create settings with old-format absolute path
	oldSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": legacy,
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

	if strings.Contains(msg, "migrated") {
		t.Errorf("must NOT migrate legacy scripts, got: %s", msg)
	}

	// Verify the legacy entry was left untouched (no re-mint).
	data, _ = os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)
	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if cmd := hooks["SessionStart"][0].Hooks[0].Command; cmd != legacy {
		t.Errorf("expected legacy command %q left untouched, got: %s", legacy, cmd)
	}
}

// TestSyncHooks_DoesNotReMintRelativePathScripts is the relative-path
// counterpart: an old-format relative-path on-session-start.sh entry is
// likewise left untouched rather than rewritten to the inline `fab hook` form.
func TestSyncHooks_DoesNotReMintRelativePathScripts(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	legacy := "bash fab/.kit/hooks/on-session-start.sh"
	// Create settings with old-format relative path
	oldSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": legacy,
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

	if strings.Contains(msg, "migrated") {
		t.Errorf("must NOT migrate legacy scripts, got: %s", msg)
	}

	data, _ = os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)
	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if cmd := hooks["SessionStart"][0].Hooks[0].Command; cmd != legacy {
		t.Errorf("expected legacy command %q left untouched, got: %s", legacy, cmd)
	}
}

// TestSyncHooks_NoArtifactWriteRegistration confirms no PostToolUse entry is
// registered (artifact-write was removed in y022). After the ioku divestment
// the whole default mapping is empty, so a fresh sync reports OK and writes no
// PostToolUse entry.
func TestSyncHooks_NoArtifactWriteRegistration(t *testing.T) {
	settingsDir := t.TempDir()
	settingsPath := filepath.Join(settingsDir, "settings.local.json")

	msg, err := syncHooks(settingsPath)
	if err != nil {
		t.Fatalf("syncHooks failed: %v", err)
	}

	if !strings.Contains(msg, "OK") {
		t.Errorf("expected OK message, got: %s", msg)
	}

	// Verify no PostToolUse entries are registered (artifact-write removed).
	data, _ := os.ReadFile(settingsPath)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string][]hookEntry
	json.Unmarshal(settings["hooks"], &hooks)

	if len(hooks["PostToolUse"]) != 0 {
		t.Errorf("expected 0 PostToolUse hooks (artifact-write removed), got %d", len(hooks["PostToolUse"]))
	}
}
