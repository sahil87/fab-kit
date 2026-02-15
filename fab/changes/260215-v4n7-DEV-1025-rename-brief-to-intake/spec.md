# Spec: Rename "brief" to "intake" + Add Intake Generation Rule

**Change**: 260215-v4n7-DEV-1025-rename-brief-to-intake
**Created**: 2026-02-15
**Affected memory**:
- `docs/memory/fab-workflow/planning-skills.md` (modify)
- `docs/memory/fab-workflow/templates.md` (modify)
- `docs/memory/fab-workflow/change-lifecycle.md` (modify)
- `docs/memory/fab-workflow/configuration.md` (modify)
- `docs/memory/fab-workflow/kit-architecture.md` (modify)
- `docs/memory/fab-workflow/clarify.md` (modify)
- `docs/memory/fab-workflow/context-loading.md` (modify)
- `docs/memory/fab-workflow/execution-skills.md` (modify)
- `docs/memory/fab-workflow/index.md` (modify)

## Non-Goals

- Renaming existing `brief.md` files in completed or in-progress change folders — these are historical artifacts
- Renaming the `/fab-new` skill — it creates changes, not just intakes
- Changing pipeline stage behavior, ordering, or state machine logic — this is a naming and documentation change only
- Modifying `lib/stageman.sh` or `lib/preflight.sh` script logic — these derive stage names from `.status.yaml` and `config.yaml`, which are updated as part of the rename

## Pipeline Stage Naming

### Requirement: Stage Identifier Rename

All pipeline references SHALL use `intake` instead of `brief` as the first-stage identifier. This is a systematic token replacement across the entire Fab kit: skill files, templates, schemas, config, and documentation.

#### Scenario: Status YAML uses intake stage key
- **GIVEN** a new change is created via `/fab-new`
- **WHEN** `.status.yaml` is initialized from the template
- **THEN** the progress map SHALL contain `intake: active` (not `brief: active`)
- **AND** all downstream stage references (e.g., `requires: [intake]`) SHALL use the new name

#### Scenario: Config defines intake stage
- **GIVEN** `fab/config.yaml` contains the stages list
- **WHEN** the first stage entry is read
- **THEN** it SHALL have `id: intake`, `generates: intake.md`
- **AND** the spec stage SHALL have `requires: [intake]`

#### Scenario: Schema validates intake stage
- **GIVEN** `fab/.kit/schemas/workflow.yaml` defines valid stage identifiers
- **WHEN** the stage enum is evaluated
- **THEN** `intake` SHALL be listed as the first stage
- **AND** `brief` SHALL NOT appear in the stage enum

### Requirement: Skill File Reference Updates

All skill files in `fab/.kit/skills/` that reference the `brief` stage or `brief.md` artifact SHALL be updated to use `intake` and `intake.md` respectively. The following substitution patterns SHALL be applied context-sensitively:

| Pattern | Replacement | Context |
|---------|-------------|---------|
| `brief` (as stage name) | `intake` | Stage dispatch tables, progress keys, transition calls, stage guards |
| `brief.md` (as artifact name) | `intake.md` | File loading, generation targets, cross-references |
| `brief:` (in YAML-like contexts) | `intake:` | Status references, progress maps |
| `brief` (in prose/comments) | `intake` | Descriptions, instructions, section headings |

#### Scenario: fab-continue dispatches on intake stage
- **GIVEN** the active stage in `.status.yaml` is `intake`
- **WHEN** `/fab-continue` runs
- **THEN** it SHALL dispatch to spec generation (same behavior as the former `brief` stage)
- **AND** the transition SHALL be `intake: done`, `spec: active`

#### Scenario: fab-clarify scans intake artifact
- **GIVEN** the current stage is `spec`
- **WHEN** `/fab-clarify` performs its taxonomy scan
- **THEN** it SHALL scan `intake.md` (not `brief.md`) for gaps in scope boundaries, affected areas, and Origin completeness

#### Scenario: fab-new generates intake artifact
- **GIVEN** a user runs `/fab-new "some description"`
- **WHEN** the change folder is initialized
- **THEN** `.status.yaml` SHALL have `intake: active`
- **AND** the generated artifact SHALL be named `intake.md`

#### Scenario: fab-switch displays intake stage
- **GIVEN** a change is at the intake stage
- **WHEN** `/fab-switch` displays the stage summary
- **THEN** it SHALL show `Stage: intake (1/6)`

#### Scenario: _context.md references intake for memory lookup
- **GIVEN** a skill loads change context per `_context.md` §3 (Memory File Lookup)
- **WHEN** the skill reads the affected memory section
- **THEN** the instructions SHALL reference "the intake's Affected Memory section" (not "the brief's Affected Memory section")

### Requirement: Affected Skill Files

The following skill files SHALL be updated (non-exhaustive list of key files — all `brief` references in `fab/.kit/skills/` are in scope):

