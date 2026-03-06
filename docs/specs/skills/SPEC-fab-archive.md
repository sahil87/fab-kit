# fab-archive

## Summary

Archives a completed change (post-hydrate) or restores an archived change. Delegates mechanical operations to `fab change archive` CLI. Handles backlog matching interactively.

## Flow

```
User invokes /fab-archive [change-name]
│
├─ Read: _preamble.md (always-load layer)
├─ Bash: fab preflight [change-name]
├─ Guard: progress.hydrate must be done
│
├── Archive Mode ────────────────────────────────────────
│  │
│  ├─ Step 1: Extract description
│  │  └─ Read: fab/changes/{name}/intake.md
│  │
│  ├─ Step 2: Run archive
│  │  └─ Bash: fab change archive <change> --description "..."
│  │     └─ (clean .pr-done, move, update index, clear pointer)
│  │
│  ├─ Step 3: Backlog matching
│  │  ├─ Read: fab/backlog.md
│  │  ├─ (keyword scan, exact-ID match)
│  │  ├─ (interactive: confirm candidates)
│  │  └─ Edit: fab/backlog.md (mark [x], move to Done)
│  │
│  └─ Step 4: Format report
│
└── Restore Mode (/fab-archive restore <name> [--switch])
   │
   ├─ Bash: fab change restore <name> [--switch]
   └─ Format report from YAML output
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble, intake.md, backlog.md |
| Edit | backlog.md (mark items done) |
| Bash | `fab preflight`, `fab change archive`, `fab change restore`, `fab change archive-list` |

### Sub-agents

None.
