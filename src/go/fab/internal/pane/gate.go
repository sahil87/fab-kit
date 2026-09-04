package pane

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// This file holds the PANE READINESS GATE and the VERIFIED DELIVERY
// choreography — the shared, pane-addressed mechanics of the pane layer. The
// provider-generic primitives (`fab pane ready` / `fab pane deliver`) run them
// directly against a pane id, and `fab dispatch ready` / `fab dispatch
// deliver` are thin record-keeping bindings over the same gate — the two
// halves of how a pane-mode worker gets its prompt now that `fab dispatch
// open` spawns the pane WITHOUT one.
//
// Why delivery moved off the spawn command: a spawn-time positional pointer is
// fire-and-forget and unverifiable (a provider CLI that silently drops it leaves
// a worker at an empty prompt while the dispatch reads `running`), and requiring
// one made pane capability hostage to whether a provider's CLI happens to accept
// a positional prompt. Typing the pointer through tmux and CHECKING that it
// landed makes delivery observable, and the same engine serves both the initial
// dispatch and a rework-cycle continuation. See docs/specs/harness-adapters.md
// § 3.
//
// Why the gate probes in two stages: the sentinel echo is only trustworthy
// once an agent owns the pane. A tmux pane starts with a SHELL holding the
// pty in COOKED mode, where the tty echoes typed characters by itself — so in
// the window between the pane spawning and the provider binary putting the
// pty into raw mode, the sentinel echoes for a reason that has nothing to do
// with an agent being ready, and every downstream verification built on the
// echo fails in the same direction (a "delivered" prompt the booting TUI
// silently discards). The gate therefore first asks who owns the pane
// (`#{pane_current_command}` — the same foreground-command signal the
// operator's agent_exited delta keys on): a shell foreground classifies
// `booting` with NO keystroke sent at all. Post-takeover, the mechanical
// classification runs in two arms: it DELEGATES to `rk mux await --ready`
// when a sentinel-capable run-kit is on PATH (gate_rk.go — "fab consumes,
// never reimplements", docs/specs/agent-messaging.md), and falls back to
// fab's own ECHO- and STABILITY-based probe when rk is absent, too old, or
// fails unexpectedly (fail-open, one stderr warning per process). Neither arm
// carries a table of known dialogs, because dialog text is a version
// treadmill and a half-matched pattern pressing Enter into an unknown screen
// is worse than stalling. Deciding what a `parked` screen wants is the
// orchestrator's judgment, over the snippet this file returns.

// Readiness is the gate's classification of a pane. The values are the exact
// strings `fab pane ready` and `fab dispatch ready` print — they are the report
// contract the orchestrator branches on, so they are named constants, never
// inline literals (the State precedent in internal/dispatch).
type Readiness string

const (
	// ReadyReady: the sentinel echoed — the pane accepts typed input.
	ReadyReady Readiness = "ready"
	// ReadyBooting: there is no agent to answer yet — either the pane's
	// foreground command is still a shell (the provider binary has not taken
	// the tty), or the sentinel did not echo on an empty or still-changing
	// screen (the TUI is plausibly still starting).
	ReadyBooting Readiness = "booting"
	// ReadyParked: a stable screen that swallowed the sentinel — a dialog,
	// survey, login wall, or wedged process is holding the input.
	ReadyParked Readiness = "parked"
)

// ReadySentinel is the probe text. It is typed LITERALLY and never submitted,
// then cleared with C-u, so it can only ever appear as un-submitted input. The
// value is deliberately unmistakable in a capture and free of shell/TUI
// metacharacters.
const ReadySentinel = "FAB-READY-PROBE"

// SnippetLines is how many trailing lines of the pane a non-`ready` report and a
// delivery failure carry, so the orchestrator can judge the screen without a
// second capture call.
const SnippetLines = 20

// captureLines is how much scrollback the gate pulls per capture. It is wider
// than SnippetLines so the echo and stability checks see more than the reported
// snippet — a sentinel typed at the bottom of a tall pane must still be found.
const captureLines = 50

