package main

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// rkOperatorHelpV32212 is `rk operator --help`'s command listing from rk
// v3.22.12 (adopt/request/track, no loop verbs), plus a decoy line that
// mentions "check" and "brief" mid-sentence.
const rkOperatorHelpV32212 = `Open (or switch to) the run-kit operator.

Use this to check on the operator and brief yourself.

Available Commands:
  adopt       Import fab's operator state file into rk's (read-only, never clobbers)
  request     Hand the server's operator agent a templated work item
  track       Manage the operator's tracked items

Flags:
  -h, --help             help for operator
`

// rkOperatorHelpV32213 is the same listing from rk v3.22.13 (captured from
// the real binary's help): adds the brief and check rows.
const rkOperatorHelpV32213 = `Open (or switch to) the run-kit operator.

Available Commands:
  adopt       Import fab's operator state file into rk's (read-only, never clobbers)
  brief       Print the run brief for the operator's tracked items
  check       Probe tracked items and report what changed since the last check
  request     Hand the server's operator agent a templated work item
  track       Manage the operator's tracked items

Flags:
  -h, --help             help for operator
`

func TestRkHelpHasHandover(t *testing.T) {
	cases := []struct {
		name string
		help string
		want bool
	}{
		{"v3.22.12", rkOperatorHelpV32212, false},
		{"v3.22.13", rkOperatorHelpV32213, true},
		{"check only", "Available Commands:\n  check   x\n", false},
		{"brief only", "Available Commands:\n  brief   x\n", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rkHelpHasHandover(tc.help); got != tc.want {
				t.Errorf("rkHelpHasHandover = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRkOperatorHandoverProbeRkAbsent drives the REAL probe with an empty
// PATH: rk absent answers false and prints nothing.
func TestRkOperatorHandoverProbeRkAbsent(t *testing.T) {
	handoverOnce, handoverResult = sync.Once{}, false
	t.Cleanup(func() { handoverOnce, handoverResult = sync.Once{}, false })
	t.Setenv("PATH", t.TempDir())
	stdout, stderr := captureStdio(t, func() {
		if probeRkOperatorHandover() {
			t.Error("probe = true with rk absent, want false")
		}
	})
	if stdout != "" || stderr != "" {
		t.Errorf("probe printed stdout=%q stderr=%q, want nothing", stdout, stderr)
	}
}

// captureStdio runs fn with os.Stdout/os.Stderr redirected to pipes and
// returns what was written.
func captureStdio(t *testing.T, fn func()) (string, string) {
	t.Helper()
	read := func(r *os.File, into *string, done chan<- struct{}) {
		var b bytes.Buffer
		_, _ = b.ReadFrom(r)
		*into = b.String()
		done <- struct{}{}
	}
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	prevOut, prevErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	var so, se string
	done := make(chan struct{}, 2)
	go read(outR, &so, done)
	go read(errR, &se, done)
	fn()
	os.Stdout, os.Stderr = prevOut, prevErr
	outW.Close()
	errW.Close()
	<-done
	<-done
	return so, se
}

// --- delegation scaffolding ---------------------------------------------------

// stubOperatorHandover sets the probe seam for one test.
func stubOperatorHandover(t *testing.T, capable bool) {
	t.Helper()
	prev := rkOperatorHandover
	rkOperatorHandover = func() bool { return capable }
	t.Cleanup(func() { rkOperatorHandover = prev })
}

// stubRkDelegate records every delegated rk argv; results[i] is the exit code
// of the i-th call (default 0), startErr makes every call fail to start.
func stubRkDelegate(t *testing.T, startErr error, results ...int) *[][]string {
	t.Helper()
	calls := [][]string{}
	prev := rkDelegateRunner
	rkDelegateRunner = func(args ...string) (int, error) {
		calls = append(calls, append([]string{}, args...))
		if startErr != nil {
			return 0, startErr
		}
		if i := len(calls) - 1; i < len(results) {
			return results[i], nil
		}
		return 0, nil
	}
	t.Cleanup(func() { rkDelegateRunner = prev })
	return &calls
}

// runFabOp drives the live command tree through run() (exit-code mapping
// included) and returns the exit code, stdout, stderr.
func runFabOp(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// handoverSandbox points operator state at an empty temp path and stubs the
// cron seam so a leak shows up as a recorded call; returns the state path and
// the cron call log.
func handoverSandbox(t *testing.T) (string, *[][]string) {
	t.Helper()
	statePath := withOperatorState(t, "")
	return statePath, stubRkCron(t, "[]", nil, nil)
}

func assertNoStateFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("operator state file exists (err=%v); a delegated verb must not touch it", err)
	}
}

func countDeprecations(stderr string) int {
	return strings.Count(stderr, "is deprecated")
}

// --- R3 / R2: track delegation ------------------------------------------------

func TestHandoverTrackAddDelegates(t *testing.T) {
	stubOperatorHandover(t, true)
	statePath, cron := handoverSandbox(t)
	calls := stubRkDelegate(t, nil)

	code, _, stderr := runFabOp("operator", "track", "add", "pr-1", "--kind", "github-pr",
		"--scope", `{"pr":1}`, "--argv", "gh", "--argv", "pr", "--fields", "state,mergedAt")
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", code, stderr)
	}
	want := [][]string{{"operator", "track", "add", "pr-1",
		"--argv", "gh", "--argv", "pr", "--fields", "state,mergedAt", "--kind", "github-pr", "--scope", `{"pr":1}`}}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("rk calls = %v, want %v", *calls, want)
	}
	if n := countDeprecations(stderr); n != 1 {
		t.Errorf("deprecation lines = %d, want 1 (stderr %q)", n, stderr)
	}
	assertNoStateFile(t, statePath)
	if len(*cron) != 0 {
		t.Errorf("cron calls = %v, want none", *cron)
	}
}

func TestHandoverTrackSimpleVerbs(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"list", []string{"track", "list", "--json", "--kind", "pane"}, []string{"operator", "track", "list", "--json", "--kind", "pane"}},
		{"rm", []string{"track", "rm", "w1"}, []string{"operator", "track", "rm", "w1"}},
		{"observe", []string{"track", "observe", "w1", "--json", `{"a":1}`, "--seen", "x", "--seen", "y"},
			[]string{"operator", "track", "observe", "w1", "--json", `{"a":1}`, "--seen", "x", "--seen", "y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubOperatorHandover(t, true)
			statePath, _ := handoverSandbox(t)
			calls := stubRkDelegate(t, nil)
			code, _, _ := runFabOp(append([]string{"operator"}, tc.args...)...)
			if code != 0 {
				t.Fatalf("exit = %d, want 0", code)
			}
			if !reflect.DeepEqual(*calls, [][]string{tc.want}) {
				t.Errorf("rk calls = %v, want %v", *calls, tc.want)
			}
			assertNoStateFile(t, statePath)
		})
	}
}

