# Quality Checklist: Rename /fab-backfill to /fab-hydrate-design

**Change**: 260212-akhp-rename-fab-backfill
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Rename Kit Skill File: `fab/.kit/skills/fab-hydrate-design.md` exists with correct frontmatter `name` and heading
- [x] CHK-002 Rename Claude Skill Directory: `.claude/skills/fab-hydrate-design/SKILL.md` symlink resolves to `fab/.kit/skills/fab-hydrate-design.md`
- [x] CHK-003 Rename Centralized Doc: `fab/docs/fab-workflow/hydrate-design.md` exists with updated heading and references
- [x] CHK-004 Update Documentation Indexes: `fab/docs/fab-workflow/index.md` and `fab/docs/index.md` reference `hydrate-design`
- [x] CHK-005 Update All Active References: all 8 cross-cutting files updated per spec table
- [x] CHK-006 Preserve Archived Changes: files under `fab/changes/archive/` are unmodified
- [x] CHK-007 Update Active Change References: `fab/changes/260212-h9k3-fab-init-family/brief.md` uses `/fab-hydrate-design`

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-008 Old paths removed: `fab/.kit/skills/fab-backfill.md` no longer exists
- [x] CHK-009 Old directory removed: `.claude/skills/fab-backfill/` no longer exists
- [x] CHK-010 Old doc removed: `fab/docs/fab-workflow/backfill.md` no longer exists
- [x] CHK-011 Symlink resolves: `.claude/skills/fab-hydrate-design/SKILL.md` is readable (not broken)

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-012 Kit skill file renamed: file at new path, old path gone, internal refs updated
- [x] CHK-013 Claude skill directory renamed: directory at new path, symlink target correct
- [x] CHK-014 Centralized doc renamed: file at new path, heading updated, command refs updated, change folder names preserved
- [x] CHK-015 Domain index updated: entry links to `hydrate-design.md`
- [x] CHK-016 Top-level docs index updated: `hydrate-design` in domain doc list

## Edge Cases & Error Handling
<!-- Error states, boundary conditions, failure modes -->
- [x] CHK-017 Historical change folder names preserved: `260209-h3v7-fab-backfill` appears as-is in renamed doc changelog
- [x] CHK-018 This change's own artifacts excluded: `260212-akhp-rename-fab-backfill` folder not modified by reference sweep

## Documentation Accuracy
<!-- Project-specific: documentation_accuracy from config -->
- [x] CHK-019 No stale command references: grep for `fab-backfill` in non-archived `.md`/`.sh`/`.yaml` files returns zero command-name hits

## Cross References
<!-- Project-specific: cross_references from config -->
- [x] CHK-020 Doc index links resolve: all `[hydrate-design](hydrate-design.md)` links point to existing files
- [x] CHK-021 Design spec references consistent: glossary, skills, user-flow all use `/fab-hydrate-design`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
