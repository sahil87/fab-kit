package agentmodels

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClaudeAliasesOnlyAndMissingFilesSilent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	rows, defaultID, err := discoverClaude(defaultEnvironment())
	if err != nil || defaultID != "" {
		t.Fatalf("defaultID=%q err=%v", defaultID, err)
	}
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.ID)
		if row.Source != SourceAlias || !reflect.DeepEqual(row.Efforts, claudeEfforts) {
			t.Errorf("row = %+v", row)
		}
	}
	if !reflect.DeepEqual(ids, claudeAliases) {
		t.Fatalf("aliases = %v", ids)
	}
}

func TestClaudeOptionsLabelsAndDefault(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, filepath.Join(home, ".claude", "settings.json"), `{
  "model": "custom-default",
  "modelPicker": {"options": [{"model": "custom", "label": "Custom Model"}]}
}`)
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	rows, warnings := discoverWith(nil, "claude", defaultEnvironment())
	if len(warnings) != 0 {
		t.Fatal(warnings)
	}
	custom := findRow(rows, "custom")
	if custom == nil || custom.Label != "Custom Model" || custom.Source != SourceTool {
		t.Fatalf("custom row = %+v", custom)
	}
	if row := findRow(rows, "custom-default"); row == nil || !row.Default || row.Source != SourceTool {
		t.Fatalf("appended default = %+v", row)
	}
	for _, id := range claudeAliases {
		if row := findRow(rows, id); row == nil || !reflect.DeepEqual(row.Efforts, claudeEfforts) {
			t.Errorf("alias %s = %+v", id, row)
		}
	}
}

func TestClaudeReplaceBuiltInOptions(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "settings.json"), `{
  "modelPicker": {"replaceBuiltInOptions": true, "options": [{"model": "only-custom"}]}
}`)
	t.Setenv("CLAUDE_CONFIG_DIR", directory)
	rows, _, err := discoverClaude(defaultEnvironment())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != "only-custom" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestClaudeLocalOverridesBaseAndConfigDir(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "settings.json"), `{
  "model": "base-default",
  "modelPicker": {"options": [{"model": "base-option"}]}
}`)
	writeTestFile(t, filepath.Join(directory, "settings.local.json"), `{
  "model": "local-default",
  "modelPicker": {"options": [{"model": "local-option"}]}
}`)
	t.Setenv("CLAUDE_CONFIG_DIR", directory)
	rows, defaultID, err := discoverClaude(defaultEnvironment())
	if err != nil || defaultID != "local-default" {
		t.Fatalf("defaultID=%q err=%v", defaultID, err)
	}
	if findRow(rows, "local-option") == nil || findRow(rows, "base-option") != nil {
		t.Fatalf("local modelPicker did not replace base: %+v", rows)
	}
}

func TestClaudeMalformedSettingsWarnsAndKeepsAliases(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "settings.json"), `{broken`)
	writeTestFile(t, filepath.Join(directory, "settings.local.json"), `{also-broken`)
	t.Setenv("CLAUDE_CONFIG_DIR", directory)
	rows, warnings := discoverWith(nil, "claude", defaultEnvironment())
	if len(warnings) != 1 || !strings.Contains(warnings[0], "settings.json") || strings.Contains(warnings[0], "\n") {
		t.Fatalf("warnings = %v", warnings)
	}
	for _, id := range claudeAliases {
		if findRow(rows, id) == nil {
			t.Errorf("alias %s missing", id)
		}
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
