package dispatch

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// fakePaneIO is a scripted tmux stand-in: captures come from a queue (the last
// one repeats once exhausted, so a test only has to script the captures it cares
// about), and every send is recorded so the choreography's key sequence can be
// asserted. It is what makes the retry path — which no pure function can express
// — testable without a tmux server.
type fakePaneIO struct {
	captures []string
	sends    []string
	// ops is every operation in order, captures included, for the assertions that
	// are about SEQUENCING rather than about what was typed.
	ops    []string
	failOn map[string]error
}

func newFakeIO(captures ...string) *fakePaneIO {
	return &fakePaneIO{captures: captures, failOn: map[string]error{}}
}

func (f *fakePaneIO) Capture(_ string, _ int) (string, error) {
	if err := f.failOn["capture"]; err != nil {
		return "", err
	}
	f.ops = append(f.ops, "capture")
	if len(f.captures) == 0 {
		return "", nil
	}
	out := f.captures[0]
	if len(f.captures) > 1 {
		f.captures = f.captures[1:]
	}
	return out, nil
}

func (f *fakePaneIO) SendLiteral(_ string, text string) error {
	if err := f.failOn["literal"]; err != nil {
		return err
	}
	f.sends = append(f.sends, "literal:"+text)
	f.ops = append(f.ops, "literal:"+text)
	return nil
}

func (f *fakePaneIO) SendKey(_ string, key string) error {
	if err := f.failOn["key"]; err != nil {
		return err
	}
	f.sends = append(f.sends, "key:"+key)
	f.ops = append(f.ops, "key:"+key)
	return nil
}

// testGate builds a Gate over a fake with every delay zeroed, so the whole
// choreography runs instantly.
func testGate(io PaneIO) *Gate { return &Gate{IO: io} }

// TestDeriveReadiness is the pure classifier's table. Every row is a screen the
// gate must name correctly without any tmux involved — the same
// exhaustively-table-tested treatment DeriveState and DerivePaneState get.
func TestDeriveReadiness(t *testing.T) {
	tests := []struct {
		name          string
		echoed        bool
		first, second string
		want          Readiness
	}{
		{"echo wins over everything", true, "anything", "anything", ReadyReady},
		{"echo wins even on a moving screen", true, "a", "b", ReadyReady},
		{"blank screen is booting, not parked", false, "", "", ReadyBooting},
		{"whitespace-only screen is booting", false, " \n\n", " \n\n", ReadyBooting},
		{"changing screen is booting", false, "loading .", "loading ..", ReadyBooting},
		{"stable non-blank screen with no echo is parked", false, "Trust this folder? [y/N]", "Trust this folder? [y/N]", ReadyParked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveReadiness(tt.echoed, tt.first, tt.second); got != tt.want {
				t.Errorf("DeriveReadiness(%v, %q, %q) = %q, want %q", tt.echoed, tt.first, tt.second, got, tt.want)
			}
		})
	}
}

// TestProbeReportsReadyOnEcho pins the happy path AND the two rules that make the
// probe safe to re-run: the sentinel is typed LITERALLY (never as tmux key names)
// and is always cleared with C-u, and no Enter is ever pressed.
func TestProbeReportsReadyOnEcho(t *testing.T) {
	io := newFakeIO("$ " + ReadySentinel)
	state, snippet, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyReady {
		t.Errorf("state = %q, want %q", state, ReadyReady)
	}
	if snippet != "" {
		t.Errorf("a ready report carries no snippet, got %q", snippet)
	}
	want := []string{"literal:" + ReadySentinel, "key:" + pane.KeyClear}
	if strings.Join(io.sends, ",") != strings.Join(want, ",") {
		t.Errorf("sends = %v, want %v", io.sends, want)
	}
}

