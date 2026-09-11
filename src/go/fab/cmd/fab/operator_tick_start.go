package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// operatorStatePathOverride is used in tests to redirect operator state-file
// I/O to a temp file instead of the real server-keyed XDG state path. It holds
// a full file path (not a directory).
var operatorStatePathOverride string

// tickPaneAgentAlive is the real process-tree checker at runtime and a narrow
// seam for tick-entry tests — reached only when a pane row's has_agent is
// null (R11: rk's tri-state short-circuits the walk otherwise).
var tickPaneAgentAlive = paneAgentAlive

func operatorTickStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tick-start",
		Short: "Increment tick_count and record last_tick_at in the server-keyed operator state file",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTickStart,
	}
	cmd.Flags().Bool("diff", false, "also probe the tracked items (pane join + due shell probes): emit deltas/candidates/needs_check/items blocks and update the baselines in the same write")
	cmd.Flags().Bool("quiet", false, "with --diff: on a no-delta tick that is not every 10th, replace the items: block with a fleet_summary: count block")
	return cmd
}

func runOperatorTickStart(cmd *cobra.Command, args []string) error {
	diff, _ := cmd.Flags().GetBool("diff")
	quiet, _ := cmd.Flags().GetBool("quiet")
	// Invalid flag combination fails before any state read/write — a rejected
	// invocation must not consume a tick.
	if quiet && !diff {
		return fmt.Errorf("--quiet requires --diff")
	}
	if diff {
		return runOperatorTickStartDiff(cmd, quiet)
	}

	yamlPath, err := operatorStatePath()
	if err != nil {
		return err
	}

	// Tolerant whole-file read (missing → empty map); unknown top-level keys
	// (and the owned sections, untouched on this flagless path — the --diff
	// path re-marshals tracked) survive the write-back. A legacy-shaped file
	// converts on this read-modify-write like any other verb (R5).
	data, err := loadOperatorState(yamlPath)
	if err != nil {
		return err
	}
	if err := convertLegacyOperatorState(data); err != nil {
		return err
	}

	// Increment tick_count
	tickCount := nextTickCount(data)

	// Capture time once so last_tick_at and stdout are consistent
	now := time.Now()

	data["tick_count"] = tickCount
	data["last_tick_at"] = now.UTC().Format(time.RFC3339)

	// Write back atomically via temp+rename (shared helper).
	if err := saveOperatorState(yamlPath, data); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tick: %d\nnow: %s\n", tickCount, now.Format("15:04"))
	return nil
}

// nextTickCount reads tick_count tolerantly (int/int64/float64) and returns
// its successor — the shared increment both tick-start paths use.
func nextTickCount(data map[string]interface{}) int {
	tickCount := 0
	if v, ok := data["tick_count"]; ok {
		switch n := v.(type) {
		case int:
			tickCount = n
		case int64:
			tickCount = int(n)
		case float64:
			tickCount = int(n)
		}
	}
	return tickCount + 1
}

// --- tick-start --diff -------------------------------------------------------

// tickQuietFullEvery is the built-in periodic full-refresh interval: under
// --quiet, every Nth tick (by post-increment tick_count) emits the full
// document (items:) even with no deltas, so a complete frame still appears
// periodically. Deliberately a constant — not a flag or config knob (matches
// the §5 hardcoded-30m idle auto-default precedent).
const tickQuietFullEvery = 10

// tickTerminusStage is the pipeline terminus — the only stage at which an
// entry with no stop_stage completes. Completion there is a display-state
// check (done/skipped), never bare stage membership: a change entering
// hydrate or ship under /fab-fff is mid-pipeline, not complete. Callers that
// deliberately park earlier (a /fab-ff run stops after hydrate) express that
// through stop_stage.
const tickTerminusStage = "review-pr"

// stageFinished reports whether a display state means the stage is over —
// the shared test for "at the terminus" and "at the stop_stage".
func stageFinished(displayState string) bool {
	return displayState == "done" || displayState == "skipped"
}

// tickFieldChange is one differing declared field of a `changed` delta
// (from/to are arbitrary JSON scalars; either may be null).
type tickFieldChange struct {
	From interface{} `yaml:"from"`
	To   interface{} `yaml:"to"`
}

