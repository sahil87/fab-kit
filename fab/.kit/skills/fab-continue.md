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

- **`<stage>`** *(optional)* — target stage to reset to. Accepted values: `spec` or `tasks`. When provided, resets `.status.yaml` to this stage and regenerates artifacts from that point forward. Used after `/fab-review` identifies issues upstream.

If no argument is provided, the skill advances to the **next** stage in sequence.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Use the `stage` and `progress` fields from preflight output for all subsequent stage guard logic (do not re-read `.status.yaml`).

---

## Normal Flow (no argument)

When called without arguments, advance to the next artifact in sequence.

### Step 1: Determine Current Stage

Determine the current stage from the preflight output's `stage` field (derived from the `active` entry in the progress map). The stage progression is:

```
spec → tasks → apply → review → archive
```

**Guard**: Check the `active` entry to determine whether to allow continuation:

| `active` entry | Action |
|---------------|--------|
| `spec` | Generate `spec.md` → set `spec: done`, `tasks: active` |
| `tasks` | Generate `tasks.md` + checklist → set `tasks: done`, `apply: active` |
| `apply` | Block: "Planning is complete. Run /fab-apply to begin implementation." |
| `review` or `archive` | Block: "Use /fab-review or /fab-archive as appropriate." |
| No `active` entry (all `done`) | Block: "Change is complete." |

If `progress.{active_stage}` is `done` and the stage is `tasks`, block with "Planning is complete. Run /fab-apply to begin implementation."

### Step 2: Load Context

Context loading varies by the target stage being generated:

#### Generating `spec.md` (spec: active)

Load:
- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/changes/{name}/brief.md` — the completed brief
- `fab/docs/index.md` — documentation landscape
- Specific centralized docs referenced by the brief's **Affected Docs** section (read each `fab/docs/{domain}/{doc}.md` listed under New, Modified, or Removed)

#### Generating `tasks.md` (tasks: active)

Load everything from the spec context above, plus:
- `fab/changes/{name}/spec.md` — the completed spec

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

Follow the **Spec Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

#### Generating `tasks.md`

Follow the **Tasks Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

#### Auto-generate Checklist

When generating tasks, also generate the quality checklist.

Follow the **Checklist Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

### Step 3b: Recompute Confidence Score

After generating the artifact, recompute the confidence score:

1. Re-count SRAD grades across **all** artifacts in the change (brief, spec, tasks — whichever exist)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the updated `confidence` block to `.status.yaml`

This ensures the score reflects any new assumptions introduced by the generated artifact.

### Step 4: Update `.status.yaml`

After successfully generating the artifact:

1. Set `progress.{completed_stage}` to `done`
2. Set `progress.{next_stage}` to `active` (two-write transition: current → done, next → active)
3. Write the recomputed `confidence` block (from Step 3b)
4. Update `last_updated` to the current ISO 8601 timestamp

**Examples of stage transitions:**

| Active entry | Generated artifact | Progress updates |
|---|---|---|
| `spec` (active) | `spec.md` | `spec: done`, `tasks: active` |
| `tasks` (active) | `tasks.md` | `tasks: done`, `apply: active` |

### Step 5: Output

Display a summary of what was generated and the appropriate next steps.

---

## Reset Flow (with stage argument)

When called with a stage argument (e.g., `/fab-continue spec`), reset to that stage and regenerate the artifact.

### Step 1: Validate Target Stage

The target stage must be one of: `spec` or `tasks`.

**Rejected targets:**
- `brief` → Output: `Cannot reset to brief — brief is not a pipeline stage. Edit brief.md directly or use /fab-clarify.`
- `apply` → Output: `Cannot reset to apply. Use /fab-apply to re-run implementation.`
- `review` → Output: `Cannot reset to review. Use /fab-review to re-run validation.`
- `archive` → Output: `Cannot reset to archive. Use /fab-archive to complete the change.`
- Any other value → Output: `Unknown stage "{value}". Valid reset targets: spec, tasks.`

### Step 2: Load Context

Load context as described in the Normal Flow Step 2, for the target stage being regenerated.

### Step 3: Reset `.status.yaml`

1. Mark the target stage's progress as `active`
2. Mark all stages **after** the target as `pending`
3. Preserve all stages **before** the target as-is (they remain `done`)

**Example**: Resetting to `spec` when current stage is `tasks`:

```yaml
progress:
  spec: active        # reset target
  tasks: pending      # invalidated
  apply: pending      # invalidated
  review: pending     # invalidated
  archive: pending    # invalidated
```

### Step 4: Regenerate Target Artifact

Regenerate the target artifact **in place** — update the existing file rather than deleting and recreating from scratch. Preserve what's still valid from the existing artifact while incorporating any new context or corrections.

1. Read the existing artifact file (e.g., `spec.md` or `tasks.md`)
2. Load the relevant template as a structural guide
3. Regenerate the content, preserving valid portions and updating what needs to change
4. Write the updated artifact back to the same file

### Step 5: Invalidate Downstream Artifacts

After regenerating the target, invalidate all downstream artifacts:

- **If target is `spec`**: Tasks are now potentially stale. Leave the file in place but note it needs regeneration. Reset its progress to `pending`.
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

### Normal Flow — Spec Generated

```
Stage: spec (active). Creating spec.md...

## Spec: {Change Name}

{spec content}

Spec created. {N} [NEEDS CLARIFICATION] markers in spec.md. Run /fab-clarify to resolve.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | {decision} | {rationale} |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.

Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)
```

### Normal Flow — Tasks Generated

```
Stage: spec (done). Creating tasks.md...

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklists/quality.md with {N} items.

Tasks created.

Next: /fab-apply
```

### Reset Flow

```
/fab-continue spec

Resetting to spec stage...
- spec: regenerating spec.md
- tasks: invalidated (pending)

## Spec: {Change Name}

{updated spec content}

Spec regenerated. Downstream artifacts (tasks) need regeneration.

Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)
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
| Reset target is `brief` | Abort with: "Cannot reset to brief — brief is not a pipeline stage. Edit brief.md directly or use /fab-clarify." |
| Reset target is `apply`, `review`, or `archive` | Abort with: "Cannot reset to {stage}. Use /fab-{stage} directly." |
| Template file missing | Abort with: "Template not found at fab/.kit/templates/{file} — kit may be corrupted." |

---

## Next Steps Reference

After `/fab-continue` completes, output the appropriate next line:

| Stage completed | Next line |
|----------------|-----------|
| spec | `Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| tasks | `Next: /fab-apply` |
| reset to spec | `Next: /fab-continue (tasks) or /fab-ff (fast-forward) or /fab-clarify (refine spec)` |
| reset to tasks | `Next: /fab-apply` |
