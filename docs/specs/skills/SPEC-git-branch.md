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
│     ├─ [no upstream] → rename guard:
│     │  Bash: fab change resolve "$(git branch --show-current)"
│     │  ├─ [resolves to no change] → git branch -m "<branch>"
│     │  └─ [matches another change] → git checkout -b "<branch>"
│     │     (other change's branch left intact; caveat: new
│     │      branch inherits the old change's HEAD)
│     └─ [has upstream] → git checkout -b "<branch>"
│
└─ Step 5: Report
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab change resolve` (argument resolution + the Step 4 rename guard on the current branch), all git operations |

### Sub-agents

None.
