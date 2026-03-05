# Spec: Interactive Pipeline Pane

**Change**: 260221-ay66-interactive-pipeline-pane
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Non-Goals

- Parallel dispatch (V1 remains serial — one change at a time)
- Modifying fab-switch behavior (stays as `claude -p`)
- Changes to manifest format or dependency resolution
- Interactive Claude sessions for fab-switch (trivial, no context benefit)

## Stageman: progress-line Command

### Requirement: Visual Progress Line Output

`stageman.sh` SHALL support a `progress-line` subcommand that produces a single-line visual pipeline progress string.

The `progress-line` command SHALL accept a `.status.yaml` file path as its sole argument.

The output SHALL be constructed by iterating stages in order from `get_progress_map()`:
- `done` stages: append stage name, joined by ` → `
- `active` stage: append stage name + ` ⏳`
- `failed` stage: append stage name + ` ✗`
- `pending` stages: omit entirely
- When all stages are done (no active, no pending): append ` ✓` to the end

The command SHALL output nothing (empty string) when all stages are `pending`.

#### Scenario: Fresh change with first stage active
- **GIVEN** a `.status.yaml` with `intake: active` and all others `pending`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout contains `intake ⏳`

#### Scenario: Mid-pipeline progress
- **GIVEN** a `.status.yaml` with `intake: done`, `spec: done`, `tasks: done`, `apply: active`, rest `pending`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout contains `intake → spec → tasks → apply ⏳`

#### Scenario: Failed at review
- **GIVEN** a `.status.yaml` with intake through apply `done`, `review: failed`, `hydrate: pending`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout contains `intake → spec → tasks → apply → review ✗`

#### Scenario: All stages complete
- **GIVEN** a `.status.yaml` with all stages `done`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout contains `intake → spec → tasks → apply → review → hydrate ✓`

#### Scenario: All stages pending
- **GIVEN** a `.status.yaml` with all stages `pending`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout is empty (no output)

#### Scenario: Single done stage, rest pending
- **GIVEN** a `.status.yaml` with `intake: done` and all others `pending`
- **WHEN** `stageman.sh progress-line <file>` is invoked
- **THEN** stdout contains `intake`
- **AND** no trailing emoji or arrow

## Pipeline: Interactive Right Pane

### Requirement: No Startup Log Pane

`run.sh` SHALL NOT create a tmux split pane at startup for `tail -f` log viewing. The startup log pane creation (`tmux split-window -h ... "tail -f '$LOG_FILE'"`) and the `LOG_PANE_ID` variable SHALL be removed.

`run.sh` SHALL continue to maintain `LOG_FILE` for its own left-pane logging and dispatch output capture.

#### Scenario: Orchestrator startup in tmux
- **GIVEN** `run.sh` is executed inside a tmux session
- **WHEN** the orchestrator starts up
- **THEN** no tmux split-window is created
- **AND** only the orchestrator pane exists

### Requirement: Deferred Pane Creation by Dispatch

`dispatch.sh` SHALL create a tmux split pane for each dispatched change containing an interactive Claude session. The pane SHALL be created per-dispatch, not at startup.

For the **first dispatch**: `dispatch.sh` SHALL use `tmux split-window -h` from the orchestrator pane to create the right panel.

For **subsequent dispatches**: `dispatch.sh` SHALL use `tmux split-window -v -t $LAST_PANE_ID` to create a vertical split within the right panel, stacking below the previous session.

`dispatch.sh` SHALL determine whether this is the first or subsequent dispatch. The first-vs-subsequent distinction MAY be communicated via argument, environment variable, or pane-existence check.

#### Scenario: First change dispatched
- **GIVEN** the orchestrator is running with no right panel
- **WHEN** `dispatch.sh` dispatches the first change
- **THEN** a horizontal split is created (`tmux split-window -h`)
- **AND** the new pane runs an interactive Claude session in the worktree directory

#### Scenario: Second change dispatched
- **GIVEN** one interactive pane already exists from a previous dispatch
- **WHEN** `dispatch.sh` dispatches a second change
- **THEN** a vertical split is created within the right panel (`tmux split-window -v`)
- **AND** the new pane stacks below the existing session

### Requirement: Interactive Claude Session for fab-ff

