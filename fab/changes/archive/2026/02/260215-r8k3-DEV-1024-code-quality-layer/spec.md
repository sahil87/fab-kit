# Spec: Add Code Quality Layer

**Change**: 260215-r8k3-DEV-1024-code-quality-layer
**Created**: 2026-02-15
**Affected memory**:
- `docs/memory/fab-workflow/configuration.md` (modify)
- `docs/memory/fab-workflow/execution-skills.md` (modify)
- `docs/memory/fab-workflow/templates.md` (modify)

## Non-Goals

- Linting or static analysis integration — this is agent guidance, not tooling
- Enforcing code quality standards outside the Fab pipeline (e.g., pre-commit hooks)
- Auto-generating `code_quality` config from codebase analysis — projects write this manually

## Configuration: `code_quality` Section

### Requirement: Optional Code Quality Config

`config.yaml` SHALL support an optional `code_quality` section. The section MUST be commented out by default in the config template, matching the pattern of other optional sections. Projects opt in by uncommenting and populating.

#### Scenario: Project without code_quality config

- **GIVEN** a `config.yaml` with no `code_quality` section (or fully commented out)
- **WHEN** `/fab-continue` runs apply or review
- **THEN** baseline code quality behavior applies (pattern extraction + two default checklist items)
- **AND** no `code_quality`-derived constraints are loaded

#### Scenario: Project with code_quality config

- **GIVEN** a `config.yaml` with an active `code_quality` section defining `principles`, `anti_patterns`, and `test_strategy`
- **WHEN** `/fab-continue` runs apply or review
- **THEN** `principles` are loaded as additional implementation constraints during apply
- **AND** `anti_patterns` are loaded as review violation checks
- **AND** `test_strategy` governs when tests are written relative to implementation

### Requirement: Code Quality Config Schema

The `code_quality` section SHALL support the following fields:

- `principles` — List of strings. Positive coding standards to follow during apply. Each principle is a directive the agent treats as a soft constraint alongside the spec and constitution.
- `anti_patterns` — List of strings. Patterns to avoid. Checked during review with specific file:line references on violation.
- `test_strategy` — Enum string. Controls test timing relative to implementation. Valid values: `test-alongside` (default — write tests as you implement), `test-after` (implement first, then tests), `tdd` (write tests before implementation).

#### Scenario: Partial code_quality config

- **GIVEN** a `code_quality` section with only `principles` defined (no `anti_patterns` or `test_strategy`)
- **WHEN** `/fab-continue` loads configuration
- **THEN** `principles` are applied during apply
- **AND** `anti_patterns` check is skipped during review
- **AND** `test_strategy` defaults to `test-alongside`

### Requirement: Config Template Update

`/fab-init` SHALL include `code_quality` in its config management:

- The config create mode template SHALL include a commented-out `code_quality` block with example values
- `/fab-init config` menu SHALL include a `code_quality` item ("coding standards for apply/review")
- The valid sections list SHALL include `code_quality`

#### Scenario: fab-init config menu

- **GIVEN** user runs `/fab-init config`
- **WHEN** the section menu is displayed
- **THEN** `code_quality` appears as menu item 9 with description "coding standards for apply/review"
- **AND** the Done option is renumbered to 10

## Apply Behavior: Pattern Extraction

### Requirement: Pattern Extraction Before Task Execution

`/fab-continue` (apply) SHALL perform pattern extraction before executing the first unchecked task. This step reads existing source files in the areas the change will touch and extracts contextual patterns.

Pattern extraction SHALL capture:

1. **Naming conventions** — variable/function/class naming style observed in surrounding code
2. **Error handling** — how the codebase handles errors (exceptions, Result types, error codes, etc.)
3. **Structure** — typical function length, module boundaries, import organization
4. **Reusable utilities** — existing helpers or shared modules that new code should use instead of reimplementing

These patterns SHALL be held as context for all subsequent task execution within the same apply run.

#### Scenario: First apply run on a change

- **GIVEN** a change with unchecked tasks targeting files in `fab/.kit/skills/`
- **WHEN** `/fab-continue` dispatches to apply behavior
- **THEN** the agent reads existing files in `fab/.kit/skills/` to extract naming, error handling, structure, and utility patterns
- **AND** these patterns inform all subsequent task implementations

#### Scenario: Resume mid-apply

- **GIVEN** a change with some tasks already checked (`[x]`) and some unchecked
- **WHEN** `/fab-continue` resumes apply
- **THEN** pattern extraction is skipped (patterns are re-derived implicitly from reading task-relevant source files)

### Requirement: Code Quality Config Integration in Apply

If `config.yaml` defines a `code_quality` section, `/fab-continue` (apply) SHALL load its `principles` as additional implementation constraints alongside extracted patterns. If `code_quality.test_strategy` is defined, it SHALL govern test timing.

#### Scenario: Apply with code_quality principles

