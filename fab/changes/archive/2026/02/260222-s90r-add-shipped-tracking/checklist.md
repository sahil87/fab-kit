# Quality Checklist: Add Shipped Tracking

**Change**: 260222-s90r-add-shipped-tracking
**Generated**: 2026-02-22
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 `ship` subcommand: Appends URL to `shipped` array in `.status.yaml`
- [ ] CHK-002 `ship` subcommand: Creates `shipped` key if missing
- [ ] CHK-003 `ship` subcommand: Deduplicates URLs silently
- [ ] CHK-004 `ship` subcommand: Updates `last_updated` timestamp
- [ ] CHK-005 `ship` subcommand: Uses atomic write (temp file → mv)
- [ ] CHK-006 `is-shipped` subcommand: Exit 0 when shipped array has entries
- [ ] CHK-007 `is-shipped` subcommand: Exit 1 when shipped array is empty or missing
- [ ] CHK-008 `is-shipped` subcommand: No stdout output
- [ ] CHK-009 CLI dispatch: `ship` and `is-shipped` registered in case block
- [ ] CHK-010 Help text: Both subcommands documented in `--help` output
- [ ] CHK-011 Template: `shipped: []` present in `fab/.kit/templates/status.yaml`
- [ ] CHK-012 Schema: `shipped` field documented in `fab/.kit/schemas/workflow.yaml`
- [ ] CHK-013 `/git-pr`: Calls `stageman.sh ship` after PR creation
- [ ] CHK-014 `/git-pr`: Graceful skip when no active change
- [ ] CHK-015 State table: Hydrate row routes to `/git-pr` as default in `_preamble.md`
- [ ] CHK-016 `changeman.sh`: `default_command hydrate` returns `/git-pr`

## Scenario Coverage

- [ ] CHK-017 First ship call on fresh status file
- [ ] CHK-018 Append second URL preserves order
- [ ] CHK-019 Duplicate URL is idempotent
- [ ] CHK-020 Missing shipped key creates the key
- [ ] CHK-021 Missing status file returns error

## Edge Cases & Error Handling

- [ ] CHK-022 `ship` with non-existent file: exits 1 with "Status file not found"
- [ ] CHK-023 `is-shipped` with non-existent file: exits 1 with "Status file not found"

## Code Quality

- [ ] CHK-024 Pattern consistency: New functions follow stageman.sh existing patterns (validation, atomic write, `last_updated` refresh)
- [ ] CHK-025 No unnecessary duplication: Reuses existing stageman.sh patterns rather than reimplementing

## Documentation Accuracy

- [ ] CHK-026 workflow.yaml documents shipped field correctly (type, semantics)
- [ ] CHK-027 Help text follows existing format conventions

## Cross References

- [ ] CHK-028 _preamble.md state table consistent with changeman.sh default_command
- [ ] CHK-029 git-pr skill references correct script paths

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