// tickDelta is one --diff event, keyed by item id. Deltas come in two
// delivery classes (R8): LEVEL-TRIGGERED (done, stale, probe_error,
// pane_death, pane_mismatch, agent_exited — re-emitted every tick until
// acked: `track rm` for done/pane deltas, `track observe` for stale,
// `track update --resume` or `track rm` for probe_error) and
// CONSUMED-ON-READ (changed, stage_advance, review_fail — baseline-diffed
// against last/scope.stage and consumed by the same-write baseline update).
type tickDelta struct {
	Kind     string
	ID       string
	Pane     string                     // pane deltas only
	Fields   map[string]tickFieldChange // changed only
	Then     *string                    // done only (the item's then, verbatim; null when none)
	Age      string                     // stale only
	Error    string                     // probe_error only
	Failures int                        // probe_error only
	Paused   bool                       // probe_error only
	Found    *string                    // pane_mismatch only (null when none resolvable)
	Command  *string                    // agent_exited only
	From     *string                    // stage_advance / review_fail only
	To       *string                    // stage_advance / review_fail only
}

// MarshalYAML emits the pinned key order (kind, id, then the kind-specific
// fields) — a struct with omitempty cannot emit pane_mismatch's `found: null`
// or done's `then: null` (the keys are load-bearing).
func (d tickDelta) MarshalYAML() (interface{}, error) {
	strNode := func(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v} }
	nullNode := func() *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"} }
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	putStr := func(k, v string) { n.Content = append(n.Content, strNode(k), strNode(v)) }
	putNullable := func(k string, v *string) {
		val := nullNode()
		if v != nil {
			val = strNode(*v)
		}
		n.Content = append(n.Content, strNode(k), val)
	}
	putVal := func(k string, v interface{}) {
		vn := nullNode()
		if v != nil {
			vn = &yaml.Node{}
			if err := vn.Encode(v); err != nil {
				vn = nullNode()
			}
		}
		n.Content = append(n.Content, strNode(k), vn)
	}
	putStr("kind", d.Kind)
	putStr("id", d.ID)
	switch d.Kind {
	case "changed":
		putVal("fields", d.Fields)
	case "done":
		putNullable("then", d.Then)
	case "stale":
		putStr("age", d.Age)
	case "probe_error":
		putStr("error", d.Error)
		putVal("failures", d.Failures)
		putVal("paused", d.Paused)
	case "pane_death":
		putStr("pane", d.Pane)
	case "pane_mismatch":
		putStr("pane", d.Pane)
		putNullable("found", d.Found)
	case "agent_exited":
		putStr("pane", d.Pane)
		putNullable("command", d.Command)
	case "stage_advance", "review_fail":
		putStr("pane", d.Pane)
		putNullable("from", d.From)
		putNullable("to", d.To)
	}
	return n, nil
}

// tickCandidate is one `candidates:` row — a pane item whose snapshot
// agent_state is waiting or idle (the §5 sweep population, computed here so
// the skill never fetches the full pane map per tick).
type tickCandidate struct {
	Pane         string  `yaml:"pane"`
	ID           string  `yaml:"id"`
	AgentState   string  `yaml:"agent_state"`   // waiting | idle
	IdleDuration *string `yaml:"idle_duration"` // non-null only for idle (upstream idle-only semantics)
}

// tickNeedsCheck is one `needs_check:` row — a due agent item the LLM must
// check this tick (the binary never runs agent probes).
type tickNeedsCheck struct {
	ID          string `yaml:"id"`
	Kind        string `yaml:"kind"`
	Age         string `yaml:"age"` // since checked_at (added_at when never checked)
	Instruction string `yaml:"instruction"`
}

// tickFleetSummary is the `fleet_summary:` block — five counts a quiet tick
// emits IN PLACE of `items:` (never both keys). tracked counts ALL not-done
// items; the four state counts cover not-done pane items only, so the
// invariant is tracked ≥ waiting + idle + active + unknown.
type tickFleetSummary struct {
	Tracked int `yaml:"tracked"`
	Waiting int `yaml:"waiting"`
	Idle    int `yaml:"idle"`
	Active  int `yaml:"active"`
	Unknown int `yaml:"unknown"`
}

// tickDiffOutput is the --diff stdout document after the tick:/now: lines —
// four YAML blocks in this pinned order, `[]` when empty. Items carries
// pre-built YAML nodes (per-kind field sets pin their own key order); Summary
// is computed during the mutation and marshaled only on the quiet path.
type tickDiffOutput struct {
	Deltas     []tickDelta      `yaml:"deltas"`
	Candidates []tickCandidate  `yaml:"candidates"`
	NeedsCheck []tickNeedsCheck `yaml:"needs_check"`
	Items      []*yaml.Node     `yaml:"items"`
	Summary    tickFleetSummary `yaml:"-"`
}

// tickDiffQuietOutput is the quiet-tick document — deltas/candidates/
// needs_check then fleet_summary in place of items (key order pinned by field
// order).
type tickDiffQuietOutput struct {
	Deltas       []tickDelta      `yaml:"deltas"`
	Candidates   []tickCandidate  `yaml:"candidates"`
	NeedsCheck   []tickNeedsCheck `yaml:"needs_check"`
	FleetSummary tickFleetSummary `yaml:"fleet_summary"`
}

