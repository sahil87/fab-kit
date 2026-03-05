# Quality Checklist: Add Non-Goals and Design Decisions to Spec Template

**Change**: 260211-r4w8-spec-template-sections
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Non-Goals placement: section appears after metadata/comments and before `## {Domain}: {Topic}` in `fab/.kit/templates/spec.md`
- [x] CHK-002 Non-Goals format: HTML comment guidance specifies bullet format `- {exclusion} — {reason}`
- [x] CHK-003 Non-Goals omission: template guidance instructs agents to omit the section entirely for trivial changes (no empty headings)
- [x] CHK-004 Design Decisions placement: section appears after domain sections and before `## Deprecated Requirements` in `fab/.kit/templates/spec.md`
- [x] CHK-005 Design Decisions entry format: template guidance specifies `1. **{Decision}`: {approach}` with Why and Rejected sub-items
- [x] CHK-006 Design Decisions omission: template guidance instructs agents to omit the section entirely for trivial changes
- [x] CHK-007 fab-continue awareness: `_generation.md` Spec Generation Procedure includes Non-Goals step (2b)
- [x] CHK-008 fab-ff awareness: covered by same `_generation.md` update as fab-continue (fab-ff delegates to same procedure)

## Behavioral Correctness

- [x] CHK-010 Existing Design Decisions step 5b in `_generation.md` is preserved and consistent with new template section
- [x] CHK-011 Both skills (fab-continue, fab-ff) follow same include/omit rules for optional sections via shared `_generation.md`

## Scenario Coverage

- [x] CHK-012 Spec with Non-Goals: template supports population from brief context with scope exclusions
- [x] CHK-013 Spec without Non-Goals: template supports full omission for straightforward changes
- [x] CHK-014 Spec with Design Decisions: template supports population with decision entries
- [x] CHK-015 Spec without Design Decisions: template supports full omission for trivial changes
- [x] CHK-016 fab-ff fast-forward: follows same rules as fab-continue for optional sections (verified via shared `_generation.md`)

## Documentation Accuracy

- [x] CHK-017 `templates.md` documents Non-Goals section: placement, bullet format, and optionality
- [x] CHK-018 `templates.md` documents Design Decisions section: placement, entry format, and optionality

## Cross References

- [x] CHK-019 `_generation.md` step numbers remain consistent after inserting Non-Goals step

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
