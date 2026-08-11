# git-branch
Creates or switches to the branch matching the active or specified change; unmatched explicit names fall back to a standalone branch, ambiguous matches STOP with candidates. Does not modify fab state.
## Flow
```
User invokes /git-branch [change-name]
├─ Bash: git rev-parse --is-inside-work-tree; fab change resolve "<name>" → [multi-match] STOP with candidates / [no match, explicit arg] standalone fallback
├─ Probes (current branch, dirty count, local + origin existence) → [on target] no-op / [local] checkout / [origin-only] checkout --track / [on main] checkout -b
├─ [other branch, no upstream] rename guard: resolve current branch → branch -m (same/no change) or checkout -b (different change)
└─ Report; create/rename with dirty tree → carried-over note
```
### Tools used
| Tool | Purpose |
|------|---------|
| Bash | `fab change resolve` (resolution + rename guard; strict exit-code form kept deliberately); all git operations |
### Sub-agents
None.
