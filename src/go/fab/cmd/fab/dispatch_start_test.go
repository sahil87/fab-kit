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
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

// setupDispatchRepo builds a repo with one active change and a config whose
// `doing` role (apply's role) points at a provider carrying a headless_command,
// then chdirs into it so resolve.FabRoot() resolves. When dispatchCmd is empty,
// the resolved provider is the native-capable built-in claude. Returns the repo
// root and the 4-char change ID.
func setupDispatchRepo(t *testing.T, dispatchCmd string) (repoRoot, id string) {
	t.Helper()
	return setupDispatchRepoWithCommands(t, dispatchCmd, "")
}

// setupDispatchRepoWithCommands is setupDispatchRepo with independent control of
// the resolved provider's TWO command fields, so the pane path (interactive_command)
// and the headless path (headless_command) can be exercised separately —
// including the case that proves there is no cross-substitution in either direction
// (one field set, the other empty). An empty pair leaves the built-in claude
// provider resolved (which carries native and both command capabilities).
func setupDispatchRepoWithCommands(t *testing.T, dispatchCmd, sessionCmd string) (repoRoot, id string) {
	t.Helper()
	// Neutralize $TMUX so pane-preference tests are HERMETIC: a test suite run from
	// inside a tmux pane would otherwise treat pane as available. Tests that
	// exercise pane availability re-set it themselves (t.Setenv is per-test and
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
	if sessionCmd != "" {
		body += "dispatch:\n  mode: pane\n"
	}
	if dispatchCmd != "" || sessionCmd != "" {
		// A cli provider carries the command field(s); the doing role points at it.
		body += "providers:\n  cli:\n"
		if dispatchCmd != "" {
			body += "    headless_command: \"" + dispatchCmd + "\"\n"
		}
		if sessionCmd != "" {
			body += "    interactive_command: \"" + sessionCmd + "\"\n"
		}
		// The role profile PINS the `doing` role's built-in model/effort explicitly.
		// Model and effort come from the RESOLVED provider's own per-role fills, and
		// `cli` is a project-defined provider with none, so an unpinned role would
		// resolve them empty. Pinning the same values the built-in carries keeps
		// every dispatch test's expectation ("the resolved `doing` profile rides the
		// spawn command") true and still derived from the canonical defaults, so a
		// model bump does not touch these tests.
		doingDefault, _ := agent.DefaultProfile(agent.RoleDoing)
		body += "agent:\n  profiles:\n    doing: { provider: cli, model: " + doingDefault.Model + ", effort: " + doingDefault.Effort + " }\n"
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
// automatic descent notices.
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
	// command (append mode), so the persisted spawn_cmd carries the `doing` role's
	// profile appended to the base command. The model is derived from the `doing`
	// role's built-in default (pinned once in agent.TestDefaultRoleProfilesArePinned)
	// so a model bump does not touch this test.
	doingDefault, _ := agent.DefaultProfile(agent.RoleDoing)
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the base command as prefix", rec.SpawnCmd)
	}
	if !strings.Contains(rec.SpawnCmd, "--model "+doingDefault.Model) {
		t.Errorf("spawn_cmd = %q, want the resolved `doing` role model appended", rec.SpawnCmd)
	}
}

func TestDispatchStart_NativeSelectionRequiresResolveAgent(t *testing.T) {
	setupDispatchRepo(t, "") // built-in claude resolves to native at the default preference

	_, err := runStart(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("expected native dispatch to be delegated to resolve-agent")
	}
	msg := err.Error()
	for _, want := range []string{"native", "fab resolve-agent apply --alias"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want %q", msg, want)
		}
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

	_, err := runStart(t, "prompt", "abcd", "apply", "--headless")
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

// TestDispatchStart_PaneFlagsAreRetiredWithARoute: `start` launches only the
// headless arm, so the two pane flags it used to accept are answered with the route
// to the verb that now owns pane mode. They stay REGISTERED (hidden) precisely so
// this guidance is reachable — a removed flag would only produce cobra's bare
// `unknown flag`, which tells a caller nothing about the open/ready/deliver
// sequence that replaced the single-shot launch.
//
// The guard fires before any launch or state write, whatever else was typed
// alongside it.
func TestDispatchStart_PaneFlagsAreRetiredWithARoute(t *testing.T) {
	for _, args := range [][]string{
		{"--pane"},
		{"--server", "work"},
		{"--pane", "--timeout", "600"},
		{"--pane", "--headless"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'exit 0'")

			_, err := runStart(t, "prompt", append([]string{"abcd", "apply"}, args...)...)
			if err == nil {
				t.Fatal("expected a refusal naming the pane verb")
			}
			for _, want := range []string{"fab dispatch open", "fab dispatch ready", "fab dispatch deliver"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error = %q, want it to name %q", err.Error(), want)
				}
			}
			// Nothing launched, nothing persisted.
			if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
				t.Errorf("no dispatch record should exist after the refusal, got %v", err)
			}
		})
	}
}

