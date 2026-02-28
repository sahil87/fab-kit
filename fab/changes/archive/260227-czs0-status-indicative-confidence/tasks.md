# Tasks: Status Indicative Confidence

**Change**: 260227-czs0-status-indicative-confidence
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add assumption count fields to intake branch of `--check-gate` output in `fab/.kit/scripts/lib/calc-score.sh` (lines ~182-191) — emit `certain`, `confident`, `tentative`, `unresolved` from local variables already computed
- [x] T002 Add assumption count fields to spec branch of `--check-gate` output in `fab/.kit/scripts/lib/calc-score.sh` (lines ~174-177) — read from `.status.yaml` confidence block and emit alongside existing gate/score/threshold/change_type

## Phase 2: Skill Updates

- [x] T003 [P] Update `fab/.kit/skills/fab-status.md` confidence display section — replace single confidence bullet with three-case stage-aware logic (intake indicative, spec+ persisted, fallback)
- [x] T004 [P] Update `.claude/skills/fab-status/SKILL.md` confidence display section — mirror the same three-case logic as T003

---

## Execution Order

- T001 and T002 are independent (different branches of the same conditional)
- T003 and T004 are independent of each other but logically depend on T001+T002 (the counts must be available for fab-status to display them)
