# Tasks: Fix Pipeline Dispatch Visibility

**Change**: 260221-2spf-fix-pipeline-dispatch-timing
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Rewrite `run_pipeline()` in `fab/.kit/scripts/pipeline/dispatch.sh` — remove the `claude -p` fab-switch call and change tmux `split-window` to start a bare `claude --dangerously-skip-permissions` session (no initial `/fab-ff` command). First dispatch uses `-h`, subsequent uses `-v -t "$LAST_PANE_ID"`. Capture pane ID.

- [x] T002 Add fab-switch delivery and `fab/current` polling to `run_pipeline()` in `fab/.kit/scripts/pipeline/dispatch.sh` — after pane creation, sleep for startup delay (~3s), send `/fab-switch $CHANGE_ID --no-branch-change` via `tmux send-keys` (text, 0.5s gap, Enter). Then poll `$wt_path/fab/current` until it matches `$CHANGE_ID`, with 2s interval and 60s timeout. Check pane alive each iteration. On timeout or pane death, mark change `failed` in manifest and return pane ID.

- [x] T003 Add fab-ff delivery to `run_pipeline()` in `fab/.kit/scripts/pipeline/dispatch.sh` — after successful `fab/current` polling, sleep ~5s (Claude turn-completion delay), then send `/fab-ff` via `tmux send-keys` (text, 0.5s gap, Enter). Return pane ID on stdout. Verify the two-line stdout contract (wt_path + pane_id) is preserved.

---

## Execution Order

- T001 → T002 → T003 (sequential: each extends `run_pipeline()`)
