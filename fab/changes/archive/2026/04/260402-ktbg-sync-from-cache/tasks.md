# Tasks: Sync From Cache

**Change**: 260402-ktbg-sync-from-cache
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Add `--shim` and `--project` flags to the sync cobra command in `src/go/fab-kit/cmd/fab-kit/main.go`. Update `syncCmd()` to accept boolean flags and pass them to `internal.Sync()`. Change `Sync()` signature to `Sync(shimOnly, projectOnly bool) error` with mutual-exclusion guard.

- [x] T002 [P] Fix `fab/.kit/scaffold/fragment-.envrc`: change `export WORKTREE_INIT_SCRIPT="fab-kit sync"` to `export WORKTREE_INIT_SCRIPT="fab sync"`.

- [x] T003 [P] ~~Create migration file `fab/.kit/migrations/0.44.0-to-0.45.0.md`~~ — already exists with the correct content (Summary, Pre-check, Changes, Verification for `.envrc` fix). No action needed. <!-- clarified: migration file 0.44.0-to-0.45.0.md already exists in the repo with exact content matching the task description -->

## Phase 2: Core Implementation

- [x] T004 Add version guard to `src/go/fab-kit/internal/sync.go`: new function `versionGuard(fabVersion, systemVersion string) error` that compares semver strings. If `fabVersion > systemVersion`, attempt `Update(systemVersion)` (call the existing update logic). If still insufficient after update, return error. The `systemVersion` is passed into `Sync()` from `main.go` (where `version` is already available via ldflags).

- [x] T005 Add cache resolution to `Sync()` in `src/go/fab-kit/internal/sync.go`: after version guard, call `EnsureCached(fabVersion)` then `CachedKitDir(fabVersion)` to get the cached kit path. Use this `cachedKitDir` as the source for all subsequent operations instead of the repo-local `kitDir`. Resolve `fabVersion` via `ResolveConfig()` (already in `config.go`).

- [x] T006 Refactor `Sync()` to use `cachedKitDir` instead of `kitDir` for: `scaffoldTreeWalk()` (pass `{cachedKitDir}/scaffold/`), `deploySkills()` (pass `{cachedKitDir}/skills/` via the `kitDir` parameter), `cleanLegacyAgents()` (pass `{cachedKitDir}` for skill list lookup), and `scaffoldDirectories()` (pass `{cachedKitDir}` for VERSION reading). The `kitDir` local variable should be renamed or removed to prevent accidental use of the repo path.

- [x] T007 Replicate hook sync logic in `src/go/fab-kit/internal/hooksync.go` (new file). Copy the mapping table, entry structures, merge logic, and deduplication from `src/go/fab/internal/hooklib/sync.go`. Adapt `Sync()` to accept `hooksDir` (from cached kit) and `settingsPath` (repo `.claude/settings.local.json`). Include path migration (relative → `$CLAUDE_PROJECT_DIR`).

- [x] T008 Integrate hook sync into `Sync()` step 4 in `src/go/fab-kit/internal/sync.go`: after skill deployment, call the new `HookSync()` function with `{cachedKitDir}/hooks/` and `{repoRoot}/.claude/settings.local.json`. Print the result status message.

- [x] T009 ~~Delete `fab/.kit/sync/5-sync-hooks.sh`~~ — file does not exist in the repo (already removed or never present). The `fab/.kit/sync/` directory contains no `.sh` files. No action needed. <!-- clarified: fab/.kit/sync/5-sync-hooks.sh does not exist in the current codebase; the sync directory has no shell scripts -->

- [x] T010 Wire flag logic into `Sync()`: when `shimOnly` is true, skip step 6 (project sync scripts). When `projectOnly` is true, skip steps 1-5 (go straight to project sync scripts). When both are false, run all steps.

## Phase 3: Integration & Edge Cases

- [x] T011 Update `src/go/fab-kit/internal/sync_test.go`: update existing tests to mock/expect cache-based paths. Add tests for: version guard (system >= project, system < project), flag mutual exclusion, `--shim` skips step 6, `--project` skips steps 1-5.

- [x] T012 [P] Add hook sync tests in `src/go/fab-kit/internal/hooksync_test.go`: test idempotent merge, path migration, duplicate detection, missing hooks dir graceful handling.

- [x] T013 [P] Update prerequisites list in `sync.go`: remove `jq` and `gh` from `requiredTools` (jq was used by old shell-based hook sync, gh only needed by download which has its own error handling). Keep: `git`, `bash`, `yq`, `direnv`.

---

## Execution Order

- T001 (flags) and T002 (scaffold fix) are independent setup tasks (T003 already done — migration exists)
- T004 (version guard) and T005 (cache resolution) must complete before T006 (refactor to use cached path)
- T007 (hook sync replication) is independent of T004-T006, can run in parallel
- T008 (integrate hook sync) depends on both T006 and T007
- T009 already done — script does not exist in repo
- T010 (flag wiring) depends on T001 and T006
- T011 depends on T004-T008, T010 (tests verify the full implementation)
- T012, T013 can run alongside T011
