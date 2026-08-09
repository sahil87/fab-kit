package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// This file covers `fab dispatch deliver` and `fab dispatch ready` at the COMMAND
// layer: the guards that decide whether a delivery may happen at all, and the
// end-to-end delivery into a real tmux pane. The send-keys choreography itself —
// echo verification, the single retry, the busy confirmation — is unit-tested
// against a scripted fake in internal/dispatch/gate_test.go, which is where the
// retry path can be forced deterministically.

// runDeliver executes `fab dispatch deliver`, returning stdout, stderr, and error.
func runDeliver(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := dispatchDeliverCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

// newTmuxPane starts a PRIVATE tmux server (its socket dir isolated per test),
// opens one session whose pane runs command — empty meaning the default shell,
// which is enough of an "agent" for the choreography's terms to mean something:
// it echoes typed text and reacts to Enter — and returns a runner bound to that
// server plus the pane id. tmux absence is a skip, never a failure.
func newTmuxPane(t *testing.T, server, command string, width int) (tmux func(args ...string) (string, error), paneID string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, server))
	tmux = func(args ...string) (string, error) {
		out, err := exec.Command("tmux", append([]string{"-L", server}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	newSession := []string{"new-session", "-d", "-s", "s", "-x", strconv.Itoa(width), "-y", "24"}
	if command != "" {
		newSession = append(newSession, command)
	}
	if out, err := tmux(newSession...); err != nil {
		t.Skipf("could not start tmux server (%v): %s", err, out)
	}
	t.Cleanup(func() { _, _ = tmux("kill-server") })
	paneID, err := tmux("display-message", "-p", "-t", "s", "#{pane_id}")
	if err != nil || paneID == "" {
		t.Fatalf("resolve pane id: %v (%q)", err, paneID)
	}
	return tmux, paneID
}

// runReady executes `fab dispatch ready`, returning stdout and error.
func runReady(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := dispatchReadyCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// TestDispatchDeliver_RefusesNonPaneDispatches pins the two shared preconditions of
// `ready` and `deliver`: both verbs only make sense against a LIVE PANE worker, and
// each refusal names what the caller should reach for instead.
func TestDispatchDeliver_RefusesNonPaneDispatches(t *testing.T) {
	t.Run("headless record", func(t *testing.T) {
		repoRoot, id := setupDispatchRepo(t, "sh -c 'exit 0'")
		dir := dispatch.DirFor(repoRoot, id)
		mustMkdir(t, dir)
		if err := dispatch.Save(dir, "apply", &dispatch.Dispatch{
			PID: 4242, PGID: 4242, SpawnCmd: "x", StartedAt: "t",
		}); err != nil {
			t.Fatal(err)
		}

		for verb, run := range map[string]func() error{
			"deliver": func() error { _, _, err := runDeliver(t, "abcd", "apply"); return err },
			"ready":   func() error { _, err := runReady(t, "abcd", "apply"); return err },
		} {
			err := run()
			if err == nil {
				t.Fatalf("%s must refuse a headless dispatch", verb)
			}
			if !strings.Contains(err.Error(), "headless") {
				t.Errorf("%s error = %q, want it to name the headless record", verb, err)
			}
		}
	})

	t.Run("dead pane", func(t *testing.T) {
		repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
		// An unreachable socket makes the recorded pane unobservable, i.e. dead.
		t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "fabtest-deliver-dead"))
		seedPaneDispatch(t, repoRoot, id, "apply", "%99", "fabtest-deliver-dead")

		_, _, err := runDeliver(t, "abcd", "apply")
		if err == nil {
			t.Fatal("deliver must refuse when the pane is gone")
		}
		if !strings.Contains(err.Error(), "fab dispatch restart") {
			t.Errorf("error = %q, want it to point at restart (the verb that opens a fresh worker)", err)
		}
	})

	t.Run("no dispatch at all", func(t *testing.T) {
		setupDispatchRepo(t, "sh -c 'exit 0'")
		_, _, err := runDeliver(t, "abcd", "apply")
		if err == nil {
			t.Fatal("deliver must error when no dispatch record exists")
		}
		if !strings.Contains(err.Error(), "no dispatch for") {
			t.Errorf("error = %q, want the family's no-dispatch message", err)
		}
	})
}

// TestDispatchDeliver_RefusesAMidStageWorker is the no-input-injection rule in
// code: between `open` and a successful `deliver` the pane holds no stage context
// and may be typed into, but a DELIVERED worker executing its stage never may. The
// pane must be genuinely alive for the refusal to be about the worker's state
// rather than the pane's absence.
func TestDispatchDeliver_RefusesAMidStageWorker(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-midstage"
	tmux, paneID := newTmuxPane(t, server, "", 80)

	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatal(err)
	}
	rec.Delivered = true
	if err := dispatch.Save(dir, "apply", rec); err != nil {
		t.Fatal(err)
	}

	// Delivered, live pane, NO result ⇒ the worker is executing its stage.
	before, err := tmux("capture-pane", "-p", "-t", paneID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runDeliver(t, "abcd", "apply")
	if err == nil {
		t.Fatal("deliver must refuse a worker that is mid-stage")
	}
	if !strings.Contains(err.Error(), "still running") {
		t.Errorf("error = %q, want it to name the running worker", err)
	}
	// And nothing was typed: the refusal precedes every send.
	if after, err := tmux("capture-pane", "-p", "-t", paneID); err == nil && after != before {
		t.Errorf("the pane changed (%q → %q); a refused delivery must send nothing", before, after)
	}

	// The sanctioned continuation case: the worker finished and sits at its prompt
	// (result present ⇒ `done`), so a continuation prompt MAY be delivered. Only the
	// guard is under test here, so a missing prompt file is enough to prove it was
	// passed — the error is about the file, not about the worker's state.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")
	_, _, err = runDeliver(t, "abcd", "apply", "--prompt-file", "nope.md")
	if err == nil || strings.Contains(err.Error(), "still running") {
		t.Errorf("a `done` worker must pass the mid-stage guard, got %v", err)
	}
}

// TestDispatchDeliver_MissingPromptFileErrorsBeforeTyping: a pointer at a file that
// is not there would type cleanly, verify cleanly, and leave the worker reading
// nothing — the silent failure the whole choreography exists to prevent. Both the
// stage's own prompt and a --prompt-file continuation are checked.
func TestDispatchDeliver_MissingPromptFileErrorsBeforeTyping(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-noprompt"
	_, paneID := newTmuxPane(t, server, "", 80)

	seedPaneDispatch(t, repoRoot, id, "apply", paneID, server) // no {stage}-prompt.md

	for _, args := range [][]string{
		{"abcd", "apply"},
		{"abcd", "apply", "--prompt-file", "does/not/exist.md"},
	} {
		_, _, err := runDeliver(t, args...)
		if err == nil {
			t.Fatalf("%v: deliver must refuse when the prompt file is absent", args)
		}
		if !strings.Contains(err.Error(), "nothing to deliver") {
			t.Errorf("%v: error = %q, want the nothing-to-deliver message", args, err)
		}
	}
}

// TestDeliveryPointerPath_PromptFileIsRepoRelative pins the anchoring of a
// caller-supplied --prompt-file. The flag is documented and exampled as
// repo-relative (`.fab-dispatch/{id}/apply-continuation.md`), but a raw os.Stat
// reads it against the CALLER's cwd — and `fab` runs from anywhere inside the
// repo, because resolve.FabRoot walks upward. Delivered from a subdirectory, that
// mismatch failed the existence check on a file that was right there, so a
// rework-cycle continuation was unreachable outside the repo root.
//
// It is exercised below the command layer because the guards above it need a live
// tmux pane; the anchoring is what is under test, not the choreography.
func TestDeliveryPointerPath_PromptFileIsRepoRelative(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	dir := dispatch.DirFor(repoRoot, id)
	rel := filepath.Join(dispatch.DirName, id, "apply-continuation.md")
	mustWrite(t, filepath.Join(repoRoot, rel), "rework: fix the thing\n")

	// The cwd the raw Stat got wrong. setupDispatchRepo's cleanup restores it.
	sub := filepath.Join(repoRoot, "src", "go")
	mustMkdir(t, sub)
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	want := filepath.ToSlash(rel)
	for _, tc := range []struct{ name, arg string }{
		{"repo-relative", rel},
		{"absolute", filepath.Join(repoRoot, rel)},
	} {
		got, err := deliveryPointerPath(dir, "apply", tc.arg)
		if err != nil {
			t.Fatalf("%s --prompt-file must resolve from a subdirectory: %v", tc.name, err)
		}
		// Both spellings name one file, so both must type the same short pointer —
		// the pane's cwd is the repo root, which is what makes it readable there.
		if got != want {
			t.Errorf("%s: pointer = %q, want the repo-relative %q", tc.name, got, want)
		}
	}

	// The anchoring must not invent a file: a relative path with nothing behind it
	// still fails the existence check, which is the silent-failure guard itself.
	if _, err := deliveryPointerPath(dir, "apply", "does/not/exist.md"); err == nil {
		t.Error("an anchored --prompt-file that is absent must still be refused")
	}
}

// TestDispatchDeliver_PartialStashIsRestored covers the failure the stash exists
// for turning on the stash itself: the result file has already been removed when
// the NEXT signal fails to stash, so a stash that dropped what it had got through
// would lose the previous cycle's result for good and leave the dispatch at
// `delivered: true` with no result — derived state `running`, which every
// recovery verb refuses.
//
// An unreadable `{stage}.exit` (here a directory, the cheapest portable way to
// make os.ReadFile fail on a path that stats fine) forces the partial failure.
func TestDispatchDeliver_PartialStashIsRestored(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-deliver-stash"
	_, paneID := newTmuxPane(t, server, "", 80)

	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "the full stage prompt\n")
	const priorResult = "stage: apply\nstatus: success\nsummary: cycle 1\n"
	mustWrite(t, dispatch.ResultPath(dir, "apply"), priorResult)
	if err := os.Mkdir(dispatch.ExitPath(dir, "apply"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runDeliver(t, "abcd", "apply"); err == nil {
		t.Fatal("a stash that cannot read a completion signal must fail the delivery")
	}
	got, err := os.ReadFile(dispatch.ResultPath(dir, "apply"))
	if err != nil {
		t.Fatalf("the already-stashed result must be put back on a partial failure: %v", err)
	}
	if string(got) != priorResult {
		t.Errorf("restored result = %q, want the untouched %q", got, priorResult)
	}
}

// TestDispatchDeliver_Integration is the end-to-end path against a real tmux pane
// running a plain shell — enough of an "agent" for the choreography's terms to
// mean something: it echoes typed text (the echo check) and reacts to Enter (the
// busy check).
//
// It pins the three observable outcomes of a successful delivery: the readiness
// probe reports `ready`, the pointer reaches the pane, and the record flips to
// delivered. Together they are what an orchestrator reads to know a worker
// actually has its prompt — the thing the previous spawn-time argument could not
// tell anyone.
func TestDispatchDeliver_Integration(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-deliver"
	tmux, paneID := newTmuxPane(t, server, "", 200)

	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)
	mustWrite(t, dispatch.PromptPath(dir, "apply"), "the full stage prompt\nline two\n")
	// A stale result from a previous cycle: delivery must clear it, or the very
	// next `wait` would return `done` on the OLD result instead of the work just
	// started.
	mustWrite(t, dispatch.ResultPath(dir, "apply"), "stage: apply\nstatus: success\n")

	// The gate first: a live shell echoes the sentinel, so the pane reads ready and
	// the report is the bare word with no snippet.
	readyOut, err := runReady(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if readyOut != string(dispatch.ReadyReady)+"\n" {
		t.Errorf("ready output = %q, want exactly %q", readyOut, string(dispatch.ReadyReady)+"\n")
	}

	stdout, _, err := runDeliver(t, "abcd", "apply")
	if err != nil {
		t.Fatalf("deliver: %v", err)
	}
	// EQUALITY, not containment: the pointer must be the REPO-RELATIVE path, and a
	// containment check is satisfied by an absolute path that merely ends with it —
	// which is exactly how a broken repo-relative rendering would read.
	wantPointer := ".fab-dispatch/" + id + "/apply-prompt.md"
	wantLine := "delivered " + id + "/apply (pane " + paneID + ", prompt " + wantPointer + ")\n"
	if stdout != wantLine {
		t.Errorf("stdout = %q, want exactly %q", stdout, wantLine)
	}

	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !rec.Delivered {
		t.Error("a verified delivery must flip the record's delivery marker")
	}
	if _, err := os.Stat(dispatch.ResultPath(dir, "apply")); !os.IsNotExist(err) {
		t.Error("delivery must clear the previous attempt's result file")
	}

	// The pane really received the POINTER — never the prompt body, which cannot
	// ride send-keys reliably.
	captured, err := tmux("capture-pane", "-p", "-t", paneID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(captured, wantPointer) {
		t.Errorf("pane captured %q, want the pointer naming %s", captured, wantPointer)
	}
	if strings.Contains(captured, "line two") {
		t.Errorf("pane received prompt BODY content (%q); only the pointer is delivered", captured)
	}
}

// TestDispatchDeliver_FailedContinuationStaysRecoverable is the fallback path the
// pane-arm resume rests on: resume is an OPTIMIZATION, so a continuation that
// fails to deliver must leave the dispatch exactly as it found it — otherwise the
// record sits at `delivered: true` with no result, derived state `running`, and
// every recovery verb refuses (`deliver` via the mid-stage guard, `open` via its
// already-running check), which would make the documented fresh-dispatch fallback
// unexecutable without an undocumented `fab dispatch kill`.
//
// A pane parked behind a wall (echo off, stable screen) is the failure: the
// readiness precondition refuses both attempts, so nothing is ever submitted.
func TestDispatchDeliver_FailedContinuationStaysRecoverable(t *testing.T) {
	repoRoot, id := setupDispatchRepoWithCommands(t, "", "claude")
	server := "fabtest-deliver-fail"
	_, paneID := newTmuxPane(t, server, parkedPaneCommand, 80)

	dir := seedPaneDispatch(t, repoRoot, id, "apply", paneID, server)
	rec, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatal(err)
	}
	rec.Delivered = true
	if err := dispatch.Save(dir, "apply", rec); err != nil {
		t.Fatal(err)
	}
	// The previous cycle's result — what makes this a CONTINUATION (state `done`,
	// the sanctioned case) rather than an initial delivery.
	const priorResult = "stage: apply\nstatus: success\nsummary: cycle 1\n"
	mustWrite(t, dispatch.ResultPath(dir, "apply"), priorResult)
	continuation := filepath.Join(dir, "apply-continuation.md")
	mustWrite(t, continuation, "rework: fix the thing\n")

	_, stderr, err := runDeliver(t, "abcd", "apply", "--prompt-file", continuation)
	if err == nil {
		t.Fatal("delivery into a parked pane must fail")
	}
	// The capture snippet is the last thing written to stderr, so it must end its
	// line — the same shared termination `ready`'s report relies on.
	if !strings.HasSuffix(stderr, "\n") {
		t.Errorf("stderr = %q, want the capture snippet to end with a newline", stderr)
	}

	got, err := os.ReadFile(dispatch.ResultPath(dir, "apply"))
	if err != nil {
		t.Fatalf("the previous result must survive a delivery that never verified: %v", err)
	}
	if string(got) != priorResult {
		t.Errorf("restored result = %q, want the untouched %q", got, priorResult)
	}

	// Which is what keeps the fallback executable: the dispatch reads `done`, not
	// `running`, so a fresh `fab dispatch open` is not refused.
	after, err := dispatch.Load(dir, "apply")
	if err != nil {
		t.Fatal(err)
	}
	running, err := priorRunning(dir, "apply", after)
	if err != nil {
		t.Fatal(err)
	}
	if running {
		t.Error("a failed continuation left the dispatch reading `running`; the fresh-dispatch fallback would be refused")
	}
}
