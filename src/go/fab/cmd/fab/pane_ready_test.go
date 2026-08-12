package main

import (
	"encoding/json"
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

// TestPaneReady_ReadyReport: a live pane at an idle shell prompt echoes the
// sentinel, so the report is exactly the classification line and nothing else.
func TestPaneReady_ReadyReport(t *testing.T) {
	server := "fabtest-paneready-ok"
	_, paneID := newTmuxPane(t, server, "", 80)

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
		_, paneID := newTmuxPane(t, server, "", 80)

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
