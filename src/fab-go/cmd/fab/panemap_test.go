package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/fab-go/internal/resolve"
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
			result := formatIdleDuration(tc.seconds)
			if result != tc.expected {
				t.Errorf("formatIdleDuration(%d) = %q, want %q", tc.seconds, result, tc.expected)
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
			result := worktreeDisplayPath(tc.wtRoot, tc.mainRoot)
			if result != tc.expected {
				t.Errorf("worktreeDisplayPath(%q, %q) = %q, want %q", tc.wtRoot, tc.mainRoot, result, tc.expected)
			}
		})
	}
}

func TestPrintPaneTable(t *testing.T) {
	t.Run("single row", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{pane: "%3", tab: "alpha", worktree: "myrepo.worktrees/alpha/", change: "260306-r3m7-add-retry-logic", stage: "apply", agent: "active"},
		}
		printPaneTable(cmd, rows)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (header + 1 row), got %d:\n%s", len(lines), output)
		}

		for _, col := range []string{"Pane", "Tab", "Worktree", "Change", "Stage", "Agent"} {
			if !strings.Contains(lines[0], col) {
				t.Errorf("header missing column %q: %q", col, lines[0])
			}
		}

		for _, val := range []string{"%3", "alpha", "myrepo.worktrees/alpha/", "260306-r3m7-add-retry-logic", "apply", "active"} {
			if !strings.Contains(lines[1], val) {
				t.Errorf("data row missing value %q: %q", val, lines[1])
			}
		}
	})

	t.Run("multi row alignment", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{pane: "%3", tab: "alpha", worktree: "myrepo.worktrees/alpha/", change: "260306-r3m7-add-retry-logic", stage: "apply", agent: "active"},
			{pane: "%12", tab: "main", worktree: "(main)", change: "260306-ab12-refactor-auth", stage: "hydrate", agent: "idle (8m)"},
		}
		printPaneTable(cmd, rows)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), output)
		}

		// Verify alignment: Worktree column starts at same position in each line
		headerWtIdx := strings.Index(lines[0], "Worktree")
		row1WtIdx := strings.Index(lines[1], "myrepo.worktrees/alpha/")
		row2WtIdx := strings.Index(lines[2], "(main)")
		if headerWtIdx != row1WtIdx || headerWtIdx != row2WtIdx {
			t.Errorf("worktree column misaligned: header=%d, row1=%d, row2=%d", headerWtIdx, row1WtIdx, row2WtIdx)
		}
	})

	t.Run("edge case placeholders", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{pane: "%5", tab: "main", worktree: "(main)", change: "(no change)", stage: "\u2014", agent: "\u2014"},
		}
		printPaneTable(cmd, rows)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if !strings.Contains(lines[1], "(no change)") {
			t.Errorf("expected (no change) in output, got: %q", lines[1])
		}
	})

	t.Run("duplicate panes same worktree", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{pane: "%3", tab: "alpha", worktree: "repo.worktrees/alpha/", change: "260306-test-change", stage: "apply", agent: "active"},
			{pane: "%5", tab: "alpha", worktree: "repo.worktrees/alpha/", change: "260306-test-change", stage: "apply", agent: "active"},
		}
		printPaneTable(cmd, rows)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines (header + 2 rows), got %d", len(lines))
		}
		if !strings.Contains(lines[1], "%3") {
			t.Errorf("first row should have %%3: %q", lines[1])
		}
		if !strings.Contains(lines[2], "%5") {
			t.Errorf("second row should have %%5: %q", lines[2])
		}
	})
}

func TestReadFabCurrent(t *testing.T) {
	t.Run("symlink present", func(t *testing.T) {
		tmp := t.TempDir()
		target := "fab/changes/260306-ab12-some-change/.status.yaml"
		if err := os.Symlink(target, tmp+"/.fab-status.yaml"); err != nil {
			t.Fatal(err)
		}

		display, folder := readFabCurrent(tmp)
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

		display, folder := readFabCurrent(tmp)
		if display != "260306-ab12-deleted-change" {
			t.Errorf("display = %q, want %q", display, "260306-ab12-deleted-change")
		}
		if folder != "260306-ab12-deleted-change" {
			t.Errorf("folder = %q, want %q", folder, "260306-ab12-deleted-change")
		}
	})

	t.Run("no symlink", func(t *testing.T) {
		tmp := t.TempDir()

		display, folder := readFabCurrent(tmp)
		if display != "(no change)" {
			t.Errorf("display = %q, want %q", display, "(no change)")
		}
		if folder != "" {
			t.Errorf("folder = %q, want empty", folder)
		}
	})
}

func TestExtractFolderFromSymlink(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{"valid target", "fab/changes/260306-ab12-some-change/.status.yaml", "260306-ab12-some-change"},
		{"empty name", "fab/changes//.status.yaml", ""},
		{"no prefix", "other/260306-ab12-some-change/.status.yaml", ""},
		{"no suffix", "fab/changes/260306-ab12-some-change/other.yaml", ""},
		{"nested slash in name", "fab/changes/foo/bar/.status.yaml", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolve.ExtractFolderFromSymlink(tc.target)
			if result != tc.expected {
				t.Errorf("resolve.ExtractFolderFromSymlink(%q) = %q, want %q", tc.target, result, tc.expected)
			}
		})
	}
}
