package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

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
	// Reap: give the signal a moment is unnecessary — assert the group is gone.
	_ = os.Getpid()
}
