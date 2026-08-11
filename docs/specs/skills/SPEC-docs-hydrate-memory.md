# docs-hydrate-memory
Hydrates `docs/memory/` from external sources, generates from codebase analysis, or backfills missing FKF frontmatter (body-preserving). All modes author to the FKF contract via the shipped `templates/memory.md`.
## Flow
```
User invokes /docs-hydrate-memory [sources...|folders...|backfill]
├─ Read: _preamble.md; Pre-flight: docs/memory/ + index.md exist
├─ Ingest (URLs/.md): WebFetch/Read sources → Write topic files from template + index stubs → self-check → regen
├─ Generate (folders/none): Glob/Read codebase → gap report → Write from template + index stubs → self-check → regen
└─ Backfill (keyword / reorg dispatch): re-scan for missing description: → Edit: prepend frontmatter (body preserved, idempotent); regen deferred when reorg-dispatched
   (regen = Bash: fab memory-index --check refuse-guard → fab memory-index)
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read/Glob/Grep/WebFetch | Sources, codebase scan, memory re-scan, `templates/memory.md` shape |
| Write/Edit | New memory files + index stubs; backfilled frontmatter |
| Bash | `fab memory-index --check` refuse-guard; `fab memory-index` regen |
### Sub-agents
None — backfill mode is dispatched by `/docs-reorg-memory`; this skill spawns none.
