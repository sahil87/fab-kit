package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file covers `fab pane deliver` at the COMMAND layer: the payload-flag
// guards, the success lines, and the end-to-end verified delivery into a real
// tmux pane. The choreography itself — echo verification, the single retry,
// the busy confirmation — is unit-tested against a scripted fake in
// internal/pane/gate_test.go, which is where the retry path can be forced
// deterministically.

// TestPaneDeliver_PayloadFlagGuard pins the exactly-one-of usage contract:
// neither flag and both flags are usage errors, raised before any tmux call.
func TestPaneDeliver_PayloadFlagGuard(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"neither payload flag", []string{"deliver", "%1"}},
		{"both payload flags", []string{"deliver", "%1", "--text", "hi", "--prompt-file", "p.md"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runPaneCmd(t, tt.args...)
			if err == nil {
				t.Fatal("expected a usage error, got nil")
			}
		})
	}
}

// TestPaneDeliver_MissingPromptFileTypesNothing: a pointer at a file that is
// not there would type cleanly, verify cleanly, and leave the pane reading
// nothing — so the existence check fires BEFORE the gate touches the keyboard,
// exits 1 naming the file, and the pane is byte-for-byte untouched.
func TestPaneDeliver_MissingPromptFileTypesNothing(t *testing.T) {
	server := "fabtest-panedeliver-nofile"
	tmux, paneID := newTmuxPane(t, server, "", 80)

	// Let the shell finish drawing its first prompt, or the pane's own startup
	// repaint — not the refused delivery — would diff the two captures.
	before := settledCapture(t, tmux, paneID)
	missing := filepath.Join(t.TempDir(), "nope.md")
	_, _, err := runPaneCmd(t, "deliver", paneID, "-L", server, "--prompt-file", missing)
	if err == nil {
		t.Fatal("a missing prompt file must fail the delivery")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error = %q, want it to name the missing file", err)
	}
	after, err := tmux("capture-pane", "-p", "-t", paneID)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("the pane changed (%q → %q); a refused delivery must type nothing", before, after)
	}
}

// settledCapture polls the pane until two consecutive captures agree (or the
// budget runs out), so assertions diff the delivery's effect, not the shell's
// own startup repaint.
func settledCapture(t *testing.T, tmux func(args ...string) (string, error), paneID string) string {
	t.Helper()
	prev := ""
	for range 20 {
		cur, err := tmux("capture-pane", "-p", "-t", paneID)
		if err != nil {
			t.Fatal(err)
		}
		if cur == prev && cur != "" {
			return cur
		}
		prev = cur
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("pane never settled to a stable screen")
	return ""
}

// TestPaneDeliver_SuccessLines drives both payloads into a live shell pane —
// which echoes typed text and reacts to Enter, enough of an "agent" for the
// choreography's terms — and pins the success report naming the pane and the
// payload source.
func TestPaneDeliver_SuccessLines(t *testing.T) {
	t.Run("literal text", func(t *testing.T) {
		server := "fabtest-panedeliver-text"
		_, paneID := newTmuxPane(t, server, "", 80)

		stdout, stderr, err := runPaneCmd(t, "deliver", paneID, "-L", server, "--text", "echo hi")
		if err != nil {
			t.Fatalf("deliver: %v (stderr %q)", err, stderr)
		}
		want := fmt.Sprintf("delivered %s (text)\n", paneID)
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
	})

	t.Run("prompt file types the pointer line", func(t *testing.T) {
		server := "fabtest-panedeliver-file"
		tmux, paneID := newTmuxPane(t, server, "", 120)

		promptPath := filepath.Join(t.TempDir(), "prompt.md")
		if err := os.WriteFile(promptPath, []byte("# do the thing\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := runPaneCmd(t, "deliver", paneID, "-L", server, "--prompt-file", promptPath)
		if err != nil {
			t.Fatalf("deliver: %v (stderr %q)", err, stderr)
		}
		want := fmt.Sprintf("delivered %s (prompt %s)\n", paneID, promptPath)
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
		// The typed payload is the dispatch-parity pointer line, verbatim on the
		// shell's transcript (the shell rejects `Read` as a command — which is
		// exactly the screen advance the delivery verified). tmux hard-wraps the
		// pane's visible lines at the pane width, so the comparison drops
		// whitespace from both sides, mirroring the gate's own wrap tolerance.
		capture, err := tmux("capture-pane", "-p", "-t", paneID)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(dropSpace(capture), dropSpace(pane.PointerPrompt(promptPath))) {
			t.Errorf("pane = %q, want it to show the pointer line %q", capture, pane.PointerPrompt(promptPath))
		}
	})
}

// dropSpace removes every whitespace rune, so a wrap-split line still matches.
func dropSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)
}

// TestPaneDeliver_ParkedPaneFailsWithSnippet: a pane that swallows the
// sentinel fails both delivery attempts (the probe is a precondition of every
// one), so the retry warning goes to stderr, the refusing screen follows under
// the snippet header, and the command exits 1.
func TestPaneDeliver_ParkedPaneFailsWithSnippet(t *testing.T) {
	server := "fabtest-panedeliver-parked"
	_, paneID := newTmuxPane(t, server, parkedPaneCommand, 80)

	_, stderr, err := runPaneCmd(t, "deliver", paneID, "-L", server, "--text", "echo hi")
	if err == nil {
		t.Fatal("a parked pane must refuse the delivery")
	}
	if !strings.Contains(err.Error(), "after 2 attempts") {
		t.Errorf("error = %q, want it to name the attempt budget", err)
	}
	if !strings.Contains(stderr, "warning: delivery attempt 1 failed") {
		t.Errorf("stderr = %q, want the retry warning", stderr)
	}
	header := fmt.Sprintf("--- pane %s, last %d lines ---", paneID, pane.SnippetLines)
	if !strings.Contains(stderr, header) || !strings.Contains(stderr, "TRUST-THIS-FOLDER-WALL") {
		t.Errorf("stderr = %q, want the snippet header %q over the refusing screen", stderr, header)
	}
}
