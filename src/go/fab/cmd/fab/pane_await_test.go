package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file covers `fab pane await` at the command layer against a real
// private-socket tmux server: the two signals, the unwaitable refusal, and
// the timeout report. The tick loop itself is unit-tested against scripted
// observers in internal/pane/await_test.go; the missing-pane / dead-socket /
// gone-mid-wait exit rows live in pane_exitcode_test.go's helper-driven
// table (the in-handler os.Exit cannot be observed in-process).
//
// The agent-state signal is simulated by writing the pane option directly —
// `tmux set-option -p @rk_agent_state "<state>:<epoch>"` — the writer
// simulation pane_send_test.go established.

// TestPaneAwait_IdleSignal: an instrumented pane whose state is already idle
// reports `idle` immediately (the first observation happens before any sleep);
// one flipping from active to idle mid-wait reports `idle` on a tick.
func TestPaneAwait_IdleSignal(t *testing.T) {
	server := "fabtest-paneawait-idle"
	tmux, paneID := newTmuxPane(t, server, "", 80)
	setState := func(state string) {
		t.Helper()
		val := state + ":" + strconv.FormatInt(time.Now().Unix(), 10)
		if out, err := tmux("set-option", "-p", "-t", paneID, pane.AgentStateOption, val); err != nil {
			t.Fatalf("set-option %s: %v: %s", val, err, out)
		}
	}

	t.Run("already idle returns immediately", func(t *testing.T) {
		setState(pane.AgentStateIdle)
		start := time.Now()
		stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server)
		if err != nil {
			t.Fatalf("await: %v", err)
		}
		if stdout != string(pane.AwaitIdle)+"\n" {
			t.Errorf("report = %q, want %q", stdout, string(pane.AwaitIdle)+"\n")
		}
		if elapsed := time.Since(start); elapsed > pane.AwaitTick {
			t.Errorf("took %v — an already-idle pane must not consume a tick", elapsed)
		}
	})

	t.Run("active flipping to idle mid-wait", func(t *testing.T) {
		setState(pane.AgentStateActive)
		go func() {
			time.Sleep(3 * pane.AwaitTick)
			val := pane.AgentStateIdle + ":" + strconv.FormatInt(time.Now().Unix(), 10)
			_, _ = tmux("set-option", "-p", "-t", paneID, pane.AgentStateOption, val)
		}()
		stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server, "--timeout", "60")
		if err != nil {
			t.Fatalf("await: %v", err)
		}
		if stdout != string(pane.AwaitIdle)+"\n" {
			t.Errorf("report = %q, want %q", stdout, string(pane.AwaitIdle)+"\n")
		}
	})
}

