# Quality Checklist: Streamline Planning Stage Dispatch

**Change**: 260227-ijql-streamline-planning-dispatch
**Generated**: 2026-02-27

## Functional Completeness

- [x] CHK-001 Template `status.yaml` initializes `intake: pending` (not `active`)
- [x] CHK-002 `/fab-new` skill calls `advance` after intake generation — intake ends as `ready`
- [x] CHK-003 `/fab-continue` dispatch table updated — `ready` planning stages finish + generate next + advance to `ready`
- [x] CHK-004 Single-dispatch rule removed from `fab-continue.md`
- [x] CHK-005 Backward-compat `active` rows present in dispatch table for interrupted generations

## Behavioral Correctness

- [x] CHK-006 Reset flow preserved — planning resets use `advance` (not `finish`) to stay at `ready`
- [x] CHK-007 Apply, review, hydrate dispatch rows unchanged
- [x] CHK-008 `workflow.yaml` and `stageman.sh` are NOT modified

## Documentation Accuracy

- [x] CHK-009 `docs/specs/skills.md` dispatch table matches `fab-continue.md` dispatch table
- [x] CHK-010 `docs/specs/skills.md` Next Steps table reflects new flow
- [x] CHK-011 `docs/specs/user-flow.md` diagrams consistent with new dispatch
- [x] CHK-012 Memory file `templates.md` reflects `intake: pending` template change
- [x] CHK-013 Memory file `planning-skills.md` describes consolidated dispatch (no single-dispatch rule)
- [x] CHK-014 Memory file `change-lifecycle.md` shows `ready` as default post-generation state

## Cross-References

- [x] CHK-015 All references to "single-dispatch rule" removed or updated across modified files
- [x] CHK-016 Spec, skills spec, and skill file dispatch tables are mutually consistent
