package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// Operator clock sync: the operator-tick cron entry (seeded by `rk operator`)
// carries no suppress guards — run-kit never infers operator intent from
// fab's private state file. Instead the tracked-set verbs TELL the clock:
// when a state mutation flips the tracked predicate, fab mutes the entry
// (tracked→untracked) or clears the mute (untracked→tracked) through rk's own
// verbs. Every rk call is LookPath-gated, argv-only, bounded by rkCronTimeout,
// and fail-silent — absent or older rk degrades to the loud-but-safe posture
// (ticks keep arriving) and never changes a verb's exit code or stdout. No
// -L <server> is passed: the operator verbs have no server flag, so rk's own
// $TMUX derivation is the addressing.

const (
	// rkCronTimeout bounds each rk subprocess (rk only writes an intent file
	// on disk, so 5s is generous).
	rkCronTimeout = 5 * time.Second
	// operatorCronTarget is the cron row's target string for the operator-tick
	// entry (a `{kind: role, role: operator}` target rendered by rk).
	operatorCronTarget = "role:operator"
	// operatorCronName is the entry's name, used as the tiebreak when several
	// rows carry the operator target.
	operatorCronName = "operator tick"
)

// operatorTracked reports whether the state holds tracked work: a non-empty
// monitored set, an autopilot block with a non-null state (an exhausted block
// — current/state null, queue/completed retained — counts as empty), any
// watch (enabled or disabled — a disabled watch still exists and may be
// re-enabled), or any unresolved kind: coordination note (a merge sequence
// keeps ticks coming even at merge-all time). A section that fails to decode
// counts as empty for that section — the predicate is a fail-silent side
// effect, never a verb-blocking error.
func operatorTracked(data map[string]interface{}) bool {
	monitored := map[string]monitoredEntry{}
	if err := operatorSection(data, "monitored", &monitored); err == nil && len(monitored) > 0 {
		return true
	}
	ap := autopilotState{}
	if err := operatorSection(data, "autopilot", &ap); err == nil && ap.State != nil {
		return true
	}
	watches := map[string]watchEntry{}
	if err := operatorSection(data, "watches", &watches); err == nil && len(watches) > 0 {
		return true
	}
	if notes, err := readNotes(data); err == nil {
		for _, n := range notes {
			if n.Kind == "coordination" && !n.Resolved {
				return true
			}
		}
	}
	return false
}

// rkCronRunner is the injectable seam for every rk cron invocation the clock
// sync makes (the rkPanesRunner / rkOperatorPath precedent): tests stub it to
// serve a fixture `rk cron list --json` document and record mute argv without
// a live rk. Args are the argv after "rk" — never a shell string. The real
// implementation is exec.LookPath-gated and bounded by rkCronTimeout; errors
// (rk absent, non-zero exit, timeout) are the caller's to swallow.
var rkCronRunner = func(args ...string) (string, error) {
	if _, err := exec.LookPath("rk"); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), rkCronTimeout)
	defer cancel()
	out, _, err := pane.RunCmdContext(ctx, "rk", args...)
	return out, err
}

// operatorCronRow is the subset of `rk cron list --json` row fields the clock
// sync reads (rk's contract, verified against rk v3.19.37): `muted` is the
// EFFECTIVE state (indefinite mute OR live lease), `muted_until` (unix
// seconds) is present only while a lease is live. Unknown fields are ignored.
type operatorCronRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Target     string `json:"target"`
	Muted      bool   `json:"muted"`
	MutedUntil int64  `json:"muted_until"`
}

// resolveOperatorCronRow runs `rk cron list --json` and selects the
// operator-tick entry: the row whose target equals operatorCronTarget, with
// name == operatorCronName as the tiebreak when several match. Zero
// candidates, an unresolved tie, an rk failure, or unparseable output is a
// silent no-op (ok=false).
func resolveOperatorCronRow() (operatorCronRow, bool) {
	out, err := rkCronRunner("cron", "list", "--json")
	if err != nil {
		return operatorCronRow{}, false
	}
	var rows []operatorCronRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		return operatorCronRow{}, false
	}
	var candidates []operatorCronRow
	for _, r := range rows {
		if r.Target == operatorCronTarget {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) == 1 {
		return candidates[0], true
	}
	var named []operatorCronRow
	for _, r := range candidates {
		if r.Name == operatorCronName {
			named = append(named, r)
		}
	}
	if len(named) == 1 {
		return named[0], true
	}
	return operatorCronRow{}, false
}

// syncOperatorClock is the edge-triggered clock side effect run after a
// successful state save: tracked→untracked issues an indefinite mute,
// untracked→tracked clears any standing mute or lease via --off. A
// non-flipping mutation issues no call — a tick costs zero rk calls and a
// user-set lease survives unrelated mutations. Fail-silent: all errors are
// swallowed and nothing is written to stdout/stderr.
func syncOperatorClock(before, after bool) {
	if before == after {
		return
	}
	row, ok := resolveOperatorCronRow()
	if !ok {
		return
	}
	if after {
		_, _ = rkCronRunner("cron", "mute", row.ID, "--off")
		return
	}
	_, _ = rkCronRunner("cron", "mute", row.ID)
}

// muteOperatorClockIfUntracked is the tick-start reconcile (the loud drift
// direction): a tick arriving on an untracked state means the entry should be
// muted, so the flip's silently-failed rk call is retried here. Level-wise but
// muted-state-aware — when the entry is already muted (indefinite or a live
// lease) the call is skipped, so the reconcile costs a mute only while
// drifted and never disturbs a user's lease. Fail-silent like
// syncOperatorClock.
func muteOperatorClockIfUntracked(data map[string]interface{}) {
	if operatorTracked(data) {
		return
	}
	row, ok := resolveOperatorCronRow()
	if !ok || row.Muted {
		return
	}
	_, _ = rkCronRunner("cron", "mute", row.ID)
}
