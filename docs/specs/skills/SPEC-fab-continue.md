# fab-continue

## Summary

Advances through the 6-stage pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. Supports reset to a given stage (legacy `tasks`/`spec` targets error with a pointer to `apply` / `/fab-clarify intake`). Handles intake (the only planning stage), execution (apply — co-generates `plan.md` `## Requirements` + `## Tasks` + `## Acceptance` at entry then runs tasks), review (sub-agent), and hydrate.

**Helpers**: Declares `helpers: [_generation, _review]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-continue [change-name] [stage]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ [if reset arg] Reset Flow
│  └─ Bash: fab status reset <change> <stage> fab-continue
│     └─ (cascades downstream to pending)
│
├─ Dispatch on current stage + state
│  (resume guard: progress.review == failed →
│   fab status start <change> review first)
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
│  │  (delegates to _review.md for sub-agent dispatch│
│  │   and findings merge; orchestration below)      │
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
│  │  Bash: fab memory-index (regenerates indexes)   │
│  │  Bash: fab status finish <change> hydrate       │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ SHIP STAGE                                      │
│  │  (delegates to /git-pr behavior)                │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW-PR STAGE                                 │
│  │  (delegates to /git-pr-review behavior — it     │
│  │   routes all terminal paths through its Step 6  │
│  │   and runs its own transitions; finish only if  │
│  │   the stage is still active after it returns;   │
│  │   timeout outcome: stage deliberately left      │
│  │   active — report and stop, no re-finish)       │
│  └─────────────────────────────────────────────────┘
│
└─ Output: summary + Next: line
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, templates, artifacts, source files, memory |
| Write | Spec, plan, memory files |
| Edit | Plan (mark `## Tasks` and `## Acceptance` items [x]), memory files |
| Bash | All `fab status` transitions, `fab score`, `fab preflight`, test execution |
| Agent | Review validation sub-agent (general-purpose) |

### Sub-agents

| Agent | Stage | Purpose |
|-------|-------|---------|
| Inward review validation (`_review.md`) | review | Spec + plan.md validation (`## Tasks` + `## Acceptance`) with test execution — dispatched in parallel with outward |
| Outward diff review (`_review.md`) | review | Holistic diff review with full repo access via Codex→Claude cascade — dispatched in parallel with inward |

> Review Behavior is delegated to `_review.md` (single source of truth for sub-agent dispatch and findings merge). `fab-continue.md` retains the Verdict section (pass/fail state transitions, rework options).

> **Subagent rule** (f006): when the Apply/Review/Hydrate behavior sections are dispatched as subagents by `/fab-ff`/`/fab-fff`, the subagent skips the finish step / §Verdict and runs no `fab status` command — the orchestrator owns all status transitions. The ship dispatch row likewise only runs `finish <change> ship` if the stage is still `active` after `/git-pr` returns (git-pr finishes ship internally).

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Plan generation | PostToolUse hook recomputes `plan.task_count`, `plan.acceptance_count`, sets `plan.generated=true` | After plan.md write (no scoring at apply — intake is authoritative) |
| Review pass | `fab status set-acceptance <change> acceptance_completed N` | After review validation |
