# fab-switch

## Summary

Switches the active change by creating the `.fab-status.yaml` symlink. Lists available changes when called with no argument. Supports deactivation via `--none`.

## Flow

```
User invokes /fab-switch [change-name] [--none]
│
├─ Read: _preamble.md (config.yaml only)
│
├── No argument ─────────────────────────────────────────
│  ├─ Bash: fab change list
│  ├─ (display numbered list with stages)
│  └─ (wait for user selection)
│     └─ Bash: fab change switch "<selected>"
│
├── --none ─────────────────────────────────────────────
│  └─ Bash: fab change switch --none
│
└── change-name ─────────────────────────────────────────
   ├─ Bash: fab change switch "<change-name>"
   │  ├─ [if multiple match] display options, ask user
   │  └─ [if no match] list available changes
   └─ Bash: fab log command "fab-switch"
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab change switch`, `fab change list`, `fab log command` |

### Sub-agents

None.
