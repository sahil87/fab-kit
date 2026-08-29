package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- tick-start --diff test scaffolding ---------------------------------------

// seedDiffState writes a state file with the given monitored entries (plus a
// tick_count and an unknown top-level key) and redirects state I/O to it.
func seedDiffState(t *testing.T, entries map[string]monitoredEntry) string {
	t.Helper()
	return seedDiffStateAt(t, entries, 5)
}

// seedDiffStateAt is seedDiffState with an explicit starting tick_count (the
// every-10th-tick cases seed 9/19/10).
func seedDiffStateAt(t *testing.T, entries map[string]monitoredEntry, tickCount int) string {
	t.Helper()
	data := map[string]interface{}{
		"tick_count": tickCount,
		"monitored":  entries,
		"custom_key": "preserve-me",
	}
	raw, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("marshal seed state: %v", err)
	}
	return withOperatorState(t, string(raw))
}

// diffEntry builds a monitored entry with the identity fields the diff path
// reads; enrolledAt doubles as last_transition (a fixed past timestamp keeps
// last_transition-change assertions robust against same-second runs).
func diffEntry(pane, repo, session, stage, enrolledAt string) monitoredEntry {
	return monitoredEntry{
		Pane:           pane,
		Repo:           repo,
		Session:        session,
		Stage:          stage,
		Branch:         "260823-" + pane + "-x",
		EnrolledAt:     enrolledAt,
		LastTransition: enrolledAt,
	}
}

// snapRow builds a snapshot row for the tickSnapshotRows stub.
func snapRow(pane, changeID, stage, display, agentState, idleDur string) paneRow {
	return paneRow{
		pane:         pane,
		changeID:     changeID,
		stage:        stage,
		displayState: display,
		agentState:   agentState,
		agentIdleDur: idleDur,
		repo:         "/snap/repo",
		session:      "snap",
	}
}

// stubSnapshot replaces the tickSnapshotRows seam and reports whether the
// snapshot fn was invoked.
func stubSnapshot(t *testing.T, rows []paneRow) *bool {
	t.Helper()
	called := false
	orig := tickSnapshotRows
	tickSnapshotRows = func() ([]paneRow, error) {
		called = true
		return rows, nil
	}
	t.Cleanup(func() { tickSnapshotRows = orig })
	return &called
}

// runTickDiff executes `tick-start --diff` and returns its stdout document.
func runTickDiff(t *testing.T) string {
	t.Helper()
	out, err := runTickDiffArgs(t, "--diff")
	if err != nil {
		t.Fatalf("tick-start --diff: %v", err)
	}
	return out
}

