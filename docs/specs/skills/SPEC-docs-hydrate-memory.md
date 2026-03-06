# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files) or generates from codebase analysis. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively.

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
│  ├─ Write: docs/memory/{domain}/{file}.md
│  └─ Edit: docs/memory/{domain}/index.md, docs/memory/index.md
│
└── Generate Mode (folders or no args) ──────────────────
   ├─ Glob/Read: scan codebase
   ├─ (interactive: present gap report)
   ├─ Write: docs/memory/{domain}/{file}.md
   └─ Edit: docs/memory/index.md
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase |
| Write | New memory files |
| Edit | Indexes |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning |

### Sub-agents

None.
