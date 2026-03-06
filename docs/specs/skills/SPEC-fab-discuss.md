# fab-discuss

## Summary

Read-only context priming for exploratory discussion. Loads the always-load layer, shows orientation summary, signals readiness. No artifact generation, no stage advancement.

## Flow

```
User invokes /fab-discuss
│
├─ Read: 7 always-load files (config, constitution, context,
│        code-quality, code-review, memory index, specs index)
├─ Bash: fab resolve --folder (check for active change)
├─ Bash: fab log command "fab-discuss"
│
└─ Output: orientation summary
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | 7 project files |
| Bash | `fab resolve`, `fab log command` |

### Sub-agents

None.
