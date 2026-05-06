---
name: fab-continue
description: "Advance to the next pipeline stage — planning, implementation, review, or hydrate — or reset to a given stage."
helpers: [_generation, _review]
---

# /fab-continue [<change-name>] [<stage>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Advance through the 7-stage Fab pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. When called with a stage argument, resets to that stage and re-runs from there.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Passed to preflight as `$1` (see `_preamble.md` §2).
- **`<stage>`** *(optional)* — reset target: `intake`, `spec`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` target errors with a pointer to `apply` / `spec` (see Error Handling).

Both may be provided in any order. Stage names are treated as reset targets; all others as change-name overrides.

---

## Pre-flight

1. Classify arguments: stage name vs. change-name override (stage names take priority)
2. Run preflight per `_preamble.md` §2
3. Use preflight's `stage` and `progress` fields for all subsequent logic

---

## Normal Flow

### Step 1: Determine Current Stage

Dispatch on preflight's derived `stage` and `display_state`. If progress is `pending`, run `fab status start <change> <stage> fab-continue` before dispatching.

**State-based dispatch**: For planning stages, `/fab-continue` consolidates work into a single invocation:
- **`ready`** (planning) → Finish the current stage, start the next, generate its artifact, and advance it to `ready`
- **`active`** (planning) → Generate the stage's artifact and advance to `ready` (backward compat for interrupted generations)
- **`active`/`ready`** (execution) → Execute the stage's behavior and finish it

| Derived stage | State | Action |
|---------------|-------|--------|
| `intake` | `ready` | finish intake → start spec → generate `spec.md` → advance spec to `ready` |
| `intake` | `active` | generate intake if missing → advance to `ready` |
| `spec` | `ready` | finish spec → start apply → execute apply (apply's entry sub-step generates `plan.md`, then runs tasks) |
| `spec` | `active` | generate `spec.md` → advance to `ready` |
| `apply` | `active`/`ready` | Execute apply (entry: generate `plan.md` if absent; main: run tasks) → on completion run `finish <change> apply fab-continue` (auto-activates review) |
| `review` | `active`/`ready` | Execute review → pass: run `finish <change> review fab-continue` (auto-activates hydrate). Fail: run `fail <change> review` then `start <change> apply fab-continue` |
| `hydrate` | `active`/`ready` | Execute hydrate → run `finish <change> hydrate fab-continue` |
| `ship` | `active`/`ready` | Execute `/git-pr` behavior → on completion `finish <change> ship fab-continue` (auto-activates review-pr) |
| `review-pr` | `active`/`ready` | Execute `/git-pr-review` behavior → pass: `finish <change> review-pr fab-continue`. Fail: `fail <change> review-pr` |
| all `done` | — | Block: "Change is complete." |

### Step 2: Load Context

Load per `_preamble.md` layers. Stage-specific additions: planning stages load intake + memory files; apply loads spec + plan (if it already exists) + source code; review adds plan + memory; hydrate loads memory index + target files.

### Step 3: SRAD + Generation

**Planning stages only**: Apply SRAD (`_preamble.md`) before generating. Budget: 1-2 unresolved questions per stage. Tentative decisions get `<!-- assumed: ... -->` markers.

| Stage | Procedure |
|-------|-----------|
| spec | **Spec Generation Procedure** (`_generation.md`) |
| apply | [Apply Behavior](#apply-behavior) — entry sub-step invokes **Plan Generation Procedure** (`_generation.md`) |
| review | **Review Behavior** (`_review.md`) |
| hydrate | [Hydrate Behavior](#hydrate-behavior) |

**Spec stage only**: After spec generation, invoke `fab score <change>` to compute the confidence score. No scoring at other stages.

### Step 4: Update `.status.yaml`

Use event commands via CLI to update `.status.yaml`. The `finish` command handles the two-write transition atomically: `fab status finish <change> <completed-stage> fab-continue`. This sets `{completed}` → `done`, auto-activates the next pending stage, refreshes `last_updated`, and updates `stage_metrics`.

For other state changes, use the appropriate event command (driver is always optional):
- `fab status start <change> <stage> fab-continue` — pending/failed → active
- `fab status advance <change> <stage>` — active → ready
- `fab status fail <change> <stage>` — active → failed (review only)
- `fab status reset <change> <stage> fab-continue` — done/ready → active (cascades downstream to pending)

### Step 5: Output

Display summary. Include Assumptions summary for planning stages. End with `Next:` per state table in `_preamble.md`.

---

## Apply Behavior

Apply runs as **two sub-steps in a single skill invocation**: a Plan Generation entry sub-step that produces `plan.md`, followed by the Task Execution main sub-step.

### Preconditions

- `spec.md` MUST exist (used as input to plan generation)

### Plan Generation (entry sub-step)

1. **If `plan.md` already exists** with at least a `## Tasks` heading: skip generation entirely. Resumability path — the existing plan is authoritative; user-edited entries are preserved. To force regeneration, the user MUST delete `plan.md` before re-running `/fab-continue`.
2. **Otherwise**: invoke the **Plan Generation Procedure** (`_generation.md`). Write `plan.md` to the change folder. The PostToolUse hook updates `plan.generated`, `plan.task_count`, and `plan.acceptance_count` on `.status.yaml` automatically.
3. Apply MUST ignore the `## Acceptance` section during the main sub-step — that section is consumed by review.