func runOperatorTickStartDiff(cmd *cobra.Command, quiet bool) error {
	// Capture time once so last_tick_at, the baselines, and stdout are
	// consistent.
	now := time.Now()
	nowStr := now.UTC().Format(time.RFC3339)
	tickCount := 0
	out := tickDiffOutput{
		Deltas:     []tickDelta{},
		Candidates: []tickCandidate{},
		NeedsCheck: []tickNeedsCheck{},
		Items:      []*yaml.Node{},
	}

	// ONE mutation: tick bookkeeping, the pane-item baseline update, and the
	// probe bookkeeping (last/checked_at/unchanged/failures/paused) land in
	// the same load → typed edit → atomic save. The edge-triggered clock sync
	// is disabled here: the reconciles below are the level-wise form, so one
	// tick never issues two mutes.
	var postState map[string]interface{}
	err := mutateOperatorStateClock(func(data map[string]interface{}) error {
		postState = data
		tickCount = nextTickCount(data)
		data["tick_count"] = tickCount
		data["last_tick_at"] = nowStr

		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			// Empty tracked list: every block is provably empty — skip the
			// pane-snapshot subprocess entirely. The no-op tick is
			// first-class: tick bookkeeping (and any legacy conversion)
			// still lands.
			return nil
		}

		var rows []paneRow
		if anyPaneItems(items) {
			rows, err = tickSnapshotRows()
			if err != nil {
				return fmt.Errorf("tick --diff snapshot: %w", err)
			}
		}
		joined, doneNow := diffPaneItems(items, rows, &out, now, nowStr)
		runDueProbes(items, &out, now)
		deriveItemDeltas(items, doneNow, &out, now)
		out.Items = buildItemRows(items, rows, joined, doneNow, now)
		out.Summary = summarizeItems(items, rows, joined, doneNow)
		data["tracked"] = items
		return nil
	}, false)
	if err != nil {
		return err
	}

	// Loud-direction reconcile: a tick arriving on an untracked state is the
	// drift signal — the entry should be muted (a flip's rk call may have
	// failed silently). Skipped entirely on a tracked state, and
	// muteOperatorClockIfUntracked itself skips an already-muted entry. The
	// schedule reconcile (B3) runs at the end of every tick.
	if postState != nil {
		muteOperatorClockIfUntracked(postState)
		reconcileOperatorSchedule(postState)
	}

	return emitTickDiffDoc(cmd.OutOrStdout(), out, quiet, tickCount, now)
}

// emitTickDiffDoc writes the tick:/now: header and the diff document. A quiet
// tick (--quiet, no deltas, empty needs_check, post-increment tickCount not a
// multiple of tickQuietFullEvery) emits fleet_summary: in place of items:;
// every other tick emits the full document. Never both keys.
func emitTickDiffDoc(w io.Writer, out tickDiffOutput, quiet bool, tickCount int, now time.Time) error {
	fmt.Fprintf(w, "tick: %d\nnow: %s\n", tickCount, now.Format("15:04"))
	full := !quiet || len(out.Deltas) > 0 || len(out.NeedsCheck) > 0 || tickCount%tickQuietFullEvery == 0
	var doc []byte
	var err error
	if full {
		doc, err = yaml.Marshal(out)
	} else {
		doc, err = yaml.Marshal(tickDiffQuietOutput{
			Deltas:       out.Deltas,
			Candidates:   out.Candidates,
			NeedsCheck:   out.NeedsCheck,
			FleetSummary: out.Summary,
		})
	}
	if err != nil {
		return fmt.Errorf("cannot marshal tick diff: %w", err)
	}
	_, err = w.Write(doc)
	return err
}

// The shell-name predicate the agent_exited check below rides lives in
// internal/pane (pane.IsShellCommand) — the shared home for the pane
// primitives both the operator and the readiness gate consume.

// resolvedSnap reports whether a snapshot display field carries a real value
// (the em dash is pane_map's unresolved sentinel).
func resolvedSnap(s string) bool {
	return s != "" && s != "—"
}

// stageOrderIndex returns the stage's pipeline position, -1 when unknown.
func stageOrderIndex(stage string) int {
	for i, s := range status.AllStages() {
		if s == stage {
			return i
		}
	}
	return -1
}

