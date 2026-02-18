# Quality Checklist: Scaffold Overlay Tree

**Change**: 260218-09fa-scaffold-overlay-tree
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Scaffold overlay tree: All 11 files exist at their new overlay paths, old flat paths removed
- [x] CHK-002 Fragment prefix: Exactly 3 files have `fragment-` prefix (.envrc, .gitignore, settings.local.json)
- [x] CHK-003 Generic tree-walk: Sections 2, 3, 4, 7, 8, 9 replaced by single tree-walk in `3-sync-workspace.sh`
- [x] CHK-004 Helper functions: `line_ensure_merge` and `json_merge_permissions` defined in `3-sync-workspace.sh`
- [x] CHK-005 fab-setup path references: All 7 scaffold paths updated to new overlay structure
- [x] CHK-006 fab-setup template detection: Bootstrap steps 1a/1b check for placeholders instead of just existence
- [x] CHK-007 Migration path reference: `0.7.0-to-0.8.0.md` references `fab/.kit/scaffold/fab/code-quality.md`

## Behavioral Correctness

- [x] CHK-008 Line-ensuring merge: `.envrc` and `.gitignore` entries are appended if missing, existing content preserved
- [x] CHK-009 JSON merge: `permissions.allow` arrays merged correctly, existing permissions preserved
- [x] CHK-010 Copy-if-absent: Non-fragment files copied only when target doesn't exist
- [x] CHK-011 Template files: config.yaml and constitution.md copied by tree-walk, fab-setup detects and overwrites them

## Scenario Coverage

- [x] CHK-012 Symlink migration: `line_ensure_merge` handles symlink target (resolve to real file, preserve content)
- [x] CHK-013 Missing jq: `json_merge_permissions` warns and skips when jq not available
- [x] CHK-014 Intermediate directories: Tree-walk creates parent directories as needed (e.g., `.claude/`)
- [x] CHK-015 Empty scaffold: Tree-walk handles empty scaffold directory without error

## Edge Cases & Error Handling

- [x] CHK-016 No regressions: Sections 1, 1b, 3, 3b, 4 unchanged and still functional
- [x] CHK-017 Idempotent: Running `fab-sync.sh` twice produces same result, no duplicate entries

## Code Quality

- [x] CHK-018 Pattern consistency: New code follows naming and structural patterns of `3-sync-workspace.sh`
- [x] CHK-019 No unnecessary duplication: Merge logic in helper functions, not duplicated per file
- [x] CHK-020 Readability: Code follows principles from `fab/code-quality.md` — no god functions, no magic strings

## Documentation Accuracy

- [x] CHK-021 Spec mapping table matches actual file locations in scaffold directory
- [x] CHK-022 fab-setup.md references match actual scaffold file paths

## Cross References

- [x] CHK-023 All files referencing old scaffold paths have been updated (fab-setup.md, migration file)
- [x] CHK-024 No stale references to old flat scaffold filenames in kit scripts

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