// TestDispatchStart_HeadlessStillRequiresHeadlessCommand is the other half of the
// no-cross-substitution rule: a provider carrying ONLY an interactive_command must not
// satisfy a headless start.
func TestDispatchStart_HeadlessStillRequiresHeadlessCommand(t *testing.T) {
	setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")

	_, err := runStart(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error: an interactive_command must not satisfy a headless start")
	}
	if !strings.Contains(err.Error(), "providers.cli.headless_command") {
		t.Errorf("error = %q, want the headless_command config-key hint", err.Error())
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

// TestDispatchStart_NativePreferenceDescendsToHeadless pins the default-mode
// descent when a custom provider has no native capability.
func TestDispatchStart_NativePreferenceDescendsToHeadless(t *testing.T) {
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
		t.Errorf("native-unavailable descent must be headless, got %+v", *rec)
	}
	wantReason := "mode: headless (descended: native unavailable)"
	if !strings.Contains(out, wantReason) {
		t.Errorf("output = %q, want the %q selection source", out, wantReason)
	}
}

// TestDispatchStart_PanePreferenceDescendsOnUnreachableTmux: a STALE $TMUX
// (inherited from a killed server) must not break a pane-preferred dispatch. The
// reachability probe fails and selection descends to headless with a one-line
// stderr notice, whereas an explicit --pane hard-errors.
func TestDispatchStart_PanePreferenceDescendsOnUnreachableTmux(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	// A private, empty TMUX_TMPDIR guarantees no server answers the default socket,
	// so the probe fails deterministically regardless of the test host's tmux.
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-stale"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	out, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("pane preference must descend, not error: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(stderr, "tmux unreachable") {
		t.Errorf("stderr = %q, want the tmux-unreachable descent notice", stderr)
	}
	wantReason := "mode: headless (descended: pane unavailable: tmux unreachable; native unavailable)"
	if !strings.Contains(out, wantReason) {
		t.Errorf("output = %q, want the %q selection source", out, wantReason)
	}

	// The record is headless-SHAPED: descent re-composed headless_command and
	// launched the wrapper, so no pane identity leaks into the record.
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("descended record reads as %q (pane=%q)", rec.Mode(), rec.Pane)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want positive", rec.PID, rec.PGID)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("descended record carries pane identity: window=%q server=%q", rec.Window, rec.Server)
	}
	// The descended mode composed headless_command, NOT interactive_command — the
	// no-cross-substitution rule survives the mode change.
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the headless_command as prefix", rec.SpawnCmd)
	}
}

// TestDispatchStart_PanePreferenceDescendsWithoutInteractiveCommand is descent shape
// (b) with a STALE $TMUX: the pane path cannot proceed for TWO
// independent reasons at once (unreachable server AND no interactive_command), and
// either way a pane preference must descend rather than error. The probe fires
// first, so the shape-(a) notice is the one printed — what this test pins is that a
// headless_command-only provider does not hard-error on composition before the
// descent decision point is even reached.
func TestDispatchStart_PanePreferenceDescendsWithoutInteractiveCommand(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "") // no interactive_command
	appendProjectConfig(t, repoRoot, "dispatch:\n  mode: pane\n")
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-nosess-stale"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	out, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("pane preference with a headless_command-only provider must descend, not error: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(stderr, "no interactive_command") {
		t.Errorf("stderr = %q, want the missing-session descent notice", stderr)
	}
	if !strings.Contains(out, "mode: headless (descended:") {
		t.Errorf("output = %q, want the descent source suffix", out)
	}
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.PID <= 0 {
		t.Errorf("descended record must be headless-shaped, got %+v", *rec)
	}
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the headless_command as prefix", rec.SpawnCmd)
	}
}

// TestDispatchStart_PanePreferenceNoInteractiveCommand_Integration is descent shape (b)
// in isolation, against a LIVE tmux server: the reachability probe PASSES, so the
// only reason the pane path cannot proceed is the missing interactive_command. This is
// the regression the review found — before pane-command composition was deferred,
// a headless_command-only provider that worked headless hard-errored inside live
// tmux, demanding an interactive_command it never needed.
//
// SAFETY: this test issues UNSCOPED (no `-L`) tmux calls on purpose, because
// pane-preferred mode passes no `-L` and must reach the server through tmux's
// own default-socket resolution. It follows the same two-mechanism isolation as
// TestDispatchStart_PanePreferenceMode_Integration and neither mechanism may be removed:
// a private TMUX_TMPDIR relocates the default socket, and $TMUX must be EMPTY while
// the server is created (a set $TMUX makes a client target ITS socket and ignore
// TMUX_TMPDIR, which would put the session on — and later kill — the real server).
// Every destructive call, kill-server included, is scoped by a VERIFIED `-S` path.
func TestDispatchStart_PanePreferenceNoInteractiveCommand_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// headless_command only — the pane path has nothing to open interactively.
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "")
	appendProjectConfig(t, repoRoot, "dispatch:\n  mode: pane\n")

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
		t.Fatalf("a headless_command-only provider must still dispatch headless inside live tmux, got: %v", err)
	}
	dir := dispatch.DirFor(repoRoot, id)
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	// The probe passed, so this is shape (b) specifically — the interactive_command
	// notice and reason, not the unreachable-tmux pair.
	if !strings.Contains(stderr, "no interactive_command") {
		t.Errorf("stderr = %q, want the no-session descent notice", stderr)
	}
	if strings.Contains(stderr, "tmux unreachable") {
		t.Errorf("stderr = %q must not carry the unreachable-tmux notice: tmux WAS reachable", stderr)
	}
	wantReason := "mode: headless (descended: pane unavailable: no interactive_command; native unavailable)"
	if !strings.Contains(out, wantReason) {
		t.Errorf("output = %q, want the %q selection source", out, wantReason)
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("descended record reads as %q (pane=%q)", rec.Mode(), rec.Pane)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want positive", rec.PID, rec.PGID)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("descended record carries pane identity: window=%q server=%q", rec.Window, rec.Server)
	}
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the headless_command as prefix", rec.SpawnCmd)
	}
	// No dispatch window was opened on the reachable server.
	if names, err := exec.Command("tmux", "-S", privateSocket,
		"list-windows", "-a", "-F", "#W").Output(); err == nil {
		if strings.Contains(string(names), dispatch.WindowName(id, "apply")) {
			t.Errorf("a dispatch window %q was opened despite descent to headless", dispatch.WindowName(id, "apply"))
		}
	}
}

