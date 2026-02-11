# Quality Checklist: Add `created_by` Attribution to Changes

**Change**: 260211-endg-add-created-by-field
**Generated**: 2026-02-11
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 `created_by` field present in `fab/.kit/templates/status.yaml` immediately after `created:`
- [x] CHK-002 `created_by` field documented in `fab/specs/templates.md` `.status.yaml` template section
- [x] CHK-003 `/fab-new` skill includes `created_by` in `.status.yaml` initialization
- [x] CHK-004 `/fab-discuss` skill includes `created_by` in `.status.yaml` initialization (new change mode)
- [x] CHK-005 `fab-status.sh` reads and displays `created_by`

## Behavioral Correctness

- [x] CHK-006 `created_by` value sourced from `git config user.name`
- [x] CHK-007 Fallback to `"unknown"` when git config unset
- [x] CHK-008 `/fab-discuss` refine mode does not modify existing `created_by` (write-once — refine mode only updates confidence block + `last_updated`, never touches `created_by`)

## Scenario Coverage

- [x] CHK-009 Status output shows `Created by:` between `Change:` and `Branch:` lines (verified via live test)
- [x] CHK-010 Status output omits `Created by:` when field is missing (verified via live test with old .status.yaml)

## Edge Cases & Error Handling

- [x] CHK-011 Missing `created_by` in archived changes does not cause errors in `/fab-status` (verified — `get_field` returns empty, `-n` test skips display)

## Documentation Accuracy

- [x] CHK-012 `fab/specs/templates.md` field notes accurately describe `created_by` behavior

## Cross References

- [x] CHK-013 `/fab-new` and `/fab-discuss` both use consistent `created_by` initialization pattern (both use `{git config user.name, or "unknown" if unset}`)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
