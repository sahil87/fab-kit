# fab-ff

## Summary

Fast-forward apply → review → hydrate (everything after intake) in one invocation. Two gates: (1) the single intake confidence gate (flat 3.0, all types), checked before the bracket; (2) review rework capped at 3 cycles. No `/fab-clarify` runs inside the bracket — clarification is intake-only. Resumable — re-running picks up from the first incomplete stage. All sub-skill invocations dispatched as sub-agents. Accepts `--force` to bypass the intake gate.

**Helpers**: Declares `helpers: [_generation, _review]` in frontmatter per `docs/specs/skills.md § Skill Helpers`.

## Flow

```
User invokes /fab-ff [change-name] [--force]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
│
├─ Gate: Intake Gate (skip if --force)
│  └─ Bash: fab score --check-gate --stage intake <change>
│     └─ STOP if < 3.0
│
├─ Step 1: Implementation (apply, with internal plan generation)
│  ├─ Bash: fab status finish <change> intake fab-ff (auto-activates apply)
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Apply)         │
│     │  Entry sub-step (skip if plan.md exists):│
│     │    Read: intake.md, _generation.md       │
│     │    Write: plan.md            ◄── HOOK    │
│     │      (## Requirements + ## Tasks +       │
│     │       ## Acceptance, co-generated)       │
│     │  (under-spec → inline SRAD assumption,   │
│     │   no clarify step)                        │
│     │  Main sub-step (Task Execution):         │
│     │    Read: plan.md ## Tasks, source files  │
│     │    Edit/Write: implementation files      │
│     │    Bash: run tests                       │
│     │    Edit: plan.md ## Tasks (mark [x])     │
│     │    Returns: completion status            │
│     └──────────────────────────────────────────┘
│  └─ Bash: fab status finish <change> apply fab-ff
│
├─ Step 2: Review (with auto-rework loop, max 3 cycles)
│  │  Review behavior is defined in `_review.md` (authoritative source
│  │  for inward + outward sub-agent dispatch and findings merge).
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Review)        │
│     │  Reads _review.md for dispatch:          │
│     │  ┌────────────────────────────────────┐  │
│     │  │ NESTED SUB-AGENT (inward):         │  │
│     │  │  Read: plan.md (## Requirements +  │  │
│     │  │   ## Tasks + ## Acceptance)+source │  │
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
│     ├─ Triage findings → fix code / revise plan / revise requirements
│     ├─ Re-dispatch apply + review sub-agents
│     ├─ Escalation rule: 2 consecutive fix-code → must escalate
│     └─ STOP after 3 failed cycles
│
├─ Step 3: Hydrate
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
| /fab-continue (Apply) | 1 | Plan co-generation (entry sub-step — ## Requirements + ## Tasks + ## Acceptance) + task execution (main sub-step). No clarify sub-agent. |
| /fab-continue (Review) | 2 | Review orchestration — reads `_review.md` to dispatch inward + outward sub-agents in parallel; merges findings |
| /fab-continue (Hydrate) | 3 | Memory hydration |

> Step 2 review behavior (inward requirements + acceptance validation and outward holistic diff review) is defined in `_review.md`. `/fab-continue` Review Behavior delegates to `_review.md`.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| pre | `fab score --check-gate --stage intake` | Before the bracket (intake gate) |
| 1 | PostToolUse hook recomputes plan counts (`plan.task_count`, `plan.acceptance_count`, `plan.acceptance_completed`); sets `plan.generated=true` | After plan.md write/edit |