// Gate timings. All three are internal constants with NO flag and NO config
// field (the `wait` tick precedent): they are mechanics, not policy. They are
// unexported like their sibling mechanics captureLines and deliveryAttempts —
// NewGate is the only way a caller gets them, and it applies them itself; tests
// reach the values through the Gate struct fields, which is also how they zero
// them.
const (
	// settleDelay is how long a send is given to reach the screen before the
	// verifying capture.
	settleDelay = 300 * time.Millisecond
	// stabilityDelay spaces the two captures whose difference separates
	// `booting` (screen still moving) from `parked` (screen stable).
	stabilityDelay = 500 * time.Millisecond
	// busyDelay is how long the worker is given to react to Enter before
	// delivery checks that the screen advanced.
	busyDelay = 800 * time.Millisecond
)

// deliveryAttempts is the total number of delivery attempts: one try plus
// exactly ONE retry. A second failure is reported with the capture rather than
// retried again — a pane that failed verification twice needs eyes, not a loop.
const deliveryAttempts = 2

// PaneIO is the tmux surface the gate uses: read the pane's foreground
// command, capture the screen, type literal text, press a named key. It is an
// interface so the whole choreography — including the retry path, which no
// pure function can express — is testable against a scripted fake with no
// tmux server, matching this package's established preference for
// table-testable decisions.
type PaneIO interface {
	CurrentCommand(paneID string) (string, error)
	Capture(paneID string, lines int) (string, error)
	SendLiteral(paneID, text string) error
	SendKey(paneID, key string) error
}

// tmuxPaneIO is the real implementation, delegating to this package's shared
// helpers so the gate and `fab pane capture` both go through
// one tmux argv builder and one stderr-enrichment convention.
type tmuxPaneIO struct{ server string }

func (t tmuxPaneIO) CurrentCommand(paneID string) (string, error) {
	return CurrentCommand(t.server, paneID)
}

func (t tmuxPaneIO) Capture(paneID string, lines int) (string, error) {
	return Capture(t.server, paneID, lines)
}

func (t tmuxPaneIO) SendLiteral(paneID, text string) error {
	return SendLiteral(t.server, paneID, text)
}

func (t tmuxPaneIO) SendKey(paneID, key string) error {
	return SendKey(t.server, paneID, key)
}

// Gate runs the readiness probe and the delivery choreography against one tmux
// server. The delay fields are populated by NewGate; a zero delay simply skips
// its sleep, which is what makes the choreography instant under test. Server is
// the tmux socket name the delegated rk arm threads to `rk mux await -L` (empty
// = the default socket); it duplicates tmuxPaneIO's own copy because the rk
// argv builder sits outside the IO interface.
type Gate struct {
	IO        PaneIO
	Server    string
	Settle    time.Duration
	Stability time.Duration
	Busy      time.Duration
}

// NewGate returns a Gate driving a real tmux server (empty server = the default
// socket) with the shipped timings.
func NewGate(server string) *Gate {
	return &Gate{
		IO:        tmuxPaneIO{server: server},
		Server:    server,
		Settle:    settleDelay,
		Stability: stabilityDelay,
		Busy:      busyDelay,
	}
}

func (g *Gate) sleep(d time.Duration) {
	if d > 0 {
		time.Sleep(d)
	}
}

// DeriveReadiness is the pure classifier: given whether the sentinel echoed and
// two captures spaced by the stability delay, it names the pane's state. Kept
// free of I/O so the whole table is testable against scripted captures, exactly
// like internal/dispatch's DeriveState/DerivePaneState.
//
//	echoed                        → ready
//	blank screen                  → booting  (nothing has been drawn yet)
//	screen changed between reads  → booting  (a TUI still painting itself)
//	stable, non-blank, no echo    → parked   (something is holding the input)
//
// The blank-screen case precedes the difference check because two identical
// EMPTY captures are stable by the letter of the rule while meaning the exact
// opposite of parked.
func DeriveReadiness(echoed bool, first, second string) Readiness {
	if echoed {
		return ReadyReady
	}
	if strings.TrimSpace(second) == "" {
		return ReadyBooting
	}
	if first != second {
		return ReadyBooting
	}
	return ReadyParked
}

