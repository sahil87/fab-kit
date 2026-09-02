package main

import (
	"fmt"
	"io"
	"sort"
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
// seam for tick-entry tests. diffMonitored itself stays pure through its
// injected func(string) bool parameter.
var tickPaneAgentAlive = paneAgentAlive

func operatorTickStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tick-start",
		Short: "Increment tick_count and record last_tick_at in the server-keyed operator state file",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTickStart,
	}
	cmd.Flags().Bool("diff", false, "also diff a pane snapshot against the monitored baseline: emit deltas/candidates/fleet blocks and update the baseline in the same write")
	cmd.Flags().Bool("quiet", false, "with --diff: on a no-delta tick that is not every 10th, replace the fleet: block with a fleet_summary: count block")
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
	// path re-marshals monitored) survive the write-back.
	data, err := loadOperatorState(yamlPath)
	if err != nil {
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
// document (fleet:) even with no deltas, so a complete frame still appears
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

// tickDelta is one --diff event. Deltas come in two delivery classes:
// LEVEL-TRIGGERED (completion, pane_death, pane_mismatch, agent_exited —
// stateless predicates over the current snapshot, re-emitted every tick until
// the entry is removed; the natural ack is `fab operator remove`, so a crash
// between diff and action loses nothing) and CONSUMED-ON-READ (stage_advance,
// review_fail — baseline-diffed and consumed by the same-write baseline
// update; a lost one costs a missed report only).
type tickDelta struct {
	Kind         string  // completion | pane_death | pane_mismatch | agent_exited | stage_advance | review_fail
	Change       string  // monitored change ID
	Pane         string  // the entry's pane
	Stage        *string // completion only
	DisplayState *string // completion only
	Found        *string // pane_mismatch only — the occupying change ID (null when none resolvable)
	Command      *string // agent_exited only — the shell now occupying the pane (e.g. zsh)
	From         *string // stage_advance / review_fail only
	To           *string // stage_advance / review_fail only
}

// MarshalYAML emits the pinned key order (kind, change, pane, then the
// kind-specific fields) — a struct with omitempty cannot emit pane_mismatch's
// `found: null` (the key is load-bearing: null means the pane hosts no
// resolvable change, vs an absent key meaning the field doesn't apply).
func (d tickDelta) MarshalYAML() (interface{}, error) {
	strNode := func(v string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v} }
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	putStr := func(k, v string) { n.Content = append(n.Content, strNode(k), strNode(v)) }
	putNullable := func(k string, v *string) {
		val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
		if v != nil {
			val = strNode(*v)
		}
		n.Content = append(n.Content, strNode(k), val)
	}
	putStr("kind", d.Kind)
	putStr("change", d.Change)
	putStr("pane", d.Pane)
	switch d.Kind {
	case "completion":
		putNullable("stage", d.Stage)
		putNullable("display_state", d.DisplayState)
	case "pane_mismatch":
		putNullable("found", d.Found)
	case "agent_exited":
		putNullable("command", d.Command)
	case "stage_advance", "review_fail":
		putNullable("from", d.From)
		putNullable("to", d.To)
	}
	return n, nil
}

// tickCandidate is one `candidates:` row — a monitored agent whose snapshot
// agent_state is waiting or idle (the §5 sweep population, computed here so
// the skill never fetches the full pane map per tick).
type tickCandidate struct {
	Pane         string  `yaml:"pane"`
	Change       string  `yaml:"change"`
	AgentState   string  `yaml:"agent_state"`   // waiting | idle
	IdleDuration *string `yaml:"idle_duration"` // non-null only for idle (upstream idle-only semantics)
}

// tickFleetRow is one `fleet:` row — the status frame's data source. Joined
// entries carry snapshot fields; dead/mismatched panes fall back to the
// baseline's repo/session/stage with null observed fields.
type tickFleetRow struct {
	Change       string  `yaml:"change"`
	Pane         string  `yaml:"pane"`
	Repo         string  `yaml:"repo"`
	Session      string  `yaml:"session"`
	Stage        *string `yaml:"stage"`
	DisplayState *string `yaml:"display_state"`
	AgentState   *string `yaml:"agent_state"`   // active | waiting | idle | null (unknown)
	IdleDuration *string `yaml:"idle_duration"` // null unless idle
	PRURL        *string `yaml:"pr_url"`        // null when none
}