`dispatch.sh` SHALL launch fab-ff in an interactive Claude session rather than `claude -p`:

```bash
CLAUDE_PANE_ID=$(tmux split-window ... -d -P -F '#{pane_id}' -c "$wt_path" \
  "claude --dangerously-skip-permissions '/fab-ff'")
```

The `claude` invocation SHALL NOT use `-p` (print mode). It SHALL use `--dangerously-skip-permissions` for automated execution. The initial prompt SHALL be `/fab-ff`.

`dispatch.sh` SHALL return the pane ID to `run.sh` via stdout and exit. `dispatch.sh` SHALL NOT poll or wait for completion.

#### Scenario: fab-ff launch
- **GIVEN** a worktree is prepared for a change
- **WHEN** `dispatch.sh` creates the interactive session
- **THEN** `claude --dangerously-skip-permissions '/fab-ff'` runs in the new pane
- **AND** the pane's working directory is the worktree path
- **AND** `dispatch.sh` outputs the pane ID to stdout
- **AND** `dispatch.sh` exits immediately (no waiting)

### Requirement: fab-switch Stays Non-Interactive

`dispatch.sh` SHALL continue to run fab-switch via `claude -p` (print mode). This is unchanged from the current implementation.

#### Scenario: fab-switch execution
- **GIVEN** a change needs to be activated in a worktree
- **WHEN** `dispatch.sh` runs fab-switch
- **THEN** `claude -p --dangerously-skip-permissions "/fab-switch ..."` is used
- **AND** the command completes before the interactive session is launched

## Pipeline: Ship via tmux send-keys

### Requirement: Ship Command via send-keys

After `run.sh` detects successful fab-ff completion (`hydrate:done` in `.status.yaml`), it SHALL push the ship command to the existing interactive Claude session:

```bash
tmux send-keys -t "$CLAUDE_PANE_ID" "/changes:ship pr" Enter
```

`run.sh` SHALL NOT use a separate `claude -p` call for shipping. The `ship()` function in `dispatch.sh` SHALL be removed entirely.

#### Scenario: Ship after successful fab-ff
- **GIVEN** the polling loop detects `hydrate:done` in the worktree's `.status.yaml`
- **WHEN** `run.sh` triggers shipping
- **THEN** `/changes:ship pr` is sent to the interactive pane via `tmux send-keys`
- **AND** the interactive session retains full fab-ff conversation context

### Requirement: Ship Completion Detection

`run.sh` SHALL detect ship completion by polling `gh pr view` from the worktree directory. When a PR exists for the change branch, shipping is considered complete.
<!-- assumed: Ship completion detection via gh pr view — inferred as best external signal for PR creation, not explicitly discussed by user -->

#### Scenario: Ship completion detected
- **GIVEN** `/changes:ship pr` has been sent to the interactive session
- **WHEN** `gh pr view` succeeds for the change branch in the worktree
- **THEN** the change is marked as completed
- **AND** polling stops for this change

#### Scenario: Ship timeout
- **GIVEN** `/changes:ship pr` has been sent to the interactive session
- **WHEN** the ship timeout (5 minutes) elapses without a PR being detected
- **THEN** the change is marked as failed
- **AND** polling stops for this change

## Pipeline: Unified Polling Loop

### Requirement: Per-Change Polling in run.sh

After `dispatch.sh` returns the pane ID, `run.sh` SHALL enter a single polling loop per dispatched change. The loop SHALL poll every 5 seconds while the tmux pane is alive.

Each poll iteration SHALL:
1. Call `stageman.sh progress-line` on the worktree's `.status.yaml`
2. Render in-place on the left pane: `printf "\r[pipeline] %s: %s (%dm %02ds)  "` with change ID, progress line, and elapsed time
3. Check for terminal states

#### Scenario: In-place progress rendering
- **GIVEN** a change is being polled at 1 minute 32 seconds elapsed
- **WHEN** the progress-line returns `spec → tasks → apply ⏳`
- **THEN** the left pane displays `[pipeline] alng: spec → tasks → apply ⏳ (1m 32s)`
- **AND** the line overwrites the previous status (no scrolling)

### Requirement: Polling State Machine

The polling loop SHALL follow this state machine:

