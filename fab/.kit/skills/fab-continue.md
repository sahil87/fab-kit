---
name: fab-continue
description: "Advance to the next pipeline stage — planning, implementation, review, or archive — or reset to a given stage."
---

# /fab-continue [<stage>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Advance through the entire 6-stage Fab pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. Covers planning (brief → spec → tasks), execution (apply), validation (review), and completion (archive). When called with a stage argument, resets to that stage and re-runs from there.

This is the primary command for moving through the Fab workflow. Developers primarily need two commands: `/fab-continue` (advance) and `/fab-clarify` (refine).

---

## Arguments

- **`<stage>`** *(optional)* — target stage to reset to. Accepted values: `brief`, `spec`, `tasks`, `apply`, `review`, `archive`. When provided, resets `.status.yaml` to this stage and re-runs from there. Used after review identifies issues upstream, or to re-run any stage.

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

When called without arguments, advance to the next stage in sequence.

### Step 1: Determine Current Stage

Determine the current stage from the preflight output's `stage` field (derived from the `active` entry in the progress map). The stage progression is:

```
brief → spec → tasks → apply → review → archive
```

**Guard**: Check the `active` entry to determine which behavior to dispatch:

| `active` entry | Action |
|---------------|--------|
| `brief` | Generate `spec.md` → set `brief: done`, `spec: active` |
| `spec` | Generate `tasks.md` + checklist → set `spec: done`, `tasks: active` |
| `tasks` | Execute apply behavior → set `tasks: done`, `apply: active` → on completion set `apply: done`, `review: active` |
| `apply` | Resume apply behavior → on completion set `apply: done`, `review: active` |
| `review` | Execute review behavior → on pass: `review: done`, `archive: active`. On fail: `review: failed`, `apply: active` |
| `archive` | Execute archive behavior → `archive: done` |
| No `active` entry (all `done`) | Block: "Change is complete." |

### Step 2: Load Context

Context loading varies by the stage being executed:

#### Planning Stages (brief, spec)

##### Generating `spec.md` (brief: active)

