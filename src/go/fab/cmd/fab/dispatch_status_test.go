package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// seedDispatch writes a {stage}.yaml with the given pid so status can derive a
// state. Uses the repo set up by setupDispatchRepo.
func seedDispatch(t *testing.T, repoRoot, id, stage string, pid int) string {
	t.Helper()
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, stage, &dispatch.Dispatch{
		PID: pid, PGID: pid, SpawnCmd: "x", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runStatus(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchStatusCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestDispatchStatus_States(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")

	// running: live pid (our own), no exit file.
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())
	if out, _ := runStatus(t, "abcd", "apply"); strings.TrimSpace(out) != "running" {
		t.Errorf("running: got %q", strings.TrimSpace(out))
	}

	// done: exit 0 + result present.
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "ok: true\n")
	if out, _ := runStatus(t, "abcd", "apply"); strings.TrimSpace(out) != "done" {
		t.Errorf("done: got %q", strings.TrimSpace(out))
	}

	// failed (no-result): exit 0, no result.
	os.Remove(dispatch.ResultPath(dir, "apply"))
	if out, _ := runStatus(t, "abcd", "apply"); strings.TrimSpace(out) != "failed (no-result)" {
		t.Errorf("no-result: got %q", strings.TrimSpace(out))
	}

	// failed: non-zero exit.
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "124\n")
	if out, _ := runStatus(t, "abcd", "apply"); strings.TrimSpace(out) != "failed" {
		t.Errorf("failed: got %q", strings.TrimSpace(out))
	}

	// orphaned: dead pid, no exit file.
	seedDispatch(t, repoRoot, id, "review", 999999)
	if out, _ := runStatus(t, "abcd", "review"); strings.TrimSpace(out) != "orphaned" {
		t.Errorf("orphaned: got %q", strings.TrimSpace(out))
	}
}

func TestDispatchStatus_JSON(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "ok: true\n")

	out, err := runStatus(t, "abcd", "apply", "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var got dispatchStatusJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if got.State != "done" || got.Stage != "apply" || got.Change != "abcd" {
		t.Errorf("json = %+v", got)
	}
	if got.Exit == nil || *got.Exit != 0 {
		t.Errorf("json exit = %v, want 0", got.Exit)
	}
}

// seedPaneDispatch writes a pane-mode {stage}.yaml so status/kill/logs can be
// exercised without a tmux server: an unresolvable pane ID reads as NOT alive,
// which is exactly the orphaned/dead-pane case.
func seedPaneDispatch(t *testing.T, repoRoot, id, stage, paneID, server string) string {
	t.Helper()
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, stage, &dispatch.Dispatch{
		Pane: paneID, Window: dispatch.WindowName(id, stage), Server: server,
		SpawnCmd: "claude", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestDispatchStatus_PaneStates covers the pane-mode three-state subset without
// tmux: a pane ID on an unreachable socket reads as dead, so result-absent is
// `orphaned` and result-present is `done` — proving result presence WINS over
// liveness (a finished worker still sitting at its prompt must not read
// `running`). The live-pane `running` case is covered by the integration test
// below, which needs a real tmux server to have an alive pane.
func TestDispatchStatus_PaneStates(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-nosrv-status"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))

	dir := seedPaneDispatch(t, repoRoot, id, "apply", "%99", server)

	// Dead pane, no result → orphaned. Notably NOT `failed`/`failed (no-result)`:
	// those are unreachable on the pane path (no exit-code channel).
	if out, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(out) != "orphaned" {
		t.Errorf("pane orphaned: got %q (err %v)", strings.TrimSpace(out), err)
	}

	// Result present → done, even though the pane is gone.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	if out, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(out) != "done" {
		t.Errorf("pane done: got %q (err %v)", strings.TrimSpace(out), err)
	}

	// An exit file must be IGNORED on the pane path — pane mode never writes one,
	// and a stale one from a prior headless attempt must not flip the derivation
	// into the five-state machine.
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "1\n")
	if out, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(out) != "done" {
		t.Errorf("pane done with a stale exit file: got %q (err %v)", strings.TrimSpace(out), err)
	}
}

// TestDispatchStatus_PaneJSON pins the --json shape for both modes: a pane object
// carries mode/pane/window and NO pid/pgid/exit; a headless object carries
// mode/pid/pgid exactly as before.
func TestDispatchStatus_PaneJSON(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "claude")
	server := "fabtest-nosrv-json"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))

	// Pane dispatch.
	seedPaneDispatch(t, repoRoot, id, "review", "%99", server)
	out, err := runStatus(t, "abcd", "review", "--json")
	if err != nil {
		t.Fatalf("status --json (pane): %v", err)
	}
	var got dispatchStatusJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if got.Mode != string(dispatch.ModePane) {
		t.Errorf("mode = %q, want %q", got.Mode, dispatch.ModePane)
	}
	if got.Pane != "%99" || got.Window != dispatch.WindowName(id, "review") {
		t.Errorf("pane identity = %q/%q", got.Pane, got.Window)
	}
	if got.PID != 0 || got.PGID != 0 || got.Exit != nil {
		t.Errorf("pane json must omit pid/pgid/exit, got %+v", got)
	}
	// The keys really are absent from the encoding, not merely zero-valued.
	for _, key := range []string{`"pid"`, `"pgid"`, `"exit"`} {
		if strings.Contains(out, key) {
			t.Errorf("pane json must not contain %s:\n%s", key, out)
		}
	}

	// Headless dispatch on the same change: mode=headless, pid/pgid present.
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "ok: true\n")
	out, err = runStatus(t, "abcd", "apply", "--json")
	if err != nil {
		t.Fatalf("status --json (headless): %v", err)
	}
	var hl dispatchStatusJSON
	if err := json.Unmarshal([]byte(out), &hl); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out)
	}
	if hl.Mode != string(dispatch.ModeHeadless) || hl.State != "done" {
		t.Errorf("headless json = %+v", hl)
	}
	if hl.PID == 0 || hl.PGID == 0 || hl.Exit == nil {
		t.Errorf("headless json must carry pid/pgid/exit, got %+v", hl)
	}
	if hl.Pane != "" || hl.Window != "" {
		t.Errorf("headless json must omit pane identity, got %+v", hl)
	}
}

// TestDispatchStatus_PaneRunning_Integration is the live half of the pane state
// matrix: an alive pane with no result reads `running`, and writing the result
// while that same pane is STILL alive flips it to `done` (the precedence rule).
// Skipped when tmux is unavailable.
func TestDispatchStatus_PaneRunning_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")

	server := "fabtest-pstat"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
	if err != nil || paneID == "" {
		t.Fatalf("resolve pane id: %v (%q)", err, paneID)
	}
	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)

	if out, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(out) != "running" {
		t.Errorf("alive pane, no result: got %q (err %v)", strings.TrimSpace(out), err)
	}

	// Result written while the pane is still alive → done wins.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	if out, err := runStatus(t, "abcd", "apply"); err != nil || strings.TrimSpace(out) != "done" {
		t.Errorf("alive pane WITH result: got %q (err %v), want done", strings.TrimSpace(out), err)
	}
}

func TestDispatchStatus_NoDispatchErrors(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, err := runStatus(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error when no dispatch exists")
	}
	if !strings.Contains(err.Error(), "no dispatch") {
		t.Errorf("error = %q", err.Error())
	}
}
