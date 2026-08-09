package dispatch

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// This file holds the PANE READINESS GATE and the VERIFIED DELIVERY
// choreography — the two halves of how a pane-mode worker gets its prompt now
// that `fab dispatch open` spawns the pane WITHOUT one.
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
// Why the gate is mechanical: classification here is purely ECHO- and
// STABILITY-based — send a sentinel, see whether it appears, see whether the
// screen is still moving. It carries NO table of known dialogs, because dialog
// text is a version treadmill and a half-matched pattern pressing Enter into an
// unknown screen is worse than stalling. Deciding what a `parked` screen wants
// is the orchestrator's judgment, over the snippet this file returns.

// Readiness is the gate's classification of a pane. The values are the exact
// strings `fab dispatch ready` prints — they are the report contract the
// orchestrator branches on, so they are named constants, never inline literals
// (the State precedent in dispatch.go).
type Readiness string

const (
	// ReadyReady: the sentinel echoed — the pane accepts typed input.
	ReadyReady Readiness = "ready"
	// ReadyBooting: the sentinel did not echo, but the screen is empty or still
	// changing — the TUI is plausibly still starting.
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

// PaneIO is the tmux surface the gate uses: capture the screen, type literal
// text, press a named key. It is an interface so the whole choreography —
// including the retry path, which no pure function can express — is testable
// against a scripted fake with no tmux server, matching this package's
// established preference for table-testable decisions.
type PaneIO interface {
	Capture(paneID string, lines int) (string, error)
	SendLiteral(paneID, text string) error
	SendKey(paneID, key string) error
}

// tmuxPaneIO is the real implementation, delegating to internal/pane's shared
// helpers so the gate, `fab pane capture`, and `fab pane send` all go through
// one tmux argv builder and one stderr-enrichment convention.
type tmuxPaneIO struct{ server string }

func (t tmuxPaneIO) Capture(paneID string, lines int) (string, error) {
	return pane.Capture(t.server, paneID, lines)
}

func (t tmuxPaneIO) SendLiteral(paneID, text string) error {
	return pane.SendLiteral(t.server, paneID, text)
}

func (t tmuxPaneIO) SendKey(paneID, key string) error {
	return pane.SendKey(t.server, paneID, key)
}

// Gate runs the readiness probe and the delivery choreography against one tmux
// server. The delay fields are populated by NewGate; a zero delay simply skips
// its sleep, which is what makes the choreography instant under test.
type Gate struct {
	IO        PaneIO
	Settle    time.Duration
	Stability time.Duration
	Busy      time.Duration
}

// NewGate returns a Gate driving a real tmux server (empty server = the default
// socket) with the shipped timings.
func NewGate(server string) *Gate {
	return &Gate{
		IO:        tmuxPaneIO{server: server},
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
// like DeriveState/DerivePaneState.
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
// It is READ-MOSTLY and idempotent (Constitution III): the only thing it writes
// to the pane is the sentinel, which is typed literally, never submitted, and
// cleared with C-u whether or not it echoed. It presses no other key and answers
// nothing.
func (g *Gate) Probe(paneID string) (Readiness, string, error) {
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
	if err := g.IO.SendKey(paneID, pane.KeyClear); err != nil {
		return "", "", err
	}
	if echoed {
		return ReadyReady, "", nil
	}
	return DeriveReadiness(false, after, settled), Snippet(settled), nil
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

	if err := g.IO.SendKey(paneID, pane.KeyClear); err != nil {
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

	if err := g.IO.SendKey(paneID, pane.KeyEnter); err != nil {
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
// WHITESPACE in both. tmux hard-wraps a pane's visible lines at the pane width,
// so a pointer longer than a narrow pane arrives in the capture split across
// lines; a plain substring test would then miss text that is plainly on screen.
// Dropping all whitespace from both sides makes the check wrap-independent
// without needing to know the pane's width.
//
// Whitespace is the only thing dropped, so the tolerance holds exactly as long
// as a wrap inserts nothing but whitespace into the line. Probed live on
// 2026-08-09 against the two TUIs that can be reached today: claude at 50 and at
// 30 columns (narrow enough to wrap mid-word) and agy at 50 columns each yield
// countWrapped == 1 for a typed pointer — both draw their input box as
// horizontal rules with NO side borders, so nothing but whitespace lands between
// the wrapped halves. A TUI that boxed its input line with vertical rules would
// interleave frame runes and read 0; kimi is unprobed and rides backlog [agik]'s
// pre-shipping echo probe. The failure mode of a wrong answer here is a loud
// double failure into the gate's escalation, never a false success.
//
// Presence is `countWrapped(...) > 0` — deliberately not a second helper, since
// a containment twin would be a near-identical squeeze pair that can drift.
func countWrapped(capture, needle string) int {
	return strings.Count(squeeze(capture), squeeze(needle))
}

// squeeze removes every whitespace rune from s.
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
