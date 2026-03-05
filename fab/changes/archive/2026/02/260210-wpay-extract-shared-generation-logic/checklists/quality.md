# Quality Checklist: Extract shared generation logic

**Change**: 260210-wpay-extract-shared-generation-logic
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Generation partial exists: `fab/.kit/skills/_generation.md` exists and contains all four generation procedures (spec, plan, tasks, checklist)
- [x] CHK-002 Spec generation procedure: Contains template loading, metadata fields, RFC 2119 keywords, GIVEN/WHEN/THEN scenarios, deprecated requirements section, `[NEEDS CLARIFICATION]` markers, Assumptions section
- [x] CHK-003 Plan generation procedure: Contains template loading, metadata fields, summary, goals/non-goals, technical context, research, decisions, risks/trade-offs, file changes, Assumptions section
- [x] CHK-004 Tasks generation procedure: Contains template loading, metadata fields, phased task breakdown (Phase 1-4), task format with IDs and markers, execution order section
- [x] CHK-005 Checklist generation procedure: Contains template loading, directory creation, category population, sequential CHK IDs, `.status.yaml` updates

## Behavioral Correctness

- [x] CHK-006 fab-continue references partial: `fab-continue.md` references `_generation.md` for all four generation procedures instead of inlining them
- [x] CHK-007 fab-ff references partial: `fab-ff.md` references `_generation.md` for all four generation procedures instead of inlining them
- [x] CHK-008 Orchestration preserved in fab-continue: Stage guards, SRAD questions, plan confirmation prompt, confidence recomputation, reset flow, and `.status.yaml` transitions remain in `fab-continue.md`
- [x] CHK-009 Orchestration preserved in fab-ff: Frontloaded questions, autonomous plan decision, auto-clarify interleaving, bail logic, resumability, and `.status.yaml` transitions remain in `fab-ff.md`

## Scenario Coverage

- [x] CHK-010 Agent reading fab-continue can follow generation references: References in `fab-continue.md` clearly direct the agent to load and follow `_generation.md` procedures
- [x] CHK-011 Agent reading fab-ff can follow generation references: References in `fab-ff.md` clearly direct the agent to load and follow `_generation.md` procedures
- [x] CHK-012 Content equivalence: Generation procedures in `_generation.md` are semantically identical to the previous inline versions (no behavioral changes)

## Edge Cases & Error Handling

- [x] CHK-013 File naming convention: `_generation.md` uses underscore prefix consistent with `_context.md`
- [x] CHK-014 Readability: Both `fab-continue.md` and `fab-ff.md` read coherently end-to-end after extraction — reference text flows naturally with surrounding orchestration

## Documentation Accuracy

- [x] CHK-015 Centralized doc updated: `fab/docs/fab-workflow/planning-skills.md` documents the `_generation.md` partial and how fab-continue/fab-ff delegate to it

## Cross References

- [x] CHK-016 No dangling references: All references to `_generation.md` use the correct relative path and section names

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
