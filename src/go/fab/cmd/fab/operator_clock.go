package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// Operator clock sync: the operator-tick cron entry (seeded by `rk operator`)
// carries no suppress guards — run-kit never infers operator intent from
// fab's private state file. Instead the tracked-set verbs TELL the clock:
// when a state mutation flips the tracked predicate, fab mutes the entry
// (tracked→untracked) or clears the mute (untracked→tracked) through rk's own
// verbs, and the schedule is DERIVED from the tracked set (B3) and applied
// through `rk cron edit` only on change. Every rk call is LookPath-gated,
// argv-only, bounded by rkCronTimeout, and fail-silent — absent or older rk
// degrades to the loud-but-safe posture (ticks keep arriving) and never
// changes a verb's exit code or stdout. No -L <server> is passed: the
// operator verbs have no server flag, so rk's own $TMUX derivation is the
// addressing.

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
	// operatorBackoffMin/Max are the derived backoff bounds for a tracked set
	// of only pane/none items (B3).
	operatorBackoffMin = "1m"
	operatorBackoffMax = "30m"
	// operatorDeliverImmediate is the derived deliver policy for pane/none
	// sets; skip-if-busy covers shell/agent sets (and clock overrides).
	operatorDeliverImmediate  = "immediate"
	operatorDeliverSkipIfBusy = "skip-if-busy"
)

// operatorTracked is the clock's tracked predicate (R6): any tracked item
// whose state is not `done` (a list of only done items counts as untracked).
// A section that fails to decode counts as empty — the predicate is a
// fail-silent side effect, never a verb-blocking error.
func operatorTracked(data map[string]interface{}) bool {
	items, err := decodeTrackedItems(data)
	if err != nil {
		return false
	}
	for _, it := range items {
		if !trackedItemDone(it) {
			return true
		}
	}
	return false
}

// rkCronRunner is the injectable seam for every rk cron invocation the clock
// sync makes (the rkPanesRunner / rkOperatorPath precedent): tests stub it to
// serve a fixture `rk cron list --json` document and record mute/edit argv
// without a live rk. Args are the argv after "rk" — never a shell string. The
// real implementation is exec.LookPath-gated and bounded by rkCronTimeout;
// errors (rk absent, non-zero exit, timeout) are the caller's to swallow.
var rkCronRunner = func(args ...string) (string, error) {
	if _, err := exec.LookPath("rk"); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), rkCronTimeout)
	defer cancel()
	out, _, err := pane.RunCmdContext(ctx, "rk", args...)
	return out, err
}

// operatorCronSchedule is the structured `schedule` field of an
// `rk cron list --json` row (r5ao): {kind, min, max} for backoff rows, {kind,
// every} for every/idle-every rows.
type operatorCronSchedule struct {
	Kind  string `json:"kind"`
	Min   string `json:"min"`
	Max   string `json:"max"`
	Every string `json:"every"`
}

