# Quality Checklist: Rework fab-ff to go all the way to archive

**Change**: 260212-bk1n-rework-fab-ff-archive
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Full Pipeline Execution: fab-ff.md includes Steps 6-8 for apply, review, and archive after planning stages
- [x] CHK-002 Execution Stage Invocation: fab-ff references standalone skill behavior ("Execute `/fab-apply` behavior", "Execute `/fab-review` behavior", "Execute `/fab-archive` behavior") — not inlined
- [x] CHK-003 No Confidence Gate: fab-ff.md Pre-flight Check has no confidence gate — only checks brief.md exists
- [x] CHK-004 Interactive Review Failure: fab-ff.md Step 7 presents rework menu with fix code, revise tasks, revise spec options
- [x] CHK-005 Apply Failure Stop: fab-ff.md Step 6 specifies STOP on unresolvable task failure with actionable message
- [x] CHK-006 Full Pipeline Resumability: fab-ff.md Resumability section covers spec, tasks, apply, review, archive
- [x] CHK-007 Phased Output: fab-ff.md Output uses `--- Planning ---`, `--- Implementation (fab-apply) ---`, `--- Review (fab-review) ---`, `--- Archive (fab-archive) ---`
- [x] CHK-008 Updated Comparison Table: fab-ff.md and fab-fff.md both have 3-column comparison tables showing full-pipeline behavior

## Behavioral Correctness

- [x] CHK-009 Planning behavior preserved: fab-ff.md Steps 1-5 unchanged — frontloaded questions, SRAD scoring, auto-clarify between stages, bail on blockers
- [x] CHK-010 fab-fff behavior unchanged: fab-fff.md retains confidence gate >= 3.0, autonomous mode, immediate bail on review failure, "Steps 1-5" planning reference

## Removal Verification

- [x] CHK-011 Deprecated planning-only scope: fab-ff.md Purpose says "entire Fab pipeline", Next Steps are context-dependent (archive/bail/failure)

## Scenario Coverage

- [x] CHK-012 Clean full pipeline scenario: "Clean Full Pipeline" output example shows all four phase headers completing
- [x] CHK-013 Resume mid-pipeline scenario: "Resume After Bail or Failure" output example shows "Skipping planning/implementation"
- [x] CHK-014 Review failure scenario: "Review Failure with Interactive Rework" output example shows rework options 1-3
- [x] CHK-015 Planning bail scenario: "Bail on Blocking Issue (Planning)" output example preserves existing bail behavior

## Edge Cases & Error Handling

- [x] CHK-016 Error handling table: fab-ff.md Error Handling table includes "Task fails during apply", "Review fails", "Archive fails" rows

## Documentation Accuracy

- [x] CHK-017 planning-skills.md: fab-ff renamed to "Fast Forward — Full Pipeline", generation flow updated to 8 steps, interactive review failure section added
- [x] CHK-018 execution-skills.md: "Pipeline invocation" paragraph added to Overview noting fab-ff and fab-fff invoke execution skills

## Cross References

- [x] CHK-019 Next Steps consistency: `_context.md` Next Steps table updated — fab-ff shows `archived` stage with bail variant row
- [x] CHK-020 fab-fff references: fab-fff.md Purpose references fab-ff as full-pipeline sibling, comparison table includes fab-ff column

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
