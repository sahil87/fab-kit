# Intake: Skip Tasks Stage for Simple Change Types

**Change**: 260423-xvaz-skip-tasks-simple-types
**Created**: 2026-04-23
**Status**: Draft

## Origin

> Discussion question: "Can we drop the 'tasks' step? How about we jump from spec to apply?"

Raised during a `/fab-discuss` workflow retrospective on 2026-04-23, after ~2 months of production use of fab-kit. The user asked whether `tasks.md` was earning its keep as a separate stage. Analysis across `src/kit/skills/` (fab-continue, fab-ff, fab-fff, _generation, _review) surfaced that `tasks.md`:

1. Is an LLM-compressed derivative of `spec.md`, consumed primarily by another LLM that re-reads `spec.md` anyway during apply (`fab-continue.md` Apply Behavior reads source files + spec + tasks).
2. Loses information in the compression — RFC 2119 `MUST`/`SHALL` requirements collapse into imperative bullets.
3. Is disproportionately the rework target: `fab-ff.md` Step 6 lists three rework paths (Fix code / Revise tasks / Revise spec), and "Revise tasks" is frequently hit as an intermediate step when "Fix code" fails twice. Signal: tasks are often the defect, not load-bearing scaffolding.
4. Adds a third generation round before any code runs (intake → spec → tasks), plus an auto-clarify pass between each.

**Related draft**: `260423-qszh-merge-tasks-checklist` — merges `tasks.md` + `checklist.md` into a single `plan.md`. Orthogonal but complementary. If that ships first, this change rephrases as "skip `plan.md` generation for simple types". If this ships first, that change operates only on types that still generate the artifact. Both can proceed independently; agents picking up one should read the other's intake before starting to understand interactions.

## Why

**Problem**: For ~50% of real-world changes (`fix`, `docs`, `chore`, `test`, `ci` types per `fab/project/config.yaml` change_types taxonomy), the tasks stage is ceremony. A typo fix or README update genuinely has one "task": make the edit. Breaking it into `T001 [P] docs/foo.md`, `T002 tests/foo.test.md` and running a full auto-clarify pass on that adds latency and zero quality.

**Consequence if unfixed**: Users adopt `/fab-ff --force` or bypass the workflow entirely for small changes, eroding the "always use the pipeline" discipline that makes the rest of fab's value (hydrate, review, memory) compound over time. Forced ceremony trains users to work around the tool.

**Why this approach over alternatives**:
- **Alternative A — always keep tasks**: status quo. Users paying tax on simple changes.
- **Alternative B — drop tasks entirely**: too aggressive. Complex `feat`/`refactor` changes genuinely benefit from a phased, resumable `[x]`-ledger (e.g., multi-session work spanning a day).
- **Alternative C — type-driven fast path** (chosen): keep tasks for types where it earns its keep (`feat`, `refactor`), skip for types where it doesn't. Configurable so teams with different norms can tune it.

## What Changes

### 1. Config key for tasks-generation policy

Add to `fab/project/config.yaml` schema:

```yaml
pipeline:
  generate_tasks:
    policy: auto      # auto | always | never
    skip_types:       # consulted only when policy = auto
      - fix
      - docs
      - chore
      - test
      - ci
```

Default (when absent from user config): `policy: auto` with the skip list above. `always` preserves current behavior. `never` always skips (e.g., for projects that consider tasks pure overhead).

### 2. Pipeline skip semantics

The `tasks` stage becomes optional. When skipped, the `.status.yaml` `progress.tasks` field transitions `pending → skipped` (new state), bypassing `active`/`ready`/`done`. Downstream consumers MUST treat `skipped` as equivalent to `done` for gating purposes (e.g., `progress.apply` auto-activates after `progress.spec: done` when tasks is skipped).

### 3. `fab-continue.md` dispatch table update

Current (abbreviated):

| Derived stage | State | Action |
|---------------|-------|--------|
| `spec` | `ready` | finish spec → start tasks → generate `tasks.md` + checklist → advance tasks to `ready` |
| `tasks` | `ready` | finish tasks → start apply → execute tasks → finish apply |

Proposed:

