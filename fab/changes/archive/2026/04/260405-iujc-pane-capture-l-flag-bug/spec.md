# Spec: Pane Capture -l Flag Bug

**Change**: 260405-iujc-pane-capture-l-flag-bug
**Created**: 2026-04-05
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changes to the Go implementation — `pane_capture.go` already registers `-l` correctly via `cmd.Flags().IntP("lines", "l", 50, ...)`
- New tests — `TestCaptureLineFlagShorthand` already verifies the shorthand registration
- Changes to any other skill or documentation file beyond `_cli-fab.md`

## Investigation Summary

Running `fab pane capture --help` shows `-l, --lines int   Number of lines to capture (default 50)` — the `-l` short flag IS registered in the deployed binary. Running `fab pane capture -l` (without a value) produces "flag needs an argument: 'l' in -l", confirming cobra recognizes the flag. The reported "unknown flag" error could not be reproduced against the current v1.3.1 binary.

**Root cause**: The `_cli-fab.md` skill documents only the short form `-l` (not the full `--lines` long form) and includes the confusing parenthetical "passed to `tmux capture-pane -l`", which conflates the user-facing flag with an internal tmux argument. This is a documentation gap, not a binary bug.

## CLI Reference Documentation

### Requirement: pane capture flag table completeness

The `fab pane capture` flag table in `src/kit/skills/_cli-fab.md` MUST document both the short form `-l` and the long form `--lines` in a single row, consistent with the cobra-generated help output (`-l, --lines int`).

#### Scenario: Agent reads _cli-fab and calls pane capture with long form
- **GIVEN** an agent reads the `_cli-fab` skill to discover pane capture flags
- **WHEN** the agent uses `--lines N` in a generated command
- **THEN** the command succeeds (long form is valid) and the agent's invocation matches documented behavior

#### Scenario: Agent reads _cli-fab and calls pane capture with short form
- **GIVEN** an agent reads the `_cli-fab` skill to discover pane capture flags
- **WHEN** the agent uses `-l N` in a generated command
- **THEN** the command succeeds and the agent's invocation matches documented behavior

### Requirement: pane capture flag description accuracy

The description for the `-l`/`--lines` flag MUST NOT reference internal implementation details (specifically, the parenthetical "passed to `tmux capture-pane -l`" SHALL be removed). The description SHOULD state only the user-visible effect: "Number of lines to capture".

#### Scenario: Flag description is unambiguous
- **GIVEN** the `_cli-fab` flag table entry for `-l`/`--lines`
- **WHEN** an agent reads the description
- **THEN** the description clearly describes the user-facing behavior without referencing the internal `tmux capture-pane` invocation

### Requirement: deployed copy synchronization

After updating `src/kit/skills/_cli-fab.md`, the deployed copy at `.claude/skills/_cli-fab/SKILL.md` MUST be synchronized via `fab sync` so agents reading the deployed copy receive the updated documentation.

#### Scenario: fab sync propagates change
- **GIVEN** `src/kit/skills/_cli-fab.md` has been updated with the corrected flag table
- **WHEN** `fab sync` is run
- **THEN** `.claude/skills/_cli-fab/SKILL.md` reflects the updated content

## Design Decisions

1. **Show both forms in one row**: Use `` `-l`, `--lines` `` in the Flag column rather than two separate rows (one for short, one for long). This matches the convention for cobra help output and keeps the table compact.
   - *Why*: The existing table uses one row per logical flag. Short and long forms are the same flag; splitting them would add visual noise with duplicated Type/Default/Description values.
   - *Rejected*: Two separate rows — redundant, inconsistent with cobra help output format.

2. **Remove implementation detail from description**: Drop "passed to `tmux capture-pane -l`" entirely rather than rewording it.
   - *Why*: It's an internal detail (the tmux `-l` argument is a coincidence of naming). Users calling `fab pane capture` don't need to know how it maps internally. Including it creates ambiguity about whether `-l` is "just" a proxy flag.
   - *Rejected*: Reword as "internally passed as `tmux capture-pane -l`" — still exposes implementation detail.

## Clarifications

### Session 2026-04-05 (auto)

| # | Action | Detail |
|---|--------|--------|
| 4 | Upgraded Confident → Certain | Design Decisions §2 documents rationale and explicitly rejects the reword alternative — self-resolving from context |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix is documentation-only; no Go changes needed | `pane_capture.go` has `IntP("lines", "l", ...)` since initial creation; deployed binary help confirms; `TestCaptureLineFlagShorthand` passes | S:95 R:95 A:95 D:95 |
| 2 | Certain | No new tests needed | Existing `TestCaptureLineFlagShorthand` already asserts `flag.Shorthand == "l"` and `DefValue == "50"` | S:90 R:90 A:90 D:90 |
| 3 | Certain | `fab sync` needed to propagate canonical source to deployed copy | Constitution states `.claude/skills/` is generated by `fab sync`; editing only canonical source leaves deployed copy stale | S:90 R:90 A:90 D:90 |
| 4 | Certain | "passed to tmux capture-pane -l" description should be removed (not just reworded) | It exposes an implementation coincidence (both fab and tmux happen to use `-l` for lines); removing it is simpler and avoids future confusion if internals change. Design Decisions §2 documents rationale and explicitly rejects the reword alternative — no user input needed. <!-- clarified: upgraded from Confident; Design Decisions §2 self-resolves this assumption --> | S:95 R:85 A:80 D:75 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
