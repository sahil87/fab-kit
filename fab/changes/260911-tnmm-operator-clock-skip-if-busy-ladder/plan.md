# Plan: Operator Clock — Single `skip-if-busy` Deliver Policy, 3m→24m Backoff Ladder, Debounce Doc Fix

**Change**: 260911-tnmm-operator-clock-skip-if-busy-ladder
**Intake**: `intake.md`

## Requirements

### Runtime: Operator clock derive (Go)

#### R1: One deliver policy on every derived branch
`deriveOperatorSchedule` in `src/go/fab/cmd/fab/operator_clock.go` MUST return `deliver: skip-if-busy` on every branch — backoff (pane/none-only sets), `every`, and `idle-every` — and the `clock_override` writer in `operator_track.go` MUST use the same single constant. No per-branch deliver constant SHALL remain: `operatorDeliverImmediate` is deleted and `operatorDeliverSkipIfBusy` is renamed to `operatorDeliver`.

- **GIVEN** a tracked set holding only `pane`/`none` items whose state is not `done`
- **WHEN** the schedule reconcile derives the clock
- **THEN** the derived value is `{kind: backoff, deliver: skip-if-busy}`
- **AND** a set holding any `shell`/`agent` item still derives `every`/`idle-every` with `deliver: skip-if-busy`

#### R2: Backoff bounds are 3m → 24m
`operatorBackoffMin` MUST be `"3m"` and `operatorBackoffMax` MUST be `"24m"`. The reconcile's emitted argv on the backoff branch SHALL be `rk cron edit <id> --backoff --min 3m --max 24m --deliver skip-if-busy`, and `cronRowMatches` SHALL treat a live row on exactly those bounds and policy as equal (no edit).

- **GIVEN** a live entry on `backoff 1m0s→30m0s / immediate` and a pane-only tracked set
- **WHEN** any `track` verb saves or `tick-start --diff` finishes
- **THEN** exactly one `rk cron edit <id> --backoff --min 3m --max 24m --deliver skip-if-busy` is issued
- **AND** a subsequent reconcile against a row already on `3m0s→24m0s / skip-if-busy` issues nothing

#### R3: Tests pin the derived values
`operator_clock_test.go` fixtures that represent "the entry already on the derived pane/none schedule" MUST carry `3m0s`/`24m0s`/`skip-if-busy`, and every expected backoff argv MUST be the R2 argv. Tests conform to the requirement, never the reverse (Constitution VII).

- **GIVEN** the updated fixtures
- **WHEN** `go test ./cmd/fab` runs from `src/go/fab`
- **THEN** every operator clock test passes, including "equal row issues nothing" and the drifted-row shell-item cases

### Kit: deployed skill prose

#### R4: Kit skills quote the steady-state entry and the uniform policy
`src/kit/skills/fab-operator.md` MUST show the ready-line example as `backoff 3m→24m · skip-if-busy`, the §4 entry-shape block as `min: 3m, max: 24m`, `debounce: 60s`, `deliver: skip-if-busy` with its lead-in reframed as the entry's **steady-state** shape (seeded by `rk operator`, `schedule`/`deliver` converged by fab's reconcile), and the Ownership paragraph as `3m`→`24m` with deliver `skip-if-busy` on every branch. `src/kit/skills/_cli-fab-operator.md` MUST carry the pane/none derive row as `--backoff --min 3m --max 24m` / `skip-if-busy`, state the uniform policy once in prose, and recompute the full-frame cadence claim (≈ every 4th tick at the 3m floor). The `1m` `--check-every` floors near lines ~220/~227 MUST NOT change. Deployed content MUST NOT cite fab-kit-only paths (Constitution V).

- **GIVEN** the two kit files after apply
- **WHEN** grepping them for `1m→30m`, `min: 60s`, `debounce: 10s`, `deliver: immediate`, `--min 1m --max 30m`, `every 10th tick`
- **THEN** there are zero hits
- **AND** the `--check-every` floor sentences still read `1m`

### Backlog

#### R5: Forward instruction carries the new ladder
`fab/backlog.md` row `[bjrk]` MUST quote `--min 3m --max 24m` in its future seed argv; nothing else in the row changes.

- **GIVEN** the backlog row after apply
- **WHEN** it is read by a future picker
- **THEN** its seed argv does not re-introduce the retired ladder

### Non-Goals
- Moving the entry's seeding into fab — backlog `[bjrk]`, blocked on run-kit `[ntde]` (`rk cron add` has no `--wake-on` flags)
- Any run-kit change; the `rk operator` seed default still writes `1m`/`30m`/`immediate` until `[ntde]` flips it
- §2 Init step 4's "run rk operator to seed it" STOP; any `rk cron add` call; skill-composed cron argv
- Sweeping dated artifacts: `docs/wiki/operator-tick-anatomy.html`, `fab/plans/sahil/26-09-11-operator-generic-tracking.md`, `fab/changes/**`
- A migration file — no user data is restructured; the one-time `rk cron edit` on first reconcile is the reconcile working
- A `docs/specs/operator.md` version row — cadence values are not a skill-shape version

