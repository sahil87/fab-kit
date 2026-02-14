# Quality Checklist: Consolidate .status.yaml Ownership into Stageman

**Change**: 260213-puow-consolidate-status-reads
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Progress Map Accessor: `get_progress_map` returns correct `stage:state` pairs for all 6 stages
- [x] CHK-002 Checklist Accessor: `get_checklist` returns `generated`, `completed`, `total` with correct defaults
- [x] CHK-003 Confidence Accessor: `get_confidence` returns all 5 fields with correct defaults
- [x] CHK-004 Rename: `_stageman.sh` exists, `stageman.sh` does not; all source lines updated
- [x] CHK-005 Resolve Change: `resolve_change` function handles all 6 scenarios (exact, substring single, substring multiple, no match, no override, missing changes dir)
- [x] CHK-006 Preflight Delegation: `fab-preflight.sh` uses stageman accessors and resolve-change; no inline `grep | sed` for progress/checklist/confidence
- [x] CHK-007 Status Delegation: `fab-status.sh` uses stageman accessors and resolve-change; `get_field`/`get_nested` helpers removed
- [x] CHK-008 get_current_stage Refactor: Uses `get_progress_map` internally; no raw `grep | sed`

## Behavioral Correctness

- [x] CHK-009 Preflight output identical: `fab-preflight.sh` YAML output is byte-for-byte identical to pre-refactor for the same input
- [x] CHK-010 Status output identical: `fab-status.sh` formatted output is identical to pre-refactor for the same input
- [x] CHK-011 get_current_stage fallback preserved: active → first pending after last done → hydrate fallback chain works

## Scenario Coverage

- [x] CHK-012 Progress map with missing stage defaults to `pending`
- [x] CHK-013 Checklist accessor with missing block returns defaults
- [x] CHK-014 Confidence accessor with missing block returns defaults (backwards compat)
- [x] CHK-015 Resolve change: exact match sets `RESOLVED_CHANGE_NAME`
- [x] CHK-016 Resolve change: single substring match resolves correctly
- [x] CHK-017 Resolve change: multiple matches returns non-zero with list
- [x] CHK-018 Resolve change: no `fab/current` returns non-zero
- [x] CHK-019 Resolve change: missing `fab/changes/` returns non-zero
- [x] CHK-020 Error messages are generic (no "Run /fab-new" in `_resolve-change.sh`)
- [x] CHK-021 Dev symlink `src/stageman/_stageman.sh` resolves correctly
- [x] CHK-022 Dev symlink `src/resolve-change/_resolve-change.sh` resolves correctly

## Edge Cases & Error Handling

- [x] CHK-023 `get_progress_map` handles `.status.yaml` with extra whitespace/indentation
- [x] CHK-024 `resolve_change` handles `fab/current` with trailing whitespace/newlines
- [x] CHK-025 `resolve_change` case-insensitive matching works

## Documentation Accuracy

- [x] CHK-026 `src/stageman/README.md` includes all 3 new accessor functions in API tables
- [x] CHK-027 `src/resolve-change/README.md` documents `resolve_change` interface accurately

## Cross References

- [x] CHK-028 No stale references to `stageman.sh` (should all be `_stageman.sh`)
- [x] CHK-029 Old `src/stageman/stageman.sh` symlink removed

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
