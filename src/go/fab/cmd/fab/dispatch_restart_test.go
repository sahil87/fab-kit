package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// runRestartCapturingStderr executes `fab dispatch restart`, returning stdout,
// stderr, and the error. NOTHING is piped on stdin on purpose: `restart`'s whole
// point is that the prompt comes from the state dir, so a test that fed stdin
// could not distinguish the two sources.
func runRestartCapturingStderr(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := dispatchRestartCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

func runRestart(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout, _, err := runRestartCapturingStderr(t, args...)
	return stdout, err
}

// seedOrphanedHeadless writes the state an ORPHANED headless attempt leaves
// behind: a record naming a dead pid, no exit file, plus a persisted prompt and a
// stale log the restart must clear.
func seedOrphanedHeadless(t *testing.T, dir, stage, prompt string) {
	t.Helper()
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, stage, &dispatch.Dispatch{
		PID: 999999, PGID: 999999, SpawnCmd: "old-command", StartedAt: "old",
	}); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dispatch.PromptPath(dir, stage), prompt)
	mustWrite(t, dispatch.LogPath(dir, stage), "stale log\n")
}

// TestDispatchRestart_RelaunchesFromPersistedPrompt is the core case: an orphaned
// attempt is relaunched with the prompt already on disk as the worker's input, the
// prompt file is left byte-identical (it is the source, not a fresh write), and the
// stale log/exit/result are cleared so the new attempt's status is uncontaminated.
func TestDispatchRestart_RelaunchesFromPersistedPrompt(t *testing.T) {
	// The worker echoes its stdin, proving the persisted prompt really reached it
	// (the wrapper redirects {stage}-prompt.md in, so the log holds the prompt
	// body). TEMPLATED with {model}/{effort} so spawn.WithProfile SUBSTITUTES
	// rather than appending flags a plain `cat` would choke on.
	repoRoot, id := setupDispatchRepo(t, `sh -c 'echo \"m={model} e={effort}\"; cat' _`)
	dir := dispatch.DirFor(repoRoot, id)
	const prompt = "the persisted stage prompt\nsecond line\n"
	seedOrphanedHeadless(t, dir, "apply", prompt)
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stale: true\n")

	out, err := runRestart(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("restart over an orphaned attempt should succeed: %v", err)
	}
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(out, "dispatched abcd/apply") {
		t.Errorf("output = %q, want a dispatched line shaped like start's", out)
	}

	// The prompt file is the restart's INPUT and must not be rewritten.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("prompt file should still exist: %v", err)
	}
	if string(promptData) != prompt {
		t.Errorf("prompt file = %q, want it left byte-identical", string(promptData))
	}

	// The record was overwritten with the new attempt (last-attempt-only), and it
	// carries no restart-specific key — a restart is a fresh attempt.
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.SpawnCmd == "old-command" {
		t.Error("record should have been overwritten by the restart")
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want the new worker's", rec.PID, rec.PGID)
	}
	if _, err := os.Stat(dispatch.ResultPath(dir, "apply")); !os.IsNotExist(err) {
		t.Error("the prior attempt's stale result should have been cleared")
	}

	// The relaunched worker really received the persisted prompt on stdin.
	waitDispatchDone(t, dir, "apply")
	logData, err := os.ReadFile(dispatch.LogPath(dir, "apply"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(logData), "the persisted stage prompt") {
		t.Errorf("worker stdin = %q, want the persisted prompt", string(logData))
	}
	if strings.Contains(string(logData), "stale log") {
		t.Error("the prior attempt's stale log should have been cleared")
	}
}

