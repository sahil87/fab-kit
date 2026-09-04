package pane

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realRKSentinelProbe captures the shipped probe before TestMain replaces it,
// so the genuine LookPath+help behavior stays testable.
var realRKSentinelProbe = rkSentinelProbe

// TestMain keeps the whole package's tests hermetic: the host machine HAS a
// sentinel-capable rk installed, and the pre-existing raw-arm tests construct
// their gates with no rk stubbing — they exercise the raw-tmux fallback by
// definition. Forcing the capability answer off by default keeps those tests
// byte-identical in behavior; rk-arm tests opt back in via forceRKCapable.
func TestMain(m *testing.M) {
	rkSentinelProbe = func() bool { return false }
	os.Exit(m.Run())
}

// forceRKCapable pins the capability answer for the duration of a test,
// bypassing the probe (the cache is set directly) and restoring both on
// cleanup.
func forceRKCapable(t *testing.T, capable bool) {
	t.Helper()
	rkCapableMu.Lock()
	prevProbe, prevKnown, prevValue := rkSentinelProbe, rkCapableKnown, rkCapableValue
	rkSentinelProbe = func() bool { return capable }
	rkCapableKnown, rkCapableValue = true, capable
	rkCapableMu.Unlock()
	t.Cleanup(func() {
		rkCapableMu.Lock()
		rkSentinelProbe, rkCapableKnown, rkCapableValue = prevProbe, prevKnown, prevValue
		rkCapableMu.Unlock()
	})
}

// stubRKAwait replaces the rk runner seam with a scripted one, returning a
// pointer to the recorded invocations (pane/server pairs) for assertion.
func stubRKAwait(t *testing.T, stdout string, stderr []byte, err error) *[][2]string {
	t.Helper()
	calls := &[][2]string{}
	prev := rkAwaitRunner
	rkAwaitRunner = func(paneID, server string) (string, []byte, error) {
		*calls = append(*calls, [2]string{paneID, server})
		return stdout, stderr, err
	}
	t.Cleanup(func() { rkAwaitRunner = prev })
	return calls
}

// captureRKWarnings replaces the warning sink and resets the once-per-process
// guard, returning a pointer to the captured warnings.
func captureRKWarnings(t *testing.T) *[]string {
	t.Helper()
	warnings := &[]string{}
	prevWarn := rkWarn
	rkWarn = func(format string, args ...interface{}) {
		*warnings = append(*warnings, fmt.Sprintf(format, args...))
	}
	rkWarnMu.Lock()
	prevWarned := rkWarned
	rkWarned = false
	rkWarnMu.Unlock()
	t.Cleanup(func() {
		rkWarn = prevWarn
		rkWarnMu.Lock()
		rkWarned = prevWarned
		rkWarnMu.Unlock()
	})
	return warnings
}

// TestRKAwaitArgs pins the delegated probe's argv: the sentinel --ready wait
// against the pane, rk's -L socket flag threaded only when a server is set,
// and the bounded internal timeout in rk's seconds unit.
func TestRKAwaitArgs(t *testing.T) {
	tests := []struct {
		name   string
		paneID string
		server string
		want   []string
	}{
		{"default socket", "%5", "", []string{"mux", "await", "--ready", "%5", "--timeout", "20"}},
		{"named server threads -L", "%5", "work", []string{"mux", "await", "--ready", "%5", "-L", "work", "--timeout", "20"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rkAwaitArgs(tt.paneID, tt.server)
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Errorf("rkAwaitArgs(%q, %q) = %v, want %v", tt.paneID, tt.server, got, tt.want)
			}
		})
	}
}

// TestRKHelpSentinelCapable is the capability probe's parsing table: the
// sentinel contract's `parked` report word is the discriminant, because
// `--ready` predates the sentinel classifier (capture-settle semantics) and a
// version string can lie.
func TestRKHelpSentinelCapable(t *testing.T) {
	tests := []struct {
		name string
		help string
		want bool
	}{
		{"sentinel help mentions parked", "--ready   Wait until the pane is boot-ready (agent state present, else a sentinel echo probe: echo = ready, no echo = parked)", true},
		{"pre-sentinel capture-settle help lacks it", "--ready   Wait until the pane is boot-ready (a settled screen is classified ready)", false},
		{"empty help", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rkHelpSentinelCapable(tt.help); got != tt.want {
				t.Errorf("rkHelpSentinelCapable(%q) = %v, want %v", tt.help, got, tt.want)
			}
		})
	}
}

