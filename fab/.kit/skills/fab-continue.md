---
name: fab-continue
description: "Advance to the next planning stage and generate its artifact, or reset to a given stage."
---

# /fab-continue [<stage>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Advance to the next planning stage and generate the corresponding artifact — or, when called with a stage argument, reset to that stage and regenerate from there. This is the primary skill for moving through the Fab workflow one step at a time.

---

## Arguments

- **`<stage>`** *(optional)* — target stage to reset to. Accepted values: `specs`, `plan`, or `tasks`. When provided, resets `.status.yaml` to this stage and regenerates artifacts from that point forward. Used after `/fab-review` identifies issues upstream.

If no argument is provided, the skill advances to the **next** stage in sequence.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `branch`, `progress`, and `checklist`

Use the `stage` and `progress` fields from preflight output for all subsequent stage guard logic (do not re-read `.status.yaml`).

---

## Normal Flow (no argument)

When called without arguments, advance to the next artifact in sequence.

### Step 1: Determine Current Stage

Read `.status.yaml` and identify the current stage. The stage progression is:

```
proposal → specs → plan → tasks → apply → review → archive
```

Find the current stage from `.status.yaml`'s `stage` field. The **next** stage is the one to generate.

**Guard**: If the current stage is `tasks` (done) or later, this skill does not apply. Output:

> `Planning is complete. Run /fab-apply to begin implementation.`

If the current stage is `apply` or later:

> `Implementation is underway. Use /fab-apply, /fab-review, or /fab-archive as appropriate.`

### Step 2: Load Context

Context loading varies by the target stage being generated:

#### Generating `spec.md` (proposal → specs)

Load:
- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/changes/{name}/proposal.md` — the completed proposal
- `fab/docs/index.md` — documentation landscape
- Specific centralized docs referenced by the proposal's **Affected Docs** section (read each `fab/docs/{domain}/{doc}.md` listed under New, Modified, or Removed)

#### Generating `plan.md` (specs → plan)

Load everything from the specs context above, plus:
- `fab/changes/{name}/spec.md` — the completed spec

#### Generating `tasks.md` (plan → tasks, or specs → tasks if plan skipped)

Load everything from the plan context above, plus:
- `fab/changes/{name}/plan.md` — the completed plan (if it exists; skip if plan was marked `skipped`)

### Step 2b: SRAD-Based Question Selection

Before generating the artifact, apply the SRAD framework (defined in `_context.md`) to decision points for the current stage:

1. **Evaluate each decision point** against the four SRAD dimensions
2. **Assign confidence grades** (Certain, Confident, Tentative, Unresolved)
3. **Ask up to 3 Unresolved questions** — prioritize those with the lowest Reversibility + lowest Agent Competence (Critical Rule). Interruption budget is 1-2 per stage for typical changes, up to 3 for highly ambiguous ones.
4. **Assume Confident and Tentative decisions** — Tentative decisions get `<!-- assumed: ... -->` markers in the artifact
5. Track all assumptions for the Assumptions summary (see Step 5)

### Step 3: Generate Artifact

Based on the target stage, generate the appropriate artifact:

#### Generating `spec.md`

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

#### Plan Decision (specs → plan transition)

Before generating a plan, evaluate whether one is warranted:

**Criteria for skipping the plan:**
- The change is small in scope (touches few files)
- The technical approach is obvious from the spec
- No significant architectural decisions are needed
- No research or library evaluation is required

**If a plan can be skipped**, propose this to the user:

> *"This change is straightforward — skip plan and go directly to tasks?"*

- **If the user agrees**: Record `plan: skipped` in `.status.yaml` (set `progress.plan` to `skipped`), then proceed directly to generating tasks (Step 3, generating `tasks.md`). Do NOT create a `plan.md` file.
- **If the user wants a plan**: Generate the plan normally (continue below).

**Note**: Unlike `/fab-ff` which autonomously decides whether to skip the plan, `/fab-continue` always confirms with the user before skipping.

#### Generating `plan.md`

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

#### Generating `tasks.md`

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

#### Auto-generate Checklist

When generating tasks (regardless of whether plan was skipped), also generate the quality checklist:

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

### Step 3b: Recompute Confidence Score

After generating the artifact, recompute the confidence score:

1. Re-count SRAD grades across **all** artifacts in the change (proposal, spec, plan, tasks — whichever exist)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the updated `confidence` block to `.status.yaml`

This ensures the score reflects any new assumptions introduced by the generated artifact.

### Step 4: Update `.status.yaml`

After successfully generating the artifact:

1. Set `stage` to the stage just completed (e.g., `specs`, `plan`, or `tasks`)
2. Set `progress.{completed_stage}` to `done`
3. If advancing past the completed stage, set `progress.{next_stage}` to `active` (only if there is a next planning stage)
4. Write the recomputed `confidence` block (from Step 3b)
5. Update `last_updated` to the current ISO 8601 timestamp

**Examples of stage transitions:**

| Previous stage | Generated artifact | New `stage` | Progress updates |
|---|---|---|---|
| `proposal` (done) | `spec.md` | `specs` | `specs: done` |
| `specs` (done) | `plan.md` | `plan` | `plan: done` |
| `specs` (done) | plan skipped → `tasks.md` | `tasks` | `plan: skipped`, `tasks: done` |
| `plan` (done) | `tasks.md` | `tasks` | `tasks: done` |

### Step 5: Output

Display a summary of what was generated and the appropriate next steps.

---

## Reset Flow (with stage argument)

When called with a stage argument (e.g., `/fab-continue specs`), reset to that stage and regenerate the artifact.

### Step 1: Validate Target Stage

The target stage must be one of: `specs`, `plan`, or `tasks`.

**Rejected targets:**
- `proposal` → Output: `Cannot reset to proposal. Run /fab-new to start a new change instead.`
- `apply` → Output: `Cannot reset to apply. Use /fab-apply to re-run implementation.`
- `review` → Output: `Cannot reset to review. Use /fab-review to re-run validation.`
- `archive` → Output: `Cannot reset to archive. Use /fab-archive to complete the change.`
- Any other value → Output: `Unknown stage "{value}". Valid reset targets: specs, plan, tasks.`

### Step 2: Load Context

Load context as described in the Normal Flow Step 2, for the target stage being regenerated.

### Step 3: Reset `.status.yaml`

1. Set `stage` to the target stage (with `active` progress)
2. Mark the target stage's progress as `active`
3. Mark all stages **after** the target as `pending`
4. Preserve all stages **before** the target as-is (they remain `done`)

**Example**: Resetting to `specs` when current stage is `tasks`:

```yaml
stage: specs
progress:
  proposal: done      # preserved
  specs: active       # reset target
  plan: pending       # invalidated
  tasks: pending      # invalidated
  apply: pending      # invalidated
  review: pending     # invalidated
  archive: pending    # invalidated