### Pattern Extraction

Before executing the first unchecked task, read existing source files in the areas the change will touch and extract:

1. **Naming conventions** — variable/function/class naming style observed in surrounding code
2. **Error handling** — how the codebase handles errors (exceptions, Result types, error codes, etc.)
3. **Structure** — typical function length, module boundaries, import organization
4. **Reusable utilities** — existing helpers or shared modules that new code should use instead of reimplementing

Hold these patterns as context for all subsequent task execution within the same apply run.

If `fab/project/code-quality.md` exists, load its `## Principles` as additional implementation constraints alongside extracted patterns. If a `## Test Strategy` section is defined, it governs test timing (default: `test-alongside`).

**Skip on resume**: When resuming mid-apply (some tasks already `[x]`), pattern extraction is skipped — patterns are re-derived implicitly from reading task-relevant source files.

### Task Execution (main sub-step)

1. Parse the `## Tasks` section of `plan.md` — content between `## Tasks` and the next `## ` heading (typically `## Execution Order` or `## Acceptance`). The `## Acceptance` section is OUT OF SCOPE for apply.
2. If a top-level `## Execution Order` heading is present in `plan.md`, parse its body separately (content between `## Execution Order` and the next `## ` heading) and use it to constrain task ordering in step 4. If absent, infer ordering from phase/`[P]`-marker conventions alone.
3. Parse tasks: `- [ ]` = remaining, `- [x]` = skip
4. If all checked: run `fab status finish <change> apply fab-continue` (auto-activates review). Stop.
5. Execute in phase order; within phases, non-`[P]` sequential, `[P]` parallelizable. Respect Execution Order constraints parsed in step 2.
6. For each unchecked task:
   1. Read source files relevant to this task
   2. Implement per spec, constitution, and extracted patterns
   3. Prefer reusing existing utilities over creating new ones
   4. Keep functions focused — if implementation exceeds the codebase's typical function size, consider extracting
   5. Write tests per `fab/project/code-quality.md` test strategy (default: `test-alongside`)
   6. Run tests, fix failures
   7. Mark `[x]` immediately
7. On completion: run `fab status finish <change> apply fab-continue` (auto-activates review).

### Resumability

Plan Generation sub-step is skipped when `plan.md` already exists (idempotent on file presence). Task Execution starts from the first unchecked item; checked items assumed complete.

---

## Review Behavior

