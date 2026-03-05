# Spec: Add Code Review Scaffold & 5 Cs of Quality

**Change**: 260218-xkkc-add-code-review-5cs-quality
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/context-loading.md`

## Non-Goals

- Changing how the review sub-agent is dispatched or structured — `fab-continue.md` review behavior remains agent-agnostic
- Making `code-review.md` mandatory — skills MUST proceed without error if missing
- Auto-migrating existing projects — existing projects pick up the scaffold on next `/fab-setup` run

## Scaffold: `fab/.kit/scaffold/code-review.md`

### Requirement: Scaffold SHALL provide populated review policy defaults

`fab/.kit/scaffold/code-review.md` SHALL contain the following sections with populated defaults (not empty headings):

1. **`## Severity Definitions`** — Defines the three-tier priority scheme: must-fix (spec mismatches, failing tests, checklist violations), should-fix (code quality issues, pattern inconsistencies), nice-to-have (style suggestions, minor improvements). Defaults MUST match the definitions currently hardcoded in `fab-continue.md` review behavior.

2. **`## Review Scope`** — Defines what the review sub-agent inspects. Default: changed files only. MAY include guidance on excluding generated code, vendor directories, or specific paths.

3. **`## False Positive Policy`** — How to suppress or override findings. Default: inline `<!-- review-ignore: {reason} -->` comments.

4. **`## Rework Budget`** — Max auto-rework cycles before escalation. Default: 3 cycles (matching current `/fab-fff` behavior).

5. **`## Project-Specific Review Rules`** — Empty section with guidance comments for project-specific rules (e.g., "all public APIs need integration tests").

Each section SHALL include an HTML comment block explaining its purpose, similar to the `code-quality.md` scaffold pattern.

#### Scenario: New project runs /fab-setup
- **GIVEN** a project without `fab/code-review.md`
- **WHEN** the user runs `/fab-setup`
- **THEN** `fab/code-review.md` is created from `fab/.kit/scaffold/code-review.md`
- **AND** all five sections contain populated defaults with guidance comments

#### Scenario: Existing project already has code-review.md
- **GIVEN** a project with an existing `fab/code-review.md`
- **WHEN** the user runs `/fab-setup`
- **THEN** the existing file is NOT overwritten

## Context Loading: Always Load Layer

### Requirement: `_context.md` SHALL include `fab/code-review.md` as a 7th Always Load file

The "Always Load" list in `fab/.kit/skills/_context.md` SHALL add `fab/code-review.md` as item 5 (after `fab/code-quality.md`, before `docs/memory/index.md`):

```
5. `fab/code-review.md` — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
```

This groups all `fab/` configuration files together, followed by the `docs/` landscape files. The file is optional — skills SHALL proceed without error if missing. This follows the same pattern as items 3 (`context.md`) and 4 (`code-quality.md`).

The exception list ("except `/fab-setup`, `/fab-status`, `/docs-hydrate-memory`") remains unchanged — these skills already skip the Always Load layer.

#### Scenario: Skill loads context with code-review.md present
- **GIVEN** `fab/code-review.md` exists
- **WHEN** any skill executes the Always Load layer
- **THEN** `code-review.md` is loaded alongside the existing 6 files

#### Scenario: Skill loads context without code-review.md
- **GIVEN** `fab/code-review.md` does not exist
- **WHEN** any skill executes the Always Load layer
- **THEN** the skill proceeds without error using the existing 6 files

## Setup: Bootstrap and Config Menu

### Requirement: `/fab-setup` SHALL scaffold `fab/code-review.md` during bootstrap

During the bootstrap sequence (Step 1b), `/fab-setup` SHALL add a new sub-step after `1b3. fab/code-quality.md`:

**`1b4. fab/code-review.md`**: If missing, copy `fab/.kit/scaffold/code-review.md` to `fab/code-review.md`. Report "Created: fab/code-review.md". If exists, skip.

This follows the identical pattern used for `context.md` (1b2) and `code-quality.md` (1b3).

#### Scenario: Fresh bootstrap creates code-review.md
- **GIVEN** a new project without `fab/code-review.md`
- **WHEN** `/fab-setup` runs bootstrap
- **THEN** `fab/code-review.md` is created from the scaffold
- **AND** the output includes "Created: fab/code-review.md"

#### Scenario: Re-run bootstrap preserves existing code-review.md
- **GIVEN** a project with a customized `fab/code-review.md`
- **WHEN** `/fab-setup` runs bootstrap again
- **THEN** the existing file is not modified

### Requirement: Config menu SHALL include `code-review.md` as item 10

The `/fab-setup config` menu SHALL add item 10:

```
10. `code-review.md` — review policy for the validation sub-agent
```

The editing behavior follows the same pattern as items 8 (`context.md`) and 9 (`code-quality.md`) — opens the file for editing, validates markdown structure.

#### Scenario: User edits code-review.md via config menu
- **GIVEN** `/fab-setup config` is invoked
- **WHEN** the user selects item 10 (`code-review.md`)
- **THEN** `fab/code-review.md` is opened for editing

## Review Behavior: Sub-Agent Context

### Requirement: Review sub-agent context SHALL include `fab/code-review.md`