// Probe classifies a pane and returns a trailing capture snippet for every
// non-`ready` answer, so the caller never needs a second capture call.
//
// AGENT TAKEOVER PRECONDITION: before anything is typed, the pane's
// foreground command is read. While it is still a shell, the provider binary
// has not taken the tty, so there is no agent to probe — and a cooked-mode
// shell echoes typed characters by itself, so the sentinel would echo for a
// reason that has nothing to do with readiness. That path reports `booting`
// and sends NOTHING, regardless of echo; the echo is the untrustworthy
// signal and is not consulted. The precondition runs ahead of BOTH
// classification arms below — rk's await deliberately classifies a cooked-
// shell echo as ready (terminals-are-one-standard), which for a dispatch pane
// is exactly the false-ready this guard exists to close, so rk is never
// invoked while a shell owns the pane.
//
// Past the precondition the classification runs in TWO ARMS. When a sentinel-
// capable run-kit is on PATH (see gate_rk.go), the mechanical classification
// delegates to `rk mux await --ready` (probeRK): state-present and sentinel-
// echo both report `ready`, `parked` and a bound-expiry `running` report
// `parked`/`booting` with a FAB-CAPTURED snippet, and `gone` surfaces as the
// ordinary dead-pane error. Any unexpected rk failure fails OPEN to the raw
// arm with one stderr warning per process. The raw-tmux arm (the rk-less
// fallback, and the whole classifier when rk is absent) is purely ECHO- and
// STABILITY-based — send a sentinel, see whether it appears, see whether the
// screen is still moving. Neither arm carries a table of known dialogs,
// because dialog text is a version treadmill and a half-matched pattern
// pressing Enter into an unknown screen is worse than stalling. Deciding what
// a `parked` screen wants is the orchestrator's judgment, over the snippet
// this file returns.
//
// It is READ-MOSTLY and idempotent (Constitution III): the only thing it
// writes to the pane is the sentinel, which is typed literally, never
// submitted, and cleared with C-u whether or not it echoed — and even that is
// skipped on the pre-takeover path. It presses no other key and answers
// nothing.
func (g *Gate) Probe(paneID string) (Readiness, string, error) {
	cmd, err := g.IO.CurrentCommand(paneID)
	if err != nil {
		return "", "", err
	}
	if IsShellCommand(cmd) {
		return g.bootingSnippet(paneID)
	}
	// rk arm: the delegated classifier. It runs ONLY here — past the takeover
	// precondition — so its sentinel is never typed into a cooked-mode shell.
	if rkSentinelCapable() {
		state, snippet, handled, err := g.probeRK(paneID)
		if err != nil {
			return "", "", err
		}
		if handled {
			return state, snippet, nil
		}
		// handled=false is the fail-open path (warnRKFallback already warned):
		// the raw-tmux arm below classifies this same probe.
	}
	if err := g.IO.SendLiteral(paneID, ReadySentinel); err != nil {
		return "", "", err
	}
	g.sleep(g.Settle)
	after, err := g.IO.Capture(paneID, captureLines)
	if err != nil {
		return "", "", err
	}
	echoed := countWrapped(after, ReadySentinel) > 0

	// BOTH stability captures are taken before the sentinel is cleared. C-u is
	// itself a keystroke, and a TUI that repaints its input line in response to it
	// would make every capture pair straddling the clear differ — so a genuinely
	// parked pane would read `booting` on every probe and only ever reach `parked`
	// by exhausting the wiring's consecutive-boot allowance. The difference the
	// classifier measures must come from the pane's own repainting, not from the
	// probe's own keystroke.
	settled := after
	if !echoed {
		g.sleep(g.Stability)
		settled, err = g.IO.Capture(paneID, captureLines)
		if err != nil {
			return "", "", err
		}
	}

	// Clear unconditionally: a sentinel that echoed must not be left in the
	// worker's input buffer, and one that did not echo may still have landed
	// somewhere the capture did not show.
	if err := g.IO.SendKey(paneID, KeyClear); err != nil {
		return "", "", err
	}
	if echoed {
		return ReadyReady, "", nil
	}
	return DeriveReadiness(false, after, settled), Snippet(settled), nil
}

