# git-rebase

## Summary

Fetches the latest main branch and rebases the current branch onto it. Guards against running on main/master and prompts the user if uncommitted changes are present. Does not modify fab state.

## Flow

```
User invokes /git-rebase
│
├─ Step 1: Bash: git rev-parse --is-inside-work-tree
│
├─ Step 2: Branch Guard
│  ├─ Bash: git branch --show-current
│  └─ [on main/master] → STOP
│
├─ Step 3: Check for Uncommitted Changes
│  ├─ Bash: git status --porcelain
│  └─ [has changes] → AskUserQuestion (stash or abort)
│     ├─ [abort] → STOP
│     └─ [stash] → git stash push -m "git-rebase: auto-stash before rebase"
│
├─ Step 4: Fetch and Rebase
│  ├─ Bash: git symbolic-ref refs/remotes/origin/HEAD (with main/master fallback)
│  ├─ Bash: git fetch origin {main_branch}
│  │  └─ [fetch fails] → report error, pop stash if applicable, STOP
│  ├─ Bash: git rebase origin/{main_branch}
│  │  └─ [conflict] → report, advise --continue/--abort, note stash, STOP
│  └─ [if stashed] → git stash pop
│     └─ [pop conflict] → report, STOP
│
└─ Step 5: Report
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | All git operations (fetch, rebase, stash, status) |
| AskUserQuestion | Prompt user on uncommitted changes (stash or abort) |

### Sub-agents

None.
