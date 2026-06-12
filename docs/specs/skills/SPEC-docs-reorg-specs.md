# docs-reorg-specs

## Summary

Analyzes spec files for themes and suggests reorganization. Read-only unless user approves. Same pattern as docs-reorg-memory but targeting `docs/specs/`. Scanning recurses into subfolders (e.g. `skills/`, `findings/`), but `docs/specs/skills/SPEC-*.md` mirrors are **reserved paths** — constitution-pinned names derived from their `src/kit/skills/` sources — read for theme analysis only, never renamed/moved/merged/split.

## Flow

```
User invokes /docs-reorg-specs
│
├─ Pre-flight: docs/specs/index.md and spec files must exist
├─ Read: all spec files (recursing into subfolders: skills/, findings/)
├─ (identify themes, propose reorganization — never migrating reserved docs/specs/skills/SPEC-*.md)
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
