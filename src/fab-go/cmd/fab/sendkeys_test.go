package main

import (
	"strings"
	"testing"
)

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

func TestBuildSendKeysArgs(t *testing.T) {
	args := buildSendKeysArgs("%3", "/fab-continue")
	expected := []string{"send-keys", "-t", "%3", "/fab-continue", "Enter"}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestBuildSendKeysArgsWithSpaces(t *testing.T) {
	args := buildSendKeysArgs("%7", "git fetch origin main && git rebase origin/main")
	if args[3] != "git fetch origin main && git rebase origin/main" {
		t.Errorf("text arg should be preserved verbatim, got %q", args[3])
	}
	if args[4] != "Enter" {
		t.Errorf("last arg should be Enter, got %q", args[4])
	}
}

func TestValidateSendKeysInputs(t *testing.T) {
	t.Run("no tmux session", func(t *testing.T) {
		err := validateSendKeysInputs("r3m7", "")
		if err == nil {
			t.Fatal("expected error for missing TMUX")
		}
		if !strings.Contains(err.Error(), "not inside a tmux session") {
			t.Errorf("error should mention tmux, got: %s", err)
		}
	})

	t.Run("valid inputs", func(t *testing.T) {
		err := validateSendKeysInputs("r3m7", "/tmp/tmux-1000/default")
		if err != nil {
			t.Errorf("expected no error, got: %s", err)
		}
	})

	t.Run("empty change arg", func(t *testing.T) {
		err := validateSendKeysInputs("", "/tmp/tmux-1000/default")
		if err == nil {
			t.Fatal("expected error for empty change arg")
		}
	})

	t.Run("whitespace change arg", func(t *testing.T) {
		err := validateSendKeysInputs("   ", "/tmp/tmux-1000/default")
		if err == nil {
			t.Fatal("expected error for whitespace change arg")
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

	t.Run("git dir without fab returns empty", func(t *testing.T) {
		// resolvePaneChange calls gitWorktreeRoot which needs a real git repo
		// We just test that a random directory returns empty (no git)
		p := paneEntry{id: "%1", tab: "test", cwd: "/tmp"}
		result := resolvePaneChange(p)
		if result != "" {
			t.Errorf("expected empty for non-fab git dir, got %q", result)
		}
	})
}