- `fab-new.md` — artifact generation target, Step 5 references, status initialization
- `fab-continue.md` — stage dispatch table, context loading, transition calls
- `fab-ff.md`, `fab-fff.md` — stage references, pipeline flow descriptions
- `fab-clarify.md` — taxonomy scan artifact references, stage guard
- `fab-init.md` — valid sections, bootstrap output
- `fab-archive.md` — artifact references
- `fab-switch.md` — stage display, artifact loading
- `fab-status.md` — stage display
- `fab-help.md` — skill descriptions mentioning brief
- `_context.md` — context loading layers, memory file lookup from intake's Affected Memory
- `_generation.md` — spec generation reading intake, tasks generation referencing intake

#### Scenario: All skill files updated consistently
- **GIVEN** a full-text search for `brief` in `fab/.kit/skills/`
- **WHEN** the search excludes the word "brief" used in its English adjective sense (e.g., "brief reason", "brief description")
- **THEN** zero results SHALL remain that refer to the former stage or artifact name

### Requirement: Documentation Updates

All documentation files in `docs/specs/` and `docs/memory/fab-workflow/` that reference the `brief` stage or `brief.md` artifact SHALL be updated to use `intake` and `intake.md`.

#### Scenario: Specs updated
- **GIVEN** the files `overview.md`, `architecture.md`, `skills.md`, `templates.md`, `user-flow.md`, `glossary.md`, `srad.md`
- **WHEN** each file is updated
- **THEN** all references to the `brief` stage and `brief.md` artifact SHALL use `intake` and `intake.md`
- **AND** the English adjective "brief" (meaning short/concise) SHALL NOT be affected

#### Scenario: Memory files updated
- **GIVEN** the memory files listed in Affected Memory
- **WHEN** each file is updated
- **THEN** all references to the `brief` stage and `brief.md` artifact SHALL use `intake` and `intake.md`
- **AND** changelog entries for this change SHALL be appended to each modified file

### Requirement: Historical Artifact Preservation

Existing change folders in `fab/changes/` (including `archive/`) SHALL NOT have their `brief.md` files renamed. These are historical artifacts tied to completed pipeline runs.

#### Scenario: Existing change folder untouched
- **GIVEN** `fab/changes/260212-1aag-DEV-1018-build-stageman2/` contains `brief.md`
- **WHEN** this rename change is applied
- **THEN** the file SHALL remain as `brief.md`
- **AND** no preflight or status errors SHALL occur when operating on historical changes

## Artifact Generation

### Requirement: Intake Generation Procedure in _generation.md

`fab/.kit/skills/_generation.md` SHALL include a new `## Intake Generation Procedure` section. This section SHALL be placed before the existing Spec Generation Procedure (since intake precedes spec in the pipeline).

The procedure SHALL include:

1. **A generation rule** emphasizing the intake is a state transfer document — downstream agents have no shared context beyond this file and the always-loaded config/constitution/memory
2. **Template loading** — read from `fab/.kit/templates/intake.md`
3. **Metadata filling** — `{CHANGE_NAME}`, `{YYMMDD-XXXX-slug}`, `{DATE}`
4. **Section-by-section guidance** — substantive content required for every section; "What Changes" as the most detailed section; include concrete examples, code blocks, specific file paths
5. **Assumptions section** — append per `_context.md` SRAD framework
6. **Output path** — write to `fab/changes/{name}/intake.md`

#### Scenario: Generation rule prevents thin artifacts
- **GIVEN** `_generation.md` contains the Intake Generation Procedure
- **WHEN** an agent reads the generation rule
- **THEN** it SHALL encounter explicit instruction that the intake is a "state transfer document" requiring concrete detail
- **AND** the instruction SHALL state that summarization or abstraction is prohibited

#### Scenario: fab-new references the procedure
- **GIVEN** `/fab-new` reaches Step 5 (artifact generation)
- **WHEN** it needs to generate the intake
- **THEN** it SHALL reference "the **Intake Generation Procedure** (`_generation.md`)"
- **AND** it SHALL NOT inline generation logic that duplicates the procedure

### Requirement: Spec Generation Procedure Update

The Spec Generation Procedure in `_generation.md` SHALL update its reference from `brief.md` to `intake.md` (Step 2 metadata reading, Step 6 assumptions starting point).

#### Scenario: Spec procedure reads intake
- **GIVEN** the Spec Generation Procedure in `_generation.md`
- **WHEN** it references the source artifact for metadata and assumptions
- **THEN** it SHALL reference `intake.md` (not `brief.md`)

### Requirement: Tasks Generation Procedure Update

The Tasks Generation Procedure in `_generation.md` SHALL update its `brief.md` reference to `intake.md` (Step 2 traceability reference).

#### Scenario: Tasks procedure references intake
- **GIVEN** the Tasks Generation Procedure in `_generation.md`
- **WHEN** it includes a traceability reference
- **THEN** it SHALL reference `intake.md` (not `brief.md`)

