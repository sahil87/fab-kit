package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

// writeRepoConfig creates <repo>/fab/project/config.yaml with the given body
// and returns the repo root.
func writeRepoConfig(t *testing.T, body string) string {
	t.Helper()
	repo := t.TempDir()
	projectDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return repo
}

func runSpawnCommandCmd(t *testing.T, repo string) string {
	t.Helper()
	cmd := spawnCommandCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if repo != "" {
		cmd.SetArgs([]string{"--repo", repo})
	}
	if err := cmd.Execute(); err != nil {
		t.Fatalf("spawn-command failed: %v\nstderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String())
}

func TestSpawnCommandCmd_Structure(t *testing.T) {
	cmd := spawnCommandCmd()
	if cmd.Use != "spawn-command" {
		t.Errorf("Use = %q, want %q", cmd.Use, "spawn-command")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	flag := cmd.Flags().Lookup("repo")
	if flag == nil {
		t.Fatal("expected --repo flag to be registered")
	}
	if flag.DefValue != "" {
		t.Errorf("expected empty default for --repo, got %q", flag.DefValue)
	}
}

func TestSpawnCommandCmd_RepoWithConfiguredCommand(t *testing.T) {
	repo := writeRepoConfig(t, "agent:\n  spawn_command: \"custom-claude --model opus\"\n")
	got := runSpawnCommandCmd(t, repo)
	if got != "custom-claude --model opus" {
		t.Errorf("spawn-command = %q, want %q", got, "custom-claude --model opus")
	}
}

func TestSpawnCommandCmd_RepoWithoutCommandFallsBack(t *testing.T) {
	repo := writeRepoConfig(t, "project:\n  name: test\n")
	got := runSpawnCommandCmd(t, repo)
	if got != spawn.DefaultSpawnCommand {
		t.Errorf("spawn-command = %q, want default %q", got, spawn.DefaultSpawnCommand)
	}
}

func TestSpawnCommandCmd_RepoMissingConfigFallsBack(t *testing.T) {
	// --repo pointing at a dir with no fab/project/config.yaml → DefaultSpawnCommand.
	repo := t.TempDir()
	got := runSpawnCommandCmd(t, repo)
	if got != spawn.DefaultSpawnCommand {
		t.Errorf("spawn-command = %q, want default %q", got, spawn.DefaultSpawnCommand)
	}
}