// TestDispatchRestart_RecordAndOutputMatchStart pins the byte-shape parity claim:
// a restart's dispatched line and record are indistinguishable from the equivalent
// `start` — same identity report, same auto-selection suffix rules, no new marker.
func TestDispatchRestart_RecordAndOutputMatchStart(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'") // clears $TMUX ⇒ auto headless
	dir := dispatch.DirFor(repoRoot, id)

	startOut, err := runStart(t, "prompt for parity\n", "abcd", "apply")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	waitDispatchDone(t, dir, "apply")
	startRec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load after start: %v", err)
	}

	restartOut, err := runRestart(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })
	restartRec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load after restart: %v", err)
	}

	// Both lines carry the same auto-selection source suffix and the same shape;
	// only the pid/pgid numbers differ, so compare after stripping them.
	stripIdentity := func(s string) string {
		open := strings.Index(s, "(")
		mode := strings.Index(s, "mode:")
		// Unexpected shape — no identity parens, or no `mode:` after them.
		// Leave the string whole so the comparison below fails with both
		// outputs printed, rather than slicing on a -1 index.
		if open < 0 || mode < open {
			return s
		}
		return s[:open] + s[mode:]
	}
	if stripIdentity(startOut) != stripIdentity(restartOut) {
		t.Errorf("restart output %q is not shaped like start's %q", restartOut, startOut)
	}
	wantReason := "mode: headless (descended: native unavailable)"
	if !strings.Contains(restartOut, wantReason) {
		t.Errorf("restart output = %q, want the %q selection source", restartOut, wantReason)
	}

	// The record shape is identical: same spawn command, same mode, no pane
	// identity, and no restart-specific key (the struct has none to gain).
	if restartRec.SpawnCmd != startRec.SpawnCmd {
		t.Errorf("spawn_cmd = %q, want start's %q", restartRec.SpawnCmd, startRec.SpawnCmd)
	}
	if restartRec.Mode() != startRec.Mode() {
		t.Errorf("mode = %q, want start's %q", restartRec.Mode(), startRec.Mode())
	}
	if restartRec.Pane != "" || restartRec.Window != "" || restartRec.Server != "" {
		t.Errorf("headless restart record carries pane identity: %+v", *restartRec)
	}
}

// TestDispatchRestart_RefusesWhenRunning: recovery must never race a live worker.
// The refusal is the same one `start` raises (same mode-aware finished signal), and
// the existing record is left untouched.
func TestDispatchRestart_RefusesWhenRunning(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)

	// A live pid (our own process) with no exit file ⇒ genuinely running.
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		PID: os.Getpid(), PGID: os.Getpid(), SpawnCmd: "live", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")

	_, err := runRestart(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected the refuse-if-running error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "already running") || !strings.Contains(msg, "fab dispatch kill") {
		t.Errorf("error = %q, want the already-running refusal pointing at kill", msg)
	}
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.SpawnCmd != "live" {
		t.Errorf("the live record was modified (spawn_cmd = %q)", rec.SpawnCmd)
	}
}

// TestDispatchRestart_RefusalPrecedesTheMissingPrompt pins the ordering claim: the
// refusal check runs BEFORE the prompt is read, so a live dispatch refuses with the
// actionable already-running message even when no prompt file exists.
func TestDispatchRestart_RefusalPrecedesTheMissingPrompt(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		PID: os.Getpid(), PGID: os.Getpid(), SpawnCmd: "live", StartedAt: "t",
	}); err != nil {
		t.Fatal(err)
	}
	// No prompt file on purpose.

	_, err := runRestart(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want the already-running refusal to win over the missing prompt", err.Error())
	}
}

// TestDispatchRestart_MissingPromptFileErrors: with no persisted prompt there is
// nothing to relaunch. The error must name the path and the `fab dispatch start`
// remedy, and nothing may be launched or persisted.
func TestDispatchRestart_MissingPromptFileErrors(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)

	_, err := runRestart(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected an error when no prompt has been persisted")
	}
	msg := err.Error()
	for _, want := range []string{"nothing to relaunch", "fab dispatch start", "apply-prompt.md"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want it to mention %q", msg, want)
		}
	}
	if _, err := dispatch.Load(dir, "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the missing-prompt error, got %v", err)
	}
}

// TestDispatchRestart_MissingPromptOverAnOrphanedRecord is the same error with a
// prior RECORD present but its prompt gone (e.g. a partial `fab dispatch clean`):
// the record must be left alone rather than overwritten by a promptless launch.
func TestDispatchRestart_MissingPromptOverAnOrphanedRecord(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		PID: 999999, PGID: 999999, SpawnCmd: "old-command", StartedAt: "old",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := runRestart(t, "abcd", "apply"); err == nil {
		t.Fatal("expected the missing-prompt error")
	}
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.SpawnCmd != "old-command" {
		t.Errorf("the prior record was overwritten (spawn_cmd = %q)", rec.SpawnCmd)
	}
}

