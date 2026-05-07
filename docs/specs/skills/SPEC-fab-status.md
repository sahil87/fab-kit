# fab-status

## Summary

Read-only status display. Shows change name, branch, stage progress (out of 7 total stages), plan progress (tasks + acceptance counts), confidence score, version drift warning, and next command suggestion.

## Flow

```
User invokes /fab-status [change-name]
│
├─ Bash: fab preflight [change-name]
├─ Read: src/kit/VERSION, fab/.kit-migration-version
├─ Bash: git branch --show-current
│
└─ Render status display
   ├─ Stage line: "Stage: {stage} ({n}/7) — {state}"
   ├─ Progress table (7 rows: intake, spec, apply, review, hydrate, ship, review-pr)
   ├─ Plan counts: "Tasks: {plan.task_count}", "Acceptance: {plan.acceptance_completed}/{plan.acceptance_count}"
   │  (or "Plan: not yet generated" when plan absent)
   ├─ Confidence line (from .status.yaml confidence block)
   └─ (agent formatting — no further tool calls)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab preflight`, `git branch --show-current` |
| Read | VERSION, migration-version |

### Sub-agents

None.
