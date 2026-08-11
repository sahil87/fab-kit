# docs-hydrate-specs
Reverse hydration: finds memory topics missing from specs, ranks top 3 gaps by impact, proposes additions with per-gap confirmation. A no-target gap proposes a new `docs/specs/{kebab-topic}.md` plus its `index.md` row.
## Flow
```
User invokes /docs-hydrate-specs [domain]
├─ Read: _preamble.md; pre-flight: memory + specs indexes exist
├─ Read: all memory + spec files → identify gaps, rank top 3
└─ Per gap: preview → yes/no/done → Edit: existing spec, or Write: new file + Edit: specs/index.md row; summary {N} of {M} applied
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | Memory files, spec files, indexes |
| Edit/Write | Approved spec edits; new spec file + index row for no-target gaps |
### Sub-agents
None.
