# Intake: Fix Orchestrator False Fail on Review

**Change**: 260221-h1l8-fix-orchestrator-false-fail-on-review
**Created**: 2026-02-21
**Status**: Draft

## Origin

> The pipeline orchestrator (run.sh) incorrectly treats any `:failed` stage in the progress map as a terminal failure for the change. But `review:failed` is a normal intermediate state in the fab-ff rework loop — fab-ff sets `review:failed` then `apply:active` to retry. The orchestrator panics before fab-ff gets a chance to rework. Fix: remove the generic `:failed` grep from poll_change in run.sh (lines 376-383). The orchestrator should only recognize terminal conditions: hydrate:done (success), pane died, or timeout. Also drop the stale [pipeline] prefix from the progress printf on line 352.

One-shot input from direct debugging session. The user observed the orchestrator marking a change as `Failed: 260221-2spf-fix-pipeline-dispatch-timing — review:failed` while the interactive Claude pane was still alive and actively reworking.

## Why

The orchestrator's polling loop in `poll_change()` uses `stageman progress-map` to detect terminal states. It correctly detects `hydrate:done` as success, but then has a catch-all that treats **any** `:failed` stage as a terminal failure:

```bash
elif echo "$progress_map" | grep -q ":failed$"; then
    local failed_stage
    failed_stage=$(echo "$progress_map" | grep ":failed$" | head -1)
    printf "\n"
    log "Failed: $resolved_id — $failed_stage"
    write_stage "$manifest_id" "failed" "$MANIFEST"
    return 0
fi
```

This is wrong because `review:failed` is a **normal intermediate state** in the fab-ff rework loop. When review fails, fab-ff sets `review:failed` then resets `apply:active` to begin a rework cycle (up to 3 auto-cycles, then interactive fallback). The orchestrator should not interpret intermediate progress states — fab-ff owns the rework lifecycle.

Without the fix, every review failure kills the pipeline dispatch for that change, making the auto-rework loop useless in pipeline mode.

## What Changes

### Remove `:failed` catch-all from `poll_change()` in `run.sh`

Delete lines 376-383 in `fab/.kit/scripts/pipeline/run.sh` — the `elif` branch that greps for `:failed$` in the progress map and marks the change as failed.

The orchestrator should rely only on these terminal conditions:
1. **`hydrate:done`** — success (already handled, line 370)
2. **Pane died** — failure (already handled, line 340)
3. **Timeout** — failure (already handled, line 357)

fab-ff manages its own rework lifecycle internally. The orchestrator's job is to wait for fab-ff to finish (detected by `hydrate:done`) or to catch infrastructure failures (pane death, timeout).

### Drop stale `[pipeline]` prefix from progress printf

Line 352 in `run.sh` has a hardcoded `[pipeline]` prefix in the progress rendering printf that doesn't go through the `log()` function. This was missed in the earlier cleanup (commit f9928f8). Change:

```bash
# Before
printf "\r[pipeline] %s: %s (%dm %02ds)  " "$resolved_id" "$progress_line" "$mins" "$secs"

# After
printf "\r%s: %s (%dm %02ds)  " "$resolved_id" "$progress_line" "$mins" "$secs"
```

## Affected Memory

- `fab-workflow/pipeline-orchestrator`: (modify) Update Stage Detection section to reflect that `:failed` is no longer treated as terminal; update Progress Rendering to remove `[pipeline]` prefix reference

## Impact

- `fab/.kit/scripts/pipeline/run.sh` — `poll_change()` function (lines ~340-405)
- Pipeline behavior: changes that hit review failure will now continue through rework instead of being marked failed
- No impact on other terminal conditions (pane death, timeout, hydrate:done)

## Open Questions

None — the fix is straightforward and the behavior is well-documented in fab-ff.md.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only remove the `:failed` catch-all, not any other terminal detection | The other terminal conditions (hydrate:done, pane death, timeout) are correct and necessary | S:95 R:90 A:95 D:95 |
| 2 | Certain | No new terminal state detection needed to replace `:failed` | fab-ff's lifecycle is self-contained — the orchestrator just needs to wait for hydrate:done or catch infrastructure failures | S:90 R:85 A:90 D:90 |
| 3 | Certain | Drop `[pipeline]` from progress printf | Consistent with earlier cleanup in the same session (commits f9928f8, 7ea2696) | S:95 R:95 A:95 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
