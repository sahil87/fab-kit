# fab-continue

## Summary

Advances through the 6-stage pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. Supports reset to a given stage (legacy `tasks`/`spec` targets error with a pointer to the `apply` and `intake` reset routes). Handles all six stages: intake (the only planning stage), apply (co-generates `plan.md` `## Requirements` + `## Tasks` + `## Acceptance` at entry then runs tasks), review (sub-agent), hydrate, ship (delegates to `/git-pr` behavior), and review-pr (delegates to `/git-pr-review` behavior).

**Failure recovery + idempotent reset** (260612-w7dp): a `review-pr`/`failed` dispatch row — keyed off `progress.review-pr == failed`, the same progress-map guard mechanism as the review row — re-executes `/git-pr-review` behavior (its Step 0 `start` accepts `failed → active` for review-pr; never `reset`, whose From-set `{done, ready, skipped}` excludes `failed`), so a failed PR review no longer falls through to "Change is complete." The ship and review-pr rows (incl. the failed row) pass the resolved change **explicitly** to `/git-pr`/`/git-pr-review` (`{name}` as the `<change>` argument — the explicit-arg contract); the ship and review-pr **`active`** rows key on `active` only — `ready` is not in either stage's AllowedStates — while the review-pr failed row keys on the progress map's `failed`. The Reset Flow handles all non-resettable target states (reset From-set `{done, ready, skipped}`): already-`active` → skip the call and proceed (re-running a reset is a state-wise no-op — Constitution III); `failed` → route via the matching failed dispatch row (`start` owns failed→active, review/review-pr only); `pending` → error with advance guidance. All recovery pointers are executable: the unexecutable `/fab-clarify intake` form is replaced by `/fab-continue intake` then argless `/fab-clarify` (argless is correct in fab-continue's own messages — the change reference of the current invocation is implied, active or `[change-name]` override, and an Error Handling note tells override users to re-run with the same `<change-name>`; cross-context sites like `_pipeline.md`'s stop guidance instead name the change in every command), with an explicit delete-`plan.md` note where plan regeneration is the intent; the `intake.md`-missing error points at `/fab-continue intake` instead of looping through plain `/fab-continue`. The Review Behavior call site reads `change_type` from `.status.yaml` and passes it in the inward sub-agent prompt per `_review.md`'s context contract.

