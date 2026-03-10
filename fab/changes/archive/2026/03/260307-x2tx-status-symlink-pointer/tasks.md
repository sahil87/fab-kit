# Tasks: Status Symlink Pointer

**Change**: 260307-x2tx-status-symlink-pointer
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [ ] T001 Add `id` field to `StatusFile` struct in `src/go/fab/internal/statusfile/statusfile.go` — add `ID string` field with `yaml:"id"` tag, ensure it serializes as the first field
- [ ] T002 [P] Add `id: {ID}` as the first line of `fab/.kit/templates/status.yaml` (before `name: {NAME}`)
- [ ] T003 [P] Update `.gitignore` — replace `fab/current` with `.fab-status.yaml`

## Phase 2: Core Implementation

- [ ] T004 Update `resolveFromCurrent()` in `src/go/fab/internal/resolve/resolve.go` — replace `fab/current` file read with `os.Readlink()` on `<repoRoot>/.fab-status.yaml`, extract folder name from symlink target path
- [ ] T005 Update `Switch()` in `src/go/fab/internal/change/change.go` — replace `fab/current` file write with `os.Remove()` + `os.Symlink()` to create `.fab-status.yaml` symlink
- [ ] T006 Update `SwitchBlank()` in `src/go/fab/internal/change/change.go` — replace `fab/current` file removal with `.fab-status.yaml` symlink removal
- [ ] T007 Update `Rename()` in `src/go/fab/internal/change/change.go` — replace `fab/current` read/write with readlink/re-symlink on `.fab-status.yaml`
- [ ] T008 Update `New()` in `src/go/fab/internal/change/change.go` — replace `{ID}` placeholder in status template alongside existing `{NAME}`, `{CREATED}`, `{CREATED_BY}` replacements
- [ ] T009 Update `readFabCurrent()` in `src/go/fab/cmd/fab/panemap.go` — replace `fab/current` file read with `os.Readlink()` on `.fab-status.yaml`

## Phase 3: Tests

- [ ] T010 Update resolve tests in `src/go/fab/test/parity/resolve_test.go` — replace `fab/current` file creation with `.fab-status.yaml` symlink creation, add broken symlink test
- [ ] T011 [P] Update change switch tests in `src/go/fab/test/parity/changeman_test.go` — assert `.fab-status.yaml` symlink target instead of `fab/current` content
- [ ] T012 [P] Update panemap tests in `src/go/fab/cmd/fab/panemap_test.go` — replace `WriteFile(fab/current, ...)` with `os.Symlink(...)`, add missing symlink test
- [ ] T013 [P] Check and update any parity panemap tests in `src/go/fab/test/parity/panemap_test.go` — symlink instead of file
- [ ] T014 Add ID field round-trip test — verify `id` field persists through `statusfile.Load()` and `statusfile.Save()`
- [ ] T015 Run full test suite via `go test ./...` in `src/go/fab/` to verify all changes pass

## Phase 4: Skills, Docs, and Migration

- [ ] T016 Update `fab/.kit/skills/_preamble.md` — replace all `fab/current` references with `.fab-status.yaml` symlink
- [ ] T017 [P] Update `fab/.kit/skills/_scripts.md` — replace `fab/current` references in command documentation
- [ ] T018 [P] Update `fab/.kit/skills/fab-switch.md` — replace `fab/current` references with `.fab-status.yaml`
- [ ] T019 [P] Update `fab/.kit/skills/fab-archive.md` — replace `fab/current` references with `.fab-status.yaml` symlink removal
- [ ] T020 [P] Update `fab/.kit/skills/fab-discuss.md` — replace `fab/current` references
- [ ] T021 [P] Update `fab/.kit/skills/fab-status.md` — replace `fab/current` references
- [ ] T022 Grep for remaining `fab/current` references in skills, specs, and memory — update all hits
- [ ] T023 Create migration file in `fab/.kit/migrations/` — convert `fab/current` to symlink, backfill `id` field, update `.gitignore`

---

## Execution Order

- T001 blocks T008 (StatusFile struct must have ID field before New() can populate it)
- T001 blocks T014 (struct must have field before round-trip test)
- T004 blocks T010 (resolve implementation before resolve tests)
- T005-T007 block T011 (change implementation before change tests)
- T009 blocks T012-T013 (panemap implementation before panemap tests)
- T015 depends on all test tasks (T010-T014)
- T016-T021 are independent of Go changes, can run in parallel
- T022 depends on T016-T021 (grep after bulk updates to catch stragglers)
- T023 is independent of code changes