```
dispatching → polling_fab_ff → shipping → polling_ship → done
                    ↓                          ↓
                  failed                     failed
```

Terminal state transitions:
- `hydrate:done` in `.status.yaml` → transition to `shipping` (push `/changes:ship pr`)
- Any `*:failed` in `.status.yaml` → transition to `failed` (log failure, stop polling)
- Ship completion (PR detected via `gh pr view`) → transition to `done`
- Pane dies unexpectedly → transition to `failed` (log error, stop polling)

#### Scenario: fab-ff completes successfully
- **GIVEN** the polling loop is in `polling_fab_ff` state
- **WHEN** `.status.yaml` shows `hydrate:done`
- **THEN** the loop transitions to `shipping`
- **AND** `/changes:ship pr` is sent to the interactive pane

#### Scenario: fab-ff fails at review
- **GIVEN** the polling loop is in `polling_fab_ff` state
- **WHEN** `.status.yaml` shows `review:failed`
- **THEN** the loop transitions to `failed`
- **AND** the failure is logged

#### Scenario: Interactive pane dies unexpectedly
- **GIVEN** the polling loop is in `polling_fab_ff` state
- **WHEN** the tmux pane no longer exists
- **THEN** the loop transitions to `failed`
- **AND** the error is logged

### Requirement: Configurable Timeouts

The polling loop SHALL support configurable timeouts:
- fab-ff timeout: 30 minutes (default)
- Ship timeout: 5 minutes (default)

When a timeout elapses, the change SHALL be marked as failed.

#### Scenario: fab-ff timeout
- **GIVEN** fab-ff has been running for 30 minutes
- **WHEN** the timeout elapses
- **THEN** the change is marked as failed
- **AND** the interactive pane is NOT killed (user can inspect)

## Pipeline: Pane Lifecycle

### Requirement: Stacked Vertical Splits

Each dispatched change SHALL get its own vertical split within the right panel area. Previous sessions SHALL remain visible and inspectable while new dispatches open below.

#### Scenario: Three changes dispatched sequentially
- **GIVEN** the orchestrator has dispatched changes A, B, and C in sequence
- **WHEN** all three sessions are running
- **THEN** the layout shows the orchestrator on the left and three stacked panes on the right

### Requirement: Pane Tracking for Cleanup

`run.sh` SHALL maintain an array of all created pane IDs. On SIGINT, `run.sh` SHALL kill all tracked panes via `tmux kill-pane`.

Sessions SHALL NOT be automatically killed on completion. They remain open for manual inspection.

#### Scenario: SIGINT cleanup
- **GIVEN** the orchestrator has created interactive panes for changes A and B
- **WHEN** the user sends SIGINT (Ctrl+C)
- **THEN** all tracked panes are killed
- **AND** the summary is printed (existing behavior preserved)

### Requirement: Pane Alive Check

`dispatch.sh` SHALL provide a `check_pane_alive()` helper function:

```bash
check_pane_alive() {
  local pane_id="$1"
  tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qx "$pane_id"
}
```

This function SHALL be usable by `run.sh` during the polling loop.

#### Scenario: Pane alive check on existing pane
- **GIVEN** an interactive Claude session is running in a tmux pane
- **WHEN** `check_pane_alive "$pane_id"` is called
- **THEN** the function returns 0 (success)

#### Scenario: Pane alive check on dead pane
- **GIVEN** an interactive session's pane has been closed
- **WHEN** `check_pane_alive "$pane_id"` is called
- **THEN** the function returns non-zero (failure)

## Pipeline: Pane ID Communication

### Requirement: dispatch.sh Returns Pane ID

`dispatch.sh` SHALL output the created pane ID to stdout as its final output line. `run.sh` SHALL capture this pane ID for polling and cleanup.

The existing worktree path output SHALL also be preserved. The output format SHALL be two lines: worktree path first, pane ID second.

#### Scenario: dispatch.sh output format
- **GIVEN** dispatch.sh creates a worktree and interactive pane
- **WHEN** dispatch.sh completes
- **THEN** stdout contains the worktree path on one line
- **AND** stdout contains the pane ID on the next line

## Deprecated Requirements

### ship() Function in dispatch.sh

