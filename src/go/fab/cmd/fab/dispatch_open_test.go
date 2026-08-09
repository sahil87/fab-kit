package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// This file covers `fab dispatch open` — pane mode's entry — and with it the
// whole pane half of the shared launch path in dispatch_start.go: prerequisites,
// the two placement shapes, worker-column stacking, and refuse-if-running.
//
// These cases used to drive `fab dispatch start --pane`. They moved here with the
// verb, and the central assertion CHANGED: the spawned pane must receive NO
// prompt. Delivery is now a separate, verified step (`fab dispatch deliver`,
// covered in dispatch_deliver_test.go), which is what decouples pane capability
// from whether a provider's CLI accepts a positional prompt.

// runOpenCapturingStderr executes `fab dispatch open` with a prompt piped on
// stdin, returning stdout, stderr, and the error.
func runOpenCapturingStderr(t *testing.T, prompt string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := dispatchOpenCmd()
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

// runOpen is runOpenCapturingStderr for the tests that assert only stdout.
func runOpen(t *testing.T, prompt string, args ...string) (string, error) {
	t.Helper()
	stdout, _, err := runOpenCapturingStderr(t, prompt, args...)
	return stdout, err
}

// TestDispatchOpen_WithoutTmuxServerErrors: pane mode requires a reachable
// tmux server and must leave no partial dispatch behind. Targeting a --server socket
// that has no running server is the deterministic way to force the failure
// regardless of whether the test host has tmux running.
func TestDispatchOpen_WithoutTmuxServerErrors(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "sh -c 'exit 0'")
	// A private, empty TMUX_TMPDIR guarantees the named socket has no server.
	server := "fabtest-unreachable"
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))

	_, err := runOpen(t, "prompt", "abcd", "apply", "--server", server)
	if err == nil {
		t.Fatal("expected a hard error when no tmux server is reachable")
	}
	msg := err.Error()
	// "pane mode", not a flag name: `open` IS the pane verb, so the message must
	// not quote a flag the caller never passed. "--headless" is the actionable
	// remedy the guidance owes.
	for _, want := range []string{"pane mode", "tmux", "--headless"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error = %q, want it to mention %q", msg, want)
		}
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after an unreachable-server error, got %v", err)
	}
}

// TestDispatchOpen_Integration drives the real pane path against an ephemeral
// tmux server: the window is created with the dispatch-window name (and no
// operator marker), the FULL prompt lands in {stage}-prompt.md, and the record
// persists the pane identity without pid/pgid.
//
// THE CENTRAL ASSERTION: the pane receives NOTHING. The composed
// interactive_command reaches tmux verbatim, so a provider CLI's positional-prompt
// grammar is irrelevant to whether it can host a pane worker — and the record
// reads as UNDELIVERED, which is what lets `deliver` tell an empty pane from a
// mid-stage one. Skipped when tmux is unavailable.
func TestDispatchOpen_Integration(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	// A stand-in interactive_command that behaves like a real agent CLI: TEMPLATED
	// with {model}/{effort} (as the built-in claude default is, so WithProfile
	// substitutes rather than appending flags), echoing any trailing argument — so
	// the test can prove NO argument was appended — then staying alive like an
	// interactive worker sitting at its prompt. `_` fills sh -c's $0 slot so a
	// positional would land in $1.
	repoRoot, id := setupDispatchRepoWithCommands(t, "",
		`sh -c 'echo \"model={model} effort={effort}\"; echo \"got:[$1]\"; sleep 30' _`)

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
	out, err := runOpen(t, fullPrompt, "abcd", "apply", "--server", server)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	// "opened", not "dispatched": the pane exists but holds no prompt yet.
	if !strings.Contains(out, "opened abcd/apply") || !strings.Contains(out, "pane ") {
		t.Errorf("output = %q, want an opened line naming the pane", out)
	}
	if strings.Contains(out, "dispatched") {
		t.Errorf("output = %q, must not claim the stage was dispatched before delivery", out)
	}

	dir := dispatch.DirFor(repoRoot, id)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("state not persisted: %v", err)
	}
	if !rec.IsPane() || rec.Mode() != dispatch.ModePane {
		t.Fatalf("record reads as %q, want pane mode (%+v)", rec.Mode(), *rec)
	}
	if rec.Delivered {
		t.Error("an opened pane must record as NOT delivered — delivery is a separate step")
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

	// The FULL prompt is on disk, ready for `deliver` to point the worker at.
	promptData, err := os.ReadFile(dispatch.PromptPath(dir, "apply"))
	if err != nil {
		t.Fatalf("prompt not persisted: %v", err)
	}
	if string(promptData) != fullPrompt {
		t.Errorf("prompt file = %q, want the full prompt", string(promptData))
	}
	// And the pane got NEITHER the prompt body NOR a pointer: the launch grammar
	// is composed verbatim, so the stand-in's $1 is empty.
	captured, err := tmux("capture-pane", "-p", "-t", rec.Pane)
	if err != nil {
		t.Fatalf("capture pane: %v", err)
	}
	if !strings.Contains(captured, "got:[]") {
		t.Errorf("pane captured %q, want an EMPTY positional — open must append nothing to interactive_command", captured)
	}
	if strings.Contains(captured, ".fab-dispatch/") || strings.Contains(captured, "line two") {
		t.Errorf("pane received prompt content (%q); open delivers nothing", captured)
	}
	// The pane is observably alive (the liveness signal status keys on).
	if !dispatch.PaneAlive(rec.Pane, server) {
		t.Errorf("pane %s should be alive right after open", rec.Pane)
	}
}

