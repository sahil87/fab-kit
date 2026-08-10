package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
)

// runPaneCmd executes a `fab pane` subcommand through the real parent (so the
// persistent --server/-L flag resolves), returning stdout, stderr, and the
// RunE error.
func runPaneCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := paneCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

// stubPaneTmux writes a fake `tmux` executable (a POSIX sh body) into a temp
// dir prepended to $PATH, so the spawn path can be exercised in-process
// without a tmux server — the stubBatchNewBinaries precedent.
func stubPaneTmux(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// paneOpenStubTmux stubs the two calls `fab pane open` makes: list-sessions
// (the reachability probe) answers successfully, and both creators print a
// pane id. Every argv is appended to $ARGLOG for shape assertions. The -L
// prefix (when --server is given) is shifted off before dispatch.
func paneOpenStubTmux(t *testing.T) string {
	t.Helper()
	argLog := filepath.Join(t.TempDir(), "argv.log")
	t.Setenv("ARGLOG", argLog)
	stubPaneTmux(t, `echo "$@" >> "$ARGLOG"
if [ "$1" = "-L" ]; then shift 2; fi
case "$1" in
list-sessions) exit 0 ;;
new-window) echo "%42" ;;
split-window) echo "%43" ;;
esac
exit 0`)
	return argLog
}

func TestPaneOpenCmd(t *testing.T) {
	t.Run("requires --provider", func(t *testing.T) {
		cmd := paneOpenCmd()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		if err := cmd.Execute(); err == nil {
			t.Fatal("expected a usage error without --provider, got nil")
		}
	})

	t.Run("role defaults to the default role", func(t *testing.T) {
		cmd := paneOpenCmd()
		role, _ := cmd.Flags().GetString("role")
		if role != agent.RoleDefault {
			t.Errorf("role default = %q, want %q", role, agent.RoleDefault)
		}
	})

	t.Run("cwd flag with -c shorthand", func(t *testing.T) {
		cmd := paneOpenCmd()
		flag := cmd.Flags().Lookup("cwd")
		if flag == nil {
			t.Fatal("expected --cwd flag to exist")
		}
		if flag.Shorthand != "c" {
			t.Errorf("expected shorthand %q, got %q", "c", flag.Shorthand)
		}
	})

	t.Run("server flag inherited from pane parent", func(t *testing.T) {
		parent := paneCmd()
		for _, c := range parent.Commands() {
			if strings.HasPrefix(c.Use, "open ") {
				if c.InheritedFlags().Lookup("server") == nil {
					t.Error("expected --server to be visible on pane open subcommand")
				}
				return
			}
		}
		t.Fatal("paneCmd did not register an open subcommand")
	})
}

// TestPaneOpen_ResolutionErrors pins the two exit-1 resolution failures: an
// unknown provider is the shared lookup failure naming the available providers,
// and a provider with no interactive_command is a hard error naming the
// provider. Both are RunE errors raised BEFORE any tmux call, so neither needs
// a server. The unknown-provider case runs outside a fab repo; the
// missing-interactive case creates a project-defined headless-only provider.
func TestPaneOpen_ResolutionErrors(t *testing.T) {
	t.Run("unknown provider names the available providers", func(t *testing.T) {
		chdirTestEnv(t, t.TempDir(), nil)
		_, _, err := runPaneCmd(t, "open", "--provider", "nosuch")
		if err == nil {
			t.Fatal("an unknown provider must fail")
		}
		if !strings.Contains(err.Error(), `unknown provider "nosuch"`) || !strings.Contains(err.Error(), "available:") {
			t.Errorf("error = %q, want the shared unknown-provider lookup failure", err)
		}
	})

	t.Run("provider without interactive_command is a hard error", func(t *testing.T) {
		agentTestRepo(t, `project:
  name: test
providers:
  myagent:
    headless_command: "myagent run"
`)
		_, _, err := runPaneCmd(t, "open", "--provider", "myagent")
		if err == nil {
			t.Fatal("a provider without interactive_command must fail")
		}
		want := `provider "myagent" has no interactive_command; configure providers.myagent.interactive_command`
		if err.Error() != want {
			t.Errorf("error = %q, want exactly %q", err, want)
		}
	})
}

func TestPaneOpen_BuiltInAgyIsPaneCapable(t *testing.T) {
	argLog := paneOpenStubTmux(t)
	chdirTestEnv(t, t.TempDir(), map[string]string{"TMUX": "", "TMUX_PANE": ""})

	out, _, err := runPaneCmd(t, "open", "--provider", "agy")
	if err != nil {
		t.Fatalf("pane open --provider agy: %v", err)
	}
	if out != "opened pane %42 (provider agy)\n" {
		t.Errorf("output = %q, want built-in agy pane-open success", out)
	}
	data, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "agy --dangerously-skip-permissions --model gemini-3.1-pro-high") {
		t.Errorf("tmux argv = %q, want resolved built-in agy interactive grammar", data)
	}
}

