package main

import (
	"fmt"
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
