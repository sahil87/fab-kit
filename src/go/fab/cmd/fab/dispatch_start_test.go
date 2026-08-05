package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// setupDispatchRepo builds a repo with one active change and a config whose
// `doing` tier (apply's tier) points at a provider carrying a dispatch_command,
// then chdirs into it so resolve.FabRoot() resolves. When dispatchCmd is empty,
// no dispatch_command is configured (the resolved provider — the built-in claude —
// has none). Returns the repo root and the 4-char change ID.
func setupDispatchRepo(t *testing.T, dispatchCmd string) (repoRoot, id string) {
	t.Helper()
	return setupDispatchRepoWithCommands(t, dispatchCmd, "")
}

// setupDispatchRepoWithCommands is setupDispatchRepo with independent control of
// the resolved provider's TWO command fields, so the pane path (session_command)
// and the headless path (dispatch_command) can be exercised separately —
// including the case that proves there is no cross-fallback in either direction
// (one field set, the other empty). An empty pair leaves the built-in claude
// provider resolved (which carries a session_command but no dispatch_command).
func setupDispatchRepoWithCommands(t *testing.T, dispatchCmd, sessionCmd string) (repoRoot, id string) {
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
	if dispatchCmd != "" || sessionCmd != "" {
		// A cli provider carries the command field(s); the doing tier points at it.
		body += "providers:\n  cli:\n"
		if dispatchCmd != "" {
			body += "    dispatch_command: \"" + dispatchCmd + "\"\n"
		}
		if sessionCmd != "" {
			body += "    session_command: \"" + sessionCmd + "\"\n"
		}
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
	// profile appended to the base command. The model is derived from the doing
	// tier's built-in default (pinned once in agent.TestDefaultTierProfilesArePinned)
	// so a model bump does not touch this test.
	doingDefault, _ := agent.DefaultTier(agent.TierDoing)
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the base command as prefix", rec.SpawnCmd)
	}
	if !strings.Contains(rec.SpawnCmd, "--model "+doingDefault.Model) {
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

// TestDispatchStart_HeadlessRecordCarriesNoPaneFields pins that the headless
// path is unchanged by pane mode: the persisted record carries pid/pgid and none
// of the pane identity, so a headless dispatch's on-disk shape and reported mode
// are byte-stable.
func TestDispatchStart_HeadlessRecordCarriesNoPaneFields(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")

	if _, err := runStart(t, "prompt", "abcd", "apply"); err != nil {
		t.Fatalf("start: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("headless record reads as %q (pane=%q)", rec.Mode(), rec.Pane)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("headless record carries pane identity: window=%q server=%q", rec.Window, rec.Server)
	}
}

// TestDispatchStart_PaneAndTimeoutMutuallyExclusive: --timeout is enforced by the
// headless `sh -c` wrapper, which pane mode never constructs, so accepting both
// would advertise a bound nothing enforces. The guard must fire before any launch
// or state write.
func TestDispatchStart_PaneAndTimeoutMutuallyExclusive(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")

	_, err := runStart(t, "prompt", "abcd", "apply", "--pane", "--timeout", "600")
	if err == nil {
		t.Fatal("expected a usage error for --pane with --timeout")
	}
	if !strings.Contains(err.Error(), "--pane") || !strings.Contains(err.Error(), "--timeout") {
		t.Errorf("error = %q, want it to name both flags", err.Error())
	}
	// Nothing launched, nothing persisted.
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the usage error, got %v", err)
	}
}

// TestDispatchStart_PaneWithoutTmuxServerErrors: --pane requires a reachable tmux
// server and must leave no partial dispatch behind. Targeting a --server socket
// that has no running server is the deterministic way to force the failure
// regardless of whether the test host has tmux running.
func TestDispatchStart_PaneWithoutTmuxServerErrors(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")
	// A private, empty TMUX_TMPDIR guarantees the named socket has no server.
	server := "fabtest-unreachable"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))

	_, err := runStart(t, "prompt", "abcd", "apply", "--pane", "--server", server)
	if err == nil {
		t.Fatal("expected a hard error when no tmux server is reachable")
	}
	msg := err.Error()
	for _, want := range []string{"--pane", "tmux"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want it to mention %q", msg, want)
		}
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after an unreachable-server error, got %v", err)
	}
}

// TestDispatchStart_PaneRequiresSessionCommand: pane mode composes the provider's
// session_command and MUST NOT fall back to dispatch_command. The provider here
// carries only a dispatch_command, so --pane must error naming the
// session_command config key — the mirror image of the headless no-fallback rule.
func TestDispatchStart_PaneRequiresSessionCommand(t *testing.T) {
	setupDispatchRepoWithCommands(t, "codex exec", "")

	_, err := runStart(t, "prompt", "abcd", "apply", "--pane")
	if err == nil {
		t.Fatal("expected an error when the resolved provider has no session_command")
	}
	msg := err.Error()
	if !strings.Contains(msg, "providers.cli.session_command") {
		t.Errorf("error = %q, want the session_command config-key hint", msg)
	}
	if !strings.Contains(msg, "doing") {
		t.Errorf("error = %q, want mention of the resolved tier", msg)
	}
}

