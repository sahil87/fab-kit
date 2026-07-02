package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
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