```

### Step 4: Regenerate Target Artifact

Regenerate the target artifact **in place** — update the existing file rather than deleting and recreating from scratch. Preserve what's still valid from the existing artifact while incorporating any new context or corrections.

1. Read the existing artifact file (e.g., `spec.md`, `plan.md`, or `tasks.md`)
2. Load the relevant template as a structural guide
3. Regenerate the content, preserving valid portions and updating what needs to change
4. Write the updated artifact back to the same file

### Step 5: Invalidate Downstream Artifacts

After regenerating the target, invalidate all downstream artifacts:

- **If target is `specs`**: The plan (if it exists and is not `skipped`) and tasks are now potentially stale. Leave the files in place but note they need regeneration. Reset their progress to `pending`.
- **If target is `plan`**: Tasks are now potentially stale. Reset tasks progress to `pending`.
- **If target is `tasks`**:
  - Reset all task checkboxes to unchecked: `- [x]` → `- [ ]`
  - Regenerate the checklist at `fab/changes/{name}/checklists/quality.md`
  - Reset `checklist.completed` to `0` in `.status.yaml`

### Step 6: Update `.status.yaml` and Report

1. Set `progress.{target}` to `done` (the target artifact has been regenerated)
2. Update `last_updated`
3. Report what was reset and what downstream artifacts were invalidated

---

## Output

### Normal Flow — Specs Generated

```
Stage: proposal (done). Creating spec.md...

## Spec: {Change Name}

{spec content}

Spec created. {N} [NEEDS CLARIFICATION] markers in spec.md. Run /fab-clarify to resolve.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | {decision} | {rationale} |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.

Next: /fab-continue (plan) or /fab-ff (fast-forward) or /fab-clarify (refine spec)
```

### Normal Flow — Plan Skipped

```
Stage: specs (done). Evaluating plan...

This change is straightforward — skip plan and go directly to tasks?

{user confirms}

Plan skipped. Creating tasks.md...

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Tasks created.

Next: /fab-apply
```

### Normal Flow — Plan Generated

```
Stage: specs (done). Creating plan.md...

## Plan: {Change Name}

{plan content}

Plan created.

## Key Decisions

| # | Decision | Rationale | Rejected |
|---|----------|-----------|----------|
| 1 | {choice made} | {why} | {alternative not chosen} |

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Tentative | {decision} | {rationale} |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review tentative assumptions.

Next: /fab-continue (tasks) or /fab-clarify (refine plan)
```

### Normal Flow — Tasks Generated

```
Stage: plan (done). Creating tasks.md...

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Tasks created.

Next: /fab-apply
```

### Reset Flow

```
/fab-continue specs

Resetting to specs stage...
- specs: regenerating spec.md
- plan: invalidated (pending)
- tasks: invalidated (pending)

## Spec: {Change Name}

{updated spec content}

Spec regenerated. Downstream artifacts (plan, tasks) need regeneration.

Next: /fab-continue (plan) or /fab-ff (fast-forward) or /fab-clarify (refine spec)
```

### Reset Flow — Tasks Reset

```
/fab-continue tasks

Resetting to tasks stage...
- tasks: regenerating tasks.md
- All task checkboxes reset to unchecked
- Checklist regenerated

## Tasks: {Change Name}

{updated tasks content}

Tasks regenerated.

Next: /fab-apply
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| Current stage is `apply` or later (normal flow) | Abort with guidance to use `/fab-apply`, `/fab-review`, or `/fab-archive` |
| Reset target is `proposal` | Abort with: "Cannot reset to proposal. Run /fab-new to start a new change instead." |
| Reset target is `apply`, `review`, or `archive` | Abort with: "Cannot reset to {stage}. Use /fab-{stage} directly." |
| Template file missing | Abort with: "Template not found at fab/.kit/templates/{file} — kit may be corrupted." |

---

## Next Steps Reference

After `/fab-continue` completes, output the appropriate next line:

| Stage completed | Next line |
|----------------|-----------|
| specs | `Next: /fab-continue (plan) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| plan | `Next: /fab-continue (tasks) or /fab-clarify (refine plan)` |
| plan (skipped) → tasks | `Next: /fab-apply` |
| tasks | `Next: /fab-apply` |
| reset to specs | `Next: /fab-continue (plan) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| reset to plan | `Next: /fab-continue (tasks) or /fab-clarify (refine plan)` |
| reset to tasks | `Next: /fab-apply` |
