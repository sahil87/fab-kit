# Quality Checklist: Unify Pipeline Commands into fab-continue

**Change**: 260212-a4bd-unify-fab-continue
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Unified Stage Advancement: `fab-continue` (no arg) dispatches correctly for all 6 active stages (brief, spec, tasks, apply, review, archive)
- [x] CHK-002 Apply Behavior: `fab-continue` executes tasks from `tasks.md` in dependency order with test-after-each and immediate `[x]` marking
- [x] CHK-003 Review Behavior: `fab-continue` performs 5-step validation (tasks, checklist, tests, spec match, doc drift) and presents rework options on failure
- [x] CHK-004 Archive Behavior: `fab-continue` validates, hydrates docs, moves to archive, updates index, marks backlog, clears pointer — in fail-safe order
- [x] CHK-005 Extended Reset: `fab-continue <stage>` accepts all 6 stages (brief, spec, tasks, apply, review, archive) as valid reset targets
- [x] CHK-006 fab-ff References: No remaining references to `/fab-apply`, `/fab-review`, `/fab-archive` as invocable commands in `fab-ff.md`
- [x] CHK-007 fab-fff References: No remaining references to `/fab-apply`, `/fab-review`, `/fab-archive` as invocable commands in `fab-fff.md`
- [x] CHK-008 Next Steps Table: `_context.md` lookup table matches the spec's simplified table

## Behavioral Correctness

- [x] CHK-009 Stage guard transitions: tasks→apply, apply→review (via apply completion), review→archive produce correct `.status.yaml` two-write transitions
- [x] CHK-010 Review rework options reference `/fab-continue` (not `/fab-apply`) and `/fab-continue spec` (not standalone `/fab-review`)
- [x] CHK-011 Execution reset does not reset task checkboxes: `fab-continue apply` preserves existing `[x]` marks
- [x] CHK-012 Normal flow review→archive: when `review` is active, fab-continue runs review (not archive); when `review: done` and `archive: active`, runs archive

## Removal Verification

- [x] CHK-013 Skill deletion: `fab-apply.md`, `fab-review.md`, `fab-archive.md` no longer exist in `fab/.kit/skills/`
- [x] CHK-014 No dangling references: zero matches for `/fab-apply`, `/fab-review`, `/fab-archive` as invocable commands outside `fab/changes/archive/` and changelog tables

## Scenario Coverage

- [x] CHK-015 Advance from tasks to apply: stage guard dispatches apply behavior when `tasks` is active
- [x] CHK-016 Resume interrupted apply: when `apply` is active, resumes from first unchecked task
- [x] CHK-017 Review pass: sets `review: done, archive: active` and outputs "Next: /fab-continue"
- [x] CHK-018 Review fail: sets `review: failed, apply: active` and presents 3 rework options
- [x] CHK-019 Archive completion: hydrates docs, moves folder, clears pointer, outputs "Next: /fab-new"
- [x] CHK-020 Reset to apply: sets `apply: active`, resets review/archive to pending
- [x] CHK-021 Reset to review: sets `review: active`, resets archive to pending
- [x] CHK-022 Reset to brief: sets `brief: active`, all others to pending, regenerates brief.md

## Edge Cases & Error Handling

- [x] CHK-023 All tasks already complete during apply dispatch: outputs "All tasks already complete" and transitions to review
- [x] CHK-024 Change already complete (all done): outputs "Change is complete."
- [x] CHK-025 Unchecked tasks block review: stops with appropriate message referencing `/fab-continue`
- [x] CHK-026 Review not passed blocks archive: stops with message referencing `/fab-continue`

## Documentation Accuracy

- [x] CHK-027 `fab/docs/fab-workflow/planning-skills.md` reflects unified command covering all stages
- [x] CHK-028 `fab/docs/fab-workflow/execution-skills.md` restructured for unified command
- [x] CHK-029 `fab/docs/fab-workflow/change-lifecycle.md` references `/fab-continue` uniformly

## Cross References

- [x] CHK-030 All skill files (non-deleted) reference `/fab-continue` instead of removed commands
- [x] CHK-031 All design specs reference `/fab-continue` instead of removed commands
- [x] CHK-032 README.md references `/fab-continue` instead of removed commands

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
