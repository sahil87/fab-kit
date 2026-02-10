# Artifact Generation Procedures

> This file defines the shared artifact generation logic used by both `/fab-continue` and `/fab-ff`.
> Each skill references these procedures instead of inlining them, ensuring generation behavior
> is authoritative in one location.
>
> **Orchestration** (stage guards, question handling, plan decisions, auto-clarify, resumability)
> remains in each skill's own file. This partial covers only the mechanics of producing each artifact.

---

## Spec Generation Procedure

1. Read the template from `fab/.kit/templates/spec.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: The human-readable name from the proposal
   - `{YYMMDD-XXXX-slug}`: The change folder name from `.status.yaml`
   - `{DATE}`: Today's date
   - `{domain}` and `{doc-name}`: From the proposal's Affected Docs section
3. For each domain/topic affected by this change, create a section with:
   - Requirements using RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
   - At least one GIVEN/WHEN/THEN scenario per requirement
4. Include a **Deprecated Requirements** section if the change removes existing requirements
5. Mark any unresolved ambiguities with `[NEEDS CLARIFICATION]` inline
6. Append an `## Assumptions` section listing all Confident and Tentative assumptions (see Assumptions Summary Block in `_context.md`)
7. Write the completed spec to `fab/changes/{name}/spec.md`

---

## Plan Generation Procedure

1. Read the template from `fab/.kit/templates/plan.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: From the proposal
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - `{DATE}`: Today's date
3. Fill in sections:
   - **Summary**: 1-2 sentences on what the change does and the chosen approach
   - **Goals / Non-Goals**: Derived from the spec requirements
   - **Technical Context**: From `fab/config.yaml` context, scoped to what this change touches
   - **Research**: Technical investigation findings (skip for straightforward changes)
   - **Decisions**: Key design decisions with rationale and rejected alternatives
   - **Risks / Trade-offs**: Known risks with mitigation strategies
   - **File Changes**: Concrete list of new, modified, and deleted files
4. Append an `## Assumptions` section listing all Confident and Tentative assumptions
5. Write the completed plan to `fab/changes/{name}/plan.md`

---

## Tasks Generation Procedure

1. Read the template from `fab/.kit/templates/tasks.md`
2. Fill in metadata fields:
   - `{CHANGE_NAME}`: From the proposal
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - If plan exists: reference `plan.md` in the header
   - If plan was skipped: omit the Plan line, include `proposal.md` reference for traceability
3. Break implementation into phased tasks:
   - **Phase 1: Setup** — scaffolding, dependencies, configuration
   - **Phase 2: Core Implementation** — primary functionality, ordered by dependency
   - **Phase 3: Integration & Edge Cases** — wiring, error states, validation
   - **Phase 4: Polish** — documentation, cleanup (only if warranted)
4. Each task follows the format: `- [ ] T{NNN} [{markers}] {description with file paths}`
   - IDs are sequential: T001, T002, ...
   - Mark parallelizable tasks with `[P]`
   - Include exact file paths in descriptions
   - Each task should be completable in one focused session
5. Include an **Execution Order** section for non-obvious dependencies
6. Write the completed tasks to `fab/changes/{name}/tasks.md`

---

## Checklist Generation Procedure

1. Read the template from `fab/.kit/templates/checklist.md`
2. The `fab/changes/{name}/checklists/` directory is created by `/fab-new` and should already exist. If it doesn't, create it.
3. Generate `fab/changes/{name}/checklists/quality.md` with:
   - `{CHANGE_NAME}`: From the proposal
   - `{YYMMDD-XXXX-slug}`: The change folder name
   - `{DATE}`: Today's date
4. Populate checklist items derived from:
   - `spec.md` — every requirement should have a corresponding CHK item under **Functional Completeness**
   - Changed requirements → **Behavioral Correctness** items
   - Deprecated requirements → **Removal Verification** items
   - Key scenarios from spec → **Scenario Coverage** items
   - Edge cases identified in spec/plan → **Edge Cases & Error Handling** items
   - Security-relevant changes → **Security** items (only if applicable)
   - Additional categories from `fab/config.yaml` `checklist.extra_categories` (if any)
5. Use sequential IDs: CHK-001, CHK-002, ...
6. Update `.status.yaml`:
   - Set `checklist.generated` to `true`
   - Set `checklist.total` to the number of checklist items generated
   - Set `checklist.completed` to `0`
