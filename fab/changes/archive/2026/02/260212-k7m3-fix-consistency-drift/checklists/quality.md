# Quality Checklist: Fix consistency drift between design, docs, and implementation

**Change**: 260212-k7m3-fix-consistency-drift
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Brief Template Header: `fab/.kit/templates/brief.md` line 1 reads `# Brief: {CHANGE_NAME}`
- [x] CHK-002 DEFERRED Timing: `fab/.kit/templates/brief.md` Open Questions comment reads "can resolve during spec"
- [x] CHK-003 Generation Partial (spec): `fab/.kit/skills/_generation.md` Spec Generation step 2 references "the brief" not "the proposal"
- [x] CHK-004 Generation Partial (tasks): `fab/.kit/skills/_generation.md` Tasks Generation step 2 references "the brief" not "the proposal"
- [x] CHK-005 Architecture Slug Count: `fab/design/architecture.md` folder naming table reads "2-6 words"
- [x] CHK-006 Glossary Slug Count: `fab/design/glossary.md` Folder name format reads "2–6 word slug"
- [x] CHK-007 Branch Integration: `fab/design/architecture.md` Git Integration section references `/fab-switch`
- [x] CHK-008 Status Schema (stage removal): `fab/design/architecture.md` .status.yaml examples have no `stage:` field
- [x] CHK-009 Status Schema (created_by): `fab/design/architecture.md` .status.yaml examples include `created_by`
- [x] CHK-010 Origin Section: `fab/design/templates.md` brief template includes `## Origin` section
- [x] CHK-011 Archive Index: `fab/design/templates.md` documents archive index maintenance
- [x] CHK-012 Backfill Terminology: `fab/docs/fab-workflow/index.md` backfill entry reads "docs and design"
- [x] CHK-013 Confidence Default: `fab/docs/fab-workflow/planning-skills.md` distinguishes count defaults (zero) from score default (5.0)

## Behavioral Correctness

- [x] CHK-014 No Stale "Proposal" References: No remaining "Proposal" references in `fab/.kit/templates/brief.md` or `fab/.kit/skills/_generation.md`
- [x] CHK-015 No Stale "2-4 word" References: No remaining "2-4 word" slug references in `fab/design/` (also fixed bonus occurrence in `skills.md`)

## Scenario Coverage

- [x] CHK-016 DEFERRED/BLOCKING Pair: `[BLOCKING]` guidance in brief template still reads "must resolve before spec" (unchanged)
- [x] CHK-017 Branch Integration Context: Architecture section explains that `/fab-new` delegates branch integration to `/fab-switch`

## Documentation Accuracy

- [x] CHK-018 Design-Implementation Alignment: All .status.yaml examples in design match the template schema in `fab/design/templates.md` (note: pre-existing `brief` key in progress map is out of scope — present before this change)
- [x] CHK-019 Cross-Layer Consistency: Each of the 11 original findings is resolved — design, docs, and implementation agree
- [x] CHK-020 No Orphaned References: No other files in `fab/.kit/` still reference "proposal" (grep confirmed 0 matches)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
- **Pre-existing issue noted**: `fab/design/templates.md` progress map keys don't include `brief`, but architecture examples and implementation template do include it. This predates this change and is not in scope.
