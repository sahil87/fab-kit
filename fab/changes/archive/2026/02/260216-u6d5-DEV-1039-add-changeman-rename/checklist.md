# Quality Checklist: Add Rename Subcommand to changeman.sh

**Change**: 260216-u6d5-DEV-1039-add-changeman-rename
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Rename Interface: `changeman.sh rename --folder <name> --slug <slug>` works end-to-end
- [x] CHK-002 Prefix Preservation: date-ID prefix extracted and preserved, only slug replaced
- [x] CHK-003 Slug Validation: same regex as `new` (`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
- [x] CHK-004 Source Folder Validation: errors when source folder doesn't exist
- [x] CHK-005 Destination Collision Detection: errors when target folder already exists
- [x] CHK-006 Same-Name Detection: errors when new slug produces identical folder name
- [x] CHK-007 Status File Update: `.status.yaml` `name` field updated after rename
- [x] CHK-008 Active Change Pointer Update: `fab/current` updated when pointing to renamed change
- [x] CHK-009 Stageman Logging: `log-command` called on successful rename
- [x] CHK-010 Help Text: `show_help()` documents rename subcommand
- [x] CHK-011 CLI Dispatch: `rename` subcommand routed to `cmd_rename`

## Behavioral Correctness

- [x] CHK-012 fab/current not modified when pointing to different change
- [x] CHK-013 fab/current not created when absent

## Scenario Coverage

- [x] CHK-014 Basic rename: folder renamed, correct output, .status.yaml updated
- [x] CHK-015 Missing --folder flag: error with usage hint
- [x] CHK-016 Missing --slug flag: error with usage hint
- [x] CHK-017 Slug with uppercase (Linear ID): accepted
- [x] CHK-018 Slug with leading/trailing hyphen: rejected

## Edge Cases & Error Handling

- [x] CHK-019 Missing source folder: `ERROR: Change folder '...' not found`
- [x] CHK-020 Destination collision: `ERROR: Folder '...' already exists`
- [x] CHK-021 Same name: `ERROR: New name is the same as current name`

## Code Quality

- [x] CHK-022 Pattern consistency: `cmd_rename` follows same structure as `cmd_new` (arg parsing, validation, error messages)
- [x] CHK-023 No unnecessary duplication: reuses existing slug validation pattern

## Documentation Accuracy

- [x] CHK-024 SPEC-changeman.md: rename subcommand fully documented in API Reference
- [x] CHK-025 Help text: matches actual rename interface

## Cross References

- [x] CHK-026 Spec scenarios covered by corresponding test cases

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
