package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

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
//
// It also scrubs $TMUX/$TMUX_PANE: $TMUX outranks TMUX_TMPDIR in tmux's
// socket resolution (-L/-S > $TMUX > TMUX_TMPDIR), so for any test run from
// inside a tmux pane an inherited $TMUX would silently redirect unscoped
// tmux calls onto the HOST server — a destructive cleanup then kills the
// host (change kgam). Empirically (tmux 3.6a) an empty $TMUX is treated as
// unset, so t.Setenv suffices. Tests that need $TMUX set (simulating a
// dispatcher inside a pane) set it themselves after this call.
func tmuxSocketDir(t *testing.T, name string) string {
	t.Helper()
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
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
