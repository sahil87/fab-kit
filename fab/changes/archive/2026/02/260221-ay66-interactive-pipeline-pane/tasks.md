# Tasks: Interactive Pipeline Pane

**Change**: 260221-ay66-interactive-pipeline-pane
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `get_progress_line()` function to `fab/.kit/scripts/lib/stageman.sh` — iterates `get_progress_map()` output, builds a single-line visual progress string (done stages joined by ` → `, active + ` ⏳`, failed + ` ✗`, pending omitted, all-done appends ` ✓`, all-pending returns empty)
- [x] T002 Wire `progress-line` CLI subcommand in `fab/.kit/scripts/lib/stageman.sh` — add case to CLI dispatch, add to help text, validate single argument (status file path)
- [x] T003 Add `check_pane_alive()` helper function to `fab/.kit/scripts/pipeline/dispatch.sh` — checks tmux pane existence via `tmux list-panes -a -F '#{pane_id}' | grep -qx "$pane_id"`

## Phase 2: Core Implementation

- [x] T004 Rewrite `dispatch.sh` `run_pipeline()` to use interactive Claude session — keep fab-switch as `claude -p`, replace fab-ff with `tmux split-window ... -d -P -F '#{pane_id}' -c "$wt_path" "claude --dangerously-skip-permissions '/fab-ff'"`. Accept `$LAST_PANE_ID` as 5th argument: empty → `split-window -h` (first dispatch), non-empty → `split-window -v -t $LAST_PANE_ID` (stacked). Return pane ID to stdout. Remove polling/waiting from dispatch.
- [x] T005 Remove `ship()` function from `fab/.kit/scripts/pipeline/dispatch.sh` — shipping moves to `run.sh` via `tmux send-keys`
- [x] T006 Update `dispatch.sh` `main()` — remove call to `ship()`, remove post-pipeline manifest stage write (run.sh now handles this), output worktree path and pane ID as two stdout lines. Accept 5th argument `last_pane_id`.
- [x] T007 Rewrite `run.sh` startup — remove `LOG_PANE_ID` variable, remove `tmux split-window -h ... "tail -f"` block. Keep `LOG_FILE` for dispatch output logging.
- [x] T008 Rewrite `run.sh` main loop dispatch call — pass `LAST_PANE_ID` as 5th arg to `dispatch.sh`, capture pane ID from dispatch stdout (last line), update `LAST_PANE_ID`, append pane ID to `PANE_IDS` tracking array
- [x] T009 Implement unified polling loop function `poll_change()` in `fab/.kit/scripts/pipeline/run.sh` — accepts change ID, pane ID, worktree path, status file path. Polls every 5s. State machine: `polling_fab_ff → shipping → polling_ship → done`, with `failed` transitions. Uses `stageman.sh progress-line` for rendering, `printf "\r[pipeline] %s: %s (%dm %02ds)  "` for in-place updates. Checks `hydrate:done` for ship trigger, `*:failed` for failure, pane alive via sourced `check_pane_alive`, `gh pr view` for ship completion.
- [x] T010 Integrate `poll_change()` into `run.sh` main loop — after dispatch returns, call `poll_change` before next iteration. Write terminal stage (`done`/`failed`) to manifest after polling completes.

## Phase 3: Integration & Edge Cases

- [x] T011 Add configurable timeouts to `poll_change()` in `run.sh` — `PIPELINE_FF_TIMEOUT` (default 1800s/30min), `PIPELINE_SHIP_TIMEOUT` (default 300s/5min). Timeout transitions to `failed`. Pane is NOT killed on timeout.
- [x] T012 Update `run.sh` SIGINT handler — kill all tracked panes in `PANE_IDS` array via `tmux kill-pane`, preserve existing summary behavior
- [x] T013 Source `dispatch.sh` helper `check_pane_alive()` in `run.sh` for use in polling loop
- [x] T014 Add test cases for `progress-line` to `src/lib/stageman/test.bats` — all-pending (empty output), first-active (`intake ⏳`), mid-pipeline (`spec → tasks → apply ⏳`), failed (`spec → tasks → apply → review ✗`), all-done (` ✓` suffix), single-done-rest-pending (`intake` with no trailing emoji)

## Execution Order

- T001 blocks T002 (function must exist before CLI wiring)
- T003 blocks T013 (helper must exist before sourcing)
- T004 blocks T006 (run_pipeline rewrite before main() update)
- T005 can run alongside T004 (independent removal)
- T007 blocks T008 (startup cleanup before main loop rewrite)
- T008 blocks T009, T010 (dispatch integration before polling loop)
- T009 blocks T010, T011 (polling function before integration and timeouts)
- T014 can run after T001+T002 (only depends on stageman changes)
