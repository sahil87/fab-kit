package spawn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommand_WithSpawnCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`agent:
  spawn_command: "custom-claude --model opus"
`), 0o644)

	got := Command(configPath)
	if got != "custom-claude --model opus" {
		t.Errorf("Command() = %q, want %q", got, "custom-claude --model opus")
	}
}

func TestCommand_EmptySpawnCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`agent:
  spawn_command: ""
`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_NoAgentSection(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`project:
  name: "test"
`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_MissingFile(t *testing.T) {
	got := Command("/nonexistent/config.yaml")
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`{{{invalid yaml`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}
