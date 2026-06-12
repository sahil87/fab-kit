# git-branch

## Summary

Creates or switches to the git branch matching the active or specified change. Falls back to creating a standalone branch if the argument doesn't match any change — but an *ambiguous* multi-match STOPs with the candidate list instead of creating a junk branch (260612-g8st). Remote-only target branches are checked out with `--track` rather than recreated as divergent locals; a dirty tree adds a non-blocking carried-over note to create/rename reports. Does not modify fab state.

## Flow

```
User invokes /git-branch [change-name]
│
├─ Step 1: Bash: git rev-parse --is-inside-work-tree
│
├─ Step 2: Resolve Change Name
│  ├─ Bash: fab change resolve "<change-name>"
│  └─ [if fails with explicit arg] branch on stderr (260612-g8st):
│     ├─ ["Multiple changes match"] → STOP with candidate list
│     │  (no branch created)
│     └─ ["No change matches" / other] → standalone fallback
│
├─ Step 3: Derive Branch Name
│  └─ (resolved name or raw argument)
│
├─ Step 4: Context-Dependent Action
│  │  (kept in sync with fab-new.md Step 11 via in-file comments)
│  ├─ Bash: git branch --show-current
│  ├─ Bash: git status --porcelain | wc -l → {dirty_count}
│  ├─ Bash: git rev-parse --verify "<branch>"
│  ├─ Bash: git rev-parse --verify "origin/<branch>"
│  │
│  ├─ [already on target] → no-op
│  ├─ [target exists locally] → git checkout "<branch>"
│  ├─ [target exists only on origin] → git checkout --track "origin/<branch>"
│  │  (never recreate a divergent local — 260612-g8st)
│  ├─ [on main/master] → git checkout -b "<branch>"
│  └─ [on other branch]
│     ├─ [no upstream] → rename guard:
│     │  Bash: fab change resolve "$(git branch --show-current)"
│     │  ├─ [resolves to no change OR to the SAME change being
│     │  │   branched (worktree placeholder)] → git branch -m "<branch>"
│     │  └─ [matches a different change] → git checkout -b "<branch>"
│     │     (other change's branch left intact; caveat: new
│     │      branch inherits the old change's HEAD)
│     └─ [has upstream] → git checkout -b "<branch>"
│
└─ Step 5: Report
   └─ create/rename rows with {dirty_count} > 0 append the non-blocking
      note: " — note: {N} uncommitted change(s) carried over from {old}"
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab change resolve` (argument resolution — stderr-keyed multi-match STOP vs no-match fallback — plus the Step 4 rename guard on the current branch), all git operations |

### Sub-agents

None.