// probeRK is the delegated classification: one bounded `rk mux await --ready`
// call whose report maps onto the frozen Readiness contract. handled=false
// (with err=nil) is the fail-open case — an unexpected rk failure or an
// unparsable report — and the caller falls through to the raw-tmux arm for the
// same probe; a CLASSIFIED outcome is an answer and is never re-classified by
// the fallback (that would double-type the sentinel).
//
//	rk outcome                          → fab result
//	ready %N (state|echo), exit 0       → ready, no snippet
//	parked %N, exit 0                   → parked + fab-captured snippet
//	running (timeout), exit 0           → booting + fab-captured snippet
//	gone %N, exit 1                     → the dead-pane error (not a classification)
//	any other exit / unparsable stdout  → fail-open (one stderr warning per process)
//
// Non-`ready` snippets come from fab's OWN Capture+Snippet path (one extra
// capture), never rk's stderr, so the `--- last N lines ---` report form and
// the --json snippet field stay byte-identical on both arms.
func (g *Gate) probeRK(paneID string) (state Readiness, snippet string, handled bool, err error) {
	out, rkStderr, runErr := rkAwaitRunner(paneID, g.Server)
	token := rkReportToken(out)
	switch {
	case runErr == nil && token == string(ReadyReady):
		return ReadyReady, "", true, nil
	case runErr == nil && token == string(ReadyParked):
		snippet, err := g.captureSnippet(paneID)
		if err != nil {
			return "", "", false, err
		}
		return ReadyParked, snippet, true, nil
	case runErr == nil && token == "running":
		snippet, err := g.captureSnippet(paneID)
		if err != nil {
			return "", "", false, err
		}
		return ReadyBooting, snippet, true, nil
	case runErr != nil && token == "gone":
		// The pane died (rk's contract: `gone` exits 1 — the non-zero exit is
		// load-bearing, so an exit-0 `gone` token would fall through to
		// fail-open): the same refusal the identity check produces, on the
		// existing dead-pane error path — rk's answer is honored, never
		// re-classified by the fallback.
		return "", "", false, &PaneNotFoundError{Pane: paneID}
	default:
		warnRKFallback(paneID, g.Server, runErr, rkStderr)
		return "", "", false, nil
	}
}

// captureSnippet is the non-`ready` evidence fetch: one capture through the
// same Snippet path every arm reports, so the orchestrator's judgment material
// is identical however the classification was reached.
func (g *Gate) captureSnippet(paneID string) (string, error) {
	capture, err := g.IO.Capture(paneID, captureLines)
	if err != nil {
		return "", err
	}
	return Snippet(capture), nil
}

// bootingSnippet is the takeover precondition's report: `booting` with the
// current screen as evidence, nothing typed.
func (g *Gate) bootingSnippet(paneID string) (Readiness, string, error) {
	snippet, err := g.captureSnippet(paneID)
	if err != nil {
		return "", "", err
	}
	return ReadyBooting, snippet, nil
}

