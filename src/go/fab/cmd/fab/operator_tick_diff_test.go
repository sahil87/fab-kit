package main

import (
	"bytes"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// --- tick-start --diff test scaffolding ---------------------------------------

// seedDiffState writes a state file with the given tracked items (plus a
// tick_count and an unknown top-level key) and redirects state I/O to it. The
// clock seams are stubbed quiet so the end-of-tick reconcile never reaches a
// real rk — tests that assert on rk calls re-stub rkCronRunner afterwards
// (the later stub wins; cleanups unwind LIFO).
func seedDiffState(t *testing.T, items []trackedItem) string {
	t.Helper()
	return seedDiffStateAt(t, items, 5)
}

// seedDiffStateAt is seedDiffState with an explicit starting tick_count (the
// every-10th-tick cases seed 9/19/10).
func seedDiffStateAt(t *testing.T, items []trackedItem, tickCount int) string {
	t.Helper()
	stubQuietClock(t)
	data := map[string]interface{}{
		"tick_count": tickCount,
		"tracked":    items,
		"custom_key": "preserve-me",
	}
	raw, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("marshal seed state: %v", err)
	}
	return withOperatorState(t, string(raw))
}

// rfc3339Ago renders now-delta for seed timestamps.
func rfc3339Ago(d time.Duration) string {
	return time.Now().UTC().Add(-d).Format(time.RFC3339)
}

// paneItem builds a fab-change pane item with the identity fields the diff
// path reads; addedAt doubles as updated_at (a fixed past timestamp keeps
// baseline-timestamp assertions robust against same-second runs).
func paneItem(id, paneID, repo, session, stage, addedAt string) trackedItem {
	return trackedItem{
		ID:        id,
		Kind:      kindFabChange,
		Probe:     probeSpec{Mode: probePane},
		DependsOn: []string{},
		Scope: map[string]interface{}{
			"pane": paneID, "repo": repo, "session": session,
			"branch": "260823-" + id + "-x", "stage": stage,
			"agent": nil, "stop_stage": nil, "spawned_by": nil, "merge_mode": nil,
		},
		Last:      map[string]interface{}{},
		AddedAt:   addedAt,
		UpdatedAt: addedAt,
	}
}

// paneItemStop is paneItem with a stop_stage.
func paneItemStop(id, paneID, repo, session, stage, stopStage, addedAt string) trackedItem {
	it := paneItem(id, paneID, repo, session, stage, addedAt)
	it.Scope["stop_stage"] = stopStage
	return it
}

// shellItem builds a shell-probe item; checkedAgo nil = never checked (always
// due). argv is a single probe-<id> token the stubbed runner keys on.
func shellItem(id string, fields []string, doneWhen string, last map[string]interface{}, checkedAt *string) trackedItem {
	var dw *string
	if doneWhen != "" {
		dw = &doneWhen
	}
	ce := "5m"
	return trackedItem{
		ID:         id,
		Kind:       kindShell,
		Probe:      probeSpec{Mode: probeShell, Argv: []string{"probe-" + id}, Fields: fields},
		CheckEvery: &ce,
		DoneWhen:   dw,
		DependsOn:  []string{},
		Scope:      map[string]interface{}{},
		Last:       last,
		CheckedAt:  checkedAt,
		AddedAt:    "2026-01-01T00:00:00Z",
		UpdatedAt:  "2026-01-01T00:00:00Z",
	}
}

// agentItem builds an agent-probe (linear) item with the given cadence and
// checked_at offset.
func agentItem(id, instruction string, checkEvery time.Duration, checkedAgo *time.Duration) trackedItem {
	ce := checkEvery.String()
	var checkedAt *string
	if checkedAgo != nil {
		s := rfc3339Ago(*checkedAgo)
		checkedAt = &s
	}
	return trackedItem{
		ID:         id,
		Kind:       kindLinear,
		Probe:      probeSpec{Mode: probeAgent, Instruction: instruction},
		CheckEvery: &ce,
		DependsOn:  []string{},
		Scope:      map[string]interface{}{"repo": "/r/a"},
		Last:       map[string]interface{}{"new": []interface{}{}},
		CheckedAt:  checkedAt,
		AddedAt:    rfc3339Ago(time.Hour),
		UpdatedAt:  rfc3339Ago(time.Hour),
	}
}

// entryStage / entryAgent read a tracked item's baseline scope fields.
func entryStage(it trackedItem) string { return scopeString(it.Scope, "stage") }
func entryAgent(it trackedItem) string { return scopeString(it.Scope, "agent") }

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

// snapRowCmd is snapRow with the pane's current foreground command set — one
// agent_exited predicate input.
func snapRowCmd(pane, changeID, stage, display, agentState, idleDur, command string) paneRow {
	r := snapRow(pane, changeID, stage, display, agentState, idleDur)
	r.command = command
	return r
}

