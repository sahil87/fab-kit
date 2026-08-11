# docs-reorg-specs
Analyzes spec files for themes and suggests reorganization; read-only until approval. `docs/specs/skills/SPEC-*.md` mirrors are reserved paths — read for theme analysis only, never renamed, moved, merged, or split.
## Flow
```
User invokes /docs-reorg-specs
├─ Pre-flight; Read: all spec files (recursing subfolders)
├─ Propose reorganization (never the reserved SPEC-*.md mirrors) → approval gate
└─ [approved] Write/Edit: moved files (bytes verbatim) + Edit: docs/specs/index.md (hand-rewritten)
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | All spec files and index |
| Write/Edit | Approved reorganizations |
### Sub-agents
None.
