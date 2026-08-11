package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// TestPaneValidationExitCode pins the pane-family exit-code scheme shared
// with window-name: 2 = pane missing, 3 = any other tmux failure (F30).
func TestPaneValidationExitCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"pane not found", &pane.PaneNotFoundError{Pane: "%5"}, 2},
		{"wrapped pane not found", fmt.Errorf("validate: %w", &pane.PaneNotFoundError{Pane: "%5"}), 2},
		{"other tmux failure", errors.New("tmux display-message: exit status 1: no server running"), 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := paneValidationExitCode(tc.err); got != tc.want {
				t.Errorf("paneValidationExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

// TestPaneVerbExitCodes pins the pane-family 2/3 scheme on the open/ready/
// deliver verbs: pane missing → 2, other tmux failure (dead socket) → 3. The
// in-handler os.Exit cannot be observed in-process, so each case re-executes
// the test binary as a child; the child branch (guarded by the env var)
// executes the cobra command and exits for real — RunE errors map to 1,
// matching the binary-wide convention.
func TestPaneVerbExitCodes(t *testing.T) {
	if os.Getenv("FAB_PANE_EXITCODE_HELPER") == "1" {
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
	// A live server for the missing-pane cases; its socket dies with the test.
	server := "fabtest-exitcodes"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	// Children run outside any fab repo, so provider resolution rides the
	// built-in table (the empty-config posture `fab pane open` documents).
	scratch := t.TempDir()
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"ready missing pane exits 2", []string{"ready", "%999", "-L", server}, 2},
		{"ready dead socket exits 3", []string{"ready", "%1", "-L", "nosuch-dead-sock"}, 3},
		{"deliver missing pane exits 2", []string{"deliver", "%999", "-L", server, "--text", "hi"}, 2},
		{"deliver dead socket exits 3", []string{"deliver", "%1", "-L", "nosuch-dead-sock", "--text", "hi"}, 3},
		{"open dead socket exits 3", []string{"open", "--provider", "kimi", "-L", "nosuch-dead-sock"}, 3},
		{"kill missing pane exits 2", []string{"kill", "%999", "-L", server}, 2},
		{"kill dead socket exits 3", []string{"kill", "%1", "-L", "nosuch-dead-sock"}, 3},
		{"await missing pane exits 2", []string{"await", "%999", "-L", server, "--file", "x"}, 2},
		{"await dead socket exits 3", []string{"await", "%1", "-L", "nosuch-dead-sock", "--file", "x"}, 3},
		{"process missing pane exits 2", []string{"process", "%999", "-L", server}, 2},
		{"process dead socket exits 3", []string{"process", "%1", "-L", "nosuch-dead-sock"}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := exec.Command(os.Args[0], append([]string{"-test.run=^TestPaneVerbExitCodes$", "--"}, tc.args...)...)
			child.Env = append(os.Environ(), "FAB_PANE_EXITCODE_HELPER=1")
			child.Dir = scratch
			out, err := child.CombinedOutput()
			got := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("child failed to run: %v (%s)", err, out)
				}
				got = exitErr.ExitCode()
			}
			if got != tc.want {
				t.Errorf("exit = %d, want %d (output: %s)", got, tc.want, out)
			}
		})
	}
}
