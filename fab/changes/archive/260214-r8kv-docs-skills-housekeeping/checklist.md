# Quality Checklist: Docs Skills Housekeeping

**Change**: 260214-r8kv-docs-skills-housekeeping
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-status.sh removed: `fab/.kit/scripts/fab-status.sh` no longer exists
- [x] CHK-002 fab-hydrate-specs renamed: file exists at `fab/.kit/skills/docs-hydrate-specs.md`, old path gone
- [x] CHK-003 fab-hydrate renamed: file exists at `fab/.kit/skills/docs-hydrate-memory.md`, old path gone
- [x] CHK-004 fab-reorg-specs renamed: file exists at `fab/.kit/skills/docs-reorg-specs.md`, old path gone
- [x] CHK-005 docs-reorg-memory created: `fab/.kit/skills/docs-reorg-memory.md` exists with correct structure
- [x] CHK-006 Symlinks regenerated: `.claude/skills/docs-hydrate-specs/`, `.claude/skills/docs-hydrate-memory/`, `.claude/skills/docs-reorg-specs/`, `.claude/skills/docs-reorg-memory/` all exist and point to correct targets
- [x] CHK-007 Stale symlinks removed: no `.claude/skills/fab-hydrate-specs/`, `.claude/skills/fab-hydrate/`, `.claude/skills/fab-reorg-specs/` directories remain

## Behavioral Correctness

- [x] CHK-008 /fab-status still works: the skill file `fab-status.md` does not reference `fab-status.sh`
- [x] CHK-009 Skill name disambiguation: references to `/fab-hydrate` (skill) are renamed but "hydrate behavior" (pipeline stage) references are preserved unchanged

## Removal Verification

- [x] CHK-010 fab-status.sh dead references: no non-archived file references `fab-status.sh` as a current/active script (only changelog entries remain, which are historical)
- [x] CHK-011 Old skill names gone: no non-archived file references `fab-hydrate-specs`, `fab-hydrate.md` (skill file), or `fab-reorg-specs` as current skill names

## Scenario Coverage

- [x] CHK-012 Scaffold discovers new skills: `_fab-scaffold.sh` glob `*.md` picks up `docs-*.md` files (confirmed: 4 new symlinks created per agent)
- [x] CHK-013 docs-reorg-memory pre-flight: skill aborts on empty/missing `fab/memory/` (verified in skill definition)
- [x] CHK-014 Archived changes untouched: no files under `fab/changes/archive/` have been modified (`git diff --stat` confirms zero changes)

## Edge Cases & Error Handling

- [x] CHK-015 fab-hydrate vs hydrate behavior: in files that mention both the standalone skill and the pipeline stage, only skill references are renamed (verified via grep sweep — pipeline stage refs like "hydrate behavior" preserved)

## Documentation Accuracy

- [x] CHK-016 kit-architecture directory listing: updated to show new skill filenames and no `fab-status.sh`
- [x] CHK-017 fab-help.sh output: lists new skill names, no old names

## Cross References

- [x] CHK-018 All memory files: references to renamed skills use new names
- [x] CHK-019 All spec files: references to renamed skills use new names
- [x] CHK-020 README.md: skill references use new names

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
