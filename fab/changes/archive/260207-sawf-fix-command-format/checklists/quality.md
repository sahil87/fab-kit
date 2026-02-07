# Quality Checklist: Fix command format — /fab:xxx to /fab-xxx

**Change**: 260207-sawf-fix-command-format
**Generated**: 2026-02-07
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Hyphen format in skills: All `/fab:` references in `fab/.kit/skills/*.md` and `_context.md` are replaced with `/fab-`
- [x] CHK-002 Hyphen format in scripts: All `/fab:` references in `fab/.kit/scripts/*.sh` are replaced with `/fab-`
- [x] CHK-003 Hyphen format in templates: All `/fab:` references in `fab/.kit/templates/*.md` are replaced with `/fab-`
- [x] CHK-004 Hyphen format in docs: All `/fab:` references in `fab/docs/**/*.md` are replaced with `/fab-`

## Behavioral Correctness
- [x] CHK-005 Next suggestions valid: "Next:" lines in skill files reference valid `/fab-xxx` skill names
- [x] CHK-006 Inline references valid: Prose references to commands in docs match actual skill names

## Removal Verification
- [x] CHK-007 No remaining colon format: Zero occurrences of `/fab:` remain in non-archived files under `fab/.kit/` and `fab/docs/`

## Scenario Coverage
- [x] CHK-008 Archived changes untouched: Files under `fab/changes/archive/` contain no modifications from this change
- [x] CHK-009 Non-command content preserved: YAML keys, file paths, and other non-command colons are unaffected

## Edge Cases & Error Handling
- [x] CHK-010 Backlog file: `fab/backlog.md` is checked and updated if it contains `/fab:` references
- [x] CHK-011 **N/A**: Other active change (`260207-m3qf`) contains `/fab:` in its proposal, but per spec, replacement scope is limited to `fab/.kit/` and `fab/docs/` — active change artifacts are transient and out of scope

## Documentation Accuracy
- [x] CHK-012 Doc index consistency: `fab/docs/index.md` references are valid after changes

## Cross References
- [x] CHK-013 Internal links: Cross-references between docs and skills remain consistent after replacement

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