// TestDispatchStart_HeadlessStillRequiresDispatchCommand is the other half of the
// no-cross-fallback rule: a provider carrying ONLY a session_command must not
// satisfy a headless start.
func TestDispatchStart_HeadlessStillRequiresDispatchCommand(t *testing.T) {
	setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")

	_, err := runStart(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error: a session_command must not satisfy a headless start")
	}
	if !strings.Contains(err.Error(), "providers.cli.dispatch_command") {
		t.Errorf("error = %q, want the dispatch_command config-key hint", err.Error())
	}
}

// TestDispatchStart_PaneMode_Integration drives the real `--pane` path against an
// ephemeral tmux server: the window is created with the dispatch-window name (and
// no operator marker), the FULL prompt lands in {stage}-prompt.md while the
// window command carries only the one-line pointer, and the record persists the
// pane identity without pid/pgid. Skipped when tmux is unavailable.
func TestDispatchStart_PaneMode_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// A stand-in session_command that behaves like a real agent CLI: TEMPLATED
	// with {model}/{effort} (as the built-in claude default is, so WithProfile
	// substitutes rather than appending flags after the prompt), taking the prompt
	// as its trailing argument. It echoes that argument — so the test can prove
	// the POINTER, not the prompt body, was delivered — then stays alive like an
	// interactive worker sitting at its prompt. `_` fills sh -c's $0 slot so the
	// prompt lands in $1.
	repoRoot, id := setupDispatchRepoWithCommands(t, "",
		`sh -c 'echo \"model={model} effort={effort}\"; echo \"got: $1\"; sleep 30' _`)

	server := "fabtest-pdisp"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	fullPrompt := "line one of a long stage prompt\nline two\nline three\n"
	out, err := runStart(t, fullPrompt, "abcd", "apply", "--pane", "--server", server)
	if err != nil {
		t.Fatalf("pane start failed: %v", err)
	}
	if !strings.Contains(out, "dispatched abcd/apply") || !strings.Contains(out, "pane ") {
		t.Errorf("output = %q, want a dispatched line naming the pane", out)
	}

	dir := dispatch.DirFor(repoRoot, id)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if !rec.IsPane() || rec.Mode() != dispatch.ModePane {
		t.Fatalf("record reads as %q, want pane mode (%+v)", rec.Mode(), *rec)
	}
	if rec.PID != 0 || rec.PGID != 0 {
		t.Errorf("pane record must not carry pid/pgid, got %d/%d", rec.PID, rec.PGID)
	}
	if rec.Server != server {
		t.Errorf("server = %q, want %q", rec.Server, server)
	}
	if want := dispatch.WindowName(id, "apply"); rec.Window != want {
		t.Errorf("window = %q, want %q", rec.Window, want)
	}

	// The window really exists under the dispatch-window name, carrying neither
	// the operator's `»` enrollment prefix nor its `›` done marker.
	windowName, err := tmux("display-message", "-p", "-t", rec.Pane, "#W")
	if err != nil {
		t.Fatalf("read window name: %v", err)
	}
	if windowName != dispatch.WindowName(id, "apply") {
		t.Errorf("tmux window name = %q, want %q", windowName, dispatch.WindowName(id, "apply"))
	}
	for _, marker := range []string{"»", "›"} {
		if strings.Contains(windowName, marker) {
			t.Errorf("window name %q must not carry the operator marker %q", windowName, marker)
		}
	}

	// The FULL prompt is on disk; the window command carried only the pointer.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("prompt not persisted: %v", err)
	}
	if string(promptData) != fullPrompt {
		t.Errorf("prompt file = %q, want the full prompt", string(promptData))
	}
	// What the worker actually RECEIVED is the one-line pointer naming the prompt
	// file — never the prompt body (which cannot ride argv/send-keys reliably).
	captured, err := tmux("capture-pane", "-p", "-t", rec.Pane)
	if err != nil {
		t.Fatalf("capture pane: %v", err)
	}
	if !strings.Contains(captured, ".fab-dispatch/"+id+"/apply-prompt.md") {
		t.Errorf("pane received %q, want the pointer naming the prompt file", captured)
	}
	if strings.Contains(captured, "line two") {
		t.Errorf("pane received prompt BODY content (%q); only the pointer may be delivered", captured)
	}
	// The pane is observably alive (the liveness signal status keys on).
	if !dispatch.PaneAlive(rec.Pane, server) {
		t.Errorf("pane %s should be alive right after start", rec.Pane)
	}
}

// TestDispatchStart_PanePointerCarriesRepoRelativePromptPath asserts the pointer
// composition independent of tmux: the worker is told to read the repo-relative
// prompt path (the window's cwd is the repo root), not the prompt body.
func TestDispatchStart_PanePointerCarriesRepoRelativePromptPath(t *testing.T) {
	pointer := dispatch.PointerPrompt(".fab-dispatch/abcd/apply-prompt.md")
	if !strings.Contains(pointer, ".fab-dispatch/abcd/apply-prompt.md") {
		t.Errorf("pointer = %q, want the repo-relative prompt path", pointer)
	}
	if strings.Contains(pointer, "\n") {
		t.Errorf("pointer = %q, want a single line (it rides as one quoted spawn argument)", pointer)
	}
}
