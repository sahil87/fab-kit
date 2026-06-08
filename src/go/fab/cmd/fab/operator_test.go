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

	operatorStatePathOverride = yamlPath
	t.Cleanup(func() { operatorStatePathOverride = "" })

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
	yamlPath := filepath.Join(dir, "operator-state.yaml")

	operatorStatePathOverride = yamlPath
	t.Cleanup(func() { operatorStatePathOverride = "" })

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

// TestStateDir verifies XDG state base resolution: XDG_STATE_HOME is honored
// only when set AND absolute; otherwise it falls back to $HOME/.local/state.
func TestStateDir(t *testing.T) {
	t.Run("XDG_STATE_HOME absolute is honored", func(t *testing.T) {
		abs := filepath.Join(t.TempDir(), "xdgstate")
		t.Setenv("XDG_STATE_HOME", abs)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		if got != abs {
			t.Errorf("stateDir() = %q, want %q", got, abs)
		}
	})

	t.Run("XDG_STATE_HOME unset falls back to HOME/.local/state", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_STATE_HOME", "")
		t.Setenv("HOME", home)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		want := filepath.Join(home, ".local", "state")
		if got != want {
			t.Errorf("stateDir() = %q, want %q", got, want)
		}
	})

	t.Run("XDG_STATE_HOME relative is ignored", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("XDG_STATE_HOME", "relative/path")
		t.Setenv("HOME", home)
		got, err := stateDir()
		if err != nil {
			t.Fatalf("stateDir() error: %v", err)
		}
		want := filepath.Join(home, ".local", "state")
		if got != want {
			t.Errorf("stateDir() = %q, want %q (relative XDG_STATE_HOME must be ignored)", got, want)
		}
	})
}

// TestSlugify verifies the socket-path slug is filesystem-safe, deterministic,
// and collision-free for distinct socket paths.
func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"typical socket path", "/tmp/tmux-1000/default", "tmp-tmux-1000-default"},
		{"custom label socket", "/private/tmp/tmux-501/work", "private-tmp-tmux-501-work"},
		{"no leading separator", "tmp/tmux-1000/default", "tmp-tmux-1000-default"},
		{"empty falls back to default", "", "default"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := slugify(tc.in)
			if got != tc.want {
				t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// Filesystem-safe: no separators remain.
			if strings.ContainsRune(got, '/') || strings.ContainsRune(got, os.PathSeparator) {
				t.Errorf("slugify(%q) = %q contains a path separator", tc.in, got)
			}
			// Deterministic: same input → same output.
			if again := slugify(tc.in); again != got {
				t.Errorf("slugify(%q) not deterministic: %q vs %q", tc.in, got, again)
			}
		})
	}

	t.Run("distinct paths produce distinct slugs", func(t *testing.T) {
		paths := []string{
			"/tmp/tmux-1000/default",
			"/tmp/tmux-1000/work",
			"/tmp/tmux-1001/default",
			"/private/tmp/tmux-501/default",
		}
		seen := make(map[string]string)
		for _, p := range paths {
			s := slugify(p)
			if prev, ok := seen[s]; ok {
				t.Errorf("slug collision: %q and %q both → %q", prev, p, s)
			}
			seen[s] = p
		}
	})
}

// TestStatePath verifies the server-keyed state path layout and that the parent
// directory is created. serverSlug shells out to tmux; here we pin stateDir via
// HOME and accept whatever slug serverSlug derives (it falls back to "default"
// when tmux is unavailable in CI).
func TestStatePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)

	got, err := StatePath("")
	if err != nil {
		t.Fatalf("StatePath() error: %v", err)
	}

	dir := filepath.Join(home, ".local", "state", "fab", "operator")
	if filepath.Dir(got) != dir {
		t.Errorf("StatePath() dir = %q, want %q", filepath.Dir(got), dir)
	}
	if filepath.Ext(got) != ".yaml" {
		t.Errorf("StatePath() = %q, want a .yaml file", got)
	}
	// Parent directory must have been created.
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("StatePath() did not create parent dir %q: err=%v", dir, err)
	}
}
