# Plan: true_impact Recorded as All-Zeros — Recompute at Ship

**Change**: 260702-h65d-true-impact-recompute-at-ship
**Intake**: `intake.md`

## Requirements

### Status: `true_impact` Ship-Time Recompute

#### R1: `WriteTrueImpact` MUST accept the `ship` stage
`WriteTrueImpact` (`src/go/fab/internal/status/true_impact.go`) SHALL compute and write the `true_impact` block when invoked with `stage == "ship"`, in addition to the existing `apply` and `hydrate` stages. Any other stage MUST remain a no-op returning `nil` (unchanged best-effort posture).

- **GIVEN** a change branch whose commits exist on `HEAD` (post-`/git-pr` Step 3a commit + 3b push)
- **WHEN** `status.Finish` invokes `WriteTrueImpact(..., "ship")` at ship-finish (`/git-pr` Step 4b)
- **THEN** the block is recomputed against `merge-base(origin/main, HEAD)...HEAD` and written with `computed_at_stage: ship`, superseding any earlier apply/hydrate value in place
- **AND** stages other than `apply`/`hydrate`/`ship` (e.g. `review`, `review-pr`) still short-circuit to a no-op

#### R2: The `ship` recompute reflects the real PR diff
When commits exist on the branch, the ship-time block SHALL carry the true (non-zero) line counts of the change, not the all-zeros produced when `HEAD == merge-base` at apply/hydrate finish in the standard commit-nothing-until-ship pipeline.

- **GIVEN** apply-finish and hydrate-finish run while the working tree is clean and `HEAD == merge-base` (three-dot diff is empty → 0/0/0)
- **WHEN** a commit lands on the branch and ship-finish then runs
- **THEN** the block reports the real added/deleted/net counts, with `excluding`/`tests` sub-blocks present per the existing lazy-omit rules
- **AND** the earlier apply/hydrate zeros are overwritten (single overwrite-on-write value, not a history)

#### R3: Documentation MUST reflect the widened stage set
Every live (non-archived, non-generated) prose/code claim that the `true_impact` block is written "at apply-finish and hydrate-finish" (or "for stages apply and hydrate only", "apply/hydrate write path", etc.) SHALL be updated to include `ship`. `computed_at_stage` documentation SHALL list `apply`, `hydrate`, or `ship`.

- **GIVEN** the stale stage-set claim restated across code comments, memory files, specs, and the kit template comment + `_cli-fab.md` prose
- **WHEN** the sweep runs
- **THEN** each live occurrence names `apply`/`hydrate`/`ship` (or is reworded to drop the apply-only implication), and no live file still asserts the two-stage-only set
- **AND** archived changes, generated `log.md`/`log.seed.md`, `docs/findings/`, and per-change `.status.yaml` files are left untouched (historical/generated records)

