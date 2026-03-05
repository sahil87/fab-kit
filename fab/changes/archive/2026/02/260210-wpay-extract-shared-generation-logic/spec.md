# Spec: Extract shared generation logic from fab-continue and fab-ff

**Change**: 260210-wpay-extract-shared-generation-logic
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/planning-skills.md`

## Skill Architecture: Shared Generation Partial

### Requirement: Generation logic SHALL be extracted into a single partial

The artifact generation logic for `spec.md`, `plan.md`, `tasks.md`, and `checklists/quality.md` SHALL be defined in a single `_generation.md` partial file at `fab/.kit/skills/_generation.md`. Both `fab-continue.md` and `fab-ff.md` SHALL reference this partial instead of inlining the generation steps.

#### Scenario: Agent reads fab-continue and encounters generation reference
- **GIVEN** an agent is executing `/fab-continue` and reaches the spec/plan/tasks generation step
- **WHEN** the agent reads the generation instructions in `fab-continue.md`
- **THEN** the instructions SHALL direct the agent to follow the generation procedures defined in `_generation.md`
- **AND** the agent SHALL load `_generation.md` and execute the referenced generation procedure

#### Scenario: Agent reads fab-ff and encounters generation reference
- **GIVEN** an agent is executing `/fab-ff` and reaches the spec/plan/tasks generation step
- **WHEN** the agent reads the generation instructions in `fab-ff.md`
- **THEN** the instructions SHALL direct the agent to follow the generation procedures defined in `_generation.md`
- **AND** the agent SHALL load `_generation.md` and execute the referenced generation procedure

### Requirement: The partial SHALL cover spec, plan, tasks, and checklist generation

`_generation.md` SHALL contain four generation procedures:

1. **Spec generation** — template loading, metadata fill, requirement sections with RFC 2119 keywords, GIVEN/WHEN/THEN scenarios, deprecated requirements, `[NEEDS CLARIFICATION]` markers, Assumptions section
2. **Plan generation** — template loading, metadata fill, summary, goals/non-goals, technical context, research, decisions, risks/trade-offs, file changes, Assumptions section
3. **Tasks generation** — template loading, metadata fill, phased task breakdown (Phase 1-4), task format (`- [ ] T{NNN} [{markers}] {description}`), execution order
4. **Checklist generation** — template loading, directory creation, category population from spec/plan, sequential CHK IDs, `.status.yaml` checklist field updates

#### Scenario: All four generation procedures are present
- **GIVEN** a reader opens `_generation.md`
- **WHEN** they search for each generation procedure
- **THEN** they SHALL find sections for spec generation, plan generation, tasks generation, and checklist generation
- **AND** each section SHALL be self-contained with all steps needed to produce the artifact

### Requirement: Each skill SHALL retain its own orchestration logic

`fab-continue.md` SHALL retain: stage guards, stage determination, SRAD question selection (per-stage, max 3), plan decision with user confirmation, confidence recomputation, reset flow, and `.status.yaml` stage transitions.

`fab-ff.md` SHALL retain: frontloaded question batching, autonomous plan decision, auto-clarify interleaving, bail logic, resumability, and `.status.yaml` stage transitions.

#### Scenario: fab-continue retains plan confirmation prompt
- **GIVEN** an agent is executing `/fab-continue` at the specs→plan transition
- **WHEN** the agent evaluates whether a plan is warranted
- **THEN** the agent SHALL follow the plan decision logic defined in `fab-continue.md` (confirm with user)
- **AND** the plan decision logic SHALL NOT be in `_generation.md`

#### Scenario: fab-ff retains autonomous plan decision
- **GIVEN** an agent is executing `/fab-ff` at the specs→plan transition
- **WHEN** the agent evaluates whether a plan is warranted
- **THEN** the agent SHALL follow the autonomous plan decision logic defined in `fab-ff.md` (no user prompt)
- **AND** the plan decision logic SHALL NOT be in `_generation.md`

### Requirement: The partial SHALL use the underscore-prefix naming convention

The file SHALL be named `_generation.md` with an underscore prefix, consistent with the existing `_context.md` partial in `fab/.kit/skills/`.

#### Scenario: File naming consistency
- **GIVEN** a developer inspects `fab/.kit/skills/`
- **WHEN** they list files starting with underscore
- **THEN** they SHALL see both `_context.md` and `_generation.md`
- **AND** both files SHALL be non-skill partials referenced by other skill files

### Requirement: Generation procedures SHALL be content-identical to current inline versions

The extracted generation procedures SHALL produce artifacts identical to what the current inline versions in `fab-continue.md` and `fab-ff.md` produce. No behavioral changes SHALL be introduced by the extraction — this is a pure refactor.

#### Scenario: Spec generation output is unchanged
- **GIVEN** an agent generates a spec using the extracted `_generation.md` procedure
- **WHEN** compared to a spec generated by the previous inline procedure
- **THEN** the output SHALL be structurally and semantically identical
- **AND** the same template, metadata fields, and section structure SHALL be used

## Deprecated Requirements

None — this change does not remove any existing requirements.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use `_generation.md` naming with underscore prefix | Consistent with existing `_context.md` partial pattern in the same directory |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
