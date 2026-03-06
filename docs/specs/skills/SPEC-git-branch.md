# git-branch

## Summary

Creates or switches to the git branch matching the active or specified change. Falls back to creating a standalone branch if the argument doesn't match any change. Does not modify fab state.

## Flow

```
User invokes /git-branch [change-name]
│
├─ Step 1: Bash: git rev-parse --is-inside-work-tree
│
├─ Step 2: Resolve Change Name
│  ├─ Bash: fab change resolve "<change-name>"
│  └─ [if fails with explicit arg] standalone fallback
│
├─ Step 3: Derive Branch Name
│  └─ (resolved name or raw argument)
│
├─ Step 4: Context-Dependent Action
│  ├─ Bash: git branch --show-current
│  ├─ Bash: git rev-parse --verify "<branch>"
│  │
│  ├─ [already on target] → no-op
│  ├─ [target exists] → git checkout "<branch>"
│  ├─ [on main/master] → git checkout -b "<branch>"
│  └─ [on other branch]
│     ├─ [no upstream] → git branch -m "<branch>"
│     └─ [has upstream] → git checkout -b "<branch>"
│
└─ Step 5: Report
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab change resolve`, all git operations |

### Sub-agents

None.