// TestDispatchRestart_NativeSelectionRequiresAgentYAML: the prologue is
// `start`'s, so native selection fails identically with re-resolution guidance.
func TestDispatchRestart_NativeSelectionRequiresAgentYAML(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "") // built-in claude: native capability
	dir := dispatch.DirFor(repoRoot, id)
	seedOrphanedHeadless(t, dir, "apply", "prompt\n")

	_, err := runRestart(t, "abcd", "apply")
	if err == nil {
		t.Fatal("expected native dispatch to be delegated to fab agent YAML")
	}
	for _, want := range []string{"native", "fab agent apply -o yaml"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want %q", err.Error(), want)
		}
	}
}

// TestDispatchRestart_ModeIsReDerivedNotInherited is the mode-re-derivation case
// that matters most for recovery: an ORPHANED PANE attempt restarted with no tmux
// (the very condition that likely killed it) must land HEADLESS. A restart is a
// fresh attempt under the current environment; inheriting the prior mode would
// reproduce the failure.
func TestDispatchRestart_ModeIsReDerivedNotInherited(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)

	// A prior PANE record whose pane is long gone (⇒ orphaned), on a socket with no
	// server, so the record unambiguously describes a dead pane dispatch.
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		SpawnCmd: "old-session-command", StartedAt: "old",
		Pane: "%999", Window: dispatch.WindowName(id, "apply"), Server: "fabtest-gone",
	}); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")
	// $TMUX is already cleared by the setup helper ⇒ auto selects headless.

	out, err := runRestart(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("restarting an orphaned pane dispatch outside tmux should land headless: %v", err)
	}
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.Mode() != dispatch.ModeHeadless {
		t.Errorf("record reads as %q (pane=%q), want a headless re-derivation", rec.Mode(), rec.Pane)
	}
	if rec.PID <= 0 || rec.PGID <= 0 {
		t.Errorf("pid/pgid = %d/%d, want the headless worker's", rec.PID, rec.PGID)
	}
	if rec.Window != "" || rec.Server != "" {
		t.Errorf("the prior pane identity leaked into the new record: %+v", *rec)
	}
	// The headless re-derivation composed headless_command, not the prior
	// attempt's interactive_command — no cross-fallback, no inherited command.
	if !strings.HasPrefix(rec.SpawnCmd, "sh -c 'exit 0'") {
		t.Errorf("spawn_cmd = %q, want the headless_command as prefix", rec.SpawnCmd)
	}
	wantReason := "mode: headless (descended: pane unavailable: no tmux; native unavailable)"
	if !strings.Contains(out, wantReason) {
		t.Errorf("output = %q, want the %q selection source", out, wantReason)
	}
}

// TestDispatchRestart_HeadlessFlagOptsOutOfAutoPane: the explicit rungs of the
// ladder work on `restart` exactly as on `start` — inside tmux, --headless forces a
// detached worker and composes with --timeout.
func TestDispatchRestart_HeadlessFlagOptsOutOfAutoPane(t *testing.T) {
	repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	seedOrphanedHeadless(t, dir, "apply", "prompt\n")
	t.Setenv("TMUX", "/tmp/tmux-1000/default,4242,0") // would auto-select pane

	out, err := runRestart(t, "abcd", "apply", "--headless", "--timeout", "600")
	if err != nil {
		t.Fatalf("--headless --timeout should compose on restart: %v", err)
	}
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
	if strings.Contains(out, "auto:") {
		t.Errorf("output = %q, want no auto-selection suffix for an explicit mode", out)
	}
}

// TestDispatchRestart_PaneAndTimeoutMutuallyExclusive: the same usage error `start`
// raises, for the same reason (--timeout is the headless wrapper's bound, and pane
// mode builds no wrapper). It must fire before anything is launched or persisted.
func TestDispatchRestart_PaneAndTimeoutMutuallyExclusive(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")

	_, err := runRestart(t, "abcd", "apply", "--pane", "--timeout", "600")
	if err == nil {
		t.Fatal("expected a usage error for --pane with --timeout")
	}
	if !strings.Contains(err.Error(), "--pane") || !strings.Contains(err.Error(), "--timeout") {
		t.Errorf("error = %q, want it to name both flags", err.Error())
	}
	if _, err := dispatch.Load(dir, "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the usage error, got %v", err)
	}
}

