package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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
	// Neutralize $TMUX so mode selection is HERMETIC: `fab dispatch start`'s mode
	// defaults to auto (pane inside tmux, headless outside), so a test suite run
	// from inside a tmux pane would otherwise auto-select pane and every headless
	// assertion below would depend on where the suite happens to run. Tests that
	// exercise auto selection re-set it themselves (t.Setenv is per-test and
	// restores on cleanup).
	t.Setenv("TMUX", "")
	// $TMUX_PANE is the second env signal — it picks the pane SHAPE (split the
	// dispatcher's window vs. open a new one). Neutralized for the same hermeticity
	// reason: a suite run from inside a tmux pane would otherwise inherit a real
	// pane id and every pane test's expected shape would depend on where the suite
	// runs. Tests that exercise the split path set it themselves.
	t.Setenv("TMUX_PANE", "")
	// $HOME is the SYSTEM config layer (~/.fab-kit/config.yaml), which the cascade
	// merges under the project file. Isolated for the same hermeticity reason as the
	// two env signals above: a developer's own `dispatch:` block (a column width, a
	// providers entry) would otherwise reach into these tests and change a resolved
	// command or a pane's expected geometry.
	t.Setenv("HOME", t.TempDir())
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
		// The tier PINS the doing-tier default profile explicitly. `cli` is a
		// cross-provider switch (the built-in doing tier names claude), and a tier
		// that switches provider no longer inherits the built-in's model/effort
		// (260805-j3cm's cross-provider cutoff — an inherited claude model would be
		// the footgun that fix closes). Pinning the same values the built-in carries
		// keeps every dispatch test's expectation ("the resolved doing-tier profile
		// rides the spawn command") true and still derived from the canonical map, so
		// a model bump does not touch these tests.
		doingDefault, _ := agent.DefaultTier(agent.TierDoing)
		body += "agent:\n  tiers:\n    doing: { provider: cli, model: " + doingDefault.Model + ", effort: " + doingDefault.Effort + " }\n"
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

// runStartCapturingStderr executes `fab dispatch start` with a prompt piped on
// stdin, returning stdout, stderr, and the error. Stderr matters for the
// auto-selection soft-fallback notices.
func runStartCapturingStderr(t *testing.T, prompt string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := dispatchStartCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetIn(strings.NewReader(prompt))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

// runStart is runStartCapturingStderr for the tests that assert only stdout — it
// delegates so the command wiring exists in exactly one place.
func runStart(t *testing.T, prompt string, args ...string) (string, error) {
	t.Helper()
	stdout, _, err := runStartCapturingStderr(t, prompt, args...)
	return stdout, err
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

// TestDispatchStart_PaneWithoutTmuxServerErrors: pane mode requires a reachable
// tmux server and must leave no partial dispatch behind. Targeting a --server socket
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
	// "pane mode", not "--pane": this invocation supplied only --server, and the
	// message must not quote a flag the caller never passed. "--headless" is the
	// actionable remedy the guidance owes.
	for _, want := range []string{"pane mode", "tmux", "--headless"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want it to mention %q", msg, want)
		}
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after an unreachable-server error, got %v", err)
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

// TestDispatchStart_PaneRefuseIfRunningHonorsTheResultFile pins that
// refuse-if-running reads the SAME finished-signal `fab dispatch status` derives
// pane state from: result presence wins over pane liveness. An interactive worker
// never exits on completion — it sits at its prompt — so a liveness-only refusal
// would report `done` from `status` while `start` refused forever, permanently
// stranding a completed attempt that the overwrite contract says is replaceable.
// Skipped when tmux is unavailable (a genuinely live pane is the whole point:
// with a dead pane the old rule would pass too).
func TestDispatchStart_PaneRefuseIfRunningHonorsTheResultFile(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)

	server := "fabtest-pdone"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	dir := dispatch.DirFor(repoRoot, id)
	if _, err := runStart(t, "first prompt", "abcd", "apply", "--pane", "--server", server); err != nil {
		t.Fatalf("first pane start failed: %v", err)
	}
	first, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !dispatch.PaneAlive(first.Pane, server) {
		t.Fatalf("pane %s should be alive; the test needs a live pane to be meaningful", first.Pane)
	}

	// While that pane is still ALIVE and carries NO result: genuinely running, so
	// a second start must refuse.
	if _, err := runStart(t, "second prompt", "abcd", "apply", "--pane", "--server", server); err == nil {
		t.Fatal("expected refusal while the pane is alive with no result file")
	} else if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want the already-running refusal", err.Error())
	}

	// The worker finishes: it writes its result and sits at its prompt (pane still
	// alive). status derives `done`, so start must now OVERWRITE rather than refuse.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	if !dispatch.PaneAlive(first.Pane, server) {
		t.Fatalf("pane %s died; the finished-but-alive case is what this test covers", first.Pane)
	}

	if _, err := runStart(t, "third prompt", "abcd", "apply", "--pane", "--server", server); err != nil {
		t.Fatalf("start over a completed pane attempt should succeed, got: %v", err)
	}
	second, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load after overwrite: %v", err)
	}
	if second.Pane == first.Pane {
		t.Errorf("record still names the old pane %s; the attempt was not overwritten", first.Pane)
	}
	// The completed attempt's stale result was cleared for the new run.
	if _, err := os.Stat(dispatch.ResultPath(dir, "apply")); !os.IsNotExist(err) {
		t.Error("stale result file should have been cleared by the overwrite")
	}
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("prompt not persisted: %v", err)
	}
	if string(promptData) != "third prompt" {
		t.Errorf("prompt = %q, want the new attempt's prompt", string(promptData))
	}
}

