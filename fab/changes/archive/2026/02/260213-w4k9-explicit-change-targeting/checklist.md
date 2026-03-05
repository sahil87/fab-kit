# Quality Checklist: Explicit Change Targeting for Workflow Commands

**Change**: 260213-w4k9-explicit-change-targeting
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Preflight override: `fab-preflight.sh` accepts `$1` and resolves change from it instead of `fab/current`
- [x] CHK-002 Status override: `fab-status.sh` accepts `$1` and displays status for the targeted change
- [x] CHK-003 Matching consistency: preflight uses same case-insensitive substring matching rules as `/fab-switch`
- [x] CHK-004 4-char ID shorthand: passing just the 4-char random ID (e.g., `r3m7`) resolves to the correct change
- [x] CHK-005 Skill arguments: all 5 workflow skills (`fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`, `fab-status`) document the `[change-name]` argument
- [x] CHK-006 Context docs: `_context.md` documents the optional override in the preflight invocation pattern

## Behavioral Correctness

- [x] CHK-007 Backward compatibility: preflight with no argument still reads `fab/current` and behaves identically to pre-change
- [x] CHK-008 Transient override: `fab/current` is never modified when `$1` is provided to preflight
- [x] CHK-009 Disambiguation: `/fab-continue` treats stage names as reset targets and other arguments as change-name overrides

## Scenario Coverage

- [x] CHK-010 Exact match: preflight resolves exact folder name match
- [x] CHK-011 Partial slug match: preflight resolves single partial match
- [x] CHK-012 Ambiguous match: preflight exits non-zero with list of matching folders
- [x] CHK-013 No match: preflight exits non-zero with descriptive error
- [x] CHK-014 Combined arguments: `/fab-continue` handles both change-name and stage arguments together

## Edge Cases & Error Handling

- [x] CHK-015 Archive exclusion: matching excludes `fab/changes/archive/` subdirectory
- [x] CHK-016 Missing .status.yaml: matched change directory missing `.status.yaml` produces appropriate error
- [x] CHK-017 Empty argument: `fab-preflight.sh ""` (empty string) falls back to `fab/current` behavior

## Documentation Accuracy

- [x] CHK-018 Skill arguments sections: each updated skill file accurately describes the `[change-name]` argument behavior
- [x] CHK-019 Execution order: skill files reference preflight invocation with the new argument pattern

## Cross References

- [x] CHK-020 Consistency: matching behavior described in skill files matches what preflight actually implements
- [x] CHK-021 Context preamble: `_context.md` preflight invocation pattern matches actual script signature

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
