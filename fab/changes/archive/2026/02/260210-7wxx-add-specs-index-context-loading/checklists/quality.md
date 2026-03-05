# Quality Checklist: Add fab/specs/index.md to context loading in apply, review, and archive

**Change**: 260210-7wxx-add-specs-index-context-loading
**Generated**: 2026-02-10
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 fab-apply context loading: `fab/.kit/skills/fab-apply.md` Context Loading section includes `fab/specs/index.md` as a numbered item
- [x] CHK-002 fab-review context loading: `fab/.kit/skills/fab-review.md` Context Loading section includes `fab/specs/index.md` as a numbered item
- [x] CHK-003 fab-archive context loading: `fab/.kit/skills/fab-archive.md` Context Loading section includes `fab/specs/index.md` as a numbered item

## Behavioral Correctness
- [x] CHK-004 Consistent description: Each addition uses the description "specifications landscape (pre-implementation design intent, human-curated)"
- [x] CHK-005 Numbered list integrity: Each addition fits naturally into the existing numbered list without breaking numbering

## Scenario Coverage
- [x] CHK-006 fab-apply scenario: Context Loading section of fab-apply.md contains specs index line with correct description
- [x] CHK-007 fab-review scenario: Context Loading section of fab-review.md contains specs index line with correct description
- [x] CHK-008 fab-archive scenario: Context Loading section of fab-archive.md contains specs index line with correct description

## Documentation Accuracy
- [x] CHK-009 Alignment with _context.md: The added lines match the always-load protocol defined in `_context.md`

## Cross References
- [x] CHK-010 No broken references: No existing cross-references in the three files are disrupted by the additions

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
