# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files) or generates from codebase analysis. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively. New/updated memory files carry a `description:` frontmatter one-liner; indexes are regenerated mechanically by `fab memory-index` (the single index writer — never hand-edited). The `description:` frontmatter is the single hand-curated index field: a new domain or sub-domain gets a `description:`-only `index.md` stub created before `fab memory-index` runs, and the generator round-trips it. Placement follows the memory-tree shape SHOULD guidance (~5–12 files/folder, depth ≤3, sub-domain at a ≥8-file cluster; `_shared`/`_unsorted` width-exempt).

## Flow

```
User invokes /docs-hydrate-memory [sources...|folders...]
│
├─ Read: _preamble.md (Context Loading override: skips the always-load layer entirely)
├─ Pre-flight: docs/memory/ and index.md must exist
│
├── Ingest Mode (URLs or .md files) ─────────────────────
│  ├─ WebFetch/Read: source files
│  ├─ (identify domains and topics)
│  ├─ Write: docs/memory/{domain}/{file}.md (with description: frontmatter)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index   (regenerates root, domain, and sub-domain indexes)
│
└── Generate Mode (folders or no args) ──────────────────
   ├─ Glob/Read: scan codebase
   ├─ (interactive: present gap report)
   ├─ Write: docs/memory/{domain}/{file}.md (with description: frontmatter)
   ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
   └─ Bash: fab memory-index   (regenerates root, domain, and sub-domain indexes)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase |
| Write | New memory files (with `description:` frontmatter) and `description:`-only index stubs for new domains/sub-domains |
| Bash | `fab memory-index` to regenerate indexes |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning |

### Sub-agents

None.
