# fab-archive

## Summary

Archives a completed change (post-hydrate) or restores an archived change. Delegates all mechanical operations — move, index, backlog mark-done, and pointer — to the `fab change archive` CLI; the skill only formats the YAML output. Backlog marking is mechanical (exact change-ID match, flipped in place), not interactive.

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
│  ├─ Step 1: Run archive
│  │  └─ Bash: fab change archive <change>   (no --description)
│  │     └─ (derive description from intake title, move, update
│  │         index, mark backlog item done, clear pointer)
│  │
│  └─ Step 2: Format report (incl. backlog: field)
│
└── Restore Mode (/fab-archive restore <name> [--switch])
   │
   ├─ Bash: fab change restore <name> [--switch]
   └─ Format report from YAML output
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble |
| Bash | `fab preflight`, `fab change archive`, `fab change restore`, `fab change archive-list` |

### Sub-agents

None.
