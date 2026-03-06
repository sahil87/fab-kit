# Tasks: Add `fab pane-map` Subcommand

**Change**: 260306-bh45-pane-map-subcommand
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Register `pane-map` subcommand in `src/fab-go/cmd/fab/main.go` — add `paneMapCmd()` to `root.AddCommand(...)` and create `src/fab-go/cmd/fab/panemap.go` with the Cobra command skeleton (Use, Short, Args, RunE stub that returns nil)

## Phase 2: Core Implementation

- [x] T002 Implement tmux pane discovery in `src/fab-go/cmd/fab/panemap.go` — execute `tmux list-panes -a -F '#{pane_id} #{pane_current_path}'`, parse each line into a `paneEntry{id, cwd}` struct. Include the `$TMUX` env var guard (print error to stderr, exit 1 if unset)
- [x] T003 Implement worktree resolution in `src/fab-go/cmd/fab/panemap.go` — for each pane, run `git -C <cwd> rev-parse --show-toplevel` to get the worktree root. Skip panes where git fails (non-git dirs). Check for `fab/` directory existence — skip panes without it. Read `fab/current` to get the active change folder name. Read `.status.yaml` via the existing `statusfile.Load()` + `status.DisplayStage()` to get stage info
- [x] T004 Implement runtime state correlation in `src/fab-go/cmd/fab/panemap.go` — for each resolved worktree, read `.fab-runtime.yaml` from the worktree root using the existing `loadRuntimeFile()` helper (move or export from `runtime.go` if needed). Look up `{change_folder}.agent.idle_since`, compute elapsed duration, format as `active`, `idle ({duration})`, or `?` (file missing)
- [x] T005 Implement human-readable duration formatting in `src/fab-go/cmd/fab/panemap.go` — helper function `formatIdleDuration(seconds int64) string` returning `{N}s`, `{N}m`, or `{N}h` (floor division)
- [x] T006 Implement relative worktree path computation in `src/fab-go/cmd/fab/panemap.go` — determine main worktree root (first entry from `git worktree list --porcelain` or the sole root), compute relative paths from its parent directory. Main worktree shows `(main)`
- [x] T007 Implement table output formatting in `src/fab-go/cmd/fab/panemap.go` — dynamically compute column widths from data, print header + data rows with aligned columns. Handle empty result case: print `No fab worktrees found in tmux panes.`

## Phase 3: Integration & Edge Cases

- [x] T008 Handle edge cases in `src/fab-go/cmd/fab/panemap.go` — panes with no active change show `(no change)` and `—` for Stage; panes in subdirectories of worktrees resolve correctly; multiple panes in same worktree each get their own row
- [x] T009 Write parity tests in `src/fab-go/test/parity/panemap_test.go` — test tmux guard (TMUX unset → error), duration formatting (seconds/minutes/hours), relative path computation, table formatting. Use the existing `setupTempRepo(t)` and `runGo(t, ...)` test helpers. Note: tests requiring actual tmux session should be skipped with `t.Skip("requires tmux")` when `$TMUX` is unset
- [x] T010 Update `fab/.kit/skills/_scripts.md` — add `fab pane-map` to the Command Reference table and add a new `## fab pane-map` section documenting the subcommand's usage, output format, and error behavior

---

## Execution Order

- T001 blocks T002–T007 (command skeleton needed first)
- T002 blocks T003 (pane discovery feeds worktree resolution)
- T003 blocks T004, T006 (worktree roots needed for runtime lookup and path computation)
- T005 is independent — can run alongside T003/T004
- T007 depends on T002–T006 (needs all data to format)
- T008 depends on T007 (edge case handling within the formatter)
- T009 depends on T007 (tests the final implementation)
- T010 is independent of implementation — can run alongside T009