**Per-stage model** (260613-l3ja): Review Behavior runs `fab resolve-agent review` **once** and applies the resolved model+effort to both reviewer sub-agents (inward + outward) and the merge (the Claude Code adapter is the Agent tool `model` param; resolution is provider-neutral — see `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution). When `/fab-continue` runs a stage **directly in the foreground**, the configured tier is **advisory only** — fab cannot switch the session model mid-run, so the skill MAY note the mismatch but MUST NOT switch.

**Helpers**: Declares `helpers: [_srad]` in frontmatter; `_generation` and `_review` are loaded **stage-conditionally** at point of use (apply entry / intake regeneration → `_generation`; Review Behavior entry → `_review`) per `_preamble.md` § Skill Helper Declaration stage-conditional loading. Hydrate/ship/review-pr invocations and apply-resumes load neither.

## Flow

```
User invokes /fab-continue [change-name] [stage]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ [if reset arg] Reset Flow
│  └─ Bash: fab status reset <change> <stage> fab-continue
│     (non-resettable target states handled first, 260612-w7dp —
│      reset From = {done, ready, skipped}: already-active → skip
│      the call, proceed (re-run is a no-op); failed → route via the
│      matching failed dispatch row (start owns failed→active);
│      pending → error with advance guidance)
│     └─ (cascades downstream to pending)
│
├─ Dispatch on current stage + state
│  (review-failed dispatch — 260611-szxd f019: progress.review == failed
│   [exhausted ff/fff rework or interrupted fail→reset] →
│   fab status reset <change> apply fab-continue, then present the
│   rework menu directly and stop for the user's choice — do NOT
│   re-run review; orchestrators re-running /fab-ff//fab-fff recover
│   via fab status start <change> review per _pipeline.md Resumability
│   instead — that autonomous path is theirs, not this skill's)
│  (review-pr-failed dispatch — 260612-w7dp: progress.review-pr ==
│   failed → re-execute /git-pr-review behavior; its Step 0 start
│   recovers failed→active — never reset, and never falls through
│   to "Change is complete.")
│
│  ┌─────────────────────────────────────────────────┐
│  │ INTAKE STAGE (the only planning stage)          │
│  │                                                 │
│  │  Read: templates, intake, memory files          │
│  │  (agent generates intake artifact via SRAD)     │
│  │  Write: intake.md                       ◄── HOOK CANDIDATE
│  │  (no scoring here — intake score is written by  │
│  │   /fab-new and /fab-clarify)                    │
│  │  Bash: fab status advance <stage>               │
│  │  (intake ready → finish intake — auto-activates │
│  │   apply; no start call)                         │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ APPLY STAGE                                     │
│  │                                                 │
│  │  Entry sub-step (skip if plan.md exists):       │
│  │    Read: intake.md, _generation.md              │
│  │    Write: plan.md                       ◄── HOOK CANDIDATE
│  │      (## Requirements + ## Tasks +              │
│  │       ## Acceptance, R#/T###/A-### IDs)         │
│  │      (under-spec → inline SRAD assumption)      │
│  │                                                 │
│  │  Main sub-step (Task Execution):                │
│  │    Read: plan.md ## Tasks, source files         │
│  │    (pattern extraction from neighboring files)  │
│  │    For each unchecked task:                     │
│  │      Read: relevant source files                │
│  │      Edit/Write: implementation files           │
│  │      Bash: run tests                            │
│  │      Edit: plan.md ## Tasks (mark [x])          │
│  │    Bash: fab status finish <change> apply       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW STAGE                                    │
│  │  (executes _review.md's Shared Review Dispatch  │
│  │   end-to-end; orchestration below)              │
│  │                                                 │
│  │  ┌──────────────────────────────────────────┐   │
│  │  │ SUB-AGENT (inward): Requirements/Accept. │   │
│  │  │  Validation (Agent tool, general-purpose)│   │
│  │  │  Read: standard subagent context,        │   │
│  │  │        plan.md (## Requirements +        │   │
│  │  │        ## Tasks + ## Acceptance),        │   │
│  │  │        source files, memory files        │   │
│  │  │  Bash: run tests                         │   │
│  │  │  Edit: plan.md ## Acceptance (mark [x])  │   │
│  │  │  Returns: must-fix/should-fix/nice-to-have   │
│  │  └──────────────────────────────────────────┘   │
│  │           ↕ parallel dispatch                   │
│  │  ┌──────────────────────────────────────────┐   │
│  │  │ SUB-AGENT (outward): Holistic Diff Review│   │
│  │  │  (Agent tool, general-purpose)           │   │
│  │  │  Receives: git diff + changed file list  │   │
│  │  │  Full repo read access                   │   │
│  │  │  Codex→Claude cascade (graceful no-op)  │   │
│  │  │  Returns: must-fix/should-fix/nice-to-have   │
│  │  └──────────────────────────────────────────┘   │
│  │                                                 │
│  │  Merge findings → single verdict set            │
│  │                                                 │
│  │  Pass:                                          │
│  │    Bash: fab status finish <change> review      │
│  │    Bash: fab status set-acceptance              │
│  │          <change> acceptance_completed N        │
│  │  Fail:                                          │
│  │    Bash: fab status fail <change> review        │
│  │    Bash: fab status reset <change> apply        │
│  │    (present rework options to user)             │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ HYDRATE STAGE                                   │
│  │                                                 │
│  │  Read: docs/memory/ files, intake.md            │
│  │  Write/Edit: docs/memory/{domain}/{file}.md     │
│  │    (with description: frontmatter; merge        │
│  │     without duplication — existing entries      │
│  │     for this change are updated in place)       │
│  │  Bash: fab memory-index — regenerates the root  │
│  │  (domains-only), domain, and sub-domain indexes │
│  │  Bash: fab status finish <change> hydrate       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ SHIP STAGE                                      │
│  │  (delegates to /git-pr behavior, passing the    │
│  │   resolved change as the explicit <change>      │
│  │   argument — 260612-w7dp)                       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW-PR STAGE                                 │
│  │  (delegates to /git-pr-review behavior, passing │
│  │   the resolved change as the explicit <change>  │
│  │   argument — 260612-w7dp; it                    │
│  │   routes all terminal paths through its Step 6  │
│  │   and runs its own transitions; finish or fail  │
│  │   only if the stage is still active after it    │
│  │   returns; timeout outcome: stage deliberately  │
│  │   left active — report and stop, no re-finish)  │
│  └─────────────────────────────────────────────────┘
│
└─ Output: summary + Next: line
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, templates, artifacts, source files, memory |
| Write | Plan (`plan.md`), memory files |
| Edit | Plan (mark `## Tasks` and `## Acceptance` items [x]), memory files |
| Bash | All `fab status` transitions, `fab preflight`, `fab memory-index`, test execution — no `fab score` (no scoring at any stage `/fab-continue` runs; intake scoring belongs to `/fab-new`/`/fab-clarify`) |
| Agent | Review validation sub-agent (general-purpose) |

### Sub-agents

| Agent | Stage | Purpose |
|-------|-------|---------|
| Inward review validation (`_review.md`) | review | `plan.md` validation (`## Requirements` + `## Tasks` + `## Acceptance`) with test execution — dispatched in parallel with outward |
| Outward diff review (`_review.md`) | review | Holistic diff review with full repo access via Codex→Claude cascade — dispatched in parallel with inward |

> Review Behavior reads `.claude/skills/_review/SKILL.md` (if not already loaded) and executes its **Shared Review Dispatch** end-to-end (Preconditions → Inward + Outward Sub-Agent Dispatch → Parallel Dispatch → Findings Merge) — `_review.md` is the single source of truth for sub-agent dispatch and findings merge. `fab-continue.md` retains the Verdict section (pass/fail state transitions, rework options).

> **Subagent rule** (f006): when the Apply/Review/Hydrate behavior sections are dispatched as subagents by `/fab-ff`/`/fab-fff`, the subagent skips the finish step / §Verdict and runs no `fab status` command — the orchestrator owns all status transitions. The ship dispatch row likewise only runs `finish <change> ship` if the stage is still `active` after `/git-pr` returns (git-pr finishes ship internally), and the review-pr row's Pass and Fail branches both carry the same only-if-still-active guard (git-pr-review's Step 6 runs its own finish/fail).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Plan generation | PostToolUse hook recomputes `plan.task_count`, `plan.acceptance_count`, sets `plan.generated=true` | After plan.md write (no scoring at apply — intake is authoritative) |
| Review pass | `fab status set-acceptance <change> acceptance_completed N` | After review validation |