// TestRKSentinelProbeAgainstAStubbedBinary runs the REAL probe (LookPath +
// `rk mux await --help`) against a scripted `rk` on PATH: capability is a
// property of the installed binary's help, so the probe is tested against a
// binary, not a stubbed function.
func TestRKSentinelProbeAgainstAStubbedBinary(t *testing.T) {
	writeRK := func(t *testing.T, help string) {
		t.Helper()
		dir := t.TempDir()
		script := "#!/bin/sh\nprintf '%s' \"$RK_STUB_HELP\"\n"
		if err := os.WriteFile(filepath.Join(dir, "rk"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", dir)
		t.Setenv("RK_STUB_HELP", help)
	}

	t.Run("help with parked is capable", func(t *testing.T) {
		writeRK(t, "echo = ready, no echo = parked")
		if !realRKSentinelProbe() {
			t.Error("probe = false, want true for a sentinel-capable help text")
		}
	})
	t.Run("help without parked is not capable", func(t *testing.T) {
		writeRK(t, "a settled screen is classified ready")
		if realRKSentinelProbe() {
			t.Error("probe = true, want false for a pre-sentinel help text")
		}
	})
	t.Run("rk absent from PATH is not capable", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		if realRKSentinelProbe() {
			t.Error("probe = true, want false when no rk is on PATH")
		}
	})
}

// TestRKSentinelCapableCachesPerProcess pins the at-most-once-per-process
// probe: the gate loop re-probes, and a per-probe `rk mux await --help`
// subprocess would tax every iteration.
func TestRKSentinelCapableCachesPerProcess(t *testing.T) {
	calls := 0
	rkCapableMu.Lock()
	prevProbe, prevKnown, prevValue := rkSentinelProbe, rkCapableKnown, rkCapableValue
	rkSentinelProbe = func() bool { calls++; return true }
	rkCapableKnown, rkCapableValue = false, false
	rkCapableMu.Unlock()
	defer func() {
		rkCapableMu.Lock()
		rkSentinelProbe, rkCapableKnown, rkCapableValue = prevProbe, prevKnown, prevValue
		rkCapableMu.Unlock()
	}()

	if !rkSentinelCapable() {
		t.Fatal("capable = false, want true from the stubbed probe")
	}
	rkSentinelCapable()
	rkSentinelCapable()
	if calls != 1 {
		t.Errorf("probe ran %d times, want exactly 1 (the answer is cached per process)", calls)
	}
}

// TestProbeRKMapping is the rk arm's classification table: each rk report maps
// onto the frozen Readiness contract, with non-`ready` snippets FAB-CAPTURED
// (never rk's stderr) so the report form stays byte-identical on both arms.
func TestProbeRKMapping(t *testing.T) {
	tests := []struct {
		name        string
		stdout      string
		stderr      []byte
		runErr      error
		want        Readiness
		wantSnippet string
	}{
		{"state fast path", "ready %17 (state)\n", nil, nil, ReadyReady, ""},
		{"sentinel echo", "ready %17 (echo)\n", nil, nil, ReadyReady, ""},
		{"parked carries a fab-captured snippet", "parked %17\n", []byte("rk's own stderr snippet\n"), nil, ReadyParked, "Trust this folder?"},
		{"timeout running maps to booting", "running\n", nil, nil, ReadyBooting, "Trust this folder?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forceRKCapable(t, true)
			calls := stubRKAwait(t, tt.stdout, tt.stderr, tt.runErr)
			io := newFakeIO("Trust this folder? [y/N]")
			state, snippet, err := testGate(io).Probe("%17")
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			if state != tt.want {
				t.Errorf("state = %q, want %q", state, tt.want)
			}
			if tt.wantSnippet == "" && snippet != "" {
				t.Errorf("snippet = %q, want empty", snippet)
			}
			if tt.wantSnippet != "" && !strings.Contains(snippet, tt.wantSnippet) {
				t.Errorf("snippet = %q, want fab's own capture containing %q (not rk's stderr)", snippet, tt.wantSnippet)
			}
			if strings.Contains(snippet, "rk's own stderr snippet") {
				t.Errorf("snippet = %q must not pass rk's stderr through", snippet)
			}
			if len(*calls) != 1 || (*calls)[0][0] != "%17" {
				t.Errorf("rk calls = %v, want exactly one for %%17", *calls)
			}
			if len(io.sends) != 0 {
				t.Errorf("sends = %v, want NOTHING typed fab-side on the rk arm", io.sends)
			}
		})
	}
}

// TestProbeRKThreadsTheServer pins that a non-default socket reaches rk as its
// own `-L` flag.
func TestProbeRKThreadsTheServer(t *testing.T) {
	forceRKCapable(t, true)
	calls := stubRKAwait(t, "ready %17 (state)\n", nil, nil)
	io := newFakeIO()
	g := testGate(io)
	g.Server = "work"
	if _, _, err := g.Probe("%17"); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0][1] != "work" {
		t.Errorf("rk calls = %v, want the server threaded through", *calls)
	}
}