// tickCompleted is the fab-change built-in completion predicate — a
// display-state check at a stage, NEVER a stage diff (a change completing at
// its final stage never changes its stage string; only display_state flips).
// stop_stage null: AT the terminus (review-pr) with display_state
// done/skipped. stop_stage set: past the stop in stage order, or AT the stop
// with display_state done/skipped (a finished stop-stage auto-activates the
// next stage, so equality alone would race the transition).
func tickCompleted(stopStage *string, stage, displayState string) bool {
	if stopStage == nil {
		return stage == tickTerminusStage && stageFinished(displayState)
	}
	oi, oStop := stageOrderIndex(stage), stageOrderIndex(*stopStage)
	if oi < 0 || oStop < 0 {
		return false
	}
	if oi > oStop {
		return true
	}
	return oi == oStop && stageFinished(displayState)
}

// anyPaneItems reports whether any item is pane-probed with a non-null
// scope.pane (only then does the snapshot subprocess run).
func anyPaneItems(items []trackedItem) bool {
	for _, it := range items {
		if it.Probe.Mode == probePane && scopeString(it.Scope, "pane") != "" {
			return true
		}
	}
	return false
}

// itemDone reports the item's done verdict for this tick: done_when firing on
// last, or the fab-change built-in firing on the snapshot (doneNow).
func itemDone(it trackedItem, doneNow map[string]bool) bool {
	return trackedItemDone(it) || doneNow[it.ID]
}

// diffPaneItems joins pane-mode items against the snapshot on scope.pane,
// emits the pane deltas and candidates, and applies the scope.stage/agent
// baseline update in place. Returns the cleanly-joined item ids and the ids
// whose fab-change built-in completion predicate fired this tick.
func diffPaneItems(items []trackedItem, rows []paneRow, out *tickDiffOutput, now time.Time, nowStr string) (joined, doneNow map[string]bool) {
	byPane := make(map[string]paneRow, len(rows))
	for _, r := range rows {
		byPane[r.pane] = r
	}
	joined = map[string]bool{}
	doneNow = map[string]bool{}

	// The process-tree agent names load lazily — only a null has_agent row
	// with a shell foreground command needs the walk (R11).
	var agents map[string]bool
	agentAlive := func(paneID string) bool {
		if agents == nil {
			agents = loadAgentBinaryNames()
		}
		return tickPaneAgentAlive(paneID, agents)
	}

	for i := range items {
		it := &items[i]
		if it.Probe.Mode != probePane {
			continue
		}
		paneID := scopeString(it.Scope, "pane")
		if paneID == "" {
			continue // pending — nothing to join against
		}
		// A persisted built-in completion (done_at) is durable: the item is
		// done whatever the pane does now, so it is neither joined nor
		// diffed — a pane that vanished after completing emits done, not
		// pane_death, and the item's then survives until track rm.
		if it.DoneAt != nil && *it.DoneAt != "" {
			doneNow[it.ID] = true
			continue
		}
		row, present := byPane[paneID]

		// pane_death: level-triggered — the item's pane is absent from the
		// snapshot. Baseline untouched; the row re-appears every tick until
		// `track rm` acks it.
		if !present {
			out.Deltas = append(out.Deltas, tickDelta{Kind: "pane_death", ID: it.ID, Pane: paneID})
			continue
		}

		// pane_mismatch: level-triggered — tmux recycles %N pane IDs across
		// server restarts while the socket-keyed state file survives, so a
		// pane now hosting a DIFFERENT change (or none) must never be diffed,
		// baseline-updated, or swept as the old agent.
		if row.changeID != it.ID {
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:  "pane_mismatch",
				ID:    it.ID,
				Pane:  paneID,
				Found: toNullable(row.changeID),
			})
			continue
		}

		// agent_exited: level-triggered — R11's has_agent tri-state first
		// (false ⇒ exited, true ⇒ alive, no process-tree walk), the walk only
		// for null (an uninstrumented pane: a shell foreground triggers the
		// lazy tree confirmation). Baseline remains untouched on exit, and the
		// pane is excluded from candidates so the §5 sweep can never type
		// into a bare shell prompt.
		exited := false
		if row.hasAgent != nil {
			exited = !*row.hasAgent
		} else if pane.IsShellCommand(row.command) {
			exited = !agentAlive(paneID)
		}
		if exited {
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:    "agent_exited",
				ID:      it.ID,
				Pane:    paneID,
				Command: toNullable(row.command),
			})
			continue
		}

		// Clean join. An unresolved snapshot stage (em dash) fabricates
		// nothing: no completion, no stage delta, baseline stage untouched.
		if resolvedSnap(row.stage) {
			var stopStage *string
			if ss := scopeString(it.Scope, "stop_stage"); ss != "" {
				stopStage = &ss
			}
			if tickCompleted(stopStage, row.stage, row.displayState) {
				doneNow[it.ID] = true
				// Persist the verdict in the same atomic write so the
				// level-triggered done survives the pane disappearing
				// before the operator acks it.
				if it.DoneAt == nil {
					doneAt := nowStr
					it.DoneAt = &doneAt
					it.UpdatedAt = nowStr
				}
			}
			// Consumed-on-read deltas, diffed against the stored baseline and
			// consumed by the baseline update below. review→apply is the
			// rework reset path — review_fail wins over stage_advance for
			// that transition. No baseline stage (added without --stage)
			// diffs against nothing.
			baseline := scopeString(it.Scope, "stage")
			if baseline != "" && row.stage != baseline {
				from, to := baseline, row.stage
				kind := "stage_advance"
				if from == "review" && to == "apply" {
					kind = "review_fail"
				}
				out.Deltas = append(out.Deltas, tickDelta{Kind: kind, ID: it.ID, Pane: paneID, From: &from, To: &to})
				it.Scope["stage"] = to
				it.UpdatedAt = nowStr
			}
		}
		// agent ← snapshot agent state, verbatim (null for unknown).
		it.Scope["agent"] = nil
		if row.agentState != "" {
			it.Scope["agent"] = row.agentState
		}
		joined[it.ID] = true

		// candidates: the §5 sweep population — waiting first, then idle.
		// Unknown/active (and mismatched/exited, above) are excluded; on
		// rk-less servers every pane reads unknown and the list stays empty.
		if row.agentState == "waiting" || row.agentState == "idle" {
			cand := tickCandidate{Pane: paneID, ID: it.ID, AgentState: row.agentState}
			if row.agentState == "idle" && row.agentIdleDur != "" {
				dur := row.agentIdleDur
				cand.IdleDuration = &dur
			}
			out.Candidates = append(out.Candidates, cand)
		}
	}

	// Candidate order: waiting class first, item id within each class.
	sort.SliceStable(out.Candidates, func(i, j int) bool {
		a, b := out.Candidates[i], out.Candidates[j]
		if a.AgentState != b.AgentState {
			return a.AgentState == "waiting"
		}
		return a.ID < b.ID
	})
	return joined, doneNow
}

