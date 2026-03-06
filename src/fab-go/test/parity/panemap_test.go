package parity

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestPaneMapTmuxGuard(t *testing.T) {
	t.Run("errors when TMUX is unset", func(t *testing.T) {
		bin := fabBinary(t)
		tmp := setupTempRepo(t)

		cmd := exec.Command(bin, "pane-map")
		cmd.Dir = tmp

		// Build environment without TMUX
		var env []string
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "TMUX=") {
				env = append(env, e)
			}
		}
		cmd.Env = env

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		exitCode := 0
		if err := cmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				t.Fatalf("running pane-map: %v", err)
			}
		}

		if exitCode != 1 {
			t.Errorf("expected exit 1, got %d", exitCode)
		}
		if !strings.Contains(stderr.String(), "Error: not inside a tmux session") {
			t.Errorf("expected tmux guard error on stderr, got: %q", stderr.String())
		}
	})
}

func TestPaneMapHelp(t *testing.T) {
	t.Run("pane-map appears in help", func(t *testing.T) {
		tmp := setupTempRepo(t)
		res := runGo(t, tmp, "--help")
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "pane-map") {
			t.Error("pane-map not found in help output")
		}
	})

	t.Run("pane-map has its own help", func(t *testing.T) {
		tmp := setupTempRepo(t)
		res := runGo(t, tmp, "pane-map", "--help")
		if res.ExitCode != 0 {
			t.Fatalf("expected exit 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "tmux pane-to-worktree mapping") {
			t.Error("expected description in pane-map help output")
		}
	})
}

func TestPaneMapIntegration(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("requires tmux")
	}

	t.Run("runs successfully in tmux", func(t *testing.T) {
		tmp := setupTempRepo(t)
		res := runGo(t, tmp, "pane-map")
		if res.ExitCode != 0 {
			t.Errorf("expected exit 0, got %d; stderr: %s", res.ExitCode, res.Stderr)
		}
		// Should have either table output or the empty message
		if res.Stdout == "" {
			t.Error("expected some output")
		}
	})
}