| Derived stage | State | Condition | Action |
|---------------|-------|-----------|--------|
| `spec` | `ready` | tasks skipped (per policy) | finish spec → **skip tasks** (set `progress.tasks: skipped`) → generate checklist → start apply → execute → finish apply |
| `spec` | `ready` | tasks generated | unchanged from today |
| `tasks` | `skipped` | — | not reachable by `/fab-continue` — apply runs instead |

### 4. Apply behavior without tasks.md

`fab-continue.md` § Apply Behavior currently requires `tasks.md`:

> Preconditions: `tasks.md` MUST exist

Change to:

> Preconditions: either `tasks.md` exists, OR `progress.tasks == skipped`. When tasks.md is absent, read `spec.md` directly and derive an ordered in-memory work plan (Pattern Extraction still runs). Work plan is not persisted.

Resume semantics (no checkboxes): on resume, re-read `spec.md` + run `git status` / `git diff` against the change's base branch to identify what has already been applied. Proceed with what's left. Coarser than task-level resume, but for types in the skip list (fix/docs/chore/test/ci), implementations are small enough that coarse resume is acceptable.

### 5. `fab-ff.md` / `fab-fff.md` skip logic

Current Step 2 (Generate `tasks.md`) becomes conditional:

```
### Step 2: Tasks (conditional)

Read change_type from .status.yaml and pipeline.generate_tasks policy from config.
If should_skip(change_type, policy):
  Run `fab status skip <change> tasks <driver>` (new CLI subcommand).
  Skip to Step 3 (Checklist).
Else:
  Generate tasks.md per existing procedure.
  Run auto-clarify on tasks.md.
```

