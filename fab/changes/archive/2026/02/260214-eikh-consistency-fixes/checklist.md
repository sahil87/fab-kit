# Quality Checklist: Consistency Fixes from 260214 Audit

**Change**: 260214-eikh-consistency-fixes
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Stage terminology: All "archive" stage references replaced with "hydrate" in overview.md, glossary.md, skills.md
- [x] CHK-002 Skill rename: All `/fab-hydrate` references replaced with `/docs-hydrate-memory` across overview.md, skills.md, glossary.md, architecture.md, user-flow.md
- [x] CHK-003 Command refs: workflow.yaml apply/review stages reference `fab-continue`
- [x] CHK-004 File case: No uppercase `SKILLS.md` or `TEMPLATES.md` cross-references remain
- [x] CHK-005 Symlink refs: architecture.md directory listing and symlink example use `docs-hydrate-memory.md`
- [x] CHK-006 Section heading: skills.md stage-6 section titled "Hydrate Behavior"
- [x] CHK-007 Docs-reorg coverage: skills.md has behavioral sections for `/docs-reorg-memory` and `/docs-reorg-specs`
- [x] CHK-008 Batch scripts: architecture.md has dedicated batch scripts documentation section
- [x] CHK-009 Internal skills: architecture.md directory listing includes `internal-consistency-check.md`, `internal-retrospect.md`, `internal-skill-optimize.md`
- [x] CHK-010 Glossary skills: glossary.md has entries for `/docs-reorg-memory` and `/docs-reorg-specs`
- [x] CHK-011 Glossary hydration: Hydration definition covers both pipeline hydration and source hydration (ingest + generate)
- [x] CHK-012 Sub-skill removal: No `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate` sections in skills.md
- [x] CHK-013 Orphaned entries: `hydrate-design` and `design-index` removed from memory/index.md

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-014 No over-correction: `/fab-archive` references (the standalone command) are preserved — only "archive" as a stage name is changed
- [x] CHK-015 Anchor integrity: Updated cross-reference links have valid anchors matching target section headings

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-016 Stage table: overview.md stage table row 6 reads "Hydrate" not "Archive"
- [x] CHK-017 Glossary stage list: glossary.md lists stages as "brief, spec, tasks, apply, review, hydrate"
- [x] CHK-018 Quick start: overview.md quick-start commands use `/docs-hydrate-memory`

## Edge Cases & Error Handling
- [x] CHK-019 Flowchart node: user-flow.md mermaid diagram uses `/docs-hydrate-memory` label

## Documentation Accuracy
<!-- Project-specific: from config.yaml extra_categories -->
- [x] CHK-020 New sections format: Added skill sections (cf07) follow existing skills.md section format
- [x] CHK-021 Batch docs accuracy: Batch script documentation matches actual script behavior per memory file

## Cross References
<!-- Project-specific: from config.yaml extra_categories -->
- [x] CHK-022 Internal consistency: No remaining `/fab-hydrate` references in any modified file
- [x] CHK-023 No orphaned anchors: Links between spec files resolve correctly after renames

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
