# Quality Checklist: Rename Scaffold & Add Kit Script Tests

**Change**: 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests
**Generated**: 2026-02-16
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 File Rename: `init-scaffold.sh` no longer exists, `sync-workspace.sh` exists and is executable
- [x] CHK-002 Worktree Hook Rename: `2-rerun-init-scaffold.sh` no longer exists, `2-rerun-sync-workspace.sh` exists and calls `sync-workspace.sh`
- [x] CHK-003 Reference Update: Zero matches for `init-scaffold` outside `fab/changes/archive/` and change artifacts
- [x] CHK-004 Bats Available: `bats` invoked successfully, both `.bats` suites execute (52 tests pass)
- [x] CHK-005 sync-workspace Directory: `src/lib/sync-workspace/` contains symlink, SPEC, and test.bats
- [x] CHK-006 changeman Directory: `src/lib/changeman/` contains symlink, SPEC, and test.bats
- [x] CHK-007 Justfile Recipes: `test-bash`, `test-rust`, and `test` recipes all exist and are functional

## Behavioral Correctness

- [x] CHK-008 Script Identity: `sync-workspace.sh` is the renamed file — only header comment changed, all behavior identical
- [x] CHK-009 Symlink Validity: Both dev symlinks resolve to their target scripts in `fab/.kit/scripts/lib/`
- [x] CHK-010 Test Runner Summary: `just test` displays per-suite pass/fail with `N/M suites passed` and overall verdict

## Scenario Coverage

- [x] CHK-011 sync-workspace tests cover: directory creation (4), VERSION logic (3), .envrc (3), index seeding (3), skill sync (4), agent generation (3), .gitignore (3), idempotency (2), error cases (2) — 28 tests total
- [x] CHK-012 changeman tests cover: new happy path (6), slug validation (4), change-id validation (3), random ID (2), collision detection (1), --help (1), error cases (5), detect_created_by (1), stageman integration (via happy path tests) — 24 tests total
- [x] CHK-013 Archived changes untouched: `git status -- fab/changes/archive/` shows no modifications
- [x] CHK-014 Idempotency tests: sync-workspace test suite includes "running twice produces no errors" and "running twice produces same file structure"

## Edge Cases & Error Handling

- [x] CHK-015 changeman: Missing slug produces "--slug is required" error
- [x] CHK-016 changeman: Leading/trailing hyphens produce "Invalid slug format" error
- [x] CHK-017 changeman: Provided change-id collision is fatal with "already in use" message
- [x] CHK-018 Test runner: Failure in one suite does not prevent others from running (verified — 3 legacy suites have pre-existing failures, but all 6 suites run)

## Code Quality

- [x] CHK-019 Pattern consistency: Test directories follow `src/lib/*/` layout exactly (symlink, SPEC, test file)
- [x] CHK-020 No unnecessary duplication: Bats test files use proper setup/teardown with isolated temp dirs

## Documentation Accuracy

- [x] CHK-021 SPEC files: Both SPEC files follow established format (Sources of Truth, Usage, API/Behavior Reference, Requirements, Testing)
- [x] CHK-022 Memory files: `kit-architecture.md` and `distribution.md` references updated to `sync-workspace.sh`

## Cross References

- [x] CHK-023 All memory file cross-references to `init-scaffold.sh` updated consistently (kit-architecture, distribution, init, hydrate, model-tiers, templates, preflight, migrations, index)
- [x] CHK-024 README.md references updated to `sync-workspace.sh`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
