package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

func runLogs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchLogsCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestDispatchLogs_PrintsAndTails(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.LogPath(dir, "apply"), "line1\nline2\nline3\n")

	// Full log.
	out, err := runLogs(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if out != "line1\nline2\nline3\n" {
		t.Errorf("full log = %q", out)
	}

	// Tail 2.
	out, err = runLogs(t, "abcd", "apply", "--tail", "2")
	if err != nil {
		t.Fatalf("logs --tail: %v", err)
	}
	if out != "line2\nline3\n" {
		t.Errorf("tail 2 = %q", out)
	}
}

func TestDispatchLogs_MissingLogClearMessage(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, err := runLogs(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error for a missing log")
	}
	if !strings.Contains(err.Error(), "no dispatch log") {
		t.Errorf("error = %q, want the clear no-log message", err.Error())
	}
}

// TestDispatchLogs_PaneModeNamesPaneCapture: a pane dispatch keeps no log file
// (an interactive worker's output is tmux scrollback), so `logs` must report that
// fact and name the pane-mode equivalent rather than emitting the generic
// missing-log message, which offers no next step.
//
// The suggested command must carry the dispatch's recorded socket as `-L
// <server>` when it has one: a pane on a non-default socket is unreachable from
// a default-socket capture, so a socket-less hint would send the reader to the
// wrong server. With no recorded server the flag is omitted, letting the reader
// inherit the same default-socket resolution the dispatch used.
func TestDispatchLogs_PaneModeNamesPaneCapture(t *testing.T) {
	for _, tc := range []struct {
		name     string
		server   string
		wantCmd  string
		wantGone string // must NOT appear
	}{
		{
			name:    "no server recorded",
			server:  "",
			wantCmd: "fab pane capture %42",
			// A bare `-L` would be a broken suggestion with no socket to name.
			wantGone: "-L",
		},
		{
			name:    "server recorded",
			server:  "work",
			wantCmd: "fab pane capture -L work %42",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
			seedPaneDispatch(t, repoRoot, id, "apply", "%42", tc.server)

			_, err := runLogs(t, "abcd", "apply")
			if err == nil {
				t.Fatal("expected the pane-mode no-log report")
			}
			msg := err.Error()
			for _, want := range []string{"--pane", tc.wantCmd} {
				if !strings.Contains(msg, want) {
					t.Errorf("error = %q, want it to mention %q", msg, want)
				}
			}
			if tc.wantGone != "" && strings.Contains(msg, tc.wantGone) {
				t.Errorf("error = %q, want it NOT to contain %q", msg, tc.wantGone)
			}
			if strings.Contains(msg, "no dispatch log") {
				t.Errorf("error = %q, want the pane-mode report, not the generic missing-log message", msg)
			}
		})
	}
}
