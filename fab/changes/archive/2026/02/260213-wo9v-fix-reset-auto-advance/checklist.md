# Quality Checklist: Fix reset flow to stop at target stage

**Change**: 260213-wo9v-fix-reset-auto-advance
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Stage derivation fallback: `fab-preflight.sh` returns first pending stage after last done when no active entry exists
- [x] CHK-002 Stage derivation fallback: `fab-status.sh` returns first pending stage after last done when no active entry exists
- [x] CHK-003 Stage derivation fallback: `stageman.sh` `get_current_stage()` returns first pending stage after last done when no active entry exists
- [x] CHK-004 Schema update: `workflow.yaml` `progression.current_stage.rule` describes three-tier fallback
- [x] CHK-005 Status pending cases: `fab-status.sh` case statement includes `{stage}:pending` entries for all stages
- [x] CHK-006 Skill definition: `fab-continue.md` Reset Flow stops at target stage (no auto-advance to next)
- [x] CHK-007 Skill definition: `fab-continue.md` includes pre-guard step for pending→active transition

## Behavioral Correctness

- [x] CHK-008 All-done fallback preserved: when all stages are `done`, derivation still returns `archive`
- [x] CHK-009 Active-entry behavior unchanged: when an `active` entry exists, derivation finds it first (fallback not triggered)
- [x] CHK-010 Normal two-write transition unchanged: forward progression in non-reset flow still uses `current: done` + `next: active`

## Scenario Coverage

- [x] CHK-011 Scenario: brief done + spec pending (no active) → derived stage is `spec`
- [x] CHK-012 Scenario: brief + spec done + tasks pending → derived stage is `tasks`
- [x] CHK-013 Scenario: all done → derived stage is `archive`
- [x] CHK-014 Scenario: review failed + apply active → derived stage is `apply` (active found)
- [x] CHK-015 Scenario: status display after reset shows correct stage and next command

## Edge Cases & Error Handling

- [x] CHK-016 Edge case: `fab-status.sh` next-command handles `tasks:pending` state correctly
- [x] CHK-017 Edge case: `fab-status.sh` next-command handles `spec:pending` state correctly

## Documentation Accuracy

- [x] CHK-018 `fab-continue.md` Reset Flow section matches the spec's stop-at-target behavior
- [x] CHK-019 `workflow.yaml` progression rule text matches the implemented three-tier fallback

## Cross References

- [x] CHK-020 All three fallback implementations (preflight, status, stageman) use the same logic
- [x] CHK-021 `fab-status.sh` pending cases match the list in the spec

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