// TestDispatchRestart_PaneAndHeadlessMutuallyExclusive: the cobra flag group fires
// during ValidateFlagGroups, before any RunE work.
func TestDispatchRestart_PaneAndHeadlessMutuallyExclusive(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'exit 0'")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")

	_, err := runRestart(t, "abcd", "apply", "--pane", "--headless")
	if err == nil {
		t.Fatal("expected a usage error for --pane with --headless")
	}
	for _, want := range []string{"pane", "headless"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to name %q", err.Error(), want)
		}
	}
	if _, err := dispatch.Load(dir, "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the usage error, got %v", err)
	}
}

// TestDispatchRestart_ExplicitPaneHardErrorsOnUnreachableTmux: the explicit/preferred
// asymmetry is `start`'s. A caller who typed --pane requested pane mode, so an
// unreachable server is a hard error with nothing persisted — no silent downgrade.
func TestDispatchRestart_ExplicitPaneHardErrorsOnUnreachableTmux(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "prompt\n")
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-restart-explicit"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	_, stderr, err := runRestartCapturingStderr(t, "abcd", "apply", "--pane")
	if err == nil {
		t.Fatal("explicit --pane must hard-error when tmux is unreachable")
	}
	if strings.Contains(stderr, "dispatch selection:") {
		t.Errorf("explicit --pane must not descend, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dir, "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the hard error, got %v", err)
	}
}

// TestDispatchRestart_PanePreferenceDescendsToHeadless is the other half of the
// asymmetry, and the case the recovery policy leans on: a stale $TMUX (inherited
// from the very server whose death orphaned the worker) must degrade to headless
// with a notice rather than failing an unattended recovery.
func TestDispatchRestart_PanePreferenceDescendsToHeadless(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	dir := dispatch.DirFor(repoRoot, id)
	seedOrphanedHeadless(t, dir, "apply", "prompt\n")
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-restart-stale"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	out, stderr, err := runRestartCapturingStderr(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("pane preference must descend on restart, not error: %v", err)
	}
	t.Cleanup(func() { waitDispatchDone(t, dir, "apply") })

	if !strings.Contains(stderr, "tmux unreachable") {
		t.Errorf("stderr = %q, want tmux-unreachable descent notice", stderr)
	}
	wantReason := "mode: headless (descended: pane unavailable: tmux unreachable; native unavailable)"
	if !strings.Contains(out, wantReason) {
		t.Errorf("output = %q, want the %q selection source", out, wantReason)
	}
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rec.IsPane() || rec.PID <= 0 {
		t.Errorf("descended record must be headless-shaped, got %+v", *rec)
	}
}

// TestDispatchRestart_PaneMode_Integration drives the real `--pane` path on a
// restart against an ephemeral tmux server: the relaunched worker gets the pointer
// to the ALREADY-PERSISTED prompt (never a re-write, never the body through tmux),
// and the record carries the new pane identity. Skipped when tmux is unavailable.
func TestDispatchRestart_PaneMode_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// TEMPLATED with {model}/{effort} (like the built-in claude default) so
	// WithProfile substitutes rather than appending flags AFTER the prompt argument.
	repoRoot, id := setupDispatchRepoWithCommands(t, "",
		`sh -c 'echo \"m={model} e={effort}\"; echo \"got: $1\"; sleep 30' _`)
	dir := dispatch.DirFor(repoRoot, id)
	const prompt = "persisted prompt line one\nline two\n"
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), prompt)
	// A prior pane record whose pane is gone ⇒ orphaned, so the restart proceeds.
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		SpawnCmd: "old", StartedAt: "old", Pane: "%999", Window: dispatch.WindowName(id, "apply"),
	}); err != nil {
		t.Fatal(err)
	}

	server := "fabtest-restart-pane"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	stdout, stderr, err := runRestartCapturingStderr(t, "abcd", "apply", "--pane", "--server", server)
	if err != nil {
		t.Fatalf("pane restart failed: %v", err)
	}
	// A pane landing is only HALF a relaunch — the pane exists, the prompt has not
	// been handed over — so the report says `opened`, and the missing half is named
	// on stderr. `restart`'s caller asked for a full relaunch, so it is told; the
	// `open` verb's caller is not, because its own name already says so.
	if !strings.Contains(stdout, "opened abcd/apply") || !strings.Contains(stdout, "pane ") {
		t.Errorf("stdout = %q, want an opened line naming the pane", stdout)
	}
	if strings.Contains(stdout, "dispatched") {
		t.Errorf("stdout = %q, must not claim the stage was dispatched before delivery", stdout)
	}
	for _, want := range []string{"fab dispatch ready", "fab dispatch deliver"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr = %q, want the hand-back note naming %q", stderr, want)
		}
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if !rec.IsPane() || rec.Pane == "%999" {
		t.Fatalf("record should name the NEW pane, got %+v", *rec)
	}
	if rec.Delivered {
		t.Error("a restarted pane must record as NOT delivered — the gate and deliver still have to run")
	}

	// The persisted prompt was reused untouched, and NOTHING rode tmux: the pane
	// holds neither the prompt body nor a pointer until `deliver` types one.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	if string(promptData) != prompt {
		t.Errorf("prompt file = %q, want it left byte-identical", string(promptData))
	}
	captured, err := tmux("capture-pane", "-p", "-t", rec.Pane)
	if err != nil {
		t.Fatalf("capture pane: %v", err)
	}
	if strings.Contains(captured, ".fab-dispatch/") || strings.Contains(captured, "line two") {
		t.Errorf("pane received %q, want nothing — a restarted pane is delivered to separately", captured)
	}
}