// --- shell probe runner (R7) ---------------------------------------------------

// probeRunner is the injectable seam for shell probe execution (the
// rkCronRunner precedent): argv-only (never a shell string), the operator's
// environment inherited, bounded by trackProbeTimeout. The returned error
// carries the composed one-line message ("exit 1: <stderr first line>",
// "timeout after 10s", …). Tests stub it to serve fixture JSON without a
// live gh.
var probeRunner = func(argv []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("empty probe argv")
	}
	if _, err := exec.LookPath(argv[0]); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), trackProbeTimeout)
	defer cancel()
	out, stderr, err := pane.RunCmdContext(ctx, argv[0], argv[1:]...)
	if err == nil {
		return out, nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout after %s", trackProbeTimeout)
	}
	msg := err.Error()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		msg = fmt.Sprintf("exit %d", ee.ExitCode())
	}
	if first, _, _ := strings.Cut(strings.TrimSpace(string(stderr)), "\n"); first != "" {
		msg += ": " + first
	}
	return "", errors.New(msg)
}

// probeDue applies the cadence rule: a null checked_at is always due;
// otherwise due when now - checked_at ≥ check_every.
func probeDue(it trackedItem, now time.Time) bool {
	if it.CheckedAt == nil || *it.CheckedAt == "" {
		return true
	}
	ce, ok := checkEveryDuration(it)
	if !ok {
		ce = trackCheckEveryDefault
	}
	t, err := time.Parse(time.RFC3339, *it.CheckedAt)
	if err != nil {
		return true
	}
	return !now.Before(t.Add(ce))
}

// runDueProbes runs every due shell probe (not done, not paused, past its
// cadence) sequentially under the trackTickBudget wall clock.
func runDueProbes(items []trackedItem, out *tickDiffOutput, now time.Time) {
	runDueProbesUntil(items, out, now, now.Add(trackTickBudget))
}

// runDueProbesUntil is runDueProbes with an explicit deadline (the budget is
// the named constant in production; tests pass a blown deadline). Items the
// budget cuts off emit probe_error "skipped (tick budget)" with failures
// untouched. Done items are never re-probed — the level-triggered done delta
// re-derives from last.
func runDueProbesUntil(items []trackedItem, out *tickDiffOutput, now, deadline time.Time) {
	for i := range items {
		it := &items[i]
		if it.Probe.Mode != probeShell || it.Paused || trackedItemDone(*it) {
			continue
		}
		if !probeDue(*it, now) {
			continue
		}
		if time.Now().After(deadline) {
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:     "probe_error",
				ID:       it.ID,
				Error:    "skipped (tick budget)",
				Failures: it.Failures,
				Paused:   it.Paused,
			})
			continue
		}
		runOneProbe(it, out, now)
	}
}