// TestProbeClearsSentinelWhenItDidNotEcho pins that C-u fires on the NON-echo
// path too: a sentinel that did not show up in the capture may still have landed
// somewhere the capture did not cover, and leaving it in a worker's input buffer
// would corrupt the prompt delivered next.
func TestProbeClearsSentinelWhenItDidNotEcho(t *testing.T) {
	io := newFakeIO("Trust this folder? [y/N]")
	state, snippet, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyParked {
		t.Errorf("state = %q, want %q", state, ReadyParked)
	}
	if !strings.Contains(snippet, "Trust this folder?") {
		t.Errorf("snippet = %q, want the parked screen", snippet)
	}
	if !contains(strings.Join(io.sends, ","), "key:"+pane.KeyClear) {
		t.Errorf("sends = %v, want a C-u clear", io.sends)
	}
	for _, s := range io.sends {
		if s == "key:"+pane.KeyEnter {
			t.Error("the probe must never press Enter")
		}
	}
}

// TestProbeReportsBootingOnMovingScreen scripts two DIFFERENT captures for the
// stability comparison: the sentinel never echoes, but the screen is repainting,
// which is a starting TUI rather than a wall.
func TestProbeReportsBootingOnMovingScreen(t *testing.T) {
	io := newFakeIO("starting .", "starting ..")
	state, _, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyBooting {
		t.Errorf("state = %q, want %q", state, ReadyBooting)
	}
}

// TestProbeCapturesTheStabilityPairBeforeClearing pins the ordering the
// booting/parked split depends on. C-u is a keystroke like any other, and a TUI
// that repaints its input line in response to it would make a capture pair
// straddling the clear differ every time — so a genuinely parked pane would read
// `booting` forever and only reach `parked` by exhausting the wiring's
// consecutive-boot allowance.
func TestProbeCapturesTheStabilityPairBeforeClearing(t *testing.T) {
	io := newFakeIO("Trust this folder? [y/N]")
	if _, _, err := testGate(io).Probe("%17"); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	want := []string{"literal:" + ReadySentinel, "capture", "capture", "key:" + pane.KeyClear}
	if strings.Join(io.ops, ",") != strings.Join(want, ",") {
		t.Errorf("ops =\n  %v\nwant\n  %v", io.ops, want)
	}
}

// TestProbeFindsWrappedSentinel pins the wrap tolerance: tmux hard-wraps a pane's
// visible lines at the pane width, so a capture can split the sentinel across
// lines. A plain substring test would call a plainly-visible sentinel absent and
// report `parked` for a perfectly ready pane.
func TestProbeFindsWrappedSentinel(t *testing.T) {
	wrapped := ReadySentinel[:5] + "\n" + ReadySentinel[5:]
	io := newFakeIO("$ " + wrapped)
	state, _, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyReady {
		t.Errorf("state = %q, want %q for a wrapped sentinel", state, ReadyReady)
	}
}

// boxedInput renders text as a kimi-style input box: a horizontal rule, each
// wrapped line between VERTICAL side rules, and a closing rule. It is the shape
// probed live on 2026-08-10 (kimi 0.34.0) — the one that defeated the former
// whitespace-only squeeze, because a wrap interleaves `││` between the halves.
func boxedInput(wrapped ...string) string {
	out := "╭" + strings.Repeat("─", 40) + "╮\n"
	for _, line := range wrapped {
		out += "│ " + line + strings.Repeat(" ", 8) + "│\n"
	}
	return out + "╰" + strings.Repeat("─", 40) + "╯\n"
}

// TestProbeFindsSentinelWrappedInABoxedInputLine is the wrap tolerance's second
// shape: a TUI that frames its input line with VERTICAL rules interleaves those
// frame runes between the wrapped halves, so a whitespace-only squeeze read the
// sentinel as absent and reported `parked` for a pane that had plainly echoed it.
func TestProbeFindsSentinelWrappedInABoxedInputLine(t *testing.T) {
	io := newFakeIO(boxedInput("> "+ReadySentinel[:5], ReadySentinel[5:]))
	state, _, err := testGate(io).Probe("%17")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if state != ReadyReady {
		t.Errorf("state = %q, want %q for a sentinel wrapped inside a side-bordered input box", state, ReadyReady)
	}
}