// TestModeCommand_KimiComposesBothModes is the flip side since 260810-ki9v: kimi's
// interactive first run and input echo WERE probed (2026-08-10, kimi 0.34.0 — the
// trust wall is an ordinary readiness-gate judgment round, and its side-bordered
// input box verifies under the gate's box-drawing-tolerant squeeze), so it ships an
// interactive_command and BOTH rungs compose from the shipped defaults.
//
// Asserted at the same seam as the agy case above, because this is where a
// regression would show as a stage silently descending to headless again.
//
// modeCommand returns the provider's TEMPLATE — substitution is the caller's next
// step (dispatchStart's spawn.WithProfile) — so the rung assertions compare against
// the shipped grammar, and the launched form is checked through WithProfile after.
func TestModeCommand_KimiComposesBothModes(t *testing.T) {
	prov, ok := agent.ResolveProvider(nil, "kimi")
	if !ok {
		t.Fatal("built-in kimi provider must resolve with no config")
	}

	for _, tc := range []struct {
		mode dispatch.Mode
		want string
	}{
		{dispatch.ModeHeadless, agent.DefaultKimiHeadlessCommand},
		{dispatch.ModePane, agent.DefaultKimiInteractiveCommand},
	} {
		got, err := modeCommand(tc.mode, prov, "apply", "kimi")
		if err != nil {
			t.Errorf("%s dispatch for the kimi built-in must compose: %v", tc.mode, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s dispatch composes %q, want the shipped %s grammar %q", tc.mode, got, tc.mode, tc.want)
		}
	}

	// The pane rung's LAUNCHED form: kimi ships no fills, so the role resolves an
	// empty model and the token-drop takes `-m` with its placeholder, leaving the
	// fixed --auto. That is the invocation a pane worker actually opens with, and
	// the one the 2026-08-10 probe ran.
	paneCmd, err := modeCommand(dispatch.ModePane, prov, "apply", "kimi")
	if err != nil {
		t.Fatalf("pane dispatch for the kimi built-in must compose: %v", err)
	}
	if got, want := spawn.WithProfile(paneCmd, "", ""), "kimi --auto"; got != want {
		t.Errorf("the launched kimi pane command = %q, want %q (the empty model must drop the -m pair)", got, want)
	}
}

// TestDispatchStart_RefusesAnAutomaticPaneLanding: when the ladder lands on pane,
// `start` stops before any state write and names the verb that owns pane mode.
// Silently launching headless instead would ignore the configured preference, and
// silently opening a pane would produce a worker with no prompt and no gate — so
// the only honest outcome is the route.
//
// This is the pane twin of TestDispatchStart_NativeSelectionRequiresResolveAgent:
// two rungs `start` cannot launch, refused the same way.
func TestDispatchStart_RefusesAnAutomaticPaneLanding(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`) // dispatch.mode: pane
	// A genuinely REACHABLE server is what makes this the pane refusal rather than
	// a descent: an unreachable one is the other test's case, where descending is
	// correct precisely so an unattended `start` keeps working.
	startPrivateTmuxWithPane(t)

	_, stderr, err := runStartCapturingStderr(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("an automatic pane landing must refuse rather than launch")
	}
	for _, want := range []string{"pane mode", "fab dispatch open", "--headless"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name %q", err.Error(), want)
		}
	}
	if strings.Contains(stderr, "dispatch selection:") {
		t.Errorf("a preferred (non-descended) selection prints no notice, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the refusal, got %v", err)
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
	current := string(body)
	if strings.HasPrefix(extra, "dispatch:\n") && strings.Contains(current, "dispatch:\n") {
		merged := "dispatch:\n  mode: pane\n" + strings.TrimPrefix(extra, "dispatch:\n")
		current = strings.Replace(current, "dispatch:\n  mode: pane\n", merged, 1)
		mustWrite(t, path, current)
		return
	}
	mustWrite(t, path, current+extra)
}
