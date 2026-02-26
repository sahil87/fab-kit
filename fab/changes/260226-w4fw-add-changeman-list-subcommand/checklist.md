# Quality Checklist: Add changeman.sh list Subcommand

**Change**: 260226-w4fw-add-changeman-list-subcommand
**Generated**: 2026-02-26
**Spec**: `spec.md`

## Functional Completeness
- [ ] CHK-001 Enumerate Active Changes: `changeman.sh list` outputs one `name:display_stage:display_state` line per change directory in `fab/changes/` (excluding `archive/`)
- [ ] CHK-002 Empty Result: exits 0 with no stdout when no changes exist
- [ ] CHK-003 Missing `.status.yaml` Handling: outputs `name:unknown:unknown` with stderr warning, does not fail the list
- [ ] CHK-004 Archive Listing: `--archive` flag scans `fab/changes/archive/` instead
- [ ] CHK-005 Stage Derivation: uses `stageman.sh display-stage` for all stage/state values
- [ ] CHK-006 Help Text: `changeman.sh --help` includes `list` in USAGE and SUBCOMMANDS

## Behavioral Correctness
- [ ] CHK-007 Existing subcommands (`new`, `rename`, `resolve`, `switch`) still function correctly after adding `list`

## Scenario Coverage
- [ ] CHK-008 Multiple Active Changes: two+ changes listed correctly
- [ ] CHK-009 Single Active Change: one change listed correctly
- [ ] CHK-010 No Changes: empty stdout, exit 0
- [ ] CHK-011 Changes Directory Missing: stderr error, exit 1
- [ ] CHK-012 One Change Missing Status: mixed output (good + unknown), stderr warning
- [ ] CHK-013 List Archived Changes: `--archive` lists from archive directory
- [ ] CHK-014 No Archived Changes: `--archive` with empty archive, exit 0

## Edge Cases & Error Handling
- [ ] CHK-015 `fab/changes/` contains only `archive/` directory: treated as empty (exit 0, no output)
- [ ] CHK-016 `stageman.sh display-stage` failure for a change: graceful fallback to `unknown:unknown`

## Code Quality
- [ ] CHK-017 Pattern consistency: `cmd_list` follows the structure and conventions of `cmd_resolve`, `cmd_switch`, and other existing functions
- [ ] CHK-018 No unnecessary duplication: reuses existing `STAGEMAN` variable and path resolution patterns
- [ ] CHK-019 Readability: function is under 50 lines, clear control flow
- [ ] CHK-020 No magic strings: uses named variables for format components

## Documentation Accuracy
- [ ] CHK-021 Help text accurately describes `list` usage and behavior
- [ ] CHK-022 Memory file update reflects actual implementation

## Cross References
- [ ] CHK-023 `changeman.sh` header comment includes `list` in the usage summary

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