// TestDeliverVerifiesEchoInABoxedInputLine is the same tolerance at the OTHER
// call site: the pointer is longer than the sentinel and therefore the line that
// actually wraps in practice, so the delivery verification is where a boxed TUI
// failed first (a doubly-verified delivery that could never verify).
func TestDeliverVerifiesEchoInABoxedInputLine(t *testing.T) {
	io := newFakeIO(
		boxedInput("> "+ReadySentinel[:5], ReadySentinel[5:]), // probe: ready
		boxedInput("> "), // after C-u: the echo baseline
		boxedInput("> "+testPointer[:30], testPointer[30:]), // the pointer echoed, wrapped inside the box
		boxedInput("> ")+"● working…",                       // after Enter: the screen advanced
	)
	warnings, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err != nil {
		t.Fatalf("Deliver: %v (snippet %q)", err, snippet)
	}
	if len(warnings) != 0 {
		t.Errorf("a clean delivery warns about nothing, got %v", warnings)
	}
}

// TestCountWrappedDropsOnlyFrameRunes pins that the normalization stays
// RANGE-SCOPED. Dropping whitespace and box-drawing runes is safe because a
// sentinel or pointer never legitimately contains them; widening it to
// punctuation generally would start matching text the pane never showed, turning
// the verifier's loud double failure into a false success.
func TestCountWrappedDropsOnlyFrameRunes(t *testing.T) {
	if got := countWrapped(boxedInput("> "+testPointer[:30], testPointer[30:]), testPointer); got != 1 {
		t.Errorf("countWrapped over a boxed wrap = %d, want 1", got)
	}
	// Same text with the pointer's own punctuation stripped: still a mismatch,
	// because only whitespace and frame runes are ignored.
	stripped := strings.ReplaceAll(testPointer, ".", "")
	if got := countWrapped(boxedInput("> "+stripped), testPointer); got != 0 {
		t.Errorf("countWrapped over a capture missing the pointer's punctuation = %d, want 0 — the drop must not extend past whitespace and box-drawing runes", got)
	}
}

// TestSnippetSkipsThePanePadding pins that the evidence a report carries is the
// screen, not the pane's blank padding: tmux pads a capture to the pane's full
// height, so a wall drawn near the top of a tall pane sits above more empty lines
// than the snippet is long, and a raw tail would show the orchestrator nothing at
// the exact moment it has to judge what is holding the input.
func TestSnippetSkipsThePanePadding(t *testing.T) {
	capture := "Trust this folder? [y/N]" + strings.Repeat("\n", 2*SnippetLines)
	if got := Snippet(capture); !strings.Contains(got, "Trust this folder?") {
		t.Errorf("Snippet = %q, want the dialog rather than the padding below it", got)
	}
}

// TestSnippetTerminatesItsLastLine pins the newline every print site relies on:
// both `fab dispatch ready` and a failed `fab dispatch deliver` write the snippet
// as the last thing on their stream, so an unterminated tail would leave the
// shell prompt sitting on the evidence line. An EMPTY snippet stays empty — a
// lone newline there would be a blank line reporting nothing.
func TestSnippetTerminatesItsLastLine(t *testing.T) {
	if got := Snippet("Trust this folder? [y/N]"); !strings.HasSuffix(got, "\n") {
		t.Errorf("Snippet = %q, want a trailing newline", got)
	}
	if got := Snippet("   \n\n"); got != "" {
		t.Errorf("Snippet = %q, want %q for an all-blank capture", got, "")
	}
}

const testPointer = "Read .fab-dispatch/abcd/apply-prompt.md and execute it."