The "Context provided to the sub-agent" list in `fab-continue.md` review behavior SHALL include `fab/code-review.md` (if present). The sub-agent SHALL use the file's severity definitions, scope boundaries, false positive policy, and rework budget when these override the skill's defaults.

When `fab/code-review.md` is absent, the sub-agent SHALL use the hardcoded defaults in the skill prompt (current behavior, unchanged).

#### Scenario: Review with project-specific severity definitions
- **GIVEN** `fab/code-review.md` defines custom severity definitions
- **WHEN** the review sub-agent performs validation
- **THEN** findings are classified using the project's definitions, not the skill defaults

#### Scenario: Review without code-review.md
- **GIVEN** `fab/code-review.md` does not exist
- **WHEN** the review sub-agent performs validation
- **THEN** the hardcoded three-tier priority scheme is used (current behavior)

## Config Scaffold: Companion Files Comment

### Requirement: Scaffold `config.yaml` SHALL reference `fab/code-review.md`

The companion files comment block at the top of `fab/.kit/scaffold/config.yaml` SHALL add:

```yaml
#   fab/code-review.md    — review policy for validation sub-agent (optional)
```

This is inserted after the `fab/code-quality.md` line.

#### Scenario: New project config.yaml includes code-review.md reference
- **GIVEN** a new project running `/fab-setup`
- **WHEN** `config.yaml` is generated from the scaffold
- **THEN** the companion files comment includes `fab/code-review.md`

## README: 5 Cs Mental Model

### Requirement: README SHALL introduce the 5 Cs of Quality

The "Code Quality as a Guardrail" section in `README.md` SHALL be expanded to introduce the 5 Cs mental model. The 5 Cs SHALL be presented as a table mapping each configuration file to the question it answers:

| C | File | Question |
|---|------|----------|
| **Constitution** | `fab/constitution.md` | What are our non-negotiable principles? |
| **Context** | `fab/context.md` | What are we working with? |
| **Code Quality** | `fab/code-quality.md` | How should code look when we write it? |
| **Code Review** | `fab/code-review.md` | What should we look for when we validate? |
| **Config** | `fab/config.yaml` | What are the project's factual settings? |

The narrative SHOULD explain the author-vs-critic distinction: `code-quality.md` guides the writing agent during apply; `code-review.md` guides the reviewing sub-agent during review. Different cognitive modes, different concerns.

The table SHOULD appear after the existing constitution/review explanation, before the "Structured Autonomy" section.

#### Scenario: Reader finds the 5 Cs table
- **GIVEN** a user reading the README
- **WHEN** they reach "Code Quality as a Guardrail"
- **THEN** the 5 Cs table is visible with all five files and their questions
- **AND** the author-vs-critic distinction is explained

## Design Decisions

1. **Created by `/fab-setup`, not sync script**: The intake referenced `3-sync-workspace.sh` for scaffolding. Actual codebase evidence shows companion markdown files (`context.md`, `code-quality.md`) are created by `/fab-setup` bootstrap (step 1b), not the sync script. The sync script handles structural assets (directories, symlinks, gitignore, settings). Following the established pattern.
   - *Why*: `/fab-setup` is the interactive bootstrap entry point. Users expect config menu items and bootstrap steps to use the same mechanism.
   - *Rejected*: Adding scaffolding to the sync script — would break the separation between structural sync (idempotent, non-interactive) and project bootstrap (interactive, config-aware).

2. **Populated defaults over empty sections**: Scaffold includes working defaults that match the current hardcoded behavior in `fab-continue.md`.
   - *Why*: A scaffold with only headings provides no value — users must read the skill prompt to understand the defaults. Populated defaults are immediately useful and serve as documentation.
   - *Rejected*: Empty sections with only guidance comments — shifts the burden to the user without providing a useful starting point.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | code-review.md is optional — skills proceed without error if missing | Same pattern as context.md and code-quality.md; backward compatible. Confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Scaffold lives at `fab/.kit/scaffold/code-review.md` | All scaffolds follow this pattern. Confirmed from intake #2 | S:95 R:90 A:95 D:95 |
| 3 | Certain | Created by `/fab-setup` bootstrap, not sync script | Codebase evidence: `context.md` and `code-quality.md` are created by `/fab-setup` step 1b, not `3-sync-workspace.sh`. Corrected from intake #3 (was Confident) | S:95 R:85 A:95 D:95 |
| 4 | Confident | Scaffold includes populated defaults matching current skill behavior | User intent is useful starting content. Existing `code-quality.md` scaffold follows this pattern. Confirmed from intake #4 | S:80 R:90 A:85 D:75 |
| 5 | Confident | README expansion under existing "Code Quality as a Guardrail" heading | User explicitly referenced this heading. Confirmed from intake #5 | S:85 R:90 A:85 D:75 |
| 6 | Certain | Memory files updated during hydrate, not during apply | Standard fab workflow — memory is post-implementation. Confirmed from intake #6 | S:95 R:90 A:95 D:95 |
| 7 | Confident | Review sub-agent respects code-review.md overrides but falls back to hardcoded defaults | Backward compatibility requires fallback. The override mechanism should be simple — if the section exists, use it; otherwise use defaults | S:75 R:85 A:80 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
