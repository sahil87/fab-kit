package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func runKill(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchKillCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestDispatchKill_AlreadyDeadIsBenign(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	// A dead pid → kill is a benign no-op with a clear report, not an error.
	seedDispatch(t, repoRoot, id, "apply", 999999)

	out, err := runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("kill of a dead dispatch should be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "already dead") {
		t.Errorf("output = %q, want the already-dead report", out)
	}
}

func TestDispatchKill_NoDispatchErrors(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, err := runKill(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error when no dispatch exists")
	}
	if !strings.Contains(err.Error(), "no dispatch") {
		t.Errorf("error = %q", err.Error())
	}
}

// TestDispatchKill_PaneAlreadyGoneIsBenign: a pane dispatch whose pane no longer
// exists (here: an unreachable socket) is the pane-mode analogue of the dead-pid
// case — a benign no-op with a clear report naming the pane, not an error.
func TestDispatchKill_PaneAlreadyGoneIsBenign(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-nosrv-kill"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)

	out, err := runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("kill of a gone pane should be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "already dead") || !strings.Contains(out, "%99") {
		t.Errorf("output = %q, want the already-dead report naming the pane", out)
	}
}

// TestDispatchKill_PaneMode_Integration kills a real pane dispatch: the pane dies
// and the dispatch then reads `orphaned` (no result file, dead pane) — the
// documented post-kill state, with no marker file written. Skipped when tmux is
// unavailable.
func TestDispatchKill_PaneMode_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")

	server := "fabtest-pkill"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	// A second window so killing the dispatch pane does not tear down the server.
	paneID, err := tmux("new-window", "-P", "-F", "#{pane_id}", "-n", dispatch.WindowName(id, "apply"), "sleep 60")
	if err != nil || paneID == "" {
		t.Fatalf("create dispatch window: %v (%q)", err, paneID)
	}
	seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)

	if !pane.PaneAlive(paneID, server) {
		t.Skip("created pane not observably alive; skipping liveness-dependent kill assertion")
	}

	out, err := runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("kill: %v", err)
	}
	if !strings.Contains(out, "killed") || !strings.Contains(out, paneID) {
		t.Errorf("output = %q, want a killed report naming the pane", out)
	}
	if pane.PaneAlive(paneID, server) {
		t.Errorf("pane %s should be gone after kill", paneID)
	}

	// Post-kill the dispatch reads orphaned (dead pane, no result file).
	if st, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(st) != "orphaned" {
		t.Errorf("post-kill status = %q (err %v), want orphaned", strings.TrimSpace(st), err)
	}

	// Idempotent re-kill.
	out, err = runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("re-kill should be benign, got: %v", err)
	}
	if !strings.Contains(out, "already dead") {
		t.Errorf("re-kill output = %q, want the already-dead report", out)
	}
}

// TestDispatchKill_AliasedPaneIsNotKilled is the targeting-verb half of the
// restart-alias contract: a record whose pane ID EXISTS but whose pane_pid no
// longer matches the pane's shell pid names an impostor, not the worker — kill
// must report already-dead and send NO `kill-pane` (the impostor survives).
// Skipped when tmux is unavailable.
func TestDispatchKill_AliasedPaneIsNotKilled(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")

	server := "fabtest-alias-kill"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	paneID, err := tmux("new-window", "-P", "-F", "#{pane_id}", "-n", dispatch.WindowName(id, "apply"), "sleep 60")
	if err != nil || paneID == "" {
		t.Fatalf("create dispatch window: %v (%q)", err, paneID)
	}
	livePID, err := pane.GetPanePID(paneID, server)
	if err != nil {
		t.Fatalf("read pane pid: %v", err)
	}

	// Seed the record by hand: the tampered pane_pid simulates the restart-alias
	// (the ID recycled onto an unrelated pane).
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		Pane: paneID, Window: dispatch.WindowName(id, "apply"), Server: server,
		PanePID: livePID + 1, SpawnCmd: "claude", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}

	out, err := runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("kill of an aliased record should be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "already dead") {
		t.Errorf("output = %q, want the already-dead report", out)
	}
	if !pane.PaneAlive(paneID, server) {
		t.Errorf("the impostor pane %s was killed — kill must never target a mismatched pane", paneID)
	}
}

func TestDispatchKill_SignalsLiveGroup(t *testing.T) {
	// Launch a real detached sleeper, then kill its group and confirm it dies.
	repoRoot, id := setupDispatchRepo(t, "sh -c 'sleep 30'")
	if _, err := runStart(t, "prompt", "abcd", "apply"); err != nil {
		t.Fatalf("start: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Never leak the detached sleeper regardless of which path we take: SIGTERM
	// the group and reap the worker at test end (both idempotent no-ops once it
	// is already dead — the reap below normally handles it on the success path).
	t.Cleanup(func() {
		_ = dispatch.KillGroup(rec.PGID)
		if p, ferr := os.FindProcess(rec.PID); ferr == nil {
			_, _ = p.Wait()
		}
	})
	if !dispatch.Alive(rec.PID) {
		t.Skip("launched sleeper not observably alive; skipping liveness-dependent kill assertion")
	}

	out, err := runKill(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("kill: %v", err)
	}
	if !strings.Contains(out, "killed") {
		t.Errorf("output = %q, want a killed report", out)
	}
	// Assert the worker actually terminated. A plain kill(pid,0) probe cannot tell
	// a live process from a zombie (the killed shell lingers unreaped because this
	// test is its parent), so both would read "alive". Instead reap it: this test
	// IS the parent, so Wait blocks until the process truly terminates and cannot
	// return while it still runs — a bounded Wait that returns proves it died.
	waited := make(chan struct{})
	go func() {
		if p, ferr := os.FindProcess(rec.PID); ferr == nil {
			_, _ = p.Wait()
		}
		close(waited)
	}()
	select {
	case <-waited:
		// reaped — the killed worker terminated
	case <-time.After(5 * time.Second):
		t.Errorf("pid %d did not terminate within 5s of kill; expected the process group to die", rec.PID)
	}
}