// runTickDiffArgs executes tick-start with the given args and returns stdout;
// the command's error is returned (not fatal) so invalid-flag cases are
// assertable.
func runTickDiffArgs(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

// tickDiffDoc is the parsed --diff stdout (after the tick:/now: header).
// Deltas stay generic maps so key PRESENCE (e.g. `found: null`) is assertable.
// FleetSummary is nil unless the quiet path emitted the block — key presence
// is the contract, asserted on raw stdout, not on parsed emptiness.
type tickDiffDoc struct {
	Deltas       []map[string]interface{} `yaml:"deltas"`
	Candidates   []tickCandidate          `yaml:"candidates"`
	Fleet        []tickFleetRow           `yaml:"fleet"`
	FleetSummary *tickFleetSummary        `yaml:"fleet_summary"`
}

// parseTickDiff splits the tick:/now: header from the YAML blocks and parses
// the blocks.
func parseTickDiff(t *testing.T, out string) tickDiffDoc {
	t.Helper()
	parts := strings.SplitN(out, "\n", 3)
	if len(parts) < 3 || !strings.HasPrefix(parts[0], "tick: ") || !strings.HasPrefix(parts[1], "now: ") {
		t.Fatalf("stdout missing tick/now header: %q", out)
	}
	var doc tickDiffDoc
	if err := yaml.Unmarshal([]byte(parts[2]), &doc); err != nil {
		t.Fatalf("parse diff doc: %v\n---\n%s", err, parts[2])
	}
	return doc
}

// findDelta returns the first delta of the given kind for the change, or nil.
func findDelta(doc tickDiffDoc, kind, change string) map[string]interface{} {
	for _, d := range doc.Deltas {
		if d["kind"] == kind && d["change"] == change {
			return d
		}
	}
	return nil
}

// --- event kinds ---------------------------------------------------------------

func TestOperatorTickDiff_AllEventKinds(t *testing.T) {
	entries := map[string]monitoredEntry{
		// completion: stage string UNCHANGED (review-pr → review-pr), only the
		// display state flipped — the case a stage-diff provably cannot catch.
		"c001": diffEntry("%1", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		"d002": diffEntry("%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane absent → pane_death
		"m003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane hosts another change
		"m004": diffEntry("%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane hosts no change
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // → stage_advance
		"r006": diffEntry("%6", "/r/a", "s1", "review", "2026-01-01T00:00:00Z"), // → review_fail
	}
	seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "c001", "review-pr", "done", "idle", "8m"),
		snapRow("%3", "zz99", "apply", "active", "active", ""),
		snapRow("%4", "", "—", "—", "active", ""),
		snapRow("%5", "a005", "review", "active", "active", ""),
		snapRow("%6", "r006", "apply", "active", "active", ""),
	})

	doc := parseTickDiff(t, runTickDiff(t))

	if len(doc.Deltas) != 6 {
		t.Fatalf("deltas = %v, want 6", doc.Deltas)
	}
	if d := findDelta(doc, "completion", "c001"); d == nil || d["stage"] != "review-pr" || d["display_state"] != "done" {
		t.Errorf("completion delta wrong: %v", d)
	}
	if d := findDelta(doc, "pane_death", "d002"); d == nil {
		t.Errorf("pane_death delta missing: %v", doc.Deltas)
	}
	if d := findDelta(doc, "pane_mismatch", "m003"); d == nil || d["found"] != "zz99" {
		t.Errorf("pane_mismatch (occupied) delta wrong: %v", d)
	}
	if d := findDelta(doc, "pane_mismatch", "m004"); d == nil {
		t.Errorf("pane_mismatch (unoccupied) delta missing: %v", doc.Deltas)
	} else if v, present := d["found"]; !present || v != nil {
		t.Errorf("pane_mismatch found = %v (present %v), want key present with null", v, present)
	}
	if d := findDelta(doc, "stage_advance", "a005"); d == nil || d["from"] != "apply" || d["to"] != "review" {
		t.Errorf("stage_advance delta wrong: %v", d)
	}
	// review→apply is the rework reset path: review_fail wins over stage_advance.
	if d := findDelta(doc, "review_fail", "r006"); d == nil || d["from"] != "review" || d["to"] != "apply" {
		t.Errorf("review_fail delta wrong: %v", d)
	}
	if d := findDelta(doc, "stage_advance", "r006"); d != nil {
		t.Errorf("review_fail transition also emitted stage_advance: %v", d)
	}
}

func TestOperatorTickDiff_LevelTriggeredReEmitUntilRemove(t *testing.T) {
	entries := map[string]monitoredEntry{
		"c001": diffEntry("%1", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		"d002": diffEntry("%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"m003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "c001", "review-pr", "done", "idle", "8m"),
		snapRow("%3", "zz99", "apply", "active", "active", ""),
	})

	for run := 1; run <= 2; run++ {
		doc := parseTickDiff(t, runTickDiff(t))
		for _, want := range [][2]string{{"completion", "c001"}, {"pane_death", "d002"}, {"pane_mismatch", "m003"}} {
			if findDelta(doc, want[0], want[1]) == nil {
				t.Errorf("run %d: %s/%s delta missing (level-triggered must re-emit): %v", run, want[0], want[1], doc.Deltas)
			}
		}
	}

	// `fab operator remove` is the ack — the entry disappears, the event stops.
	for _, id := range []string{"c001", "d002", "m003"} {
		if err := runOperatorCmd(t, operatorRemoveCmd(), id); err != nil {
			t.Fatalf("remove %s: %v", id, err)
		}
	}
	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Deltas) != 0 {
		t.Errorf("after remove, deltas = %v, want none", doc.Deltas)
	}
	if m := readMonitored(t, path); len(m) != 0 {
		t.Errorf("monitored = %v, want empty after removes", m)
	}
}

