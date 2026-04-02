# Quality Checklist: Brew Install System Shim

**Change**: 260401-46hw-brew-install-system-shim
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness
- [x] CHK-001 Homebrew formula: `Formula/fab-kit.rb` installs `fab`, `wt`, `idea`
- [x] CHK-002 Shim dispatch: `fab` resolves `fab_version`, finds cached binary, execs with argument passthrough
- [x] CHK-003 Auto-fetch: shim downloads and caches version on first invocation when not cached
- [x] CHK-004 Cache layout: `~/.fab-kit/versions/{v}/fab-go` and `~/.fab-kit/versions/{v}/kit/` populated correctly
- [x] CHK-005 `fab init`: scaffolds new project, populates `.kit/`, sets `fab_version`, runs sync
- [x] CHK-006 `fab upgrade`: downloads to cache, atomic swap of `.kit/`, updates `fab_version`, runs sync
- [x] CHK-007 `fab_version` field: added to `config.yaml`, read by shim
- [x] CHK-008 Skill path change: all `fab/.kit/bin/fab` references replaced with `fab` across skill files
- [x] CHK-009 Hook scripts: all `fab/.kit/bin/fab` or `$kit_dir/bin/fab` references replaced with `fab`
- [x] CHK-010 Sync pipeline: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` updated
- [x] CHK-011 `.envrc` scaffold: `PATH_add fab/.kit/bin` removed, `PATH_add fab/.kit/scripts` retained
- [x] CHK-012 `fab-doctor.sh`: `fab` system binary check added
- [x] CHK-013 `fab-upgrade.sh`: removed from `fab/.kit/scripts/`
- [x] CHK-014 Binary cleanup: `fab/.kit/bin/` contains only `.gitkeep`
- [x] CHK-015 Constitution: Principle V amended to require system shim
- [x] CHK-016 Migration file: created with prerequisite gate, `fab_version` addition, `.envrc` cleanup, binary removal

## Behavioral Correctness
- [x] CHK-017 Shim error when `fab_version` absent: actionable message mentioning `fab init`
- [x] CHK-018 Shim error outside fab repo: actionable message for non-repo commands vs repo commands
- [x] CHK-019 `fab upgrade` already up-to-date: shows "No update needed" without modifying files
- [x] CHK-020 `fab upgrade` migration reminder: displays when `.kit-migration-version` < new version

## Removal Verification
- [x] CHK-021 Backend override removed: no references to `FAB_BACKEND` or `.fab-backend` in active code
- [x] CHK-022 Shell dispatcher removed: `fab/.kit/bin/fab` (shell script) no longer exists
- [x] CHK-023 `fab-upgrade.sh` removed: `fab/.kit/scripts/fab-upgrade.sh` no longer exists
- [x] CHK-024 `4-get-fab-binary.sh` removed: `fab/.kit/sync/4-get-fab-binary.sh` no longer exists

## Scenario Coverage
- [x] CHK-025 Normal dispatch: shim finds config, resolves version, dispatches to cached binary
- [x] CHK-026 Auto-fetch on cache miss: shim downloads, caches, then dispatches
- [x] CHK-027 Multiple versions coexist: two repos with different `fab_version` both work
- [x] CHK-028 Init new repo: `fab init` in a clean repo produces working fab setup
- [x] CHK-029 Upgrade with version argument: `fab upgrade 0.42.1` downgrades correctly
- [x] CHK-030 Sync after upgrade: `fab-sync.sh` deploys skills correctly without binary step

## Edge Cases & Error Handling
- [x] CHK-031 No network during auto-fetch: shim exits with actionable error
- [x] CHK-032 Missing `config.yaml`: shim handles non-repo context correctly
- [x] CHK-033 Init with existing `fab/` but no `fab_version`: adds field without overwriting project files

## Code Quality
- [x] CHK-034 Pattern consistency: shim Go code follows patterns in `src/go/fab/` (Cobra, internal packages)
- [x] CHK-035 No unnecessary duplication: shim reuses Go patterns from existing `src/go/` modules
- [x] CHK-036 Readability: shim code is straightforward — no clever abstractions for simple operations

## Documentation Accuracy
- [x] CHK-037 README: install instructions updated to `brew install fab-kit` + `fab init`
- [x] CHK-038 `_cli-fab.md`: all invocation examples use `fab` not `fab/.kit/bin/fab`

## Cross References
- [x] CHK-039 **N/A**: Spec file references — skill changes in this change are invocation path updates, not behavioral changes requiring spec updates
- [x] CHK-040 Memory file references: affected memory domains listed correctly in intake and spec

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
