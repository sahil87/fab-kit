# Fab Templates

> Templates that ship in `fab/.kit/templates/`. Each is a markdown scaffold that skills fill with concrete content. Guidance comments (`<!-- -->`) are instructions to the agent, not preserved in output.

---

## proposal.md

```markdown
# Proposal: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Created**: {DATE}
**Status**: Draft

## Why

<!-- Explain the motivation. What problem does this solve? Why now? 1-3 sentences. -->

## What Changes

<!-- Be specific about new capabilities, modifications, or removals. Use bullets. -->

## Affected Specs

### New Specs
<!-- Specs being introduced. Each creates a delta file in this change's specs/ folder.
     Use kebab-case identifiers matching the centralized spec path. -->
- `{domain}/{spec-name}`: {brief description}

### Modified Specs
<!-- Existing specs whose requirements are changing. Reference by path in fab/specs/.
     Only list if spec-level behavior changes — implementation-only changes don't need deltas. -->
- `{domain}/{spec-name}`: {what requirement is changing}

### Removed Specs
<!-- Specs being fully deprecated/removed. Rare — usually individual requirements are removed via delta. -->

## Impact

<!-- Affected code areas, APIs, dependencies, systems. Helps scope the plan. -->

## Open Questions

<!-- Clarifying questions the agent couldn't resolve from context alone.
     Mark each with priority: [BLOCKING] must resolve before specs, [DEFERRED] can resolve during plan.
     Maximum 3 [BLOCKING] questions — make informed guesses for the rest. -->

- [BLOCKING] {question}
- [DEFERRED] {question}
```

**Design rationale**: OpenSpec's concise Why/What/Impact structure, plus explicit spec mapping (which delta files will be created). SpecKit's capped clarification markers prevent question-paralysis — max 3 blocking questions forces the agent to make informed guesses.

---

## spec.md (Delta Spec)

```markdown
# {Domain} Specification Changes

**Change**: {YYMMDD-XXXX-slug}
**Base spec**: `fab/specs/{domain}/{spec-name}.md`

<!--
  DELTA SPEC FORMAT
  This file describes CHANGES relative to the centralized spec.
  Only include sections that apply (omit empty ADDED/MODIFIED/REMOVED sections).

  Requirements use RFC 2119 keywords: MUST/SHALL (mandatory), SHOULD (recommended), MAY (optional).
  Every requirement MUST have at least one scenario.
  Scenarios use GIVEN/WHEN/THEN format.
-->

## ADDED Requirements

### Requirement: {Requirement Name}
{Requirement text using SHALL/MUST/SHOULD/MAY}

#### Scenario: {Scenario Name}
- **GIVEN** {precondition}
- **WHEN** {action or event}
- **THEN** {expected outcome}
- **AND** {additional outcome, if needed}

#### Scenario: {Another Scenario}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {outcome}

---

## MODIFIED Requirements

### Requirement: {Existing Requirement Name}
<!-- Must include the FULL updated requirement text, not just the diff.
     The agent uses this complete block to replace the existing requirement during hydration. -->

{Updated requirement text}

**Previous behavior**: {Brief summary of what changed, for reviewer context}

#### Scenario: {Updated or New Scenario}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {new expected outcome}

---

## REMOVED Requirements

### Requirement: {Requirement Name}
**Reason**: {Why this requirement is being removed}
**Migration**: {What replaces it, or "N/A" if simply deprecated}
```

**Design rationale**: OpenSpec's delta format (ADDED/MODIFIED/REMOVED) is the core. SpecKit's GIVEN/WHEN/THEN scenarios and RFC 2119 keywords add precision. MODIFIED requires full replacement text (not a diff) because the agent performs semantic merge during hydration. The "Previous behavior" note is for human reviewers, not the merge.

---

## plan.md

