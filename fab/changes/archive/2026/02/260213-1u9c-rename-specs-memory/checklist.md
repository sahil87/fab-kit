# Quality Checklist: Rename design/ → specs/ and docs/ → memory/

**Change**: 260213-1u9c-rename-specs-memory
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
<!-- Every requirement in spec.md has working implementation -->
- [x] CHK-001 Directory rename (design → specs): `fab/specs/` exists, `fab/design/` does not
- [x] CHK-002 Directory rename (docs → memory): `fab/memory/` exists, `fab/docs/` does not
- [x] CHK-003 Skill files updated: No `fab/docs/` or `fab/design/` references remain in `fab/.kit/skills/`
- [x] CHK-004 Script files updated: No `fab/docs/` or `fab/design/` references remain in `fab/.kit/scripts/`
- [x] CHK-005 Template files updated: No `fab/docs/` or `fab/design/` references remain in `fab/.kit/templates/`
- [x] CHK-006 Scaffold files updated: Files renamed to `specs-index.md` and `memory-index.md`, internal references updated
- [x] CHK-007 config.yaml updated: No stale path references
- [x] CHK-008 constitution.md updated: No stale path references
- [x] CHK-009 Internal cross-refs in memory/: No `fab/docs/` or `fab/design/` self-references remain
- [x] CHK-010 Internal cross-refs in specs/: No `fab/docs/` or `fab/design/` self-references remain
- [x] CHK-011 README.md updated: No stale path references
- [x] CHK-012 Scaffold script updated: `_fab-scaffold.sh` creates `fab/memory/` and `fab/specs/` (not old names)

## Behavioral Correctness
<!-- Changed requirements behave as specified, not as before -->
- [x] CHK-013 Index headers: `fab/memory/index.md` header reflects "Memory", `fab/specs/index.md` header reflects "Specs"
- [x] CHK-014 Index prose: Both index files use new folder terminology in their descriptive text

## Scenario Coverage
<!-- Key scenarios from spec.md have been exercised -->
- [x] CHK-015 All contents preserved: File count in `fab/specs/` matches former `fab/design/`, same for `fab/memory/` vs former `fab/docs/`
- [x] CHK-016 Relative links work: `../memory/index.md` in specs, `../specs/index.md` in memory resolve correctly

## Edge Cases & Error Handling
- [x] CHK-017 No stale references in source: `grep -r 'fab/docs/' fab/.kit/ fab/memory/ fab/specs/ fab/config.yaml fab/constitution.md README.md` returns zero results
- [x] CHK-018 No stale references in source: `grep -r 'fab/design/' fab/.kit/ fab/memory/ fab/specs/ fab/config.yaml fab/constitution.md README.md` returns zero results

## Documentation Accuracy
- [x] CHK-019 Archived changes untouched: Files in `fab/changes/archive/` are unmodified

## Cross References
- [x] CHK-020 Active change artifacts preserved: Other active change folders (not 260213-1u9c) are unmodified

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