// runOneProbe executes one shell probe and applies the R7 bookkeeping: a
// failure increments failures (the third sets paused); a success resets
// failures, extracts only the declared fields into last, moves unchanged
// (0 on a field delta, +1 otherwise), and emits the consumed-on-read changed
// delta. checked_at moves on every completed probe, success or failure.
func runOneProbe(it *trackedItem, out *tickDiffOutput, now time.Time) {
	nowStr := now.UTC().Format(time.RFC3339)
	it.CheckedAt = &nowStr
	stdout, err := probeRunner(it.Probe.Argv)
	if err != nil {
		it.Failures++
		if it.Failures >= trackFailurePauseCap {
			it.Paused = true
		}
		out.Deltas = append(out.Deltas, tickDelta{
			Kind:     "probe_error",
			ID:       it.ID,
			Error:    err.Error(),
			Failures: it.Failures,
			Paused:   it.Paused,
		})
		return
	}
	var obj map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(stdout), &obj); jsonErr != nil || obj == nil {
		it.Failures++
		if it.Failures >= trackFailurePauseCap {
			it.Paused = true
		}
		out.Deltas = append(out.Deltas, tickDelta{
			Kind:     "probe_error",
			ID:       it.ID,
			Error:    "stdout is not a JSON object",
			Failures: it.Failures,
			Paused:   it.Paused,
		})
		return
	}
	// A successful JSON-object probe resets the consecutive-failure count
	// before the unchanged/delta branch — the pause cap counts CONSECUTIVE
	// failures, so a success in between must not let stale failures add up.
	it.Failures = 0
	newLast := obj
	if len(it.Probe.Fields) > 0 {
		newLast = extractProbeFields(obj, it.Probe.Fields)
	}
	normalizeJSONNumbers(newLast)
	if reflect.DeepEqual(newLast, it.Last) {
		it.Unchanged++
		return
	}
	out.Deltas = append(out.Deltas, tickDelta{
		Kind:   "changed",
		ID:     it.ID,
		Fields: fieldChanges(it.Last, newLast, it.Probe.Fields),
	})
	it.Unchanged = 0
	it.Last = newLast
	it.UpdatedAt = nowStr
}

// fieldChanges computes the changed delta's per-field from/to entries over
// the declared fields (top-level or dotted). An absent side reports null.
func fieldChanges(oldLast, newLast map[string]interface{}, fields []string) map[string]tickFieldChange {
	if len(fields) == 0 {
		for k := range newLast {
			fields = append(fields, k)
		}
		for k := range oldLast {
			if _, ok := newLast[k]; !ok {
				fields = append(fields, k)
			}
		}
		sort.Strings(fields)
	}
	changes := map[string]tickFieldChange{}
	for _, f := range fields {
		segs := strings.Split(f, ".")
		from := getPathValue(oldLast, segs)
		to := getPathValue(newLast, segs)
		if !reflect.DeepEqual(from, to) {
			changes[f] = tickFieldChange{From: from, To: to}
		}
	}
	return changes
}

// getPathValue walks a dotted path into a nested object (nil when absent).
func getPathValue(obj map[string]interface{}, segs []string) interface{} {
	var cur interface{} = obj
	for _, s := range segs {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur, ok = m[s]
		if !ok {
			return nil
		}
	}
	return cur
}

// --- derived deltas (R8/R10) ---------------------------------------------------

