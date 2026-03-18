# Spec: Standard Subagent Context Template

**Change**: 260318-dzze-standard-subagent-context
**Created**: 2026-03-18
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Changing `fab-ff.md` or `fab-fff.md` — they already reference `_preamble.md § Subagent Dispatch` and inherit automatically
- Adding `docs/memory/index.md` or `docs/specs/index.md` to standard subagent context — standard context covers project principles only, not documentation navigation
- Changing which files the *parent agent* loads (the always-load layer is unchanged)

## Preamble: Standard Subagent Context

### Requirement: Standard Subagent Context Block in Dispatch Pattern

The `_preamble.md` Subagent Dispatch section SHALL include a "Standard Subagent Context" subsection that defines the project files every subagent prompt MUST include. The subsection SHALL appear after the existing 5-item dispatch pattern list (as item 6) and before the `general-purpose` subagents note.

#### Scenario: Dispatching agent constructs subagent prompt
- **GIVEN** an orchestrator or skill dispatches a subagent via the Agent tool
- **WHEN** constructing the subagent prompt
- **THEN** the prompt MUST instruct the subagent to read the following files before executing its task:
  - `fab/project/config.yaml`
  - `fab/project/constitution.md`
  - `fab/project/context.md` (skip gracefully if missing)
  - `fab/project/code-quality.md` (skip gracefully if missing)
  - `fab/project/code-review.md` (skip gracefully if missing)

#### Scenario: Optional files missing
- **GIVEN** the project does not have `context.md`, `code-quality.md`, or `code-review.md`
- **WHEN** a subagent attempts to read these files
- **THEN** the subagent SHALL skip missing optional files without error
- **AND** `config.yaml` and `constitution.md` remain required (subagent reports error if missing)

#### Scenario: Nested subagent dispatch
- **GIVEN** a subagent dispatches its own sub-subagent (e.g., review sub-agent within fab-continue)
- **WHEN** constructing the inner subagent prompt
- **THEN** the inner prompt MUST also include the standard subagent context instruction
- **AND** the same 5 files are loaded at every nesting level

### Requirement: Dispatch Pattern Updated to Reference Standard Context

The existing dispatch pattern numbered list SHALL be extended with item 6 referencing the standard subagent context. The item SHALL read: "The standard subagent context files (see below)" — pointing to the new subsection.

#### Scenario: Orchestrator skill reads dispatch pattern
- **GIVEN** an orchestrator skill reads `_preamble.md § Subagent Dispatch`
- **WHEN** it encounters the dispatch pattern list
- **THEN** item 6 directs it to include the standard subagent context files
- **AND** the subsection provides the exact file list and instruction pattern

## fab-continue: Review Behavior Context Simplification

### Requirement: Review Sub-Agent Context References Standard Subagent Context

The `fab-continue.md` Review Behavior "Context provided to the sub-agent" paragraph SHALL replace the ad-hoc `fab/project/**` file list with a reference to `_preamble.md § Standard Subagent Context`. Change-specific context (spec.md, tasks.md, checklist.md, relevant source files, target memory files) SHALL remain listed explicitly.

#### Scenario: Review sub-agent receives project principles
- **GIVEN** `/fab-continue` dispatches a review sub-agent
- **WHEN** the review sub-agent starts execution
- **THEN** it reads all 5 standard subagent context files (per `_preamble.md`)
- **AND** it also reads the change-specific files: `spec.md`, `tasks.md`, `checklist.md`, relevant source files, and target memory files from `docs/memory/`

#### Scenario: Previously ad-hoc list removed
- **GIVEN** a developer reads `fab-continue.md` Review Behavior
- **WHEN** looking for which `fab/project/**` files to provide to the sub-agent
- **THEN** the section references `_preamble.md § Standard Subagent Context` instead of listing individual files
- **AND** no `fab/project/**` file names appear inline in Review Behavior

### Requirement: Apply Behavior Code-Quality Reference Annotated

The Apply Behavior's existing `code-quality.md` reference (for `## Principles` and `## Test Strategy`) SHALL remain as-is for its specific section extraction guidance. No annotation needed — the standard context ensures the file is loaded; Apply Behavior's reference tells the agent *what to extract from it*.

#### Scenario: Apply subagent uses code-quality.md
- **GIVEN** an apply subagent loads standard subagent context
- **WHEN** it reads Apply Behavior instructions
- **THEN** `fab/project/code-quality.md` is already loaded via standard context
- **AND** Apply Behavior's `## Principles` and `## Test Strategy` references guide what to extract

## Specs: SPEC-preamble.md Creation

### Requirement: Create SPEC-preamble.md

A new `docs/specs/skills/SPEC-preamble.md` SHALL be created documenting the `_preamble.md` internal partial's flow, tool usage, sub-agents, and bookkeeping candidates — following the same format as existing SPEC files (e.g., `SPEC-fab-continue.md`).

#### Scenario: SPEC-preamble.md reflects standard subagent context
- **GIVEN** `_preamble.md` now includes the Standard Subagent Context subsection
- **WHEN** `SPEC-preamble.md` is generated
- **THEN** the flow diagram shows the standard context files in the subagent dispatch pattern
- **AND** the tools table and sub-agents table reflect the preamble's role as a shared reference (no direct tool usage or sub-agents — it's a convention document)

### Requirement: Update SPEC-fab-continue.md

The existing `docs/specs/skills/SPEC-fab-continue.md` SHALL be updated to reflect the review sub-agent's context loading change — replacing the inline file list in the flow diagram with a reference to standard subagent context.

#### Scenario: Review sub-agent box updated
- **GIVEN** the SPEC-fab-continue.md flow diagram shows review sub-agent context
- **WHEN** the spec is updated
- **THEN** the sub-agent box references "standard subagent context + change-specific files"
- **AND** individual `fab/project/**` file names are removed from the sub-agent box

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All 5 `fab/project/**` files loaded in every subagent | Confirmed from intake #1 — user explicitly confirmed | S:95 R:90 A:95 D:95 |
| 2 | Certain | Template goes in `_preamble.md` Subagent Dispatch section | Confirmed from intake #2 — user confirmed location | S:95 R:85 A:90 D:95 |
| 3 | Certain | `fab-continue.md` Review Behavior context list simplified to reference preamble | Confirmed from intake #3 — original proposal | S:90 R:85 A:90 D:90 |
| 4 | Certain | No changes needed to `fab-ff.md` or `fab-fff.md` | Confirmed from intake #4 — verified no ad-hoc context lists | S:95 R:90 A:95 D:95 |
| 5 | Confident | Template specifies "read these files first" instruction pattern | Confirmed from intake #5 — subagents need explicit file-read instructions | S:70 R:80 A:75 D:80 |
| 6 | Certain | `docs/memory/index.md` and `docs/specs/index.md` excluded from standard subagent context | Confirmed from intake #6 — user confirmed project principles only | S:95 R:85 A:95 D:95 |
| 7 | Certain | Create `docs/specs/skills/SPEC-preamble.md` as part of this change | Confirmed from intake #7 — user requested | S:95 R:85 A:90 D:95 |
| 8 | Certain | Apply Behavior code-quality.md reference left as-is | Spec-level decision — the reference tells the agent what sections to extract, standard context handles loading; no annotation needed | S:85 R:90 A:85 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