// TestPaneAwait_FileSignal: an uninstrumented pane with --file blocks until the
// contract file appears, then reports `file` — the OR-composition's other half.
func TestPaneAwait_FileSignal(t *testing.T) {
	server := "fabtest-paneawait-file"
	_, paneID := newTmuxPane(t, server, "", 80)

	t.Run("existing file returns immediately", func(t *testing.T) {
		existing := filepath.Join(t.TempDir(), "result.yaml")
		if err := os.WriteFile(existing, []byte("ok: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server, "--file", existing)
		if err != nil {
			t.Fatalf("await: %v", err)
		}
		if stdout != string(pane.AwaitFile)+"\n" {
			t.Errorf("report = %q, want %q", stdout, string(pane.AwaitFile)+"\n")
		}
		if elapsed := time.Since(start); elapsed > pane.AwaitTick {
			t.Errorf("took %v — an existing file must not consume a tick", elapsed)
		}
	})

	t.Run("file appearing mid-wait", func(t *testing.T) {
		pending := filepath.Join(t.TempDir(), "result.yaml")
		go func() {
			time.Sleep(3 * pane.AwaitTick)
			_ = os.WriteFile(pending, []byte("ok: true\n"), 0o644)
		}()
		stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server, "--file", pending, "--timeout", "60")
		if err != nil {
			t.Fatalf("await: %v", err)
		}
		if stdout != string(pane.AwaitFile)+"\n" {
			t.Errorf("report = %q, want %q", stdout, string(pane.AwaitFile)+"\n")
		}
	})
}

// TestPaneAwait_Unwaitable: an uninstrumented pane with no --file is an
// immediate RunE error — blocking to timeout would report `running` while
// meaning "I was never watching anything".
func TestPaneAwait_Unwaitable(t *testing.T) {
	server := "fabtest-paneawait-nowatch"
	_, paneID := newTmuxPane(t, server, "", 80)

	start := time.Now()
	_, _, err := runPaneCmd(t, "await", paneID, "-L", server)
	if err == nil {
		t.Fatal("an uninstrumented pane with no --file must fail")
	}
	if !strings.Contains(err.Error(), "nothing observable to wait on") {
		t.Errorf("error = %q, want the unwaitable-pane message", err)
	}
	if elapsed := time.Since(start); elapsed > pane.AwaitTick {
		t.Errorf("took %v — the unwaitable case must error immediately", elapsed)
	}
}

// TestPaneAwait_BothSignalsArmed pins the OR composition: with --file AND an
// instrumented pane both armed, whichever fires first wins — here the file
// appears while the state stays active, and the report is `file`.
func TestPaneAwait_BothSignalsArmed(t *testing.T) {
	server := "fabtest-paneawait-or"
	tmux, paneID := newTmuxPane(t, server, "", 80)
	val := pane.AgentStateActive + ":" + strconv.FormatInt(time.Now().Unix(), 10)
	if out, err := tmux("set-option", "-p", "-t", paneID, pane.AgentStateOption, val); err != nil {
		t.Fatalf("set-option %s: %v: %s", val, err, out)
	}

	pending := filepath.Join(t.TempDir(), "result.yaml")
	go func() {
		time.Sleep(3 * pane.AwaitTick)
		_ = os.WriteFile(pending, []byte("ok: true\n"), 0o644)
	}()
	stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server, "--file", pending, "--timeout", "60")
	if err != nil {
		t.Fatalf("await: %v", err)
	}
	if stdout != string(pane.AwaitFile)+"\n" {
		t.Errorf("report = %q, want %q (the file fired while the state stayed active)", stdout, string(pane.AwaitFile)+"\n")
	}
}

// TestPaneAwait_GoneMidWait drives the pane-death path through a child process
// — the in-handler os.Exit(2) for `gone` cannot be observed in-process. The
// child awaits a never-appearing file; the parent kills the pane mid-wait, and
// the next tick must report `gone` with exit 2.
func TestPaneAwait_GoneMidWait(t *testing.T) {
	if os.Getenv("FAB_PANE_AWAIT_GONE_HELPER") == "1" {
		args := os.Args
		for i, a := range args {
			if a == "--" {
				args = args[i+1:]
				break
			}
		}
		cmd := paneCmd()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	server := "fabtest-paneawait-gone"
	tmux, paneID := newTmuxPane(t, server, "", 80)

	never := filepath.Join(t.TempDir(), "never.yaml")
	child := exec.Command(os.Args[0], "-test.run=^TestPaneAwait_GoneMidWait$", "--",
		"await", paneID, "-L", server, "--file", never, "--timeout", "120")
	child.Env = append(os.Environ(), "FAB_PANE_AWAIT_GONE_HELPER=1")
	var stdout bytes.Buffer
	child.Stdout = &stdout
	if err := child.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}

	// Let the child validate and settle into the wait loop, then kill the pane.
	time.Sleep(2 * pane.AwaitTick)
	if out, err := tmux("kill-pane", "-t", paneID); err != nil {
		t.Fatalf("kill-pane: %v: %s", err, out)
	}

	err := child.Wait()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 2 {
		t.Fatalf("exit = %v, want 2 (stdout: %q)", err, stdout.String())
	}
	if stdout.String() != string(pane.AwaitGone)+"\n" {
		t.Errorf("report = %q, want %q", stdout.String(), string(pane.AwaitGone)+"\n")
	}
}

// TestPaneAwait_Timeout: expiry reports `running` and is NOT an error — the
// timeout bounds the observer, never the pane (the dispatch wait precedent).
func TestPaneAwait_Timeout(t *testing.T) {
	server := "fabtest-paneawait-tmo"
	_, paneID := newTmuxPane(t, server, "", 80)

	stdout, _, err := runPaneCmd(t, "await", paneID, "-L", server, "--file", filepath.Join(t.TempDir(), "never.yaml"), "--timeout", "1")
	if err != nil {
		t.Fatalf("a timeout is not an error: %v", err)
	}
	if stdout != string(pane.AwaitRunning)+"\n" {
		t.Errorf("report = %q, want %q", stdout, string(pane.AwaitRunning)+"\n")
	}
}
