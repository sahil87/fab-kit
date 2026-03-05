# Quality Checklist: Replace .envrc Symlink with Line-Ensuring Sync

**Change**: 260216-ymvx-DEV-1043-envrc-line-sync
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Line-Ensuring Sync: `fab-sync.sh` section 2 reads `scaffold/envrc`, skips comments/empty lines, appends missing lines to `.envrc`
- [x] CHK-002 Symlink Migration: existing symlinks are resolved to real files before line-ensuring runs
- [x] CHK-003 Line Matching: uses `grep -qxF` for exact fixed-string matching (same as .gitignore section)

## Behavioral Correctness
- [x] CHK-004 New project: when no `.envrc` exists, a normal file is created with all required lines
- [x] CHK-005 Existing .envrc: user-added lines are preserved, only missing scaffold lines appended
- [x] CHK-006 All lines present: `.envrc` is not modified, script outputs `.envrc: OK`

## Removal Verification
- [x] CHK-007 Symlink logic removed: no `ln -s` calls remain in section 2
- [x] CHK-008 No symlink references: section 2 variable names and comments reflect file-based approach (no `envrc_link`, `envrc_target`)

## Scenario Coverage
- [x] CHK-009 Scenario: New project — no .envrc: file created with scaffold lines
- [x] CHK-010 Scenario: Existing .envrc all lines present: no modification, OK output
- [x] CHK-011 Scenario: Existing .envrc missing lines: lines appended
- [x] CHK-012 Scenario: Valid symlink migration: content resolved, symlink removed, real file written
- [x] CHK-013 Scenario: Broken symlink: removed, new file created from scaffold
- [x] CHK-014 Scenario: scaffold/envrc not found: section skipped

## Edge Cases & Error Handling
- [x] CHK-015 Empty scaffold/envrc: only comments/empty lines → no changes to `.envrc` (consistent with .gitignore behavior)
- [x] CHK-016 Guard clause: section 2 wrapped in `if [ -f "$envrc_entries" ]` (skips if scaffold missing)

## Code Quality
- [x] CHK-017 Pattern consistency: section 2 structure mirrors section 7 (.gitignore) — same variable naming, loop structure, output messages
- [x] CHK-018 No unnecessary duplication: reuses the same loop/grep pattern as .gitignore, not a new approach

## Documentation Accuracy
- [x] CHK-019 Section comment updated: section 2 header comment reflects line-ensuring behavior, not symlink

## Cross References
- [x] CHK-020 scaffold/envrc unchanged: file content remains the same 3 lines

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
