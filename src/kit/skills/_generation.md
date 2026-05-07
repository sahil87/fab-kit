---
name: _generation
description: "Artifact generation procedures — shared logic for intake, spec, and plan generation used by fab-continue and fab-ff."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Artifact Generation Procedures

> This file defines the shared artifact generation logic used by both `/fab-continue` and `/fab-ff`.
> Each skill references these procedures instead of inlining them, ensuring generation behavior
> is authoritative in one location.
>
> **Orchestration** (stage guards, question handling, design decisions, auto-clarify, resumability)
> remains in each skill's own file. This partial covers only the mechanics of producing each artifact.

---

## Intake Generation Procedure

> **Generation rule**: The intake is a state transfer document — downstream agents (spec, plan)
> have NO shared context beyond this file and the always-loaded config/constitution/memory. Every section
> must contain enough concrete detail (examples, code blocks, specific values, exact behavior descriptions)
> for an agent with no conversation history to generate a complete spec. If a design decision was discussed
> with specific values — include them verbatim. Do not summarize or abstract.

1. Read the template from `$(fab kit-path)/templates/intake.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: Human-readable name from the description
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - `{DATE}`: Today's date
3. For each section (Origin, Why, What Changes, Affected Memory, Impact, Open Questions):
   - Write substantively — no placeholder text, no single-sentence descriptions
   - Include concrete examples: code blocks, YAML snippets, specific file paths, exact behavior
   - The "What Changes" section should be the most detailed — use subsections per change area
   - If a design includes specific values (config structure, template content, validation questions), reproduce them in full
4. Append `## Assumptions` section per `_preamble.md` SRAD framework
5. Write the completed intake to `fab/changes/{name}/intake.md`

---

## Spec Generation Procedure

1. Read the template from `$(fab kit-path)/templates/spec.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: The human-readable name from the intake
   - `{YYMMDD-XXXX-slug}`: The change folder name from `.status.yaml`
   - `{DATE}`: Today's date
   - `{domain}` and `{file-name}`: From the intake's Affected Memory section
2b. **Non-Goals** (optional): If the change has meaningful scope exclusions, include a `## Non-Goals` section after the metadata block and before the first domain section. Each non-goal is a bullet: `- {what is excluded} — {brief reason, if not obvious}`. Derive non-goals from the intake's scope boundaries and any explicit exclusions discussed. Omit this section entirely for straightforward changes with no meaningful exclusions.
3. For each domain/topic affected by this change, create a section with:
   - Requirements using RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
   - At least one GIVEN/WHEN/THEN scenario per requirement
4. Include a **Deprecated Requirements** section if the change removes existing requirements
5. Mark any unresolved ambiguities with `[NEEDS CLARIFICATION]` inline
5b. **Design Decisions** (optional): If the change involves architectural choices, technology selection, or non-obvious approaches, include a `## Design Decisions` section after the domain requirement sections. Each decision entry SHALL include: decision summary, rationale (why this choice), and rejected alternatives. Omit this section for straightforward changes.
6. Append an `## Assumptions` section. Read `intake.md`'s `## Assumptions` table as the starting point — confirm, upgrade, or override each intake assumption based on spec-level analysis (note the action in Rationale, e.g., "Confirmed from intake #1", "Upgraded from intake Tentative"). Add new assumptions discovered during spec generation. Include all four SRAD grades (Certain, Confident, Tentative, Unresolved) with required Scores column. The spec's Assumptions table is the sole scoring source for `fab score` (see Assumptions Summary Block in `_preamble.md`)
7. Write the completed spec to `fab/changes/{name}/spec.md`

---

## Plan Generation Procedure

> Replaces the legacy split between Tasks Generation and Checklist Generation. The unified
> procedure walks `spec.md` once and emits both an imperative Task entry and a declarative
> Acceptance entry per requirement. Single skill call, single context window — that is the
> alignment guarantee. No mid-change ID rewrites: newly generated plans use `A-NNN` for
> acceptance items; in-flight migrations preserve legacy `CHK-NNN` IDs verbatim (handled by
> the migration, not this procedure).

> **Invocation**: This procedure is invoked from `/fab-continue` Apply Behavior at apply
> entry, before any task is executed. It is not a planning-stage step.

1. Read the template from `$(fab kit-path)/templates/plan.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: From the intake (the human-readable name)
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - Keep the `Intake` and `Spec` links pointing at `intake.md` and `spec.md`
3. **Walk `spec.md` requirements once.** For each requirement, emit two entries — a Task and
   an Acceptance — paired by the same logical work item:
   - The Task entry goes under `## Tasks` and describes *what to implement, in which file*
   - The Acceptance entry goes under `## Acceptance` and describes *what must be true for
     review to pass* (a declarative outcome, not a step)
   - Cross-linking via shared IDs is OPTIONAL — readers who want it MAY annotate entries
     (e.g., `T001 ... <!-- A-001 -->`); the co-generation invariant is the alignment
     contract, not the IDs.
4. **Tasks subsection** (`## Tasks`):
   - Group by phase. Phases execute sequentially; within a phase, `[P]`-marked tasks may run
     in parallel:
     - **Phase 1: Setup** — scaffolding, dependencies, configuration
     - **Phase 2: Core Implementation** — primary functionality, ordered by dependency
     - **Phase 3: Integration & Edge Cases** — wiring, error states, validation
     - **Phase 4: Polish** — documentation, cleanup (only if warranted)
   - Each task follows the format: `- [ ] T{NNN} [{markers}] {description with file paths}`
   - IDs are sequential, three-digit: T001, T002, ...
   - Mark parallelizable tasks with `[P]`
   - Include exact file paths in descriptions
   - Each task should be completable in one focused session
   - Include an `## Execution Order` section after `## Tasks` only for non-obvious
     dependencies between tasks; omit when ordering is self-evident
5. **Acceptance subsection** (`## Acceptance`):
   - Populate items derived from:
     - `spec.md` — every requirement gets at least one item under **Functional Completeness**
     - Changed requirements → **Behavioral Correctness** items
     - Deprecated requirements → **Removal Verification** items
     - Key scenarios from spec → **Scenario Coverage** items
     - Edge cases identified in spec → **Edge Cases & Error Handling** items
     - `fab/project/code-quality.md` → **Code Quality** items. If
       `fab/project/code-quality.md` exists: one item per relevant principle from
       `## Principles`, one per relevant anti-pattern from `## Anti-Patterns` that applies to
       the change's scope, plus the two baseline items. If no `fab/project/code-quality.md`:
       include the two baseline items only (pattern consistency, no unnecessary duplication)
     - Security-relevant changes → **Security** items (only if applicable)
     - Additional categories from `fab/project/config.yaml` `checklist.extra_categories` (if
       any)
   - Each item follows the format: `- [ ] A-{NNN} {declarative outcome}`
   - IDs are sequential, three-digit, zero-padded: A-001, A-002, ...
6. Write the completed plan to `fab/changes/{name}/plan.md`. The PostToolUse hook updates
   `.status.yaml` `plan.generated`, `plan.task_count`, `plan.acceptance_count`, and
   `plan.acceptance_completed` automatically; no manual `fab status set-acceptance` calls
   are required at generation time. Skills that wish to assert the counts explicitly MAY
   call `fab status set-acceptance <change> <field> <value>` (valid fields: `generated`,
   `task_count`, `acceptance_count`, `acceptance_completed`).
