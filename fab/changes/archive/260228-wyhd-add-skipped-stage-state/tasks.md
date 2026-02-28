# Tasks: Add "skipped" Stage State (v2 — expand skip from-states)

**Change**: 260228-wyhd-add-skipped-stage-state
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Schema Change

- [x] T001 Update `skip` transition in `fab/.kit/schemas/workflow.yaml` — change `from: [pending]` to `from: [pending, active]`

## Phase 2: Implementation

- [x] T002 Update `_scripts.md` — change skip row description from `pending → skipped` to `{pending,active} → skipped`
- [x] T003 Update help text in `fab/.kit/scripts/lib/statusman.sh` — change skip description from `pending → skipped` to `{pending,active} → skipped`

## Phase 3: Tests

- [x] T004 Add test for `active → skipped` to `src/lib/statusman/test.bats` — verify skip succeeds from active state
- [x] T005 Update existing test `skip: rejects active stage` to instead test `skip: rejects ready stage` (active is now valid, ready is not)

---

## Execution Order

- T001 first (schema drives lookup_transition)
- T002, T003 are independent
- T004, T005 depend on T001
