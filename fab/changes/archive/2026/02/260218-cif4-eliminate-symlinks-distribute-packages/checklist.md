# Quality Checklist: Eliminate Symlinks, Distribute Packages via Kit

**Change**: 260218-cif4-eliminate-symlinks-distribute-packages
**Generated**: 2026-02-18
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Symlink deletion: All 5 symlinks in `src/lib/` are removed
- [x] CHK-002 Test preamble update: stageman, changeman, calc-score tests use `REPO_ROOT` pattern
- [x] CHK-003 Package move: `idea` and `wt` production code exists under `fab/.kit/packages/`
- [x] CHK-004 env-packages.sh: Script exists at `fab/.kit/scripts/env-packages.sh` and adds package bins to PATH
- [x] CHK-005 Scaffold envrc: `fab/.kit/scaffold/envrc` contains `source fab/.kit/scripts/env-packages.sh`
- [x] CHK-006 rc-init.sh delegation: `src/packages/rc-init.sh` delegates to `env-packages.sh`
- [x] CHK-007 Package test setup: idea and wt test suites find binaries at new location

## Behavioral Correctness

- [x] CHK-008 Test path resolution: stageman, changeman, calc-score tests resolve scripts without symlinks
- [x] CHK-009 Preflight tests unaffected: preflight test suite passes without changes
- [x] CHK-010 Sync-workspace tests unaffected: sync-workspace test suite passes without changes
- [x] CHK-011 wt relative sourcing: wt binaries still source `wt-common.sh` via relative path after move

## Scenario Coverage

- [x] CHK-012 Symlinks removed from working tree: `git status` shows 5 symlinks deleted
- [x] CHK-013 Package binaries in kit: `fab/.kit/packages/` contains idea and wt binaries
- [x] CHK-014 env-packages.sh no-packages scenario: sourcing with no packages/ dirs does not error
- [x] CHK-015 Test glob in justfile: `src/packages/*/tests` pattern still matches

## Edge Cases & Error Handling

- [x] CHK-016 env-packages.sh handles missing packages dir: no error when `fab/.kit/packages/` doesn't exist or has no `*/bin` dirs
- [x] CHK-017 git history preserved: `git log --follow` shows history for moved files

## Code Quality

- [x] CHK-018 Pattern consistency: New code follows naming and structural patterns of surrounding code
- [x] CHK-019 No unnecessary duplication: PATH setup centralized in env-packages.sh, not duplicated
- [x] CHK-020 Readability: env-packages.sh is concise and self-documenting
- [x] CHK-021 No god functions: All scripts stay under 50 lines

## Documentation Accuracy

- [x] CHK-022 README reflects new package location
- [x] CHK-023 No stale symlink references in script comments

## Cross References

- [x] CHK-024 Affected memory files identified: kit-architecture.md and distribution.md listed for hydrate update

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
