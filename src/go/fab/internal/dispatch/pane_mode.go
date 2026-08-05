package dispatch

import (
	"fmt"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file holds the tmux side of PANE-MODE dispatch (`fab dispatch start
// --pane`): server reachability, window creation, pane liveness, and pane kill.
//
// It lives in the platform-INDEPENDENT core rather than the build-tagged
// dispatch_posix.go / dispatch_windows.go split for the same reason WrapperArgv
// does: these are plain tmux SUBPROCESS calls with no syscall dependency, so the
// code compiles everywhere. Only the headless path's process launch/signal
// syscalls are platform-split. (Pane mode is still unusable where tmux is
// absent — but that surfaces as ServerReachable's actionable error, not a
// compile-time platform split.)
//
// Every tmux invocation goes through internal/pane's shared helpers — RunCmd
// (stdout/stderr/exec-error capture), WithServer (the `-L <name>` argv prefix),
// and StderrError (surface the child's diagnostic instead of a bare exit
// status) — so there is exactly one tmux argv builder and one stderr-enrichment
// convention in the binary.

// ServerReachable probes whether a tmux server is reachable, returning nil when
// it is and an ACTIONABLE error when it is not.
//
// The probe is a real tmux query (`tmux [-L <server>] list-sessions`), NOT an
// $TMUX environment read. That distinction is load-bearing: a dispatching
// orchestrator may itself be headless (the cross-harness case pane mode exists
// for), so an $TMUX-only gate would make --pane unusable from exactly those
// callers. With --server the probe targets that socket; without it, tmux's own
// default-socket resolution applies. This mirrors `fab resolve --pane`, where
// --server likewise replaces the $TMUX guard with socket-scoped discovery.
//
// What the probe actually distinguishes is "a server answered" from "nothing
// answered" — NOT "has sessions" from "has none". Under tmux's default
// `exit-empty on`, a server exits with its last session, so a zero-session
// server does not persist to be probed: list-sessions exits non-zero ("no server
// running") and --pane errors, correctly. Under `exit-empty off` a sessionless
// server DOES persist and list-sessions exits 0, so the probe passes; a
// subsequent new-window against it may still fail (tmux has no session to attach
// the window to), which OpenWindow surfaces as the child's own stderr via
// StderrError rather than a bare exit status. So the probe is a reachability
// gate, not a launch guarantee — the launch has its own actionable error.
func ServerReachable(server string) error {
	_, stderr, err := pane.RunCmd("tmux", pane.WithServer(server, "list-sessions")...)
	if err == nil {
		return nil
	}
	target := "the default tmux socket"
	if server != "" {
		target = fmt.Sprintf("tmux socket %q", server)
	}
	return pane.StderrError(fmt.Errorf(
		"--pane requires a reachable tmux server, but %s is unreachable; start tmux (or pass --server <name>), or drop --pane to dispatch headless",
		target), stderr)
}

// OpenWindow creates a tmux window named name with cwd dir running cmd, and
// returns the new window's PANE ID.
//
// The pane ID (not the window name) is the identity worth recording: it is
// server-global, stable for the pane's lifetime, and exempt from tmux's
// target-grammar prefix/glob resolution — so subsequent liveness probes and
// kills are exact, where a name-based target could resolve to a different
// window the user renamed into place. `-P -F '#{pane_id}'` asks new-window to
// print it, avoiding a follow-up lookup that could race a fast-exiting worker.
//
// cmd is passed as new-window's shell-command argument, so it is the WHOLE
// left-hand side including any shell expansions it carries (e.g.
// `$(basename "$(pwd)")` in the built-in claude session_command), which expand
// at invocation inside the new window — the `_cli-agents.md` § Spawn
// Composition contract.
func OpenWindow(server, name, dir, cmd string) (paneID string, err error) {
	out, stderr, err := pane.RunCmd("tmux", pane.WithServer(server,
		"new-window", "-P", "-F", "#{pane_id}", "-n", name, "-c", dir, cmd)...)
	if err != nil {
		return "", pane.StderrError(fmt.Errorf("tmux new-window: %w", err), stderr)
	}
	paneID = strings.TrimSpace(out)
	if paneID == "" {
		return "", fmt.Errorf("tmux new-window reported no pane id for window %q", name)
	}
	return paneID, nil
}

// PaneAlive reports whether a pane dispatch's tmux pane still exists — the
// pane-mode analogue of the headless path's kill(pid,0) liveness probe, and the
// signal that separates `running` from `orphaned` (see DerivePaneState).
//
// It delegates to pane.ValidatePane, which probes with a single targeted
// `display-message -t <pane> -p '#{pane_id}'` and compares the result to the
// argument. Any failure — pane gone, tmux server gone, socket error — reads as
// NOT alive, which is the correct conservative reading here: with no result
// file, an unobservable worker is orphaned, and `status` must report that rather
// than erroring out.
func PaneAlive(paneID, server string) bool {
	if paneID == "" {
		return false
	}
	return pane.ValidatePane(paneID, server) == nil
}

// KillPane kills a pane dispatch's tmux pane — the pane-mode analogue of
// KillGroup. Killing the pane takes the interactive worker down with it (the
// pane's process group is tmux's to reap), so there is no separate process
// signalling.
//
// It is IDEMPOTENT in the same shape as KillGroup: an already-gone pane is a
// benign no-op rather than an error, detected via pane.IsPaneMissing on tmux's
// stderr (and via the typed *pane.PaneNotFoundError path callers get from
// PaneAlive). No marker file is written — with no result file present the
// dispatch derives `orphaned` from the dead pane, per DerivePaneState.
func KillPane(paneID, server string) error {
	if paneID == "" {
		return fmt.Errorf("no pane recorded for this dispatch")
	}
	_, stderr, err := pane.RunCmd("tmux", pane.WithServer(server, "kill-pane", "-t", paneID)...)
	if err == nil || pane.IsPaneMissing(stderr) {
		return nil
	}
	return pane.StderrError(fmt.Errorf("tmux kill-pane: %w", err), stderr)
}