- **GIVEN** a `config.yaml` with `code_quality.principles` including "Prefer composition over inheritance"
- **WHEN** the agent implements a task that involves adding a class
- **THEN** the agent considers composition before inheritance, consistent with the principle

#### Scenario: Apply with test_strategy tdd

- **GIVEN** a `config.yaml` with `code_quality.test_strategy: tdd`
- **WHEN** the agent implements a task
- **THEN** the agent writes test expectations before writing the implementation

### Requirement: Expanded Per-Task Guidance

The current apply task execution step ("read source, implement per spec/constitution/patterns, run tests, fix failures, mark [x]") SHALL be expanded to:

1. Read source files relevant to this task
2. Implement per spec, constitution, and extracted patterns
3. Prefer reusing existing utilities over creating new ones
4. Keep functions focused — if implementation exceeds the codebase's typical function size, consider extracting
5. Write tests per `code_quality.test_strategy` (default: `test-alongside`)
6. Run tests, fix failures
7. Mark `[x]` immediately

#### Scenario: Task implementation reuses existing utility

- **GIVEN** pattern extraction identified a shared utility `lib/utils.sh`
- **WHEN** the agent implements a task that needs functionality already in `lib/utils.sh`
- **THEN** the agent reuses the utility instead of reimplementing the functionality

## Review Behavior: Code Quality Check

### Requirement: Code Quality Validation Step

`/fab-continue` (review) SHALL add a code quality check as step 6, after the existing step 5 (memory drift check). For each file modified during apply, the review SHALL verify:

1. Naming conventions consistent with surrounding code
2. Functions focused and appropriately sized
3. Error handling consistent with codebase style
4. Existing utilities reused where applicable

If `config.yaml` defines `code_quality.principles`, each applicable principle SHALL be checked. If `config.yaml` defines `code_quality.anti_patterns`, each SHALL be checked for violations.

#### Scenario: Review detects naming inconsistency

- **GIVEN** surrounding code uses `snake_case` for function names
- **WHEN** review inspects a modified file containing a new `camelCase` function
- **THEN** review flags a code quality failure with the specific file:line reference

#### Scenario: Review detects anti-pattern violation

- **GIVEN** `config.yaml` includes `anti_patterns: ["God functions (>50 lines without clear reason)"]`
- **WHEN** review inspects a modified file with a 70-line function that could be decomposed
- **THEN** review flags a code quality failure referencing the anti-pattern and file:line

### Requirement: Code Quality Failures Are Review Failures

Code quality issues detected in step 6 SHALL be treated as review failures with the same rework flow as spec mismatches. Each failure SHALL include a specific file:line reference.

#### Scenario: Code quality failure triggers rework options

- **GIVEN** review step 6 finds 2 code quality issues
- **WHEN** the review verdict is determined
- **THEN** the verdict is "fail" with the same rework options (fix code, revise tasks, revise spec)
- **AND** the failure details include file:line references for each quality issue

## Checklist: Code Quality Category

### Requirement: Code Quality Checklist Section

The checklist template SHALL include a `## Code Quality` section. This section SHALL always be included (unlike Security which requires security surface), because code quality applies to all changes that touch implementation code.

#### Scenario: Checklist generated without code_quality config

- **GIVEN** a `config.yaml` with no `code_quality` section
- **WHEN** `/fab-continue` generates `checklist.md` at the tasks stage
- **THEN** the checklist includes a `## Code Quality` section with two baseline items:
  - `CHK-{NNN} Pattern consistency: New code follows naming and structural patterns of surrounding code`
  - `CHK-{NNN} No unnecessary duplication: Existing utilities reused where applicable`

#### Scenario: Checklist generated with code_quality config

- **GIVEN** a `config.yaml` with `code_quality.principles` and `code_quality.anti_patterns`
- **WHEN** `/fab-continue` generates `checklist.md` at the tasks stage
- **THEN** the checklist includes a `## Code Quality` section with:
  - The two baseline items
  - One additional item per relevant `principle` that applies to the change's scope
  - One additional item per relevant `anti_pattern` that applies to the change's scope

### Requirement: Checklist Generation Procedure Update

The Checklist Generation Procedure in `_generation.md` SHALL add `code_quality` as a derivation source in step 4, alongside existing spec-derived sources. When no `code_quality` section exists in config, the two baseline items SHALL still be included.

#### Scenario: Checklist generation with extra_categories and code_quality

- **GIVEN** a config with `checklist.extra_categories: [documentation_accuracy]` and a `code_quality` section
- **WHEN** checklist is generated
- **THEN** the checklist includes both the `## Code Quality` section and the `## Documentation Accuracy` section

## Context Loading: Source Code Expansion

### Requirement: Apply-Stage Source Context

The Source Code Loading section in `_context.md` SHALL add an apply-specific step: when loading source code for the apply stage, also read neighboring files in the same directories to extract pattern context (naming, error handling, structure). This supports the Pattern Extraction requirement.

