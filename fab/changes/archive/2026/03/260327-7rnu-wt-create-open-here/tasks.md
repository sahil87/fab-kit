# Tasks: wt create "Open Here" Option

**Change**: 260327-7rnu-wt-create-open-here
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add "Open here" `AppInfo` entry as first item in `BuildAvailableApps()` in `src/go/wt/internal/worktree/apps.go`
- [x] T002 [P] Add `"open_here"` case to `OpenInApp()` switch in `src/go/wt/internal/worktree/apps.go` — print `cd %q` to stdout, return nil

## Phase 2: stdout Suppression

- [x] T003 Modify `src/go/wt/cmd/create.go` to track whether `open_here` was selected and skip the final `fmt.Println(wtPath)` when it was. This applies to both the menu path (line 252-258) and the direct `--worktree-open` path (line 260-271)

## Phase 3: Tests

- [x] T004 [P] Add test for `BuildAvailableApps()` confirming "Open here" is the first entry — in `src/go/wt/internal/worktree/` (new file `apps_test.go`)
- [x] T005 [P] Add test for `OpenInApp("open_here", ...)` confirming it writes `cd <path>` to stdout — in `src/go/wt/internal/worktree/apps_test.go`
- [x] T006 [P] Add integration test for `wt create --worktree-open open_here` confirming stdout contains only the `cd` line (no trailing path) — in `src/go/wt/cmd/create_test.go`

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 depends on T002 (needs the open_here case to exist)
- T004, T005, T006 can run in parallel after T001-T003
