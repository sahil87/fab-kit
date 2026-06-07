# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files) or generates from codebase analysis. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively. New/updated memory files carry a `description:` frontmatter one-liner; indexes are regenerated mechanically by `fab memory-index` (the single index writer — never hand-edited). Placement follows the memory-tree shape SHOULD guidance (~5–12 files/folder, depth ≤3, sub-domain at a ≥8-file cluster; `_shared`/`_unsorted` width-exempt).

## Flow

```
User invokes /docs-hydrate-memory [sources...|folders...]
│
├─ Read: _preamble.md (always-load layer — partial: skips config/constitution)
├─ Pre-flight: docs/memory/ and index.md must exist
│
├── Ingest Mode (URLs or .md files) ─────────────────────
│  ├─ WebFetch/Read: source files
│  ├─ (identify domains and topics)
│  ├─ Write: docs/memory/{domain}/{file}.md (with description: frontmatter)
│  └─ Bash: fab memory-index   (regenerates root + domain indexes)
│
└── Generate Mode (folders or no args) ──────────────────
   ├─ Glob/Read: scan codebase
   ├─ (interactive: present gap report)
   ├─ Write: docs/memory/{domain}/{file}.md (with description: frontmatter)
   └─ Bash: fab memory-index   (regenerates indexes)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase |
| Write | New memory files (with `description:` frontmatter) |
| Bash | `fab memory-index` to regenerate indexes |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning |

### Sub-agents

None.