Load:
- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/changes/{name}/brief.md` — the completed brief
- `fab/docs/index.md` — documentation landscape
- Specific centralized docs referenced by the brief's **Affected Docs** section (read each `fab/docs/{domain}/{doc}.md` listed under New, Modified, or Removed)

##### Generating `tasks.md` (spec: active)

Load everything from the spec context above, plus:
- `fab/changes/{name}/spec.md` — the completed spec

#### Execution Stage (tasks, apply)

Load:
- `fab/config.yaml` — project config, tech stack, conventions
- `fab/constitution.md` — project principles and constraints
- `fab/design/index.md` — specifications landscape
- `fab/changes/{name}/tasks.md` — the task list to execute
- `fab/changes/{name}/spec.md` — requirements and scenarios
- `fab/changes/{name}/brief.md` — original intent
- **Relevant source code** — read files referenced in task descriptions. Scope to files actually touched — do not load the entire codebase.

#### Review Stage

Load everything from the execution context above, plus:
- `fab/changes/{name}/checklist.md` — the quality checklist to verify
- **Centralized docs** — read `fab/docs/index.md` and the specific docs referenced by the brief's Affected Docs section, to check for doc drift
- **Relevant source code** — read files touched by the change

#### Archive Stage

Load:
- `fab/config.yaml` — project config, tech stack, conventions
- `fab/constitution.md` — project principles and constraints
- `fab/design/index.md` — specifications landscape
- `fab/changes/{name}/spec.md` — requirements and scenarios to hydrate
- `fab/changes/{name}/brief.md` — original intent, Affected Docs section
- `fab/docs/index.md` — top-level documentation index
- **Target centralized doc(s)** — read the specific docs referenced by the brief's Affected Docs section. For each doc path listed, read `fab/docs/{domain}/{name}.md` if it exists. Also read the domain index `fab/docs/{domain}/index.md`.

### Step 2b: SRAD-Based Question Selection (Planning Stages Only)

Before generating a planning artifact (spec or tasks), apply the SRAD framework (defined in `_context.md`) to decision points for the current stage:

1. **Evaluate each decision point** against the four SRAD dimensions
2. **Assign confidence grades** (Certain, Confident, Tentative, Unresolved)
3. **Ask up to 3 Unresolved questions** — prioritize those with the lowest Reversibility + lowest Agent Competence (Critical Rule). Interruption budget is 1-2 per stage for typical changes, up to 3 for highly ambiguous ones.
4. **Assume Confident and Tentative decisions** — Tentative decisions get `<!-- assumed: ... -->` markers in the artifact
5. Track all assumptions for the Assumptions summary (see Step 5)

### Step 3: Execute Stage

Based on the current active stage, execute the appropriate behavior:

#### Planning: Generating `spec.md`

Follow the **Spec Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

#### Planning: Generating `tasks.md`

Follow the **Tasks Generation Procedure** defined in `fab/.kit/skills/_generation.md`.

Also follow the **Checklist Generation Procedure** to auto-generate the quality checklist.

#### Execution: Apply Behavior

See [Apply Behavior](#apply-behavior) section below.

#### Validation: Review Behavior

See [Review Behavior](#review-behavior) section below.

#### Completion: Archive Behavior

See [Archive Behavior](#archive-behavior) section below.

### Step 3b: Recompute Confidence Score (Planning Stages Only)

After generating a planning artifact, recompute the confidence score:

1. Re-count SRAD grades across **all** artifacts in the change (brief, spec, tasks — whichever exist)
2. Apply the confidence formula (see `_context.md` Confidence Scoring section)
3. Write the updated `confidence` block to `.status.yaml`

This ensures the score reflects any new assumptions introduced by the generated artifact.

### Step 4: Update `.status.yaml`

After successfully completing the stage:

1. Set `progress.{completed_stage}` to `done`
2. Set `progress.{next_stage}` to `active` (two-write transition: current → done, next → active)
3. For planning stages: write the recomputed `confidence` block (from Step 3b)
4. Update `last_updated` to the current ISO 8601 timestamp

**Examples of stage transitions:**

| Active entry | Completed work | Progress updates |
|---|---|---|
| `brief` (active) | `spec.md` generated | `brief: done`, `spec: active` |
| `spec` (active) | `tasks.md` generated | `spec: done`, `tasks: active` |
| `tasks` (active) | All tasks executed | `tasks: done`, `apply: done`, `review: active` |
| `apply` (active) | Remaining tasks executed | `apply: done`, `review: active` |
| `review` (active) | Validation passed | `review: done`, `archive: active` |
| `review` (active) | Validation failed | `review: failed`, `apply: active` |
| `archive` (active) | Docs hydrated, change moved | `archive: done` |

### Step 5: Output

Display a summary of what was completed and the appropriate next steps.

---

## Apply Behavior

When the active stage is `tasks` or `apply`, execute implementation tasks from `tasks.md`.

### Preconditions

- `fab/changes/{name}/tasks.md` MUST exist
- If active stage is `tasks`: set `tasks: done`, `apply: active` before starting execution

### Task Execution

1. **Parse tasks**: Read `fab/changes/{name}/tasks.md` and extract all task items
   - **Unchecked items** (`- [ ]`): tasks remaining to execute
   - **Checked items** (`- [x]`): tasks already completed (skip these)
   - **Phase headers** (`## Phase N: ...`): group boundaries for execution order

2. **If all tasks are already checked**, output: "All tasks already complete. Implementation finished." Set `apply: done`, `review: active`. Stop.

3. **Determine execution order**:
   - Phases are sequential: all Phase 1 tasks before Phase 2, etc.
   - Within a phase, non-`[P]` tasks are sequential (listed order)
   - Within a phase, `[P]` tasks are parallelizable
   - Respect Execution Order section constraints (e.g., "T004 blocks T005")

4. **For each unchecked task** in execution order:
   a. Parse the task ID, markers, and description
   b. Read relevant source files referenced in the task description
   c. Implement the task following the spec, constitution, and existing patterns
   d. Run relevant tests after completion. Fix failures before proceeding.
   e. Mark the task `- [x]` immediately upon completion (not batched)
   f. Update `.status.yaml` `last_updated`

5. **On completion of all tasks**: Set `apply: done`, `review: active`. Update `last_updated`.

### Resumability

