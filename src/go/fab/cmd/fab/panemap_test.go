package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
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

func TestListAllPanesArgs(t *testing.T) {
	t.Run("no server returns single server-wide list-panes -a args", func(t *testing.T) {
		got := listAllPanesArgs("")
		want := []string{"list-panes", "-a", "-F", tmuxPaneFormat}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listAllPanesArgs(\"\") = %v, want %v", got, want)
		}
	})

	t.Run("server set prepends -L <server>", func(t *testing.T) {
		got := listAllPanesArgs("runKit")
		want := []string{"-L", "runKit", "list-panes", "-a", "-F", tmuxPaneFormat}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("listAllPanesArgs(\"runKit\") = %v, want %v", got, want)
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
		for _, field := range []string{"session", "window_index", "pane", "tab", "worktree", "repo", "change", "stage", "display_state", "agent_state", "agent_idle_duration"} {
			if !strings.Contains(output, "\""+field+"\"") {
				t.Errorf("JSON output missing field %q:\n%s", field, output)
			}
		}
	})

	t.Run("repo field populated and null when unresolved", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "(main)", repo: "/home/u/repo-a", change: "260306-r3m7-x", stage: "apply", agent: "active"},
			{session: "dev", windowIndex: 1, pane: "%5", tab: "scratch", worktree: "downloads/", repo: "—", change: "—", stage: "—", agent: "—"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		if result[0].Repo == nil || *result[0].Repo != "/home/u/repo-a" {
			t.Errorf("repo[0] = %v, want /home/u/repo-a", ptrStr(result[0].Repo))
		}
		if result[1].Repo != nil {
			t.Errorf("repo[1] should be null for unresolved repo, got %v", ptrStr(result[1].Repo))
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

// TestMainRootForPane verifies that the main-worktree root is resolved per
// distinct repo (not shared across repos) and cached by the pane's git worktree
// root. Two distinct git repos must yield two distinct main roots so panes in
// each render display paths relative to their OWN repo.
func TestMainRootForPane(t *testing.T) {
	repoA := initGitRepo(t)
	repoB := initGitRepo(t)

	wtCache := make(map[string]string)
	cache := make(map[string]string)

	wtA := worktreeRootForPane(repoA, wtCache)
	wtB := worktreeRootForPane(repoB, wtCache)
	gotA := mainRootForPane(repoA, wtA, cache)
	gotB := mainRootForPane(repoB, wtB, cache)

	if gotA == "" || gotB == "" {
		t.Fatalf("expected non-empty main roots, got A=%q B=%q", gotA, gotB)
	}
	if gotA == gotB {
		t.Errorf("distinct repos must yield distinct main roots; both = %q (the prior single-shared-root bug)", gotA)
	}
	// filepath.EvalSymlinks-tolerant comparison: git reports the resolved root.
	if filepath.Base(gotA) != filepath.Base(repoA) {
		t.Errorf("repoA main root = %q, want a path ending in %q", gotA, filepath.Base(repoA))
	}

	t.Run("non-git wtRoot short-circuits to empty main root", func(t *testing.T) {
		c := make(map[string]string)
		if got := mainRootForPane(t.TempDir(), "", c); got != "" {
			t.Errorf("empty wtRoot should yield empty main root, got %q", got)
		}
		if len(c) != 0 {
			t.Errorf("non-git path should not populate the mainRoot cache, got %v", c)
		}
	})
}

// TestWorktreeRootForPane verifies the cwd-keyed worktree-root cache: one
// `git rev-parse` per distinct cwd, with the "" non-git sentinel cached so
// failed lookups are never retried within an invocation.
func TestWorktreeRootForPane(t *testing.T) {
	t.Run("git cwd resolves and caches by cwd", func(t *testing.T) {
		repo := initGitRepo(t)
		cache := make(map[string]string)
		got := worktreeRootForPane(repo, cache)
		if got == "" {
			t.Fatal("expected non-empty worktree root for a git repo")
		}
		if filepath.Base(got) != filepath.Base(repo) {
			t.Errorf("worktree root = %q, want a path ending in %q", got, filepath.Base(repo))
		}
		cached, ok := cache[repo]
		if !ok || cached != got {
			t.Errorf("cache[%q] = (%q, %t), want (%q, true)", repo, cached, ok, got)
		}
		// Cache hit returns the same value (poison the cache to prove the
		// hit path is taken instead of re-spawning git).
		cache[repo] = "/poisoned"
		if again := worktreeRootForPane(repo, cache); again != "/poisoned" {
			t.Errorf("expected cache hit %q, got %q (git re-spawned?)", "/poisoned", again)
		}
	})

	t.Run("non-git cwd yields empty sentinel and is cached", func(t *testing.T) {
		nonGit := t.TempDir()
		cache := make(map[string]string)
		if got := worktreeRootForPane(nonGit, cache); got != "" {
			t.Errorf("non-git cwd should yield empty sentinel, got %q", got)
		}
		if cached, ok := cache[nonGit]; !ok || cached != "" {
			t.Errorf("non-git miss should be cached as \"\", got (%q, %t)", cached, ok)
		}
	})
}

// initGitRepo creates a fresh git repo in a temp dir and returns its path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed (git unavailable?): %v\n%s", args, err, out)
		}
	}
	return dir
}