func TestHandoverTrackUpdateSplit(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want [][]string
	}{
		{"mixed", []string{"track", "update", "w1", "--then", "x", "--pause"},
			[][]string{{"operator", "track", "update", "w1", "--then", "x"}, {"operator", "track", "pause", "w1"}}},
		{"pause only", []string{"track", "update", "w1", "--pause"}, [][]string{{"operator", "track", "pause", "w1"}}},
		{"resume only", []string{"track", "update", "w1", "--resume"}, [][]string{{"operator", "track", "resume", "w1"}}},
		{"text only", []string{"track", "update", "w1", "--text", "hello"}, [][]string{{"operator", "track", "note", "w1", "hello"}}},
		{"update then note", []string{"track", "update", "w1", "--check-every", "5m", "--text", "t"},
			[][]string{{"operator", "track", "update", "w1", "--check-every", "5m"}, {"operator", "track", "note", "w1", "t"}}},
		{"plain update", []string{"track", "update", "w1", "--depends-on", "a,b"},
			[][]string{{"operator", "track", "update", "w1", "--depends-on", "a,b"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubOperatorHandover(t, true)
			statePath, _ := handoverSandbox(t)
			calls := stubRkDelegate(t, nil)
			code, _, _ := runFabOp(append([]string{"operator"}, tc.args...)...)
			if code != 0 {
				t.Fatalf("exit = %d, want 0", code)
			}
			if !reflect.DeepEqual(*calls, tc.want) {
				t.Errorf("rk calls = %v, want %v", *calls, tc.want)
			}
			assertNoStateFile(t, statePath)
		})
	}
}