Start from the **first unchecked item** (`- [ ]`). All checked items are assumed complete. Re-invoking after interruption picks up exactly where it left off. The markdown checklist is the progress state.

---

## Review Behavior

When the active stage is `apply` (with `apply: done`) or `review`, validate implementation against specs and checklists.

### Preconditions

- `fab/changes/{name}/tasks.md` MUST exist
- `fab/changes/{name}/checklist.md` MUST exist
- All tasks in `tasks.md` MUST be checked `[x]`. If any are unchecked, STOP with: "{N} of {total} tasks are incomplete. Run /fab-continue to finish implementation first."

### Validation Steps

#### Step 1: Verify All Tasks Complete

Read `tasks.md` and count checked vs unchecked tasks. All must be `[x]`.

#### Step 2: Verify Quality Checklist

For **each** `CHK-*` item in `checklist.md`:
1. Read the item's criterion
2. Inspect relevant code/tests; cross-reference against `spec.md`
3. If met: mark `[x]`
4. If N/A: mark `[x]` with `**N/A**: {reason}` prefix
5. If not met: leave `[ ]` and record failure with specific reason

#### Step 3: Run Affected Tests

Run tests scoped to modules/files touched by the change (not the full suite unless pervasive).

#### Step 4: Spot-Check Spec Requirements

Compare implementation against key requirements from `spec.md`. Verify GIVEN/WHEN/THEN scenarios are handled.

#### Step 5: Check for Doc Drift

Compare implementation against centralized docs referenced in the brief. Doc drift is a **warning** (not failure) — signals work for archive.

### Review Verdict

**Pass** — all tasks `[x]`, all checklist items `[x]`, tests pass, spec matches:
- Set `review: done`, `archive: active`
- Update `checklist.completed` count
- Output review report, then: `Next: /fab-continue`

**Fail** — any check failed:
- Set `review: failed`, `apply: active`
- Update `checklist.completed` count
- Output review report with failure details, then present rework options

### Rework Options (On Failure)

Present all options and let the user choose:

**Option 1: Fix code** — Implementation bug.
1. Identify tasks needing rework from failed checklist items
2. Uncheck those tasks: `- [x]` → `- [ ]` with `<!-- rework: {reason} -->` comment
3. User runs `/fab-continue` to resume apply

**Option 2: Revise tasks** — Missing or wrong tasks.
1. Add new tasks with next sequential ID, or modify existing (uncheck modified tasks)
2. Completed unaffected tasks stay `[x]`
3. User runs `/fab-continue` to resume apply

**Option 3: Revise spec** — Requirements wrong or incomplete.
1. Run `/fab-continue spec` to reset and regenerate all downstream

---

## Archive Behavior

When the active stage is `review` (with `review: done`) or `archive`, complete the change and hydrate learnings.

### Preconditions

- `progress.review` MUST be `done`. If not: STOP with "Review has not passed. Run /fab-continue to validate implementation first."
- All tasks in `tasks.md` MUST be `[x]`
- All checklist items in `checklist.md` MUST be `[x]` (including N/A items)

### Steps

#### Step 1: Final Validation

Verify all tasks `[x]` and all checklist items `[x]`. Report: "Final validation passed."

#### Step 2: Concurrent Change Check

Scan `fab/changes/` for other active change folders (exclude current and `archive/`). For each, check if its `spec.md` references the same centralized doc paths. If overlap: warn (not block).

#### Step 3: Hydrate into `fab/docs/`

For each centralized doc referenced in the brief's Affected Docs:

- **New doc**: Create domain folder if needed, create doc from template, populate from `spec.md`, update domain index and top-level index
- **Existing doc**: Update Requirements section (add new, update changed, remove deprecated), update Design Decisions, add Changelog row, update indexes
- **Extract Design Decisions**: From `spec.md` Decisions section, include durable decisions (architectural, API, data model). Skip tactical details.
- **Changelog rows**: Most-recent-first, one-line summary of what changed for this specific doc

#### Step 4: Update `.status.yaml`

Set `progress.archive` to `done`. Update `last_updated`.

#### Step 5: Move Change Folder to Archive

Move `fab/changes/{name}/` → `fab/changes/archive/{name}/`. Create `archive/` if needed. Do NOT rename.

#### Step 6: Update Archive Index

