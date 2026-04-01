package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
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
		printPaneTable(cmd, rows, false)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines (header + 1 row), got %d:\n%s", len(lines), output)
		}

		for _, col := range []string{"Pane", "WinIdx", "Tab", "Worktree", "Change", "Stage", "Agent"} {
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
		printPaneTable(cmd, rows, false)

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
		printPaneTable(cmd, rows, false)

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
		printPaneTable(cmd, rows, false)

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

func TestMatchPanesByFolder(t *testing.T) {
	// stub resolver that returns the pane's cwd as the "folder"
	stubResolver := func(p paneEntry) string {
		return p.cwd // cwd is used as a stand-in for the resolved folder
	}

	t.Run("single match", func(t *testing.T) {
		panes := []paneEntry{
			{id: "%3", tab: "alpha", cwd: "260306-ab12-some-change"},
			{id: "%7", tab: "bravo", cwd: "260306-cd34-other-change"},
		}

		matches, warning := matchPanesByFolder(panes, "260306-ab12-some-change", stubResolver)
		if len(matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(matches))
		}
		if matches[0] != "%3" {
			t.Errorf("expected %%3, got %s", matches[0])
		}
		if warning != "" {
			t.Errorf("expected no warning, got %q", warning)
		}
	})

	t.Run("no match", func(t *testing.T) {
		panes := []paneEntry{
			{id: "%3", tab: "alpha", cwd: "260306-ab12-some-change"},
			{id: "%7", tab: "bravo", cwd: "260306-cd34-other-change"},
		}

		matches, _ := matchPanesByFolder(panes, "260306-xyz-nonexistent", stubResolver)
		if len(matches) != 0 {
			t.Fatalf("expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("multiple matches produces warning", func(t *testing.T) {
		panes := []paneEntry{
			{id: "%3", tab: "alpha", cwd: "260306-ab12-some-change"},
			{id: "%7", tab: "bravo", cwd: "260306-ab12-some-change"},
		}

		matches, warning := matchPanesByFolder(panes, "260306-ab12-some-change", stubResolver)
		if len(matches) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(matches))
		}
		if matches[0] != "%3" {
			t.Errorf("first match should be %%3, got %s", matches[0])
		}
		if warning == "" {
			t.Error("expected warning for multiple panes, got empty")
		}
		// Warning should mention the first pane ID
		if !strings.Contains(warning, "%3") {
			t.Errorf("warning should mention first pane %%3: %q", warning)
		}
	})

	t.Run("empty pane list", func(t *testing.T) {
		matches, _ := matchPanesByFolder(nil, "260306-ab12-some-change", stubResolver)
		if len(matches) != 0 {
			t.Fatalf("expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("non-matching resolver", func(t *testing.T) {
		alwaysEmpty := func(p paneEntry) string { return "" }
		panes := []paneEntry{
			{id: "%3", tab: "alpha", cwd: "260306-ab12-some-change"},
		}

		matches, _ := matchPanesByFolder(panes, "260306-ab12-some-change", alwaysEmpty)
		if len(matches) != 0 {
			t.Fatalf("expected 0 matches with empty resolver, got %d", len(matches))
		}
	})
}

func TestResolvePaneChange(t *testing.T) {
	t.Run("non-git directory returns empty", func(t *testing.T) {
		tmp := t.TempDir()
		p := paneEntry{id: "%1", tab: "test", cwd: tmp}
		result := resolvePaneChange(p)
		if result != "" {
			t.Errorf("expected empty for non-git dir, got %q", result)
		}
	})

	t.Run("non-git /tmp directory returns empty", func(t *testing.T) {
		// resolvePaneChange calls gitWorktreeRoot which needs a real git repo.
		// Here we just use a fixed non-git directory (/tmp) and expect empty.
		p := paneEntry{id: "%1", tab: "test", cwd: "/tmp"}
		result := resolvePaneChange(p)
		if result != "" {
			t.Errorf("expected empty for non-git /tmp dir, got %q", result)
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

func TestParsePaneLines(t *testing.T) {
	t.Run("standard five-field line", func(t *testing.T) {
		input := "%3\talpha\t/home/user/repo\trunK\t2\n"
		panes, err := parsePaneLines(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(panes) != 1 {
			t.Fatalf("expected 1 pane, got %d", len(panes))
		}
		p := panes[0]
		if p.id != "%3" {
			t.Errorf("id = %q, want %%3", p.id)
		}
		if p.tab != "alpha" {
			t.Errorf("tab = %q, want alpha", p.tab)
		}
		if p.cwd != "/home/user/repo" {
			t.Errorf("cwd = %q, want /home/user/repo", p.cwd)
		}
		if p.session != "runK" {
			t.Errorf("session = %q, want runK", p.session)
		}
		if p.index != 2 {
			t.Errorf("index = %d, want 2", p.index)
		}
	})

	t.Run("multiple lines", func(t *testing.T) {
		input := "%3\talpha\t/home/user/repo\trunK\t0\n%7\tbravo\t/tmp\tdev\t1\n"
		panes, err := parsePaneLines(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(panes) != 2 {
			t.Fatalf("expected 2 panes, got %d", len(panes))
		}
		if panes[0].session != "runK" {
			t.Errorf("pane 0 session = %q, want runK", panes[0].session)
		}
		if panes[1].session != "dev" {
			t.Errorf("pane 1 session = %q, want dev", panes[1].session)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		panes, err := parsePaneLines("")
		if err != nil {
			t.Fatal(err)
		}
		if len(panes) != 0 {
			t.Fatalf("expected 0 panes, got %d", len(panes))
		}
	})

	t.Run("malformed line skipped", func(t *testing.T) {
		input := "%3\talpha\n%7\tbravo\t/tmp\tdev\t1\n"
		panes, err := parsePaneLines(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(panes) != 1 {
			t.Fatalf("expected 1 pane (malformed skipped), got %d", len(panes))
		}
		if panes[0].id != "%7" {
			t.Errorf("expected %%7, got %s", panes[0].id)
		}
	})

	t.Run("non-numeric window index defaults to zero", func(t *testing.T) {
		input := "%3\talpha\t/home/user/repo\trunK\tabc\n"
		panes, err := parsePaneLines(input)
		if err != nil {
			t.Fatal(err)
		}
		if len(panes) != 1 {
			t.Fatalf("expected 1 pane, got %d", len(panes))
		}
		if panes[0].index != 0 {
			t.Errorf("index = %d, want 0 for non-numeric input", panes[0].index)
		}
	})
}

func TestSplitAgentState(t *testing.T) {
	tests := []struct {
		name             string
		agent            string
		wantState        *string
		wantIdleDuration *string
	}{
		{"active", "active", strPtr("active"), nil},
		{"em dash", "\u2014", nil, nil},
		{"unknown", "?", strPtr("unknown"), nil},
		{"idle with duration", "idle (5m)", strPtr("idle"), strPtr("5m")},
		{"idle with seconds", "idle (30s)", strPtr("idle"), strPtr("30s")},
		{"idle with hours", "idle (2h)", strPtr("idle"), strPtr("2h")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state, dur := splitAgentState(tc.agent)
			if !ptrEq(state, tc.wantState) {
				t.Errorf("state = %v, want %v", ptrStr(state), ptrStr(tc.wantState))
			}
			if !ptrEq(dur, tc.wantIdleDuration) {
				t.Errorf("idle_duration = %v, want %v", ptrStr(dur), ptrStr(tc.wantIdleDuration))
			}
		})
	}
}

func TestPrintPaneJSON(t *testing.T) {
	t.Run("active pane", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 2, pane: "%3", tab: "alpha", worktree: "myrepo.worktrees/alpha/", change: "260306-r3m7-add-retry-logic", stage: "apply", agent: "active"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 element, got %d", len(result))
		}
		r := result[0]
		if r.Session != "runK" {
			t.Errorf("session = %q, want runK", r.Session)
		}
		if r.WindowIndex != 2 {
			t.Errorf("window_index = %d, want 2", r.WindowIndex)
		}
		if r.Pane != "%3" {
			t.Errorf("pane = %q, want %%3", r.Pane)
		}
		if r.Change == nil || *r.Change != "260306-r3m7-add-retry-logic" {
			t.Errorf("change = %v, want 260306-r3m7-add-retry-logic", ptrStr(r.Change))
		}
		if r.Stage == nil || *r.Stage != "apply" {
			t.Errorf("stage = %v, want apply", ptrStr(r.Stage))
		}
		if r.AgentState == nil || *r.AgentState != "active" {
			t.Errorf("agent_state = %v, want active", ptrStr(r.AgentState))
		}
		if r.AgentIdleDuration != nil {
			t.Errorf("agent_idle_duration = %v, want null", ptrStr(r.AgentIdleDuration))
		}
	})

	t.Run("non-fab pane has null fields", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "dev", windowIndex: 0, pane: "%5", tab: "scratch", worktree: "downloads/", change: "\u2014", stage: "\u2014", agent: "\u2014"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.Change != nil {
			t.Errorf("change should be null, got %v", ptrStr(r.Change))
		}
		if r.Stage != nil {
			t.Errorf("stage should be null, got %v", ptrStr(r.Stage))
		}
		if r.AgentState != nil {
			t.Errorf("agent_state should be null, got %v", ptrStr(r.AgentState))
		}
		if r.AgentIdleDuration != nil {
			t.Errorf("agent_idle_duration should be null, got %v", ptrStr(r.AgentIdleDuration))
		}
	})

	t.Run("idle agent with duration", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 1, pane: "%7", tab: "bravo", worktree: "(main)", change: "260306-ab12-refactor-auth", stage: "review", agent: "idle (5m)"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.AgentState == nil || *r.AgentState != "idle" {
			t.Errorf("agent_state = %v, want idle", ptrStr(r.AgentState))
		}
		if r.AgentIdleDuration == nil || *r.AgentIdleDuration != "5m" {
			t.Errorf("agent_idle_duration = %v, want 5m", ptrStr(r.AgentIdleDuration))
		}
	})

	t.Run("unknown agent state", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%9", tab: "charlie", worktree: "(main)", change: "260306-xy12-test", stage: "apply", agent: "?"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.AgentState == nil || *r.AgentState != "unknown" {
			t.Errorf("agent_state = %v, want unknown", ptrStr(r.AgentState))
		}
	})

	t.Run("no-change maps to null", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "dev", windowIndex: 0, pane: "%1", tab: "main", worktree: "(main)", change: "(no change)", stage: "\u2014", agent: "\u2014"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result[0].Change != nil {
			t.Errorf("change should be null for (no change), got %v", ptrStr(result[0].Change))
		}
	})

	t.Run("JSON field names are snake_case", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "s", windowIndex: 0, pane: "%1", tab: "t", worktree: "w/", change: "c", stage: "apply", agent: "active"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		output := buf.String()
		for _, field := range []string{"session", "window_index", "pane", "tab", "worktree", "change", "stage", "agent_state", "agent_idle_duration"} {
			if !strings.Contains(output, "\""+field+"\"") {
				t.Errorf("JSON output missing field %q:\n%s", field, output)
			}
		}
	})
}

func TestPrintPaneTableWithWinIdx(t *testing.T) {
	t.Run("WinIdx column present", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 3, pane: "%3", tab: "alpha", worktree: "repo/", change: "test", stage: "apply", agent: "active"},
		}
		printPaneTable(cmd, rows, false)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if !strings.Contains(lines[0], "WinIdx") {
			t.Errorf("header missing WinIdx column: %q", lines[0])
		}
		if !strings.Contains(lines[1], "3") {
			t.Errorf("data row missing window index 3: %q", lines[1])
		}
	})

	t.Run("Session column absent when showSession is false", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "repo/", change: "test", stage: "apply", agent: "active"},
		}
		printPaneTable(cmd, rows, false)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if strings.Contains(lines[0], "Session") {
			t.Errorf("header should NOT contain Session column in single-session mode: %q", lines[0])
		}
	})

	t.Run("Session column present when showSession is true", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "repo/", change: "test", stage: "apply", agent: "active"},
			{session: "dev", windowIndex: 1, pane: "%7", tab: "bravo", worktree: "(main)", change: "other", stage: "review", agent: "idle (2m)"},
		}
		printPaneTable(cmd, rows, true)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		if !strings.Contains(lines[0], "Session") {
			t.Errorf("header missing Session column in all-sessions mode: %q", lines[0])
		}
		if !strings.Contains(lines[1], "runK") {
			t.Errorf("row 1 missing session name runK: %q", lines[1])
		}
		if !strings.Contains(lines[2], "dev") {
			t.Errorf("row 2 missing session name dev: %q", lines[2])
		}
	})

	t.Run("WinIdx between Pane and Tab", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "s", windowIndex: 5, pane: "%1", tab: "mytab", worktree: "repo/", change: "c", stage: "apply", agent: "active"},
		}
		printPaneTable(cmd, rows, false)

		header := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")[0]
		paneIdx := strings.Index(header, "Pane")
		winIdxIdx := strings.Index(header, "WinIdx")
		tabIdx := strings.Index(header, "Tab")
		if paneIdx >= winIdxIdx || winIdxIdx >= tabIdx {
			t.Errorf("column order wrong: Pane@%d WinIdx@%d Tab@%d — expected Pane < WinIdx < Tab", paneIdx, winIdxIdx, tabIdx)
		}
	})
}

func TestPaneMapMutualExclusion(t *testing.T) {
	t.Run("session and all-sessions are mutually exclusive", func(t *testing.T) {
		cmd := paneMapCmd()
		cmd.SetArgs([]string{"--session", "foo", "--all-sessions"})
		// Cobra's MarkFlagsMutuallyExclusive should produce an error
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for mutually exclusive flags, got nil")
		}
		if !strings.Contains(err.Error(), "if any flags in the group") {
			t.Errorf("expected mutual exclusion error, got: %v", err)
		}
	})
}

// helpers

func strPtr(s string) *string {
	return &s
}

func ptrStr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func ptrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
