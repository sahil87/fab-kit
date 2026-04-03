# Tasks: wt open — Add tmux session option

**Change**: 260403-o8eg-wt-open-tmux-session-option
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Add `tmux_session` app detection in `BuildAvailableApps` — insert `AppInfo{"tmux session", "tmux_session"}` after the `tmux_window` entry, guarded by `IsTmuxSession()` (`src/go/wt/internal/worktree/apps.go`)
- [x] T002 [P] Add `case "tmux_session":` to `OpenInApp` switch — create detached tmux session via `tmux new-session -d -s {sessionName} -c {path}` with error handling (`src/go/wt/internal/worktree/apps.go`)

## Phase 2: Tests

- [x] T003 Add unit tests for `tmux_session` detection and opening logic (`src/go/wt/internal/worktree/apps_test.go`)

---

## Execution Order

- T001 and T002 are independent (different functions in the same file), can run in parallel
- T003 depends on T001 and T002
