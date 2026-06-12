# fab-archive

## Summary

Archives a completed change (post-hydrate) or restores an archived change. Delegates all mechanical operations — move, index, backlog mark-done, and pointer — to the `fab change archive` CLI; the skill only formats the YAML output. Backlog marking is mechanical (exact change-ID match, flipped in place), not interactive.

**Dirty-tree disclosure** (260612-g8st): "safe to re-run" covers fab state, not git state — both modes move tracked files and edit `fab/backlog.md` / the archive index with no commit step, leaving uncommitted changes for the caller to commit (commit ownership stays with `/git-pr`; no autonomous commit step). On the soft-skip path (re-archiving an already-archived change), `ArchiveWithBacklog` still re-attempts the idempotent backlog mark, so a re-run recovers a previously-failed mark — exit-code semantics unchanged (the adjacent exit-semantics seam belongs to change k4ge).

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
│  └─ Step 2: Format report (incl. backlog: field; index: failed
│      renders ✗ with a see-stderr pointer — the move already
│      succeeded, command exits non-zero — 260612-hv7t)
│
└── Restore Mode (/fab-archive restore <name> [--switch])
   │
   ├─ Bash: fab change restore <name> [--switch]
   └─ Format report from YAML output
      (pointer: switched | skipped | failed — `failed` means the restore
       completed but --switch could not create the symlink; the report
       points at /fab-switch {name} as manual recovery — 260612-k4ge.
       index: removed | not_found | failed — `failed` means the entry
       removal write failed; restore completed, exit non-zero — 260612-hv7t)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Preamble |
| Bash | `fab preflight`, `fab change archive`, `fab change restore`, `fab change archive-list` |

### Sub-agents

None.
