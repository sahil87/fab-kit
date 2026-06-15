# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files), generates from codebase analysis, or **backfills** an existing tree's missing `description:` frontmatter. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively. **Backfill mode** (third mode — `backfill` keyword or `/docs-reorg-memory` dispatch) re-scans `docs/memory/` for topic files lacking `description:` frontmatter and adds it body-preserving (only prepends/edits frontmatter), migrating a pre-fab-kit hand-curated tree to the convention `fab memory-index` depends on. New/updated memory files carry a `description:` frontmatter one-liner; indexes are regenerated mechanically by `fab memory-index` (the single index writer — never hand-edited). The `description:` frontmatter is the single hand-curated index field: a new domain or sub-domain gets a `description:`-only `index.md` stub created before `fab memory-index` runs, and the generator round-trips it. Placement follows the memory-tree shape SHOULD guidance (~5–12 files/folder, depth ≤3, sub-domain at a ≥8-file cluster; `_shared`/`_unsorted` width-exempt).

### Backfill Mode

Distinct from generate mode (which *creates* files from source-code gaps), backfill *adds frontmatter to existing* files. Properties:

- **Independent re-scan, no caller manifest**: backfill walks `docs/memory/` itself to find topic files (non-`index.md` `.md`) missing `description:` — it receives no file list from its caller. This holds for both the direct-user form (`/docs-hydrate-memory backfill`) and the reorg-dispatched form (reorg's prompt names the operation, "backfill this tree"). The loose seam between the two independently-invocable skills. Because there is no caller manifest, **backfill takes no extra arguments** — any positional argument after the `backfill` keyword is rejected (`backfill takes no arguments — it re-scans docs/memory/ itself.`); the reorg-dispatch form never supplies extras.
- **Synthesis, curated-row-preferring**: for each file, synthesize a one-line `description:` from the file's own content (Overview / first section / H1); where an existing curated index row maps file-by-file, prefer the curated text.
- **Body-preserving**: writes `description:` as the leading frontmatter block and preserves the body byte-for-byte — never edits content.
- **Idempotent**: files already carrying `description:` are skipped, so a second pass is a no-op.
- **Index stubs**: creates missing domain/sub-domain `description:`-only `index.md` stubs (stub-before-index, per the Index Ownership model).
- **Caller-aware regen**: when dispatched by reorg (defer-regen signal in the prompt), it does NOT run `fab memory-index` (reorg runs it once at the end); when invoked directly, it runs `fab memory-index` as the final step like the other modes.
- **Scope**: pure frontmatter operation — does NOT detect/relocate tombstone rows, flatten groupings, or move files; those structural concerns belong to `/docs-reorg-memory`.

## Flow

```
User invokes /docs-hydrate-memory [sources...|folders...|backfill]   (or /docs-reorg-memory dispatch)
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
├── Generate Mode (folders or no args) ──────────────────
│  ├─ Glob/Read: scan codebase
│  ├─ (interactive: present gap report)
│  ├─ Write: docs/memory/{domain}/{file}.md (with description: frontmatter)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index   (regenerates root, domain, and sub-domain indexes)
│
└── Backfill Mode (backfill keyword / reorg dispatch) ───
   ├─ Glob/Read: re-scan docs/memory/ itself (no caller manifest) for topic files missing description:
   ├─ Read: each file's own content (+ matching curated index row if any)
   ├─ Edit: prepend description: frontmatter (body byte-preserved); skip files already having it (idempotent)
   ├─ Write: missing domain/sub-domain index.md stubs (description: only)
   └─ Bash: fab memory-index — ONLY when invoked directly (reorg-dispatched ⇒ defer regen to reorg)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase |
| Write/Edit | New memory files (with `description:` frontmatter), `description:`-only index stubs, and backfilled leading frontmatter on existing files (body preserved) |
| Bash | `fab memory-index` to regenerate indexes (skipped in backfill mode when reorg-dispatched) |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning; backfill's independent `docs/memory/` re-scan |

### Sub-agents

None. (Backfill mode is itself dispatched as a general-purpose sub-agent by `/docs-reorg-memory` during its compatibility orchestration, but `/docs-hydrate-memory` spawns no sub-agents of its own.)