### Design Decisions

#### Uniform `skip-if-busy` Deliver Policy, Backoff as Fallback Poll
**Decision**: the deliver policy is `skip-if-busy` for every derived branch (backoff, every, idle-every) and for the bounded `clock_override`; the backoff branch is a fallback poll behind `wake_on: agent-state-change` and runs `3m → 24m`.
**Why**: `immediate` injects a tick mid-turn (queued as a mid-turn message on Claude, less predictable on other providers), and a per-branch table changed the operator's delivery semantics silently whenever a shell/agent item joined or left the tracked set. A 1-minute floor buys almost no latency over the wake event and mostly buys token spend; a 30-minute ceiling exceeds the operator's own 10-minute full-frame window. Accepted cost: under `skip-if-busy` a tick landing mid-turn is dropped, so on the backoff branch a drop at a long rung delays the fallback poll by up to that rung (≤ 24m); `wake_on` fires independently, so a real fleet event never waits for the rung.
**Rejected**: `when-idle` on the backoff branch (holds the fire until idle — no tick lost, no rung paid twice — rejected for policy uniformity: a third policy on one branch rebuilds the two-way table under another name). Keeping `immediate` on the backoff branch (mid-turn injection; the two-way table).
*Introduced by*: 260911-tnmm-operator-clock-skip-if-busy-ladder

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/operator_clock.go`: set `operatorBackoffMin = "3m"`, `operatorBackoffMax = "24m"`; delete `operatorDeliverImmediate`; rename `operatorDeliverSkipIfBusy` → `operatorDeliver` with the single-policy comment; make all three `deriveOperatorSchedule` returns use `operatorDeliver`; rewrite its doc comment (schedule kind varies, deliver uniform). Rename the one call site in `src/go/fab/cmd/fab/operator_track.go` (`clock_override` writer, ~line 1012). `gofmt -l` clean. <!-- R1, R2 -->
- [x] T002 In `src/go/fab/cmd/fab/operator_clock_test.go`: fixtures `cronListBackoffJSON`/`cronListBackoffMutedJSON` → `"min":"3m0s","max":"24m0s"`, `"deliver":"skip-if-busy"`; expected argv at ~273 and ~406 → `--min 3m --max 24m --deliver skip-if-busy`; comments at ~259 and ~404 → `backoff/skip-if-busy`. Run `go test ./cmd/fab -run 'Clock|Schedule|Reconcile|TickStart'` then `go test ./cmd/fab` from `src/go/fab`. <!-- R3 -->

### Phase 4: Polish

- [x] T003 [P] In `src/kit/skills/fab-operator.md`: §2 step 5 ready-line example (~line 120) → `backoff 3m→24m · skip-if-busy` / `… · skip-if-busy · muted until 14:30`; §4 entry-shape block (~176–180) → `min: 3m, max: 24m`, `debounce: 60s`, `deliver: skip-if-busy` with the lead-in reframed to the steady-state shape (rk seeds; fab's reconcile converges `schedule`/`deliver`); §4 Ownership paragraph (~186) → `3m`→`24m`, deliver `skip-if-busy` on every branch. Verify §5/§6 and frame examples carry no stray literal. <!-- R4 -->
- [x] T004 [P] In `src/kit/skills/_cli-fab-operator.md`: derive table pane/none row (~166) → `--backoff --min 3m --max 24m` / `skip-if-busy`; add one sentence stating deliver is `skip-if-busy` on every derived branch and every override; line ~153 → "At the 3m backoff floor that is roughly every 4th tick". Leave the `--check-every` `1m` floors (~220/~227) untouched. <!-- R4 -->
- [x] T005 [P] In `fab/backlog.md` row `[bjrk]`: `--min 1m --max 30m` → `--min 3m --max 24m`. Then repo-wide grep (excluding `.claude/`, `.agents/`, `fab/changes/`, `docs/wiki/`, `fab/plans/`) for `1m→30m`, `--min 1m --max 30m`, `min: 60s`, `debounce: 10s`, `deliver: immediate`, `deliver immediate`, `every 10th tick` — the only remaining hit may be `docs/memory/runtime/operator.md` (hydrate's). Sweep found and fixed one more: the same ≈every-10th-tick arithmetic in a comment at `src/go/fab/cmd/fab/operator_tick_start.go:137`. <!-- R5, R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `deriveOperatorSchedule` returns `deliver: skip-if-busy` on the backoff, `every`, and `idle-every` branches; `operatorDeliverImmediate` no longer exists; `operator_track.go`'s override writer uses `operatorDeliver`
- [x] A-002 R2: `operatorBackoffMin`/`Max` are `"3m"`/`"24m"` and the backoff-branch argv is `--backoff --min 3m --max 24m --deliver skip-if-busy`
- [x] A-003 R3: `operator_clock_test.go` fixtures and expected argv carry the new values; `go test ./cmd/fab` passes
- [x] A-004 R4: `fab-operator.md` ready-line example, §4 entry block (values + steady-state lead-in), and Ownership paragraph carry `3m→24m`, `60s`, `skip-if-busy`
- [x] A-005 R4: `_cli-fab-operator.md` derive table row, uniform-policy sentence, and the ≈ every-4th-tick claim are updated
- [x] A-006 R5: backlog row `[bjrk]` quotes `--min 3m --max 24m`

### Behavioral Correctness

- [x] A-007 R2: a live row already on `3m0s→24m0s / skip-if-busy` with a pane-only set issues no `rk cron edit` (durations compare as durations)
- [x] A-008 R1: the `clock_override` path still writes `deliver: skip-if-busy` and the derive's override short-circuit passes it through unchanged

### Scenario Coverage

- [x] A-009 R2: a test case exercises the drifted row (`1m0s→30m0s / immediate`, or a shell-item `every` row) against the pane-only derived value and expects exactly one edit with the R2 argv

### Edge Cases & Error Handling

- [x] A-010 R4: the `--check-every` validation floor (`1m`) sentences in `_cli-fab-operator.md` are unchanged
- [x] A-011 R4: `docs/wiki/operator-tick-anatomy.html`, `fab/plans/sahil/26-09-11-operator-generic-tracking.md`, and `fab/changes/**` are not modified
- [x] A-012 R4: no deployed kit file cites a fab-kit-only path (`docs/specs/`, `docs/memory/`, `docs/site/`, `src/go/`) as a result of this change

### Code Quality

- [x] A-013 Pattern consistency: the renamed constant follows the surrounding `operator*` naming and the comment style of its const block
- [x] A-014 No unnecessary duplication: one deliver constant, referenced from both files; no literal `"skip-if-busy"` string is introduced outside the const block
- [x] A-015 No magic strings: bounds and policy remain named constants; no inline `"3m"`/`"24m"` in derive or reconcile code
- [x] A-016 Canonical sources only: edits land in `src/kit/skills/*.md`, never in `.agents/skills/` or `.claude/skills/`
- [x] A-017 CLI change ⇒ reference + tests: the emitted `rk cron edit` argv changed, so `_cli-fab-operator.md` (owner of the `fab operator` family) and the Go tests change in the same commit
- [x] A-018 Owner-or-pointer: `fab-operator.md` §4 keeps pointing at `_cli-fab-operator.md` for the derive table and does not restate it as a second table
- [x] A-019 **N/A**: hydrate-owned, verified post-hydrate — sibling sweep: the memory file documenting the clock (`docs/memory/runtime/operator.md`) is updated by hydrate in this change — clock paragraph, the "Binary Derives the Cron Schedule From the Tracked Set" decision, and the 5ubz decision's quoted ladder figures
- [x] A-020 Test strategy: touched `.go` files are `gofmt` clean and the `./cmd/fab` package tests pass before apply finishes

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the change made exactly one symbol redundant (`operatorDeliverImmediate`, deleted in the same diff); no other existing code, branch, or config became unused

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Memory edits (`docs/memory/runtime/operator.md`) are performed by hydrate, not by an apply task; the plan records them under A-019 so review checks the sweep at hydrate | Pipeline convention: apply touches code and kit prose, hydrate owns `docs/memory/`; both run inline in the light lane by the same author | S:85 R:90 A:90 D:85 |
| 2 | Certain | Five tasks ⇒ LIGHT lane (apply, hydrate, ship, review-pr inline; review dispatched) | Task count ≤ 5 per `_pipeline.md` fork rule; the change is one Go file + test + two kit files + one backlog row | S:90 R:95 A:90 D:90 |
| 3 | Confident | The `_cli-fab-operator.md` derive table keeps its Derived-deliver column (now uniform) and gains one prose sentence, rather than dropping the column | Row-level legibility; the intake permits either as long as no per-branch reading is reintroduced | S:65 R:95 A:85 D:75 |
| 4 | Confident | Tests scoped first with `-run 'Clock|Schedule|Reconcile|TickStart'`, then the whole `./cmd/fab` package; no wider run | Code-quality test strategy: scope to the affected package; the change is confined to `cmd/fab` | S:70 R:95 A:90 D:80 |

4 assumptions (2 certain, 2 confident, 0 tentative).
