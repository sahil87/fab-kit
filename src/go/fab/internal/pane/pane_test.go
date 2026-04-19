package pane

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWithServer(t *testing.T) {
	t.Run("empty server returns args verbatim", func(t *testing.T) {
		got := WithServer("", "list-panes", "-a")
		want := []string{"list-panes", "-a"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("non-empty server prepends -L", func(t *testing.T) {
		got := WithServer("runKit", "list-panes", "-a")
		want := []string{"-L", "runKit", "list-panes", "-a"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"runKit\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("no args with non-empty server returns just -L and server", func(t *testing.T) {
		got := WithServer("runKit")
		want := []string{"-L", "runKit"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"runKit\") = %v, want %v", got, want)
		}
	})

	t.Run("no args with empty server returns empty slice", func(t *testing.T) {
		got := WithServer("")
		if len(got) != 0 {
			t.Errorf("WithServer(\"\") = %v, want empty slice", got)
		}
	})

	t.Run("input args slice is not mutated across calls", func(t *testing.T) {
		original := []string{"list-panes", "-a", "-F", "#{pane_id}"}
		snapshot := make([]string, len(original))
		copy(snapshot, original)

		_ = WithServer("runKit", original...)
		_ = WithServer("runKit", original...)

		if !reflect.DeepEqual(original, snapshot) {
			t.Errorf("input slice mutated: got %v, want %v", original, snapshot)
		}
	})

	t.Run("special characters in server name passed verbatim", func(t *testing.T) {
		got := WithServer("my-socket", "list-panes")
		want := []string{"-L", "my-socket", "list-panes"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("WithServer(\"my-socket\", ...) = %v, want %v", got, want)
		}
		got2 := WithServer("socket_1", "list-panes")
		want2 := []string{"-L", "socket_1", "list-panes"}
		if !reflect.DeepEqual(got2, want2) {
			t.Errorf("WithServer(\"socket_1\", ...) = %v, want %v", got2, want2)
		}
	})
}

func TestFormatIdleDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int64
		expected string
	}{
		{"zero seconds", 0, "0s"},
		{"30 seconds", 30, "30s"},
		{"45 seconds", 45, "45s"},
		{"59 seconds", 59, "59s"},
		{"exactly 60 seconds", 60, "1m"},
		{"125 seconds", 125, "2m"},
		{"300 seconds (5m)", 300, "5m"},
		{"3599 seconds", 3599, "59m"},
		{"exactly 3600 seconds", 3600, "1h"},
		{"7500 seconds (2h)", 7500, "2h"},
		{"7200 seconds (2h exact)", 7200, "2h"},
		{"86400 seconds (24h)", 86400, "24h"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatIdleDuration(tc.seconds)
			if result != tc.expected {
				t.Errorf("FormatIdleDuration(%d) = %q, want %q", tc.seconds, result, tc.expected)
			}
		})
	}
}

func TestWorktreeDisplayPath(t *testing.T) {
	tests := []struct {
		name     string
		wtRoot   string
		mainRoot string
		expected string
	}{
		{
			"main worktree",
			"/home/user/myrepo",
			"/home/user/myrepo",
			"(main)",
		},
		{
			"child worktree",
			"/home/user/myrepo.worktrees/alpha",
			"/home/user/myrepo",
			"myrepo.worktrees/alpha/",
		},
		{
			"another child worktree",
			"/home/user/myrepo.worktrees/bravo",
			"/home/user/myrepo",
			"myrepo.worktrees/bravo/",
		},
		{
			"no main root fallback",
			"/home/user/some-repo",
			"",
			"some-repo/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := WorktreeDisplayPath(tc.wtRoot, tc.mainRoot)
			if result != tc.expected {
				t.Errorf("WorktreeDisplayPath(%q, %q) = %q, want %q", tc.wtRoot, tc.mainRoot, result, tc.expected)
			}
		})
	}
}

func TestReadFabCurrent(t *testing.T) {
	t.Run("symlink present", func(t *testing.T) {
		tmp := t.TempDir()
		target := "fab/changes/260306-ab12-some-change/.status.yaml"
		if err := os.Symlink(target, tmp+"/.fab-status.yaml"); err != nil {
			t.Fatal(err)
		}

		display, folder := ReadFabCurrent(tmp)
		if display != "260306-ab12-some-change" {
			t.Errorf("display = %q, want %q", display, "260306-ab12-some-change")
		}
		if folder != "260306-ab12-some-change" {
			t.Errorf("folder = %q, want %q", folder, "260306-ab12-some-change")
		}
	})

	t.Run("broken symlink", func(t *testing.T) {
		tmp := t.TempDir()
		target := "fab/changes/260306-ab12-deleted-change/.status.yaml"
		if err := os.Symlink(target, tmp+"/.fab-status.yaml"); err != nil {
			t.Fatal(err)
		}

		display, folder := ReadFabCurrent(tmp)
		if display != "260306-ab12-deleted-change" {
			t.Errorf("display = %q, want %q", display, "260306-ab12-deleted-change")
		}
		if folder != "260306-ab12-deleted-change" {
			t.Errorf("folder = %q, want %q", folder, "260306-ab12-deleted-change")
		}
	})

	t.Run("no symlink", func(t *testing.T) {
		tmp := t.TempDir()

		display, folder := ReadFabCurrent(tmp)
		if display != "(no change)" {
			t.Errorf("display = %q, want %q", display, "(no change)")
		}
		if folder != "" {
			t.Errorf("folder = %q, want empty", folder)
		}
	})
}

