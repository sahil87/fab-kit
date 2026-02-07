# Quality Checklist: Add `fab/specs/` Index and Clarify Specs vs Docs Distinction

**Change**: 260207-bb1q-add-specs-index
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Init creates specs index: `/fab-init` creates `fab/specs/index.md` during bootstrap
- [x] CHK-002 Init idempotent on specs index: Re-running `/fab-init` skips existing `fab/specs/index.md`
- [x] CHK-003 Specs boilerplate distinguishes from docs: Header clearly states specs = pre-implementation / planning
- [x] CHK-004 Specs boilerplate notes human-curated: Boilerplate states specs are human-curated, not auto-generated
- [x] CHK-005 Specs index is flat: No prescribed domain directory hierarchy in boilerplate
- [x] CHK-006 Docs boilerplate distinguishes from specs: Header clearly states docs = post-implementation / authoritative truth
- [x] CHK-007 Docs boilerplate references specs: Cross-reference to `fab/specs/index.md` present
- [x] CHK-008 Context loading includes specs index: `_context.md` lists `fab/specs/index.md` in Always Load
- [x] CHK-009 Init step ordering correct: Step 1d creates specs index, after 1c (docs index), before 1e (changes/)

## Scenario Coverage

- [x] CHK-010 First-run scenario: Fresh init creates both docs and specs index files
- [x] CHK-011 Re-run scenario: Existing specs index is not overwritten
- [x] CHK-012 New user reads specs index: Header is self-explanatory about purpose and distinction
- [x] CHK-013 Agent context loading: Always Load layer now has 4 files listed

## Documentation Accuracy

- [x] CHK-014 Docs index existing content preserved: Table rows in `fab/docs/index.md` unchanged
- [x] CHK-015 Init re-run output updated: Includes specs/index.md in the verification output

## Cross References

- [x] CHK-016 Specs index references docs index: Bidirectional cross-reference present
- [x] CHK-017 Init step letters sequential: 1a through 1g with no gaps or duplicates

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
