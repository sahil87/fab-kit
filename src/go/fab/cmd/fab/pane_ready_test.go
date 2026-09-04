package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file covers `fab pane ready`'s REPORT — the pane-addressed twin of
// dispatch_ready_test.go, sharing its parked/booting pane commands. The
// classification itself is unit-tested against scripted captures in
// internal/pane/gate_test.go; what is pinned here is the report form (dispatch
// parity) and that answering at all is a success.

// TestPaneReady_ReadyReport: a live pane running a non-shell foreground
// (readyPaneCommand) echoes the sentinel, so the report is exactly the
// classification line and nothing else.
func TestPaneReady_ReadyReport(t *testing.T) {
	server := "fabtest-paneready-ok"
	_, paneID := newTmuxPane(t, server, readyPaneCommand, 80)

	stdout, _, err := runPaneCmd(t, "ready", paneID, "-L", server)
	if err != nil {
		t.Fatalf("a classification is never an error: %v", err)
	}
	if stdout != string(pane.ReadyReady)+"\n" {
		t.Errorf("report = %q, want %q from the live shell", stdout, string(pane.ReadyReady)+"\n")
	}
}

// TestPaneReady_NonReadyReports pins both non-`ready` answers end to end: the
// classification line, the pane/socket lines, the capture snippet under the
// evidence header, and exit 0 — the dispatch-ready report form addressed by
// pane id.
func TestPaneReady_NonReadyReports(t *testing.T) {
	for _, tt := range []struct {
		name    string
		command string
		want    pane.Readiness
		screen  string
	}{
		{"parked behind a wall", parkedPaneCommand, pane.ReadyParked, "TRUST-THIS-FOLDER-WALL"},
		{"still booting", bootingPaneCommand, pane.ReadyBooting, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := "fabtest-paneready-" + string(tt.want)
			_, paneID := newTmuxPane(t, server, tt.command, 80)

			stdout, _, err := runPaneCmd(t, "ready", paneID, "-L", server)
			if err != nil {
				t.Fatalf("a classification is never an error: %v", err)
			}

			lines := strings.SplitN(stdout, "\n", 2)
			if lines[0] != string(tt.want) {
				t.Fatalf("report = %q, want it to open with %q", stdout, tt.want)
			}
			wants := []string{"pane: " + paneID, "server: " + server}
			if tt.screen != "" {
				wants = append(wants, fmt.Sprintf("--- last %d lines ---", pane.SnippetLines), tt.screen)
			}
			for _, want := range wants {
				if !strings.Contains(stdout, want) {
					t.Errorf("report = %q, want it to carry %q", stdout, want)
				}
			}
			if tt.screen == "" && strings.Contains(stdout, "--- last") {
				t.Errorf("report = %q, want no evidence header above an empty snippet", stdout)
			}
			if !strings.HasSuffix(stdout, "\n") {
				t.Errorf("report = %q, want it to end with a newline", stdout)
			}
		})
	}
}

