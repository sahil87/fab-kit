package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPaneWindowNameCmd(t *testing.T) {
	t.Run("registered on pane parent", func(t *testing.T) {
		parent := paneCmd()
		var sub *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "window-name" {
				sub = c
				break
			}
		}
		if sub == nil {
			t.Fatal("paneCmd did not register a window-name subcommand")
		}
	})

	t.Run("group has exactly two children", func(t *testing.T) {
		group := paneWindowNameCmd()
		children := group.Commands()
		if len(children) != 2 {
			t.Fatalf("expected 2 child commands, got %d", len(children))
		}
		names := map[string]bool{}
		for _, c := range children {
			names[strings.Fields(c.Use)[0]] = true
		}
		if !names["ensure-prefix"] {
			t.Error("ensure-prefix subcommand not registered")
		}
		if !names["replace-prefix"] {
			t.Error("replace-prefix subcommand not registered")
		}
	})

	t.Run("ensure-prefix requires exactly two args", func(t *testing.T) {
		cmd := paneWindowNameEnsurePrefixCmd()

		// Zero args
		cmd.SetArgs([]string{})
		if err := cmd.Args(cmd, []string{}); err == nil {
			t.Error("expected error for 0 args")
		}

		// One arg
		if err := cmd.Args(cmd, []string{"%5"}); err == nil {
			t.Error("expected error for 1 arg")
		}

		// Three args
		if err := cmd.Args(cmd, []string{"%5", "»", "extra"}); err == nil {
			t.Error("expected error for 3 args")
		}

		// Two args (valid)
		if err := cmd.Args(cmd, []string{"%5", "»"}); err != nil {
			t.Errorf("expected no error for 2 args, got %s", err)
		}
	})

	t.Run("replace-prefix requires exactly three args", func(t *testing.T) {
		cmd := paneWindowNameReplacePrefixCmd()

		if err := cmd.Args(cmd, []string{"%5", "»"}); err == nil {
			t.Error("expected error for 2 args")
		}
		if err := cmd.Args(cmd, []string{"%5", "»", "›", "extra"}); err == nil {
			t.Error("expected error for 4 args")
		}
		if err := cmd.Args(cmd, []string{"%5", "»", "›"}); err != nil {
			t.Errorf("expected no error for 3 args, got %s", err)
		}
	})

	t.Run("--json flag exists on both subcommands", func(t *testing.T) {
		for _, sub := range []*cobra.Command{paneWindowNameEnsurePrefixCmd(), paneWindowNameReplacePrefixCmd()} {
			flag := sub.Flags().Lookup("json")
			if flag == nil {
				t.Errorf("%s: expected --json flag", sub.Use)
				continue
			}
			if flag.Value.String() != "false" {
				t.Errorf("%s: expected --json to default to false, got %s", sub.Use, flag.Value.String())
			}
		}
	})

	t.Run("--server flag inherited from pane parent", func(t *testing.T) {
		parent := paneCmd()
		var group *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "window-name" {
				group = c
				break
			}
		}
		if group == nil {
			t.Fatal("window-name group not found under paneCmd")
		}
		for _, sub := range group.Commands() {
			flag := sub.Flags().Lookup("server")
			if flag == nil {
				flag = sub.InheritedFlags().Lookup("server")
			}
			if flag == nil {
				t.Errorf("%s: expected --server flag (inherited from pane parent)", sub.Use)
				continue
			}
			if flag.Shorthand != "L" {
				t.Errorf("%s: expected --server shorthand \"L\", got %q", sub.Use, flag.Shorthand)
			}
		}
	})
}

func TestRenameArgs(t *testing.T) {
	t.Run("empty server returns bare rename-window argv", func(t *testing.T) {
		got := renameArgs("", "%5", "»work")
		want := []string{"rename-window", "-t", "%5", "»work"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("renameArgs(\"\", ...) = %v, want %v", got, want)
		}
		for _, el := range got {
			if el == "-L" {
				t.Errorf("did not expect -L in argv for empty server, got %v", got)
			}
		}
	})

	t.Run("non-empty server prepends -L <server>", func(t *testing.T) {
		got := renameArgs("runKit", "%5", "»work")
		want := []string{"-L", "runKit", "rename-window", "-t", "%5", "»work"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("renameArgs(\"runKit\", ...) = %v, want %v", got, want)
		}
	})

	t.Run("new name with special characters is passed through verbatim", func(t *testing.T) {
		got := renameArgs("", "%5", "›spec-work with spaces")
		if got[len(got)-1] != "›spec-work with spaces" {
			t.Errorf("expected verbatim new-name, got %q", got[len(got)-1])
		}
	})

	t.Run("multi-codepoint prefix yields multi-codepoint new name", func(t *testing.T) {
		got := renameArgs("", "%5", "** work")
		if got[len(got)-1] != "** work" {
			t.Errorf("expected \"** work\", got %q", got[len(got)-1])
		}
	})
}

func TestEmitResult(t *testing.T) {
	t.Run("plain rename format", func(t *testing.T) {
		var buf bytes.Buffer
		emitResult(&buf, "%5", "work", "»work", "renamed", false)
		want := "renamed: work -> »work\n"
		if buf.String() != want {
			t.Errorf("emitResult plain rename = %q, want %q", buf.String(), want)
		}
	})

	t.Run("plain noop is empty", func(t *testing.T) {
		var buf bytes.Buffer
		emitResult(&buf, "%5", "»work", "»work", "noop", false)
		if buf.String() != "" {
			t.Errorf("emitResult plain noop = %q, want empty", buf.String())
		}
	})

	t.Run("JSON rename shape", func(t *testing.T) {
		var buf bytes.Buffer
		emitResult(&buf, "%5", "work", "»work", "renamed", true)
		var got windowNameResult
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("JSON parse failed: %s (raw: %s)", err, buf.String())
		}
		want := windowNameResult{Pane: "%5", Old: "work", New: "»work", Action: "renamed"}
		if got != want {
			t.Errorf("emitResult JSON rename = %+v, want %+v", got, want)
		}
	})

	t.Run("JSON noop shape", func(t *testing.T) {
		var buf bytes.Buffer
		emitResult(&buf, "%5", "»work", "»work", "noop", true)
		var got windowNameResult
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("JSON parse failed: %s (raw: %s)", err, buf.String())
		}
		want := windowNameResult{Pane: "%5", Old: "»work", New: "»work", Action: "noop"}
		if got != want {
			t.Errorf("emitResult JSON noop = %+v, want %+v", got, want)
		}
	})

	t.Run("JSON output ends with a newline", func(t *testing.T) {
		var buf bytes.Buffer
		emitResult(&buf, "%5", "w", "»w", "renamed", true)
		s := buf.String()
		if len(s) == 0 || s[len(s)-1] != '\n' {
			t.Errorf("expected trailing newline in JSON output, got %q", s)
		}
	})
}

func TestPaneCmdListsWindowName(t *testing.T) {
	// The parent's Long description should reference window-name alongside the others.
	parent := paneCmd()
	if !strings.Contains(parent.Long, "window-name") {
		t.Errorf("expected paneCmd.Long to mention window-name, got %q", parent.Long)
	}
}