## Template Design

### Requirement: Template File Rename

The template file SHALL be renamed from `fab/.kit/templates/brief.md` to `fab/.kit/templates/intake.md`.

#### Scenario: Template file exists at new path
- **GIVEN** `fab/.kit/templates/`
- **WHEN** listing files
- **THEN** `intake.md` SHALL exist
- **AND** `brief.md` SHALL NOT exist

### Requirement: Strengthened Template Comments

The renamed `intake.md` template SHALL include strengthened structural cues in its HTML comments to reinforce detail expectations. Specifically:

- The **"What Changes"** section comment SHALL state: "This section is the primary input for spec generation — if a design decision was made with specific values, include them here."
- The **"What Changes"** section comment SHALL instruct: "Use subsections (### per change area) for multi-part changes. Include concrete examples: code blocks, config snippets, exact behavior."
- The **"Origin"** section comment SHALL instruct: provide traceability for how the change was initiated, not just a one-liner
- The **"Why"** section comment SHALL instruct: include the problem, the consequence of not fixing it, and why this approach

#### Scenario: Agent reads strengthened template
- **GIVEN** an agent loads `fab/.kit/templates/intake.md` during `/fab-new`
- **WHEN** it reads the "What Changes" section comment
- **THEN** the comment SHALL contain explicit guidance about subsections, concrete examples, and being the primary input for spec generation

## Design Decisions

1. **"Intake" over alternatives**: `intake` was chosen over `handoff`, `charter`, `brief-detail`, and other candidates.
   - *Why*: In legal, medical, and project management contexts, "intake" means "thorough initial collection of all relevant information." It signals start-of-pipeline without implying brevity. The English word "brief" actively triggers summarization instincts in LLMs.
   - *Rejected*: `handoff` — could describe any stage boundary, not specifically the first. `charter` — too formal, implies governance. `brief-detail` — contradictory, still contains "brief."

2. **Generation rule in `_generation.md`, not `config.yaml` rules**: The intake generation procedure goes in the shared generation partial alongside spec/tasks/checklist procedures.
   - *Why*: Matches the established pattern — spec, tasks, and checklist all have their procedures in `_generation.md`. Config rules are per-stage behavioral constraints, not full generation procedures.
   - *Rejected*: Adding a rule string to `config.yaml rules.intake` — too limited for a full generation procedure with template loading, section guidance, and output paths.

3. **Defense in depth: rename + generation rule**: The rename fixes the naming instinct; the generation rule catches agents that still compress despite good naming.
   - *Why*: Neither fix alone is sufficient. The name sets expectations; the rule enforces them. Template comments provide a third layer of structural cues.
   - *Rejected*: Rename only — some agents may still compress from habit. Rule only — doesn't fix the misleading name.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Stage rename is `brief` → `intake` | Explicitly chosen — "intake" signals thorough collection (confirmed from brief #1) | S:95 R:80 A:95 D:95 |
| 2 | Certain | Generation rule goes in `_generation.md` as Intake Generation Procedure | Matches existing pattern for spec/tasks/checklist procedures (confirmed from brief #2) | S:95 R:85 A:90 D:95 |
| 3 | Certain | Existing change folders keep their `brief.md` files | Historical artifacts — renaming completed changes would be destructive (confirmed from brief #3) | S:90 R:95 A:90 D:95 |
| 4 | Certain | `/fab-new` skill name stays unchanged | The skill creates changes, not just intakes (confirmed from brief #4) | S:85 R:90 A:90 D:90 |
| 5 | Certain | This change is ordered before the code quality layer | Different concerns, explicitly discussed (confirmed from brief #5) | S:95 R:85 A:95 D:95 |
| 6 | Certain | `/fab-new` Step 5 references `_generation.md` procedure | Single source of truth pattern — same as spec/tasks/checklist (upgraded from brief #6 Confident: pattern is clearly established) | S:90 R:85 A:90 D:90 |
| 7 | Certain | Template comments strengthened alongside the rename | Rename fixes naming; template comments reinforce expected detail level. Defense in depth (upgraded from brief #7 Confident: purpose is clear and well-defined) | S:85 R:90 A:85 D:85 |
| 8 | Certain | English adjective "brief" (meaning short/concise) is not affected by the rename | Context-sensitive substitution — only the stage/artifact token is renamed, not the English word (S:90 — obvious distinction, R:95 — easy to fix if over-applied) | S:90 R:95 A:90 D:95 |
| 9 | Certain | Intake Generation Procedure placed before Spec Generation Procedure | Pipeline order: intake precedes spec, so the procedure section mirrors that order (S:85 — follows convention, R:95 — section order is trivially adjustable) | S:85 R:95 A:90 D:90 |

9 assumptions (9 certain, 0 confident, 0 tentative, 0 unresolved).