Maintain `fab/changes/archive/index.md`. If it doesn't exist, create with backfill. Prepend new entry (most-recent-first): `- **{folder-name}** — {1-2 sentence description from brief Why section}`.

#### Step 7: Mark Backlog Item Done

If the brief contains a backlog ID (in Origin section), find and mark it done in `fab/backlog.md`. Move from Backlog section to Done section.

#### Step 8: Clear Pointer

Delete `fab/current`.

### Order of Operations (Fail-Safe)

Steps 4-8 execute in this order for safety: status first (recoverable if interrupted), folder move second, index third, backlog fourth, pointer last.

---

## Reset Flow (with stage argument)

When called with a stage argument (e.g., `/fab-continue spec`), reset to that stage and re-run from there.

### Step 1: Validate Target Stage

The target stage must be one of: `brief`, `spec`, `tasks`, `apply`, `review`, `archive`.

Any other value → Output: `Unknown stage "{value}". Valid reset targets: brief, spec, tasks, apply, review, archive.`

### Step 2: Load Context

Load context as described in the Normal Flow Step 2, for the target stage.

**Special case for brief reset**: When resetting to brief, regenerate brief.md using:
- `fab/config.yaml` — project config, tech stack
- `fab/constitution.md` — project principles and constraints
- `fab/docs/index.md` — documentation landscape

### Step 3: Reset `.status.yaml`

1. Mark the target stage's progress as `active`
2. Mark all stages **after** the target as `pending`
3. Preserve all stages **before** the target as-is (they remain `done`)

**Example**: Resetting to `spec` when current stage is `apply`:

```yaml
progress:
  brief: done         # preserved
  spec: active        # reset target
  tasks: pending      # invalidated
  apply: pending      # invalidated
  review: pending     # invalidated
  archive: pending    # invalidated
```

### Step 4: Execute Target Stage

**For planning stages** (brief, spec, tasks): Regenerate the artifact **in place** — update the existing file, preserving what's still valid.

**For execution stages** (apply, review, archive): Re-run the stage's behavior (apply resumes from first unchecked task, review re-validates, archive re-hydrates). Task checkboxes are NOT reset — they reflect real implementation progress.

### Step 5: Invalidate Downstream Artifacts (Planning Resets Only)

- **If target is `brief`**: All downstream artifacts are potentially stale. Reset all to `pending`.
- **If target is `spec`**: Tasks are potentially stale. Leave file in place but reset progress to `pending`.
- **If target is `tasks`**:
  - Reset all task checkboxes: `- [x]` → `- [ ]`
  - Regenerate the checklist at `fab/changes/{name}/checklist.md`
  - Reset `checklist.completed` to `0` in `.status.yaml`

### Step 6: Update `.status.yaml` and Report

1. For planning resets: set `progress.{target}` to `done` after regeneration
2. For execution resets: the stage behavior handles its own status updates
3. Update `last_updated`
4. Report what was reset and what downstream artifacts were invalidated

---

## Output

### Normal Flow — Spec Generated

```
Stage: brief (done). Creating spec.md...

## Spec: {Change Name}

{spec content}

Spec created. {N} [NEEDS CLARIFICATION] markers in spec.md. Run /fab-clarify to resolve.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | {decision} | {rationale} |

{N} assumptions made ({C} confident, {T} tentative). Run /fab-clarify to review.

Next: /fab-continue or /fab-ff or /fab-clarify
```

### Normal Flow — Tasks Generated

```
Stage: spec (done). Creating tasks.md...

## Tasks: {Change Name}

{tasks content}

## Quality Checklist

Generated checklist.md with {N} items.

Tasks created.

Next: /fab-continue or /fab-ff
```

### Normal Flow — Apply Complete

```
Starting implementation. {N} tasks remaining.

Executing T001: {description}...
✓ T001 complete. Tests passed.

...

All {N} tasks complete. Implementation finished.

Next: /fab-continue
```

### Normal Flow — Review Passed

```
Review: {change name}

Tasks:     ✓ {total}/{total} complete
Checklist: ✓ {total}/{total} passed
Tests:     ✓ Passed
Spec:      ✓ Requirements verified
Docs:      ✓ No drift detected

Review PASSED. All checks green.

Next: /fab-continue
```

