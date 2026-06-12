# fab-switch

## Summary

Switches the active change by creating the `.fab-status.yaml` symlink. Lists available changes when called with no argument. Supports deactivation via `--none`.

The status summary printed by `fab change switch` ends with `Next: {routing_stage} (via {default_command})`, where the command is the one that drives the routing stage (`intake`/`apply`/`review`/`hydrate` → `/fab-continue`, `ship` → `/git-pr`, `review-pr` → `/git-pr-review`), aligned with `/fab-status` and the `_preamble.md` state table; only when all stages are done/skipped does it collapse to `Next: /fab-archive` (post-review off-by-one fixed in 260612-k4ge). The `Stage:` line's `{state}` qualifier enumerates all six states the `display_state` derivation can emit — `active`, `failed`, `ready`, `done`, `skipped`, `pending` (260612-w7dp; the skill formerly documented only done/active/pending).

## Flow

```
User invokes /fab-switch [change-name] [--none]
│
├─ Read: _preamble.md (no always-load files — §1 exception; config not required)
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
