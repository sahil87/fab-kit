package main

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestPaneSendCmd(t *testing.T) {
	t.Run("requires two arguments", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SetArgs([]string{"%5"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing text argument, got nil")
		}
	})

	t.Run("requires at least pane argument", func(t *testing.T) {
		cmd := paneSendCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing arguments, got nil")
		}
	})

	t.Run("no-enter flag defaults to false", func(t *testing.T) {
		cmd := paneSendCmd()
		noEnter, _ := cmd.Flags().GetBool("no-enter")
		if noEnter {
			t.Error("expected no-enter to default to false")
		}
	})

	t.Run("force flag defaults to false", func(t *testing.T) {
		cmd := paneSendCmd()
		force, _ := cmd.Flags().GetBool("force")
		if force {
			t.Error("expected force to default to false")
		}
	})

	t.Run("flag existence", func(t *testing.T) {
		cmd := paneSendCmd()

		noEnterFlag := cmd.Flags().Lookup("no-enter")
		if noEnterFlag == nil {
			t.Error("expected 'no-enter' flag to exist")
		}

		forceFlag := cmd.Flags().Lookup("force")
		if forceFlag == nil {
			t.Error("expected 'force' flag to exist")
		}
	})
}

func TestSendTextArgs(t *testing.T) {
	t.Run("empty server returns bare send-keys -l argv", func(t *testing.T) {
		got := sendTextArgs("", "%5", "hello")
		want := []string{"send-keys", "-t", "%5", "-l", "hello"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sendTextArgs(\"\", ...) = %v, want %v", got, want)
		}
		// Explicit: no -L anywhere
		for _, el := range got {
			if el == "-L" {
				t.Errorf("did not expect -L in argv for empty server, got %v", got)
			}
		}
	})

	t.Run("non-empty server prepends -L <server>", func(t *testing.T) {
		got := sendTextArgs("runKit", "%5", "hello")
		want := []string{"-L", "runKit", "send-keys", "-t", "%5", "-l", "hello"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sendTextArgs(\"runKit\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("text with special characters is passed through verbatim", func(t *testing.T) {
		got := sendTextArgs("runKit", "%5", "echo $PATH | grep foo")
		// The text is the last element — no escaping expected; argv is not a shell.
		if got[len(got)-1] != "echo $PATH | grep foo" {
			t.Errorf("expected verbatim text, got %q", got[len(got)-1])
		}
	})
}

func TestSendEnterArgs(t *testing.T) {
	t.Run("empty server returns bare send-keys Enter argv", func(t *testing.T) {
		got := sendEnterArgs("", "%5")
		want := []string{"send-keys", "-t", "%5", "Enter"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sendEnterArgs(\"\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("non-empty server prepends -L <server>", func(t *testing.T) {
		got := sendEnterArgs("runKit", "%5")
		want := []string{"-L", "runKit", "send-keys", "-t", "%5", "Enter"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sendEnterArgs(\"runKit\", ...) = %v, want %v", got, want)
		}
	})
}

func TestPaneSendServerFlag(t *testing.T) {
	t.Run("--server flag inherited from pane parent", func(t *testing.T) {
		parent := paneCmd()
		var sub *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "send <pane> <text>" {
				sub = c
				break
			}
		}
		if sub == nil {
			t.Fatal("paneCmd did not register a send subcommand")
		}
		flag := sub.Flags().Lookup("server")
		if flag == nil {
			flag = sub.InheritedFlags().Lookup("server")
		}
		if flag == nil {
			t.Fatal("expected --server flag to be visible on pane send subcommand")
		}
		if flag.Shorthand != "L" {
			t.Errorf("expected shorthand \"L\", got %q", flag.Shorthand)
		}
	})
}
