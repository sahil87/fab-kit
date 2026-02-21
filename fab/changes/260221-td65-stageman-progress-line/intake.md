# Intake: Stageman progress-line Command + Pipeline Polling

**Change**: 260221-td65-stageman-progress-line
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Add `progress-line` command to stageman that renders a visual pipeline progress string from .status.yaml. Shows done stages joined by →, active stage with ⏳, complete with ✓, failed with ✗. Then integrate into run.sh as a polling wait loop that updates both left pane (in-place via \r) and right pane (appended to log file) every 5 seconds during dispatch, showing elapsed time.

One-shot description following iterative pipeline DX improvements. The user observed that during `batch-fab-pipeline.sh` execution, both panes lacked progress visibility — the left (orchestrator) pane went silent after "Resolved", and the right (log) pane showed Claude output without structured stage indicators.

## Why

1. **No progress visibility during dispatch**: When `dispatch.sh` runs `claude -p` for a change, the orchestrator pane shows nothing for minutes. The user cannot tell if the pipeline is at tasks, apply, review, or stuck.

2. **Without this**: The only way to know what's happening is to manually read the Claude output wall of text in the log pane, or SSH into the worktree and check `.status.yaml` by hand.

3. **Approach**: A stageman `progress-line` command provides a reusable, composable building block. The pipeline orchestrator polls it every 5 seconds and renders progress in both panes with elapsed time. This keeps the rendering logic in stageman (where all status accessors live) and the polling loop in `run.sh` (where the dispatch lifecycle lives).

## What Changes

### 1. New stageman command: `progress-line`

Add `get_progress_line()` function to `fab/.kit/scripts/lib/stageman.sh` and wire it as the `progress-line` CLI subcommand.

**Input**: `.status.yaml` file path (same as all other stageman accessors)

**Output**: Single line to stdout — a visual pipeline progress string.

**Rendering logic** (reads existing `get_progress_map` output):
- Iterate stages in order from progress map
- `done` stages: append stage name, joined by ` → `
- `active` stage: append stage name + ` ⏳`
- `failed` stage: append stage name + ` ✗`
- `pending` stages: omit entirely
- All stages done (no active, no pending): append ` ✓` to the end

**Example outputs**:
```
intake ⏳                                    # just started
spec → tasks → apply ⏳                      # mid-pipeline
spec → tasks → apply → review ✗             # failed at review
spec → tasks → apply → review → hydrate ✓   # complete
```

**CLI usage**: `stageman.sh progress-line <status-file>`

### 2. Pipeline polling integration in `run.sh`

Replace `wait "$dispatch_pid"` in `fab/.kit/scripts/pipeline/run.sh` with a polling loop that:

1. Resolves the worktree path via `wt_get_worktree_path_by_name` (source `wt-common.sh` from `$KIT_DIR/packages/wt/lib/wt-common.sh`)
2. Locates `.status.yaml` and `stageman.sh` in the worktree
3. Every 5 seconds, while dispatch is running (`kill -0 $dispatch_pid`):
   - Calls `stageman progress-line` on the worktree's `.status.yaml`
   - Renders in-place on left pane via `printf "\r[pipeline] %s: %s (%dm %02ds)  "`
   - Appends a timestamped line to `$LOG_FILE` for the right pane: `[pipeline] ▸ %s: %s (%dm %02ds)\n`
4. After dispatch exits: `wait "$dispatch_pid"` to capture exit code, print newline

**Left pane** (updates in-place):
```
[pipeline] alng: spec → tasks → apply → review ⏳ (1m 32s)
```

**Right pane** (new line every 5s — the ticking timer gives confidence "something's happening"):
```
[pipeline] ▸ alng: spec → tasks → apply → review ⏳ (1m 32s)
[pipeline] ▸ alng: spec → tasks → apply → review ⏳ (1m 37s)
```

### 3. Test cases for stageman `progress-line`

Add test cases to the existing stageman test suite at `src/lib/stageman/test.bats`:

- All stages pending → empty output or first stage with ⏳
- First stage active → `intake ⏳`
- Mid-pipeline (some done, one active) → `spec → tasks → apply ⏳`
- Failed stage → `spec → tasks → apply → review ✗`
- All stages done → `spec → tasks → apply → review → hydrate ✓`
- Only one stage done, rest pending → `intake`  (done but no active — edge case)

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Add progress polling description, progress-line stageman integration, polling interval documentation

## Impact

- `fab/.kit/scripts/lib/stageman.sh` — new function + CLI entry
- `fab/.kit/scripts/pipeline/run.sh` — polling wait loop replaces plain `wait`
- `src/lib/stageman/test.bats` — new test cases
- No changes to `dispatch.sh`

## Open Questions

- None — the design is well-specified from the conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | progress-line uses existing get_progress_map | All stageman accessors build on the progress map — consistent pattern | S:90 R:95 A:95 D:95 |
| 2 | Certain | Poll interval is 5 seconds | User specified 5 seconds in the description, balances responsiveness vs overhead | S:85 R:90 A:90 D:90 |
| 3 | Certain | Pending stages are omitted from output | User specified "done stages joined by →, active stage with ⏳" — pending are not shown | S:90 R:90 A:90 D:95 |
| 4 | Confident | wt_get_worktree_path_by_name resolves worktree path in run.sh | Already used in dispatch.sh for reuse detection — same mechanism | S:80 R:85 A:85 D:90 |
| 5 | Confident | Right pane gets new line (not in-place) every 5s | User wants "ticking timer" confidence — appending lines achieves this while left pane stays clean via \r | S:75 R:85 A:80 D:80 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
