# fab-help

## Summary

Displays workflow overview and command reference. Delegates entirely to a shell script. No context loading, no file modification.

## Flow

```
User invokes /fab-help
│
├─ Bash: fab log command "fab-help"
└─ Bash: bash fab/.kit/scripts/fab-help.sh
   └─ (reads fab/.kit/VERSION, prints help text)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Bash | `fab log command`, `fab-help.sh` |

### Sub-agents

None.