// TestDispatchStart_PaneAndHeadlessMutuallyExclusive: the two explicit mode flags
// name contradictory modes, so supplying both is a usage error enforced by cobra's
// flag group — i.e. during ValidateFlagGroups, BEFORE any RunE work, so nothing is
// launched and nothing is persisted.
func TestDispatchStart_PaneAndHeadlessMutuallyExclusive(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'exit 0'")

	_, err := runStart(t, "prompt", "abcd", "apply", "--pane", "--headless")
	if err == nil {
		t.Fatal("expected a usage error for --pane with --headless")
	}
	for _, want := range []string{"pane", "headless"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name %q", err.Error(), want)
		}
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the usage error, got %v", err)
	}
}

// TestDispatchStart_HeadlessFlagOptsOutOfAutoPane: inside tmux, auto selection
// would pick pane; --headless is the explicit opt-out for an unattended run living
// in a tmux tab. --timeout composes with it (unlike with --pane), since a timeout
// is exactly the headless wrapper's bound.
func TestDispatchStart_HeadlessFlagOptsOutOfAutoPane(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	t.Setenv("TMUX", "/tmp/tmux-1000/default,4242,0") // would auto-select pane

	out, err := runStart(t, "prompt", "abcd", "apply", "--headless", "--timeout", "600")
	if err != nil {
		t.Fatalf("--headless --timeout should compose: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() {
		t.Errorf("--headless inside tmux produced a pane record (%+v)", *rec)
	}
	if rec.Timeout != 600 {
		t.Errorf("timeout = %d, want 600", rec.Timeout)
	}
	// An EXPLICIT selection's report is byte-identical to before auto existed.
	if strings.Contains(out, "auto:") {
		t.Errorf("output = %q, want no auto-selection suffix for an explicit mode", out)
	}
}

// TestDispatchStart_AutoSelectsHeadlessOutsideTmux pins the pre-auto default as
// the auto outcome outside tmux — behaviorally byte-preserving apart from the
// output line's selection-source suffix.
func TestDispatchStart_AutoSelectsHeadlessOutsideTmux(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'") // clears $TMUX

	out, err := runStart(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.PID <= 0 {
		t.Errorf("auto outside tmux must be headless, got %+v", *rec)
	}
	if !strings.Contains(out, string(dispatch.ReasonAutoNoTmux)) {
		t.Errorf("output = %q, want the %q selection source", out, dispatch.ReasonAutoNoTmux)
	}
}

// TestDispatchStart_AutoPaneSoftFallsBackToHeadless: a STALE $TMUX (inherited from
// a killed server) must not break a dispatch that never asked for a pane. Auto
// selects pane, the reachability probe fails, and the dispatch degrades to
// headless with a one-line stderr notice — where an explicit --pane hard-errors.
func TestDispatchStart_AutoPaneSoftFallsBackToHeadless(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	// A private, empty TMUX_TMPDIR guarantees no server answers the default socket,
	// so the probe fails deterministically regardless of the test host's tmux.
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-stale"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	out, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("auto pane must soft-fall-back, not error: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(stderr, dispatch.FallbackNotice) {
		t.Errorf("stderr = %q, want the fallback notice %q", stderr, dispatch.FallbackNotice)
	}
	if !strings.Contains(out, string(dispatch.ReasonAutoUnreachable)) {
		t.Errorf("output = %q, want the %q selection source", out, dispatch.ReasonAutoUnreachable)
	}

	// The record is headless-SHAPED: the fallback re-composed dispatch_command and
	// launched the wrapper, so no pane identity leaks into the record.
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("fallback record reads as %q (pane=%q)", rec.Mode(), rec.Pane)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want positive", rec.PID, rec.PGID)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("fallback record carries pane identity: window=%q server=%q", rec.Window, rec.Server)
	}
	// The fallback composed dispatch_command, NOT session_command — the
	// no-cross-fallback rule survives the mode change.
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the dispatch_command as prefix", rec.SpawnCmd)
	}
}

// TestDispatchStart_AutoPaneFallsBackWhenProviderHasNoSessionCommand is soft
// fallback shape (b) with a STALE $TMUX: the pane path cannot proceed for TWO
// independent reasons at once (unreachable server AND no session_command), and
// either way an auto-selected pane must degrade rather than error. The probe fires
// first, so the shape-(a) notice is the one printed — what this test pins is that a
// dispatch_command-only provider does not hard-error on composition before the
// fallback decision point is even reached.
func TestDispatchStart_AutoPaneFallsBackWhenProviderHasNoSessionCommand(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "") // no session_command
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-nosess-stale"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	out, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("auto pane with a dispatch_command-only provider must soft-fall-back, not error: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(stderr, "falling back to headless") {
		t.Errorf("stderr = %q, want a soft-fallback notice", stderr)
	}
	if !strings.Contains(out, "auto: ") {
		t.Errorf("output = %q, want an auto-selection source suffix", out)
	}
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.PID <= 0 {
		t.Errorf("fallback record must be headless-shaped, got %+v", *rec)
	}
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the dispatch_command as prefix", rec.SpawnCmd)
	}
}

