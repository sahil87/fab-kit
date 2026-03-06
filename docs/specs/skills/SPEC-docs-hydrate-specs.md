# docs-hydrate-specs

## Summary

Reverse hydration: identifies gaps where memory covers topics that specs don't. Proposes concise additions to specs with per-gap user confirmation. Top 3 gaps ranked by impact.

## Flow

```
User invokes /docs-hydrate-specs [domain]
│
├─ Read: _preamble.md (always-load layer)
├─ Pre-flight: memory/index.md and specs/index.md must exist
│
├─ Read: all memory files across domains
├─ Read: all spec files
├─ (identify structural gaps: memory topics not in specs)
├─ (rank top 3 by impact)
│
├─ For each gap:
│  ├─ (show exact markdown preview)
│  ├─ (ask user: confirm/skip/modify)
│  └─ Edit: docs/specs/{file}.md
│
└─ Edit: docs/specs/index.md (if new files added)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Memory files, spec files, indexes |
| Edit | Spec files, spec index |

### Sub-agents

None.
