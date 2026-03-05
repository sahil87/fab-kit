# Spec: Fix Orchestrator False Fail on Review

**Change**: 260221-h1l8-fix-orchestrator-false-fail-on-review
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Pipeline Orchestrator: Stage Detection

### Requirement: Remove `:failed` catch-all from poll_change()

The `poll_change()` function in `run.sh` SHALL NOT treat `:failed` stages in the progress map as terminal failure conditions. The `elif` branch (lines 376–383) that greps for `:failed$` and marks the change as `failed` in the manifest MUST be removed.

The orchestrator SHALL rely exclusively on these terminal conditions:
1. `hydrate:done` — success (already handled)
2. Pane death — failure (already handled)
3. Timeout — failure (already handled)

`review:failed` is a normal intermediate state in the fab-ff rework loop. fab-ff sets `review:failed` then resets `apply:active` to retry. The orchestrator MUST NOT interfere with this lifecycle.

#### Scenario: Review failure triggers rework instead of pipeline failure

- **GIVEN** a change is dispatched and fab-ff is running in its interactive pane
- **WHEN** the review stage fails and `review:failed` appears in the progress map
- **THEN** the orchestrator continues polling (does not mark the change as `failed`)
- **AND** fab-ff proceeds with its rework loop (setting `apply:active` and retrying)

#### Scenario: Hydrate done still detected as success

- **GIVEN** a change is dispatched and the polling loop is in `polling_fab_ff` state
- **WHEN** `hydrate:done` appears in the progress map
- **THEN** the orchestrator transitions to `shipping` state and sends `/changes:ship pr`

#### Scenario: Pane death still detected as failure

- **GIVEN** a change is dispatched and the polling loop is active
- **WHEN** the interactive pane dies unexpectedly (check_pane_alive returns false)
- **THEN** the orchestrator marks the change as `failed` in the manifest

#### Scenario: Timeout still detected as failure

- **GIVEN** a change is dispatched and the polling loop is in `polling_fab_ff` state
- **WHEN** the elapsed time exceeds `PIPELINE_FF_TIMEOUT`
- **THEN** the orchestrator marks the change as `failed` in the manifest

### Requirement: Remove stale `[pipeline]` prefix from progress printf

The progress rendering printf on line 352 of `run.sh` SHALL NOT include the hardcoded `[pipeline]` prefix. The format string MUST change from:

```bash
printf "\r[pipeline] %s: %s (%dm %02ds)  " "$resolved_id" "$progress_line" "$mins" "$secs"
```

to:

```bash
printf "\r%s: %s (%dm %02ds)  " "$resolved_id" "$progress_line" "$mins" "$secs"
```

This is consistent with the earlier cleanup in commit f9928f8 that removed `[pipeline]` prefixes from `log()` calls.

#### Scenario: Progress line renders without prefix

- **GIVEN** the polling loop is rendering progress for a dispatched change
- **WHEN** a progress update is printed via `printf`
- **THEN** the output format is `\r<id>: <progress> (<elapsed>)` with no `[pipeline]` prefix

## Deprecated Requirements

### `:failed` stage as terminal condition

**Reason**: `review:failed` is a normal intermediate state in the fab-ff rework loop, not a terminal condition. The catch-all grep for `:failed$` incorrectly treats any failed stage as terminal, killing the pipeline before fab-ff can rework.

**Migration**: N/A — the orchestrator's remaining terminal conditions (hydrate:done, pane death, timeout) are sufficient. fab-ff manages its own rework lifecycle.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only remove the `:failed` catch-all, preserve all other terminal detection | Confirmed from intake #1 — hydrate:done, pane death, and timeout are correct and necessary | S:95 R:90 A:95 D:95 |
| 2 | Certain | No replacement terminal state detection needed | Confirmed from intake #2 — fab-ff's rework lifecycle is self-contained; orchestrator waits for hydrate:done or infrastructure failure | S:90 R:85 A:90 D:90 |
| 3 | Certain | Drop `[pipeline]` from progress printf | Confirmed from intake #3 — consistent with earlier cleanup in commits f9928f8 and 7ea2696 | S:95 R:95 A:95 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative, 0 unresolved).
