# fab-archive

## Summary

Archives a completed change (post-hydrate) or restores an archived change. Delegates all mechanical operations — move, index, backlog mark-done, and pointer — to the `fab change archive` CLI; the skill only formats the YAML output. Backlog marking is mechanical (exact change-ID match, flipped in place), not interactive.

Since 260611-szxd (f087) the skill file is a **single document**: mode detection and both argument lists are stated once at the top; archive mode is the default body; restore-specific content lives in a `## Restore Mode` section holding only its unique Behavior/Output/Error-Handling/Key-Properties. The restore pre-flight is preserved as mode-specific content — it **waives** the standard preflight and the hydrate guard (opposite of archive mode; restore applies to any archived change regardless of state).

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
