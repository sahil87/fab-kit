# Intake: Streamline Planning Stage Dispatch

**Change**: 260227-ijql-streamline-planning-dispatch
**Created**: 2026-02-27
**Status**: Draft

## Origin

> Streamline the fab-continue dispatch so planning stages don't require double invocation. fab-new should leave intake as ready (not active), and fab-continue should finish the previous ready stage + generate the next artifact + leave it at ready, all in one invocation.

Initiated from a `/fab-discuss` session. The user identified that the current state machine requires typing `/fab-continue` twice per planning stage — once to generate the artifact (`active` → `ready`) and once to advance (`ready` → `done` + next `pending` → `active`). This results in 6 invocations to get through planning when it should be 3. The solution was designed collaboratively through iterative discussion.

## Why

The current `/fab-continue` dispatch has a **single-dispatch rule** that executes exactly one action per invocation for planning stages — either generate an artifact OR advance to the next stage, never both. This means:

1. **Double invocation per stage**: Each planning stage (intake, spec, tasks) requires two `/fab-continue` calls — one to generate, one to advance. That's 6 invocations to get through planning instead of 3.
2. **Intake is the worst case**: The first `/fab-continue` for intake doesn't even generate anything — it just acknowledges `intake.md` exists (from `/fab-new`) and moves from `active` to `ready`. Pure ceremony.
3. **Inconsistency**: Apply, review, and hydrate already collapse their work into a single invocation. Only planning stages have this double-invoke tax.

If we don't fix this, users will continue to experience unnecessary friction in the most common workflow path. The `/fab-fff` and `/fab-ff` shortcuts exist partly as workarounds for this friction.

## What Changes

### 1. `/fab-new` leaves intake as `ready` (not `active`)

Currently `/fab-new` generates `intake.md` and leaves intake as `active`. The first `/fab-continue` is a no-op that just runs `advance` to move it to `ready`.

After this change, `/fab-new` will call `stageman.sh advance` after generating the intake, so intake ends as `ready`. This means the artifact exists and the stage is open for `/fab-clarify` refinement, but the first `/fab-continue` can immediately finish intake and move to spec generation.

**Files**: `fab/.kit/skills/fab-new.md`

### 2. `/fab-continue` collapses ready→done + generate next in one invocation

The single-dispatch rule is replaced with a new dispatch model for planning stages:

- When the current stage is `ready`, `/fab-continue` finishes it (`done`), starts the next stage, generates the next artifact, and advances the next stage to `ready` — all in one invocation.
- The `ready` state remains as a refinement checkpoint — `/fab-clarify` can be run at any point while a stage is `ready`.
- Apply, review, and hydrate are unchanged (they already work in a single invocation).

New dispatch table:

| Derived stage | State | Action |
|---|---|---|
| `intake` | `ready` | finish intake → start spec → generate `spec.md` → advance spec to `ready` |
| `spec` | `ready` | finish spec → start tasks → generate `tasks.md` + checklist → advance tasks to `ready` |
| `tasks` | `ready` | finish tasks → start apply → execute tasks → finish apply |
| `apply` | `active`/`ready` | execute tasks → finish apply |
| `review` | `active`/`ready` | run review → pass: finish / fail: rework |
| `hydrate` | `active`/`ready` | hydrate memory → finish hydrate |
| `intake` | `active` | (backward compat) generate intake if missing, advance to `ready` |
| `spec` | `active` | (backward compat) generate `spec.md`, advance to `ready` |
| `tasks` | `active` | (backward compat) generate `tasks.md` + checklist, advance to `ready` |
| all `done` | — | Block: "Change is complete." |

**Files**: `fab/.kit/skills/fab-continue.md`

### 3. Template initializes `intake: pending` (not `active`)

The `status.yaml` template currently sets `intake: active`, but `changeman.sh` then calls `stageman.sh start intake fab-new` which expects `pending → active`. This fails silently — the folder and status file are created, but `stage_metrics` never gets populated for intake (no `started_at`, no `driver`, no `iterations`).

