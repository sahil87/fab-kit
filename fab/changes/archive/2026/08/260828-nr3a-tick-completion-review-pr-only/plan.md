# Plan: Tick-Start Completion Fires Only at the Pipeline Terminus

**Change**: 260828-nr3a-tick-completion-review-pr-only
**Intake**: `intake.md`

## Requirements

### Runtime: `fab operator tick-start --diff` completion predicate

#### R1: Null-`stop_stage` completion is a terminus display-state check
`tickCompleted` MUST, when the monitored entry has no `stop_stage`, return true only when the snapshot stage is `review-pr` AND `display_state` is `done` or `skipped`. The `{hydrate, ship, review-pr}` terminal set (`tickTerminalStages`) MUST be removed; bare stage membership MUST NOT complete an entry.

- **GIVEN** a monitored entry with `stop_stage: null`
- **WHEN** the snapshot shows `hydrate/active`, `hydrate/done`, `ship/active`, or `review-pr/active`
- **THEN** no `completion` delta is emitted for that entry
- **AND** when the snapshot shows `review-pr/done` or `review-pr/skipped`, a `completion` delta is emitted (level-triggered, every tick until removed)

#### R2: `stop_stage`-set branch unchanged
The `stop_stage`-set branch of `tickCompleted` (past the stop in stage order ⇒ complete; at the stop ⇒ complete only with `display_state` done/skipped) SHALL be behaviorally identical to today; the done/skipped test SHOULD be shared with R1 via one small helper rather than duplicated.

- **GIVEN** an entry with `stop_stage: review`
- **WHEN** the snapshot shows `hydrate/active` (past) or `review/done` (at, finished)
- **THEN** a `completion` delta is emitted
- **AND** `review/active` (at, unfinished) emits none

#### R3: Tests encode the corrected predicate
`TestOperatorTickDiff_CompletionPredicateBranches` MUST assert `ship/active` with null `stop_stage` emits NO completion, and MUST cover null-`stop_stage` rows `hydrate/active`, `hydrate/done`, `review-pr/active` (none) and `review-pr/done`, `review-pr/skipped` (completion).

- **GIVEN** the rewritten test
- **WHEN** `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTickDiff|TestOperatorTickStart'` runs
- **THEN** all tests pass, and reverting R1 makes the `ship/active` row fail

### Skills: completion contract text

#### R4: `_cli-fab.md` and `fab-operator.md` state the new completion rule
Every statement that a null-`stop_stage` entry completes at "a terminal stage (hydrate, ship, review-pr)" MUST be replaced by "the pipeline terminus — `review-pr` with `display_state` done/skipped". `fab-operator.md` § Stop stage MUST add that spawns which deliberately park earlier (e.g. `/fab-ff`, which stops after hydrate) enroll with `--stop-stage hydrate`, otherwise the entry never completes. The phrase class (`terminal stage`, `terminal set`, `hydrate, ship, review-pr`, `hydrate/ship/review-pr`, `terminal/stop stage`) MUST be swept across `src/kit/skills/*.md`.

- **GIVEN** the edited skills
- **WHEN** `grep -rn -i 'terminal stage\|terminal set\|hydrate, ship, review-pr\|hydrate/ship/review-pr' src/kit/skills/` runs
- **THEN** no occurrence still describes hydrate or ship as completing a null-`stop_stage` entry

#### R5: One canonical "Dependency satisfied" definition in `fab-operator.md`
§ Dependency Resolution MUST open with a **Dependency satisfied** paragraph: a `depends_on` entry is satisfied when the dependency's pipeline has completed (its `completion` delta — `review-pr` done/skipped with null `stop_stage`, or at/past its `stop_stage`) AND, for a same-repo dependency with null `stop_stage`, its PR exists (`gh pr view <dep-branch> --json url` succeeds). Enrollment, a `branch_map` entry, or a minted branch is NOT satisfaction. An unsatisfied dependency holds the spawn in both tiers, re-checked each tick, logging `"{change}: waiting on dependency {dep} ({dep.repo}) to complete."`. Consumers MUST point at the definition rather than restate it: the cross-repo bullet and § Cross-repo resolution, a new same-repo readiness gate (step 0.5, before branch lookup; the `stacked-prs` variant inherits it), autopilot per-change loop steps 5–6, the watches cross-repo barrier line, and the state-file `depends_on` comment.

