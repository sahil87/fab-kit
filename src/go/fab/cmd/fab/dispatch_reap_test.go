package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func runReap(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out, _, err := runReapErr(t, args...)
	return out, err
}

// runReapErr is runReap with the warning stream exposed, for the tests that assert
// on the fail-open notice `dispatchReapEnabled` writes to stderr.
func runReapErr(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := dispatchReapCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

// setDispatchReapDone merges the setting into the repo's project dispatch block so
// a test can exercise the disabled branch. The setup helpers isolate $HOME, so the
// system layer contributes nothing and this is the only layer in play.
func setDispatchReapDone(t *testing.T, repoRoot string, enabled bool) {
	t.Helper()
	value := "false"
	if enabled {
		value = "true"
	}
	appendProjectConfig(t, repoRoot, "dispatch:\n  reap_done: "+value+"\n")
}

// corruptProjectConfig makes the repo's project config unparseable, so config.Load
// returns its parse error (LoadPath's documented "a malformed PROJECT file keeps
// today's error behavior"). The setup helpers isolate $HOME, so the system layer
// contributes nothing and this is the only layer in play.
func corruptProjectConfig(t *testing.T, repoRoot string) {
	t.Helper()
	path := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	if err := os.WriteFile(path, []byte("dispatch:\n  reap_done: [unterminated\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

// TestDispatchReap_UnreadableConfigHeadlessIsNoOp: reap's exit contract reserves
// non-zero for exactly two real errors (no record, unresolvable change), and the
// skill wiring calls reap unconditionally after every `done`. So an unparseable
// config MUST NOT turn a headless no-op into a pipeline failure. The knob cannot
// affect a headless verdict, so it is never resolved here — no warning either.
func TestDispatchReap_UnreadableConfigHeadlessIsNoOp(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := seedDispatch(t, repoRoot, id, "apply", 999999)
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	corruptProjectConfig(t, repoRoot)

	out, errOut, err := runReapErr(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("an unreadable config must not fail a headless no-op, got: %v", err)
	}
	if !strings.Contains(out, "headless") {
		t.Errorf("output = %q, want the headless no-op reason", out)
	}
	if strings.Contains(errOut, "reap_done") {
		t.Errorf("stderr = %q, want no knob warning — the knob cannot affect a headless verdict, so it must not be resolved", errOut)
	}
}

// TestDispatchReap_UnreadableConfigFailsOpen: where the knob CAN change the outcome
// (pane + done) an unreadable config still must not error — it warns and falls back
// to the built-in default, which is what an absent key resolves to anyway. Defaulting
// to true means the guard passes, so this lands on the benign already-gone path
// (unreachable socket ⇒ dead pane) rather than the disabled no-op.
func TestDispatchReap_UnreadableConfigFailsOpen(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-nosrv-reap-badcfg"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	dir := seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	corruptProjectConfig(t, repoRoot)

	out, errOut, err := runReapErr(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("an unreadable config must fail open, not error, got: %v", err)
	}
	if !strings.Contains(errOut, "dispatch.reap_done") {
		t.Errorf("stderr = %q, want a warning naming the unresolved knob", errOut)
	}
	if strings.Contains(out, "dispatch.reap_done is false") {
		t.Errorf("output = %q, want the default (true) — failing open must not silently disable reap", out)
	}
	if !strings.Contains(out, "already gone") {
		t.Errorf("output = %q, want the guard to have passed through to the already-gone report", out)
	}
}

// TestDispatchReap_NoDispatchErrors: a missing record is the family's one REAL
// error — non-zero exit with the shared `status`/`wait` message surface. Everything
// else `reap` can encounter is a no-op.
func TestDispatchReap_NoDispatchErrors(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, err := runReap(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error when no dispatch exists")
	}
	if !strings.Contains(err.Error(), "no dispatch") {
		t.Errorf("error = %q, want the family's no-dispatch message", err.Error())
	}
}

// TestDispatchReap_HeadlessIsNoOp: the skill wiring calls `reap` unconditionally
// after every `done` CLI dispatch, so a headless record MUST be a clean no-op — the
// worker process already exited and there is nothing visual to reclaim. It is a
// no-op even when the dispatch is `done`, which is exactly when the wiring fires.
func TestDispatchReap_HeadlessIsNoOp(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := seedDispatch(t, repoRoot, id, "apply", 999999)
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")

	out, err := runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("reap of a headless dispatch must be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "headless") {
		t.Errorf("output = %q, want the headless no-op reason", out)
	}
	// Still `done` — reap touched nothing.
	if st, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(st) != "done" {
		t.Errorf("post-reap status = %q (err %v), want done", strings.TrimSpace(st), err)
	}
}

// TestDispatchReap_NotDoneIsNoOp is the reap-is-NOT-kill guard at the command layer:
// a pane dispatch that has produced no result derives `orphaned` here (dead pane, no
// result) and MUST NOT be touched. The report names the state and points at `kill`
// for the case where terminating really is wanted.
func TestDispatchReap_NotDoneIsNoOp(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-nosrv-reap"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)

	out, err := runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("reap of a non-done dispatch must be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, string(dispatch.StateOrphaned)) {
		t.Errorf("output = %q, want the no-op report naming the state", out)
	}
	if !strings.Contains(out, "kill") {
		t.Errorf("output = %q, want the report to point at `fab dispatch kill` for a live dispatch", out)
	}
}

// TestDispatchReap_DisabledIsNoOp: `dispatch.reap_done: false` is the documented
// opt-out — a done pane worker keeps its pane and its scrollback. The knob is read
// through the config cascade INSIDE the command (the skill wiring never checks it),
// so this test is what proves the gate is actually wired.
func TestDispatchReap_DisabledIsNoOp(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	setDispatchReapDone(t, repoRoot, false)
	server := "fabtest-nosrv-reap-off"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	dir := seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")

	out, err := runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("reap with the knob off must be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "dispatch.reap_done is false") {
		t.Errorf("output = %q, want the disabled-knob reason naming the key", out)
	}
}

// TestDispatchReap_PaneAlreadyGoneIsBenign: a done pane whose pane no longer exists
// (killed by hand, or lost with its tmux server) is a benign already-gone report,
// mirroring `kill`'s idempotence. Here the socket is unreachable, so the pane reads
// as dead while the result file still makes the state `done`.
func TestDispatchReap_PaneAlreadyGoneIsBenign(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-nosrv-reap-gone"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	dir := seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")

	out, err := runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("reap of a gone pane must be a benign no-op, got: %v", err)
	}
	if !strings.Contains(out, "already gone") || !strings.Contains(out, "%99") {
		t.Errorf("output = %q, want the already-gone report naming the pane", out)
	}
}

// TestDispatchReap_PaneMode_Integration is the whole point of the verb, against a
// real tmux server: a DONE worker pane stacked in a carved column is reaped, and
//
//   - the dispatching agent's pane and the SIBLING worker pane survive (kill-pane is
//     pane-ID keyed, so a split worker's death takes nothing else with it),
//   - `status` still reads `done` afterwards (DerivePaneState gives result presence
//     precedence over pane liveness, so a reaped dispatch reads done forever), and
//   - every .fab-dispatch/ file survives — reap is pane hygiene, never state cleanup.
//
// SAFETY: every tmux call is scoped to a private `-L` socket under a private
// TMUX_TMPDIR, so the destructive kill-server cleanup can never reach a real server.
func TestDispatchReap_PaneMode_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")

	server := "fabtest-preap"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "200", "-y", "50"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	// Reproduce the two-tier layout: a dispatching agent's window with a carved
	// worker column holding two stacked workers.
	dispatcher, err := tmux("new-window", "-P", "-F", "#{pane_id}", "-n", "agent", "sleep 120")
	if err != nil || dispatcher == "" {
		t.Fatalf("create dispatcher window: %v (%q)", err, dispatcher)
	}
	worker, err := tmux("split-window", "-h", "-t", dispatcher, "-P", "-F", "#{pane_id}", "sleep 120")
	if err != nil || worker == "" {
		t.Fatalf("carve worker column: %v (%q)", err, worker)
	}
	sibling, err := tmux("split-window", "-v", "-t", worker, "-P", "-F", "#{pane_id}", "sleep 120")
	if err != nil || sibling == "" {
		t.Fatalf("stack sibling worker: %v (%q)", err, sibling)
	}

	dir := seedPaneDispatch(t, repoRoot, id, "apply", worker, server)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")

	if !pane.PaneAlive(worker, server) {
		t.Skip("created pane not observably alive; skipping liveness-dependent reap assertion")
	}
	if st, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(st) != "done" {
		t.Fatalf("pre-reap status = %q (err %v), want done", strings.TrimSpace(st), err)
	}

	out, err := runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if !strings.Contains(out, "reaped") || !strings.Contains(out, worker) {
		t.Errorf("output = %q, want a reaped report naming the pane", out)
	}

	if pane.PaneAlive(worker, server) {
		t.Errorf("worker pane %s should be gone after reap", worker)
	}
	if !pane.PaneAlive(dispatcher, server) {
		t.Errorf("the dispatching agent's pane %s must survive a worker reap", dispatcher)
	}
	if !pane.PaneAlive(sibling, server) {
		t.Errorf("the sibling worker pane %s must survive a worker reap", sibling)
	}

	// The state machine is untouched: result presence still wins over pane liveness.
	if st, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(st) != "done" {
		t.Errorf("post-reap status = %q (err %v), want done — a reaped dispatch reads done forever", strings.TrimSpace(st), err)
	}
	// Reap is pane hygiene, not state cleanup: every file is still there.
	for _, p := range []string{
		dispatch.YAMLPath(dir, "apply"),
		dispatch.ResultPath(dir, "apply"),
		dispatch.PromptPath(dir, "apply"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("reap removed %s (%v) — it must clean no .fab-dispatch/ state", p, err)
		}
	}

	// Idempotent re-reap: the pane is gone, so this is the benign already-gone path.
	out, err = runReap(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("re-reap should be benign, got: %v", err)
	}
	if !strings.Contains(out, "already gone") {
		t.Errorf("re-reap output = %q, want the already-gone report", out)
	}
}
