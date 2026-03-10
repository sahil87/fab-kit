# Tasks: Merge Claude Code Hooks Into Go Binary

**Change**: 260310-bvc6-merge-hooks-into-go
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Extract runtime file helpers (`loadRuntimeFile`, `saveRuntimeFile`, `runtimeFilePath`) from `src/fab-go/cmd/fab/runtime.go` into new `src/fab-go/internal/runtime/runtime.go` package — export as `LoadFile`, `SaveFile`, `FilePath`
- [x] T002 Update `src/fab-go/cmd/fab/runtime.go` to import and use `internal/runtime` package instead of local helpers — verify `fab runtime set-idle`, `clear-idle`, `is-idle` still pass existing tests

## Phase 2: Core Implementation

- [x] T003 Create `src/fab-go/cmd/fab/hook.go` with `hookCmd()` parent command and `hookSessionStartCmd()`, `hookStopCmd()`, `hookUserPromptCmd()` subcommands — each resolves active change via `resolve.FabRoot()`/`resolve.ToFolder("")` and calls `internal/runtime` functions, swallowing all errors (returns nil always)
- [x] T004 Create `src/fab-go/internal/hooklib/artifact.go` with artifact-write logic: `ParsePayload(stdin) (filePath, error)`, `MatchArtifactPath(filePath) (changeFolder, artifact, bool)`, `InferChangeType(content) string`, `CountUncheckedTasks(content) int`, `CountChecklistItems(content) int`
- [x] T005 Create `hookArtifactWriteCmd()` in `src/fab-go/cmd/fab/hook.go` — reads stdin, calls `hooklib` functions, performs per-artifact bookkeeping using `statusfile` and `score` packages, outputs JSON to stdout, swallows all errors
- [x] T006 Create `src/fab-go/internal/hooklib/sync.go` with hook sync logic: discover `on-*.sh` in hooks dir, mapping table, read/merge/write `.claude/settings.local.json`, duplicate detection by matcher+command pair
- [x] T007 Create `hookSyncCmd()` in `src/fab-go/cmd/fab/hook.go` — calls `hooklib.Sync()`, reports created/updated/OK status to stdout
- [x] T008 Register `hookCmd()` in `src/fab-go/cmd/fab/main.go` root command

## Phase 3: Integration & Edge Cases

- [x] T009 Rewrite `fab/.kit/hooks/on-session-start.sh` as thin wrapper: `exec "$(dirname "$0")/../bin/fab" hook session-start 2>/dev/null; exit 0`
- [x] T010 [P] Rewrite `fab/.kit/hooks/on-stop.sh` as thin wrapper: `exec "$(dirname "$0")/../bin/fab" hook stop 2>/dev/null; exit 0`
- [x] T011 [P] Create `fab/.kit/hooks/on-user-prompt.sh` as thin wrapper: `exec "$(dirname "$0")/../bin/fab" hook user-prompt 2>/dev/null; exit 0`
- [x] T012 [P] Rewrite `fab/.kit/hooks/on-artifact-write.sh` as thin wrapper: `exec "$(dirname "$0")/../bin/fab" hook artifact-write 2>/dev/null; exit 0`
- [x] T013 Rewrite `fab/.kit/sync/5-sync-hooks.sh` to delegate to `fab hook sync` with binary-missing fallback warning
- [x] T014 Remove jq prerequisite check from `fab/.kit/scripts/fab-doctor.sh` (lines 92-100 approx)

## Phase 4: Tests & Documentation

- [x] T015 Create `src/fab-go/internal/runtime/runtime_test.go` — unit tests for extracted runtime helpers (load, save, file path)
- [x] T016 Create `src/fab-go/internal/hooklib/artifact_test.go` — unit tests for JSON parsing, path matching, keyword matching, task counting, checklist counting
- [x] T017 [P] Create `src/fab-go/internal/hooklib/sync_test.go` — unit tests for hook sync: fresh settings, deduplication, merge, missing scripts, preserve non-hook settings
- [x] T018 Update `src/hooks/test-on-session-start.bats` — test thin wrapper delegates to binary, binary-missing graceful fallback
- [x] T019 [P] Update `src/hooks/test-on-stop.bats` — test thin wrapper delegates to binary, binary-missing graceful fallback
- [x] T020 Update `fab/.kit/skills/_scripts.md` — add `fab hook` command group with `session-start`, `stop`, `user-prompt`, `artifact-write`, `sync` subcommands to Command Reference table and add detailed section

---

## Execution Order

- T001 blocks T002 (runtime extraction before migration)
- T002 blocks T003, T005 (runtime package used by hook commands)
- T004 blocks T005 (hooklib artifact logic used by artifact-write command)
- T006 blocks T007 (hooklib sync logic used by sync command)
- T003, T005, T007 block T008 (all hook subcommands created before registration)
- T008 blocks T009-T013, T018-T019 (binary must be buildable before shell wrappers and bats tests)
- T009-T012 are parallel (independent shell wrappers)
- T015, T016, T017 are parallel with T009-T014 (unit tests independent of shell wrappers)
- T018, T019 are parallel (independent bats test updates)
