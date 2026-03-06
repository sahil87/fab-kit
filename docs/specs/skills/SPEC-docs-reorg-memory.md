# docs-reorg-memory

## Summary

Analyzes memory files for themes and suggests reorganization. Read-only unless user approves changes. Identifies up to 10 themes, proposes reorg plan.

## Flow

```
User invokes /docs-reorg-memory
│
├─ Pre-flight: docs/memory/index.md and domain files must exist
├─ Read: all memory files across all domains
├─ (identify themes, propose reorganization)
├─ (present plan, ask for approval)
│
└─ [if approved]
   ├─ Write/Edit: reorganized memory files
   └─ Edit: docs/memory/index.md, domain indexes
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | All memory files and indexes |
| Write/Edit | Reorganized files (only with approval) |

### Sub-agents

None.