// Deliver types pointer into a pane and VERIFIES that it landed: readiness
// probe → C-u → literal pointer → capture-verify the echo → Enter → confirm the
// screen advanced. Any failed verification costs one attempt; there is exactly
// one retry, and the second failure returns an error carrying the last capture
// as snippet.
//
// Verification is what makes this different from the spawn-time argument it
// replaces: every step that could silently do nothing is checked against the
// screen.
//
// Non-fatal outcomes come back as warnings for the caller to surface, the same
// shape OpenSplitPane uses — here, the fact that a first attempt failed and was
// retried, which is worth reporting even when the delivery ultimately succeeds.
func (g *Gate) Deliver(paneID, pointer string) (warnings []error, snippet string, err error) {
	for attempt := 1; attempt <= deliveryAttempts; attempt++ {
		snippet, err = g.deliverOnce(paneID, pointer)
		if err == nil {
			return warnings, "", nil
		}
		if attempt < deliveryAttempts {
			warnings = append(warnings, fmt.Errorf("delivery attempt %d failed (%v); retrying", attempt, err))
		}
	}
	return warnings, snippet, fmt.Errorf("could not deliver the prompt to pane %s after %d attempts: %w",
		paneID, deliveryAttempts, err)
}

// deliverOnce is a single delivery attempt. It returns the last capture it took
// alongside any failure, so the caller can show the screen that refused the
// delivery.
func (g *Gate) deliverOnce(paneID, pointer string) (snippet string, err error) {
	state, evidence, err := g.Probe(paneID)
	if err != nil {
		return "", err
	}
	if state != ReadyReady {
		return evidence, fmt.Errorf("pane is %s, not ready", state)
	}

	if err := g.IO.SendKey(paneID, KeyClear); err != nil {
		return "", err
	}
	// The echo baseline is taken HERE — after this attempt's own clear, not from
	// the probe capture above. On a RETRY the probe capture still shows the
	// pointer the previous attempt typed but never submitted (a pointer left on
	// the input line is exactly what the busy-check failure looks like), so
	// baselining against it would compare 1 against 1 and report `did not echo`
	// for an attempt that typed perfectly — killing the one retry for the very
	// failure class it exists to recover. The settle is the clear's, mirroring
	// the one every other verifying capture gets: a baseline read before C-u
	// reaches the screen would carry the same leftover.
	g.sleep(g.Settle)
	cleared, err := g.IO.Capture(paneID, captureLines)
	if err != nil {
		return "", err
	}
	if err := g.IO.SendLiteral(paneID, pointer); err != nil {
		return "", err
	}
	g.sleep(g.Settle)
	typed, err := g.IO.Capture(paneID, captureLines)
	if err != nil {
		return "", err
	}
	if !newlyEchoed(cleared, typed, pointer) {
		return Snippet(typed), fmt.Errorf("the pointer line did not echo in the pane")
	}

	if err := g.IO.SendKey(paneID, KeyEnter); err != nil {
		return "", err
	}
	g.sleep(g.Busy)
	submitted, err := g.IO.Capture(paneID, captureLines)
	if err != nil {
		return "", err
	}
	// The worker went busy iff the screen moved. A submitted prompt is echoed
	// into the transcript, a spinner starts, or the input line clears — every
	// agent TUI repaints something. An unchanged screen means Enter did nothing,
	// which is exactly the silent-drop failure this choreography exists to catch.
	if submitted == typed {
		return Snippet(submitted), fmt.Errorf("the pane did not react to Enter (the prompt may not have been submitted)")
	}
	return "", nil
}

// Snippet returns the trailing SnippetLines of a capture, for the evidence a
// non-`ready` report and a failed delivery carry.
//
// Trailing blank lines are dropped FIRST: tmux pads a capture to the pane's full
// height, so a dialog drawn near the top of a tall pane sits above twenty empty
// lines and a raw tail would report a blank screen for exactly the case the
// snippet exists to show. Padding is not content, and the snippet's whole job is
// to show the orchestrator what is holding the input.
//
// A non-empty snippet is newline-TERMINATED here rather than at each print site:
// it is whole lines, and every caller writes it as the last thing on its stream,
// so an unterminated tail would leave the shell prompt sitting on the evidence.
// An empty snippet stays empty — a lone newline would be a blank line reporting
// nothing.
func Snippet(capture string) string {
	trimmed := string(Tail([]byte(strings.TrimRight(capture, " \t\r\n")), SnippetLines))
	if trimmed == "" {
		return ""
	}
	return trimmed + "\n"
}

