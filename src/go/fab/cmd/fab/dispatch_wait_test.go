package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

func runWait(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchWaitCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestDispatchWait_AlreadyTerminalReturnsImmediately covers the fast path for
// every terminal state: `wait` prints it and exits 0 without blocking, so it is
// safe to re-arm after a restart or re-run after an interruption. A generous
// --timeout is passed precisely to prove the wait never reaches it.
func TestDispatchWait_AlreadyTerminalReturnsImmediately(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())

	cases := []struct {
		name   string
		setup  func()
		expect string
	}{
		{"done", func() {
			mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
			mustWrite(t, dispatch.ResultPath(dir, "apply"), "ok: true\n")
		}, "done"},
		{"failed (no-result)", func() {
			os.Remove(dispatch.ResultPath(dir, "apply"))
		}, "failed (no-result)"},
		{"failed", func() {
			mustWrite(t, dispatch.ExitPath(dir, "apply"), "124\n")
		}, "failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			start := time.Now()
			out, err := runWait(t, "abcd", "apply", "--timeout", "600")
			if err != nil {
				t.Fatalf("wait: %v", err)
			}
			if got := strings.TrimSpace(out); got != tc.expect {
				t.Errorf("state = %q, want %q", got, tc.expect)
			}
			if elapsed := time.Since(start); elapsed > 2*time.Second {
				t.Errorf("took %v — an already-terminal dispatch must return immediately", elapsed)
			}
		})
	}

	// orphaned: a dead pid with no exit file, on a separate stage.
	seedDispatch(t, repoRoot, id, "review", 999999)
	out, err := runWait(t, "abcd", "review", "--timeout", "600")
	if err != nil {
		t.Fatalf("wait (orphaned): %v", err)
	}
	if got := strings.TrimSpace(out); got != "orphaned" {
		t.Errorf("orphaned: got %q", got)
	}
}

// TestDispatchWait_TimeoutPrintsRunningAndExitsZero pins the timeout contract:
// expiry prints the still-current state (`running`) and exits 0, so the STATE
// STRING is the discriminator a consuming skill reads — a `running` return is its
// peek-on-suspicion moment, not a failure.
func TestDispatchWait_TimeoutPrintsRunningAndExitsZero(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	// A live pid (our own) with no exit file reads `running` forever.
	seedDispatch(t, repoRoot, id, "apply", os.Getpid())

	start := time.Now()
	out, err := runWait(t, "abcd", "apply", "--timeout", "1")
	if err != nil {
		t.Fatalf("timeout must exit 0, got %v", err)
	}
	if got := strings.TrimSpace(out); got != "running" {
		t.Errorf("state = %q, want %q", got, "running")
	}
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("returned after %v — expected to block until the --timeout bound", elapsed)
	}
}

// TestDispatchWait_WakesOnStateTransition is the end-to-end event-driven case: a
// dispatch that is `running` when the wait starts and becomes `done` while it
// blocks wakes the wait and reports the new state, well before the bound.
func TestDispatchWait_WakesOnStateTransition(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())

	// Flip the dispatch to `done` shortly after the wait begins blocking.
	go func() {
		time.Sleep(300 * time.Millisecond)
		mustWriteAsync(dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
		mustWriteAsync(dispatch.ExitPath(dir, "apply"), "0\n")
	}()

	start := time.Now()
	out, err := runWait(t, "abcd", "apply", "--timeout", "60")
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if got := strings.TrimSpace(out); got != "done" {
		t.Errorf("state = %q, want %q", got, "done")
	}
	// The internal tick is 2s, so a wake well inside the 60s bound proves the loop
	// re-derived rather than sat until the timeout.
	if elapsed := time.Since(start); elapsed > 30*time.Second {
		t.Errorf("took %v — the wait did not wake on the transition", elapsed)
	}
}

// mustWriteAsync is mustWrite's goroutine-safe twin: *testing.T methods must not
// be called from a goroutine that may outlive the test, so this reports nothing
// and lets the wait's own assertion catch a failure.
func mustWriteAsync(path, content string) {
	_ = os.WriteFile(path, []byte(content), 0o644)
}

// TestDispatchWait_JSONMatchesStatus proves the two verbs share ONE render path:
// for the same on-disk record, `wait --json` and `status --json` emit
// byte-identical objects — in both modes.
func TestDispatchWait_JSONMatchesStatus(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "claude")
	server := "fabtest-nosrv-wait"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))

	// Headless, terminal.
	dir := seedDispatch(t, repoRoot, id, "apply", os.Getpid())
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "0\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "ok: true\n")

	waitOut, err := runWait(t, "abcd", "apply", "--json")
	if err != nil {
		t.Fatalf("wait --json: %v", err)
	}
	statusOut, err := runStatus(t, "abcd", "apply", "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	if waitOut != statusOut {
		t.Errorf("headless --json differs:\nwait:   %s\nstatus: %s", waitOut, statusOut)
	}
	var got dispatchStatusJSON
	if err := json.Unmarshal([]byte(waitOut), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, waitOut)
	}
	if got.State != "done" || got.Mode != string(dispatch.ModeHeadless) {
		t.Errorf("json = %+v", got)
	}

	// Pane, terminal — an unreachable socket makes the pane dead, and a present
	// result still wins, so this is a `done` pane record.
	paneDir := seedPaneDispatch(t, repoRoot, id, "review", "%99", server)
	mustWrite(t, dispatch.ResultPath(paneDir, "review"), "stage: review\nstatus: success\n")

	waitOut, err = runWait(t, "abcd", "review", "--json")
	if err != nil {
		t.Fatalf("wait --json (pane): %v", err)
	}
	statusOut, err = runStatus(t, "abcd", "review", "--json")
	if err != nil {
		t.Fatalf("status --json (pane): %v", err)
	}
	if waitOut != statusOut {
		t.Errorf("pane --json differs:\nwait:   %s\nstatus: %s", waitOut, statusOut)
	}
	for _, key := range []string{`"pid"`, `"pgid"`, `"exit"`} {
		if strings.Contains(waitOut, key) {
			t.Errorf("pane json must not contain %s:\n%s", key, waitOut)
		}
	}
}

// TestDispatchWait_NoDispatchErrors: the error surface mirrors `status` exactly —
// a missing record is a real error (non-zero), unlike a timeout.
func TestDispatchWait_NoDispatchErrors(t *testing.T) {
	setupDispatchRepo(t, "sh -c 'exit 0'")
	_, waitErr := runWait(t, "abcd", "apply")
	if waitErr == nil {
		t.Fatal("expected an error when no dispatch exists")
	}
	_, statusErr := runStatus(t, "abcd", "apply")
	if statusErr == nil {
		t.Fatal("status: expected an error when no dispatch exists")
	}
	if waitErr.Error() != statusErr.Error() {
		t.Errorf("error surfaces differ:\nwait:   %q\nstatus: %q", waitErr, statusErr)
	}
	if !strings.Contains(waitErr.Error(), "no dispatch") {
		t.Errorf("error = %q", waitErr.Error())
	}
}

// TestDispatchWait_RegisteredOnParent pins the seventh subcommand's presence and
// its argument arity on the `fab dispatch` parent.
func TestDispatchWait_RegisteredOnParent(t *testing.T) {
	var found bool
	for _, sub := range dispatchCmd().Commands() {
		if sub.Name() == "wait" {
			found = true
			if !strings.HasPrefix(sub.Use, "wait <change> <stage>") {
				t.Errorf("Use = %q", sub.Use)
			}
		}
	}
	if !found {
		t.Error("`wait` is not registered on the dispatch parent")
	}
}