```markdown
# Plan: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Created**: {DATE}
**Proposal**: `proposal.md`
**Delta specs**: `specs/`

## Summary

<!-- 1-2 sentences: what this change does + the chosen technical approach. -->

## Goals / Non-Goals

**Goals:**
<!-- What this implementation aims to achieve. Derived from the delta specs. -->

**Non-Goals:**
<!-- What is explicitly out of scope. Prevents scope creep during apply. -->

## Technical Context

<!-- Fill from project config.yaml context + research. Only include relevant fields. -->

- **Relevant stack**: {subset of tech stack that this change touches}
- **Key dependencies**: {libraries, services, APIs involved}
- **Constraints**: {performance, compatibility, security constraints}

## Research

<!-- Findings from technical investigation needed for this change.
     Skip this section for straightforward changes where the approach is obvious.
     Include: library evaluations, API docs consulted, architecture patterns considered. -->

## Decisions

<!-- Key design decisions with rationale. Format: "Decision: X. Why: Y. Alternatives rejected: Z."
     This section is the most valuable part of the plan — it captures WHY, not just WHAT. -->

1. **{Decision}**: {chosen approach}
   - *Why*: {rationale}
   - *Rejected*: {alternative and why it was worse}

## Risks / Trade-offs

<!-- Known risks with mitigation strategies. Known trade-offs and why they're acceptable. -->

## File Changes

<!-- Which files will be created, modified, or deleted. Gives a concrete scope for tasks. -->

### New Files
- `{path}`: {purpose}

### Modified Files
- `{path}`: {what changes}

### Deleted Files
- `{path}`: {why}
```

**Design rationale**: OpenSpec's Goals/Non-Goals + Decisions/Risks structure for architectural clarity. SpecKit's research phase for unknowns. The File Changes section bridges plan to tasks — it makes the scope concrete and reviewable before task generation. Constitution check from SpecKit is handled by the project's `constitution.md` being loaded as context, not as a template section.

---

## tasks.md

```markdown
# Tasks: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Plan**: `plan.md`
**Delta specs**: `specs/`

<!--
  TASK FORMAT: - [ ] {ID} [{markers}] {Description with file paths}

  Markers (optional, combine as needed):
    [P]   — Parallelizable (different files, no dependencies on other [P] tasks in same group)

  IDs are sequential: T001, T002, ...
  Include exact file paths in descriptions.
  Each task should be completable in one focused session.

  Tasks are grouped by phase. Phases execute sequentially.
  Within a phase, [P] tasks can execute in parallel.
-->

## Phase 1: Setup

<!-- Scaffolding, dependencies, configuration. No business logic. -->

- [ ] T001 {setup task with file path}
- [ ] T002 [P] {parallel setup task}
- [ ] T003 [P] {parallel setup task}

## Phase 2: Core Implementation

<!-- Primary functionality. Order by dependency — earlier tasks are prerequisites for later ones. -->

- [ ] T004 {implementation task referencing specific file path}
- [ ] T005 {task that depends on T004}
- [ ] T006 [P] {independent task}

## Phase 3: Integration & Edge Cases

<!-- Wire components together. Handle error states, edge cases, validation. -->

- [ ] T007 {integration task}
- [ ] T008 {error handling task}

## Phase 4: Polish

<!-- Documentation, cleanup, performance. Only include if warranted by the change scope. -->

- [ ] T009 {polish task}

---

## Execution Order

<!-- Summary of dependencies between tasks. Only include non-obvious dependencies. -->

- T004 blocks T005
- T006 is independent, can run alongside T004-T005
```

**Design rationale**: Simpler than SpecKit's user-story-per-phase structure — Fab changes tend to be smaller and more focused than greenfield features. OpenSpec's flat numbered groups (1.1, 1.2) were too minimal for dependency tracking, so we keep SpecKit's `[P]` parallel markers and sequential task IDs. The execution order section replaces SpecKit's elaborate dependency tree with a concise "only non-obvious deps" summary. Phases are by implementation concern (setup/core/integration/polish), not by user story, since delta changes usually touch one capability.

For larger changes that span multiple user stories, the agent should adapt by splitting Phase 2 into per-story sub-phases, following SpecKit's pattern:

```markdown
## Phase 2a: User Story — {Title} (P1)
- [ ] T004 [US1] {task}

## Phase 2b: User Story — {Title} (P2)
- [ ] T007 [US2] {task}
```

---

## checklist.md (Quality Checklist)