// TestDeliverVerifiesEchoAndSubmission pins the full happy-path choreography and
// its exact key sequence: probe, clear, type the pointer, Enter. Every step that
// could silently do nothing is checked against the screen, which is the whole
// reason delivery moved off the spawn command.
func TestDeliverVerifiesEchoAndSubmission(t *testing.T) {
	io := newFakeIO(
		"$ "+ReadySentinel, // probe: sentinel echoed ⇒ ready
		"$ ",               // after C-u: the echo baseline, input line clear
		"$ "+testPointer,   // after typing the pointer: it echoed
		"● working…",       // after Enter: the screen advanced
	)
	warnings, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err != nil {
		t.Fatalf("Deliver: %v (snippet %q)", err, snippet)
	}
	if len(warnings) != 0 {
		t.Errorf("a clean delivery warns about nothing, got %v", warnings)
	}
	want := []string{
		"literal:" + ReadySentinel,
		"key:" + pane.KeyClear,
		"key:" + pane.KeyClear,
		"literal:" + testPointer,
		"key:" + pane.KeyEnter,
	}
	if strings.Join(io.sends, ",") != strings.Join(want, ",") {
		t.Errorf("sends =\n  %v\nwant\n  %v", io.sends, want)
	}
	// The full operation sequence, captures included, because WHERE the echo
	// baseline is taken is load-bearing: it must come AFTER this attempt's C-u,
	// or a retry baselines against the pointer the previous attempt left on the
	// input line and can never verify (see TestDeliverRetriesAfterAnIgnoredEnter).
	wantOps := []string{
		"literal:" + ReadySentinel,
		"capture", // probe
		"key:" + pane.KeyClear,
		"key:" + pane.KeyClear,
		"capture", // the echo baseline — after the clear, before the pointer
		"literal:" + testPointer,
		"capture", // echo check
		"key:" + pane.KeyEnter,
		"capture", // busy check
	}
	if strings.Join(io.ops, ",") != strings.Join(wantOps, ",") {
		t.Errorf("ops =\n  %v\nwant\n  %v", io.ops, wantOps)
	}
}

// TestDeliverFailsWhenEnterDoesNothing is the silent-drop case the choreography
// exists to catch: the pointer echoed, Enter was pressed, and the screen did not
// move — so the prompt was never submitted. Both attempts see it, so the
// delivery fails with the pane's screen as evidence.
func TestDeliverFailsWhenEnterDoesNothing(t *testing.T) {
	// Both attempts: the probe reads ready, the pointer echoes, and the post-Enter
	// capture is identical to the pre-Enter one.
	io := newFakeIO(
		"$ "+ReadySentinel, // attempt 1 probe: ready
		"$ ",               // attempt 1: echo baseline after C-u
		"$ "+testPointer,   // attempt 1: the pointer echoed
		"$ "+testPointer,   // attempt 1: Enter changed nothing
		"$ "+ReadySentinel, // attempt 2 probe: ready
		"$ ",               // attempt 2: echo baseline after C-u
		"$ "+testPointer,   // attempt 2: echoed again
		"$ "+testPointer,   // attempt 2: Enter changed nothing again
	)
	warnings, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err == nil {
		t.Fatal("an unchanged screen after Enter must fail the delivery")
	}
	if !strings.Contains(err.Error(), "after 2 attempts") {
		t.Errorf("error = %q, want it to name the attempt budget", err)
	}
	if !strings.Contains(err.Error(), "did not react to Enter") {
		t.Errorf("error = %q, want it to name the unsubmitted prompt", err)
	}
	if len(warnings) != 1 {
		t.Errorf("want exactly one retry warning, got %v", warnings)
	}
	if !strings.Contains(snippet, testPointer) {
		t.Errorf("snippet = %q, want the refusing screen", snippet)
	}
}

// TestDeliverRetriesOnceThenSucceeds pins that a first-attempt failure is spent
// on a retry rather than surfaced, and that the retry's success still reports
// what happened — a delivery that needed two tries is worth knowing about even
// when it worked.
func TestDeliverRetriesOnceThenSucceeds(t *testing.T) {
	io := newFakeIO(
		"$ "+ReadySentinel, // attempt 1 probe: ready
		"$ ",               // attempt 1: echo baseline after C-u
		"$ ",               // attempt 1: the pointer did NOT echo
		"$ "+ReadySentinel, // attempt 2 probe: ready
		"$ ",               // attempt 2: echo baseline after C-u
		"$ "+testPointer,   // attempt 2: echoed
		"● working…",       // attempt 2: screen advanced
	)
	warnings, _, err := testGate(io).Deliver("%17", testPointer)
	if err != nil {
		t.Fatalf("the retry should have succeeded: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("want one retry warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0].Error(), "did not echo") {
		t.Errorf("warning = %q, want it to name the failed verification", warnings[0])
	}
}

