# docs-reorg-memory
Analyzes memory themes and tree shape, proposes reorganization; read-only until approval, then performs migrations, rewrites links, and regenerates indexes. Also orchestrates a one-time compatibility migration for pre-fab-kit trees.
## Flow
```
User invokes /docs-reorg-memory
├─ Pre-flight; Read: all memory files + one Bash: fab memory-index --check --json (feeds compatibility, shape, _unsorted/, completion chain)
├─ Diagnose: Shape Report + compatibility + duplicate coverage + _unsorted/ triage → propose reorg → approval gate
├─ [compat approved] Write: _shared/removed-domains.md; dispatch /docs-hydrate-memory backfill
└─ [approved] per migration: Write/Edit moves/splits/merges + link rewrites → Bash: fab memory-index → verify (no lost headings/dangling links) → Next: /docs-distill-memory with flagged counts
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | All memory files and indexes |
| Write/Edit | Approved moves/splits/merges, link rewrites, tombstone file, `_unsorted/` triage |
| Bash | One `fab memory-index --check --json` (four consumers); `fab memory-index` regen |
| Agent | Dispatch `/docs-hydrate-memory` backfill during compatibility orchestration |
### Sub-agents
`/docs-hydrate-memory` (backfill mode) — synthesizes `description:` frontmatter; defer-regen so reorg owns the single regen.