func TestHandoverTrackUpdateStopsOnFailure(t *testing.T) {
	stubOperatorHandover(t, true)
	handoverSandbox(t)
	calls := stubRkDelegate(t, nil, 7) // the update call fails with 7
	code, _, _ := runFabOp("operator", "track", "update", "w1", "--then", "x", "--pause")
	if code != 7 {
		t.Errorf("exit = %d, want 7", code)
	}
	if len(*calls) != 1 {
		t.Errorf("rk calls = %v, want the sequence to stop after the first", *calls)
	}
}

func TestHandoverTrackUpdatePauseResumeIsUsageError(t *testing.T) {
	stubOperatorHandover(t, true)
	handoverSandbox(t)
	calls := stubRkDelegate(t, nil)
	code, _, _ := runFabOp("operator", "track", "update", "w1", "--pause", "--resume")
	if code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if len(*calls) != 0 {
		t.Errorf("rk calls = %v, want none", *calls)
	}
}

func TestHandoverDelegateExitCodes(t *testing.T) {
	t.Run("non-zero rk exit propagates", func(t *testing.T) {
		stubOperatorHandover(t, true)
		handoverSandbox(t)
		stubRkDelegate(t, nil, 5)
		if code, _, _ := runFabOp("operator", "track", "rm", "w1"); code != 5 {
			t.Errorf("exit = %d, want 5", code)
		}
	})
	t.Run("rk cannot start", func(t *testing.T) {
		stubOperatorHandover(t, true)
		handoverSandbox(t)
		stubRkDelegate(t, errors.New("exec: rk: not found"))
		code, _, stderr := runFabOp("operator", "track", "rm", "w1")
		if code != 1 {
			t.Errorf("exit = %d, want 1", code)
		}
		if !strings.Contains(stderr, "cannot run rk") {
			t.Errorf("stderr = %q, want a 'cannot run rk' error", stderr)
		}
	})
}

// --- R4: tick-start -----------------------------------------------------------

func TestHandoverTickStartDelegates(t *testing.T) {
	stubOperatorHandover(t, true)
	statePath, cron := handoverSandbox(t)
	calls := stubRkDelegate(t, nil)
	code, _, stderr := runFabOp("operator", "tick-start", "--diff", "--quiet")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !reflect.DeepEqual(*calls, [][]string{{"operator", "check"}}) {
		t.Errorf("rk calls = %v, want [operator check]", *calls)
	}
	if n := countDeprecations(stderr); n != 1 {
		t.Errorf("deprecation lines = %d, want 1", n)
	}
	assertNoStateFile(t, statePath)
	if len(*cron) != 0 {
		t.Errorf("cron calls = %v, want none", *cron)
	}
}

// --- R5: track clock ----------------------------------------------------------

func TestHandoverTrackClockRefused(t *testing.T) {
	stubOperatorHandover(t, true)
	statePath, cron := handoverSandbox(t)
	calls := stubRkDelegate(t, nil)
	code, _, stderr := runFabOp("operator", "track", "clock", "--every", "5m", "--for", "1h")
	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "rk cron edit") || !strings.Contains(stderr, "rk cron mute") {
		t.Errorf("stderr = %q, want the rk cron pointer", stderr)
	}
	if len(*calls) != 0 || len(*cron) != 0 {
		t.Errorf("rk calls = %v / cron calls = %v, want none", *calls, *cron)
	}
	assertNoStateFile(t, statePath)
}

// --- R6: clock sync -----------------------------------------------------------

func TestHandoverClockSyncListsOnly(t *testing.T) {
	stubOperatorHandover(t, true)
	// A live state with a tracked item: the fallback would reconcile; the
	// delegated arm must not read it or touch cron.
	withOperatorState(t, "tracked:\n  - id: w1\n    kind: shell\n")
	cron := stubRkCron(t, cronListBackoffMutedJSON, nil, nil)
	calls := stubRkDelegate(t, nil)
	code, _, stderr := runFabOp("operator", "clock", "sync")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !reflect.DeepEqual(*calls, [][]string{{"cron", "list"}}) {
		t.Errorf("rk calls = %v, want [cron list]", *calls)
	}
	if len(*cron) != 0 {
		t.Errorf("cron seam calls = %v, want none", *cron)
	}
	if n := countDeprecations(stderr); n != 1 {
		t.Errorf("deprecation lines = %d, want 1", n)
	}
}

