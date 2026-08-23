# Plan: Operator Tick-Diff Offload (`fab operator tick-start --diff`)

**Change**: 260823-dbwg-operator-tick-diff-offload
**Intake**: `intake.md`

## Requirements

### CLI: `fab operator tick-start --diff`

#### R1: `--diff` flag with a single parseable stdout document; flagless path byte-identical
`fab operator tick-start` SHALL gain a `--diff` boolean flag. Flagless invocation MUST remain byte-identical to today (increments `tick_count`, writes `last_tick_at`, prints `tick: N\nnow: HH:MM`). With `--diff`, stdout SHALL be one document: the same two `tick:`/`now:` lines first, then three YAML blocks in this order and with these exact keys:

```yaml
deltas:
    - kind: completion            # completion | pane_death | pane_mismatch | stage_advance | review_fail
      change: r3m7
      pane: "%3"
      # kind-specific fields:
      #   completion    → stage, display_state
      #   pane_death    → (none)
      #   pane_mismatch → found   (the change ID actually occupying the pane; null when none resolvable)
      #   stage_advance → from, to
      #   review_fail   → from (review), to (apply)
candidates:
    - pane: "%7"
      change: k8ds
      agent_state: waiting        # waiting | idle
      idle_duration: null         # non-null only for idle (upstream idle-only semantics)
fleet:
    - change: r3m7
      pane: "%3"
      repo: /home/user/code/foo
      session: work
      stage: review-pr
      display_state: done
      agent_state: idle           # active | waiting | idle | null (unknown)
      idle_duration: 8m           # null unless idle
      pr_url: https://github.com/acme/foo/pull/412   # null when none
fleet: []                          # empty-list form when nothing monitored (likewise deltas/candidates)
```

- **GIVEN** a monitored set and a live tmux server
- **WHEN** `fab operator tick-start --diff` runs
- **THEN** stdout carries `tick:`/`now:` followed by `deltas:`, `candidates:`, and `fleet:` blocks
- **AND** `fab operator tick-start` (flagless) output and state writes are byte-identical to the pre-change binary

- **GIVEN** an empty monitored set
- **WHEN** `--diff` runs
- **THEN** output is `deltas: []`, `candidates: []`, `fleet: []`, the tick still increments, and the pane snapshot subprocess is skipped entirely (no-op tick is first-class)

#### R2: Two delivery classes
The diff engine SHALL emit events in two classes:
- **Level-triggered, re-emitted every tick until acted on** — `completion`, `pane_death`, `pane_mismatch`. All three are stateless predicates over the current snapshot (no baseline read). The natural ack is `fab operator remove` (the entry disappears, the event stops). A crash between diff and action loses nothing.
- **Consumed-on-read (baseline-diffed)** — `stage_advance` and `review_fail`, computed against the entry's stored `stage` and consumed by the same-write baseline update (R7). A lost one costs a missed report only.

- **GIVEN** a monitored entry whose change has completed (or whose pane died / was recycled)
- **WHEN** `--diff` runs on consecutive ticks with no `fab operator remove` in between
- **THEN** the `completion` (or `pane_death`/`pane_mismatch`) delta re-appears on every tick

- **GIVEN** a monitored entry whose stage advanced apply → review
- **WHEN** `--diff` runs twice
- **THEN** the first run emits `stage_advance {from: apply, to: review}` and updates the baseline; the second emits nothing for that entry

