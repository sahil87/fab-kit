package main

import (
	"errors"
	"strings"
	"testing"
)

// interactiveShellPrefix is the launch wrap spawn.WithInteractiveShell emits
// ahead of the single-quoted provider command.
const interactiveShellPrefix = `${SHELL:-/bin/sh} -i -c `

// unwrapInteractiveShell asserts that launch is a provider command wrapped by
// spawn.WithInteractiveShell — optionally followed by the interactive-spawn
// `; exec "$SHELL"` fallback when withFallback is set — and returns the inner
// command with its single-quoting undone, so a test can assert on the provider
// command exactly as the inner interactive shell will see it.
func unwrapInteractiveShell(t *testing.T, launch string, withFallback bool) string {
	t.Helper()
	rest, ok := strings.CutPrefix(launch, interactiveShellPrefix)
	if !ok {
		t.Fatalf("launch command %q does not start with the interactive-shell wrap %q", launch, interactiveShellPrefix)
	}
	if withFallback {
		if rest, ok = strings.CutSuffix(rest, `; exec "$SHELL"`); !ok {
			t.Fatalf("launch command %q does not end with the shell fallback OUTSIDE the wrap", launch)
		}
	}
	inner, err := unquoteSingle(rest)
	if err != nil {
		t.Fatalf("launch command %q: wrapped argument is not one single-quoted word: %v", launch, err)
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
	got := unwrapInteractiveShell(t, `${SHELL:-/bin/sh} -i -c 'FAB_AGENT_WORKERS='\''co'\''\'\'''\''dex'\'' claude '\''/fab-new x'\'''; exec "$SHELL"`, true)
	want := `FAB_AGENT_WORKERS='co'\''dex' claude '/fab-new x'`
	if got != want {
		t.Errorf("unwrap = %q, want %q", got, want)
	}
	if got := unwrapInteractiveShell(t, `${SHELL:-/bin/sh} -i -c 'kimi --auto'`, false); got != "kimi --auto" {
		t.Errorf("unwrap = %q, want %q", got, "kimi --auto")
	}
}