// newlyEchoed reports whether the keystrokes ADDED the pointer to the screen, by
// comparing how often it appears now against how often it appeared in the
// baseline — the capture taken after the attempt's own C-u, just before the
// pointer was typed. A plain containment test verifies a delivery whose
// keystrokes never landed whenever the pointer is already somewhere in the
// captured scrollback — reachable on a repeat continuation that reuses the same
// `--prompt-file` path, and precisely the silent-drop failure this check exists
// to catch. A leftover pointer in scrollback is in the baseline too, so the
// counting comparison keeps that protection while letting a retry verify the
// pointer it just typed.
func newlyEchoed(baseline, typed, pointer string) bool {
	return countWrapped(typed, pointer) > countWrapped(baseline, pointer)
}

// countWrapped counts needle's occurrences in a pane capture, ignoring
// WHITESPACE and BOX-DRAWING runes in both. tmux hard-wraps a pane's visible
// lines at the pane width, so a pointer longer than a narrow pane arrives in the
// capture split across lines; a plain substring test would then miss text that is
// plainly on screen. Dropping both classes from both sides makes the check
// wrap-independent without needing to know the pane's width — or how the TUI
// frames its input line.
//
// The two classes cover the two shapes a wrap can insert, both probed live:
//
//   - NO side borders (2026-08-09): claude at 50 and at 30 columns (narrow enough
//     to wrap mid-word) and agy at 50 columns each yield countWrapped == 1 with
//     whitespace alone dropped — both draw their input box as horizontal rules,
//     so nothing but whitespace lands between the wrapped halves.
//   - SIDE BORDERS (2026-08-10, kimi 0.34.0): kimi draws vertical rules down both
//     sides of its input box, so a wrap interleaves `││` between the halves and a
//     whitespace-only squeeze read 0 for a pointer plainly on screen. Dropping
//     U+2500–U+257F is what admits that class.
//
// The drop is deliberately range-scoped rather than a broader
// "ignore non-alphanumerics" normalization: a ReadySentinel and a prompt-file
// pointer line never legitimately contain frame runes, so removing them cannot
// mask a genuine mismatch, while a wider class would grow the false-positive
// surface with no probed need. The failure mode of a wrong answer here is a loud
// double failure into the gate's escalation, never a false success.
//
// Presence is `countWrapped(...) > 0` — deliberately not a second helper, since
// a containment twin would be a near-identical squeeze pair that can drift.
func countWrapped(capture, needle string) int {
	return strings.Count(squeeze(capture), squeeze(needle))
}

// boxDrawing reports whether r is in the Unicode box-drawing block — the frame
// runes a TUI draws its input box with. Named rather than inlined so the range
// the squeeze tolerates has one readable definition.
func boxDrawing(r rune) bool {
	return r >= 0x2500 && r <= 0x257F
}

// squeeze removes every whitespace and box-drawing rune from s.
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || boxDrawing(r) {
			return -1
		}
		return r
	}, s)
}

// Tail returns the last n lines of data (Go-side, no external `tail`). n <= 0
// returns the whole content unchanged. A trailing newline is treated as a line
// terminator, not an empty final line, so `Tail(data, 1)` on "a\nb\n" yields
// "b\n". Deliberately distinct from pane.go's TailLines, the pane-capture
// tailer (string, strips trailing blank screen-padding) — do not consolidate.
func Tail(data []byte, n int) []byte {
	if n <= 0 || len(data) == 0 {
		return data
	}
	s := string(data)
	// Split off a single trailing newline so it doesn't count as a blank line.
	trailing := ""
	if strings.HasSuffix(s, "\n") {
		trailing = "\n"
		s = s[:len(s)-1]
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return data
	}
	return []byte(strings.Join(lines[len(lines)-n:], "\n") + trailing)
}
