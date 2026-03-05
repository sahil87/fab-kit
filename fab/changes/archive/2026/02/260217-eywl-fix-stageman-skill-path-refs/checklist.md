# Quality Checklist: Fix Stageman Skill Path References

**Change**: 260217-eywl-fix-stageman-skill-path-refs
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Repo-root-relative stageman paths: All 32 `lib/stageman.sh` references replaced with `fab/.kit/scripts/lib/stageman.sh` across 6 skill files
- [x] CHK-002 Repo-root-relative preflight paths: Both short-form `lib/preflight.sh` references replaced in `_context.md` and `fab-status.md`
- [x] CHK-003 No script modifications: No files under `fab/.kit/scripts/` were changed

## Behavioral Correctness
- [x] CHK-004 Existing full-form preflight references preserved: `_context.md` line 27, `fab-archive.md` line 36, `fab-status.md` line 38 remain unchanged
- [x] CHK-005 No content changes beyond paths: Only path strings changed, no surrounding text or structure modified

## Scenario Coverage
- [x] CHK-006 fab-continue.md: 10 occurrences replaced
- [x] CHK-007 fab-ff.md: 8 occurrences replaced
- [x] CHK-008 fab-fff.md: 9 occurrences replaced
- [x] CHK-009 fab-clarify.md: 1 occurrence replaced
- [x] CHK-010 _generation.md: 3 occurrences replaced
- [x] CHK-011 fab-status.md: stageman (1) and preflight (1) short-form replaced

## Code Quality
- [x] CHK-012 Pattern consistency: All script path references now follow the same repo-root-relative convention as `_context.md`
- [x] CHK-013 No unnecessary duplication: No new patterns or abstractions introduced

## Documentation Accuracy
- [x] CHK-014 Path references in skill files match the actual filesystem layout

## Cross References
- [x] CHK-015 **N/A**: Memory file `execution-skills.md` retains short-form — will be updated during hydrate

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