// deriveItemDeltas emits the level-triggered item deltas that need no
// snapshot: done (every tick until track rm), stale, the paused probe_error
// re-emission, the missing-dependency probe_error — and lists due agent items
// under needs_check. List order is emission order.
func deriveItemDeltas(items []trackedItem, doneNow map[string]bool, out *tickDiffOutput, now time.Time) {
	for i := range items {
		it := &items[i]
		// R10: a depends_on id missing from the list is a probe_error-class
		// warning naming the missing id.
		for _, dep := range it.DependsOn {
			found := false
			for _, e := range items {
				if e.ID == dep {
					found = true
					break
				}
			}
			if !found {
				out.Deltas = append(out.Deltas, tickDelta{
					Kind:     "probe_error",
					ID:       it.ID,
					Error:    "unknown dependency " + dep,
					Failures: it.Failures,
					Paused:   it.Paused,
				})
				break
			}
		}
		done := itemDone(*it, doneNow)
		if done {
			out.Deltas = append(out.Deltas, tickDelta{Kind: "done", ID: it.ID, Then: it.Then})
			continue // done items are inert: no stale, no needs_check
		}
		// probe_error is level-triggered: while the item sits paused on the
		// failure cap it re-emits until `track update --resume` / `track rm`.
		if it.Paused && it.Failures >= trackFailurePauseCap {
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:     "probe_error",
				ID:       it.ID,
				Error:    fmt.Sprintf("paused after %d consecutive probe failures", it.Failures),
				Failures: it.Failures,
				Paused:   true,
			})
			continue // paused items are neither stale nor due
		}
		if it.Paused {
			continue
		}
		if it.Probe.Mode != probeAgent {
			continue
		}
		if trackItemStale(*it, now) {
			out.Deltas = append(out.Deltas, tickDelta{Kind: "stale", ID: it.ID, Age: trackItemAge(*it, now)})
		}
		if probeDue(*it, now) {
			out.NeedsCheck = append(out.NeedsCheck, tickNeedsCheck{
				ID:          it.ID,
				Kind:        it.Kind,
				Age:         trackItemAge(*it, now),
				Instruction: it.Probe.Instruction,
			})
		}
	}
}

// trackItemAge renders the age since checked_at (added_at when never
// checked) — the stale/needs_check age cell.
func trackItemAge(it trackedItem, now time.Time) string {
	base := it.AddedAt
	if it.CheckedAt != nil && *it.CheckedAt != "" {
		base = *it.CheckedAt
	}
	t, err := time.Parse(time.RFC3339, base)
	if err != nil {
		return "0s"
	}
	return formatTrackAge(now.Sub(t))
}

// --- items rows and the quiet summary (R9/R10) ----------------------------------

// tickHeldDep is trackHeldDep with this tick's done verdicts (a fab-change
// dep's built-in completion is snapshot-derived — done_when alone never sees
// it, so a chain would stick held after its dep completed).
func tickHeldDep(items []trackedItem, it trackedItem, doneNow map[string]bool) string {
	for _, dep := range it.DependsOn {
		satisfied := false
		for _, e := range items {
			if e.ID == dep && itemDone(e, doneNow) {
				satisfied = true
				break
			}
		}
		if !satisfied {
			return dep
		}
	}
	return ""
}

// tickItemState derives the item's lifecycle state (R10) with this tick's
// snapshot verdicts (joined = clean pane join, doneNow = built-in fired).
func tickItemState(items []trackedItem, it trackedItem, joined, doneNow map[string]bool, now time.Time) string {
	switch {
	case itemDone(it, doneNow):
		return "done"
	case it.Paused:
		return "paused"
	}
	if tickHeldDep(items, it, doneNow) != "" {
		return "held"
	}
	if it.Kind == kindFabChange && scopeString(it.Scope, "pane") == "" {
		return "pending"
	}
	if trackItemStale(it, now) {
		return "stale"
	}
	if it.Probe.Mode == probePane {
		if joined[it.ID] {
			return "live"
		}
		// pane dead/mismatched/exited — the level-triggered delta carries it;
		// the item is still being watched.
		return "watching"
	}
	return "watching"
}

// tickItemNext renders the next column: the item's then, "spawn" for a
// pending fab-change, "held: <dep-id>" for a held item; null otherwise.
func tickItemNext(items []trackedItem, it trackedItem, state string, doneNow map[string]bool) interface{} {
	if it.Then != nil && *it.Then != "" {
		return *it.Then
	}
	switch state {
	case "pending":
		return "spawn"
	case "held":
		return "held: " + tickHeldDep(items, it, doneNow)
	}
	return nil
}

// tickKindOrder pins the items ordering (kind → scope.repo → id).
func tickKindOrder(kind string) int {
	switch kind {
	case kindFabChange:
		return 0
	case kindGitHubPR:
		return 1
	case kindShell:
		return 2
	case kindTask:
		return 3
	case kindLinear:
		return 4
	case kindSlack:
		return 5
	case kindNote:
		return 6
	}
	return 7
}

// omap builds an ordered YAML mapping node (items rows pin their key order).
type omap struct {
	node *yaml.Node
}

func newOmap() *omap {
	return &omap{node: &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}}
}

func (o *omap) put(key string, val interface{}) *omap {
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	if val != nil {
		vn = &yaml.Node{}
		if err := vn.Encode(val); err != nil {
			vn = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
		}
	}
	o.node.Content = append(o.node.Content, kn, vn)
	return o
}