// tickDiffOutput is the --diff stdout document after the tick:/now: lines —
// three YAML blocks in this pinned order, `[]` when empty.
type tickDiffOutput struct {
	Deltas     []tickDelta     `yaml:"deltas"`
	Candidates []tickCandidate `yaml:"candidates"`
	Fleet      []tickFleetRow  `yaml:"fleet"`
}

// tickFleetSummary is the `fleet_summary:` block — five counts a quiet tick
// emits IN PLACE of `fleet:` (never both keys). Key order is pinned by the
// struct field order. Counts are taken over the same rows fleet: would have
// carried: tracked == waiting + idle + active + unknown on every quiet tick
// (dead/mismatched/exited panes emit level-triggered deltas, which force the
// full document; their baseline rows carry null agent_state, so they would
// count under unknown).
type tickFleetSummary struct {
	Tracked int `yaml:"tracked"` // one per monitored entry
	Waiting int `yaml:"waiting"` // snapshot agent_state waiting
	Idle    int `yaml:"idle"`    // snapshot agent_state idle
	Active  int `yaml:"active"`  // snapshot agent_state active
	Unknown int `yaml:"unknown"` // null/empty/em-dash agent_state
}

// summarizeFleet reduces fleet rows to the quiet tick's count block. Any
// agent_state outside waiting/idle/active (null, or an unresolved sentinel
// like the em dash) counts as unknown.
func summarizeFleet(rows []tickFleetRow) tickFleetSummary {
	s := tickFleetSummary{Tracked: len(rows)}
	for _, r := range rows {
		if r.AgentState == nil {
			s.Unknown++
			continue
		}
		switch *r.AgentState {
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

// tickDiffQuietOutput is the quiet-tick document — deltas/candidates then
// fleet_summary in place of fleet (key order pinned by field order).
type tickDiffQuietOutput struct {
	Deltas       []tickDelta      `yaml:"deltas"`
	Candidates   []tickCandidate  `yaml:"candidates"`
	FleetSummary tickFleetSummary `yaml:"fleet_summary"`
}

func runOperatorTickStartDiff(cmd *cobra.Command, quiet bool) error {
	// Capture time once so last_tick_at, the baseline's last_transition, and
	// stdout are consistent.
	now := time.Now()
	nowStr := now.UTC().Format(time.RFC3339)
	tickCount := 0
	out := tickDiffOutput{Deltas: []tickDelta{}, Candidates: []tickCandidate{}, Fleet: []tickFleetRow{}}

	// ONE mutation: tick bookkeeping AND the monitored-baseline update land in
	// the same load → typed edit → atomic save — --diff is the authoritative
	// baseline writer for the monitored entries' observed fields.
	err := mutateOperatorState(func(data map[string]interface{}) error {
		tickCount = nextTickCount(data)
		data["tick_count"] = tickCount
		data["last_tick_at"] = nowStr

		monitored := map[string]monitoredEntry{}
		if err := operatorSection(data, "monitored", &monitored); err != nil {
			return err
		}
		if len(monitored) == 0 {
			// Empty monitored set: every block is provably empty (all are
			// monitored-scoped) — skip the pane-snapshot subprocess entirely.
			// The no-op tick is first-class: tick bookkeeping still lands.
			return nil
		}

		rows, err := tickSnapshotRows()
		if err != nil {
			return fmt.Errorf("tick --diff snapshot: %w", err)
		}

		agents := loadAgentBinaryNames()
		agentAlive := func(paneID string) bool {
			return tickPaneAgentAlive(paneID, agents)
		}
		diffMonitored(monitored, rows, agentAlive, &out, nowStr)
		data["monitored"] = monitored
		return nil
	})
	if err != nil {
		return err
	}

	return emitTickDiffDoc(cmd.OutOrStdout(), out, quiet, tickCount, now)
}

// emitTickDiffDoc writes the tick:/now: header and the diff document. A quiet
// tick (--quiet, no deltas, post-increment tickCount not a multiple of
// tickQuietFullEvery) emits fleet_summary: in place of fleet:; every other
// tick emits the full document. Never both keys.
func emitTickDiffDoc(w io.Writer, out tickDiffOutput, quiet bool, tickCount int, now time.Time) error {
	fmt.Fprintf(w, "tick: %d\nnow: %s\n", tickCount, now.Format("15:04"))
	var doc []byte
	var err error
	if quiet && len(out.Deltas) == 0 && tickCount%tickQuietFullEvery != 0 {
		doc, err = yaml.Marshal(tickDiffQuietOutput{
			Deltas:       out.Deltas,
			Candidates:   out.Candidates,
			FleetSummary: summarizeFleet(out.Fleet),
		})
	} else {
		doc, err = yaml.Marshal(out)
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

// tickCompleted is the completion predicate — a display-state check at a
// stage, NEVER a stage diff (a change completing at its final stage never
// changes its stage string; only display_state flips). stop_stage null: AT
// the terminus (review-pr) with display_state done/skipped. stop_stage set:
// past the stop in stage order, or AT the stop with display_state
// done/skipped (a finished stop-stage auto-activates the next stage, so
// equality alone would race the transition).
func tickCompleted(entry monitoredEntry, stage, displayState string) bool {
	if entry.StopStage == nil {
		return stage == tickTerminusStage && stageFinished(displayState)
	}
	oi, oStop := stageOrderIndex(stage), stageOrderIndex(*entry.StopStage)
	if oi < 0 || oStop < 0 {
		return false
	}
	if oi > oStop {
		return true
	}
	return oi == oStop && stageFinished(displayState)
}

// baselineFleetRow renders the fleet row for a dead or mismatched pane:
// baseline identity fields, null observed fields.
func baselineFleetRow(id string, e monitoredEntry) tickFleetRow {
	return tickFleetRow{
		Change:  id,
		Pane:    e.Pane,
		Repo:    e.Repo,
		Session: e.Session,
		Stage:   toNullable(e.Stage),
	}
}

// joinedFleetRow renders the fleet row for a cleanly-joined pane from the
// snapshot (em-dash/unknown sentinels → null; an unresolved snapshot repo
// falls back to the enrollment record so the frame never blanks identity).
func joinedFleetRow(id string, e monitoredEntry, r paneRow) tickFleetRow {
	repo := r.repo
	if !resolvedSnap(repo) {
		repo = e.Repo
	}
	agentState, idleDur := agentJSONFields(r.agentState, r.agentIdleDur)
	return tickFleetRow{
		Change:       id,
		Pane:         e.Pane,
		Repo:         repo,
		Session:      r.session,
		Stage:        toNullable(r.stage),
		DisplayState: toNullable(r.displayState),
		AgentState:   agentState,
		IdleDuration: idleDur,
		PRURL:        toNullable(r.prURL),
	}
}

// diffMonitored joins the monitored baseline against the pane snapshot,
// populates out (deltas/candidates/fleet), and applies the baseline update
// in place (the caller writes monitored back in the same mutation). now is
// the tick's single captured timestamp (RFC 3339) so last_transition stays
// consistent with last_tick_at.
func diffMonitored(monitored map[string]monitoredEntry, rows []paneRow, agentAlive func(string) bool, out *tickDiffOutput, now string) {
	byPane := make(map[string]paneRow, len(rows))
	for _, r := range rows {
		byPane[r.pane] = r
	}

	// Deterministic processing order (delta emission order within a class).
	ids := make([]string, 0, len(monitored))
	for id := range monitored {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// fleetRows carries the sort keys (repo/session live on the row; the
	// enrollment timestamp does not) until the final ordering pass.
	type fleetSortRow struct {
		row        tickFleetRow
		enrolledAt string
	}
	fleetRows := make([]fleetSortRow, 0, len(monitored))

	for _, id := range ids {
		entry := monitored[id]
		row, present := byPane[entry.Pane]

		// pane_death: level-triggered — the entry's pane is absent from the
		// snapshot. Baseline untouched; the row re-appears every tick until
		// `fab operator remove` acks it.
		if !present {
			out.Deltas = append(out.Deltas, tickDelta{Kind: "pane_death", Change: id, Pane: entry.Pane})
			fleetRows = append(fleetRows, fleetSortRow{baselineFleetRow(id, entry), entry.EnrolledAt})
			continue
		}

		// pane_mismatch: level-triggered — tmux recycles %N pane IDs across
		// server restarts while the socket-keyed state file survives, so a
		// pane now hosting a DIFFERENT change (or none) must never be diffed,
		// baseline-updated, or swept as the old agent.
		if row.changeID != id {
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:   "pane_mismatch",
				Change: id,
				Pane:   entry.Pane,
				Found:  toNullable(row.changeID),
			})
			fleetRows = append(fleetRows, fleetSortRow{baselineFleetRow(id, entry), entry.EnrolledAt})
			continue
		}

		// agent_exited: level-triggered — a shell foreground triggers a lazy
		// process-tree confirmation. Positive live-agent evidence suppresses
		// the delta and falls through to the clean join; false (including any
		// PID/tree-walk error) preserves the exited behavior. Baseline remains
		// untouched on exit (the snapshot's stale idle must not be trusted),
		// and the pane is excluded from candidates so the §5 sweep can never
		// type into a bare shell prompt.
		if pane.IsShellCommand(row.command) && !agentAlive(entry.Pane) {
			cmd := row.command
			out.Deltas = append(out.Deltas, tickDelta{
				Kind:    "agent_exited",
				Change:  id,
				Pane:    entry.Pane,
				Command: &cmd,
			})
			fleetRows = append(fleetRows, fleetSortRow{baselineFleetRow(id, entry), entry.EnrolledAt})
			continue
		}

		// Clean join. An unresolved snapshot stage (em dash) fabricates
		// nothing: no completion, no stage delta, baseline stage untouched.
		if resolvedSnap(row.stage) {
			if tickCompleted(entry, row.stage, row.displayState) {
				out.Deltas = append(out.Deltas, tickDelta{
					Kind:         "completion",
					Change:       id,
					Pane:         entry.Pane,
					Stage:        toNullable(row.stage),
					DisplayState: toNullable(row.displayState),
				})
			}
			// Consumed-on-read deltas, diffed against the stored baseline and
			// consumed by the baseline update below. review→apply is the
			// rework reset path — review_fail wins over stage_advance for
			// that transition. No baseline stage (enrolled without --stage)
			// diffs against nothing — enrollment emits no synthetic event.
			if entry.Stage != "" && row.stage != entry.Stage {
				from, to := entry.Stage, row.stage
				kind := "stage_advance"
				if from == "review" && to == "apply" {
					kind = "review_fail"
				}
				out.Deltas = append(out.Deltas, tickDelta{Kind: kind, Change: id, Pane: entry.Pane, From: &from, To: &to})
			}
			// Baseline update: stage ← snapshot, last_transition touched iff
			// the stage value changed (fab operator update's semantics).
			if row.stage != entry.Stage {
				entry.Stage = row.stage
				entry.LastTransition = now
			}
		}
		// agent ← snapshot agent state, verbatim (empty for unknown).
		entry.Agent = row.agentState
		monitored[id] = entry

		// candidates: the §5 sweep population — waiting first, then idle.
		// Unknown/active (and mismatched, above) are excluded; on rk-less
		// servers every pane reads unknown and the list stays empty.
		if row.agentState == "waiting" || row.agentState == "idle" {
			cand := tickCandidate{Pane: entry.Pane, Change: id, AgentState: row.agentState}
			if row.agentState == "idle" && row.agentIdleDur != "" {
				dur := row.agentIdleDur
				cand.IdleDuration = &dur
			}
			out.Candidates = append(out.Candidates, cand)
		}

		fleetRows = append(fleetRows, fleetSortRow{joinedFleetRow(id, entry, row), entry.EnrolledAt})
	}

	// Candidate order: waiting class first, change ID within each class.
	sort.SliceStable(out.Candidates, func(i, j int) bool {
		a, b := out.Candidates[i], out.Candidates[j]
		if a.AgentState != b.AgentState {
			return a.AgentState == "waiting"
		}
		return a.Change < b.Change
	})

	// Fleet order: repo → session → enrolled_at → change ID (the frame's
	// rendering order).
	sort.SliceStable(fleetRows, func(i, j int) bool {
		a, b := fleetRows[i], fleetRows[j]
		if a.row.Repo != b.row.Repo {
			return a.row.Repo < b.row.Repo
		}
		if a.row.Session != b.row.Session {
			return a.row.Session < b.row.Session
		}
		if a.enrolledAt != b.enrolledAt {
			return a.enrolledAt < b.enrolledAt
		}
		return a.row.Change < b.row.Change
	})
	for _, fr := range fleetRows {
		out.Fleet = append(out.Fleet, fr.row)
	}
}