// TestParsePRNumber verifies the trailing /pull/<n> segment parse, including
// trailing path/query/fragment tolerance, last-/pull/-wins, non-positive
// rejection, and the unparseable edge cases.
func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantNum int
		wantOK  bool
	}{
		{"canonical PR URL", "https://github.com/org/repo/pull/42", 42, true},
		{"trailing path tolerated", "https://github.com/org/repo/pull/42/files", 42, true},
		{"query string tolerated", "https://github.com/org/repo/pull/42?w=1", 42, true},
		{"fragment tolerated", "https://github.com/org/repo/pull/42#issuecomment-99", 42, true},
		{"query before fragment tolerated", "https://github.com/org/repo/pull/42?w=1#diff", 42, true},
		{"last /pull/ wins", "https://github.com/pull/owner/pull/7", 7, true},
		{"no /pull/ segment", "https://github.com/org/repo/issues/42", 0, false},
		{"non-numeric segment", "https://github.com/org/repo/pull/abc", 0, false},
		{"zero rejected", "https://github.com/org/repo/pull/0", 0, false},
		{"negative rejected", "https://github.com/org/repo/pull/-1", 0, false},
		{"empty string", "", 0, false},
		{"trailing slash only", "https://github.com/org/repo/pull/", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNum, gotOK := parsePRNumber(tc.url)
			if gotNum != tc.wantNum || gotOK != tc.wantOK {
				t.Errorf("parsePRNumber(%q) = (%d, %t), want (%d, %t)", tc.url, gotNum, gotOK, tc.wantNum, tc.wantOK)
			}
		})
	}
}

// TestPrintPaneJSONPRFields verifies the pr_url / pr_number JSON fields: a
// valid URL is surfaced with its parsed number, an empty URL yields both null,
// and a malformed URL keeps pr_url but nulls pr_number.
func TestPrintPaneJSONPRFields(t *testing.T) {
	t.Run("valid PR URL surfaces pr_url and parsed pr_number", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "(main)", change: "260306-r3m7-x", stage: "ship", agent: "active", prURL: "https://github.com/org/repo/pull/42"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		r := result[0]
		if r.PRURL == nil || *r.PRURL != "https://github.com/org/repo/pull/42" {
			t.Errorf("pr_url = %v, want the URL", ptrStr(r.PRURL))
		}
		if r.PRNumber == nil || *r.PRNumber != 42 {
			t.Errorf("pr_number = %v, want 42", ptrInt(r.PRNumber))
		}
	})

	t.Run("empty PR URL yields both null", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "(main)", change: "260306-r3m7-x", stage: "apply", agent: "active", prURL: ""},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.PRURL != nil {
			t.Errorf("pr_url should be null for empty URL, got %v", ptrStr(r.PRURL))
		}
		if r.PRNumber != nil {
			t.Errorf("pr_number should be null for empty URL, got %v", ptrInt(r.PRNumber))
		}
	})

	t.Run("malformed PR URL keeps pr_url but nulls pr_number", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "(main)", change: "260306-r3m7-x", stage: "ship", agent: "active", prURL: "https://github.com/org/repo/issues/7"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.PRURL == nil || *r.PRURL != "https://github.com/org/repo/issues/7" {
			t.Errorf("pr_url = %v, want the malformed URL preserved", ptrStr(r.PRURL))
		}
		if r.PRNumber != nil {
			t.Errorf("pr_number should be null for unparseable URL, got %v", ptrInt(r.PRNumber))
		}
	})

	t.Run("pr_url and pr_number JSON field names present", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "s", windowIndex: 0, pane: "%1", tab: "t", worktree: "w/", change: "c", stage: "apply", agent: "active", prURL: "https://github.com/org/repo/pull/1"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}
		output := buf.String()
		for _, field := range []string{"pr_url", "pr_number"} {
			if !strings.Contains(output, "\""+field+"\"") {
				t.Errorf("JSON output missing field %q:\n%s", field, output)
			}
		}
	})
}

