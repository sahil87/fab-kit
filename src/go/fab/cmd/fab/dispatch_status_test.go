package main

import (
	"bytes"
	"encoding/json"
	"os"
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
