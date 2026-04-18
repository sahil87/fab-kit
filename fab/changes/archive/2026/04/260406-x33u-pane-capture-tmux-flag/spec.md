# Spec: Pane Capture Tmux Flag Fix

**Change**: 260406-x33u-pane-capture-tmux-flag
**Created**: 2026-04-06
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing the `fab pane capture` CLI flag (`-l`/`--lines`) — the cobra flag shorthand stays as-is; only the internal tmux argument changes.
- Modifying the operator's capture integration to route through `fab pane capture` — the skill uses raw tmux for simplicity; fixing the documented command is sufficient.

## Pane Capture: tmux Argument Correctness

### Requirement: Correct tmux capture-pane line-limiting flag

`capturePaneContent` in `src/go/fab/cmd/fab/pane_capture.go` SHALL invoke `tmux capture-pane` with `-S -N` (where N is the line count) rather than `-l N`. Specifically, the argument list MUST be:

```
["capture-pane", "-t", paneID, "-p", "-S", fmt.Sprintf("-%d", lines)]
```

The `-S` flag sets the start line relative to the bottom of the pane's scrollback buffer. A value of `-N` captures the last N lines. The `-l` flag does not exist on `capture-pane` and causes tmux to return an error.

#### Scenario: Default line count

- **GIVEN** `fab pane capture %5` is invoked (no `--lines` flag)
- **WHEN** `capturePaneContent("%5", 50)` is called internally
- **THEN** the tmux command executed is `tmux capture-pane -t %5 -p -S -50`
- **AND** tmux returns the last 50 lines of pane `%5` without error

#### Scenario: Custom line count

- **GIVEN** `fab pane capture %5 -l 20` is invoked
- **WHEN** `capturePaneContent("%5", 20)` is called internally
- **THEN** the tmux command executed is `tmux capture-pane -t %5 -p -S -20`
- **AND** tmux returns the last 20 lines without error

#### Scenario: Invalid old flag rejected

- **GIVEN** any version of tmux that does not support `-l` on `capture-pane`
- **WHEN** the old `tmux capture-pane -t %5 -p -l 50` is invoked
- **THEN** tmux exits with an error, causing `fab pane capture` to fail

### Requirement: Testable argument construction

The tmux argument construction SHALL be extracted into a pure function `capturePaneArgs(paneID string, lines int) []string` so that the correct flag ordering can be verified without executing tmux.

#### Scenario: Args function returns correct slice

- **GIVEN** `capturePaneArgs("%5", 50)` is called
- **WHEN** the function executes
- **THEN** it returns `["capture-pane", "-t", "%5", "-p", "-S", "-50"]`

#### Scenario: Negative offset for any positive line count

- **GIVEN** `capturePaneArgs("%3", 100)` is called
- **WHEN** the function executes
- **THEN** it returns `["capture-pane", "-t", "%3", "-p", "-S", "-100"]`

## Pane Capture: Operator Skill Documentation

### Requirement: Operator skill documents correct tmux command

The Question Detection section of `src/kit/skills/fab-operator.md` SHALL document `tmux capture-pane -t <pane> -p -S -20` (not `-l 20`) as the capture command used for idle agent question detection.

#### Scenario: Skill documents valid flag

- **GIVEN** a developer reads the Question Detection section of `fab-operator.md`
- **WHEN** they copy the documented tmux command for manual testing
- **THEN** the command is `tmux capture-pane -t <pane> -p -S -20` and executes successfully

### Requirement: Operator skill spec updated to match skill

<!-- clarified: constitution requires SPEC-*.md updated when skill file changes; SPEC-fab-operator.md line 58 documents the same incorrect -l 20 flag and must be corrected -->
`docs/specs/skills/SPEC-fab-operator.md` SHALL be updated to replace `-l 20` with `-S -20` in both the Question Detection section (line 58) and the Auto-Nudge summary entry (line 21).

#### Scenario: Spec reflects corrected command

- **GIVEN** the operator skill spec (`SPEC-fab-operator.md`) has been updated
- **WHEN** a developer reads the Auto-Nudge → Question Detection section
- **THEN** the capture command shown is `tmux capture-pane -t <pane> -p -S -20`
- **AND** the summary entry in the feature list also shows `-S -20`

## Design Decisions

1. **Extract `capturePaneArgs` as a pure function**: The argument list is moved to a testable helper rather than inlined in `capturePaneContent`.
   - *Why*: Follows the constitution rule that CLI Go changes MUST include corresponding test updates. A pure function can be unit-tested without spawning tmux.
   - *Rejected*: Testing via `exec.Command` mock injection — adds complexity (dependency injection pattern, interface wrapping) that's not warranted for a one-line fix. A pure args helper is minimal overhead with full testability.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Replace `-l N` with `-S -N` in `capturePaneContent` | Confirmed from intake #1. tmux `capture-pane` has no `-l` flag; `-S -N` is the standard way to capture the last N lines | S:90 R:90 A:90 D:90 |
| 2 | Certain | Fix skill doc in `fab-operator.md` line 244 | Confirmed from intake #2. Skill documents raw tmux command used in operator question detection; must be correct | S:85 R:90 A:85 D:85 |
| 3 | Certain | Extract `capturePaneArgs` as a pure function for testability | Constitution requires Go CLI changes to include test updates. A pure function is the minimal testable abstraction | S:85 R:85 A:85 D:85 |
| 4 | Confident | No change to cobra `-l`/`--lines` flag definition | Confirmed from intake #3. The cobra CLI flag is independent of the tmux argument; both happen to use `-l` but for different reasons | S:80 R:85 A:80 D:80 |
| 5 | Confident | `_cli-fab.md` documentation needs no update | Confirmed from intake #4. `_cli-fab.md` documents the `fab pane capture` CLI flags, not the internal tmux invocation | S:75 R:80 A:80 D:80 |
| 6 | Certain | `docs/specs/skills/SPEC-fab-operator.md` must also be updated | Clarified — constitution requires SPEC-*.md updated whenever a skill file changes; SPEC-fab-operator.md contains the same incorrect `-l 20` flag at lines 21 and 58 | S:90 R:90 A:95 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
