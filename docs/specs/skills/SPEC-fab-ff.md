# fab-ff

## Summary

Fast-forward through hydrate: intake through hydrate in one invocation. Three gates: (1) intake indicative confidence >= 3.0, (2) spec confidence >= per-type threshold, (3) review rework capped at 3 cycles. Resumable — re-running picks up from first incomplete stage. All sub-skill invocations dispatched as sub-agents. Accepts `--force` to bypass confidence gates (intake + spec).

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
├─ Step 2: Generate tasks.md
│  ├─ Read: templates, spec.md
│  ├─ Write: tasks.md                                    ◄── HOOK CANDIDATE
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-clarify [AUTO-MODE]      │
│     │  Read: tasks.md                          │
│     │  (autonomous gap resolution)             │
│     │  Edit: tasks.md                          │
│     │  Returns: {resolved, blocking, non_blocking} │
│     └──────────────────────────────────────────┘
│     └─ BAIL if blocking > 0
│
├─ Step 3: Generate checklist.md
│  └─ Write: checklist.md                                ◄── HOOK CANDIDATE
│
├─ Step 4: Planning Complete
│  ├─ Bash: fab status finish <change> tasks fab-ff
│  ├─ Bash: fab status set-checklist generated true      ◄── bookkeeping
│  ├─ Bash: fab status set-checklist total <N>           ◄── bookkeeping
│  └─ Bash: fab status set-checklist completed 0         ◄── bookkeeping
│
├─ Step 5: Implementation
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Apply)         │
│     │  Read: tasks.md, spec.md, source files   │
│     │  Edit/Write: implementation files        │
│     │  Bash: run tests                         │
│     │  Edit: tasks.md (mark [x])               │
│     │  Returns: completion status              │
│     └──────────────────────────────────────────┘
│  └─ Bash: fab status finish <change> apply fab-ff
│
├─ Step 6: Review (with auto-rework loop, max 3 cycles)
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /fab-continue (Review)        │
│     │  ┌────────────────────────────────────┐  │
│     │  │ NESTED SUB-AGENT: Review validator │  │
│     │  │  Read: all artifacts + source      │  │
│     │  │  Bash: run tests                   │  │
│     │  │  Edit: checklist.md                │  │
│     │  │  Returns: findings                 │  │
│     │  └────────────────────────────────────┘  │
│     │  Returns: pass/fail + findings           │
│     └──────────────────────────────────────────┘
│  ├─ Pass: Bash: fab status finish <change> review
│  └─ Fail: Auto-rework loop
│     ├─ Bash: fab status fail + reset
│     ├─ Triage findings → fix code / revise tasks / revise spec
│     ├─ Re-dispatch apply + review sub-agents
│     ├─ Escalation rule: 2 consecutive fix-code → must escalate
│     └─ STOP after 3 failed cycles
│
├─ Step 7: Hydrate
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
| /fab-clarify [AUTO-MODE] | 1, 2 | Autonomous gap resolution after spec/tasks generation |
| /fab-continue (Apply) | 5 | Task execution |
| /fab-continue (Review) | 6 | Review validation (itself spawns a nested review sub-agent) |
| /fab-continue (Hydrate) | 7 | Memory hydration |

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 1 | `fab score --check-gate` | After spec.md write |
| 4 | `fab status set-checklist` (3 calls) | After checklist.md write |
