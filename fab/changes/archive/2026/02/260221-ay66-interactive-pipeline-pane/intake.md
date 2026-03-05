# Intake: Interactive Pipeline Pane

**Change**: 260221-ay66-interactive-pipeline-pane
**Created**: 2026-02-21
**Status**: Draft

## Origin

> Replace the pipeline's right pane from a passive `tail -f` log viewer to an interactive Claude session. Keep fab-switch as `claude -p`, but run fab-ff in an interactive Claude session in the right tmux pane. After fab-ff completes (detected by polling .status.yaml), push `/changes:ship pr` to the session via `tmux send-keys`. This gives visibility into Claude's reasoning, enables user intervention, and improves ship quality since the session retains fab-ff context.

Evaluation discussion preceded this change. The user observed that the current pipeline architecture — three separate `claude -p` calls with output piped to a log file — loses context between steps and provides poor visibility (wall of raw text in the log pane). The interactive model preserves conversation context across fab-ff and ship, and gives the user a live view of Claude's reasoning with the ability to intervene.

## Why

1. **Context loss between pipeline steps**: The current architecture runs 3 separate `claude -p` calls per change (fab-switch, fab-ff, ship). Each is a fresh Claude session with no shared memory. The ship step — which generates commit messages and PR descriptions — has zero context about what fab-ff just did. It resorts to a raw natural language prompt ("Commit all changes and create a PR...") rather than using the `/changes:ship` skill, producing generic output.

2. **Without this**: Shipping continues to produce contextually shallow commit messages and PR descriptions. The user has no way to intervene during fab-ff execution (e.g., to help with a review failure). The right pane remains a passive log dump with no interactivity.

3. **Approach — hybrid model**: Keep `fab-switch` as `claude -p` (trivial, no context benefit), then open an interactive Claude session in the right tmux pane for `fab-ff`. The orchestrator polls `.status.yaml` for completion, then pushes `/changes:ship pr` to the same session via `tmux send-keys`. This preserves full conversation context for shipping while keeping the orchestrator in control of sequencing.

## Subsumes

- **260221-td65-stageman-progress-line** — the `progress-line` stageman command and left-pane polling are absorbed into this change. td65 will be deleted.

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

### 2. Right pane: interactive Claude session instead of `tail -f`

In `fab/.kit/scripts/pipeline/run.sh`, replace the current log-tail pane creation:

```bash
# Current — passive log viewer
LOG_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' \
  "tail -f '$LOG_FILE'")
```

With a deferred pane approach — the right pane is created per-dispatch by `dispatch.sh` (not at startup), since each change needs its own interactive session in its own worktree. `run.sh` no longer creates a right pane at startup. It still maintains `LOG_FILE` for its own left-pane logging.

### 3. Dispatch: interactive Claude for fab-ff + ship

In `fab/.kit/scripts/pipeline/dispatch.sh`, the `run_pipeline()` function changes from:

```bash
# Current — two separate claude -p calls
claude -p --dangerously-skip-permissions "/fab-switch ..."
claude -p --dangerously-skip-permissions "/fab-ff"
```

To a hybrid approach:

1. **fab-switch**: stays as `claude -p` (fast, no context needed)
2. **fab-ff**: opens an interactive Claude session in a tmux split pane:
   ```bash
   CLAUDE_PANE_ID=$(tmux split-window -h -d -P -F '#{pane_id}' -c "$wt_path" \
     "claude --dangerously-skip-permissions '/fab-ff'")
   ```
3. **Ship**: on successful completion (detected by `run.sh` polling loop — see section 5), `run.sh` pushes the ship command to the interactive session:
   ```bash
   tmux send-keys -t "$CLAUDE_PANE_ID" "/changes:ship pr" Enter
   ```

`dispatch.sh` returns the pane ID to `run.sh` and exits. It does **not** poll or wait — all waiting is done by `run.sh`'s unified polling loop.

The `ship()` function is replaced entirely — no more raw natural language `claude -p` prompt.

### 4. Pane lifecycle: stacked vertical splits

Each dispatched change gets its own vertical split within the right panel area — stacking them top-to-bottom like the multi-agent code review layout. Previous sessions remain visible and inspectable while new dispatches open below.

**Layout progression**:
```
┌──────────┬──────────┐     ┌──────────┬──────────┐     ┌──────────┬──────────┐
│          │ change-A │     │          │ change-A │     │          │ change-A │
│  orches- │ (claude) │  →  │  orches- ├──────────┤  →  │  orches- ├──────────┤
│  trator  │          │     │  trator  │ change-B │     │  trator  │ change-B │
│          │          │     │          │ (claude) │     │          ├──────────┤
│          │          │     │          │          │     │          │ change-C │
└──────────┴──────────┘     └──────────┴──────────┘     └──────────┴──────────┘
```

- **First dispatch**: `tmux split-window -h` from the orchestrator pane (creates right panel)
- **Subsequent dispatches**: `tmux split-window -v -t $LAST_PANE_ID` (vertical split within the right panel, stacking below the previous session)
- **All pane IDs tracked**: `run.sh` maintains an array of pane IDs for SIGINT cleanup
- **No automatic killing**: sessions stay open for inspection. User can close individual panes manually. SIGINT kills all tracked panes on orchestrator shutdown.
- Pane IDs communicated from `dispatch.sh` back to `run.sh` via stdout (same pattern as worktree path)

