package pane

import (
	"os"
	"testing"
)

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
		// Symlink to non-existent target — readlink still works
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
		content := "test-change:\n  agent:\n    idle_since: 1234567890\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		m, err := LoadRuntimeFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m["test-change"]; !ok {
			t.Error("expected test-change key in runtime data")
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

func TestResolveAgentState(t *testing.T) {
	t.Run("empty folder returns empty state", func(t *testing.T) {
		state, dur := ResolveAgentState("/tmp", "")
		if state != "" {
			t.Errorf("state = %q, want empty", state)
		}
		if dur != "" {
			t.Errorf("duration = %q, want empty", dur)
		}
	})

	t.Run("missing runtime file returns unknown", func(t *testing.T) {
		tmp := t.TempDir()
		state, dur := ResolveAgentState(tmp, "test-change")
		if state != "unknown" {
			t.Errorf("state = %q, want unknown", state)
		}
		if dur != "" {
			t.Errorf("duration = %q, want empty", dur)
		}
	})

	t.Run("active agent (no idle_since)", func(t *testing.T) {
		tmp := t.TempDir()
		path := tmp + "/.fab-runtime.yaml"
		content := "test-change:\n  agent:\n    pid: 12345\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		state, _ := ResolveAgentState(tmp, "test-change")
		if state != "active" {
			t.Errorf("state = %q, want active", state)
		}
	})

	t.Run("change not in runtime returns active", func(t *testing.T) {
		tmp := t.TempDir()
		path := tmp + "/.fab-runtime.yaml"
		content := "other-change:\n  agent:\n    idle_since: 1234567890\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		state, _ := ResolveAgentState(tmp, "test-change")
		if state != "active" {
			t.Errorf("state = %q, want active", state)
		}
	})
}