func strOrNil(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func strOrNilV(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// buildItemRows renders the items: block — one row per item, ordered kind →
// scope.repo → id, with the per-kind field sets from intake B2.
func buildItemRows(items []trackedItem, rows []paneRow, joined, doneNow map[string]bool, now time.Time) []*yaml.Node {
	byPane := make(map[string]paneRow, len(rows))
	for _, r := range rows {
		byPane[r.pane] = r
	}
	idx := make([]int, len(items))
	for i := range items {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		x, y := items[idx[a]], items[idx[b]]
		if ox, oy := tickKindOrder(x.Kind), tickKindOrder(y.Kind); ox != oy {
			return ox < oy
		}
		if rx, ry := scopeString(x.Scope, "repo"), scopeString(y.Scope, "repo"); rx != ry {
			return rx < ry
		}
		return x.ID < y.ID
	})
	out := make([]*yaml.Node, 0, len(items))
	for _, i := range idx {
		it := items[i]
		state := tickItemState(items, it, joined, doneNow, now)
		next := tickItemNext(items, it, state, doneNow)
		r := newOmap().put("id", it.ID).put("kind", it.Kind).put("state", state)
		switch it.Probe.Mode {
		case probePane:
			putPaneRowFields(r, it, byPane[scopeString(it.Scope, "pane")], joined[it.ID])
		case probeShell, probeAgent:
			r.put("repo", strOrNilV(scopeString(it.Scope, "repo"))).
				put("last", it.Last)
			if it.Probe.Mode == probeAgent {
				seen := it.Seen
				if seen == nil {
					seen = []string{}
				}
				r.put("seen", seen)
			}
			r.put("checked_at", strOrNil(it.CheckedAt)).
				put("check_every", strOrNil(it.CheckEvery)).
				put("unchanged", it.Unchanged)
		default: // none — task / note
			r.put("repo", strOrNilV(scopeString(it.Scope, "repo")))
			if it.Kind == kindNote {
				r.put("text", it.Text).
					put("updated_at", strOrNilV(it.UpdatedAt))
			}
		}
		r.put("next", next)
		out = append(out, r.node)
	}
	return out
}

// putPaneRowFields fills the pane-item row: snapshot fields on a clean join
// (em-dash/unknown sentinels → null; an unresolved snapshot repo falls back
// to the item's scope), baseline identity with null observed fields
// otherwise.
func putPaneRowFields(r *omap, it trackedItem, row paneRow, isJoined bool) {
	paneID := scopeString(it.Scope, "pane")
	r.put("pane", strOrNilV(paneID))
	if !isJoined {
		r.put("repo", strOrNilV(scopeString(it.Scope, "repo"))).
			put("session", strOrNilV(scopeString(it.Scope, "session"))).
			put("stage", strOrNilV(scopeString(it.Scope, "stage"))).
			put("display_state", nil).
			put("agent_state", nil).
			put("idle_duration", nil).
			put("pr_url", nil).
			put("checked_at", nil)
		return
	}
	repo := row.repo
	if !resolvedSnap(repo) {
		repo = scopeString(it.Scope, "repo")
	}
	agentState, idleDur := agentJSONFields(row.agentState, row.agentIdleDur)
	r.put("repo", strOrNilV(repo)).
		put("session", strOrNilV(row.session)).
		put("stage", strOrNil(rowStage(row))).
		put("display_state", strOrNil(rowDisplay(row))).
		put("agent_state", agentState).
		put("idle_duration", idleDur).
		put("pr_url", strOrNilV(row.prURL)).
		put("checked_at", nil) // pane items: the snapshot is the state (rendered live)
}

func rowStage(r paneRow) *string {
	if !resolvedSnap(r.stage) {
		return nil
	}
	return &r.stage
}

func rowDisplay(r paneRow) *string {
	if !resolvedSnap(r.displayState) {
		return nil
	}
	return &r.displayState
}

// summarizeItems reduces the items to the quiet tick's count block: tracked
// counts all not-done items; the four state counts cover not-done pane items
// (a pane item without a live reading — dead, mismatched, exited, pending —
// counts as unknown), so tracked ≥ waiting + idle + active + unknown.
func summarizeItems(items []trackedItem, rows []paneRow, joined, doneNow map[string]bool) tickFleetSummary {
	byPane := make(map[string]paneRow, len(rows))
	for _, r := range rows {
		byPane[r.pane] = r
	}
	s := tickFleetSummary{}
	for _, it := range items {
		if itemDone(it, doneNow) {
			continue
		}
		s.Tracked++
		if it.Probe.Mode != probePane {
			continue
		}
		if !joined[it.ID] {
			s.Unknown++
			continue
		}
		switch byPane[scopeString(it.Scope, "pane")].agentState {
		case "waiting":
			s.Waiting++
		case "idle":
			s.Idle++
		case "active":
			s.Active++
		default:
			s.Unknown++
		}
	}
	return s
}