- **GIVEN** a same-repo `depends_on` whose dependency is still in the monitored set and not complete
- **WHEN** the operator reaches Same-repo resolution
- **THEN** the spawn is held (no branch lookup, no cherry-pick) and the wait is logged
- **AND** once the dep's `completion` delta is observed and its PR exists, resolution proceeds to branch lookup

### Memory

#### R6: `docs/memory/runtime/operator.md` reflects present truth
Hydrate MUST rewrite the removal-trigger, tick step 2 "Pipeline completion", removal-path Design Decision, cross-repo barrier, and dependency-resolution Design Decision statements to the new rules (R1, R4, R5), and add a four-field Design Decision entry for the terminus-only completion + dependency-satisfied definition.

- **GIVEN** the hydrated memory
- **WHEN** `grep -n -i 'terminal stage\|hydrate, ship, or review-pr' docs/memory/runtime/operator.md` runs
- **THEN** no occurrence describes hydrate/ship as a null-`stop_stage` completion trigger

### Non-Goals
- Any heuristic that completes a parked `/fab-ff` run (hydrate/done, ship pending) without a `stop_stage` — `stop_stage` is the mechanism for early parking.
- Changing `tick-start --diff` output shape, `fab operator` verbs, or autopilot Go code.

### Design Decisions

#### Completion terminus is review-pr done/skipped, not a terminal stage set
**Decision**: With `stop_stage: null`, `tickCompleted` is `stage == review-pr && display_state ∈ {done, skipped}`; the `{hydrate, ship, review-pr}` set is deleted.
**Why**: The set fired on stage *entry* under `/fab-fff` — `hydrate/active` and `ship/active` emitted `completion` every tick (100% reproducible 2026-08-28), driving premature `fab operator remove`. Requiring done/skipped mirrors the at-the-stop branch and keeps `review-pr/active` (awaiting Copilot) incomplete.
**Rejected**: Keeping `{hydrate, ship}` with an added `display_state` check — a `hydrate/done` snapshot is a transient window in `/fab-fff` (finish auto-activates ship), so it still races; `/fab-ff` runs express early parking via `--stop-stage hydrate`.
*Introduced by*: 260828-nr3a-tick-completion-review-pr-only