// TestDeliverRetriesAfterAnIgnoredEnter is the retry's OTHER failure class, and
// the one the choreography most exists for: the pointer typed and echoed, Enter
// did nothing, and the un-submitted pointer is therefore still sitting on the
// input line when the retry starts. The retry must be able to verify the pointer
// it re-types over that leftover — which it can only do if the echo baseline is
// taken after its own C-u. Baselining against the probe capture instead compares
// one leftover against one freshly typed pointer and reports `did not echo`,
// hiding the real cause (`did not react to Enter`) in the attempt-1 warning and
// making the documented single retry unable to recover anything.
func TestDeliverRetriesAfterAnIgnoredEnter(t *testing.T) {
	io := newFakeIO(
		"$ "+ReadySentinel,             // attempt 1 probe: ready
		"$ ",                           // attempt 1: echo baseline after C-u
		"$ "+testPointer,               // attempt 1: the pointer echoed
		"$ "+testPointer,               // attempt 1: Enter did nothing
		"$ "+testPointer+ReadySentinel, // attempt 2 probe: the sentinel lands after the LEFTOVER pointer
		"$ ",                           // attempt 2: C-u wiped the leftover — the baseline holds no pointer
		"$ "+testPointer,               // attempt 2: the re-typed pointer echoed
		"● working…",                   // attempt 2: the screen advanced
	)
	warnings, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err != nil {
		t.Fatalf("the retry after an ignored Enter should have succeeded: %v (snippet %q)", err, snippet)
	}
	if len(warnings) != 1 {
		t.Fatalf("want one retry warning, got %v", warnings)
	}
	if !strings.Contains(warnings[0].Error(), "did not react to Enter") {
		t.Errorf("warning = %q, want it to name the ignored Enter — the cause the retry was spent on", warnings[0])
	}
}

// TestDeliverDoesNotVerifyAPointerAlreadyOnScreen is the false-verify hole in the
// one check the whole choreography exists to make trustworthy: on a repeat
// continuation reusing the same --prompt-file path, the pointer can already be in
// the captured scrollback, so a containment test would confirm an echo whose
// keystrokes never landed. The check is therefore against the screen as it stood
// after the clear and before the pointer was typed: the pointer must have been
// ADDED. Scrollback survives C-u, so moving the baseline behind the clear (what
// makes the retry work at all) leaves this protection exactly as strong.
func TestDeliverDoesNotVerifyAPointerAlreadyOnScreen(t *testing.T) {
	// One repeating capture holding the sentinel (so the probe reads ready) and a
	// pointer left over from an earlier delivery — and nothing the send adds.
	io := newFakeIO("$ " + testPointer + "\n$ " + ReadySentinel)
	_, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err == nil {
		t.Fatal("a pointer that was already on screen must not verify a delivery")
	}
	if !strings.Contains(err.Error(), "did not echo") {
		t.Errorf("error = %q, want the failed echo verification", err)
	}
	if !strings.Contains(snippet, testPointer) {
		t.Errorf("snippet = %q, want the screen that failed verification", snippet)
	}
}

// TestDeliverRefusesAParkedPane pins that delivery will not type a prompt into a
// pane that is not ready: the readiness probe is a precondition of every attempt,
// not a courtesy the caller may skip.
func TestDeliverRefusesAParkedPane(t *testing.T) {
	io := newFakeIO("Trust this folder? [y/N]")
	_, snippet, err := testGate(io).Deliver("%17", testPointer)
	if err == nil {
		t.Fatal("a parked pane must refuse delivery")
	}
	if !strings.Contains(err.Error(), string(ReadyParked)) {
		t.Errorf("error = %q, want it to name the parked classification", err)
	}
	if !strings.Contains(snippet, "Trust this folder?") {
		t.Errorf("snippet = %q, want the parked screen", snippet)
	}
	for _, s := range io.sends {
		if s == "literal:"+testPointer {
			t.Error("the pointer must not be typed into a pane that is not ready")
		}
	}
}

