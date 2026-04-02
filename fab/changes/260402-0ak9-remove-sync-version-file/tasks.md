# Tasks: Remove Sync Version File

**Change**: 260402-0ak9-remove-sync-version-file
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Go Changes

- [x] T001 Remove `writeSyncVersionStamp()` function and its call site from `src/go/fab-kit/internal/sync.go`
- [x] T002 Update `checkSyncStaleness()` in `src/go/fab/internal/preflight/preflight.go` to read `fab_version` from `fab/project/config.yaml` instead of `fab/.kit-sync-version`. Use inline YAML parse. Single warning message: `⚠ Skills may be out of sync — run fab sync to refresh (engine {kitVersion}, project {configVersion})`

## Phase 2: Tests

- [x] T003 Update or remove tests for `writeSyncVersionStamp` in `src/go/fab-kit/internal/sync_test.go` (if any exist) — No tests exist for `writeSyncVersionStamp`; all existing tests pass
- [x] T004 Update tests for `checkSyncStaleness` in `src/go/fab/internal/preflight/preflight_test.go` (if any exist) to verify new comparison logic: versions match → no warning, versions differ → warning, missing files → no warning — No tests exist for `checkSyncStaleness`; all existing tests pass

## Phase 3: Config & Scaffold

- [x] T005 [P] Remove `fab/.kit-sync-version` line from `.gitignore`
- [x] T006 [P] Remove `fab/.kit-sync-version` line from `fab/.kit/scaffold/fragment-.gitignore`
- [x] T007 [P] Create migration file `fab/.kit/migrations/0.45.1-to-0.46.0.md` — instruct removal of orphaned `fab/.kit-sync-version`, handle missing file gracefully

## Phase 4: Documentation

- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — remove `.kit-sync-version` from version tracking inventory, update preserved/replaced file lists, update changelog
- [x] T009 [P] Update `docs/memory/fab-workflow/preflight.md` — update validation check 1b to describe VERSION vs config.yaml comparison, update changelog
- [x] T010 [P] Update `docs/memory/fab-workflow/distribution.md` — remove from preserved files list, update sync staleness detection section

---

## Execution Order

- T001 and T002 are independent (different binaries)
- T003 depends on T001; T004 depends on T002
- Phase 3 and Phase 4 are independent of each other and can run after Phase 2
