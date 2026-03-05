# Quality Checklist: Extract scaffold content into fab/.kit/scaffold/

**Change**: 260213-3njv-scaffold-dir
**Generated**: 2026-02-13
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Scaffold directory: `fab/.kit/scaffold/` exists with `envrc`, `gitignore-entries`, `docs-index.md`, `design-index.md`
- [x] CHK-002 Script updated: `_fab-scaffold.sh` reads from scaffold files — no remaining heredocs for docs/index.md or design/index.md sections
- [x] CHK-003 Old envrc removed: `fab/.kit/envrc` does not exist
- [x] CHK-004 Kit-architecture doc: `scaffold/` appears in directory tree listing
- [x] CHK-005 Init doc: delegation table rows reference scaffold paths
- [x] CHK-006 Distribution doc: bootstrap description mentions scaffold files

## Behavioral Correctness
- [x] CHK-007 .envrc symlink: new symlinks target `fab/.kit/scaffold/envrc` (not `fab/.kit/envrc`)
- [x] CHK-008 docs/index.md content: file created from `scaffold/docs-index.md` matches expected initial content
- [x] CHK-009 design/index.md content: file created from `scaffold/design-index.md` matches expected initial content
- [x] CHK-010 .gitignore loop: processes entries from `scaffold/gitignore-entries` correctly (each entry checked independently)

## Scenario Coverage
- [x] CHK-011 Fresh install: running `_fab-scaffold.sh` creates `.envrc` → `fab/.kit/scaffold/envrc`
- [x] CHK-012 Broken symlink repair: `.envrc` pointing to old `fab/.kit/envrc` is repaired to `fab/.kit/scaffold/envrc` — tested live, broken symlink case also added to script
- [x] CHK-013 Idempotent: existing `docs/index.md` and `design/index.md` not overwritten on re-run — verified via second run
- [x] CHK-014 .gitignore append: missing entries appended, existing entries not duplicated — uses `grep -qxF` for exact match
- [x] CHK-015 .gitignore create: if no `.gitignore` exists, created with all scaffold entries — tracked via `gitignore_existed` flag

## Edge Cases & Error Handling
- [x] CHK-016 Comment lines: lines starting with `#` in `gitignore-entries` are skipped — `[[ "$entry" == \#* ]]` pattern
- [x] CHK-017 Empty lines: blank lines in `gitignore-entries` are skipped — `[[ -z "$entry" ]]` check

## Documentation Accuracy
- [x] CHK-018 Kit-architecture directory tree accurately reflects actual `fab/.kit/` structure after change
- [x] CHK-019 Init delegation table paths match actual `_fab-scaffold.sh` behavior

## Cross References
- [x] CHK-020 All three centralized docs (kit-architecture, init, distribution) have changelog entries for this change

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (archive)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
