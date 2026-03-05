# Spec: Fix Pipeline Dispatch Visibility

**Change**: 260221-2spf-fix-pipeline-dispatch-timing
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Pipeline Dispatch: Visible fab-switch

### Requirement: Interactive pane SHALL be created before fab-switch

`dispatch.sh`'s `run_pipeline()` function SHALL create the interactive tmux pane as its first action, before executing fab-switch. The pane SHALL start a bare `claude --dangerously-skip-permissions` session (no initial skill command). This ensures the user immediately sees a Claude session appear in the right panel after dispatch begins.

#### Scenario: First dispatch creates horizontal split with bare session

- **GIVEN** `LAST_PANE_ID` is empty (first dispatch)
- **WHEN** `run_pipeline()` is called
- **THEN** a `tmux split-window -h -d -P -F '#{pane_id}' -c "$wt_path" "claude --dangerously-skip-permissions"` is executed
- **AND** the pane ID is captured for subsequent use

#### Scenario: Subsequent dispatch creates vertical split with bare session

- **GIVEN** `LAST_PANE_ID` contains a valid pane ID (not first dispatch)
- **WHEN** `run_pipeline()` is called
- **THEN** a `tmux split-window -v -t "$LAST_PANE_ID" -d -P -F '#{pane_id}' -c "$wt_path" "claude --dangerously-skip-permissions"` is executed

### Requirement: fab-switch SHALL be sent to the interactive pane via send-keys

After the interactive pane is created and a short startup delay, `dispatch.sh` SHALL send `/fab-switch $CHANGE_ID --no-branch-change` to the pane via `tmux send-keys`. The command text and Enter keystroke SHALL be sent separately with a small gap.

#### Scenario: fab-switch command delivered to interactive pane

- **GIVEN** the interactive Claude session pane has been created
- **WHEN** `dispatch.sh` is ready to activate the change
- **THEN** it sleeps for a configurable startup delay (to let Claude initialize)
- **AND** sends the text `/fab-switch $CHANGE_ID --no-branch-change` via `tmux send-keys -t "$pane_id"`
- **AND** sleeps briefly (0.5s)
- **AND** sends `Enter` via `tmux send-keys -t "$pane_id"`

### Requirement: dispatch.sh SHALL poll fab/current to confirm switch completion

After sending the fab-switch command, `dispatch.sh` SHALL poll `$wt_path/fab/current` until its content matches `$CHANGE_ID`. Polling SHALL use a configurable interval and timeout. Timeout SHALL mark the change as `failed` in the manifest.

#### Scenario: fab/current matches expected change ID

- **GIVEN** fab-switch has been sent to the interactive pane
- **WHEN** `$wt_path/fab/current` contains exactly `$CHANGE_ID`
- **THEN** polling stops and dispatch proceeds to send fab-ff

#### Scenario: fab/current polling times out

- **GIVEN** fab-switch has been sent to the interactive pane
- **WHEN** the polling timeout is reached without `fab/current` matching
- **THEN** the change is marked `failed` in the manifest
- **AND** `dispatch.sh` returns without sending fab-ff
- **AND** the pane ID is still returned so `run.sh` can track it

#### Scenario: Pane dies during fab-switch polling

- **GIVEN** fab-switch has been sent to the interactive pane
- **WHEN** the tmux pane is no longer alive during polling
- **THEN** the change is marked `failed` in the manifest
- **AND** `dispatch.sh` returns without sending fab-ff

### Requirement: fab-ff SHALL be sent after switch confirmation with a delay

Once `fab/current` confirms the switch, `dispatch.sh` SHALL wait for Claude to finish its fab-switch turn output, then send `/fab-ff` to the pane. The same split send-keys pattern SHALL be used (text, gap, Enter).

#### Scenario: fab-ff sent after successful switch

- **GIVEN** `fab/current` polling confirmed the switch completed
- **WHEN** the post-switch delay elapses
- **THEN** sends `/fab-ff` via `tmux send-keys -t "$pane_id"`
- **AND** sleeps briefly (0.5s)
- **AND** sends `Enter` via `tmux send-keys -t "$pane_id"`
- **AND** the pane ID is returned to `run.sh` for polling

