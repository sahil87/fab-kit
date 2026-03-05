# Spec: Fix consistency drift between design, docs, and implementation

**Change**: 260212-k7m3-fix-consistency-drift
**Created**: 2026-02-12
**Affected docs**: `fab/docs/fab-workflow/index.md`, `fab/docs/fab-workflow/planning-skills.md`

## Implementation: Template and Terminology Fixes

### Requirement: Brief Template Header
The `fab/.kit/templates/brief.md` header SHALL read `# Brief: {CHANGE_NAME}`, replacing the current `# Proposal: {CHANGE_NAME}`.

#### Scenario: New brief generated from template
- **GIVEN** the template `fab/.kit/templates/brief.md` exists
- **WHEN** an agent reads the header line
- **THEN** it reads `# Brief: {CHANGE_NAME}`

### Requirement: DEFERRED Timing in Brief Template
The `fab/.kit/templates/brief.md` Open Questions comment SHALL state `[DEFERRED] can resolve during spec`, replacing the current `before tasks` wording, aligning with the design spec and glossary.

#### Scenario: DEFERRED guidance matches design and glossary
- **GIVEN** the template `fab/.kit/templates/brief.md` exists
- **WHEN** an agent reads the Open Questions comment
- **THEN** the `[DEFERRED]` guidance reads "can resolve during spec"
- **AND** the `[BLOCKING]` guidance reads "must resolve before spec" (unchanged)

### Requirement: Brief References in _generation.md
The `fab/.kit/skills/_generation.md` SHALL use "brief" wherever it currently says "proposal" when referencing the brief artifact. Specifically:
- Spec Generation step 2: `{CHANGE_NAME}` description → "from the brief"
- Tasks Generation step 2: `{CHANGE_NAME}` description → "from the brief"

#### Scenario: Spec generation references brief
- **GIVEN** `/fab-continue` or `/fab-ff` is generating `spec.md`
- **WHEN** it reads the Spec Generation Procedure step 2
- **THEN** the `{CHANGE_NAME}` field description reads "The human-readable name from the brief"

#### Scenario: Tasks generation references brief
- **GIVEN** `/fab-continue` or `/fab-ff` is generating `tasks.md`
- **WHEN** it reads the Tasks Generation Procedure step 2
- **THEN** the `{CHANGE_NAME}` field description reads "From the brief"

## Design: Architecture Updates

### Requirement: Slug Word Count in Architecture
The `fab/design/architecture.md` folder naming table SHALL state the slug is "2-6 words from description", replacing "2-4 words".

#### Scenario: Architecture slug word count
- **GIVEN** a developer reads `fab/design/architecture.md`
- **WHEN** they check the slug component in the folder naming table
- **THEN** it reads "2-6 words from description"

### Requirement: Branch Integration Attribution
The `fab/design/architecture.md` Git Integration section SHALL attribute branch creation/adoption to `/fab-switch`, replacing all references to `/fab-new` as the branch-handling skill.

#### Scenario: Git integration references fab-switch
- **GIVEN** a developer reads `fab/design/architecture.md` Git Integration
- **WHEN** they check which skill handles Create/Adopt/Skip branch options
- **THEN** the section references `/fab-switch`
- **AND** `/fab-new` is described as delegating branch integration to `/fab-switch`

### Requirement: Remove stage Field from Architecture Examples
The `fab/design/architecture.md` .status.yaml examples SHALL NOT include a separate `stage:` field. The current stage is derived from the `progress` map's `active` entry, as already documented in `fab/design/templates.md`.

#### Scenario: Spec-stage status example
- **GIVEN** a developer reads the spec-stage .status.yaml example in architecture.md
- **WHEN** they check the YAML structure
- **THEN** there is no `stage: spec` line
- **AND** `progress.spec` is `active`

#### Scenario: Review-stage status example
- **GIVEN** a developer reads the review-stage .status.yaml example in architecture.md
- **WHEN** they check the YAML structure
- **THEN** there is no `stage: review` line
- **AND** `progress.review` is `active`

### Requirement: Add created_by to Architecture Examples
The `fab/design/architecture.md` .status.yaml examples SHALL include the `created_by` field after `created`, matching the template in `fab/design/templates.md`.

#### Scenario: Status examples include created_by
- **GIVEN** a developer reads any .status.yaml example in architecture.md
- **WHEN** they check the YAML fields
- **THEN** `created_by: {value}` appears between `created` and `branch`

## Design: Templates Updates

### Requirement: Origin Section in Brief Template
The `fab/design/templates.md` brief template SHALL include a `## Origin` section between the metadata block and `## Why`, matching the implementation template at `fab/.kit/templates/brief.md`.

#### Scenario: Design brief template includes Origin
- **GIVEN** a developer reads `fab/design/templates.md`
- **WHEN** they check the brief.md template structure
- **THEN** `## Origin` appears after the `**Status**: Draft` metadata line and before `## Why`
- **AND** the guidance comment describes preserving the user's raw input

### Requirement: Archive Index Maintenance Documentation
The `fab/design/templates.md` SHALL document that `/fab-archive` maintains an `index.md` inside `fab/changes/archive/` listing all completed changes with their date and summary.

#### Scenario: Design templates describe archive index
- **GIVEN** a developer reads `fab/design/templates.md`
- **WHEN** they look for archive index documentation
- **THEN** they find a description of the archive index maintenance step performed by `/fab-archive`

## Design: Glossary Update

### Requirement: Slug Word Count in Glossary
The `fab/design/glossary.md` folder name format entry SHALL state "2–6 word slug", replacing "2–4 word slug".

#### Scenario: Glossary slug word count
- **GIVEN** a developer reads `fab/design/glossary.md`
- **WHEN** they check the Folder name format definition
- **THEN** it reads "2–6 word slug"

## Centralized Docs: Fixes

### Requirement: Backfill Description Terminology
The `fab/docs/fab-workflow/index.md` backfill entry description SHALL read "structural gap detection between docs and design", replacing "docs and specs".

#### Scenario: Index backfill entry uses correct terminology
- **GIVEN** an agent reads `fab/docs/fab-workflow/index.md`
- **WHEN** it checks the backfill doc description
- **THEN** the description includes "between docs and design" (not "docs and specs")

### Requirement: Confidence Score Default Clarification
The `fab/docs/fab-workflow/planning-skills.md` SHALL clarify that the template confidence defaults have counts at zero and score at 5.0, replacing the ambiguous "all zeros" phrasing.

#### Scenario: Planning-skills docs describe defaults accurately
- **GIVEN** an agent reads `fab/docs/fab-workflow/planning-skills.md`
- **WHEN** it encounters the confidence score template defaults description
- **THEN** the text distinguishes between count defaults (zero) and score default (5.0)
- **AND** the phrase "all zeros" is replaced with accurate phrasing
