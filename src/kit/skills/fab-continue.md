---
name: fab-continue
description: "Advance the active change one pipeline stage — intake, apply, review, hydrate, ship, or review-pr — or reset to a given stage."
helpers: [_srad]
---

# /fab-continue [<change-name>] [<stage>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

> **Stage-conditional helpers** (see `_preamble.md` § Skill Helper Declaration): `_generation` and `_review` are deliberately NOT in this skill's frontmatter `helpers:`. Read `.claude/skills/_generation/SKILL.md` only when generating an artifact (apply entry with no `plan.md`, or intake-`active` regeneration), and `.claude/skills/_review/SKILL.md` only when entering Review Behavior. Hydrate, ship, review-pr, and apply-resumes need neither.

---

## Purpose

Advance through the 6-stage Fab pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. When called with a stage argument, resets to that stage and re-runs from there.

> **Per-stage model (foreground = advisory-only)**: per-stage model selection (`fab resolve-agent <stage>` → `agent.tiers`, see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution) is honored fully only on **orchestrated/sub-agent** dispatch — which is where `/fab-continue` consumes it: its Review Behavior resolves the reviewer sub-agents' profile. When `/fab-continue` itself runs a stage **directly in the foreground**, fab cannot switch the session model mid-run, so the configured tier is **advisory only** — the skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch. By design.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Passed to preflight as `$1` (see `_preamble.md` §2).
- **`<stage>`** *(optional)* — reset target: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. The legacy `tasks` and `spec` targets error with a pointer to the `apply` and `intake` reset routes (see Error Handling).

Both may be provided in any order. Stage names are treated as reset targets; all others as change-name overrides.

---

## Pre-flight

1. Classify arguments: stage name vs. change-name override (stage names take priority)
2. Run preflight per `_preamble.md` §2
3. Use preflight's `stage` and `progress` fields for all subsequent logic

---

## Normal Flow

### Step 1: Determine Current Stage

Dispatch on preflight's derived `stage` and `display_state`. If progress is `pending`, run `fab status start <change> <stage> fab-continue` before dispatching. **Review-failed dispatch**: if `progress.review` is `failed` (an exhausted `/fab-ff`/`/fab-fff` rework loop, or an interrupted fail→reset sequence), do NOT re-run review — use the `review`/`failed` row below: present the rework menu directly. (Orchestrator re-runs of `/fab-ff`/`/fab-fff` instead recover via `fab status start <change> review` per `_pipeline.md` Resumability — that autonomous path is theirs, not this skill's.) **Review-pr-failed dispatch**: if `progress.review-pr` is `failed` (a failed PR-review run — `gh` missing, no PR found, or a processing error), use the `review-pr`/`failed` row below: re-execute `/git-pr-review` behavior — a FAILED PR review MUST NOT fall through to the "all `done`" row and read as complete.

**State-based dispatch**: Intake is the only planning stage. `/fab-continue` consolidates work into a single invocation:
- **`ready`** (intake) → Finish intake (auto-activates apply), then execute apply (its entry sub-step generates `plan.md`, then runs tasks)
- **`active`** (intake) → Generate intake if missing and advance to `ready` (backward compat for interrupted generations)
- **`active`/`ready`** (execution) → Execute the stage's behavior and finish it

