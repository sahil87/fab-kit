package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
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

// TestIdleGate exercises the pure three-state send gate extracted from
// runPaneSend. It pins BOTH message contracts (the "not idle (state: <state>)"
// refusal and the distinct unknown/--force refusal) so a future reword of
// either message trips this test. This is the unit half of the A-014 coverage
// the review flagged as missing; TestPaneSendGate_Integration is the
// end-to-end half against a real tmux server.
func TestIdleGate(t *testing.T) {
	t.Run("idle permits the send", func(t *testing.T) {
		if err := idleGate("%5", strPtr(pane.AgentStateIdle)); err != nil {
			t.Errorf("idle should permit send, got error: %v", err)
		}
	})

	t.Run("active refuses with three-state-aware message", func(t *testing.T) {
		err := idleGate("%5", strPtr(pane.AgentStateActive))
		if err == nil {
			t.Fatal("active must refuse")
		}
		if err.Error() != "agent in pane %5 is not idle (state: active)" {
			t.Errorf("active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("waiting refuses with the same not-idle shape", func(t *testing.T) {
		err := idleGate("%5", strPtr(pane.AgentStateWaiting))
		if err == nil {
			t.Fatal("waiting must refuse")
		}
		if err.Error() != "agent in pane %5 is not idle (state: waiting)" {
			t.Errorf("waiting refusal message drifted: %q", err.Error())
		}
	})

	t.Run("unknown refuses with a distinct message naming --force", func(t *testing.T) {
		err := idleGate("%5", nil)
		if err == nil {
			t.Fatal("unknown must refuse")
		}
		msg := err.Error()
		// The unknown refusal must be DISTINCT from the not-idle shape and must
		// point the caller at --force.
		if strings.Contains(msg, "is not idle (state:") {
			t.Errorf("unknown refusal must not reuse the not-idle shape: %q", msg)
		}
		if !strings.Contains(msg, "--force") {
			t.Errorf("unknown refusal must name --force: %q", msg)
		}
		if !strings.Contains(msg, pane.AgentStateOption) {
			t.Errorf("unknown refusal should name the %s option: %q", pane.AgentStateOption, msg)
		}
	})
}

// tmuxSocketPathBudget is a conservative cap for the full tmux socket path
// ($TMUX_TMPDIR/tmux-$UID/<name>): macOS caps sun_path at 104 bytes
// including the terminating NUL.
const tmuxSocketPathBudget = 103

// tmuxSocketPathLen returns the length of the socket path tmux would bind
// for a server named name under TMUX_TMPDIR dir.
func tmuxSocketPathLen(dir, name string) int {
	return len(filepath.Join(dir, "tmux-"+strconv.Itoa(os.Getuid()), name))
}

// tmuxSocketDir returns a per-test private directory for TMUX_TMPDIR so the
// test's tmux socket dies with the test — tmux never unlinks its socket on
// server exit, so a socket in the shared /tmp/tmux-$UID would leak on every
// run (change 0j0t). Prefers t.TempDir(); when the resulting socket path
// would exceed the sun_path budget (long $TMPDIR bases on macOS), it falls
// back to a short /tmp dir removed via t.Cleanup — never a skip. Shared by
// the integration tests in this file and panemap_test.go.
func tmuxSocketDir(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	if tmuxSocketPathLen(dir, name) > tmuxSocketPathBudget {
		short, err := os.MkdirTemp("/tmp", "fabtest-")
		if err != nil {
			t.Fatalf("create short TMUX_TMPDIR fallback: %v", err)
		}
		t.Cleanup(func() { _ = os.RemoveAll(short) })
		dir = short
	}
	return dir
}

// TestTmuxSocketDirLengthGuard pins the sun_path length guard: the returned
// dir must always exist and always yield a socket path within the budget —
// including when the t.TempDir() candidate would blow it (the fallback
// branch), which a long server name forces deterministically.
func TestTmuxSocketDirLengthGuard(t *testing.T) {
	t.Run("short name fits the budget", func(t *testing.T) {
		name := "fabtest"
		dir := tmuxSocketDir(t, name)
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("returned dir must exist: %v", err)
		}
		if got := tmuxSocketPathLen(dir, name); got > tmuxSocketPathBudget {
			t.Errorf("socket path over budget: %d > %d (dir %q)", got, tmuxSocketPathBudget, dir)
		}
	})

	t.Run("over-budget candidate falls back to a short dir", func(t *testing.T) {
		// Long enough that any t.TempDir()-based candidate exceeds the budget,
		// short enough that the /tmp fallback still fits.
		name := strings.Repeat("n", tmuxSocketPathBudget-40)
		dir := tmuxSocketDir(t, name)
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("returned dir must exist: %v", err)
		}
		if got := tmuxSocketPathLen(dir, name); got > tmuxSocketPathBudget {
			t.Errorf("fallback did not fit the budget: %d > %d (dir %q)", got, tmuxSocketPathBudget, dir)
		}
	})
}

