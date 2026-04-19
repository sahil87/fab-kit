package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
)

func TestParseTmuxServer(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"empty", "", ""},
		{"named socket fabKit", "/tmp/tmux-1001/fabKit,8671,0", "fabKit"},
		{"default socket", "/tmp/tmux-1001/default,8671,0", "default"},
		{"only path", "/tmp/tmux-1001/foo", "foo"},
		{"trailing comma after empty path", ",8671,0", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTmuxServer(tc.env)
			if got != tc.want {
				t.Errorf("parseTmuxServer(%q) = %q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

// hookTestEnv isolates env vars and cwd for a hook test and restores them on
// t.Cleanup.
func hookTestEnv(t *testing.T, cwd string, envOverrides map[string]string) {
	t.Helper()
	origEnv := map[string]string{}
	for k := range envOverrides {
		origEnv[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
	for k, v := range envOverrides {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	// Chdir for resolve.FabRoot()
	origWd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
}

// setupHookRepoRoot creates a temp repo with a fab/ subdir and returns
// (repoRoot, fabRoot).
func setupHookRepoRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	fabRoot := filepath.Join(root, "fab")
	if err := os.MkdirAll(fabRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, fabRoot
}

func TestHookStop_WellFormedPayload(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "/tmp/tmux-1001/fabKit,8671,0",
		envTmuxPane: "%15",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-stop-1","transcript_path":"/tmp/t.jsonl","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop returned error: %v", err)
	}

	m, err := runtime.LoadFile(runtime.FilePath(fabRoot))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	agents, ok := m["_agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _agents map")
	}
	got, ok := agents["uuid-stop-1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry for uuid-stop-1")
	}
	if got["idle_since"] == nil {
		t.Error("expected idle_since to be set")
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server = %v, want fabKit", got["tmux_server"])
	}
	if got["transcript_path"] != "/tmp/t.jsonl" {
		t.Errorf("transcript_path = %v", got["transcript_path"])
	}
}

func TestHookStop_MissingSessionID(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop returned error: %v", err)
	}

	// File should NOT have been created.
	if _, err := os.Stat(runtime.FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Errorf("expected no runtime file; got stat err=%v", err)
	}
}

func TestHookStop_MalformedPayload(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader("not-json"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop should swallow errors, got: %v", err)
	}
	if _, err := os.Stat(runtime.FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Errorf("expected no runtime file on malformed input")
	}
}

func TestHookStop_NoTmuxOmitsFields(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-nt","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-nt"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if _, present := got["tmux_pane"]; present {
		t.Errorf("tmux_pane should be absent, got %v", got["tmux_pane"])
	}
	if _, present := got["tmux_server"]; present {
		t.Errorf("tmux_server should be absent, got %v", got["tmux_server"])
	}
}

func TestHookStop_TmuxPaneOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "%15",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-pane","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-pane"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", got["tmux_pane"])
	}
	if _, present := got["tmux_server"]; present {
		t.Errorf("tmux_server should be omitted when $TMUX unset, got %v", got["tmux_server"])
	}
}

func TestHookStop_TmuxServerOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "/tmp/tmux-1001/fabKit,8671,0",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-srv","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-srv"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server = %v, want fabKit", got["tmux_server"])
	}
	if _, present := got["tmux_pane"]; present {
		t.Errorf("tmux_pane should be omitted, got %v", got["tmux_pane"])
	}
}

func TestHookSessionStart_DeletesEntry(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	// Seed an entry first.
	is := int64(1000)
	if err := runtime.WriteAgent(fabRoot, "uuid-ss", runtime.AgentEntry{IdleSince: &is, TmuxPane: "%5"}); err != nil {
		t.Fatal(err)
	}

	cmd := hookSessionStartCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-ss","hook_event_name":"SessionStart"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-ss"]; present {
		t.Error("expected entry to be deleted")
	}
}

func TestHookUserPrompt_ClearsIdleOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	// Seed with idle and tmux info.
	is := int64(1000)
	if err := runtime.WriteAgent(fabRoot, "uuid-up", runtime.AgentEntry{
		IdleSince:  &is,
		TmuxPane:   "%5",
		TmuxServer: "fabKit",
		Change:     "260417-2fbb",
	}); err != nil {
		t.Fatal(err)
	}

	cmd := hookUserPromptCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-up","hook_event_name":"UserPromptSubmit"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-up"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry to remain")
	}
	if _, present := got["idle_since"]; present {
		t.Error("expected idle_since to be cleared")
	}
	if got["tmux_pane"] != "%5" {
		t.Errorf("tmux_pane lost: %v", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server lost: %v", got["tmux_server"])
	}
	if got["change"] != "260417-2fbb" {
		t.Errorf("change lost: %v", got["change"])
	}
}
