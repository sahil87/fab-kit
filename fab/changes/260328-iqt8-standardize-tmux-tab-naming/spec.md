# Spec: Standardize Tmux Tab Naming for Spawned Agents

**Change**: 260328-iqt8-standardize-tmux-tab-naming
**Created**: 2026-03-28
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing operator6 tab naming — only operator7 is in scope
- Changing the operator's own tab name (`operator`) — only spawned agent tabs are affected
- Any runtime code changes — this is a skill file (markdown) update only

## Operator7: Tmux Tab Naming

### Requirement: Consistent Tab Name Format

All `tmux new-window` invocations in `/fab-operator7` that spawn agent tabs SHALL use the format `⚡<wt>` where `<wt>` is the worktree name (the `--worktree-name` value passed to `wt create`, or the auto-generated name).

There SHALL be no space between the `⚡` character and the worktree name.
<!-- assumed: No space between ⚡ and worktree name — more compact, standard for emoji prefixes in terminal tabs -->

#### Scenario: Spawn agent for existing change
- **GIVEN** a change `260328-iqt8-standardize-tmux-tab-naming` exists with a worktree named `swift-fox`
- **WHEN** the operator spawns an agent via the "From existing change" path
- **THEN** the tmux tab name SHALL be `⚡swift-fox`

#### Scenario: Spawn agent from raw text
- **GIVEN** the user provides raw text "fix login after password reset" and a worktree named `bold-elk` is created
- **WHEN** the operator spawns an agent via the "From raw text" path
- **THEN** the tmux tab name SHALL be `⚡bold-elk`

#### Scenario: Spawn agent from backlog ID
- **GIVEN** a backlog item `iqt8` and a worktree named `calm-wave` is created
- **WHEN** the operator spawns an agent via the "From backlog ID or Linear issue" path
- **THEN** the tmux tab name SHALL be `⚡calm-wave`

### Requirement: All Spawn Paths Use Same Pattern

The tab naming pattern `⚡<wt>` SHALL be applied uniformly across all four `tmux new-window -n` occurrences in the skill file:

1. The generic "Open agent tab" example in the Spawning an Agent section
2. The "From existing change" path
3. The "From raw text" path
4. The "From backlog ID or Linear issue" path

No spawn path SHALL use a different naming scheme (such as `fab-<id>` or `fab-<wt>`).

#### Scenario: Verify all paths match
- **GIVEN** the operator7 skill file after this change
- **WHEN** searching for `tmux new-window -n` patterns
- **THEN** all matches SHALL use the `⚡<wt>` format
- **AND** no matches SHALL contain `fab-<id>` or `fab-<wt>` as the tab name argument

## Deprecated Requirements

### Tab Name Format `fab-<id>`

**Reason**: The `fab-<id>` format is unreliable because the change ID doesn't exist at spawn time for new changes (raw text and backlog paths). The worktree name is always available.
**Migration**: Replace with `⚡<wt>` format.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `⚡` (zap emoji) as prefix | Confirmed from intake #1 — backlog item explicitly specifies ⚡ | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use worktree name as the identifier | Confirmed from intake #2 — always available at spawn time | S:90 R:90 A:90 D:95 |
| 3 | Confident | No space between ⚡ and worktree name | Confirmed from intake #3 — compact format standard for emoji prefixes | S:60 R:95 A:70 D:60 |
| 4 | Certain | All four spawn paths use the same pattern | Confirmed from intake #4 — standardization is the explicit goal | S:90 R:90 A:90 D:95 |
| 5 | Confident | Only operator7 in scope, not operator6 | Confirmed from intake #5 — backlog specifically targets fab-operator7 | S:70 R:85 A:70 D:70 |
| 6 | Certain | No spec file exists for operator7 under docs/specs/skills/ | Verified by search — no SPEC-fab-operator7 file found; constitution rule applies only when spec exists | S:95 R:95 A:95 D:95 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