#### Scenario: Apply loads neighboring files

- **GIVEN** a task references `fab/.kit/skills/fab-continue/fab-continue.md`
- **WHEN** source code context is loaded for apply
- **THEN** other files in `fab/.kit/skills/fab-continue/` are also read for pattern extraction
- **AND** files in `fab/.kit/skills/` (sibling directories) MAY be sampled for broader pattern context

### Requirement: Review-Stage Source Context

The Source Code Loading section in `_context.md` SHALL add a review-specific step: re-read modified files to validate consistency with surrounding code. This supports the Code Quality Validation requirement.

#### Scenario: Review re-reads modified files

- **GIVEN** apply modified 3 files
- **WHEN** source code context is loaded for review
- **THEN** all 3 modified files are re-read
- **AND** their surrounding code (same directory) is read for consistency comparison

## Hydrate Behavior: Pattern Capture

### Requirement: Optional Pattern Capture

`/fab-continue` (hydrate) SHALL include an optional step 5: if the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section.

This step SHALL be skipped for implementations that follow existing patterns without introducing new ones.

#### Scenario: Change introduces new pattern

- **GIVEN** a change that introduced a new shared utility pattern in `lib/`
- **WHEN** hydrate runs
- **THEN** the new pattern is noted in the relevant memory file's Design Decisions section
- **AND** the entry includes the change name for traceability

#### Scenario: Change follows existing patterns

- **GIVEN** a change that implemented using existing codebase patterns without introducing new ones
- **WHEN** hydrate runs
- **THEN** pattern capture step 5 is skipped
- **AND** no new Design Decisions entries are created for implementation patterns

## Design Decisions

1. **Code quality is guidance, not gating**: The code quality layer provides context and checks, not hard gates. Pattern extraction informs the agent; review flags issues for rework. There is no blocking gate that prevents apply from starting.
   - *Why*: Matches Fab's lightweight philosophy (constitution loaded implicitly, not gated — see configuration memory). A gate would add friction without proportional benefit.
   - *Rejected*: Pre-apply quality gate that blocks execution until patterns are reviewed — too heavyweight, slows down the pipeline.

2. **Always-included checklist section vs. opt-in**: Code Quality is always present in the checklist (with at least 2 baseline items), unlike Security which is conditional.
   - *Why*: Every code change has quality concerns. Requiring opt-in would mean most projects get no quality checking by default. The baseline items (pattern consistency, utility reuse) are universally applicable.
   - *Rejected*: Conditional inclusion (like Security) — would miss quality checks on projects that haven't configured `code_quality`.

3. **Two baseline items as minimum**: When no `code_quality` config exists, exactly two checklist items are generated: pattern consistency and no unnecessary duplication.
   - *Why*: These are universal, non-controversial, and verifiable by code inspection. More items without project-specific context would be generic to the point of being unhelpful.
   - *Rejected*: More baseline items (e.g., function size, error handling) — too opinionated without project context.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Changes go into kit skills/templates, not project-specific files | Confirmed from brief #1 — this is a framework improvement for all fab-kit users | S:95 R:90 A:95 D:95 |
| 2 | Certain | `code_quality` config is optional/commented-out by default | Confirmed from brief #2 — matches existing config patterns | S:90 R:95 A:90 D:90 |
| 3 | Certain | fab-ff/fff don't need direct changes | Confirmed from brief #3 — they delegate to fab-continue behaviors | S:95 R:90 A:95 D:95 |
| 4 | Certain | Code quality checklist items always included (not opt-in) | Confirmed from brief #4 — unlike Security, quality applies to all changes | S:85 R:85 A:85 D:90 |
| 5 | Confident | Pattern Extraction runs once before first task, not per-task | Confirmed from brief #5 — per-task extraction is redundant; patterns for the change area are stable across tasks | S:80 R:85 A:70 D:75 |
| 6 | Confident | Pattern capture in hydrate is optional, not mandatory | Confirmed from brief #6 — most implementations follow existing patterns | S:75 R:90 A:70 D:70 |
| 7 | Confident | Baseline checklist items: pattern consistency + no unnecessary duplication | Confirmed from brief #7 — universal and non-controversial quality dimensions | S:70 R:85 A:70 D:65 |
| 8 | Confident | test_strategy options: test-alongside (default), test-after, tdd | Confirmed from brief #8 — covers main approaches without over-complicating | S:75 R:85 A:65 D:70 |
| 9 | Certain | code_quality fields are all optional (partial config supported) | Config pattern: each field independent, missing fields use defaults | S:85 R:90 A:90 D:90 |
| 10 | Confident | Code quality review failures use same rework flow as spec mismatches | Review already has structured rework options; adding a parallel flow would be confusing | S:80 R:80 A:75 D:80 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
