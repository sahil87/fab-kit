# docs-hydrate-specs

## Summary

Reverse hydration: identifies gaps where memory covers topics that specs don't. Proposes concise additions to specs with per-gap user confirmation. Top 3 gaps ranked by impact. When no existing spec is a suitable home for a gap, it proposes a new `docs/specs/{kebab-topic}.md` under the same per-gap confirmation; on yes it creates the file and adds its `docs/specs/index.md` row — the one index edit the skill makes.

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
│  ├─ (show exact markdown preview; no suitable target → propose new docs/specs/{kebab-topic}.md)
│  ├─ (ask user: yes / no / done — alias: skip rest)
│  ├─ yes, existing target → Edit: docs/specs/{file}.md
│  └─ yes, no-target gap → Write: docs/specs/{kebab-topic}.md
│                          + Edit: docs/specs/index.md (add row — the one index edit)
│
└─ (summary: {N} of {M} gaps applied)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Memory files, spec files, indexes |
| Edit | Spec files; spec index row when a no-target gap creates a new file |
| Write | New spec file for a confirmed no-target gap |

### Sub-agents

None.