// TestDispatchOpen_RefuseIfRunningHonorsTheResultFile pins that
// refuse-if-running reads the SAME finished-signal `fab dispatch status` derives
// pane state from: result presence wins over pane liveness. An interactive worker
// never exits on completion — it sits at its prompt — so a liveness-only refusal
// would report `done` from `status` while `open` refused forever, permanently
// stranding a completed attempt that the overwrite contract says is replaceable.
// Skipped when tmux is unavailable (a genuinely live pane is the whole point:
// with a dead pane the old rule would pass too).
func TestDispatchOpen_RefuseIfRunningHonorsTheResultFile(t *testing.T) {
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
	if _, err := runOpen(t, "first prompt", "abcd", "apply", "--server", server); err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	first, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !dispatch.PaneAlive(first.Pane, server) {
		t.Fatalf("pane %s should be alive; the test needs a live pane to be meaningful", first.Pane)
	}

	// While that pane is still ALIVE and carries NO result: genuinely running, so
	// a second open must refuse.
	if _, err := runOpen(t, "second prompt", "abcd", "apply", "--server", server); err == nil {
		t.Fatal("expected refusal while the pane is alive with no result file")
	} else if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want the already-running refusal", err.Error())
	}

	// The worker finishes: it writes its result and sits at its prompt (pane still
	// alive). status derives `done`, so open must now OVERWRITE rather than refuse.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	if !dispatch.PaneAlive(first.Pane, server) {
		t.Fatalf("pane %s died; the finished-but-alive case is what this test covers", first.Pane)
	}

	if _, err := runOpen(t, "third prompt", "abcd", "apply", "--server", server); err != nil {
		t.Fatalf("open over a completed pane attempt should succeed, got: %v", err)
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

// TestDispatchOpen_WithoutInteractiveCommandPersistsNothing: `open` selects pane
// EXPLICITLY, so a provider with no interactive_command is a hard error with
// nothing launched and nothing persisted — never a silent descent to headless,
// which is the opposite of what this verb's caller asked for. tmux is deliberately
// REACHABLE here (an ephemeral server), so the missing interactive_command is the
// sole failure cause and the error cannot be the probe's.
//
// It is also the pane half of the no-cross-substitution rule: pane mode composes
// interactive_command and must never substitute headless_command, so the error
// names the stage's resolved role and the exact config key. Its headless mirror is
// TestDispatchStart_HeadlessStillRequiresHeadlessCommand.
func TestDispatchOpen_WithoutInteractiveCommandPersistsNothing(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "") // no interactive_command

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

	_, stderr, err := runOpenCapturingStderr(t, "prompt", "abcd", "apply", "--server", server)
	if err == nil {
		t.Fatal("open must hard-error when the provider has no interactive_command")
	}
	if !strings.Contains(err.Error(), "providers.cli.interactive_command") {
		t.Errorf("error = %q, want the interactive_command config-key hint", err.Error())
	}
	if !strings.Contains(err.Error(), "doing") {
		t.Errorf("error = %q, want mention of the resolved role", err.Error())
	}
	if strings.Contains(stderr, "dispatch selection:") {
		t.Errorf("open must not print a descent notice, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the hard error, got %v", err)
	}
}