// TestDispatchStart_AutoPaneNoSessionCommand_Integration is soft fallback shape (b)
// in isolation, against a LIVE tmux server: the reachability probe PASSES, so the
// only reason the pane path cannot proceed is the missing session_command. This is
// the regression the review found — before pane-command composition was deferred,
// a dispatch_command-only provider that worked headless hard-errored inside live
// tmux, demanding a session_command it never needed.
//
// SAFETY: this test issues UNSCOPED (no `-L`) tmux calls on purpose, because
// auto-selected pane mode passes no `-L` and must reach the server through tmux's
// own default-socket resolution. It follows the same two-mechanism isolation as
// TestDispatchStart_AutoPaneMode_Integration and neither mechanism may be removed:
// a private TMUX_TMPDIR relocates the default socket, and $TMUX must be EMPTY while
// the server is created (a set $TMUX makes a client target ITS socket and ignore
// TMUX_TMPDIR, which would put the session on — and later kill — the real server).
// Every destructive call, kill-server included, is scoped by a VERIFIED `-S` path.
func TestDispatchStart_AutoPaneNoSessionCommand_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// dispatch_command only — the pane path has nothing to open interactively.
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "")

	socketDir := tmuxSocketDir(t, "default")
	t.Setenv("TMUX_TMPDIR", socketDir)
	if v := os.Getenv("TMUX"); v != "" {
		t.Fatalf("refusing to run: $TMUX = %q must be empty so TMUX_TMPDIR isolates "+
			"this test's unscoped tmux calls from the real server", v)
	}
	if out, err := exec.Command("tmux", "new-session", "-d", "-s", "s", "-x", "80", "-y", "24").CombinedOutput(); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, strings.TrimSpace(string(out)))
	}
	// Prove the server bound the PRIVATE socket before registering a kill-server
	// cleanup, and scope that cleanup to the verified path with an explicit `-S`.
	privateSocket := filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")
	if _, err := os.Stat(privateSocket); err != nil {
		t.Fatalf("refusing to continue: tmux did not bind the private socket %s (%v) — "+
			"the server may be the real one, and killing it is unsafe", privateSocket, err)
	}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", privateSocket, "kill-server").Run()
	})

	// Only NOW is $TMUX set — after the destructive cleanup is socket-scoped.
	t.Setenv("TMUX", privateSocket+",1,0")

	out, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("a dispatch_command-only provider must still dispatch headless inside live tmux, got: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	// The probe passed, so this is shape (b) specifically — the session_command
	// notice and reason, not the unreachable-tmux pair.
	if !strings.Contains(stderr, dispatch.FallbackNoticeNoSessionCommand) {
		t.Errorf("stderr = %q, want the shape-(b) notice %q", stderr, dispatch.FallbackNoticeNoSessionCommand)
	}
	if strings.Contains(stderr, dispatch.FallbackNotice) {
		t.Errorf("stderr = %q must not carry the unreachable-tmux notice: tmux WAS reachable", stderr)
	}
	if !strings.Contains(out, string(dispatch.ReasonAutoNoSessionCommand)) {
		t.Errorf("output = %q, want the %q selection source", out, dispatch.ReasonAutoNoSessionCommand)
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("fallback record reads as %q (pane=%q)", rec.Mode(), rec.Pane)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want positive", rec.PID, rec.PGID)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("fallback record carries pane identity: window=%q server=%q", rec.Window, rec.Server)
	}
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the dispatch_command as prefix", rec.SpawnCmd)
	}
	// No dispatch window was opened on the reachable server.
	if names, err := exec.Command("tmux", "-S", privateSocket,
		"list-windows", "-a", "-F", "#W").Output(); err == nil {
		if strings.Contains(string(names), dispatch.WindowName(id, "apply")) {
			t.Errorf("a dispatch window %q was opened despite the headless fallback", dispatch.WindowName(id, "apply"))
		}
	}
}

// TestDispatchStart_ExplicitPaneWithoutSessionCommandPersistsNothing is the
// explicit half of shape (b): the hard error is unchanged and — since composition
// now happens after validation — still leaves no dispatch record behind. tmux is
// deliberately REACHABLE here (an ephemeral server), so the missing session_command
// is the sole failure cause and the error cannot be the probe's.
//
// It is also the sole home of the no-cross-fallback rule's PANE half (folded in
// from the former TestDispatchStart_PaneRequiresSessionCommand, which asserted the
// same hint against the ambient default socket and so failed on a host with no
// tmux running): pane mode composes session_command and must never substitute
// dispatch_command, so the error names the stage's resolved tier and the exact
// session_command config key. Its headless mirror is
// TestDispatchStart_HeadlessStillRequiresDispatchCommand.
func TestDispatchStart_ExplicitPaneWithoutSessionCommandPersistsNothing(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "") // no session_command

	server := "fabtest-explicit-nosess"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	_, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply", "--pane", "--server", server)
	if err == nil {
		t.Fatal("explicit --pane must hard-error when the provider has no session_command")
	}
	if !strings.Contains(err.Error(), "providers.cli.session_command") {
		t.Errorf("error = %q, want the session_command config-key hint", err.Error())
	}
	if !strings.Contains(err.Error(), "doing") {
		t.Errorf("error = %q, want mention of the resolved tier", err.Error())
	}
	if strings.Contains(stderr, "falling back to headless") {
		t.Errorf("explicit --pane must not soft-fall-back, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the hard error, got %v", err)
	}
}

// TestDispatchStart_ExplicitPaneStillHardErrorsOnUnreachableTmux is the other half
// of the asymmetry: a caller who typed --pane asked for watchability, so a silent
// downgrade would defeat the request. Nothing is launched and nothing persisted.
func TestDispatchStart_ExplicitPaneStillHardErrorsOnUnreachableTmux(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-explicit"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	_, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply", "--pane")
	if err == nil {
		t.Fatal("explicit --pane must hard-error when tmux is unreachable")
	}
	if strings.Contains(stderr, dispatch.FallbackNotice) {
		t.Errorf("explicit --pane must not print the soft-fallback notice, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the hard error, got %v", err)
	}
}

