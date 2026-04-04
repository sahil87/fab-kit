# Quality Checklist: Sync From Cache

**Change**: 260402-ktbg-sync-from-cache
**Generated**: 2026-04-02
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Cache resolution: `Sync()` reads all kit content from `CachedKitDir(fab_version)`, not from `fab/.kit/`
- [x] CHK-002 Version guard: sync verifies `fab_version <= system version` and attempts auto-update when insufficient
- [x] CHK-003 Ensure cache: `EnsureCached(fab_version)` is called before any kit content is accessed
- [x] CHK-004 Scaffold tree-walk: scaffold files are read from `{cachedKitDir}/scaffold/`
- [x] CHK-005 Skill deployment: skills are read from `{cachedKitDir}/skills/` for all agents
- [x] CHK-006 Hook sync absorbed: hook registration runs as Go code within step 4, no shell script invoked
- [x] CHK-007 `--shim` flag: steps 1-5 execute, step 6 skipped
- [x] CHK-008 `--project` flag: steps 1-5 skipped, step 6 executes
- [x] CHK-009 fragment-.envrc fix: scaffold contains `WORKTREE_INIT_SCRIPT="fab sync"`
- [x] CHK-010 Migration file: exists and replaces `fab-kit sync` with `fab sync` in `.envrc`

## Behavioral Correctness

- [x] CHK-011 6-step pipeline: steps execute in specified order (prereqs → version guard → cache → scaffold → direnv → project scripts)
- [x] CHK-012 Flag mutual exclusion: `--shim` and `--project` together produce error
- [x] CHK-013 Hook sync idempotent: re-running sync does not duplicate hook entries in `settings.local.json`
- [x] CHK-014 Prerequisites updated: `jq` and `gh` removed from required tools list

## Removal Verification

- [x] CHK-015 `5-sync-hooks.sh` deleted: `fab/.kit/sync/5-sync-hooks.sh` no longer exists
- [x] CHK-016 No repo `kitDir` reads: `Sync()` does not read from `fab/.kit/` for any kit content

## Scenario Coverage

- [x] CHK-017 Normal sync: full pipeline works with cached version
- [x] CHK-018 System version too old: version guard triggers update or fails with clear message
- [x] CHK-019 Version not cached: `EnsureCached` downloads before proceeding
- [x] CHK-020 Config missing: sync exits with appropriate error message

## Edge Cases & Error Handling

- [x] CHK-021 Local-versions priority: when both caches exist, local-versions is preferred
- [x] CHK-022 Download failure: sync exits with error including version and network hint
- [x] CHK-023 Missing config.yaml: sync errors with "Not in a fab-managed repo" message

## Code Quality

- [x] CHK-024 Pattern consistency: new code follows naming and structural patterns of existing sync.go functions
- [x] CHK-025 No unnecessary duplication: hook sync replication is minimal and self-contained
- [x] CHK-026 No god functions: new functions stay under 50 lines where possible

## Documentation Accuracy

- [x] CHK-027 Migration file follows standard format (Summary, Pre-check, Changes, Verification)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