// TestProbeRKGoneIsTheDeadPaneError pins that rk's `gone` is an ERROR on the
// existing dead-pane path — never a classification, never a fallback trigger.
func TestProbeRKGoneIsTheDeadPaneError(t *testing.T) {
	forceRKCapable(t, true)
	stubRKAwait(t, "gone %17\n", nil, fmt.Errorf("exit status 1"))
	warnings := captureRKWarnings(t)
	io := newFakeIO("$ " + ReadySentinel)
	_, _, err := testGate(io).Probe("%17")
	var notFound *PaneNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want the dead-pane error (PaneNotFoundError)", err)
	}
	if len(io.sends) != 0 {
		t.Errorf("sends = %v, want the raw arm NEVER to run after rk answered gone", io.sends)
	}
	if len(*warnings) != 0 {
		t.Errorf("warnings = %v, want none — gone is an answer, not a failure", *warnings)
	}
}

// TestProbeRKFailOpen is the fail-open table: any UNEXPECTED rk outcome (a
// non-zero exit, unparsable stdout, empty stdout) warns once and lets the
// raw-tmux arm classify the same probe — a failure of the delegation is not a
// classification.
func TestProbeRKFailOpen(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		stderr []byte
		runErr error
	}{
		{"non-zero exit", "", []byte("boom\n"), fmt.Errorf("exit status 127")},
		{"unparsable report token", "confused %17\n", nil, nil},
		{"empty stdout on exit 0", "", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forceRKCapable(t, true)
			stubRKAwait(t, tt.stdout, tt.stderr, tt.runErr)
			warnings := captureRKWarnings(t)
			// The raw arm then sees the sentinel echo and classifies ready —
			// proof the fallback classified THIS probe.
			io := newFakeIO("$ " + ReadySentinel)
			state, _, err := testGate(io).Probe("%17")
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			if state != ReadyReady {
				t.Errorf("state = %q, want %q from the raw-tmux fallback", state, ReadyReady)
			}
			if !strings.Contains(strings.Join(io.sends, ","), "literal:"+ReadySentinel) {
				t.Errorf("sends = %v, want the raw arm's sentinel typed after fail-open", io.sends)
			}
			if len(*warnings) != 1 {
				t.Errorf("warnings = %v, want exactly one fail-open warning", *warnings)
			}
		})
	}
}

// TestProbeRKWarningFiresOncePerProcess pins the warn-once rule: a per-probe
// warning would spam the gate loop's re-probes.
func TestProbeRKWarningFiresOncePerProcess(t *testing.T) {
	forceRKCapable(t, true)
	stubRKAwait(t, "", []byte("boom\n"), fmt.Errorf("exit status 127"))
	warnings := captureRKWarnings(t)
	io := newFakeIO("$ " + ReadySentinel)
	for i := 0; i < 3; i++ {
		if _, _, err := testGate(io).Probe("%17"); err != nil {
			t.Fatalf("Probe %d: %v", i, err)
		}
	}
	if len(*warnings) != 1 {
		t.Errorf("warnings = %v, want exactly ONE per process across repeated probes", *warnings)
	}
}

// TestProbeRKNeverRunsWhileAShellOwnsThePane pins the load-bearing ordering:
// the takeover precondition runs AHEAD of the rk arm. rk's await deliberately
// classifies a cooked-shell echo as ready (terminals-are-one-standard), which
// for a dispatch pane is exactly the 57mp false-ready — so a shell foreground
// reports booting and rk is never invoked.
func TestProbeRKNeverRunsWhileAShellOwnsThePane(t *testing.T) {
	forceRKCapable(t, true)
	calls := stubRKAwait(t, "ready %17 (echo)\n", nil, nil)
	io := newFakeIO("$ " + ReadySentinel) // the cooked-mode false-echo shape
	io.command = "zsh"
	state, snippet, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyBooting {
		t.Errorf("state = %q, want %q while a shell owns the pane", state, ReadyBooting)
	}
	if len(*calls) != 0 {
		t.Errorf("rk calls = %v, want NONE on the shell-foreground path", *calls)
	}
	if len(io.sends) != 0 {
		t.Errorf("sends = %v, want NOTHING typed on the shell-foreground path", io.sends)
	}
	if snippet == "" {
		t.Error("a booting report carries the screen snippet, got empty")
	}
}

// TestProbeRawArmWhenNotCapable pins the rk-less/too-old degradation: a
// negative capability answer keeps the probe on the raw-tmux arm with no rk
// invocation and no warning (absence degrades silently — only unexpected
// failures warn).
func TestProbeRawArmWhenNotCapable(t *testing.T) {
	forceRKCapable(t, false)
	calls := stubRKAwait(t, "ready %17 (echo)\n", nil, nil)
	warnings := captureRKWarnings(t)
	io := newFakeIO("$ " + ReadySentinel)
	state, _, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyReady {
		t.Errorf("state = %q, want %q from the raw arm", state, ReadyReady)
	}
	if len(*calls) != 0 {
		t.Errorf("rk calls = %v, want NONE when rk is not sentinel-capable", *calls)
	}
	if len(*warnings) != 0 {
		t.Errorf("warnings = %v, want none — rk absence/incapability is silent", *warnings)
	}
	if !strings.Contains(strings.Join(io.sends, ","), "literal:"+ReadySentinel) {
		t.Errorf("sends = %v, want the raw arm's sentinel typed", io.sends)
	}
}
