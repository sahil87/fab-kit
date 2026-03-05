# Quality Checklist: Add Conventions Section to config.yaml

**Change**: 260213-r3m7-add-conventions-section
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Conventions section schema: `config.yaml` contains a `conventions:` top-level key with `branch_naming`, `pr_title`, and `backlog` as commented-out example keys
- [x] CHK-002 Template documentation: Section includes header comment explaining purpose and per-key inline comments with example values
- [x] CHK-003 Centralized doc update: `fab/docs/fab-workflow/configuration.md` documents the `conventions` section with key definitions and purpose

## Behavioral Correctness
- [x] CHK-004 Section placement: `conventions:` appears after `source_paths:` and before `stages:` in `config.yaml`
- [x] CHK-005 Existing sections unchanged: `naming:`, `git:`, and all other existing config sections remain identical

## Scenario Coverage
- [x] CHK-006 Config with conventions: A config.yaml with populated conventions keys is valid YAML and loads without error
- [x] CHK-007 Config without conventions: Omitting the `conventions:` section entirely does not break any existing skill behavior
- [x] CHK-008 Partial conventions: A conventions section with only some keys present is valid YAML

## Edge Cases & Error Handling
- [x] CHK-009 All keys optional: No skill assumes conventions keys exist — the section and all keys within are optional

## Documentation Accuracy
- [x] CHK-010 Doc completeness: Configuration doc lists all three initial keys with name, type, and description
- [x] CHK-011 Relationship documented: Doc explains how `conventions` complements (not replaces) `naming` and `git`

## Cross References
- [x] CHK-012 Doc index consistency: `fab/docs/fab-workflow/index.md` entry for `configuration.md` remains accurate

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
