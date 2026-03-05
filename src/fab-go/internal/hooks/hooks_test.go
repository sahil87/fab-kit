package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_EmptyCommand(t *testing.T) {
	err := Run("/tmp", "")
	if err != nil {
		t.Errorf("empty command should return nil, got: %v", err)
	}
}

func TestRun_SuccessfulCommand(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)

	err := Run(fabRoot, "true")
	if err != nil {
		t.Errorf("'true' command should succeed, got: %v", err)
	}
}

func TestRun_FailingCommand(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)

	err := Run(fabRoot, "false")
	if err == nil {
		t.Error("'false' command should fail")
	}
}

func TestRun_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)

	// Create a sentinel file in the repo root (parent of fabRoot)
	sentinel := filepath.Join(dir, "sentinel.txt")
	os.WriteFile(sentinel, []byte("ok"), 0o644)

	// The hook should run with dir as cwd (parent of fabRoot)
	err := Run(fabRoot, "test -f sentinel.txt")
	if err != nil {
		t.Errorf("hook should find sentinel.txt in repo root: %v", err)
	}
}

func TestRun_ShellFeatures(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)

	// Pipes and redirects should work
	err := Run(fabRoot, "echo hello | grep hello > /dev/null")
	if err != nil {
		t.Errorf("shell features should work: %v", err)
	}
}

func TestRun_ScriptPath(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0o755)

	// Create a script in the repo root
	scriptPath := filepath.Join(dir, "check.sh")
	os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := Run(fabRoot, "./check.sh")
	if err != nil {
		t.Errorf("script path should work: %v", err)
	}
}
