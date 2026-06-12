# fab-discuss

## Summary

Read-only context priming for exploratory discussion. Loads the always-load layer, shows orientation summary, signals readiness. No artifact generation, no stage advancement.

## Flow

```
User invokes /fab-discuss
│
├─ Read: always-load layer per _preamble.md §1 (full 7 files —
│        config, constitution, context, code-quality,
│        code-review, memory index, specs index)
├─ Bash: fab resolve --folder (check for active change)
├─ Read: fab/changes/{name}/.status.yaml (if active — derive current stage from
│        the progress map: the stage holding active or ready, or failed for a
│        parked review/review-pr; all done/skipped = change complete)
├─ Bash: fab log command "fab-discuss"
│
└─ Output: orientation summary
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | 7 always-load files; active change's `.status.yaml` for stage derivation |
| Bash | `fab resolve`, `fab log command` |

### Sub-agents

None.