// snapRowAgent is snapRowCmd with rk's has_agent tri-state set (nil =
// unknown/uninstrumented → the process-tree walk decides).
func snapRowAgent(pane, changeID, stage, display, agentState, idleDur, command string, hasAgent *bool) paneRow {
	r := snapRowCmd(pane, changeID, stage, display, agentState, idleDur, command)
	r.hasAgent = hasAgent
	return r
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

// stubPaneAgentAlive replaces the process-tree liveness seam used at tick entry.
func stubPaneAgentAlive(t *testing.T, fn func(string, map[string]bool) bool) {
	t.Helper()
	orig := tickPaneAgentAlive
	tickPaneAgentAlive = fn
	t.Cleanup(func() { tickPaneAgentAlive = orig })
}

// stubProbes replaces the probeRunner seam: handler serves (stdout, err) per
// probe argv; every call is recorded and returned.
func stubProbes(t *testing.T, handler func(argv []string) (string, error)) *[][]string {
	t.Helper()
	calls := [][]string{}
	orig := probeRunner
	probeRunner = func(argv []string) (string, error) {
		calls = append(calls, append([]string{}, argv...))
		return handler(argv)
	}
	t.Cleanup(func() { probeRunner = orig })
	return &calls
}

// recheckItem rewinds an item's checked_at so the next tick probes it again
// (consecutive ticks in a test run within one cadence window; real ticks are
// at least a minute apart, so a 10-minute rewind always re-dues the item).
func recheckItem(t *testing.T, path, id string) {
	t.Helper()
	ago := rfc3339Ago(10 * time.Minute)
	err := mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		for i := range items {
			if items[i].ID == id {
				items[i].CheckedAt = &ago
			}
		}
		data["tracked"] = items
		return nil
	})
	if err != nil {
		t.Fatalf("recheck %s: %v", id, err)
	}
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
// Deltas/needs_check/items stay generic maps so key PRESENCE (e.g.
// `found: null`, `then: null`) is assertable. FleetSummary is nil unless the
// quiet path emitted the block — key presence is the contract, asserted on
// raw stdout, not on parsed emptiness.
type tickDiffDoc struct {
	Deltas       []map[string]interface{} `yaml:"deltas"`
	Candidates   []tickCandidate          `yaml:"candidates"`
	NeedsCheck   []map[string]interface{} `yaml:"needs_check"`
	Items        []map[string]interface{} `yaml:"items"`
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

// findDelta returns the first delta of the given kind for the item, or nil.
func findDelta(doc tickDiffDoc, kind, id string) map[string]interface{} {
	for _, d := range doc.Deltas {
		if d["kind"] == kind && d["id"] == id {
			return d
		}
	}
	return nil
}

// findItem returns the items: row for the id, or nil.
func findItem(doc tickDiffDoc, id string) map[string]interface{} {
	for _, it := range doc.Items {
		if it["id"] == id {
			return it
		}
	}
	return nil
}

// --- pane event kinds (deltas keyed by id) --------------------------------------

func TestOperatorTickDiff_AllEventKinds(t *testing.T) {
	stubPaneAgentAlive(t, func(string, map[string]bool) bool { return false })
	items := []trackedItem{
		// done: stage string UNCHANGED (review-pr → review-pr), only the
		// display state flipped — the case a stage-diff provably cannot catch.
		paneItem("c001", "%1", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		paneItem("d002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane absent → pane_death
		paneItem("m003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane hosts another change
		paneItem("m004", "%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane hosts no change
		paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // → stage_advance
		paneItem("r006", "%6", "/r/a", "s1", "review", "2026-01-01T00:00:00Z"), // → review_fail
		paneItem("e007", "%7", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),  // pane fell back to a shell
	}
	seedDiffState(t, items)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "c001", "review-pr", "done", "idle", "8m"),
		snapRow("%3", "zz99", "apply", "active", "active", ""),
		snapRow("%4", "", "—", "—", "active", ""),
		snapRow("%5", "a005", "review", "active", "active", ""),
		snapRow("%6", "r006", "apply", "active", "active", ""),
		snapRowCmd("%7", "e007", "apply", "active", "idle", "2m", "zsh"),
	})

	doc := parseTickDiff(t, runTickDiff(t))

	if len(doc.Deltas) != 7 {
		t.Fatalf("deltas = %v, want 7", doc.Deltas)
	}
	d := findDelta(doc, "done", "c001")
	if d == nil {
		t.Fatalf("done delta missing: %v", doc.Deltas)
	}
	if v, present := d["then"]; !present || v != nil {
		t.Errorf("done delta then = %v (present %v), want key present with null", v, present)
	}
	if d := findDelta(doc, "pane_death", "d002"); d == nil || d["pane"] != "%2" {
		t.Errorf("pane_death delta wrong: %v", doc.Deltas)
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
	// The pane is present and change-matched, but hosts a shell → agent_exited.
	if d := findDelta(doc, "agent_exited", "e007"); d == nil || d["command"] != "zsh" {
		t.Errorf("agent_exited delta wrong: %v", d)
	}
}

func TestOperatorTickDiff_AgentExited(t *testing.T) {
	stubPaneAgentAlive(t, func(string, map[string]bool) bool { return false })

	t.Run("shell command emits agent_exited with the command field", func(t *testing.T) {
		seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowCmd("%1", "e001", "apply", "active", "", "", "bash")})

		doc := parseTickDiff(t, runTickDiff(t))
		d := findDelta(doc, "agent_exited", "e001")
		if d == nil || d["command"] != "bash" {
			t.Fatalf("agent_exited delta wrong: %v", d)
		}
	})

	t.Run("non-shell and empty commands emit nothing", func(t *testing.T) {
		calls := 0
		stubPaneAgentAlive(t, func(string, map[string]bool) bool {
			calls++
			return false
		})
		seedDiffState(t, []trackedItem{
			paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
			paneItem("e002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
			paneItem("e003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
			paneItem("e004", "%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		})
		stubSnapshot(t, []paneRow{
			snapRowCmd("%1", "e001", "apply", "active", "active", "", "claude"),
			snapRowCmd("%2", "e002", "apply", "active", "active", "", "kimi"),
			snapRowCmd("%3", "e003", "apply", "active", "active", "", "node"),
			snapRowCmd("%4", "e004", "apply", "active", "active", "", ""), // legacy line, no command
		})

		doc := parseTickDiff(t, runTickDiff(t))
		if len(doc.Deltas) != 0 {
			t.Errorf("deltas = %v, want none for non-shell/empty commands", doc.Deltas)
		}
		if calls != 0 {
			t.Errorf("process-tree checks = %d, want zero for non-shell/empty commands", calls)
		}
	})

	t.Run("shell with live agent follows clean join", func(t *testing.T) {
		var checkedPane string
		stubPaneAgentAlive(t, func(paneID string, agents map[string]bool) bool {
			checkedPane = paneID
			if !agents["claude"] || !agents["codex"] {
				t.Errorf("tick checker received incomplete agent names: %v", agents)
			}
			return true
		})
		path := seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowCmd("%1", "e001", "review", "active", "idle", "5m", "zsh")})

		doc := parseTickDiff(t, runTickDiff(t))
		if checkedPane != "%1" {
			t.Errorf("liveness checked pane %q, want %%1", checkedPane)
		}
		if d := findDelta(doc, "agent_exited", "e001"); d != nil {
			t.Errorf("live agent emitted agent_exited: %v", d)
		}
		if d := findDelta(doc, "stage_advance", "e001"); d == nil || d["from"] != "apply" || d["to"] != "review" {
			t.Errorf("stage_advance after live-agent confirmation wrong: %v", d)
		}
		if len(doc.Candidates) != 1 || doc.Candidates[0].ID != "e001" || doc.Candidates[0].AgentState != "idle" {
			t.Errorf("candidates after live-agent confirmation = %v, want idle e001", doc.Candidates)
		}
		if e := readTracked(t, path)["e001"]; entryStage(e) != "review" || entryAgent(e) != "idle" {
			t.Errorf("live-agent baseline = %+v, want stage review / agent idle", e.Scope)
		}
	})

	t.Run("basename match: a full shell path matches, a lookalike name does not", func(t *testing.T) {
		seedDiffState(t, []trackedItem{
			paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
			paneItem("e002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		})
		stubSnapshot(t, []paneRow{
			snapRowCmd("%1", "e001", "apply", "active", "", "", "/usr/bin/fish"),
			snapRowCmd("%2", "e002", "apply", "active", "active", "", "zshrc-lint"),
		})

		doc := parseTickDiff(t, runTickDiff(t))
		if d := findDelta(doc, "agent_exited", "e001"); d == nil || d["command"] != "/usr/bin/fish" {
			t.Errorf("agent_exited for /usr/bin/fish wrong: %v", d)
		}
		if d := findDelta(doc, "agent_exited", "e002"); d != nil {
			t.Errorf("agent_exited emitted for zshrc-lint (not a shell basename): %v", d)
		}
	})

	t.Run("stale idle agent state yields no candidate and leaves the baseline untouched", func(t *testing.T) {
		path := seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		// The agent exited; the pane option still carries its last reading
		// (idle) — exactly the stale-idle trap the command predicate exists for.
		stubSnapshot(t, []paneRow{snapRowCmd("%1", "e001", "apply", "active", "idle", "5m", "zsh")})

		doc := parseTickDiff(t, runTickDiff(t))
		if len(doc.Candidates) != 0 {
			t.Errorf("candidates = %v, want none (a bare shell must never be swept)", doc.Candidates)
		}
		if e := readTracked(t, path)["e001"]; entryStage(e) != "apply" || entryAgent(e) != "" {
			t.Errorf("exited entry baseline = %+v, want untouched (stage apply, agent empty)", e.Scope)
		}
	})

	t.Run("level-triggered: re-emits every tick until removed", func(t *testing.T) {
		seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowCmd("%1", "e001", "apply", "active", "", "", "zsh")})

		for run := 1; run <= 2; run++ {
			doc := parseTickDiff(t, runTickDiff(t))
			if findDelta(doc, "agent_exited", "e001") == nil {
				t.Errorf("run %d: agent_exited missing (level-triggered must re-emit): %v", run, doc.Deltas)
			}
		}
	})

	t.Run("mismatch wins over agent_exited", func(t *testing.T) {
		calls := 0
		stubPaneAgentAlive(t, func(string, map[string]bool) bool {
			calls++
			return true
		})
		seedDiffState(t, []trackedItem{paneItem("a001", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		// A recycled pane hosting no change AND a shell: exactly one delta,
		// pane_mismatch.
		stubSnapshot(t, []paneRow{snapRowCmd("%3", "", "—", "—", "", "", "zsh")})

		doc := parseTickDiff(t, runTickDiff(t))
		if len(doc.Deltas) != 1 {
			t.Fatalf("deltas = %v, want exactly one", doc.Deltas)
		}
		if findDelta(doc, "pane_mismatch", "a001") == nil {
			t.Errorf("pane_mismatch missing: %v", doc.Deltas)
		}
		if d := findDelta(doc, "agent_exited", "a001"); d != nil {
			t.Errorf("mismatched pane also emitted agent_exited: %v", d)
		}
		if calls != 0 {
			t.Errorf("process-tree checks = %d, want zero because pane_mismatch wins", calls)
		}
	})
}

// TestOperatorTickDiff_HasAgentTriState covers R11: has_agent false ⇒ exited
// (no process-tree walk, even with a live-looking agent_state and a non-shell
// command), true ⇒ alive (no walk, even on a shell foreground), null ⇒ the
// pre-existing walk decides. A-035: a row lacking the key entirely decodes to
// the same nil the walk handles.
func TestOperatorTickDiff_HasAgentTriState(t *testing.T) {
	walks := 0
	stubPaneAgentAlive(t, func(string, map[string]bool) bool { walks++; return false })
	f, tr := false, true

	t.Run("has_agent false emits agent_exited without the walk", func(t *testing.T) {
		seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowAgent("%1", "e001", "apply", "active", "active", "", "claude", &f)})
		doc := parseTickDiff(t, runTickDiff(t))
		if d := findDelta(doc, "agent_exited", "e001"); d == nil || d["command"] != "claude" {
			t.Errorf("agent_exited wrong: %v", d)
		}
		if walks != 0 {
			t.Errorf("walks = %d, want 0 (has_agent false short-circuits)", walks)
		}
	})

	t.Run("has_agent true suppresses agent_exited on a shell foreground without the walk", func(t *testing.T) {
		seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowAgent("%1", "e001", "apply", "active", "active", "", "zsh", &tr)})
		doc := parseTickDiff(t, runTickDiff(t))
		if d := findDelta(doc, "agent_exited", "e001"); d != nil {
			t.Errorf("has_agent true emitted agent_exited: %v", d)
		}
		if walks != 0 {
			t.Errorf("walks = %d, want 0", walks)
		}
	})

	t.Run("has_agent null falls back to the process-tree walk", func(t *testing.T) {
		seedDiffState(t, []trackedItem{paneItem("e001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
		stubSnapshot(t, []paneRow{snapRowAgent("%1", "e001", "apply", "active", "active", "", "zsh", nil)})
		doc := parseTickDiff(t, runTickDiff(t))
		if walks != 1 {
			t.Errorf("walks = %d, want 1 (null has_agent reaches the walk)", walks)
		}
		if d := findDelta(doc, "agent_exited", "e001"); d == nil {
			t.Errorf("agent_exited missing after the walk reported dead: %v", doc.Deltas)
		}
	})
}

func TestOperatorTickDiff_LevelTriggeredReEmitUntilRemove(t *testing.T) {
	items := []trackedItem{
		paneItem("c001", "%1", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		paneItem("d002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("m003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}
	path := seedDiffState(t, items)
	stubSnapshot(t, []paneRow{
		snapRow("%1", "c001", "review-pr", "done", "idle", "8m"),
		snapRow("%3", "zz99", "apply", "active", "active", ""),
	})

	for run := 1; run <= 2; run++ {
		doc := parseTickDiff(t, runTickDiff(t))
		for _, want := range [][2]string{{"done", "c001"}, {"pane_death", "d002"}, {"pane_mismatch", "m003"}} {
			if findDelta(doc, want[0], want[1]) == nil {
				t.Errorf("run %d: %s/%s delta missing (level-triggered must re-emit): %v", run, want[0], want[1], doc.Deltas)
			}
		}
	}

	// `track rm` is the ack — the item disappears, the event stops.
	for _, id := range []string{"c001", "d002", "m003"} {
		if err := runOperatorCmd(t, operatorTrackRmCmd(), id); err != nil {
			t.Fatalf("track rm %s: %v", id, err)
		}
	}
	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Deltas) != 0 {
		t.Errorf("after rm, deltas = %v, want none", doc.Deltas)
	}
	if m := readTracked(t, path); len(m) != 0 {
		t.Errorf("tracked = %v, want empty after removes", m)
	}
}

func TestOperatorTickDiff_StageAdvanceConsumedOnRead(t *testing.T) {
	path := seedDiffState(t, []trackedItem{paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
	stubSnapshot(t, []paneRow{snapRow("%5", "a005", "review", "active", "active", "")})

	doc := parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "stage_advance", "a005"); d == nil {
		t.Fatalf("run 1: stage_advance missing: %v", doc.Deltas)
	}
	if got := entryStage(readTracked(t, path)["a005"]); got != "review" {
		t.Errorf("baseline stage = %q after run 1, want review", got)
	}

	doc = parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "stage_advance", "a005"); d != nil {
		t.Errorf("run 2: stage_advance re-emitted (consumed-on-read must not): %v", d)
	}
}

func TestOperatorTickDiff_CompletionPredicateBranches(t *testing.T) {
	items := []trackedItem{
		// stop_stage set, snapshot PAST the stop in stage order → done.
		paneItemStop("s001", "%1", "/r/a", "s1", "intake", "intake", "2026-01-01T00:00:00Z"),
		// stop_stage set, AT the stop with done → done.
		paneItemStop("s002", "%2", "/r/a", "s1", "review", "review", "2026-01-01T00:00:00Z"),
		// stop_stage set, AT the stop but still active → NOT done.
		paneItemStop("s003", "%3", "/r/a", "s1", "review", "review", "2026-01-01T00:00:00Z"),
		// stop_stage set, past the stop → done (even mid-stage).
		paneItemStop("s004", "%4", "/r/a", "s1", "review", "review", "2026-01-01T00:00:00Z"),
		// stop_stage null, ship still running → NOT done (mid-pipeline under
		// /fab-fff; the regression guard for the spurious hydrate/ship completions).
		paneItem("s005", "%5", "/r/a", "s1", "ship", "2026-01-01T00:00:00Z"),
		// stop_stage null, non-terminal → NOT done.
		paneItem("s006", "%6", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		// stop_stage null, hydrate running → NOT done.
		paneItem("s007", "%7", "/r/a", "s1", "hydrate", "2026-01-01T00:00:00Z"),
		// stop_stage null, hydrate done but pipeline continues (the transient
		// finish→ship-start window, or a parked /fab-ff run) → NOT done.
		paneItem("s008", "%8", "/r/a", "s1", "hydrate", "2026-01-01T00:00:00Z"),
		// stop_stage null, AT the terminus but still active (awaiting a PR review) → NOT done.
		paneItem("s009", "%9", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		// stop_stage null, terminus done → done.
		paneItem("s010", "%10", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
		// stop_stage null, terminus skipped (review-pr disabled at ship) → done.
		paneItem("s011", "%11", "/r/a", "s1", "review-pr", "2026-01-01T00:00:00Z"),
	}
	seedDiffState(t, items)
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
		if findDelta(doc, "done", id) == nil {
			t.Errorf("done for %s missing: %v", id, doc.Deltas)
		}
	}
	for _, id := range []string{"s003", "s005", "s006", "s007", "s008", "s009"} {
		if d := findDelta(doc, "done", id); d != nil {
			t.Errorf("done for %s emitted, want none: %v", id, d)
		}
	}
}

// --- baseline write ------------------------------------------------------------

func TestOperatorTickDiff_BaselineUpdateSameWrite(t *testing.T) {
	path := seedDiffState(t, []trackedItem{
		paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("b006", "%6", "/r/a", "s1", "review", "2026-01-01T00:00:00Z"),
	})
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

	m := readTracked(t, path)
	// Stage changed → baseline advanced and updated_at refreshed to the
	// tick's single captured timestamp (consistent with last_tick_at).
	if e := m["a005"]; entryStage(e) != "review" || entryAgent(e) != "waiting" {
		t.Errorf("a005 = %+v, want stage review / agent waiting", e.Scope)
	} else if e.UpdatedAt != state["last_tick_at"] {
		t.Errorf("a005 updated_at = %q, want last_tick_at %v (single captured tick timestamp)", e.UpdatedAt, state["last_tick_at"])
	}
	// Stage unchanged → updated_at preserved, agent still updated.
	if e := m["b006"]; entryStage(e) != "review" || entryAgent(e) != "active" {
		t.Errorf("b006 = %+v, want stage review / agent active", e.Scope)
	} else if e.UpdatedAt != "2026-01-01T00:00:00Z" {
		t.Errorf("b006 updated_at = %q, want untouched (stage unchanged)", e.UpdatedAt)
	}
}

func TestOperatorTickDiff_UnresolvedStageFabricatesNothing(t *testing.T) {
	path := seedDiffState(t, []trackedItem{paneItem("u007", "%7", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
	stubSnapshot(t, []paneRow{snapRow("%7", "u007", "—", "—", "waiting", "")})

	doc := parseTickDiff(t, runTickDiff(t))
	if len(doc.Deltas) != 0 {
		t.Errorf("deltas = %v, want none for an unresolved snapshot stage", doc.Deltas)
	}
	if e := readTracked(t, path)["u007"]; entryStage(e) != "apply" {
		t.Errorf("baseline stage = %q, want untouched (apply)", entryStage(e))
	} else if entryAgent(e) != "waiting" {
		t.Errorf("baseline agent = %q, want snapshot value (waiting)", entryAgent(e))
	}
}

func TestOperatorTickDiff_MismatchedPaneBaselineUntouched(t *testing.T) {
	path := seedDiffState(t, []trackedItem{paneItem("m003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
	stubSnapshot(t, []paneRow{snapRow("%3", "zz99", "review", "active", "waiting", "")})

	runTickDiff(t)
	if e := readTracked(t, path)["m003"]; entryStage(e) != "apply" || entryAgent(e) != "" {
		t.Errorf("mismatched entry baseline = %+v, want untouched (stage apply, agent empty)", e.Scope)
	}
}

// --- candidates / items --------------------------------------------------------

func TestOperatorTickDiff_Candidates(t *testing.T) {
	seedDiffState(t, []trackedItem{
		paneItem("w001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("w002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("i003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("a004", "%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("u005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("m006", "%6", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	})
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
		t.Fatalf("candidates = %v, want 3 (waiting+idle, pane items only)", doc.Candidates)
	}
	// waiting first (sorted by item id), then idle.
	wantOrder := []string{"w001", "w002", "i003"}
	for i, want := range wantOrder {
		if doc.Candidates[i].ID != want {
			t.Errorf("candidates[%d].id = %q, want %q", i, doc.Candidates[i].ID, want)
		}
	}
	if doc.Candidates[0].IdleDuration != nil {
		t.Errorf("waiting candidate idle_duration = %v, want null", *doc.Candidates[0].IdleDuration)
	}
	if d := doc.Candidates[2].IdleDuration; d == nil || *d != "8m" {
		t.Errorf("idle candidate idle_duration = %v, want 8m", d)
	}
}

func TestOperatorTickDiff_Items(t *testing.T) {
	items := []trackedItem{
		paneItem("f001", "%1", "/r/b", "s2", "review-pr", "2026-01-03T00:00:00Z"),
		paneItem("f002", "%2", "/r/a", "s2", "apply", "2026-01-02T00:00:00Z"),
		paneItem("f003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("f004", "%4", "/r/a", "s1", "review", "2026-01-04T00:00:00Z"), // pane dead
	}
	seedDiffState(t, items)
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
	if len(doc.Items) != 4 {
		t.Fatalf("items = %v, want 4 rows (all pane items)", doc.Items)
	}
	// Order: kind (all fab-change) → repo → id. f003/f004 (/r/a) before
	// f002 (/r/a, s2) — repo ties break on id: f002 < f003 < f004 within /r/a.
	wantOrder := []string{"f002", "f003", "f004", "f001"}
	for i, want := range wantOrder {
		if doc.Items[i]["id"] != want {
			t.Errorf("items[%d].id = %v, want %q (order %v)", i, doc.Items[i]["id"], want, wantOrder)
		}
	}
	// Joined row carries snapshot fields incl. pr_url; checked_at is null
	// (pane items render live).
	f1 := findItem(doc, "f001")
	if f1["stage"] != "review-pr" || f1["display_state"] != "done" {
		t.Errorf("f001 joined row wrong: %v", f1)
	}
	if f1["pr_url"] != "https://github.com/acme/foo/pull/412" {
		t.Errorf("f001 pr_url = %v", f1["pr_url"])
	}
	if f1["agent_state"] != "idle" || f1["idle_duration"] != "12m" {
		t.Errorf("f001 agent fields = %v/%v", f1["agent_state"], f1["idle_duration"])
	}
	if v, present := f1["checked_at"]; !present || v != nil {
		t.Errorf("f001 checked_at = %v (present %v), want key present with null", v, present)
	}
	if f1["state"] != "done" {
		t.Errorf("f001 state = %v, want done (built-in predicate fired)", f1["state"])
	}
	// Dead pane → baseline fallback row: scope identity, nulls elsewhere.
	f4 := findItem(doc, "f004")
	if f4["repo"] != "/r/a" || f4["session"] != "s1" || f4["stage"] != "review" {
		t.Errorf("f004 fallback identity wrong: %v", f4)
	}
	for _, k := range []string{"display_state", "agent_state", "idle_duration", "pr_url", "checked_at"} {
		if v, present := f4[k]; !present || v != nil {
			t.Errorf("f004 %s = %v (present %v), want key present with null", k, v, present)
		}
	}
}

// --- shell probe runner (R7) ------------------------------------------------------

func TestOperatorTickDiff_ShellProbeFlipEmitsChangedAndDone(t *testing.T) {
	// R7 first GIVEN + A-028: last {state: OPEN}; the probe prints MERGED with
	// an undeclared extra field → last stores only the declared fields,
	// unchanged resets, changed carries the per-field from/to, done fires
	// (done_when true) with then verbatim.
	checkedAt := rfc3339Ago(10 * time.Minute)
	it := shellItem("pr-913", []string{"state", "mergedAt"}, `state == "MERGED"`,
		map[string]interface{}{"state": "OPEN"}, &checkedAt)
	then := "spawn n34 in ~/code/hexokit via /fab-fff"
	it.Then = &then
	path := seedDiffState(t, []trackedItem{it})
	stubSnapshot(t, nil)
	calls := stubProbes(t, func(argv []string) (string, error) {
		return `{"state":"MERGED","mergedAt":"2026-09-11T16:35:10Z","extra":1}`, nil
	})

	doc := parseTickDiff(t, runTickDiff(t))

	if len(*calls) != 1 || (*calls)[0][0] != "probe-pr-913" {
		t.Fatalf("probe calls = %v, want exactly one for pr-913", *calls)
	}
	d := findDelta(doc, "changed", "pr-913")
	if d == nil {
		t.Fatalf("changed delta missing: %v", doc.Deltas)
	}
	fields, _ := d["fields"].(map[string]interface{})
	state, _ := fields["state"].(map[string]interface{})
	if state["from"] != "OPEN" || state["to"] != "MERGED" {
		t.Errorf("changed state = %v, want {from: OPEN, to: MERGED}", state)
	}
	mergedAt, _ := fields["mergedAt"].(map[string]interface{})
	if mergedAt["from"] != nil || mergedAt["to"] != "2026-09-11T16:35:10Z" {
		t.Errorf("changed mergedAt = %v, want {from: null, to: …}", mergedAt)
	}
	done := findDelta(doc, "done", "pr-913")
	if done == nil || done["then"] != then {
		t.Errorf("done delta wrong (then must be verbatim): %v", done)
	}

	got := readTracked(t, path)["pr-913"]
	want := map[string]interface{}{"state": "MERGED", "mergedAt": "2026-09-11T16:35:10Z"}
	if !reflect.DeepEqual(got.Last, want) {
		t.Errorf("last = %v, want %v (no extra)", got.Last, want)
	}
	if got.Unchanged != 0 || got.Failures != 0 {
		t.Errorf("unchanged/failures = %d/%d, want 0/0", got.Unchanged, got.Failures)
	}
}

func TestOperatorTickDiff_ProbeFailuresPauseOnThird(t *testing.T) {
	// R7 second GIVEN + A-032: three consecutive failing probes → the third's
	// probe_error carries failures: 3, paused: true; the fourth tick does not
	// probe and re-emits probe_error (level-triggered while paused).
	checkedAt := rfc3339Ago(10 * time.Minute)
	path := seedDiffState(t, []trackedItem{shellItem("deploy", []string{"status"}, `status == "green"`,
		map[string]interface{}{"status": "red"}, &checkedAt)})
	stubSnapshot(t, nil)
	calls := stubProbes(t, func(argv []string) (string, error) {
		return "", errors.New("exit 1: gh: Not Found")
	})

	for tick := 1; tick <= 3; tick++ {
		if tick > 1 {
			recheckItem(t, path, "deploy")
		}
		doc := parseTickDiff(t, runTickDiff(t))
		d := findDelta(doc, "probe_error", "deploy")
		if d == nil {
			t.Fatalf("tick %d: probe_error missing: %v", tick, doc.Deltas)
		}
		if d["error"] != "exit 1: gh: Not Found" {
			t.Errorf("tick %d: error = %v", tick, d["error"])
		}
		if d["failures"] != tick {
			t.Errorf("tick %d: failures = %v, want %d", tick, d["failures"], tick)
		}
		wantPaused := tick == 3
		if d["paused"] != wantPaused {
			t.Errorf("tick %d: paused = %v, want %v", tick, d["paused"], wantPaused)
		}
	}
	if len(*calls) != 3 {
		t.Fatalf("probe calls = %d, want 3", len(*calls))
	}

	// Fourth tick: not probed; the paused probe_error re-emits.
	doc := parseTickDiff(t, runTickDiff(t))
	if len(*calls) != 3 {
		t.Errorf("probe calls = %d on the paused tick, want still 3", len(*calls))
	}
	d := findDelta(doc, "probe_error", "deploy")
	if d == nil || d["paused"] != true || d["failures"] != 3 {
		t.Errorf("paused re-emission wrong: %v", d)
	}
}

func TestOperatorTickDiff_ProbeErrorShapes(t *testing.T) {
	// A-032: non-JSON stdout, a timeout, and a non-zero exit each count as one
	// failure.
	tests := []struct {
		name    string
		stdout  string
		err     error
		wantErr string
	}{
		{"non-JSON stdout", `not json`, nil, "stdout is not a JSON object"},
		{"JSON array stdout", `[1,2]`, nil, "stdout is not a JSON object"},
		{"timeout", "", errors.New("timeout after 10s"), "timeout after 10s"},
		{"non-zero exit", "", errors.New("exit 1"), "exit 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkedAt := rfc3339Ago(10 * time.Minute)
			path := seedDiffState(t, []trackedItem{shellItem("x", []string{"a"}, `a == 1`, map[string]interface{}{"a": 0}, &checkedAt)})
			stubSnapshot(t, nil)
			stubProbes(t, func(argv []string) (string, error) { return tc.stdout, tc.err })

			doc := parseTickDiff(t, runTickDiff(t))
			d := findDelta(doc, "probe_error", "x")
			if d == nil || d["error"] != tc.wantErr || d["failures"] != 1 || d["paused"] != false {
				t.Errorf("probe_error = %v, want error %q failures 1 paused false", d, tc.wantErr)
			}
			if it := readTracked(t, path)["x"]; it.Failures != 1 || it.Paused {
				t.Errorf("item failures/paused = %d/%v, want 1/false", it.Failures, it.Paused)
			}
		})
	}
}

func TestOperatorTickDiff_ProbeBudgetSkipsWithoutFailures(t *testing.T) {
	// A-032: items the 60 s tick budget cuts off emit probe_error
	// "skipped (tick budget)" with failures untouched — exercised at the
	// helper with a past deadline (the real budget is the named constant).
	checkedAt := rfc3339Ago(10 * time.Minute)
	items := []trackedItem{
		shellItem("a", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, &checkedAt),
		shellItem("b", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, &checkedAt),
	}
	out := tickDiffOutput{Deltas: []tickDelta{}}
	now := time.Now()
	runDueProbesUntil(items, &out, now, now.Add(-time.Second)) // budget already blown
	if len(out.Deltas) != 2 {
		t.Fatalf("deltas = %v, want two skipped probe_errors", out.Deltas)
	}
	for _, d := range out.Deltas {
		if d.Kind != "probe_error" || d.Error != "skipped (tick budget)" {
			t.Errorf("delta = %+v, want skipped (tick budget)", d)
		}
	}
	for _, it := range items {
		if it.Failures != 0 || it.CheckedAt == nil || *it.CheckedAt != checkedAt {
			t.Errorf("item %s touched by a budget skip: failures %d checked_at %v", it.ID, it.Failures, it.CheckedAt)
		}
	}
}

func TestOperatorTickDiff_ProbeDueSetAndUnchangedCounter(t *testing.T) {
	// Due set: checked_at null is always due; inside the cadence is not due.
	// A probe returning identical fields bumps unchanged and emits nothing.
	recent := rfc3339Ago(time.Minute)
	items := []trackedItem{
		shellItem("due-null", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, nil),
		shellItem("not-due", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, &recent),
		shellItem("same", []string{"s"}, `s == 9`, map[string]interface{}{"s": 0}, nil),
	}
	path := seedDiffState(t, items)
	stubSnapshot(t, nil)
	calls := stubProbes(t, func(argv []string) (string, error) { return `{"s":0}`, nil })

	doc := parseTickDiff(t, runTickDiff(t))
	if len(*calls) != 2 {
		t.Errorf("probe calls = %v, want due-null and same only", *calls)
	}
	if len(doc.Deltas) != 0 {
		t.Errorf("deltas = %v, want none (identical fields)", doc.Deltas)
	}
	m := readTracked(t, path)
	if got := m["due-null"].Unchanged; got != 1 {
		t.Errorf("due-null unchanged = %d, want 1", got)
	}
	if got := m["same"].Unchanged; got != 1 {
		t.Errorf("same unchanged = %d, want 1", got)
	}
	if m["not-due"].CheckedAt == nil || *m["not-due"].CheckedAt != recent {
		t.Errorf("not-due checked_at moved — it must not be probed: %v", m["not-due"].CheckedAt)
	}
}

// --- agent items: stale / needs_check (R8) ---------------------------------------

func TestOperatorTickDiff_StaleAgentItemNeedsCheck(t *testing.T) {
	// R8 GIVEN + A-029: an agent item 11 minutes past a 5m cadence emits stale
	// with age and appears in needs_check; observe clears both.
	ago := 11 * time.Minute
	items := []trackedItem{agentItem("linear-bugs", "list issues …", 5*time.Minute, &ago)}
	seedDiffState(t, items)
	stubSnapshot(t, nil)

	doc := parseTickDiff(t, runTickDiff(t))
	d := findDelta(doc, "stale", "linear-bugs")
	if d == nil || d["age"] != "11m" {
		t.Errorf("stale delta wrong: %v (want age 11m)", d)
	}
	if len(doc.NeedsCheck) != 1 {
		t.Fatalf("needs_check = %v, want one row", doc.NeedsCheck)
	}
	nc := doc.NeedsCheck[0]
	if nc["id"] != "linear-bugs" || nc["kind"] != "linear" || nc["age"] != "11m" || nc["instruction"] != "list issues …" {
		t.Errorf("needs_check row wrong: %v", nc)
	}

	// observe clears stale (checked_at resets) and the due listing.
	if err := runOperatorCmd(t, operatorTrackObserveCmd(), "linear-bugs", "--json", `{"new":[]}`); err != nil {
		t.Fatalf("observe: %v", err)
	}
	doc = parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "stale", "linear-bugs"); d != nil {
		t.Errorf("stale re-emitted after observe: %v", d)
	}
	if len(doc.NeedsCheck) != 0 {
		t.Errorf("needs_check = %v after observe, want empty", doc.NeedsCheck)
	}
}

func TestOperatorTickDiff_NeedsCheckForcesFullDocument(t *testing.T) {
	// A-022: a non-empty needs_check forces the full document even on a quiet
	// non-10th tick.
	ago := 11 * time.Minute
	seedDiffState(t, []trackedItem{agentItem("linear-bugs", "list issues", 5*time.Minute, &ago)}) // tick 5 → 6
	stubSnapshot(t, nil)

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	assertDocKeys(t, out, false)
	if !strings.Contains(out, "needs_check:") || !strings.Contains(out, "linear-bugs") {
		t.Errorf("quiet tick with a due agent item must emit needs_check in the full doc:\n%s", out)
	}
}

// --- item state derivation (R10) --------------------------------------------------

func TestOperatorTickDiff_HeldPendingChain(t *testing.T) {
	// R10 GIVEN + A-030: ef56 (fab-change, pane null, depends_on k8ds) renders
	// held while k8ds is live, pending once k8ds is done.
	k8ds := paneItem("k8ds", "%7", "/r/a", "s1", "review", "2026-01-01T00:00:00Z")
	ef56 := paneItem("ef56", "", "/r/a", "s1", "", "2026-01-01T00:00:00Z")
	ef56.Scope["pane"] = nil
	ef56.DependsOn = []string{"k8ds"}
	seedDiffState(t, []trackedItem{k8ds, ef56})
	stubSnapshot(t, []paneRow{snapRow("%7", "k8ds", "review", "active", "active", "")})

	doc := parseTickDiff(t, runTickDiff(t))
	if row := findItem(doc, "ef56"); row["state"] != "held" || row["next"] != "held: k8ds" {
		t.Errorf("ef56 = state %v next %v, want held / held: k8ds", row["state"], row["next"])
	}
	if row := findItem(doc, "k8ds"); row["state"] != "live" {
		t.Errorf("k8ds state = %v, want live", row["state"])
	}

	// k8ds completes (review-pr done) → ef56 goes pending with next: spawn.
	stubSnapshot(t, []paneRow{snapRow("%7", "k8ds", "review-pr", "done", "idle", "1m")})
	doc = parseTickDiff(t, runTickDiff(t))
	if findDelta(doc, "done", "k8ds") == nil {
		t.Fatalf("done delta for k8ds missing: %v", doc.Deltas)
	}
	if row := findItem(doc, "ef56"); row["state"] != "pending" || row["next"] != "spawn" {
		t.Errorf("ef56 = state %v next %v, want pending / spawn", row["state"], row["next"])
	}
}

func TestOperatorTickDiff_UnknownDependencyEmitsProbeError(t *testing.T) {
	// R10: a depends_on id missing from the list is a probe_error-class
	// warning naming the missing id; the item renders held.
	it := paneItem("ef56", "", "/r/a", "s1", "", "2026-01-01T00:00:00Z")
	it.Scope["pane"] = nil
	it.DependsOn = []string{"zz"}
	seedDiffState(t, []trackedItem{it})
	stubSnapshot(t, nil)

	doc := parseTickDiff(t, runTickDiff(t))
	d := findDelta(doc, "probe_error", "ef56")
	if d == nil || d["error"] != "unknown dependency zz" {
		t.Errorf("probe_error wrong: %v, want unknown dependency zz", d)
	}
	if row := findItem(doc, "ef56"); row["state"] != "held" || row["next"] != "held: zz" {
		t.Errorf("ef56 = state %v next %v, want held / held: zz", row["state"], row["next"])
	}
}

func TestOperatorTickDiff_ItemStates(t *testing.T) {
	// State coverage over the items rows: done (shell, done_when true),
	// paused, watching (shell, done_when false), live (pane), pending.
	merged := shellItem("pr-9", []string{"state"}, `state == "MERGED"`, map[string]interface{}{"state": "MERGED"}, nil)
	open := shellItem("pr-10", []string{"state"}, `state == "MERGED"`, map[string]interface{}{"state": "OPEN"}, nil)
	open.CheckedAt = strPtr(rfc3339Ago(time.Minute)) // not due → no probe needed
	pausedIt := shellItem("pr-11", []string{"state"}, `state == "MERGED"`, map[string]interface{}{"state": "OPEN"}, nil)
	pausedIt.Paused = true
	pausedIt.Failures = 1 // user-paused (not the failure cap) → no probe_error re-emission
	queued := paneItem("ef56", "", "/r/a", "s1", "", "2026-01-01T00:00:00Z")
	queued.Scope["pane"] = nil
	live := paneItem("k8ds", "%7", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")
	seedDiffState(t, []trackedItem{merged, open, pausedIt, queued, live})
	stubSnapshot(t, []paneRow{snapRow("%7", "k8ds", "apply", "active", "active", "")})
	calls := stubProbes(t, func(argv []string) (string, error) {
		return "", errors.New("must not probe: not due / paused / done")
	})

	doc := parseTickDiff(t, runTickDiff(t))
	for id, want := range map[string]string{
		"pr-9": "done", "pr-10": "watching", "pr-11": "paused", "ef56": "pending", "k8ds": "live",
	} {
		if row := findItem(doc, id); row["state"] != want {
			t.Errorf("%s state = %v, want %s", id, row["state"], want)
		}
	}
	if len(*calls) != 0 {
		t.Errorf("probes ran for done/not-due/paused items: %v", *calls)
	}
	// done is level-triggered: pr-9 emits done every tick until track rm.
	if d := findDelta(doc, "done", "pr-9"); d == nil {
		t.Errorf("done delta for pr-9 missing (level-triggered): %v", doc.Deltas)
	}
	if d := findDelta(doc, "probe_error", "pr-11"); d != nil {
		t.Errorf("user-paused item emitted probe_error (failures < cap): %v", d)
	}
}

// --- quiet tick (--diff --quiet) -----------------------------------------------

// assertDocKeys asserts the items/fleet_summary key presence contract on raw
// stdout (the contract is "the key is absent", not "the key is empty").
func assertDocKeys(t *testing.T, out string, wantSummary bool) {
	t.Helper()
	hasSummary := strings.Contains(out, "fleet_summary:")
	hasItems := strings.Contains(out, "items:")
	if wantSummary && (!hasSummary || hasItems) {
		t.Errorf("want fleet_summary present, items absent:\n%s", out)
	}
	if !wantSummary && (!hasItems || hasSummary) {
		t.Errorf("want items present, fleet_summary absent:\n%s", out)
	}
}

func TestOperatorTickDiff_QuietNoDeltasEmitsSummary(t *testing.T) {
	seedDiffState(t, []trackedItem{
		paneItem("w001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("a002", "%2", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
	}) // tick 5 → 6: not a multiple of 10
	stubSnapshot(t, []paneRow{
		snapRow("%1", "w001", "apply", "active", "waiting", ""),
		snapRow("%2", "a002", "apply", "active", "active", ""),
	})

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	assertDocKeys(t, out, true)
	for _, block := range []string{"deltas: []", "candidates:", "needs_check: []"} {
		if !strings.Contains(out, block) {
			t.Errorf("stdout missing %q (all blocks are always emitted):\n%s", block, out)
		}
	}
	// Block order is pinned: deltas, candidates, needs_check, then fleet_summary.
	di, ci, ni, fi := strings.Index(out, "deltas:"), strings.Index(out, "candidates:"), strings.Index(out, "needs_check:"), strings.Index(out, "fleet_summary:")
	if !(di >= 0 && di < ci && ci < ni && ni < fi) {
		t.Errorf("block order wrong (want deltas < candidates < needs_check < fleet_summary):\n%s", out)
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

func TestOperatorTickDiff_QuietSummaryMixedItems(t *testing.T) {
	// R9 GIVEN + A-022: three pane items (active, idle, waiting) and two shell
	// items, no deltas → fleet_summary {tracked: 5, waiting: 1, idle: 1,
	// active: 1, unknown: 0}; the shell items count only in tracked.
	recent := rfc3339Ago(time.Minute)
	seedDiffState(t, []trackedItem{
		paneItem("w001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("i003", "%3", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		paneItem("a004", "%4", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z"),
		shellItem("s1", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, &recent),
		shellItem("s2", []string{"s"}, `s == 1`, map[string]interface{}{"s": 0}, &recent),
	})
	stubSnapshot(t, []paneRow{
		snapRow("%1", "w001", "apply", "active", "waiting", ""),
		snapRow("%3", "i003", "apply", "active", "idle", "8m"),
		snapRow("%4", "a004", "apply", "active", "active", ""),
	})
	calls := stubProbes(t, func(argv []string) (string, error) { return `{"s":0}`, nil })

	out, err := runTickDiffArgs(t, "--diff", "--quiet")
	if err != nil {
		t.Fatalf("tick-start --diff --quiet: %v", err)
	}
	assertDocKeys(t, out, true)
	if len(*calls) != 0 {
		t.Errorf("probes = %v, want none (both shell items inside their cadence)", *calls)
	}
	doc := parseTickDiff(t, out)
	want := tickFleetSummary{Tracked: 5, Waiting: 1, Idle: 1, Active: 1, Unknown: 0}
	if doc.FleetSummary == nil || *doc.FleetSummary != want {
		t.Fatalf("fleet_summary = %v, want %+v", doc.FleetSummary, want)
	}
	s := *doc.FleetSummary
	if s.Tracked < s.Waiting+s.Idle+s.Active+s.Unknown {
		t.Errorf("invariant violated: tracked < waiting+idle+active+unknown: %+v", s)
	}
}

func TestOperatorTickDiff_QuietWithDeltaEmitsFullItems(t *testing.T) {
	seedDiffState(t, []trackedItem{paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
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

func TestOperatorTickDiff_QuietEveryTenthTickEmitsFullItems(t *testing.T) {
	seed := func(t *testing.T, tickCount int) {
		t.Helper()
		seedDiffStateAt(t, []trackedItem{paneItem("w001", "%1", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")}, tickCount)
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
	path := seedDiffState(t, []trackedItem{paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})

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

func TestOperatorTickDiff_QuietEmptyTracked(t *testing.T) {
	t.Run("non-10th tick emits all-zero summary", func(t *testing.T) {
		path := withOperatorState(t, "tick_count: 5\ntracked: []\ncustom_key: preserve-me\n")
		stubQuietClock(t)
		called := stubSnapshot(t, nil)

		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if *called {
			t.Error("snapshot fn invoked on an empty tracked list — must be skipped")
		}
		assertDocKeys(t, out, true)
		for _, block := range []string{"deltas: []", "candidates: []", "needs_check: []", "tracked: 0", "waiting: 0", "idle: 0", "active: 0", "unknown: 0"} {
			if !strings.Contains(out, block) {
				t.Errorf("stdout missing %q:\n%s", block, out)
			}
		}
		if state := readStateFile(t, path); state["tick_count"] != 6 {
			t.Errorf("tick_count = %v, want 6 (no-op tick still increments)", state["tick_count"])
		}
	})

	t.Run("10th tick emits items: []", func(t *testing.T) {
		withOperatorState(t, "tick_count: 9\ntracked: []\ncustom_key: preserve-me\n")
		stubQuietClock(t)
		called := stubSnapshot(t, nil)

		out, err := runTickDiffArgs(t, "--diff", "--quiet")
		if err != nil {
			t.Fatalf("tick-start --diff --quiet: %v", err)
		}
		if *called {
			t.Error("snapshot fn invoked on an empty tracked list — must be skipped")
		}
		assertDocKeys(t, out, false)
		if !strings.Contains(out, "items: []") {
			t.Errorf("10th tick missing items: []:\n%s", out)
		}
	})
}

// --- no-op tick + flagless parity ----------------------------------------------

func TestOperatorTickDiff_EmptyTrackedSkipsSnapshot(t *testing.T) {
	path := withOperatorState(t, "tick_count: 5\ntracked: []\ncustom_key: preserve-me\n")
	stubQuietClock(t)
	called := stubSnapshot(t, nil)

	out := runTickDiff(t)

	if *called {
		t.Error("snapshot fn invoked on an empty tracked list — must be skipped")
	}
	for _, block := range []string{"deltas: []", "candidates: []", "needs_check: []", "items: []"} {
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
	path := seedDiffState(t, []trackedItem{paneItem("a005", "%5", "/r/a", "s1", "apply", "2026-01-01T00:00:00Z")})
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
	for _, k := range []string{"deltas", "candidates", "needs_check", "items"} {
		if _, ok := state[k]; ok {
			t.Errorf("flagless state file gained %q key", k)
		}
	}
	// The snapshot would say review/waiting — the flagless path must not diff.
	if e := readTracked(t, path)["a005"]; entryStage(e) != "apply" || entryAgent(e) != "" {
		t.Errorf("flagless path touched the baseline: %+v", e.Scope)
	}
}

// TestOperatorTickDiff_ChangedConsumedOnRead (A-021): a field delta emits
// changed once; the same-write last update consumes it, so the next tick with
// identical probe output emits nothing (and bumps unchanged).
func TestOperatorTickDiff_ChangedConsumedOnRead(t *testing.T) {
	it := shellItem("pr-1", []string{"state"}, `state == "MERGED"`, map[string]interface{}{"state": "OPEN"}, nil)
	path := seedDiffState(t, []trackedItem{it})
	stubSnapshot(t, nil)
	stubProbes(t, func(argv []string) (string, error) { return `{"state":"OPEN","n":1}`, nil })

	doc := parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "changed", "pr-1"); d != nil {
		t.Fatalf("run 1: changed emitted with identical state (only undeclared n is new): %v", d)
	}

	// Probe flips the declared field → changed fires once.
	recheckItem(t, path, "pr-1")
	stubProbes(t, func(argv []string) (string, error) { return `{"state":"REVIEW"}`, nil })
	doc = parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "changed", "pr-1"); d == nil {
		t.Fatalf("run 2: changed missing on the flip: %v", doc.Deltas)
	}

	recheckItem(t, path, "pr-1")
	doc = parseTickDiff(t, runTickDiff(t))
	if d := findDelta(doc, "changed", "pr-1"); d != nil {
		t.Errorf("run 3: changed re-emitted (consumed-on-read must not): %v", d)
	}
}