Step 3 (Checklist) is unchanged — checklist always generates because review still depends on it. (Unless intake #2 `merge-tasks-checklist` ships first; coordinate.)

### 6. `fab status` CLI — new `skip` subcommand

```
fab status skip <change> <stage> [driver]
```

Semantics: stage must currently be `pending`. Transitions `pending → skipped`. Cannot skip `intake`, `spec`, `apply`, `review`, `hydrate` (not applicable — only `tasks` is skippable in this change; extensibility left for future).

Auto-activation: when the preceding stage `finish`es, if the next stage is `skipped`, auto-advance until a non-skipped stage is reached and `start` that.

### 7. `/fab-continue tasks` reset behavior

If user explicitly runs `/fab-continue tasks` on a change where tasks was skipped, this is an override request. Run the full tasks generation procedure and transition `progress.tasks: skipped → active` (treated as a downstream reset — cascade to apply/review/hydrate per existing reset rules).

### 8. Review behavior (unchanged)

`_review.md` preconditions currently: `tasks.md` and `checklist.md` MUST exist. Change to: `checklist.md` MUST exist; `tasks.md` MAY exist. Inward sub-agent's "Tasks complete: all `[x]` in `tasks.md`" check becomes "if tasks.md exists: all [x]; else: skip". All other review steps are unaffected.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) — document skipped stage, skip conditions, progress block semantics
- `fab-workflow/execution-skills`: (modify) — `/fab-continue`, `/fab-ff`, `/fab-fff` dispatch when tasks skipped
- `fab-workflow/planning-skills`: (modify) — conditional tasks generation
- `fab-workflow/configuration`: (modify) — new `pipeline.generate_tasks` config key
- `fab-workflow/schemas`: (modify) — `.status.yaml` adds `skipped` state value
- `fab-workflow/templates`: (modify) — note tasks.md is now optional
- `fab-workflow/model-tiers`: (modify, if applicable) — skip policy may affect cost/latency story

## Impact

### Code
- `src/kit/skills/fab-continue.md` — dispatch table, Apply Behavior preconditions, reset flow
- `src/kit/skills/fab-ff.md` — Step 2 conditional, Step 6 review preconditions
- `src/kit/skills/fab-fff.md` — Step 2 conditional, Step 6 review preconditions
- `src/kit/skills/_generation.md` — Tasks Generation Procedure gains preamble: "skipped for types per config"
- `src/kit/skills/_review.md` — precondition softening
- `src/kit/skills/_preamble.md` — State Table (maybe add a row for skipped tasks)

### Binary (`fab` Go CLI)
- `fab status skip` subcommand (new)
- `fab status finish` auto-advance past skipped stages
- `fab change list` / `fab-status` display of skipped stage
- `fab score` unchanged (scoring is spec-gated, tasks-independent)

### Templates & schema
- `src/kit/templates/status.yaml` — document `skipped` as a valid stage state
- `fab/project/config.yaml` schema — new `pipeline` block

### Specs
- `docs/specs/skills.md` — pipeline description per skill
- `docs/specs/overview.md` — 6-stage → "up to 6-stage" (or rephrase to avoid stage-count brittleness)
- `docs/specs/change-types.md` — skip-list defaults cross-referenced

### Migrations
- `src/kit/migrations/` — one migration file (Constitution § Additional Constraints: user-data restructuring MUST ship as a migration). Purpose: for any existing in-flight change with a stale `.status.yaml` schema that doesn't recognize `skipped`, backfill the field definition. Low blast radius — field is additive.

## Open Questions

- Should `fix` always skip, even for complex multi-file bug fixes? (Leaning: yes; if it's really a refactor-sized fix, user can override per change via a future flag, or simply run `/fab-continue tasks` post-spec.)
- Should `feat` types with intake word-count below a threshold also auto-skip? (Leaning: no — keeps policy type-only, avoids heuristic complexity.)
- Does the skipped state need to persist in hydrate records for memory auditability? (Leaning: no — hydrate records final memory state, not pipeline shape.)
- How does this interact with `/fab-clarify tasks`? If tasks was skipped, clarify should either (a) error "no tasks artifact" or (b) offer to generate tasks inline. Leaning: (a) with suggestion to `/fab-continue tasks` first.
- Is the `skipped` state value a new enum member on `.status.yaml` `progress.*`, or do we reuse `done` with a `skipped: true` sibling flag? Trade-off: new enum is cleaner for consumers; flag is additive and backwards-compatible with existing parsers.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change applies to fab-kit itself (src/kit/skills/, fab binary), not downstream user projects beyond new config key surface | Derived from Origin — this IS the fab-kit meta-project | S:95 R:95 A:95 D:95 |
| 2 | Certain | Constitution Principle III (idempotent operations) requires skipped stages be re-runnable via reset | Direct constitution quote | S:95 R:95 A:95 D:95 |
| 3 | Certain | Migrations MUST ship per Constitution Additional Constraints when user data schema changes | Direct constitution quote | S:95 R:95 A:95 D:95 |
| 4 | Confident | Default skip list: `fix`, `docs`, `chore`, `test`, `ci` | Matches `docs/specs/change-types.md` light-tier taxonomy; excludes `feat` and `refactor` which materially benefit from task phasing | S:80 R:75 A:80 D:75 |
| 5 | Confident | `skipped` is a new state value (not a `done+flag`) | Cleaner for the 5+ consumers (/fab-status, /fab-switch, fab change list, orchestrators, display tables). One-time additive migration cost. | S:75 R:65 A:80 D:70 |
| 6 | Confident | Checklist still generates even when tasks is skipped | Review depends on checklist; decoupling them makes this change smaller and non-breaking to review | S:80 R:80 A:85 D:75 |
| 7 | Confident | Apply without tasks reads spec.md + derives in-memory ordered plan (not persisted) | Simpler than persistent phase structure; acceptable because skip policy targets small changes | S:70 R:60 A:75 D:65 |
| 8 | Tentative | Resume semantics for apply without task checkboxes: re-read spec + `git diff` against base to detect applied work | No precedent in codebase; plausible but untested. Could instead require `--resume-all` flag for user consent. | S:55 R:50 A:60 D:50 |
| 9 | Tentative | `/fab-continue tasks` on a skipped change is treated as a reset (active → cascade downstream) | Consistent with existing reset behavior but may surprise users who expect a softer override | S:55 R:60 A:65 D:50 |
| 10 | Unresolved | Should the `skip_types` list default to `[fix, docs, chore, test, ci]` or be empty (opt-in) in the global default? | Defaulting includes 5 types in the new fast path immediately — high utility but a behavioral change. Empty default is safer but most users won't discover the optimization. | S:40 R:50 A:55 D:35 |
| 11 | Unresolved | Interaction with intake #2 (`260423-qszh-merge-tasks-checklist`): does this change target `tasks.md`, or `plan.md`'s Tasks section? | Depends on ship order. Both drafts are orthogonal until one lands; agent picking this up must check that change's status and rebase terminology accordingly. | S:40 R:45 A:50 D:40 |

11 assumptions (3 certain, 4 confident, 2 tentative, 2 unresolved).
