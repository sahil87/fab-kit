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

// TestWithProfile verifies the doing-tier flag injection: both flags appended at
// the END in order model→effort (last-wins), each flag omitted entirely when its
// value is empty, and an all-empty profile leaving spawnCmd untouched.
func TestWithProfile(t *testing.T) {
	const base = "claude --dangerously-skip-permissions --effort xhigh"

	tests := []struct {
		name   string
		model  string
		effort string
		want   string
	}{
		{
			name:   "both present appended in order at end",
			model:  "claude-opus-4-8",
			effort: "high",
			want:   base + " --model claude-opus-4-8 --effort high",
		},
		{
			name:   "empty model only appends just effort",
			model:  "",
			effort: "high",
			want:   base + " --effort high",
		},
		{
			name:   "empty effort only appends just model",
			model:  "claude-opus-4-8",
			effort: "",
			want:   base + " --model claude-opus-4-8",
		},
		{
			name:   "both empty leaves spawnCmd unchanged",
			model:  "",
			effort: "",
			want:   base,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WithProfile(base, tc.model, tc.effort)
			if got != tc.want {
				t.Errorf("WithProfile(%q, %q, %q) = %q, want %q", base, tc.model, tc.effort, got, tc.want)
			}
		})
	}
}