// TestDispatchRestart_SplitsTheDispatchersWindow_Integration pins that `restart`
// inherits the pane SHAPE decision from the shared launch tail with no
// restart-specific branch: issued from inside a tmux pane, it splits THAT pane's
// window exactly as `start` does — including when the prior attempt's record names
// a WINDOW-shaped dispatch, since the shape is re-derived from the current
// environment rather than inherited from the record.
func TestDispatchRestart_SplitsTheDispatchersWindow_Integration(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "persisted prompt\n")
	// An ORPHANED prior attempt whose pane is gone, so the restart proceeds.
	if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
		SpawnCmd: "old", StartedAt: "old", Pane: "%999", Window: dispatch.WindowName(id, "apply"),
	}); err != nil {
		t.Fatal(err)
	}

	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)

	out, err := runRestart(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("restart from inside a tmux pane failed: %v", err)
	}
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "split") || !strings.Contains(out, "title "+title) {
		t.Errorf("output = %q, want the split report naming the pane title %q", out, title)
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if rec.Pane == "%999" {
		t.Fatalf("record should name the NEW pane, got %+v", *rec)
	}
	if got, want := paneWindow(t, tmuxScoped, rec.Pane), paneWindow(t, tmuxScoped, dispatcherPane); got != want {
		t.Errorf("restarted worker pane %s is in window %s, want the dispatcher's window %s", rec.Pane, got, want)
	}
	if got := paneTitle(t, tmuxScoped, rec.Pane); got != title {
		t.Errorf("pane title = %q, want %q", got, title)
	}
}