| Derived stage | State | Action |
|---------------|-------|--------|
| `intake` | `ready` | finish intake (auto-activates apply) → execute apply (apply's entry sub-step generates `plan.md` — including its `## Requirements` — then runs tasks) |
| `intake` | `active` | generate intake if missing (read `.claude/skills/_generation/SKILL.md` first — Intake Generation Procedure) → advance to `ready` |
| `apply` | `active`/`ready` | Execute apply (entry: generate `plan.md` if absent; main: run tasks) → on completion run `finish <change> apply fab-continue` (auto-activates review) |
| `review` | `active`/`ready` | Execute review → pass: run `finish <change> review fab-continue` (auto-activates hydrate). Fail: run `fail <change> review` then `reset <change> apply fab-continue` |
| `review` | `failed` | *(Keys on `progress.review == failed` via the guard above. Preflight does surface a parked failure — `display_stage`/`display_state` read `review`/`failed` via DisplayStage's failed tier — but the derived routing `stage` lands on the next pending stage, so the progress map is the reliable key.)* Run `reset <change> apply fab-continue` (the same post-fail reset the Verdict fail path runs — review cascades to `pending`, apply re-activates), then present the rework menu (Review Behavior § Verdict, **Fail** options table) directly and stop for the user's choice — do NOT re-run review first |
| `hydrate` | `active`/`ready` | Execute hydrate → run `finish <change> hydrate fab-continue` |
| `ship` | `active` | *(`ready` is unreachable — ship's AllowedStates are `{pending, active, done, skipped}`.)* Execute `/git-pr` behavior **with the resolved change as the explicit `<change>` argument** (`/git-pr {name}` — transient override; its branch guard verifies the checked-out branch) → git-pr finishes ship internally (its Step 4b); only if the stage is still `active` after it returns, run `finish <change> ship fab-continue` (auto-activates review-pr) |
| `review-pr` | `active` | *(`ready` is unreachable — review-pr's AllowedStates are `{pending, active, done, failed, skipped}`.)* Execute `/git-pr-review` behavior **with the resolved change as the explicit `<change>` argument** (`/git-pr-review {name}` — same transient-override + branch-guard contract) → it routes all terminal paths through its Step 6 and runs its own transitions. Pass/no-reviews: only if the stage is still `active` after it returns, run `finish <change> review-pr fab-continue`. Timeout (Copilot review requested but not yet available): the stage is deliberately left `active` — report and stop, no re-finish. Fail: only if the stage is still `active` after it returns (its Step 6 normally runs `fail` itself), run `fail <change> review-pr` |
| `review-pr` | `failed` | *(Keys on `progress.review-pr == failed` via the guard above — the same progress-map mechanism as the `review`/`failed` row.)* Re-execute `/git-pr-review` behavior **with the resolved change as the explicit `<change>` argument** — its Step 0 runs `fab status start <change> review-pr`, and the CLI's review-pr `start` transition accepts `failed → active`; from there it routes terminal paths through its Step 6 with the same only-if-still-active guards as the row above. Do NOT route through `reset` — reset's From-set is `{done, ready, skipped}` (excludes `failed`); the CLI would error |
| all `done` | — | Block: "Change is complete." |

### Step 2: Load Context

Load per `_preamble.md` layers. Stage-specific additions: intake loads memory files; apply loads intake + plan (if it already exists) + source code; review adds plan + memory; hydrate loads memory index + target files.

### Step 3: SRAD + Generation

**Intake only**: Apply SRAD (`_srad.md`, loaded via `helpers:`) before generating. Budget: 1-2 unresolved questions. Tentative decisions get `<!-- assumed: ... -->` markers. (Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md` `## Assumptions` — not as questions or markers.)

| Stage | Procedure |
|-------|-----------|
| apply | [Apply Behavior](#apply-behavior) — entry sub-step invokes **Plan Generation Procedure** (`_generation.md`, read at point of use), which co-generates `## Requirements` + `## Tasks` + `## Acceptance` |
| review | **Review Behavior** (`_review.md`, read at point of use) |
| hydrate | [Hydrate Behavior](#hydrate-behavior) |

**No scoring at any stage `/fab-continue` runs.** Intake scoring is authoritative and is performed by `/fab-new` / `/fab-clarify`; `/fab-continue` operates only at apply and later, where there is no scoring.

### Step 4: Update `.status.yaml`

Use event commands via CLI to update `.status.yaml`. The `finish` command handles the two-write transition atomically: `fab status finish <change> <completed-stage> fab-continue`. This sets `{completed}` → `done`, auto-activates the next pending stage, refreshes `last_updated`, and updates `stage_metrics`.

For other state changes, use the appropriate event command (driver is always optional):
- `fab status start <change> <stage> fab-continue` — pending → active (plus failed → active for review/review-pr only)
- `fab status advance <change> <stage>` — active → ready
- `fab status fail <change> <stage>` — active → failed (review/review-pr only)
- `fab status reset <change> <stage> fab-continue` — done/ready/skipped → active (cascades downstream to pending)

### Step 5: Output

Display summary. Include Assumptions summary for planning stages. End with `Next:` per state table in `_preamble.md`.

---

## Apply Behavior

Apply runs as **two sub-steps in a single skill invocation**: a Plan Generation entry sub-step that produces `plan.md`, followed by the Task Execution main sub-step.

> **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): do NOT run any `fab status` command — skip the finish steps below and return results only. The orchestrator owns all status transitions.

### Preconditions

- `intake.md` MUST exist (used as input to plan generation — requirements are derived from the intake design)

### Plan Generation (entry sub-step)

1. **If `plan.md` already exists** with at least a `## Tasks` heading: skip generation entirely. Resumability path — the existing plan is authoritative; user-edited entries are preserved. To force regeneration, the user MUST delete `plan.md` before re-running `/fab-continue`.
2. **Otherwise**: read `.claude/skills/_generation/SKILL.md` (if not already loaded), then invoke the **Plan Generation Procedure**. Write `plan.md` to the change folder. The PostToolUse hook updates `plan.generated`, `plan.task_count`, and `plan.acceptance_count` on `.status.yaml` automatically.
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
   2. Implement per the plan's `## Requirements`, constitution, and extracted patterns
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

Read `.claude/skills/_review/SKILL.md` (if not already loaded), then execute its **Shared Review Dispatch** end-to-end (Preconditions → Inward + Outward Sub-Agent Dispatch → Parallel Dispatch → Findings Merge). The `_review.md` skill defines both sub-agent dispatches (inward + outward) run in parallel, their preconditions, validation steps, structured output format, and the findings merge procedure. When dispatching the inward sub-agent, read `change_type` from the change's `.status.yaml` and pass it in the prompt per `_review.md`'s context contract (its Steps 7–8 skip condition keys on it).

> **Per-stage model resolution**: before dispatching the reviewer sub-agents, run `fab resolve-agent review` **once** and apply the resolved model/effort to BOTH reviewers (inward + outward) and the merge (empty model ⇒ omit/inherit; empty effort ⇒ omit) — per `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution. The Claude Code adapter is the Agent tool `model` parameter; the resolution itself is provider-neutral.

> **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): skip §Verdict entirely — do NOT run any `fab status` command; return the merged findings with pass/fail status only. The orchestrator owns the finish/fail/reset transitions (and the rework loop).

### Verdict

**Pass**: Run `fab status finish <change> review fab-continue` (auto-activates hydrate). Update acceptance progress via `fab status set-acceptance <change> acceptance_completed <N>`. Output report + `Next: {per state table}`.

**Fail** (manual rework — `/fab-continue` only): Run `fab status fail <change> review` then `fab status reset <change> apply fab-continue`. Update acceptance progress via `fab status set-acceptance <change> acceptance_completed <N>`. Present findings with priority annotations, then offer rework options:

| Option | When | Action |
|--------|------|--------|
| Fix code | Implementation bug (must-fix / should-fix items) | Uncheck affected tasks in `plan.md` `## Tasks` with `<!-- rework: {reason} -->`, run `/fab-continue` |
| Revise plan | Missing/wrong tasks or acceptance items | Edit `plan.md` directly, run `/fab-continue` |
| Revise requirements | Requirements wrong | Edit `plan.md` `## Requirements` plus the downstream `## Tasks`/`## Acceptance` it affects, then re-run `/fab-continue` (apply). For a fundamentally wrong intake, run `/fab-continue intake` first (resets to intake and regenerates it), refine via `/fab-clarify`, and delete `plan.md` so apply re-derives `## Requirements` from the revised intake. |

The applying agent triages review comments by priority — not all comments need to be implemented. Must-fix items are always addressed. Should-fix items are addressed when clear and low-effort. Nice-to-have items may be acknowledged but deferred.

---

## Hydrate Behavior

> **When invoked as a subagent** (dispatched by `/fab-ff`/`/fab-fff`): skip step 5 (the finish) — do NOT run any `fab status` command; return completion status only. The orchestrator owns the transition.

### Preconditions

- `progress.review` MUST be `done`. If not: STOP.
- All items in `plan.md` `## Tasks` and `## Acceptance` MUST be `[x]`

### Steps

1. Final validation: all `## Tasks` and `## Acceptance` items in `plan.md` are `[x]`
2. Concurrent change check: warn on overlap with other changes referencing same memory paths
3. **Read `## Deletion Candidates`** from `plan.md` when present — informational only. Hydrate MAY reference candidates in memory updates (e.g., a Design Decision noting follow-up cleanup). Hydrate MUST NOT generate or modify the section (generation is review's responsibility) and MUST treat an absent section as "no findings" without error
4. Hydrate `docs/memory/`: create new files/domains (each carrying a `description:` frontmatter one-liner), update existing (Requirements, Design Decisions, Changelog, keep `description:` accurate). **Merge without duplication**: before appending to a target memory file, check it for an existing entry referencing this change (by change name) and update that entry in place instead of appending a duplicate — the same "replaced in place (not duplicated)" contract as `docs-hydrate-memory.md` and `_review.md`'s `## Deletion Candidates`. Then run `fab memory-index` to regenerate the root (domains-only), domain, and sub-domain indexes — never hand-edit index rows or "Last Updated" cells. **Shape SHOULD guidance**: aim for ~5–12 files/folder, depth ≤3, introduce a sub-domain only for a cohesive ≥8-file cluster; `_shared/` and `_unsorted/` are width-exempt. Heed any non-fatal shape warnings `fab memory-index` prints (advisory only).
5. Run `fab status finish <change> hydrate fab-continue`
6. **Pattern capture** *(optional)*: If the change introduced non-obvious implementation patterns that future changes should follow (e.g., a new error handling approach, a reusable abstraction), note them in the relevant memory file's Design Decisions section with the change name for traceability. Skip for implementations that follow existing patterns without introducing new ones

---

## Reset Flow (with stage argument)

1. **Validate**: Must be one of the 6 stage names. If `tasks` or `spec` is passed, error with: `"tasks"/"spec" stages were removed — use /fab-continue apply to re-run apply (delete plan.md first to force regeneration), or /fab-continue intake then /fab-clarify to rework the intake.`
2. **Load context** for the target stage
3. **Reset `.status.yaml`**: Reset's From-set is `{done, ready, skipped}` — handle the non-resettable current states first:
   - Target already **`active`** (e.g., re-running an interrupted reset): skip this call — the state is already what the reset would produce; proceed directly to step 4 (re-running a reset is a state-wise no-op — Constitution III).
   - Target **`failed`** (review/review-pr only — no other stage can hold it): do NOT reset — `failed` recovery belongs to `start` (failed → active). Stop the Reset Flow and follow the matching `failed` dispatch row in Step 1 instead (review → post-fail reset + rework menu; review-pr → re-execute `/git-pr-review` behavior).
   - Target **`pending`**: error — `Stage '{stage}' has not run yet — nothing to reset. Run /fab-continue to advance to it.`
   Otherwise run `fab status reset <change> <stage> fab-continue`. This atomically sets the target stage → `active` and cascades all downstream stages → `pending`. Stages before the target are preserved.
4. **Execute**: Intake reset regenerates the intake artifact. Execution stages (apply onward) re-run (task checkboxes NOT reset; `plan.md` is also preserved on disk — to force plan regeneration the user MUST delete `plan.md` before re-running `/fab-continue`).
5. **Invalidate downstream**: intake reset → all downstream pending. The `reset` command handles the status cascading automatically.
6. **Post-execution**: For the **intake reset**, after regenerating the artifact, use `fab status advance <change> <stage>` to move intake back to `ready` and stop there — **do not** run `finish`, to avoid auto-activating apply. For **execution resets**, use the normal `finish` commands, which will auto-activate the next pending stage.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `intake.md` missing for apply entry | "No intake.md found. Run /fab-continue intake to regenerate the intake first." *(the intake reset route — plain `/fab-continue` would re-enter apply and hit this same error)* |
| `plan.md` missing `## Acceptance` for review | "plan.md missing Acceptance section." |
| Incomplete tasks for review | "{N} of {total} tasks incomplete." |
| Review not passed for hydrate | "Review has not passed." |
| Reset target `tasks` or `spec` | `"tasks"/"spec" stages were removed — use /fab-continue apply to re-run apply (delete plan.md first to force regeneration), or /fab-continue intake then /fab-clarify to rework the intake.` |
| Unknown reset target | "Unknown stage. Valid: intake, apply, review, hydrate, ship, review-pr." |
| Template file missing | "Template not found — kit may be corrupted." |

> Recovery commands in these messages are shown argless: the change reference of the current invocation is implied (active, or the `[change-name]` override). When this invocation targeted an override change, re-run the suggested command with the same `<change-name>`.

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Yes — planning regenerates, apply resumes, review re-validates, hydrate merges without duplication (existing entries for this change are updated in place) |
| Modifies source code? | Yes — during apply |
| Modifies `docs/memory/`? | Yes — during hydrate |
| Moves change folder / removes `.fab-status.yaml`? | No — use `/fab-archive` |
