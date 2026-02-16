---
name: fab-continue
description: "Advance to the next pipeline stage — planning, implementation, review, or hydrate — or reset to a given stage."
---

# /fab-continue [<change-name>] [<stage>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Advance through the 6-stage Fab pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. When called with a stage argument, resets to that stage and re-runs from there.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of `fab/current`. Passed to preflight as `$1` (see `_context.md` §2).
- **`<stage>`** *(optional)* — reset target: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`.

Both may be provided in any order. Stage names are treated as reset targets; all others as change-name overrides.

---

## Pre-flight

1. Classify arguments: stage name vs. change-name override (stage names take priority)
2. Run preflight per `_context.md` §2
3. Log invocation: `lib/stageman.sh log-command <change_dir> "fab-continue" "<stage-arg-if-any>"`
4. Use preflight's `stage` and `progress` fields for all subsequent logic

---

## Normal Flow

### Step 1: Determine Current Stage

Dispatch on preflight's derived `stage`. If progress is `pending`, set to `active` before dispatching.

| Derived stage | Action |
|---------------|--------|
| `intake` | Generate `spec.md` → `intake: done`, `spec: active` |
| `spec` | Generate `tasks.md` + checklist → `spec: done`, `tasks: active` |
| `tasks` | Execute apply → `tasks: done`, `apply: active` → on completion `apply: done`, `review: active` |
| `apply` | Resume apply → on completion `apply: done`, `review: active` |
| `review` | Execute review → pass: `review: done`, `hydrate: active`. Fail: `review: failed`, `apply: active` |
| `hydrate` | Execute hydrate → `hydrate: done` |
| all `done` | Block: "Change is complete." |

### Step 2: Load Context

Load per `_context.md` layers. Stage-specific additions: planning stages load intake + memory files; apply loads spec + tasks + source code; review adds checklist + memory; hydrate loads memory index + target files.

### Step 3: SRAD + Generation

**Planning stages only**: Apply SRAD (`_context.md`) before generating. Budget: 1-2 unresolved questions per stage. Tentative decisions get `<!-- assumed: ... -->` markers.

| Stage | Procedure |
|-------|-----------|
| spec | **Spec Generation Procedure** (`_generation.md`) |
| tasks | **Tasks Generation Procedure** + **Checklist Generation Procedure** (`_generation.md`) |
| apply | [Apply Behavior](#apply-behavior) |
| review | [Review Behavior](#review-behavior) |
| hydrate | [Hydrate Behavior](#hydrate-behavior) |

**Spec stage only**: After spec generation, invoke `fab/.kit/scripts/lib/calc-score.sh $change_dir` to compute the confidence score. No scoring at other stages.

### Step 4: Update `.status.yaml`

Two-write transition via CLI: `lib/stageman.sh transition <file> <completed-stage> <next-stage> fab-continue`. This atomically sets `{completed}` → `done`, `{next}` → `active`, refreshes `last_updated`, and updates `stage_metrics`.

For single-state changes, use: `lib/stageman.sh set-state <file> <stage> <state> [fab-continue]` (driver required when state is `active`).

### Step 5: Output

Display summary. Include Assumptions summary for planning stages. End with `Next:` per state table in `_context.md`.

---

## Apply Behavior

### Preconditions

- `tasks.md` MUST exist
- If stage is `tasks`: run `lib/stageman.sh transition <file> tasks apply fab-continue` before starting

### Pattern Extraction

Before executing the first unchecked task, read existing source files in the areas the change will touch and extract:

1. **Naming conventions** — variable/function/class naming style observed in surrounding code
2. **Error handling** — how the codebase handles errors (exceptions, Result types, error codes, etc.)
3. **Structure** — typical function length, module boundaries, import organization
4. **Reusable utilities** — existing helpers or shared modules that new code should use instead of reimplementing

Hold these patterns as context for all subsequent task execution within the same apply run.

If `config.yaml` defines a `code_quality` section, load its `principles` as additional implementation constraints alongside extracted patterns. If `code_quality.test_strategy` is defined, it governs test timing (default: `test-alongside`).

**Skip on resume**: When resuming mid-apply (some tasks already `[x]`), pattern extraction is skipped — patterns are re-derived implicitly from reading task-relevant source files.

### Task Execution

1. Parse tasks: `- [ ]` = remaining, `- [x]` = skip
2. If all checked: run `lib/stageman.sh transition <file> apply review fab-continue`. Stop.
3. Execute in phase order; within phases, non-`[P]` sequential, `[P]` parallelizable. Respect Execution Order constraints.
4. For each unchecked task:
   1. Read source files relevant to this task
   2. Implement per spec, constitution, and extracted patterns
   3. Prefer reusing existing utilities over creating new ones
   4. Keep functions focused — if implementation exceeds the codebase's typical function size, consider extracting
   5. Write tests per `code_quality.test_strategy` (default: `test-alongside`)
   6. Run tests, fix failures
   7. Mark `[x]` immediately
5. On completion: run `lib/stageman.sh transition <file> apply review fab-continue`

### Resumability

Starts from first unchecked item. Checked items assumed complete.

---

## Review Behavior

### Preconditions

- `tasks.md` and `checklist.md` MUST exist
- All tasks MUST be `[x]`. If not: STOP with "{N} of {total} tasks are incomplete."

### Validation Steps

1. **Tasks complete**: All `[x]` in `tasks.md`
2. **Quality checklist**: Inspect code/tests per CHK item. Mark `[x]` if met, `[x] **N/A**: {reason}` if N/A, leave `[ ]` with reason if not met
3. **Run affected tests**: Scoped to touched modules/files
4. **Spot-check spec**: Verify key requirements and GIVEN/WHEN/THEN scenarios
5. **Memory drift check**: Compare implementation against referenced memory (warning only)
6. **Code quality check**: For each file modified during apply, verify:
   - Naming conventions consistent with surrounding code
   - Functions focused and appropriately sized
   - Error handling consistent with codebase style
   - Existing utilities reused where applicable
   - If `config.yaml` defines `code_quality.principles`, check each applicable principle
   - If `config.yaml` defines `code_quality.anti_patterns`, check for violations
   - Code quality issues are review failures with specific file:line references (same rework flow as spec mismatches)

### Verdict

**Pass**: Run `lib/stageman.sh transition <file> review hydrate fab-continue`. Run `lib/stageman.sh log-review <change_dir> "passed"`. Update checklist via `lib/stageman.sh set-checklist <file> completed <N>`. Output report + `Next: {per state table}`

**Fail**: Run `lib/stageman.sh set-state <file> review failed` then `lib/stageman.sh set-state <file> apply active fab-continue`. Run `lib/stageman.sh log-review <change_dir> "failed" "<rework-option>"` after user selects rework. Update checklist via `lib/stageman.sh set-checklist <file> completed <N>`. Output failure details + rework options:

| Option | When | Action |
|--------|------|--------|
| Fix code | Implementation bug | Uncheck affected tasks with `<!-- rework: {reason} -->`, run `/fab-continue` |
| Revise tasks | Missing/wrong tasks | Add/modify tasks, run `/fab-continue` |
| Revise spec | Requirements wrong | Run `/fab-continue spec` to reset downstream |

---

## Hydrate Behavior

### Preconditions

- `progress.review` MUST be `done`. If not: STOP.
- All tasks and checklist items MUST be `[x]`

### Steps

1. Final validation: all tasks and checklist `[x]`
2. Concurrent change check: warn on overlap with other changes referencing same memory paths
3. Hydrate `docs/memory/`: create new files/domains, update existing (Requirements, Design Decisions, Changelog), update indexes
4. Run `lib/stageman.sh set-state <file> hydrate done`
5. **Pattern capture** *(optional)*: If the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section with the change name for traceability. Skip for implementations that follow existing patterns without introducing new ones

---

## Reset Flow (with stage argument)

1. **Validate**: Must be one of the 6 stage names
2. **Load context** for the target stage
3. **Reset `.status.yaml`**: Use `lib/stageman.sh set-state <file> <stage> <state> [driver]` for each stage — target → `active` (with driver `fab-continue`), all after → `pending`, all before → preserved
4. **Execute**: Planning stages regenerate artifact. Execution stages re-run (task checkboxes NOT reset).
5. **Invalidate downstream** (planning resets only): intake reset → all downstream pending; spec reset → tasks pending; tasks reset → reset all `[x]` → `[ ]`, regenerate checklist
6. **Post-execution**: Planning resets set target to `done` but do NOT advance next to `active` (prevents auto-advancing into stale content). Execution resets use normal transitions.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `tasks.md` missing for apply | "No tasks.md found. Run /fab-continue to generate tasks first." |
| `checklist.md` missing for review | "No checklist found. Run /fab-continue to generate it first." |
| Incomplete tasks for review | "{N} of {total} tasks incomplete." |
| Review not passed for hydrate | "Review has not passed." |
| Unknown reset target | "Unknown stage. Valid: intake, spec, tasks, apply, review, hydrate." |
| Template file missing | "Template not found — kit may be corrupted." |

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Yes — planning regenerates, apply resumes, review re-validates |
| Modifies source code? | Yes — during apply |
| Modifies `docs/memory/`? | Yes — during hydrate |
| Moves change folder / clears `fab/current`? | No — use `/fab-archive` |
