# Quality Checklist: Batch Script Frontmatter for fab-help Discovery

**Change**: 260221-alng-batch-script-frontmatter
**Generated**: 2026-02-21
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 shell_frontmatter_field function: `shell_frontmatter_field <file> <field>` returns the unquoted value from `# ---` delimited blocks
- [x] CHK-002 Batch script frontmatter: All 3 batch scripts have `# ---` frontmatter blocks with `name` and `description` fields
- [x] CHK-003 fab-help.sh discovery: Running `fab-help.sh` shows a "Batch Operations" group with all 3 batch scripts listed
- [x] CHK-004 Centralized group mapping: Batch script group assignment is in `fab-help.sh` (not in frontmatter)
- [x] CHK-005 Alignment consistency: Batch script names are included in `max_len` calculation for column alignment

## Behavioral Correctness
- [x] CHK-006 Old comment headers removed: The `# script-name — description` comment lines are removed from all 3 batch scripts (replaced by frontmatter)
- [x] CHK-007 No `/` prefix on batch scripts: Batch script display names appear without leading `/` (they are shell commands, not slash-commands)
- [x] CHK-008 Existing frontmatter_field unchanged: The original `frontmatter_field` function in `frontmatter.sh` is not modified

## Scenario Coverage
- [x] CHK-009 Parse name field: `shell_frontmatter_field <file> name` returns correct unquoted name
- [x] CHK-010 Parse quoted description: `shell_frontmatter_field <file> description` returns value with quotes stripped
- [x] CHK-011 Script without frontmatter: `shell_frontmatter_field` returns empty string for files without `# ---` block
- [x] CHK-012 Bare `---` not matched: `shell_frontmatter_field` ignores non-`#`-prefixed `---` markers
- [x] CHK-013 Batch script without frontmatter skipped: A `batch-*.sh` file without frontmatter does not appear in help output

## Edge Cases & Error Handling
- [x] CHK-014 Field not found: `shell_frontmatter_field <file> nonexistent` returns empty string
- [x] CHK-015 Missing name or description: Batch scripts without `name` or `description` in frontmatter are skipped by the scan loop

## Code Quality
- [x] CHK-016 Pattern consistency: `shell_frontmatter_field` follows the same coding patterns as `frontmatter_field` (sed-based, no external dependencies)
- [x] CHK-017 No unnecessary duplication: The scan loop reuses `format_entry` and existing rendering patterns from `fab-help.sh`
- [x] CHK-018 No god functions: New code segments are focused and concise

## Documentation Accuracy
- [x] CHK-019 Frontmatter content matches spec: Script names and descriptions match the table in spec.md exactly

## Cross References
- [x] CHK-020 Spec traceability: Each requirement in spec.md has at least one corresponding task and checklist item

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
