# Quality Checklist: Reorganize src/ and kit script internals

**Change**: 260214-q7f2-reorganize-src
**Generated**: 2026-02-14
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 src/lib/ structure: All 4 test dirs exist under `src/lib/` and not under `src/`
- [x] CHK-002 src/scripts/ structure: `fab-release.sh` exists at `src/scripts/fab-release.sh`
- [x] CHK-003 Kit lib/ structure: All 5 internal scripts exist under `fab/.kit/scripts/lib/` with no underscore prefix
- [x] CHK-004 Kit scripts/ cleaned: No underscore-prefixed scripts remain in `fab/.kit/scripts/`
- [x] CHK-005 fab-release.sh removed from kit: `fab/.kit/scripts/fab-release.sh` no longer exists
- [x] CHK-006 Justfile updated: Test glob is `src/lib/*/test.sh`
- [x] CHK-007 Dev .envrc updated: `PATH_add src/scripts` present

## Behavioral Correctness

- [x] CHK-008 Symlinks valid: All 4 `src/lib/*/` symlinks resolve correctly to `fab/.kit/scripts/lib/` targets
- [x] CHK-009 preflight sources updated: `lib/preflight.sh` sources `stageman.sh` and `resolve-change.sh` from `lib/`
- [x] CHK-010 calc-score sources updated: `lib/calc-score.sh` sources `stageman.sh` from `lib/`
- [x] CHK-011 fab-upgrade.sh calls lib/init-scaffold.sh
- [x] CHK-012 batch-archive-change.sh sources lib/resolve-change.sh
- [x] CHK-013 batch-switch-change.sh sources lib/resolve-change.sh

## Removal Verification

- [x] CHK-014 Underscore prefix deprecated: No `_`-prefixed scripts in `fab/.kit/scripts/` (all moved to `lib/`)
- [x] CHK-015 fab-release.sh removed from kit: Not present in `fab/.kit/scripts/`

## Scenario Coverage

- [x] CHK-016 Tests discovered: `just test` finds and runs all 4 test suites from `src/lib/`
- [x] CHK-017 Scaffold .envrc unchanged: `fab/.kit/scaffold/envrc` has no modifications

## Edge Cases & Error Handling

- [x] CHK-018 No broken symlinks: All symlinks in `src/lib/*/` point to existing targets
- [x] CHK-019 Self-referencing scripts: `_stageman.sh` internal comment/doc references updated to reflect new path

## Documentation Accuracy

- [x] CHK-020 Skill _context.md: All internal script path references updated
- [x] CHK-021 Skill fab-continue.md: calc-score and stageman references updated
- [x] CHK-022 Skill fab-init.md: init-scaffold references updated
- [x] CHK-023 Skill fab-status.md: preflight reference updated
- [x] CHK-024 Skill fab-archive.md: preflight reference updated
- [x] CHK-025 Skill fab-clarify.md: calc-score reference updated
- [x] CHK-026 Skill fab-ff.md: stageman references updated
- [x] CHK-027 Skill fab-fff.md: stageman references updated
- [x] CHK-028 Skill _generation.md: stageman references updated

## Cross References

- [x] CHK-029 No stale references: grep for `_preflight.sh`, `_calc-score.sh`, `_stageman.sh`, `_resolve-change.sh`, `_init_scaffold.sh` returns zero hits outside of git history and this change's artifacts

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