Follow **Review Behavior** (`_review.md`). The `_review.md` skill defines both sub-agent dispatches (inward + outward) run in parallel, their preconditions, validation steps, structured output format, and the findings merge procedure.

### Verdict

**Pass**: Run `fab status finish <change> review fab-continue` (auto-activates hydrate). Update acceptance progress via `fab status set-acceptance <change> acceptance_completed <N>`. Output report + `Next: {per state table}`.

**Fail** (manual rework — `/fab-continue` only): Run `fab status fail <change> review` then `fab status reset <change> apply fab-continue`. Update acceptance progress via `fab status set-acceptance <change> acceptance_completed <N>`. Present findings with priority annotations, then offer rework options:

| Option | When | Action |
|--------|------|--------|
| Fix code | Implementation bug (must-fix / should-fix items) | Uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: {reason} -->`, run `/fab-continue` |
| Revise plan | Missing/wrong tasks or acceptance items | Edit `plan.md` directly, run `/fab-continue` |
| Revise spec | Requirements wrong | Run `/fab-continue spec` to reset downstream |

The applying agent triages review comments by priority — not all comments need to be implemented. Must-fix items are always addressed. Should-fix items are addressed when clear and low-effort. Nice-to-have items may be acknowledged but deferred.

---

## Hydrate Behavior

### Preconditions

- `progress.review` MUST be `done`. If not: STOP.
- All items in `plan.md` `## Tasks` and `## Acceptance` MUST be `[x]`

### Steps

1. Final validation: all `## Tasks` and `## Acceptance` items in `plan.md` are `[x]`
2. Concurrent change check: warn on overlap with other changes referencing same memory paths
3. Hydrate `docs/memory/`: create new files/domains, update existing (Requirements, Design Decisions, Changelog), update indexes
4. Run `fab status finish <change> hydrate fab-continue`
5. **Pattern capture** *(optional)*: If the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section with the change name for traceability. Skip for implementations that follow existing patterns without introducing new ones

---

## Reset Flow (with stage argument)

1. **Validate**: Must be one of the 7 stage names. If `tasks` is passed, error with: `"tasks" stage was removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify spec.`
2. **Load context** for the target stage
3. **Reset `.status.yaml`**: Run `fab status reset <change> <stage> fab-continue`. This atomically sets the target stage → `active` and cascades all downstream stages → `pending`. Stages before the target are preserved.
4. **Execute**: Planning stages regenerate artifact. Execution stages re-run (task checkboxes NOT reset; `plan.md` is also preserved on disk — to force plan regeneration the user MUST delete `plan.md` before re-running `/fab-continue`).
5. **Invalidate downstream** (planning resets only): intake reset → all downstream pending; spec reset → apply pending. The `reset` command handles the status cascading automatically.
6. **Post-execution**: For **planning resets**, after regenerating the artifact, use `fab status advance <change> <stage>` to move the target stage back to `ready` and stop there — **do not** run `finish`, to avoid auto-activating the next pending stage. For **execution resets**, use the normal `finish` commands, which will auto-activate the next pending stage.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `spec.md` missing for apply entry | "No spec.md found. Run /fab-continue to generate the spec first." |
| `plan.md` missing `## Acceptance` for review | "plan.md missing Acceptance section." |
| Incomplete tasks for review | "{N} of {total} tasks incomplete." |
| Review not passed for hydrate | "Review has not passed." |
| Reset target `tasks` | `"tasks" stage was removed — use /fab-continue apply (regenerates plan.md and re-runs) or /fab-clarify spec.` |
| Unknown reset target | "Unknown stage. Valid: intake, spec, apply, review, hydrate, ship, review-pr." |
| Template file missing | "Template not found — kit may be corrupted." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Yes — planning regenerates, apply resumes, review re-validates |
| Modifies source code? | Yes — during apply |
| Modifies `docs/memory/`? | Yes — during hydrate |
| Moves change folder / removes `.fab-status.yaml`? | No — use `/fab-archive` |
