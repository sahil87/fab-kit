package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGitRepoRoot_ReturnsPath(t *testing.T) {
	// This test runs inside the fab-kit repo, so gitRepoRoot should succeed
	root, err := gitRepoRoot()
	if err != nil {
		t.Skipf("not in a git repo: %v", err)
	}
	if root == "" {
		t.Error("gitRepoRoot() returned empty string")
	}
}

func TestOperatorCmd_Structure(t *testing.T) {
	cmd := operatorCmd()
	if cmd.Use != "operator" {
		t.Errorf("Use = %q, want %q", cmd.Use, "operator")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}

	// Verify tick-start and time subcommands are registered
	subNames := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subNames[sub.Use] = true
	}
	if !subNames["tick-start"] {
		t.Error("operator command missing tick-start subcommand")
	}
	if !subNames["time"] {
		t.Error("operator command missing time subcommand")
	}
}

// TestOperatorTickStart_IncrementsCount verifies that tick-start increments
// an existing tick_count, writes last_tick_at, preserves other fields, and
// outputs the correct stdout format.
func TestOperatorTickStart_IncrementsCount(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, ".fab-operator.yaml")

	initial := map[string]interface{}{
		"tick_count": 5,
		"monitored":  map[string]interface{}{},
	}
	raw, err := yaml.Marshal(initial)
	if err != nil {
		t.Fatalf("marshal initial yaml: %v", err)
	}
	if err := os.WriteFile(yamlPath, raw, 0644); err != nil {
		t.Fatalf("write initial yaml: %v", err)
	}

	operatorRepoRootOverride = dir
	t.Cleanup(func() { operatorRepoRootOverride = "" })

	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("tick-start failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tick: 6") {
		t.Errorf("stdout %q missing 'tick: 6'", out)
	}
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}

	// Read back and verify YAML
	updated, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read updated yaml: %v", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(updated, &result); err != nil {
		t.Fatalf("unmarshal updated yaml: %v", err)
	}
	if result["tick_count"] != 6 {
		t.Errorf("tick_count = %v, want 6", result["tick_count"])
	}
	lastTickAt, _ := result["last_tick_at"].(string)
	if lastTickAt == "" {
		t.Error("last_tick_at is empty or missing")
	}
	if _, ok := result["monitored"]; !ok {
		t.Error("monitored field was not preserved")
	}
}

// TestOperatorTickStart_MissingFile verifies that tick-start creates
// .fab-operator.yaml with tick_count=1 when the file does not exist.
func TestOperatorTickStart_MissingFile(t *testing.T) {
	dir := t.TempDir()

	operatorRepoRootOverride = dir
	t.Cleanup(func() { operatorRepoRootOverride = "" })

	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("tick-start failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "tick: 1") {
		t.Errorf("stdout %q missing 'tick: 1'", out)
	}

	// Verify file was created
	yamlPath := filepath.Join(dir, ".fab-operator.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read created yaml: %v", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal created yaml: %v", err)
	}
	if result["tick_count"] != 1 {
		t.Errorf("tick_count = %v, want 1", result["tick_count"])
	}
	lastTickAt, _ := result["last_tick_at"].(string)
	if lastTickAt == "" {
		t.Error("last_tick_at is empty or missing in created file")
	}
}

// TestOperatorTime_NoInterval verifies that 'fab operator time' with no flags
// outputs exactly one line matching 'now: HH:MM' and no 'next:' line.
func TestOperatorTime_NoInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("time failed: %v", err)
	}

	out := stdout.String()
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}
	if strings.Contains(out, "next:") {
		t.Errorf("stdout %q should not contain 'next:' when --interval not given", out)
	}
}

// TestOperatorTime_WithInterval verifies that --interval 3m produces both
// 'now: HH:MM' and 'next: HH:MM' in stdout.
func TestOperatorTime_WithInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--interval", "3m"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("time --interval 3m failed: %v", err)
	}

	out := stdout.String()
	hhmmRe := regexp.MustCompile(`now: \d\d:\d\d`)
	nextRe := regexp.MustCompile(`next: \d\d:\d\d`)
	if !hhmmRe.MatchString(out) {
		t.Errorf("stdout %q missing 'now: HH:MM'", out)
	}
	if !nextRe.MatchString(out) {
		t.Errorf("stdout %q missing 'next: HH:MM'", out)
	}
}

// TestOperatorTime_InvalidInterval verifies that an invalid --interval string
// causes the command to return an error (exit 1) and produce no stdout output.
func TestOperatorTime_InvalidInterval(t *testing.T) {
	cmd := operatorTimeCmd()
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetOut(&stdoutBuf)
	cmd.SetErr(&stderrBuf)
	cmd.SetArgs([]string{"--interval", "notaduration"})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for invalid --interval, got nil")
	}
	if stdoutBuf.Len() != 0 {
		t.Errorf("expected no stdout on error, got %q", stdoutBuf.String())
	}
}