```markdown
# Quality Checklist: {CHANGE_NAME}

**Change**: {YYMMDD-XXXX-slug}
**Generated**: {DATE}
**Delta specs**: `specs/`

<!--
  AUTO-GENERATED by /fab:continue or /fab:ff at the tasks stage.

  This checklist validates that the IMPLEMENTATION matches the SPECS.
  Items are derived from:
    1. Delta specs — every ADDED/MODIFIED requirement should have coverage
    2. Plan decisions — technical choices should be reflected in code
    3. Project constitution — project-wide quality standards

  Categories below are defaults. Add project-specific categories from config.yaml.
  /fab:verify checks all items. ALL must pass before /fab:archive.
-->

## Functional Completeness
<!-- Every ADDED requirement has working implementation -->
- [ ] CHK-001 {requirement name}: {specific verifiable criterion}
- [ ] CHK-002 {requirement name}: {criterion}

## Behavioral Correctness
<!-- Every MODIFIED requirement behaves as specified, not as before -->
- [ ] CHK-003 {requirement name}: {what changed and how to verify}

## Removal Verification
<!-- Every REMOVED requirement is actually gone -->
- [ ] CHK-004 {requirement name}: {confirm removed, no dead code}

## Scenario Coverage
<!-- Key scenarios from delta specs have been exercised -->
- [ ] CHK-005 {scenario name}: {how to verify — test exists, manual check, etc.}
- [ ] CHK-006 {scenario name}: {verification method}

## Edge Cases & Error Handling
<!-- Error states, boundary conditions, failure modes -->
- [ ] CHK-007 {edge case}: {what should happen}

## Security
<!-- Only include if the change has security surface -->
- [ ] CHK-008 {security concern}: {verification}

## Notes

- Check items as you verify: `- [x]`
- All items must pass before `/fab:archive`
- If an item is not applicable, replace `[ ]` with `[N/A]` and explain why
```

**Design rationale**: Unlike SpecKit's checklist (which tests requirement *quality* — "are the specs well-written?"), Fab's checklist tests *implementation fidelity* — "does the code match the specs?" This is because Fab has explicit spec review during the specs stage, making a separate requirement-quality checklist redundant. The categories map directly to delta operations (ADDED -> completeness, MODIFIED -> correctness, REMOVED -> cleanup) plus cross-cutting concerns.

---

## Centralized Spec Format (`fab/specs/`)

Centralized specs are the **source of truth** for what the system does. They are organized hierarchically with index files for navigation.

### Directory Structure

```
fab/specs/
├── index.md                    # Top-level index: lists all domains
├── auth/
│   ├── index.md                # Domain index: lists all specs in auth/
│   ├── authentication.md       # Individual spec
│   └── authorization.md
├── payments/
│   ├── index.md
│   ├── checkout.md
│   └── refunds.md
└── users/
    ├── index.md
    └── registration.md
```

### Top-Level Index (`fab/specs/index.md`)

```markdown
# Specifications Index

> Source of truth for system behavior. Updated by `/fab:archive` hydration.

| Domain | Description | Specs |
|--------|-------------|-------|
| [auth](auth/index.md) | Authentication and authorization | authentication, authorization |
| [payments](payments/index.md) | Payment processing and billing | checkout, refunds |
| [users](users/index.md) | User management | registration |
```

### Domain Index (`fab/specs/{domain}/index.md`)

```markdown
# {Domain} Specifications

| Spec | Description | Last Updated |
|------|-------------|-------------|
| [authentication](authentication.md) | User login, session management, OAuth | {DATE} |
| [authorization](authorization.md) | Roles, permissions, access control | {DATE} |
```

### Individual Spec (`fab/specs/{domain}/{name}.md`)

```markdown
# {Spec Name} Specification

**Domain**: {domain}
**Last hydrated**: {DATE} (from {change-name})

## Overview

<!-- 1-2 sentences describing what this spec covers. -->

## Requirements

### Requirement: {Requirement Name}
{Requirement text using SHALL/MUST/SHOULD/MAY}

#### Scenario: {Scenario Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {expected outcome}

### Requirement: {Another Requirement}
{text}

#### Scenario: {Scenario Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {expected outcome}

## History

<!-- Auto-maintained by /fab:archive. Most recent first. -->

| Change | Date | Summary |
|--------|------|---------|
| {change-name} | {DATE} | {one-line summary of what changed} |
```

**Design rationale**: The index-based hierarchy solves discoverability — agents and humans can navigate from top-level down to any requirement without scanning folders. The History table in each spec provides traceability back to the change that introduced each modification, which is critical for understanding *why* a requirement exists. Domain indexes include "Last Updated" so stale specs are visible at a glance.

### Hydration Rules

When `/fab:archive` hydrates delta specs into centralized specs:

1. **New spec file**: If the delta references a spec that doesn't exist yet, create it from the individual spec template and add it to the domain index. If the domain doesn't exist, create the domain folder and add it to the top-level index.
2. **Existing spec file**: Read the delta's ADDED/MODIFIED/REMOVED sections and update the centralized spec semantically (not mechanically). Minimize edits to unchanged sections.
3. **Index updates**: Update domain index "Last Updated" column. Add new entries if new specs were created.
4. **History row**: Append a row to the spec's History table with the change name, date, and one-line summary.