// TestPaneReady_JSON pins the --json shape for all three classifications —
// always exactly one object (the window-name precedent), state/pane/server/
// snippet, snippet "" on a blank screen, server null on the default socket —
// and that every classification still exits 0 (the report object is the sole
// discriminator).
func TestPaneReady_JSON(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		server := "fabtest-paneready-json-ok"
		_, paneID := newTmuxPane(t, server, readyPaneCommand, 80)

		stdout, _, err := runPaneCmd(t, "ready", paneID, "-L", server, "--json")
		if err != nil {
			t.Fatalf("a classification is never an error: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if got["state"] != string(pane.ReadyReady) || got["pane"] != paneID || got["snippet"] != "" {
			t.Errorf("json = %v", got)
		}
		if got["server"] != server {
			t.Errorf("server = %v, want %q", got["server"], server)
		}
	})

	t.Run("parked carries the snippet", func(t *testing.T) {
		server := "fabtest-paneready-json-parked"
		_, paneID := newTmuxPane(t, server, parkedPaneCommand, 80)

		stdout, _, err := runPaneCmd(t, "ready", paneID, "-L", server, "--json")
		if err != nil {
			t.Fatalf("a classification is never an error: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if got["state"] != string(pane.ReadyParked) {
			t.Errorf("state = %v, want parked", got["state"])
		}
		snippet, _ := got["snippet"].(string)
		if !strings.Contains(snippet, "TRUST-THIS-FOLDER-WALL") {
			t.Errorf("snippet = %q, want the refusing screen", snippet)
		}
	})

	t.Run("booting carries a blank snippet", func(t *testing.T) {
		server := "fabtest-paneready-json-boot"
		_, paneID := newTmuxPane(t, server, bootingPaneCommand, 80)

		stdout, _, err := runPaneCmd(t, "ready", paneID, "-L", server, "--json")
		if err != nil {
			t.Fatalf("a classification is never an error: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if got["state"] != string(pane.ReadyBooting) {
			t.Errorf("state = %v, want booting", got["state"])
		}
		if got["pane"] != paneID || got["server"] != server {
			t.Errorf("json = %v", got)
		}
		if got["snippet"] != "" {
			t.Errorf("snippet = %q, want \"\" on a blank booting screen", got["snippet"])
		}
	})

	t.Run("server null on the default socket", func(t *testing.T) {
		// No -L anywhere: the server lives on the DEFAULT socket under a private
		// TMUX_TMPDIR, so the empty --server value exercises the toNullable null
		// mapping instead of the socket-label path.
		if _, err := exec.LookPath("tmux"); err != nil {
			t.Skip("tmux not available")
		}
		socketDir := tmuxSocketDir(t, "default")
		t.Setenv("TMUX_TMPDIR", socketDir)
		// $TMUX outranks TMUX_TMPDIR in tmux's socket resolution (-L/-S > $TMUX
		// > TMUX_TMPDIR): run from inside a tmux pane, an inherited $TMUX would
		// land the bare new-session below on the HOST server — and the
		// kill-server cleanup would kill the host. Scrub it at process level so
		// both this test's tmux calls and the command under test (which shells
		// out to tmux itself) resolve through the private TMUX_TMPDIR.
		// Empirically (tmux 3.6a) an empty $TMUX is treated as unset, so
		// t.Setenv suffices. tmuxSocketDir also scrubs — this restates the
		// guard at the one site that starts a bare default-socket server.
		t.Setenv("TMUX", "")
		t.Setenv("TMUX_PANE", "")
		tmux := func(args ...string) (string, error) {
			out, err := exec.Command("tmux", args...).CombinedOutput()
			return strings.TrimSpace(string(out)), err
		}
		if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
			t.Skipf("could not start tmux server (%v): %s", err, out)
		}
		// Prove the server bound the PRIVATE socket before registering a
		// destructive cleanup, and scope that cleanup to the verified path with
		// an explicit -S — never a bare kill-server (the recorded repo
		// discipline; see startPrivateTmuxWithPane in dispatch_start_test.go).
		privateSocket := filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")
		if _, err := os.Stat(privateSocket); err != nil {
			t.Fatalf("refusing to continue: tmux did not bind the private socket %s (%v) — "+
				"the server may be the real one, and killing it is unsafe", privateSocket, err)
		}
		t.Cleanup(func() { _, _ = tmux("-S", privateSocket, "kill-server") })
		paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
		if err != nil || paneID == "" {
			t.Fatalf("resolve pane id: %v (%q)", err, paneID)
		}

		stdout, _, err := runPaneCmd(t, "ready", paneID, "--json")
		if err != nil {
			t.Fatalf("a classification is never an error: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("unmarshal: %v\n%s", err, stdout)
		}
		if s, ok := got["server"]; !ok || s != nil {
			t.Errorf("server = %v (present %v), want JSON null", s, ok)
		}
	})
}

// TestPaneReady_RKGoneExits2 pins the delegated arm's dead-pane mapping end to
// end: with a sentinel-capable rk on PATH (a stub) reporting `gone` for the
// pane, `fab pane ready` exits 2 — the documented dead-pane path — not the
// generic-failure 3. The pane stays ALIVE for the whole run (ValidatePane and
// the takeover precondition both pass against it); only rk's report is
// scripted, which is exactly the "pane died mid-await" case the mapping
// exists for. The in-handler os.Exit requires the child-process re-exec (the
// TestPaneVerbExitCodes pattern).
func TestPaneReady_RKGoneExits2(t *testing.T) {
	if os.Getenv("FAB_PANE_READY_RK_HELPER") == "1" {
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

	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// A live NON-shell pane (readyPaneCommand's foreground is `cat`): the
	// takeover precondition passes and the rk arm engages.
	server := "fabtest-paneready-rkgone"
	_, paneID := newTmuxPane(t, server, readyPaneCommand, 80)

	// The stub rk answers the capability probe (its help mentions `parked`)
	// and reports the pane `gone` (exit 1) on the delegated await.
	bin := t.TempDir()
	stub := `#!/bin/sh
if [ "$1 $2 $3" = "mux await --help" ]; then
	printf 'echo = ready, no echo = parked\n'
	exit 0
fi
if [ "$1 $2 $3" = "mux await --ready" ]; then
	echo "gone $4"
	exit 1
fi
exit 1
`
	if err := os.WriteFile(filepath.Join(bin, "rk"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}

	child := exec.Command(os.Args[0], "-test.run=^TestPaneReady_RKGoneExits2$", "--", "ready", paneID, "-L", server)
	child.Env = append(os.Environ(),
		"FAB_PANE_READY_RK_HELPER=1",
		// Opt out of TestMain's not-capable rk stub (which would shadow this
		// test's stub in the child re-exec) and into the delegated arm.
		"FAB_TEST_RK_ARM=1",
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := child.CombinedOutput()
	got := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("child failed to run: %v (%s)", err, out)
		}
		got = exitErr.ExitCode()
	}
	if got != 2 {
		t.Errorf("exit = %d, want 2 — rk's gone is the dead-pane path (output: %s)", got, out)
	}
}