// TestPaneOpen_SpawnShapes drives the spawn end to end against a stub tmux:
// the reachability probe answers, the creator prints a pane id, and every argv
// is logged. Pins the split-vs-window decision, the output lines, and the
// no-dispatch-record guarantee.
func TestPaneOpen_SpawnShapes(t *testing.T) {
	readLog := func(t *testing.T, argLog string) []string {
		t.Helper()
		data, err := os.ReadFile(argLog)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}

	t.Run("outside tmux opens an unnamed window", func(t *testing.T) {
		argLog := paneOpenStubTmux(t)
		dir := t.TempDir()
		chdirTestEnv(t, dir, map[string]string{"TMUX_PANE": ""})

		stdout, _, err := runPaneCmd(t, "open", "--provider", "kimi")
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if stdout != "opened pane %42 (provider kimi)\n" {
			t.Errorf("stdout = %q, want the pane id line", stdout)
		}
		calls := readLog(t, argLog)
		if len(calls) != 2 || !strings.HasPrefix(calls[0], "list-sessions") {
			t.Fatalf("tmux calls = %v, want the reachability probe then the spawn", calls)
		}
		spawn := calls[1]
		if !strings.HasPrefix(spawn, "new-window ") {
			t.Errorf("spawn = %q, want an unnamed new-window", spawn)
		}
		for _, frag := range []string{"-P", "-F", "#{pane_id}", "-c " + dir, "kimi --auto"} {
			if !strings.Contains(spawn, frag) {
				t.Errorf("spawn = %q, want it to carry %q", spawn, frag)
			}
		}
		if strings.Contains(spawn, " -n ") {
			t.Errorf("spawn = %q, want NO window name (titling is dispatch's convention)", spawn)
		}
		if _, err := os.Stat(filepath.Join(dir, ".fab-dispatch")); !os.IsNotExist(err) {
			t.Errorf("fab pane open must write no .fab-dispatch state")
		}
	})

	t.Run("inside a tmux pane spawns a plain split", func(t *testing.T) {
		argLog := paneOpenStubTmux(t)
		dir := t.TempDir()
		chdirTestEnv(t, dir, map[string]string{"TMUX_PANE": "%7"})

		stdout, _, err := runPaneCmd(t, "open", "--provider", "kimi")
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if stdout != "opened pane %43 (provider kimi)\n" {
			t.Errorf("stdout = %q, want the pane id line", stdout)
		}
		calls := readLog(t, argLog)
		spawn := calls[len(calls)-1]
		if !strings.HasPrefix(spawn, "split-window ") {
			t.Errorf("spawn = %q, want a plain split of the current window", spawn)
		}
		if strings.Contains(spawn, " -t ") || strings.Contains(spawn, " -l ") {
			t.Errorf("spawn = %q, want no target/size (placement is dispatch policy)", spawn)
		}
	})

	t.Run("--server forces a window and prints the socket", func(t *testing.T) {
		argLog := paneOpenStubTmux(t)
		chdirTestEnv(t, t.TempDir(), map[string]string{"TMUX_PANE": "%7"})

		stdout, _, err := runPaneCmd(t, "open", "--provider", "kimi", "--server", "work")
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		want := "opened pane %42 (provider kimi)\nserver: work\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
		calls := readLog(t, argLog)
		spawn := calls[len(calls)-1]
		if !strings.HasPrefix(spawn, "-L work new-window ") {
			t.Errorf("spawn = %q, want a socket-scoped unnamed window even from inside tmux", spawn)
		}
	})
}
