# Tasks: Pane Capture Tmux Flag Fix

**Change**: 260406-x33u-pane-capture-tmux-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Extract `capturePaneArgs(paneID string, lines int) []string` as a pure helper and update `capturePaneContent` to use it — in `src/go/fab/cmd/fab/pane_capture.go`
- [x] T002 [P] Fix documented tmux command in Question Detection section of `src/kit/skills/fab-operator.md` (line 244): replace `tmux capture-pane -t <pane> -p -l 20` with `tmux capture-pane -t <pane> -p -S -20`
- [x] T003 [P] Fix documented tmux command in `docs/specs/skills/SPEC-fab-operator.md` (lines 21 and 58): replace `-l 20` with `-S -20`

## Phase 2: Tests

- [x] T004 Add `TestCapturePaneArgs` in `src/go/fab/cmd/fab/pane_capture_test.go` — verify that `capturePaneArgs` returns the correct `-S -N` argument slice for default (50) and custom (20) line counts

---

## Execution Order

- T001, T002, T003 are independent and can run in parallel
- T004 depends on T001 (tests the newly extracted `capturePaneArgs` function)