// TestDispatchStart_AutoPaneMode_Integration drives the real AUTO path against an
// ephemeral tmux server: with $TMUX set to that server's socket and NO mode flag,
// the dispatch opens a window on the current server (no -L passed) and reports the
// `auto: tmux` selection source. Skipped when tmux is unavailable.
func TestDispatchStart_AutoPaneMode_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'echo \"got: $1\"; sleep 30' _`)

	// An ephemeral server on the DEFAULT socket name under a private TMUX_TMPDIR.
	// The default socket (rather than a `-L` label) is what makes this test
	// meaningful: auto-selected pane mode passes NO `-L`, so the dispatch must
	// reach the server through tmux's own default-socket resolution — exactly as a
	// real dispatch inside a user's tmux session does.
	//
	// SAFETY: this test is the one place in the suite that runs an UNSCOPED (no
	// `-L`) tmux new-session/kill-server, so socket isolation is the only thing
	// standing between its cleanup and a developer's real tmux server. Two
	// mechanisms enforce it, and neither may be removed:
	//   1. TMUX_TMPDIR points at a private per-test dir, which relocates tmux's
	//      default socket out of the shared /tmp/tmux-$UID.
	//   2. $TMUX must be EMPTY while the server is created and killed — a set
	//      $TMUX makes a client target ITS socket and ignore TMUX_TMPDIR
	//      entirely, which would put the session on (and later kill) the real
	//      server. setupDispatchRepoWithCommands neutralizes it above; the guard
	//      below re-asserts it rather than trusting call order, and $TMUX is only
	//      set AFTER the destructive cleanup is scoped by a verified socket path.
	socketDir := tmuxSocketDir(t, "default")
	t.Setenv("TMUX_TMPDIR", socketDir)
	if v := os.Getenv("TMUX"); v != "" {
		t.Fatalf("refusing to run: $TMUX = %q must be empty so TMUX_TMPDIR isolates "+
			"this test's unscoped tmux calls from the real server", v)
	}
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", args...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	// Prove the server we just started really bound the PRIVATE socket before
	// registering a kill-server cleanup, and pin every later call (including the
	// cleanup) to that verified path with an explicit `-S`. An unscoped
	// kill-server is never issued.
	privateSocket := filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")
	if _, err := os.Stat(privateSocket); err != nil {
		t.Fatalf("refusing to continue: tmux did not bind the private socket %s (%v) — "+
			"the server may be the real one, and killing it is unsafe", privateSocket, err)
	}
	tmuxScoped := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-S", privateSocket}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	t.Cleanup(func() { _, _ = tmuxScoped("kill-server") })

	// $TMUX's real shape is "<socket-path>,<pid>,<session-id>"; only its PRESENCE
	// drives selection, while the socket the dispatch actually reaches comes from
	// tmux's own default resolution under this TMUX_TMPDIR.
	t.Setenv("TMUX", privateSocket+",1,0")

	out, err := runStart(t, "auto prompt\n", "abcd", "apply")
	if err != nil {
		t.Fatalf("auto pane start failed: %v", err)
	}
	if !strings.Contains(out, "pane ") || !strings.Contains(out, string(dispatch.ReasonAutoTmux)) {
		t.Errorf("output = %q, want a pane report carrying the %q source", out, dispatch.ReasonAutoTmux)
	}

	dir := dispatch.DirFor(repoRoot, id)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if !rec.IsPane() {
		t.Fatalf("auto inside tmux must be a pane dispatch, got %+v", *rec)
	}
	// Auto-selected pane targets the CURRENT server — no --server was given, so
	// none is recorded (status/kill inherit the default-socket resolution).
	if rec.Server != "" {
		t.Errorf("server = %q, want empty (auto targets the current server)", rec.Server)
	}
	if want := dispatch.WindowName(id, "apply"); rec.Window != want {
		t.Errorf("window = %q, want %q", rec.Window, want)
	}
	// The window really exists on that server (read through the verified socket).
	windowName, err := tmuxScoped("display-message", "-p", "-t", rec.Pane, "#W")
	if err != nil {
		t.Fatalf("read window name: %v", err)
	}
	if windowName != dispatch.WindowName(id, "apply") {
		t.Errorf("tmux window name = %q, want %q", windowName, dispatch.WindowName(id, "apply"))
	}
}

// TestDispatchStart_ServerFlagImpliesPaneUnderAuto: --server exists solely to
// target a pane's socket, so naming one selects pane mode even with $TMUX unset.
// Reaching the reachability error (rather than a headless launch) is what proves
// the pane branch was taken.
func TestDispatchStart_ServerFlagImpliesPaneUnderAuto(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	server := "fabtest-implies-pane"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server)) // no server on this socket

	_, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply", "--server", server)
	if err == nil {
		t.Fatal("--server with no --pane/--headless must select pane and hit the reachability error")
	}
	if !strings.Contains(err.Error(), "tmux") {
		t.Errorf("error = %q, want the tmux reachability error (proving the pane branch)", err.Error())
	}
	// --server is an EXPLICIT pane signal, so the failure is hard, not a fallback.
	if strings.Contains(stderr, dispatch.FallbackNotice) {
		t.Errorf("--server is an explicit pane signal; it must not soft-fall-back, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist, got %v", err)
	}
}

// startPrivateTmuxWithPane starts an ephemeral tmux server on a PRIVATE socket,
// verifies the socket really is private, registers a socket-scoped teardown, and
// then points $TMUX/$TMUX_PANE at that server's first pane — the environment a
// worktree AGENT dispatching its own stage workers actually has. It returns a
// tmux runner pinned to the verified socket path and the dispatcher's pane ID.
//
// SAFETY (the recorded repo discipline — none of these steps is optional):
//  1. $TMUX must be EMPTY on entry. A set $TMUX makes every tmux client target
//     ITS socket and ignore TMUX_TMPDIR entirely, which would put this test's
//     session on — and later KILL — the developer's real server. Hard-fail, never
//     skip: skipping would silently stop covering the split path.
//  2. TMUX_TMPDIR points at a private per-test dir, relocating tmux's default
//     socket out of the shared /tmp/tmux-$UID.
//  3. The server's binding of that private socket is VERIFIED (os.Stat) before any
//     destructive cleanup is registered — if tmux bound something else, the server
//     may be the real one and killing it is unsafe.
//  4. Every later call, kill-server included, is scoped by explicit `-S <verified
//     path>`. No unscoped kill-server is ever issued.
//
// The default socket NAME (rather than a `-L` label) is deliberate: the split path
// is only reachable without `--server`, so the dispatch must find the server
// through tmux's own default-socket resolution under this TMUX_TMPDIR — exactly as
// a real agent's dispatch inside its own tmux session does.
func startPrivateTmuxWithPane(t *testing.T) (tmuxScoped func(args ...string) (string, error), dispatcherPane string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	socketDir := tmuxSocketDir(t, "default")
	t.Setenv("TMUX_TMPDIR", socketDir)
	if v := os.Getenv("TMUX"); v != "" {
		t.Fatalf("refusing to run: $TMUX = %q must be empty so TMUX_TMPDIR isolates "+
			"this test's unscoped tmux calls from the real server", v)
	}
	if out, err := exec.Command("tmux", "new-session", "-d", "-s", "s", "-x", "200", "-y", "50").CombinedOutput(); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, strings.TrimSpace(string(out)))
	}
	privateSocket := filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")
	if _, err := os.Stat(privateSocket); err != nil {
		t.Fatalf("refusing to continue: tmux did not bind the private socket %s (%v) — "+
			"the server may be the real one, and killing it is unsafe", privateSocket, err)
	}
	tmuxScoped = func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-S", privateSocket}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	t.Cleanup(func() { _, _ = tmuxScoped("kill-server") })

	// The session's sole pane stands in for the dispatching agent's own pane.
	dispatcherPane, err := tmuxScoped("list-panes", "-t", "s", "-F", "#{pane_id}")
	if err != nil || dispatcherPane == "" {
		t.Fatalf("could not read the dispatcher pane id: %v (%q)", err, dispatcherPane)
	}

	// $TMUX/$TMUX_PANE are set only NOW — after the destructive cleanup is scoped by
	// the verified socket path. $TMUX's real shape is "<socket>,<pid>,<session>";
	// only its PRESENCE drives mode selection, while $TMUX_PANE is the split anchor.
	t.Setenv("TMUX", privateSocket+",1,0")
	t.Setenv("TMUX_PANE", dispatcherPane)
	return tmuxScoped, dispatcherPane
}