### Normal Flow — Review Failed

```
Review: {change name}

Tasks:     ✓ {total}/{total} complete
Checklist: ✗ {passed}/{total} passed
  - CHK-007: {failure reason}
Tests:     ✓ Passed
Spec:      ✓ Requirements verified
Docs:      ✓ No drift detected

Review FAILED. {N} issue(s) found.

Rework options:
  1. Fix code — uncheck {N} tasks for rework, then /fab-continue
  2. Revise tasks — add/modify tasks in tasks.md, then /fab-continue
  3. Revise spec — /fab-continue spec (resets all downstream)

Which option? (1-3)
```

### Normal Flow — Archive Complete

```
Archive: {change name}

Validation: ✓ All tasks and checklist items complete
Concurrent: ✓ No conflicts

Hydrated docs:
  - fab/docs/{domain}/{name}.md (updated)

Status:   ✓ archive: done
Archived: ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Reset Flow — Planning Reset

```
/fab-continue spec

Resetting to spec stage...
- spec: regenerating spec.md
- tasks: invalidated (pending)

## Spec: {Change Name}

{updated spec content}

Spec regenerated. Downstream artifacts (tasks) need regeneration.

Next: /fab-continue or /fab-ff or /fab-clarify
```

### Reset Flow — Execution Reset

```
/fab-continue apply

Resetting to apply stage...
- apply: active
- review: pending
- archive: pending

Resuming implementation. {M} of {N} tasks already complete, {R} remaining.

{task execution output}

Next: /fab-continue
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| No active entry (all done) | Output: "Change is complete." |
| `tasks.md` missing when apply needed | Abort with: "No tasks.md found. Run /fab-continue to generate tasks first." |
| `checklist.md` missing when review needed | Abort with: "No checklist found. Run /fab-continue to generate the checklist first." |
| Unchecked tasks when review attempted | Abort with: "{N} of {total} tasks are incomplete. Run /fab-continue to finish implementation first." |
| Review not passed when archive attempted | Abort with: "Review has not passed. Run /fab-continue to validate implementation first." |
| Unknown reset target | Abort with: "Unknown stage \"{value}\". Valid reset targets: brief, spec, tasks, apply, review, archive." |
| Template file missing (planning stages) | Abort with: "Template not found at fab/.kit/templates/{file} — kit may be corrupted." |
| Test failure during apply | Fix implementation, re-run tests, repeat until passing |
| All tasks already `[x]` during apply | Set apply: done, output "All tasks already complete." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — progresses through all 6 stages |
| Idempotent? | **Yes** — safe to re-invoke; planning regenerates, apply resumes from first unchecked task, review re-validates, archive recovers from interruption |
| Modifies planning artifacts? | **Yes** — generates spec.md, tasks.md, checklist.md |
| Modifies tasks.md? | **Yes** — marks tasks `[x]` during apply; unchecks during review rework |
| Modifies checklist.md? | **Yes** — marks items `[x]` during review |
| Modifies source code? | **Yes** — during apply behavior |
| Modifies `fab/docs/`? | **Yes** — during archive behavior (hydration) |
| Updates `.status.yaml`? | **Yes** — after each stage completion |
| Moves change folder? | **Yes** — during archive (to `fab/changes/archive/`) |
| Clears `fab/current`? | **Yes** — during archive |

---

## Next Steps Reference

After `/fab-continue` completes, output the appropriate next line:

| Stage completed | Next line |
|----------------|-----------|
| spec | `Next: /fab-continue or /fab-ff or /fab-clarify` |
| tasks | `Next: /fab-continue or /fab-ff` |
| apply | `Next: /fab-continue` |
| review (pass) | `Next: /fab-continue` |
| review (fail) | *(contextual rework options)* |
| archive | `Next: /fab-new <description> (start next change)` |
| reset to brief | `Next: /fab-continue or /fab-clarify` |
| reset to spec | `Next: /fab-continue or /fab-ff or /fab-clarify` |
| reset to tasks | `Next: /fab-continue or /fab-ff` |
| reset to apply | `Next: /fab-continue` |
| reset to review | `Next: /fab-continue` |
| reset to archive | `Next: /fab-new <description> (start next change)` |