func TestLoadRuntimeFile(t *testing.T) {
	t.Run("missing file returns error", func(t *testing.T) {
		_, err := LoadRuntimeFile("/nonexistent/path/.fab-runtime.yaml")
		if err == nil {
			t.Error("expected error for missing file")
		}
		if !os.IsNotExist(err) {
			t.Errorf("expected IsNotExist error, got: %v", err)
		}
	})

	t.Run("valid yaml file", func(t *testing.T) {
		tmp := t.TempDir()
		path := tmp + "/.fab-runtime.yaml"
		content := "_agents:\n  uuid-1:\n    idle_since: 1234567890\n    tmux_pane: \"%5\"\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		m, err := LoadRuntimeFile(path)
		if err != nil {
			t.Fatal(err)
		}
		agents, ok := m["_agents"].(map[string]interface{})
		if !ok {
			t.Fatal("expected _agents map")
		}
		if _, ok := agents["uuid-1"]; !ok {
			t.Error("expected uuid-1 key in _agents")
		}
	})

	t.Run("empty file returns empty map", func(t *testing.T) {
		tmp := t.TempDir()
		path := tmp + "/.fab-runtime.yaml"
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}

		m, err := LoadRuntimeFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %d entries", len(m))
		}
	})
}

// writeRuntimeFixture writes a .fab-runtime.yaml into wtRoot with the given
// content. Used by matching tests to seed fixtures.
func writeRuntimeFixture(t *testing.T, wtRoot, content string) {
	t.Helper()
	path := filepath.Join(wtRoot, ".fab-runtime.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAgentState(t *testing.T) {
	t.Run("empty paneID returns empty", func(t *testing.T) {
		state, dur := ResolveAgentState("/tmp", "", "")
		if state != "" || dur != "" {
			t.Errorf("got state=%q dur=%q, want empty", state, dur)
		}
	})

	t.Run("missing runtime file returns empty state", func(t *testing.T) {
		tmp := t.TempDir()
		state, _ := ResolveAgentState(tmp, "%15", "")
		if state != "" {
			t.Errorf("state = %q, want empty", state)
		}
	})

	t.Run("idle entry matched by pane", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    idle_since: 1000
    tmux_pane: "%15"
`)
		state, dur := ResolveAgentState(tmp, "%15", "")
		if state != "idle" {
			t.Errorf("state = %q, want idle", state)
		}
		if dur == "" {
			t.Error("expected non-empty idle duration")
		}
	})

	t.Run("active entry matched by pane", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    tmux_pane: "%15"
    pid: 1000
`)
		state, _ := ResolveAgentState(tmp, "%15", "")
		if state != "active" {
			t.Errorf("state = %q, want active", state)
		}
	})

	t.Run("no matching pane returns empty", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    idle_since: 1000
    tmux_pane: "%99"
`)
		state, _ := ResolveAgentState(tmp, "%15", "")
		if state != "" {
			t.Errorf("state = %q, want empty for no-match", state)
		}
	})

	t.Run("discussion-mode agent (no change) still resolves", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    idle_since: 1000
    tmux_pane: "%15"
`)
		state, _ := ResolveAgentState(tmp, "%15", "")
		if state != "idle" {
			t.Errorf("expected idle state even without change; got %q", state)
		}
	})

	t.Run("server disambiguation — wrong server is skipped", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-a:
    idle_since: 1000
    tmux_pane: "%3"
    tmux_server: "fabKit"
  uuid-b:
    idle_since: 2000
    tmux_pane: "%3"
    tmux_server: "runKit"
`)
		_, dur := ResolveAgentState(tmp, "%3", "runKit")
		// Duration should correspond to uuid-b (idle_since=2000, more recent).
		// We don't check exact duration, only that a match was found.
		if dur == "" {
			t.Error("expected match for runKit server")
		}
	})

	t.Run("active entry beats idle entry for same pane", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-idle:
    idle_since: 1000
    tmux_pane: "%5"
  uuid-active:
    tmux_pane: "%5"
`)
		state, _ := ResolveAgentState(tmp, "%5", "")
		if state != "active" {
			t.Errorf("state = %q, want active (active entry should win over idle)", state)
		}
	})
}