// paneWindow reads a pane's window id through the verified socket.
func paneWindow(t *testing.T, tmuxScoped func(...string) (string, error), paneID string) string {
	t.Helper()
	out, err := tmuxScoped("display-message", "-p", "-t", paneID, "#{window_id}")
	if err != nil {
		t.Fatalf("read window id for %s: %v (%q)", paneID, err, out)
	}
	return out
}

// paneTitle reads a pane's title through the verified socket.
func paneTitle(t *testing.T, tmuxScoped func(...string) (string, error), paneID string) string {
	t.Helper()
	out, err := tmuxScoped("display-message", "-p", "-t", paneID, "#{pane_title}")
	if err != nil {
		t.Fatalf("read pane title for %s: %v (%q)", paneID, err, out)
	}
	return out
}

// TestDispatchStart_SplitPane_Integration is the change's core case: a dispatcher
// that IS a tmux pane gets its stage worker as a PANE SPLIT INTO ITS OWN WINDOW,
// not as a separate window. The worker's identity rides the pane TITLE (a split
// pane has no window name of its own), the pane ID is what the record stores, and
// the dispatcher's window count is unchanged — which is the reported problem this
// change fixes (every worker used to surface as another run-kit window).
func TestDispatchStart_SplitPane_Integration(t *testing.T) {
	// TEMPLATED with {model}/{effort} (as the built-in claude session_command is), so
	// spawn.WithProfile SUBSTITUTES rather than appending flags after the prompt
	// argument — which would otherwise make $1 the appended `--model`.
	repoRoot, id := setupDispatchRepoWithCommands(t, "",
		`sh -c 'echo \"m={model} e={effort}\"; echo \"got: $1\"; sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)

	windowsBefore, err := tmuxScoped("list-windows", "-a", "-F", "#{window_id}")
	if err != nil {
		t.Fatalf("list windows: %v", err)
	}

	out, err := runStart(t, "the stage prompt\n", "abcd", "apply")
	if err != nil {
		t.Fatalf("split dispatch failed: %v", err)
	}
	// The split shape's own report: "split, title …" rather than "window …".
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "split") || !strings.Contains(out, "title "+title) {
		t.Errorf("output = %q, want the split report naming the pane title %q", out, title)
	}
	// Auto selection still explains itself.
	if !strings.Contains(out, string(dispatch.ReasonAutoTmux)) {
		t.Errorf("output = %q, want the %q selection source", out, dispatch.ReasonAutoTmux)
	}

	rec, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if !rec.IsPane() || rec.Mode() != dispatch.ModePane {
		t.Fatalf("record reads as %q, want pane mode (%+v)", rec.Mode(), *rec)
	}
	// The record keeps storing the SAME identity string — no schema change.
	if rec.Window != title {
		t.Errorf("window = %q, want the identity string %q", rec.Window, title)
	}

	// THE CLAIM: the worker's pane lives in the DISPATCHER's window.
	if got, want := paneWindow(t, tmuxScoped, rec.Pane), paneWindow(t, tmuxScoped, dispatcherPane); got != want {
		t.Errorf("worker pane %s is in window %s, want the dispatcher's window %s", rec.Pane, got, want)
	}
	// And no new window was opened.
	windowsAfter, err := tmuxScoped("list-windows", "-a", "-F", "#{window_id}")
	if err != nil {
		t.Fatalf("list windows: %v", err)
	}
	if windowsAfter != windowsBefore {
		t.Errorf("window list changed from %q to %q; a split dispatch must open NO window", windowsBefore, windowsAfter)
	}
	if names, err := tmuxScoped("list-windows", "-a", "-F", "#W"); err == nil && strings.Contains(names, title) {
		t.Errorf("a window named %q exists (%q); the identity must ride the pane title in the split shape", title, names)
	}

	// Identity rides the pane title, unmarked by the operator's prefixes.
	if got := paneTitle(t, tmuxScoped, rec.Pane); got != title {
		t.Errorf("pane title = %q, want %q", got, title)
	}
	for _, marker := range []string{"»", "›"} {
		if strings.Contains(title, marker) {
			t.Errorf("pane title %q must not carry the operator marker %q", title, marker)
		}
	}

	// The pane is alive and received the POINTER, not the prompt body — the pane-id
	// keyed probes work identically in the split shape.
	if !dispatch.PaneAlive(rec.Pane, "") {
		t.Errorf("pane %s should be alive right after start", rec.Pane)
	}
	captured, err := tmuxScoped("capture-pane", "-p", "-t", rec.Pane)
	if err != nil {
		t.Fatalf("capture pane: %v", err)
	}
	if !strings.Contains(captured, ".fab-dispatch/"+id+"/apply-prompt.md") {
		t.Errorf("pane received %q, want the pointer naming the prompt file", captured)
	}
}

// TestDispatchStart_SplitPanesStackInTheRightColumn is the placement rule: with a
// live recorded sibling already present, the SECOND dispatch splits THAT pane rather
// than the dispatcher's, so workers stack down a right-hand column instead of each
// one halving the dispatcher's pane again. All three panes share one window.
//
// REGRESSION (260807-g4a5): the first worker's pane TITLE is deliberately clobbered
// before the second dispatch, reproducing what a harness running inside the worker
// does within seconds of spawn (Claude Code rewrites the pane title via terminal
// escapes). The title-keyed probe this replaced found nothing in that state, so every
// later worker re-split the dispatcher and carved another full-height column until
// the dispatching agent was a sliver. Record-keyed detection is title-independent, so
// stacking must survive the clobber — this test fails against the old implementation.
func TestDispatchStart_SplitPanesStackInTheRightColumn(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)
	dir := dispatch.DirFor(repoRoot, id)
	dispatcherWindow := paneWindow(t, tmuxScoped, dispatcherPane)

	if _, err := runStart(t, "apply prompt", "abcd", "apply"); err != nil {
		t.Fatalf("first split dispatch failed: %v", err)
	}
	first, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load apply: %v", err)
	}
	if !dispatch.PaneAlive(first.Pane, "") {
		t.Fatalf("pane %s must be alive for the sibling probe to find it", first.Pane)
	}

	// The clobber: the worker's harness owns its pane title from here on.
	if out, err := tmuxScoped("select-pane", "-t", first.Pane, "-T", "✳ some harness title"); err != nil {
		t.Fatalf("could not clobber the worker's pane title: %v (%q)", err, out)
	}

	// A DIFFERENT stage, so this is a second concurrent worker rather than an
	// overwrite of the first. It must be the OTHER doing-tier stage (review-pr):
	// the fixture points only the doing tier at the `cli` provider, so a stage
	// outside it (e.g. review → the review tier) resolves the BUILT-IN claude
	// provider and launches the real `claude` CLI — alive on a dev box where
	// claude exists (a false local pass, plus a stray real agent session in the
	// test server), dead instantly on CI where it doesn't.
	if _, err := runStart(t, "review-pr prompt", "abcd", "review-pr"); err != nil {
		t.Fatalf("second split dispatch failed: %v", err)
	}
	second, err := dispatch.Load(dir, "review-pr")
	if err != nil {
		t.Fatalf("Load review-pr: %v", err)
	}

	// All three panes in ONE window: the dispatcher plus both workers.
	for _, p := range []struct{ label, id string }{{"first worker", first.Pane}, {"second worker", second.Pane}} {
		if got := paneWindow(t, tmuxScoped, p.id); got != dispatcherWindow {
			t.Errorf("%s pane %s is in window %s, want the dispatcher's window %s", p.label, p.id, got, dispatcherWindow)
		}
	}
	panes, err := tmuxScoped("list-panes", "-t", dispatcherPane, "-F", "#{pane_id}")
	if err != nil {
		t.Fatalf("list panes: %v", err)
	}
	if n := len(strings.Split(panes, "\n")); n != 3 {
		t.Errorf("window holds %d panes (%q), want 3 (dispatcher + two workers)", n, panes)
	}

	// The stacking rule: the second worker split the FIRST WORKER, not the
	// dispatcher — so it sits directly below the first worker and shares its left
	// edge, while the dispatcher (which was split horizontally) does not.
	firstLeft := paneFormat(t, tmuxScoped, first.Pane, "#{pane_left}")
	secondLeft := paneFormat(t, tmuxScoped, second.Pane, "#{pane_left}")
	dispatcherLeft := paneFormat(t, tmuxScoped, dispatcherPane, "#{pane_left}")
	if secondLeft != firstLeft {
		t.Errorf("second worker's left edge = %s, want the first worker's %s (it must split the SIBLING, stacking the column)",
			secondLeft, firstLeft)
	}
	if firstLeft == dispatcherLeft {
		t.Errorf("worker column left edge %s equals the dispatcher's %s; the first split must be horizontal (-h), carving a right column",
			firstLeft, dispatcherLeft)
	}
	if paneFormat(t, tmuxScoped, second.Pane, "#{pane_top}") == paneFormat(t, tmuxScoped, first.Pane, "#{pane_top}") {
		t.Error("both workers share a top edge; the second split must be vertical (-v), stacking below the first")
	}

	// Titles are still SET at spawn — for identification only, now that placement no
	// longer reads them. The second worker's is untouched (only the first was
	// clobbered above), and the clobbered one proves the point: placement found it
	// anyway.
	if got := paneTitle(t, tmuxScoped, second.Pane); got != dispatch.WindowName(id, "review-pr") {
		t.Errorf("second worker's pane title = %q, want %q", got, dispatch.WindowName(id, "review-pr"))
	}
	if got := paneTitle(t, tmuxScoped, first.Pane); got == dispatch.WindowName(id, "apply") {
		t.Error("the first worker's title was expected to STAY clobbered — the regression scenario did not hold")
	}

	// Killing one worker pane leaves the dispatcher's window (and the other worker)
	// intact — plain tmux kill-pane semantics, which is why KillPane needed no change.
	if err := dispatch.KillPane(second.Pane, ""); err != nil {
		t.Fatalf("KillPane: %v", err)
	}
	if dispatch.PaneAlive(second.Pane, "") {
		t.Errorf("pane %s should be gone after KillPane", second.Pane)
	}
	if !dispatch.PaneAlive(dispatcherPane, "") {
		t.Error("killing a split worker pane must leave the dispatcher's pane alive")
	}
	if !dispatch.PaneAlive(first.Pane, "") {
		t.Error("killing one worker pane must leave the sibling worker alive")
	}
	if got := paneWindow(t, tmuxScoped, dispatcherPane); got != dispatcherWindow {
		t.Errorf("dispatcher window changed to %s, want %s intact", got, dispatcherWindow)
	}
}

// TestDispatchStart_UnreadableRecordWarnsAndStillLaunches is the record-read half of
// the degradation contract (the tmux-probe half is covered by the sized-split retry
// tests): a corrupt {stage}.yaml in the checkout's dispatch tree must reach
// launchPane's stderr warning AND still launch the worker. Before this, the record
// walk swallowed every read failure, so a broken tree silently produced blind
// placement with nothing in the output to explain it.
//
// A live recorded sibling sits alongside the corrupt record, so the test also pins
// the partial-set behavior: the readable record still wins the intersection and the
// worker STACKS — a read failure degrades the probe, it does not discard it.
func TestDispatchStart_UnreadableRecordWarnsAndStillLaunches(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)
	dir := dispatch.DirFor(repoRoot, id)

	if _, err := runStart(t, "apply prompt", "abcd", "apply"); err != nil {
		t.Fatalf("first split dispatch failed: %v", err)
	}
	first, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load apply: %v", err)
	}

	// A record the walk cannot parse, in a second change's dir so it neither
	// overwrites nor is overwritten by the dispatch under test.
	corrupt := dispatch.DirFor(repoRoot, "zzzz")
	if err := os.MkdirAll(corrupt, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dispatch.YAMLPath(corrupt, "apply"), "pane: [unterminated\n")

	_, stderr, err := runStartCapturingStderr(t, "review-pr prompt", "abcd", "review-pr")
	if err != nil {
		t.Fatalf("a corrupt record must degrade placement, not fail the dispatch: %v", err)
	}
	if !strings.Contains(stderr, "worker-column placement probe failed") {
		t.Errorf("stderr = %q, want the degraded-probe warning", stderr)
	}
	// The warning must name WHAT failed and WHERE the worker went, so the degraded
	// placement is explainable from output alone.
	for _, want := range []string{"zzzz/apply.yaml", "stacking the worker under pane " + first.Pane} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want it to name %q", stderr, want)
		}
	}

	second, err := dispatch.Load(dir, "review-pr")
	if err != nil {
		t.Fatalf("the worker must still have launched and been recorded: %v", err)
	}
	if !dispatch.PaneAlive(second.Pane, "") {
		t.Errorf("worker pane %s is not alive; the degraded probe must not cost the dispatch", second.Pane)
	}
	// The readable sibling still won the intersection: same left edge, distinct top.
	if got, want := paneFormat(t, tmuxScoped, second.Pane, "#{pane_left}"),
		paneFormat(t, tmuxScoped, first.Pane, "#{pane_left}"); got != want {
		t.Errorf("second worker's left edge = %s, want the sibling's %s (a partial record read still stacks)", got, want)
	}
	if paneFormat(t, tmuxScoped, second.Pane, "#{pane_top}") == paneFormat(t, tmuxScoped, first.Pane, "#{pane_top}") {
		t.Error("both workers share a top edge; the stacking split must be vertical (-v)")
	}
	if got := paneWindow(t, tmuxScoped, second.Pane); got != paneWindow(t, tmuxScoped, dispatcherPane) {
		t.Errorf("worker landed in window %s, want the dispatcher's", got)
	}
}

// paneFormat reads any tmux format string for a pane through the verified socket.
func paneFormat(t *testing.T, tmuxScoped func(...string) (string, error), paneID, format string) string {
	t.Helper()
	out, err := tmuxScoped("display-message", "-p", "-t", paneID, format)
	if err != nil {
		t.Fatalf("read %s for %s: %v (%q)", format, paneID, err, out)
	}
	return out
}

// paneInt reads a numeric tmux format string for a pane.
func paneInt(t *testing.T, tmuxScoped func(...string) (string, error), paneID, format string) int {
	t.Helper()
	raw := paneFormat(t, tmuxScoped, paneID, format)
	n, err := strconv.Atoi(raw)
	if err != nil {
		t.Fatalf("%s for %s is not numeric: %q", format, paneID, raw)
	}
	return n
}

// appendProjectConfig appends to the repo's project config, for tests that need a
// key the shared fixture does not write.
func appendProjectConfig(t *testing.T, repoRoot, extra string) {
	t.Helper()
	path := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(body)+extra)
}

// TestDispatchStart_CarvingSplitSizesTheWorkerColumn is the sizing rule: the
// column-CARVING split runs `-l <n>%`, so the dispatching agent — the pane the user
// is actually watching — keeps the rest of the window instead of being halved. The
// width comes from `dispatch.column_width` (default 35), and the STACKING split that
// follows leaves the left/right separator untouched: the column invariant.
func TestDispatchStart_CarvingSplitSizesTheWorkerColumn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra string
		want  int
	}{
		{"default width", "", config.DefaultDispatchColumnWidth},
		{"configured width", "dispatch:\n  column_width: 20\n", 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
			if tc.extra != "" {
				appendProjectConfig(t, repoRoot, tc.extra)
			}
			tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)
			dir := dispatch.DirFor(repoRoot, id)
			windowWidth := paneInt(t, tmuxScoped, dispatcherPane, "#{window_width}")

			if _, err := runStart(t, "apply prompt", "abcd", "apply"); err != nil {
				t.Fatalf("split dispatch failed: %v", err)
			}
			first, err := dispatch.Load(dir, "apply")
			if err != nil {
				t.Fatalf("Load apply: %v", err)
			}

			// tmux sizes the NEW pane to the requested percentage of the window; the
			// ±1 tolerance absorbs integer rounding and the separator column.
			gotWidth := paneInt(t, tmuxScoped, first.Pane, "#{pane_width}")
			wantWidth := windowWidth * tc.want / 100
			if gotWidth < wantWidth-1 || gotWidth > wantWidth+1 {
				t.Errorf("worker column is %d cols of a %d-col window, want ~%d (%d%%)",
					gotWidth, windowWidth, wantWidth, tc.want)
			}
			// The point of the sizing: the dispatcher keeps the majority.
			dispatcherWidth := paneInt(t, tmuxScoped, dispatcherPane, "#{pane_width}")
			if dispatcherWidth <= gotWidth {
				t.Errorf("dispatcher kept %d cols vs the worker's %d; a sized carve must leave the dispatcher more",
					dispatcherWidth, gotWidth)
			}

			// A second worker STACKS: the column's width — and therefore the
			// dispatcher's — is unchanged, because `-v` never moves the left/right
			// separator.
			if _, err := runStart(t, "review-pr prompt", "abcd", "review-pr"); err != nil {
				t.Fatalf("second split dispatch failed: %v", err)
			}
			if got := paneInt(t, tmuxScoped, dispatcherPane, "#{pane_width}"); got != dispatcherWidth {
				t.Errorf("dispatcher width changed to %d after a stacking split, want %d unchanged (the column invariant)",
					got, dispatcherWidth)
			}
			second, err := dispatch.Load(dir, "review-pr")
			if err != nil {
				t.Fatalf("Load review-pr: %v", err)
			}
			if got := paneInt(t, tmuxScoped, second.Pane, "#{pane_width}"); got != gotWidth {
				t.Errorf("stacked worker is %d cols wide, want the column's %d (a `-v` split must not resize the column)",
					got, gotWidth)
			}
		})
	}
}

// TestDispatchStart_ServerFlagKeepsTheNewWindowShape: `--server <name>` targets a
// possibly DIFFERENT tmux socket, on which the caller's own $TMUX_PANE id is
// meaningless — so naming one keeps the pre-split new-window shape even when the
// caller is itself sitting in a pane. $TMUX_PANE is deliberately set to a pane id
// from ANOTHER (private, `-L`-labelled) server here, which is exactly the
// cross-socket confusion the rule prevents.
func TestDispatchStart_ServerFlagKeepsTheNewWindowShape(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)

	server := "fabtest-splitsrv"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	// A pane id that exists on THIS server but is not the target of a --server run
	// on a different socket; setting it proves --server, not its emptiness, is what
	// keeps the window shape.
	t.Setenv("TMUX_PANE", "%0")

	out, err := runStart(t, "prompt", "abcd", "apply", "--pane", "--server", server)
	if err != nil {
		t.Fatalf("--server pane dispatch failed: %v", err)
	}
	title := dispatch.WindowName(id, "apply")
	// Byte-identical to the pre-split report shape.
	if !strings.Contains(out, "window "+title) || strings.Contains(out, "split") {
		t.Errorf("output = %q, want the unchanged new-window report %q", out, "window "+title)
	}

	rec, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	// A real WINDOW carries the name (the window shape's carrier).
	windowName, err := tmux("display-message", "-p", "-t", rec.Pane, "#W")
	if err != nil {
		t.Fatalf("read window name: %v", err)
	}
	if windowName != title {
		t.Errorf("tmux window name = %q, want %q — --server must keep the new-window shape", windowName, title)
	}
	// And that window is the pane's ONLY pane: nothing was split.
	panes, err := tmux("list-panes", "-t", rec.Pane, "-F", "#{pane_id}")
	if err != nil {
		t.Fatalf("list panes: %v", err)
	}
	if panes != rec.Pane {
		t.Errorf("worker window holds panes %q, want only the worker's own %q", panes, rec.Pane)
	}
}

// TestDispatchStart_NoTmuxPaneKeepsTheNewWindowShape: a headless orchestrator
// dispatching with an explicit --pane has no pane of its own to split, so the
// new-window shape is unchanged — byte-identically, report included. This is the
// pre-split behavior TestDispatchStart_PaneMode_Integration already covers via
// --server; this test pins the OTHER window rung (no $TMUX_PANE) on the DEFAULT
// socket, where a stray $TMUX_PANE would otherwise be honored.
func TestDispatchStart_NoTmuxPaneKeepsTheNewWindowShape(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, _ := startPrivateTmuxWithPane(t)
	// The orchestrator is not a pane: unset the anchor the helper just set.
	t.Setenv("TMUX_PANE", "")

	out, err := runStart(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "window "+title) || strings.Contains(out, "split") {
		t.Errorf("output = %q, want the unchanged new-window report %q", out, "window "+title)
	}

	rec, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	windowName, err := tmuxScoped("display-message", "-p", "-t", rec.Pane, "#W")
	if err != nil {
		t.Fatalf("read window name: %v", err)
	}
	if windowName != title {
		t.Errorf("tmux window name = %q, want %q — an unset $TMUX_PANE must keep the new-window shape", windowName, title)
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
