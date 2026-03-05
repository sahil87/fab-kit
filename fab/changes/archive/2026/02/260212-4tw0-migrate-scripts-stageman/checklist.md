# Quality Checklist: Migrate Scripts to Use Stage Manager

**Change**: 260212-4tw0-migrate-scripts-stageman
**Generated**: 2026-02-12
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fab-status.sh sources stageman.sh and all hardcoded stage/state logic is removed
- [x] CHK-002 fab-preflight.sh sources stageman.sh and all hardcoded stage iteration is removed
- [x] CHK-003 fab-preflight.sh calls validate_status_file and exits non-zero on invalid status
- [x] CHK-004 fab-help.sh is unchanged (static help text preserved per spec)
- [x] CHK-005 MIGRATION.md moved to change folder, removed from fab/.kit/schemas/
- [x] CHK-006 schemas/README.md moved to fab/docs/fab-workflow/schemas.md, removed from fab/.kit/schemas/
- [x] CHK-007 src/stageman/ consolidated: SUMMARY.md, SPEC.md, CHANGELOG.md deleted, README.md rewritten
- [x] CHK-008 All dangling references updated (root README, stageman.sh help, docs/index.md)

## Behavioral Correctness

- [x] CHK-009 fab-status.sh output format is byte-identical before/after migration
- [x] CHK-010 fab-preflight.sh YAML output structure is identical before/after migration
- [x] CHK-011 Stage count in display (`N/6`) is computed dynamically, not hardcoded

## Removal Verification

- [x] CHK-012 No hardcoded stage lists remain in fab-status.sh (no `brief spec tasks apply review archive` literal)
- [x] CHK-013 No hardcoded stage-number case statement remains in fab-status.sh
- [x] CHK-014 No hardcoded symbol() function remains in fab-status.sh
- [x] CHK-015 No hardcoded per-stage progress variables (p_brief, p_spec, etc.) remain in fab-status.sh
- [x] CHK-016 No hardcoded stage list remains in fab-preflight.sh progress extraction
- [x] CHK-017 src/stageman/SUMMARY.md, SPEC.md, CHANGELOG.md are deleted
- [x] CHK-018 fab/.kit/schemas/README.md and MIGRATION.md are deleted

## Scenario Coverage

- [x] CHK-019 fab-status.sh runs successfully with an active change and displays correct progress
- [x] CHK-020 fab-preflight.sh runs successfully and emits valid YAML with all stage fields
- [x] CHK-021 fab-preflight.sh catches invalid .status.yaml via validate_status_file
- [x] CHK-022 stageman.sh --test passes all self-tests
- [x] CHK-023 src/stageman/test-simple.sh passes

## Edge Cases & Error Handling

- [x] CHK-024 fab-status.sh handles missing .status.yaml gracefully (existing behavior preserved)
- [x] CHK-025 fab-status.sh handles no active change gracefully (existing behavior preserved)
- [x] CHK-026 fab-preflight.sh migration shim for missing brief field still works

## Documentation Accuracy

- [x] CHK-027 fab/docs/fab-workflow/schemas.md contains no stageman API function signatures (no duplication)
- [x] CHK-028 src/stageman/README.md contains complete API reference (nothing lost from SPEC.md)
- [x] CHK-029 fab/docs/fab-workflow/index.md includes schemas entry

## Cross References

- [x] CHK-030 grep for SPEC.md, SUMMARY.md, CHANGELOG.md, schemas/README across *.md and *.sh returns no unexpected matches

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
