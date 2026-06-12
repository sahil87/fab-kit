# fab-help

## Summary

Displays workflow overview and command reference — the `/fab-*`, `/git-*`, and `/docs-*` skills grouped by category, plus the `fab batch` operations and the companion packages (wt, idea). Delegates to `fab fab-help` Go subcommand. No context loading, no file modification.

## Flow

```
User invokes /fab-help
│
├─ Bash: fab log command "fab-help"
└─ Bash: fab fab-help
   └─ (reads kit version from cache, scans skill frontmatter, prints grouped help text)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab log command`, `fab fab-help` |

### Sub-agents

None.
