# Quality Checklist: Fix broken template links in fab-hydrate

**Change**: 260208-4wg3-fix-hydrate-links
**Generated**: 2026-02-08
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Domain Index link: Line 97 points to `../../specs/templates.md#domain-index-fabdocsdomainindexmd`
- [x] CHK-002 Individual Doc link: Line 104 points to `../../specs/templates.md#individual-doc-fabdomainnamemd`
- [x] CHK-003 Top-Level Index link: Line 124 points to `../../specs/templates.md#top-level-index-fabdocsindexmd`

## Behavioral Correctness
- [x] CHK-004 No behavioral change: Skill instructions (logic, steps, output format) remain identical except for link targets

## Removal Verification
- [x] CHK-005 No residual old paths: Zero references to `doc/fab-spec/TEMPLATES.md` remain in `fab/.kit/skills/fab-hydrate.md`

## Documentation Accuracy
- [x] CHK-006 Anchor fragments valid: Each anchor fragment matches an actual heading in `fab/specs/templates.md`

## Cross References
- [x] CHK-007 Relative path correctness: From `fab/.kit/skills/fab-hydrate.md`, `../../specs/templates.md` correctly resolves to `fab/specs/templates.md`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