func TestOperatorTickDiff_StageAdvanceConsumedOnRead(t *testing.T) {
	entries := map[string]monitoredEntry{
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{snapRow("%5", "a005", "review", "active", "active", "")})

	doc := parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "stage_advance", "a005"); d == nil {
		t.Fatalf("run 1: stage_advance missing: %v", doc.Deltas)
	}
	if got := readMonitored(t, path)["a005"].Stage; got != "review" {
		t.Errorf("baseline stage = %q after run 1, want review", got)
	}

	doc = parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "stage_advance", "a005"); d != nil {
		t.Errorf("run 2: stage_advance re-emitted (consumed-on-read must not): %v", d)
	}
}

func TestOperatorTickDiff_CompletionPredicateBranches(t *testing.T) {
	stopIntake := "intake"
	stopReview := "review"
	entries := map[string]monitoredEntry{
		// stop_stage set, snapshot PAST the stop in stage order → complete.
		"s001": {Pane: "%1", Repo: "/r/a", Session: "s1", Stage: "intake", StopStage: &stopIntake, EnrolledAt: "2026-01-01T00:00:00Z", LastTransition: "2026-01-01T00:00:00Z"},
		// stop_stage set, AT the stop with done → complete.
		"s002": {Pane: "%2", Repo: "/r/a", Session: "s1", Stage: "review", StopStage: &stopReview, EnrolledAt: "2026-01-01T00:00:00Z", LastTransition: "2026-01-01T00:00:00Z"},
		// stop_stage set, AT the stop but still active → NOT complete.
		"s003": {Pane: "%3", Repo: "/r/a", Session: "s1", Stage: "review", StopStage: &stopReview, EnrolledAt: "2026-01-01T00:00:00Z", LastTransition: "2026-01-01T00:00:00Z"},
		// stop_stage set, past the stop → complete (even mid-stage).
		"s004": {Pane: "%4", Repo: "/r/a", Session: "s1", Stage: "review", StopStage: &stopReview, EnrolledAt: "2026-01-01T00:00:00Z", LastTransition: "2026-01-01T00:00:00Z"},
		// stop_stage null, ship still running → NOT complete (mid-pipeline under
		// /fab-fff; the regression guard for the spurious hydrate/ship completions).
		"s005": diffEntry("%5", "/r/a", "s1", "ship", "2026-01-01T00:00:00Z"),
		// stop_stage null, non-terminal → NOT complete.
		"s006": diffEntry("%6", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		// stop_stage null, hydrate running → NOT complete.
		"s007": diffEntry("%7", "/r/a", "s1", "hydrate", "2026-01-01T00:00:00Z"),
		// stop_stage null, hydrate done but pipeline continues (the transient
		// finish→ship-start window, or a parked /fab-ff run) → NOT complete.
		"s008": diffEntry("%8", "/r/a", "s1", "hydrate", "2026-01-01T00:00:00Z"),
		// stop_stage null, AT the terminus but still active (awaiting a PR review) → NOT complete.
		"s009": diffEntry("%9", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		// stop_stage null, terminus done → complete.
		"s010": diffEntry("%10", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		// stop_stage null, terminus skipped (review-pr disabled at ship) → complete.
		"s011": diffEntry("%11", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "s001", "apply", "ready", "active", ""),
		snapRow("%2", "s002", "review", "done", "idle", "3m"),
		snapRow("%3", "s003", "review", "active", "active", ""),
		snapRow("%4", "s004", "hydrate", "active", "active", ""),
		snapRow("%5", "s005", "ship", "active", "active", ""),
		snapRow("%6", "s006", "apply", "active", "active", ""),
		snapRow("%7", "s007", "hydrate", "active", "active", ""),
		snapRow("%8", "s008", "hydrate", "done", "idle", "2m"),
		snapRow("%9", "s009", "review-pr", "active", "active", ""),
		snapRow("%10", "s010", "review-pr", "done", "idle", "5m"),
		snapRow("%11", "s011", "review-pr", "skipped", "idle", "5m"),
	})

	doc := parseTickDiff(t, runTickDiff(t))
	for _, id := range []string{"s001", "s002", "s004", "s010", "s011"} {
		if findDelta(doc, "completion", id) == nil {
			t.Errorf("completion for %s missing: %v", id, doc.Deltas)
		}
	}
	for _, id := range []string{"s003", "s005", "s006", "s007", "s008", "s009"} {
		if d := findDelta(doc, "completion", id); d != nil {
			t.Errorf("completion for %s emitted, want none: %v", id, d)
		}
	}
}

// --- baseline write ------------------------------------------------------------

func TestOperatorTickDiff_BaselineUpdateSameWrite(t *testing.T) {
	entries := map[string]monitoredEntry{
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"b006": diffEntry("%6", "/r/a", "s1", "review", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%5", "a005", "review", "active", "waiting", ""),
		snapRow("%6", "b006", "review", "active", "active", ""),
	})

	runTickDiff(t)

	state := readStateFile(t, path)
	if state["tick_count"] != 6 {
		t.Errorf("tick_count = %v, want 6", state["tick_count"])
	}
	if state["custom_key"] != "preserve-me" {
		t.Errorf("unknown top-level key lost: %v", state)
	}
	if ts, _ := state["last_tick_at"].(string); ts == "" {
		t.Error("last_tick_at missing")
	}

	m := readMonitored(t, path)
	// Stage changed → baseline advanced and last_transition refreshed to the
	// tick's single captured timestamp (consistent with last_tick_at).
	if e := m["a005"]; e.Stage != "review" || e.Agent != "waiting" {
		t.Errorf("a005 = %+v, want stage review / agent waiting", e)
	} else if e.LastTransition != state["last_tick_at"] {
		t.Errorf("a005 last_transition = %q, want last_tick_at %v (single captured tick timestamp)", e.LastTransition, state["last_tick_at"])
	}
	// Stage unchanged → last_transition preserved, agent still updated.
	if e := m["b006"]; e.Stage != "review" || e.Agent != "active" {
		t.Errorf("b006 = %+v, want stage review / agent active", e)
	} else if e.LastTransition != "2026-01-01T00:00:00Z" {
		t.Errorf("b006 last_transition = %q, want untouched (stage unchanged)", e.LastTransition)
	}
}

func TestOperatorTickDiff_UnresolvedStageFabricatesNothing(t *testing.T) {
	entries := map[string]monitoredEntry{
		"u007": diffEntry("%7", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{snapRow("%7", "u007", "—", "—", "waiting", "")})

	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Deltas) != 0 {
		t.Errorf("deltas = %v, want none for an unresolved snapshot stage", doc.Deltas)
	}
	if e := readMonitored(t, path)["u007"]; e.Stage != "apply" {
		t.Errorf("baseline stage = %q, want untouched (apply)", e.Stage)
	} else if e.Agent != "waiting" {
		t.Errorf("baseline agent = %q, want snapshot value (waiting)", e.Agent)
	}
	// The mismatched/dead entries' baselines stay untouched — covered by the
	// mismatch case here: no baseline write despite a live row.
}

func TestOperatorTickDiff_MismatchedPaneBaselineUntouched(t *testing.T) {
	entries := map[string]monitoredEntry{
		"m003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{snapRow("%3", "zz99", "review", "active", "waiting", "")})

	runTickDiff(t)
	if e := readMonitored(t, path)["m003"]; e.Stage != "apply" || e.Agent != "" {
		t.Errorf("mismatched entry baseline = %+v, want untouched (stage apply, agent empty)", e)
	}
}

// --- candidates / fleet --------------------------------------------------------

func TestOperatorTickDiff_Candidates(t *testing.T) {
	entries := map[string]monitoredEntry{
		"w001": diffEntry("%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"w002": diffEntry("%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"i003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"a004": diffEntry("%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"u005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"m006": diffEntry("%6", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "w001", "apply", "active", "waiting", ""),
		snapRow("%2", "w002", "apply", "active", "waiting", ""),
		snapRow("%3", "i003", "apply", "active", "idle", "8m"),
		snapRow("%4", "a004", "apply", "active", "active", ""),
		snapRow("%5", "u005", "apply", "active", "", ""),        // unknown → excluded
		snapRow("%6", "zz99", "apply", "active", "waiting", ""), // mismatched → excluded
	})

	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Candidates) != 3 {
		t.Fatalf("candidates = %v, want 3 (waiting+idle, monitored-only)", doc.Candidates)
	}
	// waiting first (sorted by change ID), then idle.
	wantOrder := []string{"w001", "w002", "i003"}
	for i, want := range wantOrder {
		if doc.Candidates[i].Change != want {
			t.Errorf("candidates[%d].change = %q, want %q", i, doc.Candidates[i].Change, want)
		}
	}
	if doc.Candidates[0].IdleDuration != nil {
		t.Errorf("waiting candidate idle_duration = %v, want null", *doc.Candidates[0].IdleDuration)
	}
	if d := doc.Candidates[2].IdleDuration; d == nil || *d != "8m" {
		t.Errorf("idle candidate idle_duration = %v, want 8m", d)
	}
}

func TestOperatorTickDiff_Fleet(t *testing.T) {
	entries := map[string]monitoredEntry{
		"f001": diffEntry("%1", "/r/b", "s2", "review-pr", "2026-01-03T00:00:00Z"),
		"f002": diffEntry("%2", "/r/a", "s2", "apply", "2026-01-02T00:00:00Z"),
		"f003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"f004": diffEntry("%4", "/r/a", "s1", "review", "2026-01-04T00:00:00Z"), // pane dead
	}
	seedDiffState(t, entries)
	row1 := snapRow("%1", "f001", "review-pr", "done", "idle", "12m")
	row1.prURL = "https://github.com/acme/foo/pull/412"
	row1.repo = "/r/b"
	row1.session = "s2"
	row2 := snapRow("%2", "f002", "apply", "active", "active", "")
	row2.repo = "/r/a"
	row2.session = "s2"
	row3 := snapRow("%3", "f003", "apply", "active", "waiting", "")
	row3.repo = "/r/a"
	row3.session = "s1"
	stubSnapshot(t, []paneRow{row1, row2, row3})

	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Fleet) != 4 {
		t.Fatalf("fleet = %v, want 4 rows (all monitored entries)", doc.Fleet)
	}
	// Order: repo → session → enrolled_at → change ID.
	wantOrder := []string{"f003", "f004", "f002", "f001"}
	for i, want := range wantOrder {
		if doc.Fleet[i].Change != want {
			t.Errorf("fleet[%d].change = %q, want %q (order %v)", i, doc.Fleet[i].Change, want, wantOrder)
		}
	}
	// Joined row carries snapshot fields incl. pr_url.
	f1 := doc.Fleet[3]
	if f1.Stage == nil || *f1.Stage != "review-pr" || f1.DisplayState == nil || *f1.DisplayState != "done" {
		t.Errorf("f001 joined row wrong: %+v", f1)
	}
	if f1.PRURL == nil || *f1.PRURL != "https://github.com/acme/foo/pull/412" {
		t.Errorf("f001 pr_url = %v", f1.PRURL)
	}
	if f1.AgentState == nil || *f1.AgentState != "idle" || f1.IdleDuration == nil || *f1.IdleDuration != "12m" {
		t.Errorf("f001 agent fields = %v/%v", f1.AgentState, f1.IdleDuration)
	}
	// Dead pane → baseline fallback row: baseline repo/session/stage, nulls elsewhere.
	f4 := doc.Fleet[1]
	if f4.Repo != "/r/a" || f4.Session != "s1" || f4.Stage == nil || *f4.Stage != "review" {
		t.Errorf("f004 fallback identity wrong: %+v", f4)
	}
	if f4.DisplayState != nil || f4.AgentState != nil || f4.IdleDuration != nil || f4.PRURL != nil {
		t.Errorf("f004 fallback observed fields = %v/%v/%v/%v, want all null", f4.DisplayState, f4.AgentState, f4.IdleDuration, f4.PRURL)
	}
}

// --- quiet tick (--diff --quiet) -----------------------------------------------

// assertDocKeys asserts the fleet/fleet_summary key presence contract on raw
// stdout (the contract is "the key is absent", not "the key is empty").
func assertDocKeys(t *testing.T, out string, wantSummary bool) {
	t.Helper()
	hasSummary := strings.Contains(out, "fleet_summary:")
	hasFleet := strings.Contains(out, "fleet:")
	if wantSummary && (!hasSummary || hasFleet) {
		t.Errorf("want fleet_summary present, fleet absent:\n%s", out)
	}
	if !wantSummary && (!hasFleet || hasSummary) {
		t.Errorf("want fleet present, fleet_summary absent:\n%s", out)
	}
}

func TestOperatorTickDiff_QuietNoDeltasEmitsSummary(t *testing.T) {
	entries := map[string]monitoredEntry{
		"w001": diffEntry("%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"a002": diffEntry("%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, entries) // tick 5 → 6: not a multiple of 10
	stubSnapshot(t, []paneRow{
		snapRow("%1", "w001", "apply", "active", "waiting", ""),
		snapRow("%2", "a002", "apply", "active", "active", ""),
	})

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	assertDocKeys(t, out, true)
	for _, block := range []string{"deltas: []", "candidates:"} {
		if !strings.Contains(out, block) {
			t.Errorf("stdout missing %q (candidates are always emitted):\n%s", block, out)
		}
	}
	// Block order is pinned: deltas, candidates, then fleet_summary.
	di, ci, fi := strings.Index(out, "deltas:"), strings.Index(out, "candidates:"), strings.Index(out, "fleet_summary:")
	if !(di >= 0 && di < ci && ci < fi) {
		t.Errorf("block order wrong (want deltas < candidates < fleet_summary):\n%s", out)
	}

	doc := parseTickDiff(t, out)
	if len(doc.Deltas) != 0 {
		t.Errorf("deltas = %v, want none", doc.Deltas)
	}
	if doc.FleetSummary == nil {
		t.Fatalf("parsed doc missing fleet_summary:\n%s", out)
	}
	want := tickFleetSummary{Tracked: 2, Waiting: 1, Idle: 0, Active: 1, Unknown: 0}
	if *doc.FleetSummary != want {
		t.Errorf("fleet_summary = %+v, want %+v", *doc.FleetSummary, want)
	}
}

func TestOperatorTickDiff_QuietWithDeltaEmitsFullFleet(t *testing.T) {
	entries := map[string]monitoredEntry{
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{snapRow("%5", "a005", "review", "active", "active", "")})

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	// A delta (any kind) forces the full document.
	assertDocKeys(t, out, false)
	if doc := parseTickDiff(t, out); findDelta(doc, "stage_advance", "a005") == nil {
		t.Errorf("stage_advance delta missing: %v", doc.Deltas)
	}
}

func TestOperatorTickDiff_QuietEveryTenthTickEmitsFullFleet(t *testing.T) {
	seed := func(t *testing.T, tickCount int) {
		t.Helper()
		seedDiffStateAt(t, map[string]monitoredEntry{
			"w001": diffEntry("%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		}, tickCount)
		stubSnapshot(t, []paneRow{snapRow("%1", "w001", "apply", "active", "waiting", "")})
	}
	for _, tc := range []struct {
		name        string
		seedCount   int
		wantSummary bool
	}{
		{"9 to 10 is full", 9, false},
		{"19 to 20 is full", 19, false},
		{"10 to 11 is quiet", 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seed(t, tc.seedCount)
			out, err := runTickDiffArgs(t, "--diff", "--quiet")
			if err != nil {
				t.Fatalf("tick-start --diff --quiet: %v", err)
			}
			assertDocKeys(t, out, tc.wantSummary)
		})
	}
}

func TestOperatorTickStart_QuietRequiresDiff(t *testing.T) {
	path := seedDiffState(t, map[string]monitoredEntry{
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	})

	out, err := runTickDiffArgs(t, "--quiet")
	if err == nil {
		t.Fatalf("--quiet without --diff succeeded, stdout = %q", out)
	}
	if !strings.Contains(err.Error(), "--quiet requires --diff") {
		t.Errorf("error = %v, want '--quiet requires --diff'", err)
	}
	// The guard fires before any state I/O — the invalid invocation consumes
	// no tick.
	if state := readStateFile(t, path); state["tick_count"] != 5 {
		t.Errorf("tick_count = %v, want 5 (invalid flag combo must not tick)", state["tick_count"])
	}
}

func TestOperatorTickDiff_QuietSummaryCounts(t *testing.T) {
	entries := map[string]monitoredEntry{
		"w001": diffEntry("%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"w002": diffEntry("%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"i003": diffEntry("%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"a004": diffEntry("%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		"u006": diffEntry("%6", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, entries)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "w001", "apply", "active", "waiting", ""),
		snapRow("%2", "w002", "apply", "active", "waiting", ""),
		snapRow("%3", "i003", "apply", "active", "idle", "8m"),
		snapRow("%4", "a004", "apply", "active", "active", ""),
		snapRow("%5", "a005", "apply", "active", "active", ""),
		snapRow("%6", "u006", "apply", "active", "", ""), // null agent_state → unknown
	})

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	assertDocKeys(t, out, true)
	// The five keys are pinned in order: tracked, waiting, idle, active, unknown.
	last := -1
	for _, k := range []string{"tracked:", "waiting:", "idle:", "active:", "unknown:"} {
		i := strings.Index(out, k)
		if i <= last {
			t.Errorf("fleet_summary key %q missing or out of pinned order:\n%s", k, out)
		}
		last = i
	}

	doc := parseTickDiff(t, out)
	want := tickFleetSummary{Tracked: 6, Waiting: 2, Idle: 1, Active: 2, Unknown: 1}
	if doc.FleetSummary == nil || *doc.FleetSummary != want {
		t.Fatalf("fleet_summary = %v, want %+v", doc.FleetSummary, want)
	}
	s := *doc.FleetSummary
	if s.Tracked != s.Waiting+s.Idle+s.Active+s.Unknown {
		t.Errorf("tracked != waiting+idle+active+unknown: %+v", s)
	}
}

func TestOperatorTickDiff_QuietEmptyMonitored(t *testing.T) {
	t.Run("non-10th tick emits all-zero summary", func(t *testing.T) {
		path := withOperatorState(t, "tick_count: 5\nmonitored: {}\ncustom_key: preserve-me\n")
		called := stubSnapshot(t, nil)

		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if *called {
			t.Error("snapshot fn invoked on empty monitored set — must be skipped")
		}
		assertDocKeys(t, out, true)
		for _, block := range []string{"deltas: []", "candidates: []", "tracked: 0", "waiting: 0", "idle: 0", "active: 0", "unknown: 0"} {
			if !strings.Contains(out, block) {
				t.Errorf("stdout missing %q:\n%s", block, out)
			}
		}
		if state := readStateFile(t, path); state["tick_count"] != 6 {
			t.Errorf("tick_count = %v, want 6 (no-op tick still increments)", state["tick_count"])
		}
	})

	t.Run("10th tick emits fleet: []", func(t *testing.T) {
		withOperatorState(t, "tick_count: 9\nmonitored: {}\ncustom_key: preserve-me\n")
		called := stubSnapshot(t, nil)

		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if *called {
			t.Error("snapshot fn invoked on empty monitored set — must be skipped")
		}
		assertDocKeys(t, out, false)
		if !strings.Contains(out, "fleet: []") {
			t.Errorf("10th tick missing fleet: []:\n%s", out)
		}
	})
}

// --- no-op tick + flagless parity ----------------------------------------------

func TestOperatorTickDiff_EmptyMonitoredSkipsSnapshot(t *testing.T) {
	path := withOperatorState(t, "tick_count: 5\nmonitored: {}\ncustom_key: preserve-me\n")
	called := stubSnapshot(t, nil)

	out := runTickDiff(t)

	if *called {
		t.Error("snapshot fn invoked on empty monitored set — must be skipped")
	}
	for _, block := range []string{"deltas: []", "candidates: []", "fleet: []"} {
		if !strings.Contains(out, block) {
			t.Errorf("stdout missing %q:\n%s", block, out)
		}
	}
	if !strings.HasPrefix(out, "tick: 6\nnow: ") {
		t.Errorf("stdout header wrong: %q", out)
	}
	if state := readStateFile(t, path); state["tick_count"] != 6 {
		t.Errorf("tick_count = %v, want 6 (no-op tick still increments)", state["tick_count"])
	}
}

func TestOperatorTickStart_FlaglessByteIdentical(t *testing.T) {
	entries := map[string]monitoredEntry{
		"a005": diffEntry("%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, entries)
	called := stubSnapshot(t, []paneRow{snapRow("%5", "a005", "review", "active", "waiting", "")})

	cmd := operatorTickStartCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("flagless tick-start: %v", err)
	}

	if !regexp.MustCompile(`^tick: 6\nnow: \d\d:\d\d\n$`).MatchString(stdout.String()) {
		t.Errorf("flagless stdout = %q, want exactly 'tick: 6\\nnow: HH:MM\\n'", stdout.String())
	}
	if *called {
		t.Error("flagless path invoked the snapshot seam")
	}
	state := readStateFile(t, path)
	for _, k := range []string{"deltas", "candidates", "fleet"} {
		if _, ok := state[k]; ok {
			t.Errorf("flagless state file gained %q key", k)
		}
	}
	// The snapshot would say review/waiting — the flagless path must not diff.
	if e := readMonitored(t, path)["a005"]; e.Stage != "apply" || e.Agent != "" {
		t.Errorf("flagless path touched the baseline: %+v", e)
	}
}
