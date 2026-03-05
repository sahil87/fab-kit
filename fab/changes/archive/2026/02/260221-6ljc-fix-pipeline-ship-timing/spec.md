# Spec: Fix Pipeline Ship Timing

**Change**: 260221-6ljc-fix-pipeline-ship-timing
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Pipeline Shipping: Delayed Send-Keys

### Requirement: run.sh SHALL wait before sending /changes:ship pr

After detecting `hydrate:done` in the progress map, `run.sh`'s `poll_change()` function SHALL sleep for a configurable delay before sending the ship command. This delay allows Claude to finish its turn output (summary, `Next:` line, bake timer) and return to the input prompt.

#### Scenario: hydrate:done detected, ship sent after delay

- **GIVEN** the `polling_fab_ff` state detects `hydrate:done` in the progress map
- **WHEN** the orchestrator prepares to send the ship command
- **THEN** it logs "fab-ff complete: {id} — waiting for Claude to finish turn..."
- **AND** sleeps for approximately 8 seconds
- **AND** logs "Sending /changes:ship pr"
- **AND** proceeds to send the command

### Requirement: send-keys SHALL be split into text and Enter

The ship command text and the Enter keystroke SHALL be sent as two separate `tmux send-keys` calls with a small gap between them. This prevents the Enter from being buffered alongside the text and lost when Claude's UI is not yet ready.

#### Scenario: Split send-keys for ship command

- **GIVEN** the post-hydrate delay has elapsed
- **WHEN** the orchestrator sends the ship command
- **THEN** it sends `/changes:ship pr` via `tmux send-keys -t "$pane_id"`
- **AND** sleeps for 0.5 seconds
- **AND** sends `Enter` via a separate `tmux send-keys -t "$pane_id"` call
- **AND** transitions to the `shipping` state

#### Scenario: Ship command succeeds and PR is detected

- **GIVEN** the ship command was sent with split send-keys
- **WHEN** the `shipping` state polls `gh pr view`
- **THEN** it detects the PR and marks the change `done`
- **AND** the behavior is identical to current shipping logic (no other changes)

### Requirement: Existing polling and timeout logic SHALL NOT change

The `polling_fab_ff` timeout (`PIPELINE_FF_TIMEOUT`), the `shipping` timeout (`PIPELINE_SHIP_TIMEOUT`), pane-alive checks, progress rendering, and failure detection logic SHALL remain unchanged. Only the moment of ship command delivery is affected.

#### Scenario: Ship timeout still applies after delayed send

- **GIVEN** the ship command was sent after the delay
- **WHEN** `PIPELINE_SHIP_TIMEOUT` elapses without PR detection
- **THEN** the change is marked `failed` as before

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | 8-second delay before sending ship command | Confirmed from intake #1. User reported ~5s needed; 8s provides margin for slower machines. Easily tunable via constant | S:70 R:90 A:60 D:75 |
| 2 | Confident | Split send-keys into text + Enter with 0.5s gap | Confirmed from intake #2. Standard tmux automation pattern; prevents keystroke buffering issues | S:65 R:95 A:70 D:80 |

2 assumptions (0 certain, 2 confident, 0 tentative, 0 unresolved).