#### R3: Completion is a display-state/terminal-stage predicate, never a stage diff
`completion` SHALL be detected from the snapshot row's `(stage, display_state)` pair (the `status.DisplayStage` values `fab pane map` already resolves):
- `stop_stage` null: complete when snapshot stage ∈ {`hydrate`, `ship`, `review-pr`} (today's §4 step 2 terminal set; a fully-done pipeline reads `review-pr`/`done` and is contained in it).
- `stop_stage` set: complete when stage-order(snapshot stage) > stage-order(`stop_stage`), OR snapshot stage == `stop_stage` with `display_state` ∈ {`done`, `skipped`} (a finished stop-stage auto-activates the next stage, so an order comparison is required — equality alone would race the transition).

- **GIVEN** a monitored entry at `review-pr` whose review-pr just finished (stage string unchanged, `display_state` flipped to `done`)
- **WHEN** `--diff` runs
- **THEN** a `completion` delta is emitted (a stage-diff provably cannot detect this)

- **GIVEN** an entry with `stop_stage: intake` whose intake finished (snapshot now `apply`/`ready`)
- **WHEN** `--diff` runs
- **THEN** a `completion` delta is emitted

#### R4: `pane_mismatch` cross-check (recycled pane IDs)
For each monitored entry whose pane IS present in the snapshot, the entry's change ID SHALL be cross-checked against the snapshot row's resolved change ID (the 4-char ID extracted from the row's change folder). A mismatch — including a pane now resolving to no change at all — SHALL emit a level-triggered `pane_mismatch` delta carrying `found` (the occupying change ID, or null). A mismatched pane MUST NOT be diffed, baseline-updated, listed in `candidates:`, or joined for `fleet:` fields (its fleet row falls back to baseline fields per R6). Rationale: tmux recycles `%N` pane IDs across server restarts while the socket-keyed state file survives (pane-identity-keying contract, #612).

- **GIVEN** a monitored entry `{r3m7, pane %3}` and a snapshot where `%3` now hosts change `ab12`
- **WHEN** `--diff` runs
- **THEN** `pane_mismatch {change: r3m7, pane: "%3", found: ab12}` is emitted, no other delta references `%3`, and `%3` is not a candidate

#### R5: `candidates:` block — monitored-only, waiting-first
`candidates:` SHALL list monitored agents whose snapshot row's `agent_state` is `waiting` or `idle` — `waiting` entries first, then `idle`, deterministically ordered (sorted by change ID within each class). Each row carries `pane`, `change`, `agent_state`, `idle_duration`. Unknown-state (null) and `active` panes are excluded; mismatched panes are excluded (R4). Population is monitored entries only — on rk-less servers every pane reads unknown, so `candidates:` is empty, identical to today's §5 sweep policy.

- **GIVEN** monitored entries with agent states {waiting, idle, active, unknown}
- **WHEN** `--diff` runs
- **THEN** `candidates:` lists only the waiting and idle entries, waiting first, each with `idle_duration` (non-null for idle)

#### R6: `fleet:` block — the status frame's data source
`fleet:` SHALL carry one row per monitored entry (all of them): `change`, `pane`, `repo`, `session`, `stage`, `display_state`, `agent_state`, `idle_duration`, `pr_url`. Fields come from the snapshot join when the pane matched (R4); a dead or mismatched pane's row falls back to the baseline's stored `repo`/`session`/`stage` with null `display_state`/`agent_state`/`idle_duration`/`pr_url`. Rows are deterministically ordered (repo, then session, then enrollment timestamp, then change ID) to match the frame's rendering order. Without this block the skill would keep fetching the full pane map and the offload would evaporate.

- **GIVEN** three monitored entries across two repos, one with a shipped PR
- **WHEN** `--diff` runs
- **THEN** `fleet:` has three rows grouped repo-first, and the shipped entry's row carries its `pr_url`

#### R7: Baseline updated in the same atomic mutation; `--diff` is the authoritative baseline writer
The `--diff` path SHALL perform tick bookkeeping (`tick_count`, `last_tick_at`) and the baseline update in ONE `mutateOperatorState` call (load tolerant → typed edit → atomic temp+rename). For each cleanly-joined entry (pane present, change ID matches): `stage` ← snapshot stage, `agent` ← snapshot agent state (verbatim; empty for unknown), and `last_transition` touched **iff** the stage value changed (preserving `fab operator update`'s semantics). Dead/mismatched entries' baselines stay untouched. A snapshot row whose stage is unresolved (em-dash → null) leaves the baseline stage untouched and emits no `stage_advance`. Unknown top-level keys survive (tolerant-read/typed-write posture); the `monitored` section is re-marshaled typed. The stale comment at `operator_tick_start.go:30–31` ("and the owned sections, untouched here") SHALL be corrected — with `--diff` the `monitored` section IS touched.

- **GIVEN** a `--diff` run that observed a stage advance
- **WHEN** the state file is read back
- **THEN** the entry's `stage` reflects the snapshot, `last_transition` was refreshed, `tick_count`/`last_tick_at` advanced in the same write, and unknown top-level keys survived

### Go: panemap extraction

#### R8: `collectPaneRows` extraction + snapshot seam; `fab pane map` unchanged
`panemap.go`'s discovery+resolve pipeline (the `discoverPanesViaRK` → `discoverPanes` fallback plus the per-pane cache/resolve loop inside `runPaneMap`) SHALL be extracted into `collectPaneRows(mode sessionMode, sessionName, server string) ([]paneRow, error)`; `runPaneMap` calls it (pure extraction — `fab pane map` behavior byte-identical, existing panemap tests pass unchanged). A package-var seam `var tickSnapshotRows = func() ([]paneRow, error) { return collectPaneRows(sessionAll, "", "") }` (same file, the `rkPanesRunner`/`operatorStatePathOverride` precedent) SHALL be the `--diff` path's snapshot source, stubbable in tests. `paneRow` SHALL gain an internal `changeID` field (4-char ID extracted from the resolved change folder name in `resolvePane`) for the R4 join — absent from table and JSON output.

- **GIVEN** the existing panemap unit tests
- **WHEN** they run against the extracted code
- **THEN** they pass without modification, and `fab pane map`/`--json` output is unchanged

### Skill: coupled §4 rewrite (same change — Critical Coupling)

#### R9: `fab-operator.md` §4 tick rewired to `--diff`
`src/kit/skills/fab-operator.md` SHALL be rewired (§4 only; §5 detection mechanics, § Notes, and §6 auto-merge untouched):
- **Tick step 1 (Snapshot)**: run `fab operator tick-start --diff`; parse `tick:`/`now:`, act on `deltas:`, and render the status frame **from the `fleet:` block**. The per-tick `fab pane map --all-sessions --json` call and the per-tick `fab operator state` read are dropped (the watch pass reads state on its own step as today; § Operator State File's "read on startup and every tick" sentence is corrected accordingly).
- **Level-triggered semantics stated**: `completion`/`pane_death`/`pane_mismatch` re-emit every tick until acted on; `fab operator remove` is the ack (ties into step 5 Removals); `pane_mismatch` is report + remove, never a candidate. Deltas are acted on before answers (a completion removes the entry and skips its answer).
- **Step 2 (Auto-nudge)**: the per-tick sweep population is the `candidates:` block (waiting-first then idle — computed to match today's §5 policy exactly); §5 capture/guard/answer mechanics unchanged.
- **Step 6 (Observed-field updates)**: the per-tick `fab operator update` stage/agent bookkeeping is REMOVED from the diff path — `--diff` owns the baseline write; `fab operator update` stays for non-baseline field edits (e.g. `stop_stage`).
- **Status Frame Format**: note the frame's data source is the tick's `fleet:` block (watch table still fed by the watch pass).
- **Version-skew fallback (one line in §4)**: if `tick-start --diff` errors as an unknown flag (new skill, old installed binary — bottle lag is a recorded recurring lesson), fall back to the flagless tick (full pane map + per-pane capture + `update` bookkeeping) for the session and report the mismatch once.

- **GIVEN** the rewritten §4
- **WHEN** a tick runs against a new binary
- **THEN** the tick is `tick-start --diff` + delta handling + fleet-sourced frame, with no per-tick pane-map/state/`update` calls
- **AND** against an old binary, the unknown-flag error routes to the flagless fallback for the session

#### R10: `_cli-fab.md` § fab operator tick-start updated (constitution-mandated)
`src/kit/skills/_cli-fab.md` § fab operator tick-start SHALL document the `[--diff]` signature, the output contract (delivery classes, `candidates:`, `fleet:`, kind-specific fields), the authoritative-baseline-writer semantics (`last_transition` iff stage changed), the empty-monitored behavior, and that the flagless form is unchanged — shipped simultaneously with the CLI change.

- **GIVEN** the updated section
- **WHEN** an agent reads it
- **THEN** it can parse `--diff` output and knows `update` is no longer the diff path's baseline writer

#### R11: Stale-claim sweep of aggregate restatements
The old per-tick claims ("run `fab pane map --all-sessions --json` … read the state via `fab operator state`", step-6 `update` bookkeeping) SHALL be grepped repo-wide and updated where they restate the TICK's mechanics — known site: `docs/specs/skills.md` § /fab-operator partial flow. Exclusions: `docs/specs/findings/` (dated historical review documents), §1 "re-derive before every action" prose (the on-demand `fab pane map` surface survives — only the per-tick snapshot changes), §2 Init step 3 (startup orientation display), and `docs/memory/runtime/operator.md` (hydrate-stage work, not apply).

- **GIVEN** the completed apply
- **WHEN** `grep -rn "pane map --all-sessions" docs/specs src/kit/skills` runs (excluding findings/ and `_cli-fab.md`)
- **THEN** no remaining hit describes the operator's per-tick snapshot as a pane-map + state read

### Tests

#### R12: tick-diff test coverage
A new `operator_tick_diff_test.go` (or extension of `operator_test.go`) SHALL cover, via the `operatorStatePathOverride` + `tickSnapshotRows` seams:
- every event kind from a seeded baseline + stubbed snapshot;
- `completion`/`pane_death`/`pane_mismatch` re-emit across consecutive runs until the entry is removed;
- completion via the display-state/terminal-stage predicate (stage string unchanged, display flips — the case a stage-diff cannot catch), including the `stop_stage` order-comparison;
- baseline updated in the same write (`stage`/`agent`/`last_transition`-iff-stage-changed; unknown top-level keys preserved);
- candidate ordering (waiting first) + `idle_duration` presence; unknown/active/mismatched exclusion;
- `fleet:` block shape incl. dead-pane baseline fallback row;
- empty monitored ⇒ `deltas: []`/`candidates: []`/`fleet: []`, tick still increments, snapshot fn not invoked;
- flagless path byte-identical (existing tests keep passing).

- **GIVEN** `go test ./src/go/fab/cmd/fab/`
- **WHEN** the suite runs
- **THEN** all new and existing tests pass

### Non-Goals

- `fab pane questions` and any §5 detection-mechanics change — Change 4 (B2).
- `fab operator state --frame` (C3), event wake (C1) — follow-ups.
- No migration: additive behavior, old state files load fine, no schema field added to `monitored` (see DD below).
- `docs/memory/runtime/pane-commands.md` re-scope — belongs to B2.

### Design Decisions

#### review_fail needs no new baseline field
**Decision**: `review_fail` = baseline `stage == review` AND snapshot stage == `apply` (the rework reset path re-activates apply, so the stage string alone carries the signal). The conditional additive `display_state` baseline field the intake held open (assumption 13) is NOT added; the `monitored` schema stays byte-identical.
**Why**: The fail+reset choreography always lands the change at `apply`/`active`, which `status.DisplayStage` surfaces as stage `apply` — a plain stage comparison detects it. A parked review exhaustion (stage `review`, state `failed`) is not a rework event; it surfaces through the fleet row's `display_state` and stuck detection.
**Rejected**: Storing last-known `display_state` per entry — a schema addition with no consumer once the stage comparison is shown sufficient.
*Introduced by*: 260823-dbwg-operator-tick-diff-offload

#### Snapshot skipped entirely on an empty monitored set
**Decision**: `--diff` with zero monitored entries performs no pane enumeration — tick bookkeeping plus empty blocks only.
**Why**: The blocks are provably empty (candidates/fleet are monitored-scoped; every delta references a monitored entry); skipping saves the tmux/rk subprocess on the quietest tick.
**Rejected**: Always snapshotting — pays enumeration for an output that cannot change.
*Introduced by*: 260823-dbwg-operator-tick-diff-offload

#### Completed-but-prompting panes stay candidates
**Decision**: The binary does not subtract completion/death deltas from `candidates:` beyond the R4 mismatch exclusion (dead panes are structurally absent; completed entries whose agent sits `waiting`/`idle` still appear).
**Why**: "Deltas-first" is skill policy (a completion removes the entry and skips its answer — plan § Edge cases); encoding it binary-side would couple the diff engine to answer-model choreography.
**Rejected**: Binary-side candidate suppression on completion — duplicates a policy the skill must state anyway.
*Introduced by*: 260823-dbwg-operator-tick-diff-offload

## Tasks

### Phase 2: Core Implementation

- [x] T001 Extract `collectPaneRows(mode, sessionName, server)` from `runPaneMap` in `src/go/fab/cmd/fab/panemap.go` (discovery fallback + cache/resolve loop, pure extraction); add internal `paneRow.changeID` populated in `resolvePane` via `resolve.ExtractID(folderName)` (absent from table/JSON); add `var tickSnapshotRows = func() ([]paneRow, error) { return collectPaneRows(sessionAll, "", "") }` <!-- R8 -->
- [x] T002 Add `--diff` flag to `operator_tick_start.go`: branch in `runOperatorTickStart`, flagless path untouched; `--diff` path snapshots via `tickSnapshotRows` (skipping when monitored empty), runs ONE `mutateOperatorState` doing tick bookkeeping + baseline update (`stage`/`agent`, `last_transition` iff stage changed); correct the stale owned-sections comment at lines 30–31 <!-- R1, R7 -->
- [x] T003 Implement the level-triggered predicates over the joined snapshot: `completion` (terminal set {hydrate, ship, review-pr} when `stop_stage` null; stage-order > stop, or == stop with display_state done/skipped, when set), `pane_death` (pane absent), `pane_mismatch` (changeID mismatch incl. null; `found` field; excluded from join/baseline/candidates) <!-- R2, R3, R4 -->
- [x] T004 Implement consumed-on-read deltas from the baseline diff: `stage_advance {from, to}` and `review_fail {from: review, to: apply}` (review→apply wins over stage_advance for that transition; unresolved snapshot stage emits nothing and leaves baseline untouched) <!-- R2 -->
- [x] T005 Emit `candidates:` (monitored-only, waiting-first then idle, sorted by change ID within class; `pane`/`change`/`agent_state`/`idle_duration`; unknown/active/mismatched excluded) <!-- R5 -->
- [x] T006 Emit `fleet:` (one row per monitored entry; snapshot-joined fields, baseline fallback for dead/mismatched panes; ordered repo → session → enrolled_at → change ID) and serialize the full `--diff` stdout document (tick/now lines + three YAML blocks, empty-list forms) <!-- R6, R1 -->

### Phase 3: Integration & Edge Cases

- [x] T007 Write `src/go/fab/cmd/fab/operator_tick_diff_test.go`: all R12 cases via the `operatorStatePathOverride` + `tickSnapshotRows` seams (event kinds, re-emit-until-remove, display-state completion + stop_stage ordering, same-write baseline + unknown-key survival, candidate ordering/exclusions, fleet shape + fallback row, empty-monitored short-circuit asserting the snapshot fn is not called, flagless byte-identical) <!-- R12 -->
- [x] T008 Run `gofmt -l src/go/fab/` (must be clean) and `go test ./src/go/fab/cmd/fab/` — all new and existing tests green (existing panemap tests unmodified per R8) <!-- R12 -->

### Phase 4: Coupled skill + doc updates

- [x] T009 Rewrite `src/kit/skills/fab-operator.md` §4 per R9: tick step 1 → `tick-start --diff` + deltas + fleet-sourced frame; step 2 population = `candidates:`; step 5 removals as level-triggered ack; step 6 `update` bookkeeping removed from the diff path (kept for non-baseline edits); § Operator State File read-cadence sentence corrected; Status Frame data-source note; the one-line version-skew fallback. §5/§ Notes/§6 untouched <!-- R9 -->
- [x] T010 Update `src/kit/skills/_cli-fab.md` § fab operator tick-start with the `[--diff]` signature and full output/baseline-writer contract per R10 <!-- R10 -->
- [ ] T011 Sweep: grep `pane map --all-sessions` + per-tick `fab operator state`/`update` claims across `docs/specs/` and `src/kit/skills/` (excluding `findings/`), update tick-mechanics restatements — known site `docs/specs/skills.md` § /fab-operator flow; leave §1 re-derive-before-action, §2 Init step 3, and memory files (hydrate) alone <!-- R11 -->

## Execution Order

- T001 blocks T002–T006 (the seam); T002 blocks T003–T006 (the mutation skeleton); T003–T006 then T007–T008; Phase 4 is independent of Phase 3 but lands after core so the documented contract matches the code.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `--diff` emits the pinned single-document contract; flagless `tick-start` output and writes are byte-identical
- [ ] A-002 R2: level-triggered vs consumed-on-read classes behave per spec (re-emit vs consume)
- [ ] A-003 R3: completion fires from the display-state/terminal-stage predicate incl. both `stop_stage` branches
- [ ] A-004 R4: `pane_mismatch` emitted on recycled panes with `found`; mismatched panes never diffed/baselined/candidates
- [ ] A-005 R5: `candidates:` is monitored-only, waiting-first, with `idle_duration`, unknown/active excluded
- [ ] A-006 R6: `fleet:` carries all monitored entries with the nine fields and baseline fallback rows
- [ ] A-007 R7: baseline + tick bookkeeping land in one atomic write; `last_transition` touched iff stage changed; unknown top-level keys survive; stale comment corrected
- [ ] A-008 R8: `collectPaneRows` extracted, `fab pane map` byte-identical, existing panemap tests pass unmodified; `tickSnapshotRows` seam present
- [ ] A-009 R9: skill §4 rewired (diff tick, fleet frame, candidates population, step-6 bookkeeping removed, skew fallback); §5 mechanics, § Notes, §6 untouched
- [ ] A-010 R10: `_cli-fab.md` tick-start section documents `--diff` fully, in the same change

### Behavioral Correctness

- [ ] A-011 R7: after the §4 rewrite, no skill step instructs a per-tick `fab operator update` stage/agent write (the under-reporting coupling is closed)
- [ ] A-012 R9: the rewritten tick performs no per-tick `fab pane map` or `fab operator state` call (watch pass excepted)

### Scenario Coverage

- [ ] A-013 R12: every R12 test case exists and passes via the two seams; the stage-string-unchanged completion case is explicitly covered

### Edge Cases & Error Handling

- [ ] A-014 R1: empty monitored set → empty-list blocks, tick increments, snapshot subprocess skipped
- [ ] A-015 R4: pane resolving to no change (null `found`) handled as mismatch, not death
- [ ] A-016 R9: old-binary unknown-flag error routes to the documented flagless fallback line

### Code Quality

- [ ] A-017 Pattern consistency: new Go follows the operator-verb file conventions (seam vars, `mutateOperatorState`, typed sections) and the skill edits follow owner-or-pointer (no restating owned rules alongside pointers)
- [ ] A-018 No unnecessary duplication: snapshot logic reused via `collectPaneRows` — no second enumeration path; diff output structs defined once
- [ ] A-019 Canonical source only: skill edits in `src/kit/skills/`, never `.claude/skills/`
- [ ] A-020 CLI ⇒ docs + tests: `_cli-fab.md` updated and tests shipped in the same change (Constitution Additional Constraints)
- [ ] A-021 Sibling sweep done up front: aggregate-spec restatements of the tick mechanics updated per R11 (findings/ excluded)
- [ ] A-022 gofmt clean on all touched Go files

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Terminal set for `stop_stage`-null completion = {hydrate, ship, review-pr}, exactly today's §4 step 2 policy; `stop_stage` uses stage-order comparison (past-stop, or at-stop with done/skipped) | Preserves today's removal semantics; equality-only would race `finish`'s auto-activation of the next stage | S:85 R:80 A:90 D:85 |
| 2 | Confident | `review_fail` detected from stage diff alone (review→apply); no `display_state` baseline field added — `monitored` schema unchanged | Fail+reset always lands apply/active, so DisplayStage's stage carries the signal; resolves intake assumption 13's conditional as not-needed | S:70 R:80 A:85 D:75 |
| 3 | Certain | Output serialization pinned as R1's block (keys `deltas`/`candidates`/`fleet`, kind-specific fields, empty-list forms), tick/now lines retained verbatim | Intake grants exact spelling as an apply-time decision within the named shape; skill is the sole consumer | S:70 R:90 A:90 D:80 |
| 4 | Confident | Candidates include completed-but-prompting panes; deltas-first suppression stays skill policy | Plan § Edge cases states independent outputs with skill acting deltas-first | S:70 R:85 A:80 D:75 |
| 5 | Certain | Seam shape: plain `collectPaneRows` fn + `tickSnapshotRows` package var in panemap.go; `paneRow` gains internal `changeID` (not serialized) | Matches the named `rkPanesRunner`/`operatorStatePathOverride` precedent; ID join needs the folder-derived 4-char ID | S:80 R:85 A:90 D:85 |
| 6 | Confident | `fleet:` includes ALL monitored entries, with baseline-fallback rows for dead/mismatched panes | The frame must render every tracked change; dropping unjoined rows would blank frame rows exactly when attention is needed | S:65 R:80 A:85 D:75 |
| 7 | Certain | Empty monitored set skips the pane-enumeration subprocess entirely | Output is provably empty; matches the intake's cheap-no-op-tick requirement | S:80 R:90 A:90 D:85 |
| 8 | Confident | Unresolved (em-dash/null) snapshot stage: baseline untouched, no stage_advance, no completion | Cannot compare against nothing; transient unresolvable status must not fabricate events or clobber a known baseline | S:60 R:80 A:85 D:75 |
| 9 | Confident | Sweep excludes `docs/specs/findings/` (dated historical reviews) and the §1 re-derive-before-action + §2 Init orientation surfaces | Findings are point-in-time records; the on-demand pane-map surface is unchanged by design | S:70 R:85 A:85 D:80 |

9 assumptions (4 certain, 5 confident, 0 tentative).
