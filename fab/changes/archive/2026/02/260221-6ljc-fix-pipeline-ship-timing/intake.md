# Intake: Fix Pipeline Ship Timing

**Change**: 260221-6ljc-fix-pipeline-ship-timing
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Add delay before sending /changes:ship pr to tmux pane and split send-keys for reliability

User observed that after `hydrate:done` is detected, the `/changes:ship pr` command appears typed in the Claude session but `Enter` never registers — the command sits unsubmitted.

## Why

When `run.sh` detects `hydrate:done` in `.status.yaml`, it immediately sends `/changes:ship pr` via `tmux send-keys`. But Claude writes the status update *before* finishing its turn output (summary, `Next:` line, bake timer). The `send-keys` fires while Claude is still outputting, so the `Enter` keypress gets swallowed.

If Enter doesn't register, the orchestrator's `shipping` state polls `gh pr view` waiting for a PR that will never be created, eventually hitting `PIPELINE_SHIP_TIMEOUT` (5 minutes) and marking the change `failed` — even though fab-ff completed successfully.

## What Changes

### Add delay and split send-keys in `run.sh`

In `fab/.kit/scripts/pipeline/run.sh`, the `poll_change()` function's `polling_fab_ff` case currently does:

```bash
tmux send-keys -t "$pane_id" "/changes:ship pr" Enter
```

Replace with:

```bash
sleep 8
tmux send-keys -t "$pane_id" "/changes:ship pr"
sleep 0.5
tmux send-keys -t "$pane_id" Enter
```

Add log lines so the user sees what's happening during the wait ("waiting for Claude to finish turn..." then "Sending /changes:ship pr").

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update run.sh shipping section to document the delay and split send-keys approach

## Impact

- `fab/.kit/scripts/pipeline/run.sh` — `poll_change()` function, `polling_fab_ff` case only
- No changes to dispatch.sh, manifest format, polling logic, or progress rendering

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | 8-second delay before sending ship command | User reported ~5s needed; adding margin for slower machines. Easily tunable | S:70 R:90 A:60 D:75 |
| 2 | Confident | Split send-keys into text + Enter with 0.5s gap | Prevents Enter being part of the same buffered key sequence; standard tmux automation pattern | S:65 R:95 A:70 D:80 |

2 assumptions (0 certain, 2 confident, 0 tentative, 0 unresolved).
