# fab-help

## Summary

Displays workflow overview and command reference. Delegates to `fab fab-help` Go subcommand. No context loading, no file modification.

## Flow

```
User invokes /fab-help
│
├─ Bash: fab log command "fab-help"
└─ Bash: fab fab-help
   └─ (scans fab/.kit/skills/*.md frontmatter, prints grouped help text)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab log command`, `fab fab-help` |

### Sub-agents

None.