// --- R2: deprecation placement ------------------------------------------------

func TestHandoverDeprecationLine(t *testing.T) {
	t.Run("time prints it on stderr only, stdout unchanged", func(t *testing.T) {
		stubOperatorHandover(t, true)
		code, stdout, stderr := runFabOp("operator", "time")
		if code != 0 {
			t.Fatalf("exit = %d, want 0", code)
		}
		if n := countDeprecations(stderr); n != 1 {
			t.Errorf("deprecation lines = %d, want 1 (stderr %q)", n, stderr)
		}
		if strings.Contains(stdout, "deprecated") || !strings.HasPrefix(stdout, "now: ") {
			t.Errorf("stdout = %q, want only the time line", stdout)
		}
	})
	t.Run("fallback arm prints none", func(t *testing.T) {
		stubOperatorHandover(t, false)
		code, _, stderr := runFabOp("operator", "time")
		if code != 0 || stderr != "" {
			t.Errorf("exit = %d stderr = %q, want 0 and empty", code, stderr)
		}
	})
	t.Run("agent commands never print it", func(t *testing.T) {
		stubOperatorHandover(t, true)
		_, _, stderr := runFabOp("agent", "--help")
		if strings.Contains(stderr, "deprecated") {
			t.Errorf("fab agent printed a deprecation: %q", stderr)
		}
	})
}

func TestHandoverBareLauncherPrintsNoDeprecation(t *testing.T) {
	stubOperatorHandover(t, true)
	prevProbe, prevExec := rkOperatorPath, execOperator
	t.Cleanup(func() { rkOperatorPath, execOperator = prevProbe, prevExec })
	rkOperatorPath = func() (string, bool) { return "/fake/bin/rk", true }
	execOperator = func(string, []string, []string) error { return nil }
	code, _, stderr := runFabOp("operator")
	if code != 0 || strings.Contains(stderr, "deprecated") {
		t.Errorf("exit = %d stderr = %q, want 0 and no deprecation", code, stderr)
	}
}

func TestOperatorReplacement(t *testing.T) {
	for verb, want := range map[string]string{
		"track add":   "rk operator track add",
		"tick-start":  "rk operator check",
		"clock sync":  "rk cron list",
		"track clock": "rk cron edit",
		"time":        "",
		"state":       "",
	} {
		if got := operatorReplacement(verb); got != want {
			t.Errorf("operatorReplacement(%q) = %q, want %q", verb, got, want)
		}
	}
}

// --- R7: cron-write gate ------------------------------------------------------

func TestHandoverGateIssuesNoCronWrites(t *testing.T) {
	stubOperatorHandover(t, true)
	calls := stubRkCron(t, cronListBackoffMutedJSON, nil, nil)
	tracked := map[string]interface{}{"tracked": []interface{}{map[string]interface{}{
		"id": "w1", "kind": "shell", "probe": map[string]interface{}{"mode": "shell", "check_every": "5m"},
	}}}
	mutedRow := operatorCronRow{ID: "cron-op", Muted: true}

	syncOperatorClock(false, true)
	syncOperatorClock(true, false)
	reconcileOperatorMute(tracked, mutedRow)
	reconcileOperatorMute(map[string]interface{}{}, operatorCronRow{ID: "cron-op"})
	muteOperatorClockIfUntracked(map[string]interface{}{})
	reconcileOperatorSchedule(tracked)
	if row, ok := ensureOperatorCronRow(); ok {
		t.Errorf("ensureOperatorCronRow = %v, true; want not-ok under a capable rk", row)
	}
	if row, ok := seedOperatorCronRow(); ok {
		t.Errorf("seedOperatorCronRow = %v, true; want not-ok under a capable rk", row)
	}
	if len(*calls) != 0 {
		t.Errorf("rk cron calls = %v, want none", *calls)
	}
}

// TestHandoverGateOffRestoresFallback pins that the same calls DO write when
// the probe is false (the gate is the only difference).
func TestHandoverGateOffRestoresFallback(t *testing.T) {
	stubOperatorHandover(t, false)
	calls := stubRkCron(t, cronListBackoffMutedJSON, nil, nil)
	syncOperatorClock(false, true)
	wantMutes(t, *calls, []string{"cron-op", "--off"})
}
