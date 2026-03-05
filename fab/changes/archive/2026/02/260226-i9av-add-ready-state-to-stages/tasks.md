# Tasks: Add Ready State to Stage Lifecycle

**Change**: 260226-i9av-add-ready-state-to-stages
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Schema

- [x] T001 Update `fab/.kit/schemas/workflow.yaml`: add `ready` state (id, symbol `◷`, description, terminal: false), remove `skipped` state definition. Update each stage's `allowed_states` to include `ready` and remove `skipped`. Update default transitions (add `active→ready`, `ready→done`; remove `skipped` from `active→[done,skipped]`). Update review-specific transitions to include `active→[ready,done,failed]` and `ready→[done]`. Update progression rules to include `ready` alongside `active`. Remove `skipped` from `next_stage` rule, `prerequisites` rule, and `terminal_states` rule.

## Phase 2: Core Scripts

- [x] T002 Update `fab/.kit/scripts/lib/stageman.sh` progression functions: (a) `get_progress_line()` — add `ready)` case rendering the `ready` symbol from workflow.yaml in the visual chain (visible, not hidden like pending). (b) `get_current_stage()` — update tier 1 to match both `active` and `ready` states (first stage with `state=active` or `state=ready`). (c) `get_display_stage()` — add `ready` as tier 2 (after active, before done): if a stage is `ready`, return it as the display stage with state `ready`.

- [x] T003 Update `fab/.kit/scripts/lib/stageman.sh` state mutation/validation functions: (a) `_apply_metrics_side_effect()` — add `ready)` case as no-op (preserve existing metrics, don't set `completed_at`). (b) `set_stage_state()` — ensure `ready` does NOT require a `driver` parameter (only `active` requires driver). (c) `validate_status_file()` — ensure `ready` is NOT counted in `active_count` (only `active` state increments the count).

- [x] T004 [P] Update `fab/.kit/scripts/lib/preflight.sh` — ensure `get_current_stage` call (which delegates to stageman) correctly returns `ready` stages as current stage. Verify `display_stage` and `display_state` fields output `ready` state correctly (e.g., `display_stage: spec`, `display_state: ready`).

- [x] T005 [P] Update `fab/.kit/scripts/lib/calc-score.sh` — add `--stage intake` support: when `SCORE_STAGE=intake`, read `intake.md` (instead of `spec.md`) for the Assumptions table. Use intake-stage `expected_min` thresholds from the existing `get_expected_min()` function (already has intake column). For `--check-gate --stage intake`, use fixed threshold 3.0 (not per-type dynamic thresholds).

## Phase 3: Skill Updates

- [x] T006 [P] Update `fab/.kit/skills/fab-continue.md` — add dispatch split based on `display_state`: when `active`, generate artifact then describe setting state to `ready`; when `ready`, describe advancing to next stage (`set-state <stage> done`, `set-state <next> active`). Update the dispatch table to show both paths. Ensure single-dispatch rule text covers both actions.

- [x] T007 [P] Update `fab/.kit/skills/fab-ff.md` — targeted edits (~30%): (a) Update purpose/description line. (b) Pre-flight step 2: change spec prerequisite to intake prerequisite + add intake gate (`calc-score.sh --check-gate --stage intake`). (c) Pre-flight step 3: move confidence gate to a new step after spec generation (not pre-start). (d) Insert new step: "Generate spec.md" (follow Spec Generation Procedure from `_generation.md`) + auto-clarify + spec gate check. (e) Review failure: remove "Interactive Fallback" section (lines 98-106); after 3 cycles, stop instead of falling back to interactive. (f) Update error handling table.

- [x] T008 [P] Update `fab/.kit/skills/fab-fff.md` — update contrast text with `/fab-ff` in purpose section to reflect new `/fab-ff` scope (both start from intake; difference is gates). No behavioral changes to `/fab-fff` itself.

- [x] T009 [P] Update `fab/.kit/skills/fab-clarify.md` — update stage guard: accept `ready` state (in addition to `active`) for planning stages. Change "stage MUST be `intake`, `spec`, or `tasks`" to "stage MUST be `intake`, `spec`, or `tasks` with state `active` or `ready`".

- [x] T010 [P] Update `fab/.kit/skills/_preamble.md` — (a) Update State Table if needed (existing state-keyed routing works since skills check `display_state`). (b) Update Confidence Gate Thresholds section to note `/fab-ff` intake gate (indicative score >= 3.0) alongside spec gate. (c) Update state vocabulary references if any hardcode the old 4-state set.

## Phase 4: Tests

- [x] T011 Update `src/lib/stageman/test.bats` — update existing `progress-line` tests: add `ready` state to test fixtures, verify `ready` symbol appears in output. Add new test: "progress-line: ready stage shown" with `spec: ready`.

- [x] T012 Add new tests to `src/lib/stageman/test.bats`: (a) "set-state: ready succeeds without driver" — verify `set-state <file> spec ready` exits 0 with no driver. (b) "set-state: active still requires driver" — verify `set-state <file> spec active` fails without driver. (c) "stage-metrics: set-state ready is no-op" — verify metrics preserved after ready. (d) "current-stage: returns ready stage" — verify `current-stage` returns a `ready` stage. (e) "display-stage: shows ready" — verify `display-stage` returns `stage:ready`.

- [x] T013 Update `src/lib/stageman/test.bats` — update validation tests: add test "validate-status-file: one active and one ready is valid" (active_count=1). Remove or update any tests referencing `skipped` state.

---

## Execution Order

- T001 blocks T002, T003 (schema must be updated before stageman functions)
- T002 blocks T011, T012, T013 (progression functions must work before tests)
- T003 blocks T012, T013 (mutation/validation functions must work before tests)
- T004, T005 are independent of T002-T003, can run in parallel after T001
- T006-T010 (skills) are independent of each other and of T002-T005 (markdown edits, no runtime dependency)
- T011-T013 depend on T001-T003 (tests verify script behavior)