func TestResolveAgentStateWithCache(t *testing.T) {
	t.Run("cache reuse across panes in same worktree", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    idle_since: 1000
    tmux_pane: "%15"
  uuid-2:
    tmux_pane: "%16"
`)
		cache := make(map[string]interface{})
		s1 := ResolveAgentStateWithCache(tmp, "%15", "", cache)
		s2 := ResolveAgentStateWithCache(tmp, "%16", "", cache)

		if s1 != "idle (" && !isIdlePrefix(s1) {
			// Formatting uses "idle (...)"; match prefix.
			t.Errorf("s1 = %q, expected idle prefix", s1)
		}
		if s2 != "active" {
			t.Errorf("s2 = %q, want active", s2)
		}
		if _, present := cache[tmp]; !present {
			t.Error("expected cache to have been populated for worktree")
		}
	})

	t.Run("no match returns em dash", func(t *testing.T) {
		tmp := t.TempDir()
		writeRuntimeFixture(t, tmp, `_agents:
  uuid-1:
    idle_since: 1000
    tmux_pane: "%99"
`)
		cache := make(map[string]interface{})
		got := ResolveAgentStateWithCache(tmp, "%15", "", cache)
		if got != "\u2014" {
			t.Errorf("got %q, want em dash", got)
		}
	})

	t.Run("missing file returns em dash", func(t *testing.T) {
		tmp := t.TempDir()
		cache := make(map[string]interface{})
		got := ResolveAgentStateWithCache(tmp, "%15", "", cache)
		if got != "\u2014" {
			t.Errorf("got %q, want em dash for missing file", got)
		}
	})

	t.Run("empty paneID returns em dash", func(t *testing.T) {
		cache := make(map[string]interface{})
		got := ResolveAgentStateWithCache("/tmp", "", "", cache)
		if got != "\u2014" {
			t.Errorf("got %q, want em dash for empty paneID", got)
		}
	})
}

// isIdlePrefix reports whether s starts with "idle (" — used for loose
// matching in tests where exact duration depends on wall-clock time.
func isIdlePrefix(s string) bool {
	return len(s) >= 6 && s[:6] == "idle ("
}

func TestFindAgentByPane(t *testing.T) {
	t.Run("exact pane match, no server", func(t *testing.T) {
		data := map[string]interface{}{
			"_agents": map[string]interface{}{
				"u-1": map[string]interface{}{
					"tmux_pane":  "%5",
					"idle_since": 1000,
				},
			},
		}
		entry, ok := findAgentByPane(data, "%5", "")
		if !ok {
			t.Fatal("expected match")
		}
		if entry["idle_since"] == nil {
			t.Error("expected idle_since to be present")
		}
	})

	t.Run("server disambiguation", func(t *testing.T) {
		data := map[string]interface{}{
			"_agents": map[string]interface{}{
				"u-a": map[string]interface{}{
					"tmux_pane":   "%3",
					"tmux_server": "fabKit",
					"idle_since":  1000,
				},
				"u-b": map[string]interface{}{
					"tmux_pane":   "%3",
					"tmux_server": "runKit",
					"idle_since":  2000,
				},
			},
		}
		entry, ok := findAgentByPane(data, "%3", "runKit")
		if !ok {
			t.Fatal("expected match")
		}
		ts, _ := asInt64(entry["idle_since"])
		if ts != 2000 {
			t.Errorf("expected uuid-b (idle_since=2000), got %v", ts)
		}
	})

	t.Run("empty server matches entries with any server", func(t *testing.T) {
		data := map[string]interface{}{
			"_agents": map[string]interface{}{
				"u-a": map[string]interface{}{
					"tmux_pane":   "%3",
					"tmux_server": "fabKit",
					"idle_since":  1000,
				},
			},
		}
		// Caller passes empty server — entry matches because empty-server
		// caller matches any entry.
		_, ok := findAgentByPane(data, "%3", "")
		if !ok {
			t.Error("expected match with empty caller server")
		}
	})

	t.Run("entry with empty server matches any caller server", func(t *testing.T) {
		data := map[string]interface{}{
			"_agents": map[string]interface{}{
				"u-a": map[string]interface{}{
					"tmux_pane":  "%3",
					"idle_since": 1000,
				},
			},
		}
		_, ok := findAgentByPane(data, "%3", "anything")
		if !ok {
			t.Error("expected match when entry has no tmux_server")
		}
	})

	t.Run("no _agents map returns no match", func(t *testing.T) {
		data := map[string]interface{}{}
		if _, ok := findAgentByPane(data, "%3", ""); ok {
			t.Error("expected no match when _agents absent")
		}
	})

	t.Run("most-recent idle wins among idle entries", func(t *testing.T) {
		data := map[string]interface{}{
			"_agents": map[string]interface{}{
				"u-old": map[string]interface{}{
					"tmux_pane":  "%5",
					"idle_since": 1000,
				},
				"u-new": map[string]interface{}{
					"tmux_pane":  "%5",
					"idle_since": 2000,
				},
			},
		}
		entry, ok := findAgentByPane(data, "%5", "")
		if !ok {
			t.Fatal("expected match")
		}
		ts, _ := asInt64(entry["idle_since"])
		if ts != 2000 {
			t.Errorf("got idle_since=%v, want newer (2000)", ts)
		}
	})
}
