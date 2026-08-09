package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// This file covers `fab dispatch ready`'s REPORT — the surface the orchestrator's
// judgment rounds read. The classification itself is unit-tested against scripted
// captures in internal/dispatch/gate_test.go; what can only be pinned here is what
// a non-`ready` answer actually prints, and that answering at all is a success.
//
// The `ready` case rides the end-to-end delivery test in dispatch_deliver_test.go,
// where a live shell echoes the sentinel.

// parkedPaneCommand is a pane that swallows typed input on a stable screen —
// mechanically indistinguishable from a first-run trust dialog, which is the
// point: the gate classifies by echo and stability, never by dialog text. `stty
// -echo` stops the tty echoing what is typed, and `sleep` never reads it.
const parkedPaneCommand = "stty -echo; echo TRUST-THIS-FOLDER-WALL; sleep 300"

// bootingPaneCommand is the same swallowed input on a screen that has not been
// drawn yet — a TUI still starting rather than a wall holding the input.
const bootingPaneCommand = "stty -echo; sleep 300"

// TestDispatchReady_RefusesAMidStageWorker: the probe is a SENDER — it types the
// sentinel and presses C-u — so it is bound by the same no-input-injection rule as
// `deliver`. Against a delivered worker still executing its stage it must refuse
// before touching the keyboard; against the same worker once its result is present
// (`done`, the sanctioned continuation case) it must still answer.
func TestDispatchReady_RefusesAMidStageWorker(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-ready-midstage"
	tmux, paneID := newTmuxPane(t, server, "", 80)

	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatal(err)
	}
	rec.Delivered = true
	if err := dispatch.Save(dir, "apply", rec); err != nil {
		t.Fatal(err)
	}

	before, err := tmux("capture-pane", "-p", "-t", paneID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runReady(t, "abcd", "apply"); err == nil {
		t.Fatal("ready must refuse a worker that is mid-stage")
	} else if !strings.Contains(err.Error(), "still running") {
		t.Errorf("error = %q, want it to name the running worker", err)
	}
	if after, err := tmux("capture-pane", "-p", "-t", paneID); err == nil && after != before {
		t.Errorf("the pane changed (%q → %q); a refused probe must send nothing", before, after)
	}

	// Result present ⇒ `done`: the worker is back at its prompt, so probing it
	// ahead of a continuation delivery is exactly what the gate is for.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	out, err := runReady(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("a `done` worker must pass the mid-stage guard: %v", err)
	}
	if out != string(dispatch.ReadyReady)+"\n" {
		t.Errorf("report = %q, want %q from the live shell", out, string(dispatch.ReadyReady)+"\n")
	}
}

// TestDispatchReady_NonReadyReports pins both non-`ready` answers end to end: the
// classification line, the pane/socket lines the judgment rounds need to send raw
// `tmux send-keys` (`status --json` carries the pane but not the socket), the
// capture snippet that lets the orchestrator see WHAT is holding the input, and
// exit 0 — an observed answer is a success however inconvenient it is, exactly as
// with `fab dispatch wait`'s timeout.
//
// The evidence header rides the snippet: a pane that has drawn nothing yet (the
// booting row) reports the classification and the pane/socket lines and stops,
// because a header with nothing under it reads as a truncated report.
func TestDispatchReady_NonReadyReports(t *testing.T) {
	for _, tt := range []struct {
		name    string
		command string
		want    dispatch.Readiness
		screen  string
	}{
		{"parked behind a wall", parkedPaneCommand, dispatch.ReadyParked, "TRUST-THIS-FOLDER-WALL"},
		{"still booting", bootingPaneCommand, dispatch.ReadyBooting, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
			server := "fabtest-ready-" + string(tt.want)
			_, paneID := newTmuxPane(t, server, tt.command, 80)
			seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)

			out, err := runReady(t, "abcd", "apply")
			if err != nil {
				t.Fatalf("a classification is never an error: %v", err)
			}

			lines := strings.SplitN(out, "\n", 2)
			if lines[0] != string(tt.want) {
				t.Fatalf("report = %q, want it to open with %q", out, tt.want)
			}
			wants := []string{"pane: " + paneID, "server: " + server}
			if tt.screen != "" {
				wants = append(wants, fmt.Sprintf("--- last %d lines ---", dispatch.SnippetLines), tt.screen)
			}
			for _, want := range wants {
				if !strings.Contains(out, want) {
					t.Errorf("report = %q, want it to carry %q", out, want)
				}
			}
			if tt.screen == "" && strings.Contains(out, "--- last") {
				t.Errorf("report = %q, want no evidence header above an empty snippet", out)
			}
			// The snippet is the last thing written, so the report must end its
			// line — otherwise the shell prompt lands on the evidence.
			if !strings.HasSuffix(out, "\n") {
				t.Errorf("report = %q, want it to end with a newline", out)
			}
		})
	}
}
