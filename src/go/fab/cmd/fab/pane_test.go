package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPaneCmdHelp(t *testing.T) {
	t.Run("lists all four subcommands", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := paneCmd()
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs([]string{})
		// Invoking with no subcommand should print help
		_ = cmd.Execute()

		output := buf.String()
		for _, sub := range []string{"map", "capture", "send", "process"} {
			if !strings.Contains(output, sub) {
				t.Errorf("help output should list %q subcommand:\n%s", sub, output)
			}
		}
	})
}

func TestOldPaneMapRemoved(t *testing.T) {
	t.Run("pane-map is not a root command", func(t *testing.T) {
		root := &cobra.Command{
			Use:           "fab",
			SilenceUsage:  true,
			SilenceErrors: true,
		}
		root.AddCommand(paneCmd())

		// Verify "pane-map" is not recognized
		root.SetArgs([]string{"pane-map"})
		err := root.Execute()
		if err == nil {
			t.Fatal("expected error for 'pane-map' (should be unknown command)")
		}
	})

	t.Run("pane map works", func(t *testing.T) {
		parent := paneCmd()
		var found bool
		for _, sub := range parent.Commands() {
			if sub.Name() == "map" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'map' subcommand under 'pane' parent")
		}
	})
}

func TestPaneCmdSubcommandCount(t *testing.T) {
	cmd := paneCmd()
	subs := cmd.Commands()
	if len(subs) != 4 {
		names := make([]string, len(subs))
		for i, s := range subs {
			names[i] = s.Name()
		}
		t.Errorf("expected 4 subcommands, got %d: %v", len(subs), names)
	}
}
