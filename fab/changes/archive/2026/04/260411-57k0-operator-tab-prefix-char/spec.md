# Spec: Operator Tab Prefix Character

**Change**: 260411-57k0-operator-tab-prefix-char
**Created**: 2026-04-11
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Operator Skill: Tab Prefix Replacement

### Requirement: Replace ⚡ with » in tab naming

All `tmux new-window -n` invocations in `/fab-operator` that spawn agent tabs SHALL use the format `»<wt>` where `<wt>` is the worktree name. There SHALL be no space between the `»` character and the worktree name.

The `»` character (U+00BB, right-pointing double angle quotation mark) is a single-width Unicode character that renders consistently across terminal emulators without the double-width alignment issues of the ⚡ emoji.

#### Scenario: Existing change spawn

- **GIVEN** the operator spawns an agent for an existing change with worktree name `swift-fox`
- **WHEN** the `tmux new-window` command executes
- **THEN** the tmux tab name SHALL be `»swift-fox`

#### Scenario: Raw text spawn

- **GIVEN** the operator spawns an agent from raw text with worktree name `bold-elk`
- **WHEN** the `tmux new-window` command executes
- **THEN** the tmux tab name SHALL be `»bold-elk`

#### Scenario: Backlog ID spawn

- **GIVEN** the operator spawns an agent from a backlog ID with worktree name `calm-wave`
- **WHEN** the `tmux new-window` command executes
- **THEN** the tmux tab name SHALL be `»calm-wave`

### Requirement: Uniform application across all spawn paths

The tab naming pattern `»<wt>` SHALL be applied uniformly across all four `tmux new-window -n` occurrences in the skill file:

1. Generic spawn sequence (line ~305)
2. "From existing change" path (line ~367)
3. "From raw text" path (line ~376)
4. "From backlog ID or Linear issue" path (line ~384)

#### Scenario: Consistency verification

- **GIVEN** the `src/kit/skills/fab-operator.md` file
- **WHEN** searching for `tmux new-window -n` invocations
- **THEN** all matches SHALL use the `»<wt>` format
- **AND** zero occurrences of `⚡` SHALL remain in the file

## Documentation: Memory Update

### Requirement: Update execution-skills memory

The "Standardized Tmux Tab Naming" decision in `docs/memory/fab-workflow/execution-skills.md` SHALL be updated to reflect the `»` prefix replacing `⚡`. All references to `⚡<wt>` in the decision block and changelog entry SHALL be updated to `»<wt>`.

#### Scenario: Memory reflects new prefix

- **GIVEN** the memory file `docs/memory/fab-workflow/execution-skills.md`
- **WHEN** reading the "Standardized Tmux Tab Naming" decision block (line ~554-558)
- **THEN** all references SHALL use `»<wt>` instead of `⚡<wt>`
- **AND** the decision text SHALL note the prefix was changed from `⚡` to `»` for terminal width compatibility

#### Scenario: Changelog entry updated

- **GIVEN** the changelog table in `docs/memory/fab-workflow/execution-skills.md`
- **WHEN** reading the `260328-iqt8-standardize-tmux-tab-naming` entry (line ~571)
- **THEN** the summary SHALL reference `»<wt>` instead of `⚡<wt>`

## Documentation: Spec Update

### Requirement: Update operator spec version table

The version table entry for v8 in `docs/specs/operator.md` SHALL be updated to reference `»<wt>` instead of `⚡<wt>`.

#### Scenario: Spec version table reflects new prefix

- **GIVEN** the spec file `docs/specs/operator.md`
- **WHEN** reading the v8 version table entry (line ~22)
- **THEN** the entry SHALL reference `»<wt>` tab naming instead of `⚡<wt>`

## Deprecated Requirements

### ⚡ (zap emoji) as tab prefix

**Reason**: Double-width Unicode rendering causes tmux tab bar misalignment and console output formatting issues.
**Migration**: Replace with `»` (right guillemet, U+00BB) — single-width, visually distinct, no shell conflicts.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `»` (right guillemet, U+00BB) as replacement prefix | Confirmed from intake #1 — user explicitly chose this character | S:95 R:90 A:95 D:95 |
| 2 | Certain | No space between prefix and worktree name | Confirmed from intake #2 — carries forward from original convention | S:95 R:95 A:95 D:95 |
| 3 | Certain | Same 4 locations in `fab-operator.md` need updating | Confirmed from intake #3 — grep verified | S:95 R:95 A:95 D:95 |
| 4 | Certain | No Go binary changes needed | Confirmed from intake #4 — grep verified | S:95 R:95 A:95 D:95 |
| 5 | Certain | Memory changelog entry updated in-place, not appended | This change modifies the existing `260328-iqt8` entry rather than adding a new changelog row — the tab naming decision is the same, only the character changed | S:90 R:90 A:85 D:90 |

5 assumptions (5 certain, 0 confident, 0 tentative, 0 unresolved).
