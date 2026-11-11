package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

// interactiveShellPrefix is the launch wrap spawn.WithInteractiveShell emits
// ahead of the single-quoted provider command.
const interactiveShellPrefix = `${SHELL:-/bin/sh} -i -c `

// unwrapInteractiveShell asserts that launch is a provider command wrapped by
// spawn.WithInteractiveShell — with the interactive-spawn `; exec "$SHELL"`
// fallback as the wrapped command's tail when withFallback is set (INSIDE the
// wrap) — and returns the inner command with its single-quoting and the
// fallback undone, so a test can assert on the agent command exactly as the
// inner interactive shell will run it.
func unwrapInteractiveShell(t *testing.T, launch string, withFallback bool) string {
	t.Helper()
	rest, ok := strings.CutPrefix(launch, interactiveShellPrefix)
	if !ok {
		t.Fatalf("launch command %q does not start with the interactive-shell wrap %q", launch, interactiveShellPrefix)
	}
	inner, err := unquoteSingle(rest)
	if err != nil {
		t.Fatalf("launch command %q: wrapped argument is not one single-quoted word: %v", launch, err)
	}
	if withFallback {
		if inner, ok = strings.CutSuffix(inner, `; exec "$SHELL"`); !ok {
			t.Fatalf("launch command %q does not end with the shell fallback INSIDE the wrap", launch)
		}
	}
	return inner
}

// assertWrappedPaneLaunch asserts a dispatch pane worker was STARTED under the
// interactive-shell wrap (tmux's #{pane_start_command} — the creator's
// shell-command argument, as tmux re-quotes it) while the record's SpawnCmd
// keeps the unwrapped resolved provider command.
func assertWrappedPaneLaunch(t *testing.T, tmuxScoped func(...string) (string, error), paneID, spawnCmd string) {
	t.Helper()
	start := paneFormat(t, tmuxScoped, paneID, "#{pane_start_command}")
	if !strings.Contains(start, "SHELL:-/bin/sh} -i -c ") {
		t.Errorf("pane start command = %q, want the provider launched under the interactive-shell wrap", start)
	}
	if strings.Contains(start, `exec "$SHELL"`) || strings.Contains(start, `exec \"`) {
		t.Errorf("pane start command = %q, a dispatch worker must carry NO shell fallback", start)
	}
	if strings.Contains(spawnCmd, " -i -c ") {
		t.Errorf("record spawn_cmd = %q, want the UNWRAPPED resolved command", spawnCmd)
	}
}

// capturedSpawnInner takes a stub-tmux capture (one argv element per line), picks
// the LAST line — the new-window shell-command argument of an interactive spawn
// — and returns its unwrapped inner command (asserting wrap + fallback grammar).
func capturedSpawnInner(t *testing.T, capture []byte) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(string(capture), "\n"), "\n")
	return unwrapInteractiveShell(t, lines[len(lines)-1], true)
}

// unquoteSingle reverses shellquote.Single: a run of '…' segments joined by
// \' escapes, forming exactly one shell word.
func unquoteSingle(s string) (string, error) {
	var b strings.Builder
	for s != "" {
		switch {
		case strings.HasPrefix(s, `\'`):
			b.WriteByte('\'')
			s = s[2:]
		case strings.HasPrefix(s, "'"):
			end := strings.IndexByte(s[1:], '\'')
			if end < 0 {
				return "", errUnterminatedQuote
			}
			b.WriteString(s[1 : 1+end])
			s = s[2+end:]
		default:
			return "", errUnquotedText
		}
	}
	return b.String(), nil
}

var (
	errUnterminatedQuote = errors.New("unterminated single quote")
	errUnquotedText      = errors.New("text outside single quotes")
)

func TestUnwrapInteractiveShell(t *testing.T) {
	got := unwrapInteractiveShell(t, `${SHELL:-/bin/sh} -i -c 'FAB_AGENT_WORKERS='\''co'\''\'\'''\''dex'\'' claude '\''/fab-new x'\''; exec "$SHELL"'`, true)
	want := `FAB_AGENT_WORKERS='co'\''dex' claude '/fab-new x'`
	if got != want {
		t.Errorf("unwrap = %q, want %q", got, want)
	}
	if got := unwrapInteractiveShell(t, `${SHELL:-/bin/sh} -i -c 'kimi --auto'`, false); got != "kimi --auto" {
		t.Errorf("unwrap = %q, want %q", got, "kimi --auto")
	}
}

// TestInteractiveSpawnFallbackKeepsThePane runs the real interactive-spawn
// composition — WithInteractiveShell(WithShellFallback(agent)) — in a scratch
// tmux pane under each shell the wrap can land on, and proves the pane is still
// a live, input-accepting shell after the agent exits. With the fallback OUTSIDE
// the wrap, bash and zsh die on their first tty read and the pane closes.
func TestInteractiveSpawnFallbackKeepsThePane(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	shells := map[string]string{"sh": "/bin/sh"}
	for _, name := range []string{"bash", "zsh"} {
		if path, err := exec.LookPath(name); err == nil {
			shells[name] = path
		}
	}
	for name, shellPath := range shells {
		t.Run(name, func(t *testing.T) {
			server := "fabtest-fallback-" + name
			t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
			t.Setenv("SHELL", shellPath)
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("ZDOTDIR", home)
			// zsh's new-user wizard only runs when no startup file exists.
			if err := os.WriteFile(filepath.Join(home, ".zshrc"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			tmux := func(args ...string) (string, error) {
				out, err := exec.Command("tmux", append([]string{"-L", server, "-f", "/dev/null"}, args...)...).CombinedOutput()
				return strings.TrimSpace(string(out)), err
			}
			if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
				t.Skipf("could not start tmux server (%v): %s", err, out)
			}
			t.Cleanup(func() { _, _ = tmux("kill-server") })
			if _, err := tmux("set-option", "-g", "remain-on-exit", "on"); err != nil {
				t.Fatal(err)
			}
			launch := spawn.WithInteractiveShell(spawn.WithShellFallback("sleep 0.3"))
			paneID, err := tmux("new-window", "-d", "-P", "-F", "#{pane_id}", launch)
			if err != nil {
				t.Fatalf("new-window: %v (%s)", err, paneID)
			}
			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) {
				if dead, _ := tmux("display-message", "-p", "-t", paneID, "#{pane_dead}"); dead == "1" {
					t.Fatalf("pane died after the agent exited — the shell fallback did not take over")
				}
				_, _ = tmux("send-keys", "-t", paneID, `echo FALL""BACK-ALIVE`, "Enter")
				if out, _ := tmux("capture-pane", "-p", "-t", paneID); strings.Contains(out, "FALLBACK-ALIVE\n") || strings.HasSuffix(out, "FALLBACK-ALIVE") {
					return
				}
				time.Sleep(300 * time.Millisecond)
			}
			out, _ := tmux("capture-pane", "-p", "-t", paneID)
			t.Fatalf("no shell answered in the pane after the agent exited; screen:\n%s", out)
		})
	}
}