// TestDispatchOpen_StillHardErrorsOnUnreachableTmux is the other half of the
// explicit-selection rule: a caller who ran `open` requested pane mode, so a
// silent downgrade would defeat the request. Nothing is launched and nothing
// persisted.
func TestDispatchOpen_StillHardErrorsOnUnreachableTmux(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "sh -c 'exit 0'", "sh -c 'sleep 30' _")
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-explicit"))
	t.Setenv("TMUX", "/tmp/tmux-dead/default,9999,0")

	_, stderr, err := runOpenCapturingStderr(t, "prompt", "abcd", "apply")
	if err == nil {
		t.Fatal("open must hard-error when tmux is unreachable")
	}
	if strings.Contains(stderr, "dispatch selection:") {
		t.Errorf("open must not print a descent notice, stderr = %q", stderr)
	}
	if _, err := dispatch.Load(dispatch.DirFor(repoRoot, id), "apply"); !os.IsNotExist(err) {
		t.Errorf("no dispatch record should exist after the hard error, got %v", err)
	}
}

// TestDispatchOpen_SplitPane_Integration is the two-tier hierarchy's core case: a
// dispatcher that IS a tmux pane gets its stage worker as a PANE SPLIT INTO ITS
// OWN WINDOW, not as a separate window. The worker's identity rides the pane TITLE
// (a split pane has no window name of its own), the pane ID is what the record
// stores, and the dispatcher's window count is unchanged — so a worker never costs
// a window in the operator's tab bar.
func TestDispatchOpen_SplitPane_Integration(t *testing.T) {
	// TEMPLATED with {model}/{effort} (as the built-in claude interactive_command is), so
	// spawn.WithProfile SUBSTITUTES rather than appending flags.
	repoRoot, id := setupDispatchRepoWithCommands(t, "",
		`sh -c 'echo \"m={model} e={effort}\"; echo \"got:[$1]\"; sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)

	windowsBefore, err := tmuxScoped("list-windows", "-a", "-F", "#{window_id}")
	if err != nil {
		t.Fatalf("list windows: %v", err)
	}

	out, err := runOpen(t, "the stage prompt\n", "abcd", "apply")
	if err != nil {
		t.Fatalf("split open failed: %v", err)
	}
	// The split shape's own report: "split, title …" rather than "window …".
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "split") || !strings.Contains(out, "title "+title) {
		t.Errorf("output = %q, want the split report naming the pane title %q", out, title)
	}
	// `open` selects pane EXPLICITLY, so there is no selection-reason suffix — the
	// verb itself is the explanation.
	if strings.Contains(out, "mode: ") {
		t.Errorf("output = %q, want no selection-reason suffix on an explicit verb", out)
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

	// The pane is alive and — as in the window shape — received NOTHING.
	if !dispatch.PaneAlive(rec.Pane, "") {
		t.Errorf("pane %s should be alive right after open", rec.Pane)
	}
	captured, err := tmuxScoped("capture-pane", "-p", "-t", rec.Pane)
	if err != nil {
		t.Fatalf("capture pane: %v", err)
	}
	if strings.Contains(captured, ".fab-dispatch/") {
		t.Errorf("pane received %q, want no prompt pointer — delivery is `fab dispatch deliver`", captured)
	}
}

// TestDispatchOpen_SplitPanesStackInTheRightColumn is the placement rule: with a
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
func TestDispatchOpen_SplitPanesStackInTheRightColumn(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)
	dir := dispatch.DirFor(repoRoot, id)
	dispatcherWindow := paneWindow(t, tmuxScoped, dispatcherPane)

	if _, err := runOpen(t, "apply prompt", "abcd", "apply"); err != nil {
		t.Fatalf("first split open failed: %v", err)
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
	// overwrite of the first. It must be the OTHER `doing`-role stage (review-pr):
	// the fixture points only the doing role at the `cli` provider, so a stage
	// outside it (e.g. review → the review role) resolves the BUILT-IN claude
	// provider and launches the real `claude` CLI — alive on a dev box where
	// claude exists (a false local pass, plus a stray real agent session in the
	// test server), dead instantly on CI where it doesn't.
	if _, err := runOpen(t, "review-pr prompt", "abcd", "review-pr"); err != nil {
		t.Fatalf("second split open failed: %v", err)
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

// TestDispatchOpen_UnreadableRecordWarnsAndStillLaunches is the record-read half of
// the degradation contract (the tmux-probe half is covered by the sized-split retry
// tests): a corrupt {stage}.yaml in the checkout's dispatch tree must reach
// launchPane's stderr warning AND still launch the worker. Before this, the record
// walk swallowed every read failure, so a broken tree silently produced blind
// placement with nothing in the output to explain it.
//
// A live recorded sibling sits alongside the corrupt record, so the test also pins
// the partial-set behavior: the readable record still wins the intersection and the
// worker STACKS — a read failure degrades the probe, it does not discard it.
func TestDispatchOpen_UnreadableRecordWarnsAndStillLaunches(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, dispatcherPane := startPrivateTmuxWithPane(t)
	dir := dispatch.DirFor(repoRoot, id)

	if _, err := runOpen(t, "apply prompt", "abcd", "apply"); err != nil {
		t.Fatalf("first split open failed: %v", err)
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

	_, stderr, err := runOpenCapturingStderr(t, "review-pr prompt", "abcd", "review-pr")
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

// TestDispatchOpen_CarvingSplitSizesTheWorkerColumn is the sizing rule: the
// column-CARVING split runs `-l <n>%`, so the dispatching agent — the pane the user
// is actually watching — keeps the rest of the window instead of being halved. The
// width comes from `dispatch.column_width` (default 35), and the STACKING split that
// follows leaves the left/right separator untouched: the column invariant.
func TestDispatchOpen_CarvingSplitSizesTheWorkerColumn(t *testing.T) {
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

			if _, err := runOpen(t, "apply prompt", "abcd", "apply"); err != nil {
				t.Fatalf("split open failed: %v", err)
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
			if _, err := runOpen(t, "review-pr prompt", "abcd", "review-pr"); err != nil {
				t.Fatalf("second split open failed: %v", err)
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

// TestDispatchOpen_ServerFlagKeepsTheNewWindowShape: `--server <name>` targets a
// possibly DIFFERENT tmux socket, on which the caller's own $TMUX_PANE id is
// meaningless — so naming one keeps the new-window shape even when the caller is
// itself sitting in a pane. $TMUX_PANE is deliberately set to a pane id from
// ANOTHER (private, `-L`-labelled) server here, which is exactly the cross-socket
// confusion the rule prevents.
func TestDispatchOpen_ServerFlagKeepsTheNewWindowShape(t *testing.T) {
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

	out, err := runOpen(t, "prompt", "abcd", "apply", "--server", server)
	if err != nil {
		t.Fatalf("--server open failed: %v", err)
	}
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "window "+title) || strings.Contains(out, "split") {
		t.Errorf("output = %q, want the new-window report %q", out, "window "+title)
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

// TestDispatchOpen_NoTmuxPaneKeepsTheNewWindowShape: a headless orchestrator
// running `open` has no pane of its own to split, so the new-window shape applies.
// TestDispatchOpen_Integration already covers that rung via --server; this test
// pins the OTHER window rung (no $TMUX_PANE) on the DEFAULT socket, where a stray
// $TMUX_PANE would otherwise be honored.
func TestDispatchOpen_NoTmuxPaneKeepsTheNewWindowShape(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", `sh -c 'sleep 30' _`)
	tmuxScoped, _ := startPrivateTmuxWithPane(t)
	// The orchestrator is not a pane: unset the anchor the helper just set.
	t.Setenv("TMUX_PANE", "")

	out, err := runOpen(t, "prompt", "abcd", "apply")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	title := dispatch.WindowName(id, "apply")
	if !strings.Contains(out, "window "+title) || strings.Contains(out, "split") {
		t.Errorf("output = %q, want the new-window report %q", out, "window "+title)
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
