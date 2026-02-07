---
name: fab-ff
description: "Fast-forward through all remaining planning stages in one pass to reach implementation quickly."
---

# /fab:ff

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Fast-forward through all remaining planning stages in one pass. Generates specs, optionally a plan, and tasks (with quality checklist) — all in a single invocation with at most one round of questions upfront. Designed for small, well-understood changes where you want to reach implementation quickly.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `branch`, `progress`, and `checklist`

Then verify the stage-specific precondition using the preflight output:

4. Verify that `progress.proposal` is `done`

**If `progress.proposal` is not `done`, STOP.** Output:

> `Proposal is not complete. Finish the proposal first with /fab:new or /fab:continue, then run /fab:ff.`

---

## Context Loading

Load all context upfront since fast-forward traverses all planning stages:

1. `fab/config.yaml` — project config, tech stack
2. `fab/constitution.md` — project principles and constraints
3. `fab/changes/{name}/proposal.md` — the completed proposal
4. `fab/docs/index.md` — documentation landscape
5. Specific centralized docs referenced by the proposal's **Affected Docs** section (read each `fab/docs/{domain}/{doc}.md` listed under New, Modified, or Removed)

---

## Behavior

### Step 1: Frontload All Questions

Scan the proposal for ambiguities across **all** planning stages (specs, plan, tasks). Consider:

- **Spec ambiguities**: Are any requirements vague? Multiple interpretations? Missing acceptance criteria? Edge cases unaddressed?
- **Plan ambiguities**: Are there architectural choices not yet decided? Unknown dependencies? Technology selection needed?
- **Task ambiguities**: Is the scope unclear enough that task breakdown would require guessing?

Collect everything that needs user input into a **single batch of questions**. Ask once, then proceed without further interruption.

**If there are questions:**

Present them all in a single numbered list. Wait for the user to answer all of them. Then proceed to Step 2 without asking anything else.

**If the proposal is clear and unambiguous:**

Skip questions entirely and proceed directly to Step 2.

**The goal: at most one Q&A round, then heads-down generation.**

### Step 2: Generate `spec.md`

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
5. Incorporate answers from Step 1 to resolve any ambiguities — the spec should have no `[NEEDS CLARIFICATION]` markers
6. Write the completed spec to `fab/changes/{name}/spec.md`

Update `.status.yaml`:
- Set `progress.specs` to `done`
- Update `last_updated`

### Step 3: Plan Decision (Autonomous)

Evaluate whether a `plan.md` is warranted. **Unlike `/fab:continue`, this decision is made autonomously** — do NOT ask the user. The fast-forward flow should not be interrupted.

**Criteria for skipping the plan:**
- The change is small in scope (touches few files)
- The technical approach is obvious from the spec
- No significant architectural decisions are needed
- No research or library evaluation is required
- It's a single-file change, straightforward CRUD, or a well-known pattern

**If the plan is NOT warranted (skip):**
1. Record `progress.plan` as `skipped` in `.status.yaml`
2. Do NOT create a `plan.md` file
3. Proceed directly to Step 4

**If the plan IS warranted (generate):**

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
4. Write the completed plan to `fab/changes/{name}/plan.md`
5. Update `.status.yaml`:
   - Set `progress.plan` to `done`
   - Update `last_updated`

### Step 4: Generate `tasks.md`

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

### Step 5: Auto-generate Quality Checklist

1. Read the template from `fab/.kit/templates/checklist.md`
2. Create the directory `fab/changes/{name}/checklists/` if it doesn't exist
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

### Step 6: Update `.status.yaml`

After all artifacts are generated:

1. Set `stage` to `tasks`
2. Set `progress.tasks` to `done`
3. Set `checklist.generated` to `true`
4. Set `checklist.total` to the number of checklist items generated
5. Set `checklist.completed` to `0`
6. Update `last_updated` to the current ISO 8601 timestamp

---

## Output

### Clear Proposal (no questions needed)

```
Fast-forwarding from proposal...

## Spec: {Change Name}

{spec content}

Spec created.

## Plan Decision

{Plan skipped — change is straightforward.}
OR
{plan content}

Plan created.

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Fast-forward complete — specs, {plan/no plan}, tasks, and checklist generated.

Next: /fab:apply
```

### Ambiguous Proposal (questions first)

```
Fast-forwarding from proposal...

Before I can generate all planning artifacts, I need to resolve a few ambiguities:

1. {question about spec scope}
2. {question about technical approach}
3. {question about edge case}

{user answers}

## Spec: {Change Name}

{spec content incorporating answers}

Spec created.

## Plan Decision

{decision and output}

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Fast-forward complete — specs, {plan/no plan}, tasks, and checklist generated.

Next: /fab:apply
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `progress.proposal` is not `done` | Abort with: "Proposal is not complete. Finish the proposal first with /fab:new or /fab:continue, then run /fab:ff." |
| Template file missing | Abort with: "Template not found at fab/.kit/templates/{file} — kit may be corrupted." |
| Specs already done (stage is `specs` or later) | Fast-forward from current position — generate only remaining artifacts (skip spec if already done, skip plan if already done/skipped) |

---

## Key Difference from `/fab:continue`

| Behavior | `/fab:continue` | `/fab:ff` |
|----------|-----------------|-----------|
| Questions | Asked per-stage as needed | Frontloaded: one batch upfront, then no interruption |
| Plan decision | Proposes skip to user, waits for confirmation | Decides autonomously — no user confirmation |
| Stages per invocation | One stage at a time | All remaining planning stages in one pass |
| Best for | Deliberate, step-by-step planning | Quick changes with clear requirements |

---

## Next Steps Reference

After `/fab:ff` completes:

`Next: /fab:apply`
