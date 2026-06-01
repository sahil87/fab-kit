---
name: _generation
description: "Artifact generation procedures — shared logic for intake and plan generation used by fab-continue and fab-ff."
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

> **Generation rule**: The intake is a state transfer document — the downstream apply-entry agent
> (which co-generates `plan.md`) has NO shared context beyond this file and the always-loaded
> config/constitution/memory. Every section must contain enough concrete detail (examples, code blocks,
> specific values, exact behavior descriptions) for an agent with no conversation history to generate a
> complete plan (requirements + tasks + acceptance). If a design decision was discussed with specific
> values — include them verbatim. Do not summarize or abstract.

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

## Plan Generation Procedure

> Merges requirement generation (formerly the standalone Spec Generation Procedure / `spec.md`),
> tasks generation, and acceptance generation into a single walk. The procedure derives
> `## Requirements` from the intake, then — in the same pass — emits an imperative Task entry and a
> declarative Acceptance entry per requirement. **One skill call, one context window** co-generating
> all three sections is the strongest alignment guarantee: the same agent that writes a requirement
> immediately consumes it. No mid-change ID rewrites: newly generated plans use `R#` for requirements,
> `T{NNN}` for tasks, and `A-{NNN}` for acceptance items; in-flight migrations preserve legacy
> `CHK-NNN` IDs verbatim (handled by the migration, not this procedure).

> **Invocation**: This procedure is invoked from `/fab-continue` Apply Behavior at apply
> entry, before any task is executed. It is not a planning-stage step. There is no `spec` stage and
> no separate `spec.md` artifact — the canonical artifact flow is `intake.md → plan.md → code`.

1. Read the template from `$(fab kit-path)/templates/plan.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: From the intake (the human-readable name)
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - Keep the `Intake` link pointing at `intake.md`
3. **Generate `## Requirements` from the intake-derived design** (absorbs the former Spec
   Generation Procedure):
   - For each domain/topic affected by this change, create a `### {Domain}: {Topic}` section with
     RFC 2119 requirement statements (MUST, SHALL, SHOULD, MAY), each with a stable `R#` ID and at
     least one GIVEN/WHEN/THEN scenario.
   - Include a `### Non-Goals` subsection (optional) for meaningful scope exclusions; a
     `### Design Decisions` subsection (optional) for architectural choices (summary + rationale +
     rejected alternatives); and a `### Deprecated Requirements` subsection if the change removes
     existing requirements.
   - **No `[NEEDS CLARIFICATION]` markers.** Those are an intake-only construct (a human still needs
     to decide). An under-specified requirement encountered here is resolved inline as a graded SRAD
     assumption (Certain/Confident/Tentative) recorded in the plan's `## Assumptions` section — not
     as a marker. (Apply does not *clarify*; it *decides and records*.)
   - **Legacy `spec.md` ingestion (one-release back-compat)**: if a leftover `spec.md` exists in the
     change folder AND `plan.md` does not yet have a `## Requirements` section, fold the spec.md
     requirement body (domain sections, scenarios, Non-Goals/Design Decisions/Deprecated
     Requirements) into `## Requirements` instead of regenerating from scratch. Annotate the section
     `<!-- migrated from spec.md -->`. Do not move spec.md's `## Assumptions` table.
4. **Walk the `## Requirements` just generated.** For each requirement, emit two entries — a Task and
   an Acceptance — paired by the same logical work item:
   - The Task entry goes under `## Tasks` and describes *what to implement, in which file*
   - The Acceptance entry goes under `## Acceptance` and describes *what must be true for
     review to pass* (a declarative outcome, not a step)
   - **Traceability is REQUIRED** (not optional): each `## Tasks` item MUST carry a `<!-- R# -->`
     trace annotation naming the requirement it implements, and each `## Acceptance` item MUST name
     the requirement it accepts (e.g., `A-001 R2: {outcome}`). The chain
     `R# → T# → test → A#` is what lets the autonomous (apply ↔ review) loop localize a failing
     acceptance item back to its requirement and converge.
5. **Tasks subsection** (`## Tasks`):
   - Group by phase. Phases execute sequentially; within a phase, `[P]`-marked tasks may run
     in parallel:
     - **Phase 1: Setup** — scaffolding, dependencies, configuration
     - **Phase 2: Core Implementation** — primary functionality, ordered by dependency
     - **Phase 3: Integration & Edge Cases** — wiring, error states, validation
     - **Phase 4: Polish** — documentation, cleanup (only if warranted)
   - Each task follows the format: `- [ ] T{NNN} [{markers}] {description with file paths} <!-- R# -->`
   - IDs are sequential, three-digit: T001, T002, ...
   - Mark parallelizable tasks with `[P]`
   - Include exact file paths in descriptions
   - Each task should be completable in one focused session
   - Each task MUST carry a `<!-- R# -->` trace annotation naming the requirement it implements
   - Include an `## Execution Order` section after `## Tasks` only for non-obvious
     dependencies between tasks; omit when ordering is self-evident
6. **Acceptance subsection** (`## Acceptance`):
   - Populate items derived from:
     - `## Requirements` — every requirement gets at least one item under **Functional Completeness**
     - Changed requirements → **Behavioral Correctness** items
     - Deprecated requirements → **Removal Verification** items
     - Key scenarios from `## Requirements` → **Scenario Coverage** items
     - Edge cases identified in `## Requirements` → **Edge Cases & Error Handling** items
     - `fab/project/code-quality.md` → **Code Quality** items. If
       `fab/project/code-quality.md` exists: one item per relevant principle from
       `## Principles`, one per relevant anti-pattern from `## Anti-Patterns` that applies to
       the change's scope, plus the two baseline items. If no `fab/project/code-quality.md`:
       include the two baseline items only (pattern consistency, no unnecessary duplication)
     - Security-relevant changes → **Security** items (only if applicable)
     - Additional categories from `fab/project/config.yaml` `checklist.extra_categories` (if
       any)
   - Each item follows the format: `- [ ] A-{NNN} R#: {declarative outcome}` — naming the
     requirement it accepts (REQUIRED)
   - IDs are sequential, three-digit, zero-padded: A-001, A-002, ...
7. Write the completed plan to `fab/changes/{name}/plan.md`. The PostToolUse hook updates
   `.status.yaml` `plan.generated`, `plan.task_count`, `plan.acceptance_count`, and
   `plan.acceptance_completed` automatically; no manual `fab status set-acceptance` calls
   are required at generation time. Skills that wish to assert the counts explicitly MAY
   call `fab status set-acceptance <change> <field> <value>` (valid fields: `generated`,
   `task_count`, `acceptance_count`, `acceptance_completed`).
