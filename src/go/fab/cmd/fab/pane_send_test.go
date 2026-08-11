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

	t.Run("answer flag defaults to false", func(t *testing.T) {
		cmd := paneSendCmd()
		answer, _ := cmd.Flags().GetBool("answer")
		if answer {
			t.Error("expected answer to default to false")
		}
	})

	t.Run("flag existence", func(t *testing.T) {
		cmd := paneSendCmd()

		noEnterFlag := cmd.Flags().Lookup("no-enter")
		if noEnterFlag == nil {
			t.Error("expected 'no-enter' flag to exist")
		}

		answerFlag := cmd.Flags().Lookup("answer")
		if answerFlag == nil {
			t.Error("expected 'answer' flag to exist")
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

// TestIdleGate exercises the pure send-gate matrix extracted from runPaneSend
// (four states × plain/--answer; --force never reaches the gate). It pins ALL
// message contracts — the plain "not idle (state: <state>)" refusal, the
// --answer active refusal, and the unknown-state warn-and-proceed warning — so
// a future reword of any trips this test. This is the unit half of the
// coverage; TestPaneSendGate_Integration is the end-to-end half against a real
// tmux server.
func TestIdleGate(t *testing.T) {
	t.Run("idle permits the send without a warning", func(t *testing.T) {
		warning, err := idleGate("%5", strPtr(pane.AgentStateIdle), false)
		if err != nil {
			t.Errorf("idle should permit send, got error: %v", err)
		}
		if warning != "" {
			t.Errorf("idle should not warn, got %q", warning)
		}
	})

	t.Run("active refuses with three-state-aware message", func(t *testing.T) {
		_, err := idleGate("%5", strPtr(pane.AgentStateActive), false)
		if err == nil {
			t.Fatal("active must refuse")
		}
		if err.Error() != "agent in pane %5 is not idle (state: active)" {
			t.Errorf("active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("waiting refuses with the same not-idle shape", func(t *testing.T) {
		_, err := idleGate("%5", strPtr(pane.AgentStateWaiting), false)
		if err == nil {
			t.Fatal("waiting must refuse")
		}
		if err.Error() != "agent in pane %5 is not idle (state: waiting)" {
			t.Errorf("waiting refusal message drifted: %q", err.Error())
		}
	})

	t.Run("unknown warns and proceeds", func(t *testing.T) {
		warning, err := idleGate("%5", nil, false)
		if err != nil {
			t.Fatalf("unknown must not refuse: %v", err)
		}
		if warning != "agent state unknown — sending anyway" {
			t.Errorf("unknown warning drifted: %q", warning)
		}
	})

	t.Run("answer: idle permits the send without a warning", func(t *testing.T) {
		warning, err := idleGate("%5", strPtr(pane.AgentStateIdle), true)
		if err != nil {
			t.Errorf("idle should permit send under --answer, got error: %v", err)
		}
		if warning != "" {
			t.Errorf("idle should not warn under --answer, got %q", warning)
		}
	})

	t.Run("answer: waiting permits the send — the primary --answer case", func(t *testing.T) {
		warning, err := idleGate("%5", strPtr(pane.AgentStateWaiting), true)
		if err != nil {
			t.Errorf("waiting must send under --answer, got error: %v", err)
		}
		if warning != "" {
			t.Errorf("waiting should not warn under --answer, got %q", warning)
		}
	})

	t.Run("answer: active still refuses, state-naming message", func(t *testing.T) {
		_, err := idleGate("%5", strPtr(pane.AgentStateActive), true)
		if err == nil {
			t.Fatal("active must refuse under --answer")
		}
		if err.Error() != "agent in pane %5 is active (--answer permits idle and waiting only)" {
			t.Errorf("--answer active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("answer: unknown warns and proceeds (same posture as plain send)", func(t *testing.T) {
		warning, err := idleGate("%5", nil, true)
		if err != nil {
			t.Fatalf("unknown must not refuse under --answer: %v", err)
		}
		if warning != "agent state unknown — sending anyway" {
			t.Errorf("unknown warning drifted under --answer: %q", warning)
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
// the intake — the actual writer does not exist yet). This is the end-to-end
// coverage: it exercises the codex-pane scenario (a pane the old Claude-only
// _agents pipeline could never see) end-to-end, proving the gate refuses
// active/waiting, WARNS and sends on unknown (the foreign-pane posture), sends
// on idle, and that --force bypasses the gate. Skipped when tmux is unavailable.
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
	// --no-enter avoids submitting a stray line into the pane's shell. stderr is
	// captured so the unknown-state warning is assertable.
	runSend := func(args ...string) (string, error) {
		var errBuf strings.Builder
		cmd := paneCmd()
		cmd.SetArgs(append([]string{"send", "-L", server, "--no-enter"}, args...))
		cmd.SetErr(&errBuf)
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		err := cmd.Execute()
		return errBuf.String(), err
	}

	t.Run("active refuses (codex pane, previously invisible)", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		_, err := runSend(paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for active state")
		}
		if err.Error() != "agent in pane "+paneID+" is not idle (state: active)" {
			t.Errorf("active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("waiting refuses", func(t *testing.T) {
		setState(t, pane.AgentStateWaiting, 1751800000)
		_, err := runSend(paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for waiting state")
		}
		if err.Error() != "agent in pane "+paneID+" is not idle (state: waiting)" {
			t.Errorf("waiting refusal message drifted: %q", err.Error())
		}
	})

	t.Run("unknown (option unset) warns and sends", func(t *testing.T) {
		unsetState(t)
		stderr, err := runSend(paneID, "hi")
		if err != nil {
			t.Fatalf("unknown must warn and send, got error: %v", err)
		}
		if !strings.Contains(stderr, "warning: agent state unknown — sending anyway") {
			t.Errorf("stderr = %q, want the unknown-state warning", stderr)
		}
	})

	t.Run("idle sends without a warning", func(t *testing.T) {
		setState(t, pane.AgentStateIdle, time.Now().Unix())
		stderr, err := runSend(paneID, "true")
		if err != nil {
			t.Errorf("idle should send, got error: %v", err)
		}
		if stderr != "" {
			t.Errorf("idle should not warn, got stderr %q", stderr)
		}
	})

	t.Run("--force bypasses the gate on a non-idle pane", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		if _, err := runSend("--force", paneID, "true"); err != nil {
			t.Errorf("--force should bypass the active gate, got error: %v", err)
		}
	})

	t.Run("--answer sends to a waiting pane (the operator auto-answer case)", func(t *testing.T) {
		setState(t, pane.AgentStateWaiting, 1751800000)
		stderr, err := runSend("--answer", paneID, "true")
		if err != nil {
			t.Errorf("--answer should send to waiting, got error: %v", err)
		}
		if stderr != "" {
			t.Errorf("--answer to waiting should not warn, got stderr %q", stderr)
		}
	})

	t.Run("--answer still refuses an active pane", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		_, err := runSend("--answer", paneID, "hi")
		if err == nil {
			t.Fatal("expected refusal for active state under --answer")
		}
		if err.Error() != "agent in pane "+paneID+" is active (--answer permits idle and waiting only)" {
			t.Errorf("--answer active refusal message drifted: %q", err.Error())
		}
	})

	t.Run("--answer on unknown state warns and sends (posture parity with plain send)", func(t *testing.T) {
		unsetState(t)
		stderr, err := runSend("--answer", paneID, "true")
		if err != nil {
			t.Fatalf("--answer on unknown must warn and send, got error: %v", err)
		}
		if !strings.Contains(stderr, "warning: agent state unknown — sending anyway") {
			t.Errorf("stderr = %q, want the unknown-state warning", stderr)
		}
	})

	t.Run("--answer --force together behaves as --force (state check skipped)", func(t *testing.T) {
		setState(t, pane.AgentStateActive, 1751800000)
		if _, err := runSend("--answer", "--force", paneID, "true"); err != nil {
			t.Errorf("--force must win over --answer on an active pane, got error: %v", err)
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
