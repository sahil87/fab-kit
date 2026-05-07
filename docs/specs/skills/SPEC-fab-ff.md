# fab-ff

## Summary

Fast-forward through hydrate: intake through hydrate in one invocation. Three gates: (1) intake indicative confidence >= 3.0, (2) spec confidence >= per-type threshold, (3) review rework capped at 3 cycles. Resumable — re-running picks up from first incomplete stage. All sub-skill invocations dispatched as sub-agents. Accepts `--force` to bypass confidence gates (intake + spec).

**Helpers**: Declares `helpers: [_generation, _review]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-ff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Gate 1: Intake Gate (skip if --force)
│  └─ Bash: fab score --check-gate --stage intake <change>
│     └─ STOP if < 3.0
│
├─ Step 1: Generate spec.md
│  ├─ Bash: fab status finish <change> intake fab-ff
│  ├─ Read: templates, intake.md, memory files
│  ├─ Write: spec.md                                     ◄── HOOK CANDIDATE
│  ├─ Gate 2: Spec Gate (skip if --force)
│  │  └─ Bash: fab score --check-gate <change>
│  │     └─ STOP if below threshold
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-clarify [AUTO-MODE]      │
│     │  Read: spec.md                           │
│     │  (autonomous gap resolution)             │
│     │  Edit: spec.md                           │
│     │  Returns: {resolved, blocking, non_blocking} │
│     └──────────────────────────────────────────┘
│     └─ BAIL if blocking > 0
│
├─ Step 2: Implementation (apply, with internal plan generation)
│  ├─ Bash: fab status finish <change> spec fab-ff (auto-activates apply)
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Apply)         │
│     │  Entry sub-step (skip if plan.md exists):│
│     │    Read: spec.md, _generation.md         │
│     │    Write: plan.md            ◄── HOOK    │
│     │      (## Tasks + ## Acceptance)          │
│     │  ┌────────────────────────────────────┐  │
│     │  │ NESTED SUB-AGENT:                  │  │
│     │  │ /fab-clarify [AUTO-MODE] target=plan│ │
│     │  │  (autonomous gap resolution on    │  │
│     │  │   plan.md after generation)        │  │
│     │  │  Returns: {resolved, blocking,...} │  │
│     │  └────────────────────────────────────┘  │
│     │     └─ BAIL if blocking > 0              │
│     │  Main sub-step (Task Execution):         │
│     │    Read: plan.md ## Tasks, source files  │
│     │    Edit/Write: implementation files      │
│     │    Bash: run tests                       │
│     │    Edit: plan.md ## Tasks (mark [x])     │
│     │    Returns: completion status            │
│     └──────────────────────────────────────────┘
│  └─ Bash: fab status finish <change> apply fab-ff
│
├─ Step 3: Review (with auto-rework loop, max 3 cycles)
│  │  Review behavior is defined in `_review.md` (authoritative source
│  │  for inward + outward sub-agent dispatch and findings merge).
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Review)        │
│     │  Reads _review.md for dispatch:          │
│     │  ┌────────────────────────────────────┐  │
│     │  │ NESTED SUB-AGENT (inward):         │  │
│     │  │  Read: spec.md + plan.md + source  │  │
│     │  │  Bash: run tests                   │  │
│     │  │  Edit: plan.md ## Acceptance       │  │
│     │  │  Returns: findings                 │  │
│     │  └────────────────────────────────────┘  │
│     │  ┌────────────────────────────────────┐  │
│     │  │ NESTED SUB-AGENT (outward):        │  │
│     │  │  Receives: diff + changed files    │  │
│     │  │  Full repo access                  │  │
│     │  │  Codex→Claude cascade              │  │
│     │  │  Returns: findings                 │  │
│     │  └────────────────────────────────────┘  │
│     │  Merge findings → Returns: pass/fail     │
│     └──────────────────────────────────────────┘
│  ├─ Pass: Bash: fab status finish <change> review
│  └─ Fail: Auto-rework loop
│     ├─ Bash: fab status fail + reset
│     ├─ Triage findings → fix code / revise plan / revise spec
│     ├─ Re-dispatch apply + review sub-agents
│     ├─ Escalation rule: 2 consecutive fix-code → must escalate
│     └─ STOP after 3 failed cycles
│
├─ Step 4: Hydrate
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Hydrate)       │
│     │  Read/Write/Edit: docs/memory/ files     │
│     │  Bash: fab status finish <change> hydrate│
│     └──────────────────────────────────────────┘
│
└─ Pipeline complete.
```

### Sub-agents

| Agent | Step | Purpose |
|-------|------|---------|
| /fab-clarify [AUTO-MODE] | 1 (spec), 2 (plan) | Autonomous gap resolution after spec generation and after plan generation |
| /fab-continue (Apply) | 2 | Plan generation (entry sub-step) + task execution (main sub-step) |
| /fab-continue (Review) | 3 | Review orchestration — reads `_review.md` to dispatch inward + outward sub-agents in parallel; merges findings |
| /fab-continue (Hydrate) | 4 | Memory hydration |

> Step 3 review behavior (inward spec + plan validation and outward holistic diff review) is defined in `_review.md`. `/fab-continue` Review Behavior delegates to `_review.md`.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 1 | `fab score --check-gate` | After spec.md write |
| 2 | PostToolUse hook recomputes plan counts (`plan.task_count`, `plan.acceptance_count`, `plan.acceptance_completed`); sets `plan.generated=true` | After plan.md write/edit |
