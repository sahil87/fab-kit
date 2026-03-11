# Tasks: Drop runtime is-idle from Operator

**Change**: 260311-ftrh-drop-runtime-idle-from-operator
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Edit `fab/.kit/skills/fab-operator1.md` — Purpose section: replace "via `fab pane-map` and `fab runtime`" with "via `fab pane-map`"
- [x] T002 [P] Edit `fab/.kit/skills/fab-operator1.md` — State Re-derivation section: remove the `fab runtime is-idle` bullet, keep only `fab pane-map`
- [x] T003 [P] Edit `fab/.kit/skills/fab-operator1.md` — UC1 Broadcast: change "via `fab runtime` state in the pane map" to "via the Agent column in the pane map"
- [x] T004 [P] Edit `fab/.kit/skills/fab-operator1.md` — UC6 Unstick: replace "Confirm the target agent is idle via `fab/.kit/bin/fab runtime`" with "Confirm the target agent is idle via the Agent column in the pane map"
- [x] T005 [P] Edit `fab/.kit/skills/fab-operator1.md` — Pre-Send Validation: replace "`fab/.kit/bin/fab runtime is-idle <change>` or read the Agent column from the pane map" with "Read the Agent column from the pane map"
- [x] T006 [P] Edit `fab/.kit/skills/fab-operator1.md` — Autopilot per-change loop: replace "poll `fab pane-map` + `fab runtime is-idle`" with "poll `fab pane-map`"

## Phase 2: Spec Updates

- [x] T007 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Summary: remove `fab runtime` from observation primitives
- [x] T008 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Primitives table: remove the `fab runtime is-idle` row
- [x] T009 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Per-change loop monitoring: remove `fab runtime is-idle`
- [x] T010 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Pre-send validation: replace `runtime is-idle` with pane-map Agent column
- [x] T011 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Always re-derive state: remove `fab runtime is-idle`
- [x] T012 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Agent busy detection: replace `fab runtime is-idle` with pane-map Agent column
- [x] T013 [P] Edit `docs/specs/skills/SPEC-fab-operator1.md` — Relationship table: remove the `fab runtime is-idle` row

## Phase 3: Verification

- [x] T014 Verify no residual `runtime is-idle` or `fab runtime` references remain in the two edited files

---

## Execution Order

- All Phase 1 tasks (T001–T006) are independent and can run in parallel
- All Phase 2 tasks (T007–T013) are independent and can run in parallel
- Phase 1 and Phase 2 can run in parallel (different files)
- T014 depends on all prior tasks completing