#### Dependency satisfied = pipeline completed + PR exists, never branch minted
**Decision**: One definition in `fab-operator.md` § Dependency Resolution; both tiers, autopilot, and watches point at it. Same-repo gains a readiness gate that holds the spawn.
**Why**: `branch_map` is written by `enroll` at spawn, so "branch resolves" was true the instant a dep's agent started — a same-repo `depends_on` on a running dep cherry-picked an unfinished branch. Autopilot was protected only by the (buggy) `completion` signal. PR-exists is the cheapest observable proof the branch is pushed and stable for cherry-pick / stacked retarget.
**Rejected**: Branch resolvable ⇒ satisfied (the implicit status quo).
*Introduced by*: 260828-nr3a-tick-completion-review-pr-only

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/operator_tick_start.go`: delete `tickTerminalStages`, add `tickTerminusStage = "review-pr"` const and a `stageFinished(displayState string) bool` helper; rewrite `tickCompleted`'s null-`stop_stage` branch to `stage == tickTerminusStage && stageFinished(displayState)` and reuse the helper in the at-the-stop branch; update the doc comments <!-- R1 R2 -->
- [x] T002 In `src/go/fab/cmd/fab/operator_tick_diff_test.go` `TestOperatorTickDiff_CompletionPredicateBranches`: flip `s005` (`ship/active`) to expect no completion; add null-`stop_stage` rows for `hydrate/active`, `hydrate/done`, `review-pr/active` (none) and `review-pr/done`, `review-pr/skipped` (completion); run `go test ./src/go/fab/cmd/fab/ -run 'TestOperatorTickDiff|TestOperatorTickStart'` <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T003 [P] In `src/kit/skills/_cli-fab.md` § fab operator tick-start "Detection semantics": replace the `{hydrate, ship, review-pr}` terminal-set sentence with the review-pr done/skipped terminus rule <!-- R4 -->
- [x] T004 [P] In `src/kit/skills/fab-operator.md`: reword § Stop stage (+ `--stop-stage hydrate` rule for parked spawns), § Removal, tick step 1 `completion` bullet, tick step 5, status legend `complete` row; add the **Dependency satisfied** paragraph at the top of § Dependency Resolution and repoint the cross-repo bullet, § Cross-repo resolution, same-repo step 0.5 readiness gate, autopilot loop steps 5–6, watches barrier line, and the state-file `depends_on` comment <!-- R4 R5 -->
- [x] T005 Sweep `src/kit/skills/*.md` for the phrase class (`terminal stage`, `terminal set`, `hydrate, ship, review-pr`, `hydrate/ship/review-pr`, `terminal/stop stage`, `reaches its stop_stage`) and fix every remaining stale claim; run `fab sync` so deployed copies match <!-- R4 R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `tickCompleted` with nil `StopStage` returns true only for `review-pr` + done/skipped; `tickTerminalStages` no longer exists in the package
- [x] A-002 R2: The `stop_stage`-set branch's behavior is unchanged (rows `s001`–`s004` of the predicate test still pass unmodified)
- [x] A-003 R3: The predicate test covers all six null-`stop_stage` rows from R3 and passes
- [x] A-004 R4: `_cli-fab.md` and `fab-operator.md` describe null-`stop_stage` completion as review-pr done/skipped; § Stop stage carries the `--stop-stage hydrate` parking rule
- [x] A-005 R5: `fab-operator.md` § Dependency Resolution opens with the **Dependency satisfied** definition and every listed consumer points at it (no restatement)

### Behavioral Correctness

- [x] A-006 R1: Reverting T001 makes the `ship/active` row of the predicate test fail (the test is a real regression guard)
- [x] A-007 R5: Same-repo resolution has a readiness gate that holds the spawn before branch lookup when the dep is still monitored and not complete

### Removal Verification

- [x] A-008 R4: `grep -rn -i 'terminal stage\|terminal set\|hydrate, ship, review-pr\|hydrate/ship/review-pr\|terminal/stop stage' src/kit/skills/` yields no occurrence claiming hydrate/ship complete a null-`stop_stage` entry

### Scenario Coverage

- [x] A-009 R1: `TestOperatorTickDiff_AllEventKinds` and `TestOperatorTickDiff_LevelTriggeredReEmitUntilRemove` (both seed `review-pr/done`) still pass

### Edge Cases & Error Handling

- [x] A-010 R1: `review-pr/skipped` (review-pr disabled at ship) emits completion; `review-pr/active` does not

### Code Quality

- [x] A-011 Pattern consistency: helper/const naming follows the `tick*` prefix convention of the file
- [x] A-012 No unnecessary duplication: the done/skipped check appears once (helper), not twice
- [x] A-013 Owner-or-pointer: `fab-operator.md` consumers point at § Dependency satisfied rather than restating it; `_cli-fab.md` owns the CLI detection-semantics statement
- [x] A-014 Sibling sweep: skill phrase class swept up front (T005), including `_cli-fab.md` + `fab-operator.md` + any other `src/kit/skills/*.md` hit
- [x] A-015 CLI change discipline: Go change ships with test updates and the `_cli-fab.md` semantics update (constitution Additional Constraints)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only `TestOperatorTickDiff_CompletionPredicateBranches` needs rewriting; `AllEventKinds` and `LevelTriggered…` already seed `review-pr/done` | Read from the test file | S:90 R:95 A:95 D:95 |
| 2 | Confident | PR-exists check applies only to same-repo deps with null `stop_stage`; parked deps (explicit `stop_stage`) are satisfied by their completion delta alone | A dep parked at hydrate has no PR by design; its local branch is still cherry-pickable | S:60 R:85 A:75 D:65 |

2 assumptions (1 certain, 1 confident, 0 tentative).
