# fab-continue

## Summary

Advances through the 8-stage pipeline one step at a time. Each invocation handles the current stage's work and transitions to the next. Supports reset to a given stage. Handles planning (spec, tasks), execution (apply), review (sub-agent), and hydrate.

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
│
│  ┌─────────────────────────────────────────────────┐
│  │ PLANNING STAGES (intake/spec/tasks)             │
│  │                                                 │
│  │  Bash: fab status finish <prev-stage>           │
│  │  Read: templates, intake, spec, memory files    │
│  │  (agent generates artifact via SRAD)            │
│  │  Write: spec.md / tasks.md / checklist.md   ◄── HOOK CANDIDATE
│  │                                                 │
│  │  [spec stage only]                              │
│  │  Bash: fab score <change>               ◄── bookkeeping
│  │                                                 │
│  │  [tasks stage]                                  │
│  │  Bash: fab status set-checklist ...     ◄── bookkeeping
│  │                                                 │
│  │  Bash: fab status advance <stage>               │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ APPLY STAGE                                     │
│  │                                                 │
│  │  Read: tasks.md, spec.md, source files          │
│  │  (pattern extraction from neighboring files)    │
│  │  For each unchecked task:                       │
│  │    Read: relevant source files                  │
│  │    Edit/Write: implementation files             │
│  │    Bash: run tests                              │
│  │    Edit: tasks.md (mark [x])                    │
│  │  Bash: fab status finish <change> apply         │
│  └─────────────────────────────────────────────────┘
│
│  ┌─────────────────────────────────────────────────┐
│  │ REVIEW STAGE                                    │
│  │                                                 │
│  │  ┌──────────────────────────────────────────┐   │
│  │  │ SUB-AGENT: Review Validation             │   │
│  │  │  (Agent tool, general-purpose)           │   │
│  │  │                                          │   │
│  │  │  Read: standard subagent context         │   │
│  │  │        (per _preamble.md)                │   │
│  │  │  Read: spec.md, tasks.md, checklist.md,  │   │
│  │  │        source files, memory files        │   │
│  │  │  Bash: run tests                         │   │
│  │  │  Edit: checklist.md (mark [x])           │   │
│  │  │  Returns: structured findings            │   │
│  │  └──────────────────────────────────────────┘   │
│  │                                                 │
│  │  Pass:                                          │
│  │    Bash: fab status finish <change> review      │
│  │    Bash: fab status set-checklist completed N   │
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
│  │  Edit: docs/memory/index.md, domain indexes     │
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
│  │  (delegates to /git-pr-review behavior)         │
│  └─────────────────────────────────────────────────┘
│
└─ Output: summary + Next: line
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, templates, artifacts, source files, memory |
| Write | Spec, tasks, checklist, memory files |
| Edit | Tasks (mark [x]), checklist (mark [x]), memory files |
| Bash | All `fab status` transitions, `fab score`, `fab preflight`, test execution |
| Agent | Review validation sub-agent (general-purpose) |

### Sub-agents

| Agent | Stage | Purpose |
|-------|-------|---------|
| Review validation | review | Fresh-perspective code review with spec/checklist/test checks |

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Spec generation | `fab score <change>` | After spec.md write |
| Tasks generation | `fab status set-checklist ... total N` | After tasks.md write |
| Tasks generation | `fab status set-checklist generated true` | After checklist.md write |
| Review pass | `fab status set-checklist completed N` | After checklist validation |