// TestDeliverPropagatesIOFailure pins that a tmux failure SURFACES rather than
// being absorbed into a verification verdict: a broken socket is not a screen the
// worker refused. The retry is still spent on it — Deliver retries any failed
// attempt, I/O included — so the outcome is the error, carried out of the second
// attempt, with the ordinary attempt-1 retry warning alongside it.
func TestDeliverPropagatesIOFailure(t *testing.T) {
	io := newFakeIO("$ " + ReadySentinel)
	io.failOn["capture"] = fmt.Errorf("no server running")
	if _, _, err := testGate(io).Deliver("%17", testPointer); err == nil {
		t.Fatal("a tmux capture failure must surface")
	}
}

// TestDeliveredMarkerIsPaneOnlyAndOmittedWhenFalse pins the record's additive
// shape: `delivered` appears only once it is true, so an `open`ed pane reads as
// undelivered by ABSENCE and every headless record's bytes are unchanged.
//
// The assertions read the MARSHALLED YAML, not the struct field: `omitempty` is
// the whole contract here, and a round-tripped struct reads false either way — it
// cannot tell an omitted key from a `delivered: false` one written into every
// headless record on disk.
func TestDeliveredMarkerIsPaneOnlyAndOmittedWhenFalse(t *testing.T) {
	dir := t.TempDir()

	readRecord := func(stage string) string {
		t.Helper()
		data, err := os.ReadFile(YAMLPath(dir, stage))
		if err != nil {
			t.Fatalf("read %s record: %v", stage, err)
		}
		return string(data)
	}

	if err := Save(dir, "apply", &Dispatch{Pane: "%17", Window: "fab-abcd-apply", SpawnCmd: "claude", StartedAt: "t"}); err != nil {
		t.Fatalf("Save opened: %v", err)
	}
	if got := readRecord("apply"); strings.Contains(got, "delivered") {
		t.Errorf("an `open`ed pane record carries a delivery key:\n%s", got)
	}
	opened, err := Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load opened: %v", err)
	}
	if opened.Delivered {
		t.Error("a pane record written by `open` must read as not delivered")
	}

	opened.Delivered = true
	if err := Save(dir, "apply", opened); err != nil {
		t.Fatalf("Save delivered: %v", err)
	}
	if got := readRecord("apply"); !strings.Contains(got, "delivered: true") {
		t.Errorf("a delivered pane record must carry `delivered: true`:\n%s", got)
	}
	delivered, err := Load(dir, "apply")
	if err != nil {
		t.Fatalf("Load delivered: %v", err)
	}
	if !delivered.Delivered {
		t.Error("the delivery marker must round-trip")
	}

	if err := Save(dir, "review", &Dispatch{PID: 7, PGID: 7, SpawnCmd: "codex exec", StartedAt: "t"}); err != nil {
		t.Fatalf("Save headless: %v", err)
	}
	if got := readRecord("review"); strings.Contains(got, "delivered") {
		t.Errorf("a headless record's bytes changed — it now carries a delivery key:\n%s", got)
	}
	headless, err := Load(dir, "review")
	if err != nil {
		t.Fatalf("Load headless: %v", err)
	}
	if headless.Delivered {
		t.Error("a headless record must never carry a delivery marker")
	}
}

// TestDeliveredChangesNoState pins that the marker is bookkeeping, not a state:
// the pane derivation is a function of the result file and pane liveness alone,
// exactly as before.
func TestDeliveredChangesNoState(t *testing.T) {
	for _, delivered := range []bool{false, true} {
		rec := &Dispatch{Pane: "%17", Delivered: delivered}
		if !rec.IsPane() || rec.Mode() != ModePane {
			t.Errorf("delivered=%v changed the derived mode", delivered)
		}
		if got := DerivePaneState(true, false); got != StateDone {
			t.Errorf("delivered=%v: result-present derivation = %q, want %q", delivered, got, StateDone)
		}
	}
}
