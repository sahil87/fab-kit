package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// setupDispatchRepo builds a repo with one active change and a config whose
// `doing` tier (apply's tier) points at a provider carrying a dispatch_command,
// then chdirs into it so resolve.FabRoot() resolves. When dispatchCmd is empty,
// no dispatch_command is configured (the resolved provider — the built-in claude —
// has none). Returns the repo root and the 4-char change ID.
func setupDispatchRepo(t *testing.T, dispatchCmd string) (repoRoot, id string) {
	t.Helper()
	repoRoot = t.TempDir()
	folder := "260310-abcd-my-change"
	id = "abcd"
	changeDir := filepath.Join(repoRoot, "fab", "changes", folder)
	mustMkdir(t, changeDir)
	mustWrite(t, filepath.Join(changeDir, ".status.yaml"), execTestStatusYAML)
	mustWrite(t, filepath.Join(changeDir, "intake.md"), "# Intake: My Change\n")
	if err := os.Symlink("fab/changes/"+folder+"/.status.yaml", filepath.Join(repoRoot, ".fab-status.yaml")); err != nil {
		t.Fatal(err)
	}

	projectDir := filepath.Join(repoRoot, "fab", "project")
	mustMkdir(t, projectDir)
	body := "project:\n  name: test\n"
	if dispatchCmd != "" {
		// A cli provider carries the dispatch_command; the doing tier points at it.
		body += "providers:\n  cli:\n    dispatch_command: \"" + dispatchCmd + "\"\n"
		body += "agent:\n  tiers:\n    doing: { provider: cli }\n"
	}
	mustWrite(t, filepath.Join(projectDir, "config.yaml"), body)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	return repoRoot, id
}

// waitDispatchDone blocks until the detached worker for dir/stage has written
// its exit file — the wrapper's final write (`echo $? > exit`). Register it as
// a cleanup after every real launch: the worker is detached, so without it a
// test can return while the worker is still dropping files into the TempDir,
// and the TempDir RemoveAll cleanup races it (a file landing between the
// list-entries and unlinkat steps fails with ENOTEMPTY). Cleanups run LIFO, so
// this always runs before the TempDir removal registered in setupDispatchRepo.
func waitDispatchDone(t *testing.T, dir, stage string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(dispatch.ExitPath(dir, stage)); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("dispatch worker for %s/%s did not write its exit file before teardown", dir, stage)
}

// runStart executes `fab dispatch start` with a prompt piped on stdin.
func runStart(t *testing.T, prompt string, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchStartCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetIn(strings.NewReader(prompt))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestDispatchStart_LaunchesAndPersistsState(t *testing.T) {
	// A benign, fast-exiting command so the detached launch has real pid/pgid.
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")

	out, err := runStart(t, "the stage prompt\n", "abcd", "apply")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !strings.Contains(out, "dispatched abcd/apply") {
		t.Errorf("output = %q, want dispatched line", out)
	}

	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	// Prompt persisted.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("prompt not persisted: %v", err)
	}
	if string(promptData) != "the stage prompt\n" {
		t.Errorf("prompt = %q", string(promptData))
	}

	// State persisted with a pid/pgid and the resolved spawn_cmd.
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want positive", rec.PID, rec.PGID)
	}
	// spawn.WithProfile appends the resolved --model/--effort to a non-templated
	// command (append mode), so the persisted spawn_cmd carries the doing-tier
	// profile (claude-fable-5 / xhigh) appended to the base command.
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the base command as prefix", rec.SpawnCmd)
	}
	if !strings.Contains(rec.SpawnCmd, "--model claude-fable-5") {
		t.Errorf("spawn_cmd = %q, want the resolved doing-tier model appended", rec.SpawnCmd)
	}
}

func TestDispatchStart_NoDispatchCommandErrors(t *testing.T) {
	setupDispatchRepo(t, "") // resolved provider (built-in claude) has no dispatch_command

	_, err := runStart(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error when the resolved provider has no dispatch_command")
	}
	msg := err.Error()
	if !strings.Contains(msg, "doing") || !strings.Contains(msg, "dispatch_command") {
		t.Errorf("error = %q, want mention of tier 'doing' and dispatch_command", msg)
	}
	// Must name the config key to set (no fallback to a session command).
	if !strings.Contains(msg, "providers.claude.dispatch_command") {
		t.Errorf("error = %q, want the config-key hint", msg)
	}
}

func TestDispatchStart_RefusesWhenRunning(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)

	// Simulate a running dispatch: a live pid (our own process), no exit file.
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		PID: os.Getpid(), PGID: os.Getpid(), SpawnCmd: "x", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := runStart(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("expected refuse-if-running error")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want already-running refusal", err.Error())
	}
}

func TestDispatchStart_OverwritesCompletedAttempt(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)

	// A completed prior attempt: a dead pid + an exit file + a stale result/log.
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		PID: 999999, PGID: 999999, SpawnCmd: "old", StartedAt: "old",
	}); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dispatch.ExitPath(dir, "apply"), "1\n")
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stale: true\n")
	mustWrite(t, dispatch.LogPath(dir, "apply"), "stale log\n")

	if _, err := runStart(t, "new prompt", "abcd", "apply"); err != nil {
		t.Fatalf("start over a completed attempt should succeed: %v", err)
	}
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	// The stale exit/result/log are cleared so the new run's status is clean.
	if _, err := os.Stat(dispatch.ExitPath(dir, "apply")); !os.IsNotExist(err) {
		// The command may finish and re-write exit before assertion; accept
		// either absent OR the fresh run's own value, but never the stale "1".
		data, _ := os.ReadFile(dispatch.ExitPath(dir, "apply"))
		if strings.TrimSpace(string(data)) == "1" {
			t.Error("stale exit code should have been cleared")
		}
	}
	if _, err := os.Stat(dispatch.ResultPath(dir, "apply")); !os.IsNotExist(err) {
		t.Error("stale result file should have been cleared")
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.SpawnCmd == "old" {
		t.Error("state should have been overwritten with the new attempt")
	}
}

func TestDispatchStart_TimeoutWrapsCommand(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")

	if _, err := runStart(t, "prompt", "abcd", "apply", "--timeout", "600"); err != nil {
		t.Fatalf("start with timeout failed: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.Timeout != 600 {
		t.Errorf("timeout = %d, want 600", rec.Timeout)
	}
}