### Non-Goals
- `internal/impact/` — the shortstat math is correct; not touched.
- `internal/prmeta/` and `fab pr-meta`/`fab impact` CLI — recompute live already; no signature change.
- `/git-pr` skill flow — already runs `fab status finish ship` post-commit; no behavior change.
- Migration or backfill of historical all-zero blocks — merged branches are gone; blocks self-correct on next ship. No schema change (the block shape is unchanged; `ship` is a new value for the existing free-string `computed_at_stage`).
- Adding `review-pr`-finish recompute — deferred as a trivial future one-line widening (intake assumption #6).

### Design Decisions
1. **Widen the stage gate rather than change the apply/hydrate measurement semantics**: add `ship` to the accepted set in `WriteTrueImpact`. — *Why*: ship-finish is the earliest pipeline point where the branch tip exists (after `/git-pr` commits+pushes), so `base...HEAD` finally measures the true PR diff; no new call site is needed (`status.Finish` already passes the stage on every finish, and `/git-pr` Step 4c commits the resulting `.status.yaml` update). — *Rejected*: (a) making apply/hydrate include the working tree (two-dot / `git diff <base>`) — changes the measurement's meaning, diverges from the PR diff, and goes stale on the next edit; (b) reading `fab pr-meta`'s live computation back into the block — inverts the data flow and couples the block to PR creation.
2. **Keep the apply/hydrate-finish writes**: they are harmless in the standard flow (zeros until ship supersedes via `computed_at_stage`) and produce real values in non-standard flows where commits exist before ship (adopted off-pipeline changes, manual mid-apply commits).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Widen the stage gate in `src/go/fab/internal/status/true_impact.go` (~line 23) from `if stage != "apply" && stage != "hydrate"` to `if stage != "apply" && stage != "hydrate" && stage != "ship"`, and update the function doc comment (lines 19–20: "Stage MUST be one of: apply, hydrate … apply-finish and hydrate-finish hooks compute the block per spec assumption #16") to name `apply`, `hydrate`, `ship` and explain the ship recompute rationale <!-- R1 -->
- [x] T002 Update the stale comment at `src/go/fab/internal/status/status.go:195` ("Compute and write true_impact for apply/hydrate finish (best-effort).") to include ship <!-- R3 -->

### Phase 2: Tests

- [x] T003 Extend `src/go/fab/internal/status/true_impact_test.go` with a ship-recompute test reproducing the bug scenario: apply-finish then hydrate-finish while `HEAD == merge-base` (clean tree → zeros/no meaningful diff), then a new commit on the branch, then ship-finish → non-zero block with `computed_at_stage: ship` superseding the earlier write <!-- R1 R2 -->

### Phase 3: Documentation Sweep

- [x] T004 [P] Update `docs/memory/pipeline/schemas.md` § `true_impact` Block: line ~204 ("at apply-finish and hydrate-finish" → include ship + note ship is the authoritative write in the standard pipeline / apply-hydrate writes are zeros until commits exist / the run-kit all-zeros motivation), line ~228 (`computed_at_stage` values `apply` or `hydrate` → add `ship`), line ~236 ("invokes the helper for stages `apply` and `hydrate` only" → `apply`, `hydrate`, and `ship`) <!-- R3 -->
- [x] T005 [P] Update `docs/memory/_shared/configuration.md:78` — "the apply/hydrate `true_impact` write path" → "the apply/hydrate/ship `true_impact` write path" <!-- R3 -->
- [x] T006 [P] Update `docs/specs/templates.md`: line 59 (the `# true_impact: lazily created on first apply-finish` comment mirror) and line 73 ("written lazily by the apply-finish and hydrate-finish hooks") to reflect the widened stage set <!-- R3 -->
- [x] T007 [P] Update the `# true_impact: lazily created on first apply-finish (no placeholder here).` comment in `src/kit/templates/status.yaml:28` (canonical kit source) to drop the apply-only implication <!-- R3 -->
- [x] T008 [P] Update `src/kit/skills/_cli-fab.md:444` prose ("the apply-finish + hydrate-finish hooks") to include ship — canonical kit prose that restates the stage set joins the sweep class per the intake's own qualifier <!-- R3 -->
- [x] T009 [P] Update the lazy-creation comment in `src/go/fab/internal/statusfile/statusfile.go:90` ("Created lazily on first apply-finish") to drop the apply-only implication, for whole-class consistency <!-- R3 -->
- [x] T010 Verify the sweep with a repo-wide grep (`apply-finish and hydrate-finish` / `apply/hydrate` / `apply and hydrate` over live files) — confirm no live (non-archive, non-generated, non-findings) file still asserts the two-stage-only set <!-- R3 -->

### Phase 4: Validation

- [x] T011 Run `go test ./internal/status/...` from `src/go/fab`; widen to `go build ./...` / `go test ./...` if the status package changes ripple <!-- R1 R2 -->

## Execution Order

- T001 → T003 (test targets the widened gate) → T011
- T002 and T004–T010 are independent of each other and of the code/test path (documentation + non-behavioral comments)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `WriteTrueImpact` computes and writes the block for `stage == "ship"`; `apply` and `hydrate` still write; all other stages remain a no-op returning nil
- [x] A-002 R3: Every live occurrence of the "apply-finish and hydrate-finish" / "apply/hydrate only" stage-set claim (code comments, `docs/memory/pipeline/schemas.md`, `docs/memory/_shared/configuration.md`, `docs/specs/templates.md`, `src/kit/templates/status.yaml`, `src/kit/skills/_cli-fab.md`, `statusfile.go`) names `apply`/`hydrate`/`ship` or is reworded to drop the apply-only implication

### Behavioral Correctness

- [x] A-003 R2: In the repro scenario (apply/hydrate finish at `HEAD == merge-base` → zeros; commit; ship finish), the ship-time block carries the real non-zero counts with `computed_at_stage: ship`, overwriting the earlier value
- [x] A-004 R1: A non-apply/hydrate/ship stage (e.g. `review`) leaves `.status.yaml` free of a `true_impact` block (existing `TestWriteTrueImpact_NonApplyStageIsNoOp` still passes)

### Scenario Coverage

- [x] A-005 R1 R2: `go test ./internal/status/...` passes, including the new ship-recompute test and all pre-existing `true_impact_test.go` cases

### Edge Cases & Error Handling

- [x] A-006 R1: The ship path preserves the best-effort posture — a merge-base/git failure at ship-finish emits a one-line stderr warning and returns nil without failing the stage transition (shared code path, unchanged)

### Code Quality

- [x] A-007 Pattern consistency: The gate edit is a minimal single-condition addition; doc/comment wording matches the surrounding style; the new test follows the existing `setupGitRepo`/`withCwd`/reload-and-assert pattern in `true_impact_test.go`
- [x] A-008 No unnecessary duplication: The ship recompute reuses the existing `WriteTrueImpact` body and `impact.ComputeForRepo` — no new call site, no duplicated shortstat logic

### Documentation Accuracy

- [x] A-009 documentation_accuracy: No live file still claims the block is written at apply/hydrate only; `computed_at_stage` docs list `apply`/`hydrate`/`ship`; archived + generated (`log.md`/`log.seed.md`) + `docs/findings/` records are correctly left untouched

### Cross-References

- [x] A-010 cross_references: The `docs/memory/pipeline/schemas.md` ↔ `docs/memory/_shared/configuration.md` cross-links and the `_cli-fab.md` consumer list remain consistent after the sweep (no dangling or contradictory stage-set claims across the linked files)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix = add `ship` to `WriteTrueImpact`'s stage gate (one condition); recompute fires through the existing `status.Finish` → `/git-pr` Step 4b path, post-commit | User-chosen framing, verified in source (`status.go:196`, gate at `true_impact.go:23`); no new call site | S:90 R:85 A:95 D:90 |
| 2 | Certain | Keep apply/hydrate-finish writes; ship supersedes via `computed_at_stage` (overwrite-on-write) | Harmless zeros in standard flow, real values for early-commit flows (adopt/manual commits); intake assumption #3 | S:85 R:85 A:90 D:85 |
| 3 | Confident | Include `src/kit/skills/_cli-fab.md:444` and `statusfile.go:90` in the sweep even though the intake said "no `_cli-fab.md` change expected" | The intake's own qualifier folds any prose restating the stage set into the sweep class; both are live canonical sources asserting the two-stage set. No command *signature* changes (the intake's actual premise holds) | S:70 R:90 A:80 D:75 |
| 4 | Confident | Reword the "lazily created on first apply-finish" comments (kit template, `docs/specs/templates.md:59`, `statusfile.go:90`) to drop the apply-only phrasing rather than list all three stages | These are lazy-creation claims, not stage-set enumerations; apply-finish is still the first write in the standard flow (just zeros), so "first stage-finish" is the accurate, minimal wording | S:65 R:90 A:80 D:70 |
| 5 | Confident | Exclude archived changes, generated `log.md`/`log.seed.md`, `docs/findings/`, and per-change `.status.yaml` from the sweep | These are historical/generated records, not live design documentation; editing them would be churn with no correctness value and would fight the memory-index generator | S:75 R:85 A:85 D:80 |
| 6 | Confident | New test simulates ship-finish by committing after apply/hydrate, using the existing `setupGitRepo` two-commit fixture as the "branch tip exists" state | The fixture already diverges from `origin/main` with a real commit, so calling `WriteTrueImpact(..., "ship")` against it yields non-zero counts — directly modeling the post-`/git-pr` branch-tip state | S:70 R:85 A:85 D:75 |

6 assumptions (2 certain, 4 confident, 0 tentative).