### 5. Unified polling loop in `run.sh`

After `dispatch.sh` returns the pane ID, `run.sh` enters a single polling loop per dispatched change. This loop serves two purposes: left-pane progress rendering and completion/failure detection.

Every 5 seconds, while the tmux pane is alive:
1. Calls `stageman progress-line` on the worktree's `.status.yaml`
2. Renders in-place on the left pane via `printf "\r[pipeline] %s: %s (%dm %02ds)  "` (change ID, progress line, elapsed time)
3. Checks for terminal states:
   - `hydrate:done` → push `/changes:ship pr` to the interactive pane via `tmux send-keys`, then continue polling for ship completion
   - Any `*:failed` → log failure, stop polling this change
   - Ship completion (detected via `gh pr view` from worktree) → log success, stop polling
4. If the tmux pane dies unexpectedly → log error, stop polling

**Left pane** (updates in-place):
```
[pipeline] alng: spec → tasks → apply → review ⏳ (1m 32s)
```

**State machine for the loop**:
```
dispatching → polling_fab_ff → shipping → polling_ship → done
                    ↓                          ↓
                  failed                     failed
```

The loop uses `stageman progress-line` for the visual rendering and `stageman display-stage` (or reads the progress map directly) for the terminal-state checks. The configurable timeout defaults to 30 minutes for fab-ff and 5 minutes for ship.

### 6. Helper in dispatch.sh

**`check_pane_alive()`** — verifies tmux pane still exists:
```bash
check_pane_alive() {
  local pane_id="$1"
  tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qx "$pane_id"
}
```

### 7. Test cases for stageman `progress-line`

Add test cases to the existing stageman test suite at `src/lib/stageman/test.bats`:

- All stages pending → empty output
- First stage active → `intake ⏳`
- Mid-pipeline (some done, one active) → `spec → tasks → apply ⏳`
- Failed stage → `spec → tasks → apply → review ✗`
- All stages done → `spec → tasks → apply → review → hydrate ✓`
- Only one stage done, rest pending → `intake` (done but no active — edge case)

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update dispatch architecture from 3x `claude -p` to hybrid model, document interactive pane lifecycle, unified polling loop, progress-line stageman command, `/changes:ship` integration

## Impact

- `fab/.kit/scripts/lib/stageman.sh` — new `progress-line` function + CLI entry (absorbed from td65)
- `fab/.kit/scripts/pipeline/dispatch.sh` — rewrite `run_pipeline()`, remove `ship()`, return pane ID to caller
- `fab/.kit/scripts/pipeline/run.sh` — remove startup log-pane creation, add unified polling loop (progress rendering + completion detection + ship trigger), update SIGINT handler for tracked panes
- `src/lib/stageman/test.bats` — new test cases for `progress-line`
- `docs/memory/fab-workflow/pipeline-orchestrator.md` — update design decisions

## Open Questions

- None — stacking model resolved the pane lifecycle question.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch stays as `claude -p` | User explicitly stated this — trivial operation with no context benefit from interactive mode | S:95 R:90 A:95 D:95 |
| 2 | Certain | fab-ff runs as interactive Claude initial prompt | User explicitly requested interactive session for fab-ff | S:95 R:80 A:90 D:95 |
| 3 | Certain | `/changes:ship pr` pushed via `tmux send-keys` | User explicitly asked about this — ship skill auto-detects context, `pr` selects the right flow | S:90 R:85 A:90 D:90 |
| 4 | Certain | Single polling loop lives in `run.sh`, not `dispatch.sh` | User confirmed — `dispatch.sh` spawns the pane and returns; `run.sh` owns all waiting, progress rendering, and completion detection | S:95 R:90 A:95 D:95 |
| 5 | Confident | Right pane created per-dispatch, not at startup | Each change needs its own worktree CWD — can't reuse a single pane across changes without killing/recreating | S:70 R:85 A:85 D:75 |
| 6 | Tentative | Ship completion detected via `gh pr view` | Not discussed by user — inferred as the best external signal for PR creation. Alternative: poll for pushed commits, or just wait a fixed timeout | S:50 R:80 A:70 D:60 |
<!-- assumed: Ship completion detection via gh pr view — inferred as best external signal, not explicitly discussed -->
| 7 | Certain | Stacked vertical splits — sessions never killed | User explicitly requested this (screenshot of multi-agent review layout as reference). Each dispatch adds a vertical split below the previous one. All sessions stay open for inspection. SIGINT cleans up all. | S:95 R:90 A:95 D:95 |
| 8 | Certain | progress-line uses existing get_progress_map | All stageman accessors build on the progress map — consistent pattern (from td65) | S:90 R:95 A:95 D:95 |
| 9 | Certain | Pending stages omitted from progress-line output | User specified done→active→failed rendering only (from td65) | S:90 R:90 A:90 D:95 |
| 10 | Certain | Poll interval is 5 seconds | User-specified, balances responsiveness vs yq overhead (from td65) | S:85 R:90 A:90 D:90 |

10 assumptions (6 certain, 1 confident, 1 tentative, 0 unresolved).