### Requirement: dispatch.sh output contract SHALL remain unchanged

`dispatch.sh` SHALL continue to output exactly two lines to stdout: worktree path (line 1) and pane ID (line 2). `run.sh` captures these. All log messages SHALL go to stderr. This contract MUST NOT change.

#### Scenario: run.sh receives expected output

- **GIVEN** dispatch.sh completes successfully
- **WHEN** run.sh parses dispatch output
- **THEN** line 1 is the worktree path
- **AND** line 2 is the tmux pane ID
- **AND** run.sh proceeds to poll_change() as before

## Non-Goals

- Changing `run.sh`'s polling loop, progress rendering, or shipping logic — those are separate concerns (ship-timing is a separate change: 260221-6ljc)
- Modifying the manifest format or validation logic
- Changing how `run.sh` handles SIGINT or the summary output

## Design Decisions

1. **Bare session + send-keys instead of command argument**
   - *Why*: `tmux split-window ... "claude --dangerously-skip-permissions '/fab-ff'"` starts Claude with a pre-loaded command — but we need fab-switch to run first. Starting bare and sending commands via send-keys gives us sequencing control while keeping both commands visible in the interactive pane.
   - *Rejected*: Passing fab-switch as the initial command and fab-ff via send-keys after — fab-switch output would scroll past quickly and the UX benefit is the same either way. Starting bare is simpler and more consistent (both commands go through send-keys).

2. **Poll fab/current instead of parsing pane output**
   - *Why*: fab-switch's primary side effect is writing `fab/current`. Polling a file is deterministic and doesn't depend on tmux pane output parsing, which is fragile (ANSI escapes, Claude's streaming output, etc.).
   - *Rejected*: `tmux capture-pane` + grep for "fab/current →" — fragile, timing-dependent, and the output format could change.

3. **Dispatch.sh owns the switch polling, not run.sh**
   - *Why*: dispatch.sh already blocks during this phase (previously on `claude -p`). Moving polling to run.sh would complicate the state machine (new state between "dispatched" and "polling_fab_ff"). dispatch.sh blocking on switch completion before returning the pane ID keeps the dispatch contract simple.
   - *Rejected*: Adding a `switching` state to run.sh's poll_change() — unnecessary complexity.

## Deprecated Requirements

### fab-switch via claude -p (print mode)
**Reason**: Replaced by sending fab-switch to the interactive pane via send-keys. The `claude -p` approach provided no user feedback and was pure overhead.
**Migration**: Interactive pane with send-keys provides the same functionality with full visibility.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch runs as Claude skill via send-keys, not replaced with direct file write | Confirmed from intake #1. User explicitly required keeping the abstraction | S:95 R:90 A:95 D:95 |
| 2 | Certain | Interactive pane created with bare `claude --dangerously-skip-permissions` | Confirmed from intake #4. Pane must exist before send-keys can target it | S:90 R:95 A:90 D:90 |
| 3 | Confident | Poll `fab/current` for switch completion detection | Confirmed from intake #2. Deterministic file check; simpler than pane output parsing | S:75 R:90 A:80 D:75 |
| 4 | Confident | ~5 second delay before sending fab-ff after switch completes | Confirmed from intake #3. Claude needs turn-completion time before accepting new input | S:70 R:90 A:60 D:75 |
| 5 | Confident | Claude startup delay before sending fab-switch (~3-5s) | New: bare Claude session needs time to initialize before accepting input. Exact value tunable | S:60 R:90 A:55 D:70 |
| 6 | Certain | dispatch.sh stdout contract unchanged (wt_path + pane_id) | Spec requirement. run.sh parses these two lines — changing would break the orchestrator | S:95 R:80 A:95 D:95 |
| 7 | Confident | 60-second timeout for fab/current polling | New: fab-switch via Claude typically takes 5-15s; 60s provides ample margin without masking real failures | S:50 R:90 A:65 D:70 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
