package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

func TestListPanesArgs(t *testing.T) {
	t.Run("no name, no server returns bare list-panes args", func(t *testing.T) {
		got := listPanesArgs("", "")
		want := []string{"list-panes", "-s", "-F", tmuxPaneFormat}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listPanesArgs(\"\", \"\") = %v, want %v", got, want)
		}
	})

	t.Run("name set, no server appends -t <name>", func(t *testing.T) {
		got := listPanesArgs("main", "")
		want := []string{"list-panes", "-s", "-F", tmuxPaneFormat, "-t", "main"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listPanesArgs(\"main\", \"\") = %v, want %v", got, want)
		}
	})

	t.Run("no name, server set prepends -L <server>", func(t *testing.T) {
		got := listPanesArgs("", "runKit")
		want := []string{"-L", "runKit", "list-panes", "-s", "-F", tmuxPaneFormat}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listPanesArgs(\"\", \"runKit\") = %v, want %v", got, want)
		}
	})

	t.Run("name and server both set prepend -L and append -t", func(t *testing.T) {
		got := listPanesArgs("main", "runKit")
		want := []string{"-L", "runKit", "list-panes", "-s", "-F", tmuxPaneFormat, "-t", "main"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listPanesArgs(\"main\", \"runKit\") = %v, want %v", got, want)
		}
	})
}

func TestListSessionsArgs(t *testing.T) {
	t.Run("no server returns bare list-sessions args", func(t *testing.T) {
		got := listSessionsArgs("")
		want := []string{"list-sessions", "-F", "#{session_name}"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listSessionsArgs(\"\") = %v, want %v", got, want)
		}
	})

	t.Run("server set prepends -L <server>", func(t *testing.T) {
		got := listSessionsArgs("runKit")
		want := []string{"-L", "runKit", "list-sessions", "-F", "#{session_name}"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listSessionsArgs(\"runKit\") = %v, want %v", got, want)
		}
	})
}

func TestPaneMapServerFlag(t *testing.T) {
	t.Run("--server flag is registered via persistent flag", func(t *testing.T) {
		// The persistent --server flag is registered on paneCmd. Verify it
		// is visible to paneMapCmd as a persistent flag (inherited).
		parent := paneCmd()
		// Find the map subcommand.
		var mapSub *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "map" {
				mapSub = c
				break
			}
		}
		if mapSub == nil {
			t.Fatal("paneCmd did not register a map subcommand")
		}
		flag := mapSub.Flags().Lookup("server")
		if flag == nil {
			// Persistent flags on parent may only be visible via InheritedFlags.
			flag = mapSub.InheritedFlags().Lookup("server")
		}
		if flag == nil {
			t.Fatal("expected --server flag to be visible on pane map subcommand")
		}
		if flag.Shorthand != "L" {
			t.Errorf("expected shorthand \"L\", got %q", flag.Shorthand)
		}
		if flag.DefValue != "" {
			t.Errorf("expected empty default, got %q", flag.DefValue)
		}
	})
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

// TestPrintPaneJSON_DiscussionMode verifies the three-axis independence
// of change/agent fields in JSON output. A discussion-mode row has a null
// change but a populated agent — previously impossible in the old
// folder-keyed schema.
func TestPrintPaneJSON_DiscussionMode(t *testing.T) {
	t.Run("discussion-mode pane populates agent_state with null change", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		// Discussion mode: change and stage are em-dash; agent is populated.
		rows := []paneRow{
			{session: "main", windowIndex: 2, pane: "%15", tab: "scratch", worktree: "(main)", change: "(no change)", stage: "\u2014", agent: "idle (2m)"},
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
			t.Errorf("change should be null in discussion mode, got %v", ptrStr(r.Change))
		}
		if r.Stage != nil {
			t.Errorf("stage should be null in discussion mode, got %v", ptrStr(r.Stage))
		}
		if r.AgentState == nil || *r.AgentState != "idle" {
			t.Errorf("agent_state = %v, want idle", ptrStr(r.AgentState))
		}
		if r.AgentIdleDuration == nil || *r.AgentIdleDuration != "2m" {
			t.Errorf("agent_idle_duration = %v, want 2m", ptrStr(r.AgentIdleDuration))
		}
	})

	t.Run("discussion-mode active agent has agent_state active, null change", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "main", windowIndex: 0, pane: "%15", tab: "scratch", worktree: "(main)", change: "(no change)", stage: "\u2014", agent: "active"},
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
		if r.AgentState == nil || *r.AgentState != "active" {
			t.Errorf("agent_state = %v, want active", ptrStr(r.AgentState))
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
