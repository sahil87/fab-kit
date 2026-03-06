# fab-switch

## Summary

Switches the active change by writing to `fab/current`. Lists available changes when called with no argument. Supports deactivation via `--blank`.

## Flow

```
User invokes /fab-switch [change-name] [--blank]
│
├─ Read: _preamble.md (config.yaml only)
│
├── No argument ─────────────────────────────────────────
│  ├─ Bash: fab change list
│  ├─ (display numbered list with stages)
│  └─ (wait for user selection)
│     └─ Bash: fab change switch "<selected>"
│
├── --blank ─────────────────────────────────────────────
│  └─ Bash: fab change switch --blank
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
