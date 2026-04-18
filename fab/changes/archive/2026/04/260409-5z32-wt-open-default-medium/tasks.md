# Tasks: wt open — Support "default" as App Value

**Change**: 260409-5z32-wt-open-default-medium
**Intake**: [intake.md](intake.md)

## Phase 1: Core Implementation

- [x] T001 [P] Add `ResolveDefaultApp()` function to `src/go/wt/internal/worktree/apps.go`
- [x] T002 Add `"default"` handling to `wt open --app` in `src/go/wt/cmd/open.go` (before `ResolveApp` call, lines 81-101)
- [x] T003 Add `"default"` handling to `wt create --worktree-open` in `src/go/wt/cmd/create.go` (new branch alongside `"prompt"` and `"skip"`, lines 253-287)

## Phase 2: Tests

- [x] T004 [P] Add unit tests for `ResolveDefaultApp()` in `src/go/wt/internal/worktree/apps_test.go`
- [x] T005 [P] Add integration test for `--app default` in `src/go/wt/cmd/open_test.go`
- [x] T006 [P] Add integration test for `--worktree-open default` in `src/go/wt/cmd/create_test.go`

## Execution Order

- T001 first (helper needed by T002, T003)
- T002, T003 after T001 (both depend on the helper)
- T004, T005, T006 after their respective implementation tasks (T001, T002, T003)