// TestResolvePanePRURL verifies resolvePane surfaces the LAST entry of the
// .status.yaml prs: list (sourced from the already-loaded status file), and
// leaves prURL empty when the list is absent/empty.
func TestResolvePanePRURL(t *testing.T) {
	// writeChangeStatus sets up a git repo with a fab change whose .status.yaml
	// carries the given prs body, plus the .fab-status.yaml symlink. Returns the
	// worktree root (used as the pane cwd).
	writeChangeStatus := func(t *testing.T, prsBody string) string {
		t.Helper()
		wtRoot := initGitRepo(t)
		folder := "260609-r7ju-pane-map-pr-fields"
		changeDir := filepath.Join(wtRoot, "fab", "changes", folder)
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		statusYAML := "id: r7ju\nname: " + folder + "\nprogress:\n  intake:\n    state: done\n  apply:\n    state: active\n" + prsBody
		statusPath := filepath.Join(changeDir, ".status.yaml")
		if err := os.WriteFile(statusPath, []byte(statusYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		// .fab-status.yaml symlink points at the change's .status.yaml (relative
		// target matches the ExtractFolderFromSymlink contract).
		symlinkTarget := filepath.Join("fab", "changes", folder, ".status.yaml")
		symlinkPath := filepath.Join(wtRoot, ".fab-status.yaml")
		if err := os.Symlink(symlinkTarget, symlinkPath); err != nil {
			t.Fatal(err)
		}
		return wtRoot
	}

	t.Run("prURL is the last entry of a multi-URL prs list", func(t *testing.T) {
		prsBody := "prs:\n  - https://github.com/org/repo/pull/41\n  - https://github.com/org/repo/pull/42\n"
		wtRoot := writeChangeStatus(t, prsBody)

		p := paneEntry{id: "%1", tab: "alpha", cwd: wtRoot, session: "runK", index: 0}
		row, ok := resolvePane(p, wtRoot, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.prURL != "https://github.com/org/repo/pull/42" {
			t.Errorf("prURL = %q, want the LAST URL (pull/42)", row.prURL)
		}
	})

	t.Run("prURL empty when prs list absent", func(t *testing.T) {
		wtRoot := writeChangeStatus(t, "")

		p := paneEntry{id: "%1", tab: "alpha", cwd: wtRoot, session: "runK", index: 0}
		row, ok := resolvePane(p, wtRoot, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.prURL != "" {
			t.Errorf("prURL = %q, want empty for absent prs list", row.prURL)
		}
	})

	t.Run("prURL empty when prs list empty", func(t *testing.T) {
		wtRoot := writeChangeStatus(t, "prs: []\n")

		p := paneEntry{id: "%1", tab: "alpha", cwd: wtRoot, session: "runK", index: 0}
		row, ok := resolvePane(p, wtRoot, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.prURL != "" {
			t.Errorf("prURL = %q, want empty for empty prs list", row.prURL)
		}
	})
}

// TestPrintPaneJSONDisplayState verifies the nullable display_state JSON
// field ([dkn3]): populated alongside stage, null under the same conditions
// as stage, and present by name in the encoded output.
func TestPrintPaneJSONDisplayState(t *testing.T) {
	t.Run("populated display_state accompanies stage", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "runK", windowIndex: 0, pane: "%3", tab: "alpha", worktree: "(main)", change: "260306-r3m7-x", stage: "apply", displayState: "active", agent: "active"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		r := result[0]
		if r.Stage == nil || *r.Stage != "apply" {
			t.Errorf("stage = %v, want apply", ptrStr(r.Stage))
		}
		if r.DisplayState == nil || *r.DisplayState != "active" {
			t.Errorf("display_state = %v, want active", ptrStr(r.DisplayState))
		}
	})

	t.Run("display_state null exactly when stage null", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "dev", windowIndex: 0, pane: "%5", tab: "scratch", worktree: "downloads/", change: "—", stage: "—", displayState: "", agent: "—"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		r := result[0]
		if r.Stage != nil {
			t.Errorf("stage should be null, got %v", ptrStr(r.Stage))
		}
		if r.DisplayState != nil {
			t.Errorf("display_state should be null when stage is null, got %v", ptrStr(r.DisplayState))
		}
	})

	t.Run("display_state JSON field name present", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "s", windowIndex: 0, pane: "%1", tab: "t", worktree: "w/", change: "c", stage: "apply", displayState: "ready", agent: "active"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "\"display_state\"") {
			t.Errorf("JSON output missing field \"display_state\":\n%s", buf.String())
		}
	})
}

// TestResolvePaneDisplayState verifies resolvePane populates displayState
// from the state half of status.DisplayStage (previously discarded), and
// leaves it empty when the pane has no resolvable change.
func TestResolvePaneDisplayState(t *testing.T) {
	t.Run("active apply stage yields display_state active", func(t *testing.T) {
		wtRoot := initGitRepo(t)
		folder := "260612-pw3k-display-state"
		changeDir := filepath.Join(wtRoot, "fab", "changes", folder)
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		statusYAML := "id: pw3k\nname: " + folder + "\nprogress:\n  intake: done\n  apply: active\n"
		if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		symlinkTarget := filepath.Join("fab", "changes", folder, ".status.yaml")
		if err := os.Symlink(symlinkTarget, filepath.Join(wtRoot, ".fab-status.yaml")); err != nil {
			t.Fatal(err)
		}

		p := paneEntry{id: "%1", tab: "alpha", cwd: wtRoot, session: "runK", index: 0}
		row, ok := resolvePane(p, wtRoot, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.stage != "apply" {
			t.Errorf("stage = %q, want apply", row.stage)
		}
		if row.displayState != "active" {
			t.Errorf("displayState = %q, want active", row.displayState)
		}
	})

	t.Run("no resolvable change leaves displayState empty", func(t *testing.T) {
		wtRoot := initGitRepo(t)
		p := paneEntry{id: "%1", tab: "alpha", cwd: wtRoot, session: "runK", index: 0}
		row, ok := resolvePane(p, wtRoot, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.displayState != "" {
			t.Errorf("displayState = %q, want empty for no resolvable change", row.displayState)
		}
	})
}

// TestPrintPaneTableDisplayStateUnchanged asserts the table output is
// unaffected by the JSON-only display_state field: no new column, identical
// bytes whether or not displayState is set.
func TestPrintPaneTableDisplayStateUnchanged(t *testing.T) {
	rows := []paneRow{
		{session: "runK", windowIndex: 2, pane: "%3", tab: "alpha", worktree: "(main)", change: "260306-r3m7-x", stage: "apply", displayState: "active", agent: "active"},
	}

	var withState bytes.Buffer
	cmdWith := &cobra.Command{}
	cmdWith.SetOut(&withState)
	printPaneTable(cmdWith, rows, true)

	rowsNoState := make([]paneRow, len(rows))
	copy(rowsNoState, rows)
	for i := range rowsNoState {
		rowsNoState[i].displayState = ""
	}
	var withoutState bytes.Buffer
	cmdWithout := &cobra.Command{}
	cmdWithout.SetOut(&withoutState)
	printPaneTable(cmdWithout, rowsNoState, true)

	if withState.String() != withoutState.String() {
		t.Errorf("table output differs when displayState is set vs cleared:\nwith:\n%s\nwithout:\n%s", withState.String(), withoutState.String())
	}
	if strings.Contains(withState.String(), "display_state") {
		t.Errorf("table output should not contain display_state column:\n%s", withState.String())
	}
}

// TestPrintPaneTablePRFieldsUnchanged asserts the table output is unaffected by
// the JSON-only PR fields: no pr_url/pr_number columns, and the column set is
// byte-identical to the pre-change table for the same rows.
func TestPrintPaneTablePRFieldsUnchanged(t *testing.T) {
	rows := []paneRow{
		{session: "runK", windowIndex: 2, pane: "%3", tab: "alpha", worktree: "myrepo.worktrees/alpha/", change: "260306-r3m7-add-retry-logic", stage: "ship", agent: "active", prURL: "https://github.com/org/repo/pull/42"},
		{session: "dev", windowIndex: 1, pane: "%5", tab: "scratch", worktree: "downloads/", change: "(no change)", stage: "—", agent: "—", prURL: ""},
	}

	var withPR bytes.Buffer
	cmdWith := &cobra.Command{}
	cmdWith.SetOut(&withPR)
	printPaneTable(cmdWith, rows, true)

	// Identical rows but with prURL cleared — the table must render the same
	// bytes, proving prURL never reaches the table.
	rowsNoPR := make([]paneRow, len(rows))
	copy(rowsNoPR, rows)
	for i := range rowsNoPR {
		rowsNoPR[i].prURL = ""
	}
	var withoutPR bytes.Buffer
	cmdWithout := &cobra.Command{}
	cmdWithout.SetOut(&withoutPR)
	printPaneTable(cmdWithout, rowsNoPR, true)

	if withPR.String() != withoutPR.String() {
		t.Errorf("table output differs when prURL is set vs cleared:\nwith:\n%s\nwithout:\n%s", withPR.String(), withoutPR.String())
	}

	output := withPR.String()
	for _, absent := range []string{"pr_url", "pr_number", "PR", "https://github.com"} {
		if strings.Contains(output, absent) {
			t.Errorf("table output should not contain %q (JSON-only field leaked into table):\n%s", absent, output)
		}
	}
	// Header carries exactly the existing columns and no PR column.
	header := strings.Split(strings.TrimRight(output, "\n"), "\n")[0]
	for _, col := range []string{"Session", "Pane", "WinIdx", "Tab", "Worktree", "Change", "Stage", "Agent"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing expected column %q: %q", col, header)
		}
	}
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

func ptrInt(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return strconv.Itoa(*p)
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

// TestPrintPaneJSONDisplayState verifies the display_state JSON field: a
// change-bearing row surfaces the state half of DisplayStage, an em-dash
// sentinel row (no resolvable change) emits null, and the field sits
// immediately after stage in the output.
func TestPrintPaneJSONDisplayState(t *testing.T) {
	t.Run("change-bearing pane surfaces display_state", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		// Parked shipped change — the motivating case: stage review-pr, state done.
		rows := []paneRow{
			{session: "main", windowIndex: 2, pane: "%5", tab: "dkn3", worktree: "fab-kit.worktrees/dkn3/", change: "260612-dkn3-pane-map-display-state", stage: "review-pr", displayState: "done", agent: "—"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		r := result[0]
		if r.Stage == nil || *r.Stage != "review-pr" {
			t.Errorf("stage = %v, want review-pr", ptrStr(r.Stage))
		}
		if r.DisplayState == nil || *r.DisplayState != "done" {
			t.Errorf("display_state = %v, want done", ptrStr(r.DisplayState))
		}
	})

	t.Run("pane without a change emits null display_state", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "dev", windowIndex: 0, pane: "%7", tab: "scratch", worktree: "downloads/", change: "—", stage: "—", displayState: "—", agent: "—"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}

		var result []paneJSON
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if result[0].DisplayState != nil {
			t.Errorf("display_state should be null, got %v", ptrStr(result[0].DisplayState))
		}
	})

	t.Run("display_state key placed immediately after stage", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		rows := []paneRow{
			{session: "s", windowIndex: 0, pane: "%1", tab: "t", worktree: "w/", change: "c", stage: "apply", displayState: "active", agent: "active"},
		}
		if err := printPaneJSON(cmd, rows); err != nil {
			t.Fatal(err)
		}
		output := buf.String()
		stageIdx := strings.Index(output, "\"stage\"")
		dsIdx := strings.Index(output, "\"display_state\"")
		agentIdx := strings.Index(output, "\"agent_state\"")
		if stageIdx < 0 || dsIdx < 0 || agentIdx < 0 {
			t.Fatalf("missing expected keys in output:\n%s", output)
		}
		if !(stageIdx < dsIdx && dsIdx < agentIdx) {
			t.Errorf("key order wrong: stage@%d display_state@%d agent_state@%d — want stage < display_state < agent_state", stageIdx, dsIdx, agentIdx)
		}
	})
}

// TestResolvePaneDisplayState verifies resolvePane captures the state half of
// status.DisplayStage for a change-bearing pane and leaves the em-dash
// sentinel when the worktree has no fab change.
func TestResolvePaneDisplayState(t *testing.T) {
	t.Run("change-bearing pane captures DisplayStage state", func(t *testing.T) {
		wtRoot := initGitRepo(t)
		folder := "260612-dkn3-pane-map-display-state"
		changeDir := filepath.Join(wtRoot, "fab", "changes", folder)
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			t.Fatal(err)
		}
		statusYAML := "id: dkn3\nname: " + folder + "\nprogress:\n  intake: done\n  apply: active\n  review: pending\n  hydrate: pending\n  ship: pending\n  review-pr: pending\n"
		if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
			t.Fatal(err)
		}
		symlinkTarget := filepath.Join("fab", "changes", folder, ".status.yaml")
		if err := os.Symlink(symlinkTarget, filepath.Join(wtRoot, ".fab-status.yaml")); err != nil {
			t.Fatal(err)
		}

		p := paneEntry{id: "%1", tab: "dkn3", cwd: wtRoot, session: "main", index: 0}
		row, ok := resolvePane(p, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.stage != "apply" {
			t.Errorf("stage = %q, want apply", row.stage)
		}
		if row.displayState != "active" {
			t.Errorf("displayState = %q, want active", row.displayState)
		}
	})

	t.Run("fab worktree without a change keeps the em-dash sentinel", func(t *testing.T) {
		wtRoot := initGitRepo(t)
		// fab/ dir exists but no change and no .fab-status.yaml symlink.
		if err := os.MkdirAll(filepath.Join(wtRoot, "fab", "changes"), 0o755); err != nil {
			t.Fatal(err)
		}

		p := paneEntry{id: "%2", tab: "scratch", cwd: wtRoot, session: "main", index: 1}
		row, ok := resolvePane(p, wtRoot, "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.displayState != "—" {
			t.Errorf("displayState = %q, want em-dash sentinel", row.displayState)
		}
	})

	t.Run("non-git pane early-return row carries the em-dash sentinel", func(t *testing.T) {
		nonGit := t.TempDir()
		p := paneEntry{id: "%3", tab: "misc", cwd: nonGit, session: "dev", index: 2}
		row, ok := resolvePane(p, "", "", make(map[string]interface{}))
		if !ok {
			t.Fatal("resolvePane returned ok=false")
		}
		if row.displayState != "—" {
			t.Errorf("displayState = %q, want em-dash sentinel", row.displayState)
		}
	})
}

// TestPrintPaneTableDisplayStateUnchanged asserts the table output is
// unaffected by the JSON-only display_state field: rendering identical rows
// with displayState set vs cleared yields byte-identical tables.
func TestPrintPaneTableDisplayStateUnchanged(t *testing.T) {
	rows := []paneRow{
		{session: "main", windowIndex: 2, pane: "%5", tab: "dkn3", worktree: "fab-kit.worktrees/dkn3/", change: "260612-dkn3-pane-map-display-state", stage: "review-pr", displayState: "done", agent: "—"},
		{session: "dev", windowIndex: 1, pane: "%7", tab: "scratch", worktree: "downloads/", change: "—", stage: "—", displayState: "—", agent: "—"},
	}

	var withState bytes.Buffer
	cmdWith := &cobra.Command{}
	cmdWith.SetOut(&withState)
	printPaneTable(cmdWith, rows, true)

	rowsNoState := make([]paneRow, len(rows))
	copy(rowsNoState, rows)
	for i := range rowsNoState {
		rowsNoState[i].displayState = ""
	}
	var withoutState bytes.Buffer
	cmdWithout := &cobra.Command{}
	cmdWithout.SetOut(&withoutState)
	printPaneTable(cmdWithout, rowsNoState, true)

	if withState.String() != withoutState.String() {
		t.Errorf("table output differs when displayState is set vs cleared:\nwith:\n%s\nwithout:\n%s", withState.String(), withoutState.String())
	}
	if strings.Contains(withState.String(), "display_state") {
		t.Errorf("table output should not contain a display_state column:\n%s", withState.String())
	}
}
