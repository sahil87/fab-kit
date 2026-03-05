# Intake: Fix Pipeline Dispatch Visibility

**Change**: 260221-2spf-fix-pipeline-dispatch-timing
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Fix pipeline orchestrator: the fab-switch step runs invisibly via `claude -p` before the interactive pane exists, leaving the user with no feedback for several seconds.

User observed that after `run.sh` resolves a change, there's a long invisible gap before the right-side tmux pane appears. During this gap, `dispatch.sh` is running `claude -p --dangerously-skip-permissions "/fab-switch $CHANGE_ID --no-branch-change"` — a full Claude session that produces no visible output because the interactive pane hasn't been created yet and stdout is redirected to `/dev/null`.

## Why

The `claude -p` call for fab-switch takes 5-15 seconds during which the left pane shows no activity and the right pane doesn't exist yet. Users perceive the pipeline as stuck. The interactive session should be the very first thing created so the user immediately sees activity.

## What Changes

### Move fab-switch into the interactive pane

Currently in `dispatch.sh`, `run_pipeline()` does two sequential steps:
1. Run `claude -p` for fab-switch (invisible, blocking)
2. Create tmux pane with `claude --dangerously-skip-permissions '/fab-ff'`

Reverse this: create the interactive Claude session first (with no initial command — just start `claude --dangerously-skip-permissions`), then send `/fab-switch $CHANGE_ID --no-branch-change` + Enter into the pane via `tmux send-keys`.

The pane appears immediately, giving the user visual feedback that something is happening.

### Poll `fab/current` for switch completion

After sending the fab-switch command, `dispatch.sh` (or `run.sh`) needs to know when the switch is done before sending `/fab-ff`. Poll the worktree's `fab/current` file — when it contains the expected change ID, the switch is complete.

### Send `/fab-ff` after switch completes

Once `fab/current` confirms the switch, send `/fab-ff` + Enter to the interactive pane with a short delay (similar pattern to the ship-timing fix — allow Claude to finish its turn before accepting new input). Use the same split send-keys pattern: text first, then Enter after a small gap.

### Summary of new flow in `dispatch.sh`

```
1. create_worktree()          — unchanged
2. provision_artifacts()      — unchanged
3. validate_prerequisites()   — unchanged
4. tmux split-window (start bare `claude --dangerously-skip-permissions`)
5. sleep + send-keys: "/fab-switch $CHANGE_ID --no-branch-change" + Enter
6. poll fab/current until it contains $CHANGE_ID (with timeout)
7. sleep + send-keys: "/fab-ff" + Enter
8. return pane ID to run.sh   — unchanged
```

Steps 5-7 add blocking time to `dispatch.sh`, but it was already blocking on the `claude -p` call. The difference is the user now sees all of it happening in the right pane.

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update dispatch.sh section to reflect the new flow — interactive pane created first, fab-switch sent via send-keys, fab/current polling, fab-ff sent after. Update the "Hybrid Model" design decision rationale.

## Impact

- `fab/.kit/scripts/pipeline/dispatch.sh` — `run_pipeline()` function rewritten
- No changes to `run.sh` polling, progress rendering, shipping, or SIGINT handling
- No changes to manifest format or any other scripts
- `tmux split-window` command changes: no longer passes `/fab-ff` as the initial command

## Open Questions

None — the approach is well-understood. The polling/timeout values are the only tuning decisions and those are easily adjusted post-implementation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch stays as a Claude skill invocation, not replaced with direct file write | User explicitly requested keeping the abstraction; fab-switch may have side effects beyond writing fab/current | S:95 R:90 A:95 D:95 |
| 2 | Confident | Poll `fab/current` to detect switch completion | fab-switch writes this file as its primary output; polling a file is simpler and more reliable than parsing tmux pane output | S:75 R:90 A:80 D:75 |
| 3 | Confident | ~5 second delay before sending fab-ff after switch completes | Same pattern as the ship-timing fix; Claude needs time to finish its turn before accepting new input | S:70 R:90 A:60 D:75 |
| 4 | Certain | Start interactive pane with bare `claude --dangerously-skip-permissions` (no initial command) | The pane needs to be live before we can send-keys to it; initial command is now sent via send-keys | S:90 R:95 A:90 D:90 |

4 assumptions (2 certain, 2 confident, 0 tentative, 0 unresolved).
