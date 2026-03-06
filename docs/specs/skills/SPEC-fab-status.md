# fab-status

## Summary

Read-only status display. Shows change name, branch, stage progress, checklist, confidence score, version drift warning, and next command suggestion.

## Flow

```
User invokes /fab-status [change-name]
│
├─ Bash: fab preflight [change-name]
├─ Read: fab/.kit/VERSION, fab/.kit-migration-version
├─ Bash: git branch --show-current
│
└─ Render status display
   └─ (agent formatting — no further tool calls)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab preflight`, `git branch --show-current` |
| Read | VERSION, migration-version |

### Sub-agents

None.
