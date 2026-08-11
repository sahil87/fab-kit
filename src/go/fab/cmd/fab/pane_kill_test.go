package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file covers `fab pane kill` at the command layer against a real
// private-socket tmux server: the kill itself, the success report, and the
// record-free guarantee. The missing-pane / dead-socket exit rows live in
// pane_exitcode_test.go's helper-driven table (the in-handler os.Exit cannot
// be observed in-process).

// TestPaneKill_Integration: a live pane is killed and reported; a second
// probe shows it gone. The default-socket form omits the `server:` line, the
// -L form carries it (the `open` output precedent).
func TestPaneKill_Integration(t *testing.T) {
	t.Run("kills a live pane", func(t *testing.T) {
		server := "fabtest-panekill"
		tmux, paneID := newTmuxPane(t, server, "", 80)

		stdout, _, err := runPaneCmd(t, "kill", paneID, "-L", server)
		if err != nil {
			t.Fatalf("kill: %v", err)
		}
		want := "killed " + paneID + "\nserver: " + server + "\n"
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
		if _, err := tmux("display-message", "-p", "-t", paneID, "#{pane_id}"); err == nil {
			t.Errorf("pane %s still resolves after the kill", paneID)
		}
	})

	t.Run("no dispatch state is written", func(t *testing.T) {
		server := "fabtest-panekill-rec"
		tmux, paneID := newTmuxPane(t, server, "", 80)
		dir := t.TempDir()
		chdirTestEnv(t, dir, nil)

		if _, _, err := runPaneCmd(t, "kill", paneID, "-L", server); err != nil {
			t.Fatalf("kill: %v", err)
		}
		if out, err := tmux("list-panes", "-a", "-F", "#{pane_id}"); err == nil && strings.Contains(out, paneID) {
			t.Errorf("pane %s still listed after the kill", paneID)
		}
		if _, err := os.Stat(filepath.Join(dir, ".fab-dispatch")); !os.IsNotExist(err) {
			t.Errorf("fab pane kill must write no .fab-dispatch state")
		}
	})
}