// TestPaneSendGate_Integration drives the full `fab pane send` command against
// a real ephemeral tmux server, simulating run-kit's rk agent-setup writer via
// `tmux set-option -p @rk_agent_state "<state>:<epoch>"` (the writer directed by
// the intake — the actual writer does not exist yet). This is the A-014
// coverage: it exercises the codex-pane scenario (a pane the old Claude-only
// _agents pipeline could never see) end-to-end, proving the gate refuses
// active/waiting/unknown, sends on idle, and that --force bypasses the gate.
// Skipped when tmux is unavailable.
func TestPaneSendGate_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}

	// Private TMUX_TMPDIR (process env — the command under test shells out to
	// `tmux -L` itself and must resolve the same socket dir) makes the socket
	// die with the test; a short fixed name keeps the path in budget.
	server := "fabtest-send"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
	if err != nil || paneID == "" {
		t.Fatalf("resolve pane id: %v (%q)", err, paneID)
	}

	setState := func(t *testing.T, state string, epoch int64) {
		t.Helper()
		val := state + ":" + strconv.FormatInt(epoch, 10)
		if out, err := tmux("set-option", "-p", "-t", paneID, pane.AgentStateOption, val); err != nil {
			t.Fatalf("set-option %s: %v: %s", val, err, out)
		}
	}
	unsetState := func(t *testing.T) {
		t.Helper()
		// -u removes the pane option, restoring the unknown (absent) case.
		if out, err := tmux("set-option", "-pu", "-t", paneID, pane.AgentStateOption); err != nil {
			t.Fatalf("unset-option: %v: %s", err, out)
		}
	}

	// runSend invokes the real command via cobra so the whole path
	// (ValidatePane → ResolvePaneContext → idleGate → send-keys) is exercised.
	// --no-enter avoids submitting a stray line into the pane's shell.
	runSend := func(args ...string) error {
		cmd := paneCmd()
		cmd.SetArgs(append([]string{"send", "-L", server, "--no-enter"}, args...))
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return cmd.Execute()
	}

	t.Run("active refuses (codex pane, previously invisible)", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		err := runSend(paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for active state")
		}
		if err.Error() != "agent in pane "+paneID+" is not idle (state: active)" {
			t.Errorf("active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("waiting refuses", func(t *testing.T) {
		setState(t, pane.AgentStateWaiting, 1751800000)
		err := runSend(paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for waiting state")
		}
		if err.Error() != "agent in pane "+paneID+" is not idle (state: waiting)" {
			t.Errorf("waiting refusal message drifted: %q", err.Error())
		}
	})

	t.Run("unknown (option unset) refuses distinctly", func(t *testing.T) {
		unsetState(t)
		err := runSend(paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for unknown state")
		}
		if !strings.Contains(err.Error(), "--force") {
			t.Errorf("unknown refusal must name --force: %q", err.Error())
		}
	})

	t.Run("idle sends", func(t *testing.T) {
		setState(t, pane.AgentStateIdle, time.Now().Unix())
		if err := runSend(paneID, "true"); err != nil {
			t.Errorf("idle should send, got error: %v", err)
		}
	})

	t.Run("--force bypasses the gate on a non-idle pane", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		if err := runSend("--force", paneID, "true"); err != nil {
			t.Errorf("--force should bypass the active gate, got error: %v", err)
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
