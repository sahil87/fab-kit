# Quality Checklist: Fix stage guard to check progress value instead of stage name

**Change**: 260210-0p4e-fix-stage-guard-progress-check
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Guard checks progress value: The guard logic in `fab-continue.md` Step 1 checks `progress.{stage}` value (not just `stage` field name) when determining whether to allow continuation
- [x] CHK-002 Preflight output used: The guard logic uses the `progress` map from preflight output (parsed in Pre-flight Check step), not re-read from `.status.yaml`

## Behavioral Correctness

- [x] CHK-003 Tasks active allows resume: When `stage: tasks` and `progress.tasks: active`, the guard allows task generation to resume (does NOT block with "Planning is complete")
- [x] CHK-004 Tasks done blocks: When `stage: tasks` and `progress.tasks: done`, the guard blocks with "Planning is complete. Run /fab-apply to begin implementation."
- [x] CHK-005 Specs active allows resume: When `stage: specs` and `progress.specs: active`, the guard allows spec generation to resume
- [x] CHK-006 Apply or later blocks: When `stage: apply` (or review/archive) with `progress.apply: active` or `done`, the guard blocks with "Implementation is underway..." (unchanged behavior)

## Scenario Coverage

- [x] CHK-007 Tasks active scenario: Verified behavior when change has `stage: tasks`, `progress.tasks: active` — guard allows resumption
- [x] CHK-008 Tasks done scenario: Verified behavior when change has `stage: tasks`, `progress.tasks: done` — guard blocks appropriately
- [x] CHK-009 Specs active scenario: Verified behavior when change has `stage: specs`, `progress.specs: active` — guard allows resumption
- [x] CHK-010 Apply+ scenario: Verified behavior when stage is `apply` or later — guard blocks regardless of progress value

## Edge Cases & Error Handling

- [x] CHK-011 Progress value consistency: Guard correctly distinguishes `'done'` vs `'active'` vs `'pending'` values from progress map
- [x] CHK-012 All planning stages covered: Guard logic applies correct check to specs, plan, and tasks stages

## Documentation Accuracy

- [x] CHK-013 Guard description updated: The skill documentation in `fab-continue.md` accurately describes the guard logic checking progress values

## Cross References

- [x] CHK-014 planning-skills.md consistency: The documentation in `fab/docs/fab-workflow/planning-skills.md` will be updated during archive to reflect the corrected guard behavior

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-XXX **N/A**: {reason}`
