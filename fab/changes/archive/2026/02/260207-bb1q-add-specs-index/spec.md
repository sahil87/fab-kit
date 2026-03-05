# Spec: Add `fab/specs/` Index and Clarify Specs vs Docs Distinction

**Change**: 260207-bb1q-add-specs-index
**Created**: 2026-02-07
**Affected docs**: `fab/docs/fab-workflow/init.md`, `fab/docs/fab-workflow/context-loading.md` (modified); `fab/docs/fab-workflow/specs-index.md` (new)

## Fab Init: Specs Index Creation

### Requirement: Create `fab/specs/index.md` During Bootstrap

`/fab-init` SHALL create `fab/specs/index.md` as a new idempotent bootstrap step, alongside the existing `fab/docs/index.md` step. If the file already exists, it SHALL be skipped with a status message.

#### Scenario: First run — specs index does not exist
- **GIVEN** `fab/specs/index.md` does not exist
- **WHEN** `/fab-init` runs
- **THEN** `fab/specs/` directory is created
- **AND** `fab/specs/index.md` is created with the specs index boilerplate
- **AND** the output includes "Created: fab/specs/index.md"

#### Scenario: Re-run — specs index already exists
- **GIVEN** `fab/specs/index.md` already exists
- **WHEN** `/fab-init` runs
- **THEN** the file is not overwritten
- **AND** the output includes "specs/index.md already exists — skipping"

## Specs Index: Boilerplate Content

### Requirement: Specs Index Boilerplate SHALL Distinguish Specs from Docs

The `fab/specs/index.md` boilerplate MUST begin with a clear explanation that specs are *pre-implementation* artifacts — what you planned, the conceptual design intent. It MUST note that specs are human-curated, flat in structure, and size-controlled. It SHOULD reference `fab/docs/index.md` as the complementary post-implementation layer.

#### Scenario: New user reads specs index
- **GIVEN** a freshly initialized project
- **WHEN** a user or agent reads `fab/specs/index.md`
- **THEN** the header clearly states specs are pre-implementation / planning artifacts
- **AND** the header contrasts specs with docs (post-implementation truth)
- **AND** the body contains an empty table for spec entries

### Requirement: Specs Index Structure SHALL Be Flat

The `fab/specs/index.md` MUST NOT prescribe a domain-based directory hierarchy. Specs MAY be organized by the human in any structure they choose. The index simply lists what exists.

#### Scenario: Human adds a spec file
- **GIVEN** `fab/specs/index.md` exists with the boilerplate
- **WHEN** a human creates a new spec file (e.g., `fab/specs/auth-design.md`)
- **THEN** the human manually adds a row to the index table
- **AND** no automated tooling enforces directory structure

## Docs Index: Updated Boilerplate

### Requirement: Docs Index Boilerplate SHALL Clarify Post-Implementation Role

The `fab/docs/index.md` boilerplate MUST include a header that clearly states docs are *post-implementation* artifacts — what happened, the authoritative source of truth for system behavior. It SHOULD reference `fab/specs/index.md` as the complementary pre-implementation layer.

#### Scenario: Existing docs index updated with distinction header
- **GIVEN** `fab/docs/index.md` exists (current boilerplate has a one-line description)
- **WHEN** this change is applied
- **THEN** the top of `fab/docs/index.md` includes a clear description distinguishing docs from specs
- **AND** existing table content is preserved

## Context Loading: Include Specs Index

### Requirement: `fab/specs/index.md` SHALL Be in the Always Load Layer

`fab/.kit/skills/_context.md` SHALL list `fab/specs/index.md` as a fourth file in the "Always Load" context layer, alongside `config.yaml`, `constitution.md`, and `fab/docs/index.md`.

#### Scenario: Agent loads context for any skill
- **GIVEN** a skill triggers the Always Load context layer
- **WHEN** the agent reads the context preamble
- **THEN** it reads `fab/specs/index.md` in addition to the existing three files
- **AND** the agent is aware of the specs landscape before generating artifacts

## Fab-Init Skill: Updated Step Ordering

### Requirement: Specs Index Step SHALL Follow Docs Index Step

In `fab/.kit/skills/fab-init.md`, the new step for creating `fab/specs/index.md` SHALL be inserted immediately after the existing step 1c (docs/index.md). It SHOULD be labeled step 1d, with subsequent steps re-lettered accordingly.

#### Scenario: Init skill reads linearly
- **GIVEN** a developer reads `fab-init.md`
- **WHEN** they reach the structural bootstrap section
- **THEN** step 1c creates `fab/docs/index.md`
- **AND** step 1d creates `fab/specs/index.md`
- **AND** step 1e creates `fab/changes/`

## Deprecated Requirements

(none)