**Reason**: The `ship()` function used a separate `claude -p` call with a raw natural language prompt for shipping. This produced contextually shallow commit messages and PR descriptions because it had no shared context with the fab-ff session.

**Migration**: Replaced by `tmux send-keys` pushing `/changes:ship pr` to the interactive session in `run.sh`.

### Startup Log Pane in run.sh

**Reason**: The startup `tail -f` pane was a passive log viewer that provided poor visibility. Each change now gets its own interactive pane created at dispatch time.

**Migration**: Right pane is created per-dispatch by `dispatch.sh`. `LOG_FILE` is still maintained for orchestrator logging.

## Design Decisions

1. **Hybrid model (fab-switch as `claude -p`, fab-ff as interactive)**
   - *Why*: fab-switch is trivial (no context benefit from interactivity), while fab-ff produces rich conversation context that ship needs. Keeping fab-switch non-interactive avoids unnecessary overhead.
   - *Rejected*: All-interactive (wasteful for fab-switch), all-`claude -p` (loses context for shipping).

2. **Polling in run.sh, not dispatch.sh**
   - *Why*: dispatch.sh spawns the pane and exits. run.sh already owns the main loop and SIGINT handling. Centralizing all waiting in run.sh keeps dispatch.sh simple and stateless.
   - *Rejected*: dispatch.sh waits for completion (blocks serial dispatch, duplicates SIGINT handling).

3. **Stacked vertical splits (sessions never auto-killed)**
   - *Why*: Users need to inspect failed or completed sessions. Killing panes loses diagnostic context. Vertical stacking provides natural visual grouping.
   - *Rejected*: Single reusable right pane (can't inspect previous sessions), horizontal splits (poor use of screen width).

4. **progress-line builds on get_progress_map**
   - *Why*: All stageman accessors use the progress map. Consistent internal pattern, no new data source needed.
   - *Rejected*: Direct YAML parsing in progress-line (duplicates get_progress_map logic).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch stays as `claude -p` | User explicitly stated — trivial, no context benefit. Confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | fab-ff runs as interactive Claude initial prompt | User explicitly requested interactive session. Confirmed from intake #2 | S:95 R:80 A:90 D:95 |
| 3 | Certain | `/changes:ship pr` pushed via `tmux send-keys` | User explicitly discussed — ship skill auto-detects context. Confirmed from intake #3 | S:90 R:85 A:90 D:90 |
| 4 | Certain | Polling loop lives in `run.sh`, not `dispatch.sh` | User confirmed — dispatch spawns, run owns all waiting. Confirmed from intake #4 | S:95 R:90 A:95 D:95 |
| 5 | Confident | Right pane created per-dispatch, not at startup | Each change needs its own worktree CWD. Confirmed from intake #5 | S:70 R:85 A:85 D:75 |
| 6 | Tentative | Ship completion detected via `gh pr view` | Not discussed by user — inferred as best external signal for PR creation. Alternative: poll for pushed commits, fixed timeout. Confirmed from intake #6 | S:50 R:80 A:70 D:60 |
| 7 | Certain | Stacked vertical splits — sessions never auto-killed | User explicitly requested with screenshot reference. Confirmed from intake #7 | S:95 R:90 A:95 D:95 |
| 8 | Certain | progress-line uses existing get_progress_map | Consistent with all stageman accessors. Confirmed from intake #8 | S:90 R:95 A:95 D:95 |
| 9 | Certain | Pending stages omitted from progress-line output | User specified done→active→failed rendering. Confirmed from intake #9 | S:90 R:90 A:90 D:95 |
| 10 | Certain | Poll interval is 5 seconds | User-specified. Confirmed from intake #10 | S:85 R:90 A:90 D:90 |
| 11 | Confident | dispatch.sh determines first-vs-subsequent via argument or pane check | Not explicitly specified — implementation detail. `run.sh` can pass a "last pane ID" argument (empty for first dispatch) | S:60 R:90 A:80 D:65 |
| 12 | Confident | check_pane_alive helper lives in dispatch.sh, sourced by run.sh | Intake specifies dispatch.sh location — run.sh needs it for polling. Shared helper pattern consistent with wt-common.sh sourcing | S:65 R:90 A:85 D:70 |

12 assumptions (8 certain, 3 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