// operatorCronRow is the subset of `rk cron list --json` row fields the clock
// sync reads (rk's contract, verified against rk v3.19.46): `muted` is the
// EFFECTIVE state (indefinite mute OR live lease), `muted_until` (unix
// seconds) is present only while a lease is live. Unknown fields are ignored.
type operatorCronRow struct {
	ID         string               `json:"id"`
	Name       string               `json:"name"`
	Target     string               `json:"target"`
	Muted      bool                 `json:"muted"`
	MutedUntil int64                `json:"muted_until"`
	Schedule   operatorCronSchedule `json:"schedule"`
	Deliver    string               `json:"deliver"`
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

// --- derived schedule (B3, R12/R13) -------------------------------------------

// clockOverride is the owned top-level clock_override block written by
// `track clock` (R13): the bounded user override the reconcile applies until
// `until` passes, after which it reverts and is removed from the file.
type clockOverride struct {
	Schedule clockOverrideSchedule `yaml:"schedule" json:"schedule"`
	Deliver  string                `yaml:"deliver" json:"deliver"`
	Until    string                `yaml:"until" json:"until"`
}

// clockOverrideSchedule is the override's schedule ({kind: every|idle-every,
// every: <dur>}).
type clockOverrideSchedule struct {
	Kind  string `yaml:"kind" json:"kind"`
	Every string `yaml:"every" json:"every"`
}

// readClockOverride decodes the clock_override block (nil when absent or
// undecodable — the fail-silent posture).
func readClockOverride(data map[string]interface{}) *clockOverride {
	raw, ok := data["clock_override"]
	if !ok || raw == nil {
		return nil
	}
	o := clockOverride{}
	if err := operatorSection(data, "clock_override", &o); err != nil {
		return nil
	}
	return &o
}

// expireClockOverride deletes a clock_override whose until has passed (or is
// unparseable — it could never apply). It runs pre-save inside
// mutateOperatorStateClock so the removal lands in the verb's own atomic
// write (R13).
func expireClockOverride(data map[string]interface{}, now time.Time) {
	o := readClockOverride(data)
	if o == nil {
		return
	}
	until, err := time.Parse(time.RFC3339, o.Until)
	if err == nil && now.Before(until) {
		return
	}
	delete(data, "clock_override")
}

// operatorSchedule is one derived (or overridden) clock setting.
type operatorSchedule struct {
	kind    string // backoff | every | idle-every
	every   string // every/idle-every duration string ("" for backoff)
	deliver string
}

// deriveOperatorSchedule computes the clock schedule from the tracked set per
// the B3 table: empty (no not-done items) → ok=false (muted — unchanged);
// only pane/none items → backoff 1m→30m, deliver immediate; any shell/agent
// item → idle-every min(check_every) when the operator pane carries an
// agent-state epoch, else every min(check_every) — both deliver skip-if-busy.
// A live clock_override (until in the future) wins over the derived value.
func deriveOperatorSchedule(items []trackedItem, override *clockOverride, epoch bool, now time.Time) (operatorSchedule, bool) {
	if override != nil {
		if until, err := time.Parse(time.RFC3339, override.Until); err == nil && now.Before(until) {
			return operatorSchedule{kind: override.Schedule.Kind, every: override.Schedule.Every, deliver: override.Deliver}, true
		}
	}
	live := []trackedItem{}
	for _, it := range items {
		if !trackedItemDone(it) {
			live = append(live, it)
		}
	}
	if len(live) == 0 {
		return operatorSchedule{}, false
	}
	minEvery := ""
	var minDur time.Duration
	for _, it := range live {
		if it.Probe.Mode != probeShell && it.Probe.Mode != probeAgent {
			continue
		}
		d, ok := checkEveryDuration(it)
		text := trackCheckEveryDefaultText
		if !ok {
			d = trackCheckEveryDefault
		} else {
			text = *it.CheckEvery
		}
		if minEvery == "" || d < minDur {
			minDur, minEvery = d, text
		}
	}
	if minEvery == "" {
		return operatorSchedule{kind: "backoff", deliver: operatorDeliverImmediate}, true
	}
	if epoch {
		return operatorSchedule{kind: "idle-every", every: minEvery, deliver: operatorDeliverSkipIfBusy}, true
	}
	return operatorSchedule{kind: "every", every: minEvery, deliver: operatorDeliverSkipIfBusy}, true
}

// cronRowMatches reports whether the resolved cron row already carries the
// derived schedule and deliver — duration strings compare as durations
// ("2m0s" == "2m").
func cronRowMatches(row operatorCronRow, s operatorSchedule) bool {
	if row.Deliver != s.deliver || row.Schedule.Kind != s.kind {
		return false
	}
	if s.kind == "backoff" {
		return durationsEqual(row.Schedule.Min, operatorBackoffMin) && durationsEqual(row.Schedule.Max, operatorBackoffMax)
	}
	return durationsEqual(row.Schedule.Every, s.every)
}

func durationsEqual(a, b string) bool {
	da, errA := time.ParseDuration(a)
	db, errB := time.ParseDuration(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return da == db
}

// reconcileOperatorSchedule applies the derived schedule to the operator-tick
// cron entry: exactly one `rk cron edit <id> …` when the derived schedule or
// deliver differs from the entry's structured row, none when equal. A muted
// or leased entry is still edited (the schedule takes effect when the lease
// expires). Fail-silent: never changes the verb's exit, stdout, or saved
// state.
func reconcileOperatorSchedule(data map[string]interface{}) {
	items, err := decodeTrackedItems(data)
	if err != nil {
		return
	}
	epoch := false
	if anyProbedItems(items) {
		epoch = operatorPaneEpoch()
	}
	sched, ok := deriveOperatorSchedule(items, readClockOverride(data), epoch, time.Now())
	if !ok {
		return
	}
	row, ok := resolveOperatorCronRow()
	if !ok {
		return
	}
	if cronRowMatches(row, sched) {
		return
	}
	argv := []string{"cron", "edit", row.ID}
	switch sched.kind {
	case "backoff":
		argv = append(argv, "--backoff", "--min", operatorBackoffMin, "--max", operatorBackoffMax)
	case "every":
		argv = append(argv, "--every", sched.every)
	case "idle-every":
		argv = append(argv, "--idle-every", sched.every)
	}
	argv = append(argv, "--deliver", sched.deliver)
	_, _ = rkCronRunner(argv...)
}

// anyProbedItems reports whether any not-done item is shell/agent-probed
// (only then does the epoch question matter).
func anyProbedItems(items []trackedItem) bool {
	for _, it := range items {
		if trackedItemDone(it) {
			continue
		}
		if it.Probe.Mode == probeShell || it.Probe.Mode == probeAgent {
			return true
		}
	}
	return false
}

// operatorWindowRoleRunner is the seam for the operator-window lookup
// (`tmux list-windows` with the @rk_win_role window option) — the
// rkCronRunner posture: LookPath-gated, argv-only, errors swallowed by the
// caller.
var operatorWindowRoleRunner = func() (string, error) {
	if _, err := exec.LookPath("tmux"); err != nil {
		return "", err
	}
	out, _, err := pane.RunCmd("tmux", "list-windows", "-a", "-F", "#{window_id}\t#{@rk_win_role}")
	return out, err
}

// operatorPaneEpoch reports whether the operator pane's `rk mux panes --json`
// row carries a non-null agent_state — the idle-epoch signal choosing
// --idle-every over --every (rk's --idle-every keys on the pane's idle epoch,
// produced by the same agent-state instrumentation). The operator pane is the
// pane whose window carries @rk_win_role=operator, else the current
// $TMUX_PANE. Any failure or no-match → false (no epoch).
func operatorPaneEpoch() bool {
	out, err := rkPanesRunner("")
	if err != nil {
		return false
	}
	var rows []rkPaneRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return false
	}
	winID := ""
	if wins, err := operatorWindowRoleRunner(); err == nil {
		for _, line := range strings.Split(strings.TrimRight(wins, "\n"), "\n") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 && parts[1] == "operator" {
				winID = parts[0]
				break
			}
		}
	}
	self := os.Getenv("TMUX_PANE")
	for _, r := range rows {
		if winID != "" {
			if r.WindowID == winID {
				return r.AgentState != nil
			}
			continue
		}
		if self != "" && r.Pane == self {
			return r.AgentState != nil
		}
	}
	return false
}
