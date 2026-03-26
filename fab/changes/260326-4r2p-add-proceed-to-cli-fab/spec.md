# Spec: Add fab-proceed to operator skill Pipeline References

**Change**: 260326-4r2p-add-proceed-to-cli-fab
**Created**: 2026-03-26
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying `_cli-fab.md` — the previous changes to that file have been reverted
- Changing operator behavior logic — only the Pipeline Reference knowledge section is updated
- Documenting `/fab-proceed` internals — operators only need to know what it does, not how

## Operator Skills: Pipeline Reference Update

### Requirement: fab-operator7 Pipeline Reference SHALL include `/fab-proceed`

The Pipeline Reference section in `fab/.kit/skills/fab-operator7.md` (§6 Coordination Patterns) SHALL list `/fab-proceed` with a brief description: auto-detects state and runs needed prefix steps before `/fab-fff`.

#### Scenario: Operator7 decides how to start an agent on a new backlog item
- **GIVEN** operator7 has loaded its Pipeline Reference at startup
- **WHEN** it needs to send a command to an agent for a backlog item that has no intake yet
- **THEN** `/fab-proceed` SHALL be available in its command vocabulary
- **AND** the operator MAY choose `/fab-proceed` instead of manually chaining `/fab-new` → `/fab-switch` → `/git-branch` → `/fab-fff`

### Requirement: fab-operator6 Pipeline Reference SHALL include `/fab-proceed`

The Pipeline Reference section in `fab/.kit/skills/fab-operator6.md` (§6 Coordination Patterns) SHALL list `/fab-proceed` with the same description used in operator7.

#### Scenario: Operator6 decides how to start an agent on an existing intake
- **GIVEN** operator6 has loaded its Pipeline Reference at startup
- **WHEN** it needs to send a command to an agent for a change that has an intake but isn't activated
- **THEN** `/fab-proceed` SHALL be available in its command vocabulary

### Requirement: `/fab-proceed` SHALL be categorized as a pipeline command

`/fab-proceed` orchestrates both setup and pipeline steps, but its primary purpose is running the full pipeline. It SHALL be listed under **Pipeline commands** alongside `/fab-continue`, `/fab-fff`, and `/fab-ff`.

#### Scenario: Pipeline Reference categorization
- **GIVEN** the Pipeline Reference has Setup commands, Pipeline commands, and Maintenance categories
- **WHEN** `/fab-proceed` is added
- **THEN** it SHALL appear in the **Pipeline commands** line
- **AND** its description SHALL indicate it auto-detects state and delegates to `/fab-fff`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target is operator skill files, not _cli-fab.md | Confirmed from intake #1 — user corrected the target | S:95 R:95 A:95 D:95 |
| 2 | Certain | Both operator6 and operator7 need the update | Confirmed from intake #2 — verified both have identical Pipeline Reference sections | S:95 R:90 A:95 D:95 |
| 3 | Certain | List under Pipeline commands category | Upgraded from intake #3 Confident → Certain — fab-proceed's purpose is running the full pipeline; it delegates setup steps as prefix | S:85 R:90 A:90 D:85 |
| 4 | Certain | docs change type | Confirmed from intake #4 — only markdown files modified | S:95 R:95 A:95 D:95 |

4 assumptions (4 certain, 0 confident, 0 tentative, 0 unresolved).