// TestDispatchRestart_StacksInTheWorkerColumn_Integration pins that `restart`
// inherits the record-keyed COLUMN PLACEMENT too, not just the shape: with another
// stage's worker already live in the dispatcher's window, a restarted worker stacks
// UNDER that sibling rather than carving a second column out of the dispatcher. The
// sibling's pane title is clobbered first, since a restart is exactly the moment a
// long-running worker's harness has already rewritten it.
func TestDispatchRestart_StacksInTheWorkerColumn_Integration(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	dir := dispatch.DirFor(repoRoot, id)
	mustMkdir(t, dir)
	mustWrite(t, dispatch.PromptPath(dir, "review-pr"), "persisted prompt\n")
	// An ORPHANED prior attempt for the stage being restarted, so the restart proceeds.
	if err := dispatch.Save(dir, "review-pr", &dispatch.Dispatch{
		SpawnCmd: "old", StartedAt: "old", Pane: "%999", Window: dispatch.WindowName(id, "review-pr"),
	}); err != nil {
		t.Fatal(err)
	}

	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)

	// The live sibling: a genuine `open` for the OTHER `doing`-role stage, which
	// carves the column. (Only the doing role points at the fixture's `cli` provider,
	// so a stage outside it would launch the real claude CLI.)
	if _, err := runOpen(t, "apply prompt", "abcd", "apply"); err != nil {
		t.Fatalf("sibling dispatch failed: %v", err)
	}
	sibling, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load apply: %v", err)
	}
	if out, err := tmuxScoped("select-pane", "-t", sibling.Pane, "-T", "✳ some harness title"); err != nil {
		t.Fatalf("could not clobber the sibling's pane title: %v (%q)", err, out)
	}

	if _, err := runRestart(t, "abcd", "review-pr"); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	rec, err := dispatch.Load(dir, "review-pr")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}

	// Stacked: same left edge as the sibling (it split the SIBLING), distinct top
	// edge, and the dispatcher's own column untouched.
	if got, want := paneFormat(t, tmuxScoped, rec.Pane, "#{pane_left}"),
		paneFormat(t, tmuxScoped, sibling.Pane, "#{pane_left}"); got != want {
		t.Errorf("restarted worker's left edge = %s, want the sibling's %s (it must stack the column, not carve a second one)", got, want)
	}
	if paneFormat(t, tmuxScoped, rec.Pane, "#{pane_top}") == paneFormat(t, tmuxScoped, sibling.Pane, "#{pane_top}") {
		t.Error("restarted worker shares the sibling's top edge; the stacking split must be vertical (-v)")
	}
	if got, want := paneFormat(t, tmuxScoped, dispatcherPane, "#{pane_left}"), "0"; got != want {
		t.Errorf("dispatcher's left edge = %s, want %s — the left/right separator must not move", got, want)
	}
	if n := len(strings.Split(mustTmux(t, tmuxScoped, "list-panes", "-t", dispatcherPane, "-F", "#{pane_id}"), "\n")); n != 3 {
		t.Errorf("window holds %d panes, want 3 (dispatcher + sibling + restarted worker)", n)
	}
}

// mustTmux runs a scoped tmux command, failing the test on error.
func mustTmux(t *testing.T, tmuxScoped func(...string) (string, error), args ...string) string {
	t.Helper()
	out, err := tmuxScoped(args...)
	if err != nil {
		t.Fatalf("tmux %v: %v (%q)", args, err, out)
	}
	return out
}

// TestDispatchRestart_PaneRefuseHonorsTheResultFile: a finished-but-still-alive
// pane worker reads `done`, so a restart over it must OVERWRITE rather than refuse
// — the same finished-signal rule `start` applies, so the two never disagree.
func TestDispatchRestart_PaneRefuseHonorsTheResultFile(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	dir := dispatch.DirFor(repoRoot, id)

	server := "fabtest-restart-pdone"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux := func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	if out, err := tmux("new-session", "-d", "-s", "s", "-x", "80", "-y", "24"); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })

	if _, err := runOpen(t, "first prompt", "abcd", "apply", "--server", server); err != nil {
		t.Fatalf("seed pane open failed: %v", err)
	}
	first, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !pane.PaneAlive(first.Pane, server) {
		t.Fatalf("pane %s should be alive; the test needs a live pane to be meaningful", first.Pane)
	}

	// Alive with NO result ⇒ genuinely running, so a restart must refuse.
	if _, err := runRestart(t, "abcd", "apply", "--pane", "--server", server); err == nil {
		t.Fatal("expected refusal while the pane is alive with no result file")
	} else if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want the already-running refusal", err.Error())
	}

	// The worker writes its result and sits at its prompt ⇒ `done`, so a restart
	// over it now overwrites.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	if _, err := runRestart(t, "abcd", "apply", "--pane", "--server", server); err != nil {
		t.Fatalf("restart over a completed pane attempt should succeed, got: %v", err)
	}
	second, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load after overwrite: %v", err)
	}
	if second.Pane == first.Pane {
		t.Errorf("record still names the old pane %s; the attempt was not overwritten", first.Pane)
	}
	if _, err := os.Stat(dispatch.ResultPath(dir, "apply")); !os.IsNotExist(err) {
		t.Error("stale result file should have been cleared by the restart")
	}
	// The prompt `start` persisted is what the restart reused.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	if string(promptData) != "first prompt" {
		t.Errorf("prompt = %q, want the original start's prompt reused unchanged", string(promptData))
	}
}