The fix: template sets all stages to `pending` (the pre-start state). `changeman.sh` calls `start intake` to properly transition to `active` with metrics. Then `/fab-new` (the skill) generates the artifact and calls `advance` to move to `ready`.

Clean responsibility separation:

| Layer | Responsibility |
|---|---|
| Template (`status.yaml`) | Initialize all stages as `pending` |
| Script (`changeman.sh`) | Create folder, init from template, `start intake` → `active` with metrics |
| Skill (`fab-new`) | Generate artifact, `advance intake` → `ready` |

**Files**: `fab/.kit/templates/status.yaml`

### 4. Update specs and diagrams

The spec files need to reflect the new dispatch model:

- `docs/specs/skills.md` — update the dispatch table, Next Steps table, and `/fab-continue` description
- `docs/specs/user-flow.md` — update the flow diagrams showing the user interaction pattern

**Files**: `docs/specs/skills.md`, `docs/specs/user-flow.md`

### Resulting user flow

```
/fab-new       →  generates intake.md, intake: active → ready
/fab-continue  →  finish intake, generate spec.md, spec: active → ready
/fab-continue  →  finish spec, generate tasks.md + checklist, tasks: active → ready
/fab-continue  →  finish tasks, execute tasks, apply: active → done, review: active
/fab-continue  →  run review, review: active → done, hydrate: active
/fab-continue  →  hydrate memory, hydrate: active → done
```

One `/fab-new` + five `/fab-continue`. Each invocation does meaningful work. Users can run `/fab-clarify` at any `ready` checkpoint before continuing.

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update `/fab-new` and `/fab-continue` dispatch behavior, remove single-dispatch rule
- `fab-workflow/execution-skills`: (modify) Minor update to reflect dispatch consistency between planning and execution stages
- `fab-workflow/change-lifecycle`: (modify) Update state transition examples to reflect `ready` as the default post-generation state

## Impact

- **`fab/.kit/templates/status.yaml`** — change `intake: active` → `intake: pending`
- **`fab/.kit/skills/fab-new.md`** — add `advance` call after intake generation
- **`fab/.kit/skills/fab-continue.md`** — rewrite dispatch table, remove single-dispatch rule, update planning stage behavior
- **`docs/specs/skills.md`** — update dispatch table, Next Steps table, `/fab-continue` description
- **`docs/specs/user-flow.md`** — update flow diagrams
- **No changes to**: `workflow.yaml` (state machine unchanged — `initial_state: active` describes the target of the first transition, not the template default), `stageman.sh` (events unchanged)

## Open Questions

None — the design was fully resolved during the `/fab-discuss` session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `/fab-new` leaves intake as `ready`, not `active` or `done` | Discussed — user explicitly chose `ready` to preserve `/fab-clarify` checkpoint | S:95 R:85 A:95 D:95 |
| 2 | Certain | `/fab-continue` finishes previous `ready` stage + generates next artifact + leaves next at `ready` in one invocation | Discussed — user explicitly confirmed this is the desired behavior | S:95 R:80 A:90 D:95 |
| 3 | Certain | `ready` state serves as refinement checkpoint for `/fab-clarify` | Discussed — user explicitly rejected `done` because it would require a reset to clarify | S:90 R:90 A:95 D:95 |
| 4 | Certain | No changes to `workflow.yaml` or `stageman.sh` | The existing state machine already supports all needed transitions | S:90 R:95 A:95 D:90 |
| 5 | Certain | Template initializes `intake: pending`, `changeman.sh start` transitions to `active` | Discussed — fixes pre-existing bug where `start` fails on `active` template default; clean layer separation | S:95 R:90 A:95 D:95 |
| 6 | Certain | Apply, review, hydrate behavior unchanged | These already work correctly in single invocations | S:90 R:95 A:95 D:95 |
| 7 | Confident | `active` rows in planning dispatch are backward-compat only | For changes interrupted mid-generation; normal path goes through `ready` | S:70 R:90 A:85 D:80 |
| 8 | Certain | This is a `refactor` change type | Restructuring dispatch behavior, no new functionality | S:90 R:95 A:95 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
