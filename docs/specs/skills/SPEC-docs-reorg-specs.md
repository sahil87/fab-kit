# docs-reorg-specs

## Summary

Analyzes spec files for themes and suggests reorganization. Read-only unless user approves. Same pattern as docs-reorg-memory but targeting `docs/specs/`.

## Flow

```
User invokes /docs-reorg-specs
│
├─ Pre-flight: docs/specs/index.md and spec files must exist
├─ Read: all spec files
├─ (identify themes, propose reorganization)
├─ (present plan, ask for approval)
│
└─ [if approved]
   ├─ Write/Edit: reorganized spec files
   └─ Edit: docs/specs/index.md
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | All spec files and index |
| Write/Edit | Reorganized files (only with approval) |

### Sub-agents

None.
