# Quality Checklist: Rename _context.md to _preamble.md

**Change**: 260221-5tj7-rename-context-to-preamble
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Rename preamble file: `fab/.kit/skills/_preamble.md` exists with identical content to original `_context.md`
- [x] CHK-002 Rename preamble file: `fab/.kit/skills/_context.md` no longer exists
- [x] CHK-003 Update preamble instruction line: All 15 skill files reference `_preamble.md` instead of `_context.md`
- [x] CHK-004 Update self-reference: `_preamble.md` blockquote references itself, not `_context.md`
- [x] CHK-005 Update memory files: All memory files under `docs/memory/` reference `_preamble.md`
- [x] CHK-006 Update spec files: All spec files under `docs/specs/` reference `_preamble.md`
- [x] CHK-007 Update directory tree: `kit-architecture.md` directory tree shows `_preamble.md`

## Behavioral Correctness
- [x] CHK-008 Instruction line format preserved: Updated lines use exact format `Read and follow the instructions in fab/.kit/skills/_preamble.md before proceeding.`
- [x] CHK-009 No content changes: The preamble file content is unchanged beyond the self-reference update

## Scenario Coverage
- [x] CHK-010 File exists at new path after rename: `_preamble.md` exists, `_context.md` does not
- [x] CHK-011 Skill file references updated: Grep for `_context.md` in `fab/.kit/skills/` returns zero matches (excluding archive)
- [x] CHK-012 Archive files untouched: Files in `fab/changes/archive/` retain original `_context.md` references

## Edge Cases & Error Handling
- [x] CHK-013 No stray references: Grep for `_context.md` across live repo files (excluding archive and this change's artifacts) returns zero matches

## Code Quality
- [x] CHK-014 Pattern consistency: Reference updates follow the exact same path format used elsewhere in each file
- [x] CHK-015 No unnecessary duplication: No new files created, no redundant changes

## Documentation Accuracy
- [x] CHK-016 Memory file content accurate: Updated references in memory files correctly point to the renamed file
- [x] CHK-017 Directory tree listing accurate: `kit-architecture.md` tree shows `_preamble.md` with same description

## Cross References
- [x] CHK-018 All cross-file references consistent: No file references `_context.md` while another references `_preamble.md` for the same concept

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
