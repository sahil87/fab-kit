# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files), generates from codebase analysis, or **backfills** an existing tree's missing FKF frontmatter. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively. **Backfill mode** (third mode — `backfill` keyword or `/docs-reorg-memory` dispatch) re-scans `docs/memory/` for topic files lacking `description:` frontmatter and adds the FKF frontmatter (`type: memory` + `description:`) body-preserving (only prepends/edits leading frontmatter), migrating a pre-fab-kit hand-curated tree to the convention `fab memory-index` depends on. All three modes author to the FKF contract (`docs/specs/fkf.md`): new/updated/backfilled memory files carry the FKF frontmatter pair — `type: memory` (constant, §3.1) plus a curated `description:` one-liner (§3.2); ingest/generate files carry **no `## Changelog` section** (§3.3 — change history lives in the per-folder generated `log.md`, §6); memory↔memory cross-links use the bundle-relative `/...` form (§7). Indexes are regenerated mechanically by `fab memory-index` (the single index writer — never hand-edited). **Refuse-before-regen guard**: every `fab memory-index` regen step is preceded by `fab memory-index --check`; on **exit 2** (destructive loss — curated description→`—`, dropped tombstone row, or flattened grouping) the skill refuses to regenerate and points to `/docs-reorg-memory` — the orchestrator for all three tier-2 categories, which relocates tombstone rows itself and dispatches this skill's backfill mode for the descriptions (backfill alone does not relocate tombstones). As the primary pre-fab-kit-tree entry point, this protects the *first* regen of a legacy tree. The guard is a **no-op on born-compatible fab-kit trees** (provably never exit 2) — it is not dead code; it only fires on a pre-fab-kit tree reached via ingest/generate before backfill. (Backfill mode only *adds* frontmatter, never destroys, so by the time *it* regenerates the guard is already satisfied.) The `description:` frontmatter is the single hand-curated index field: a new domain or sub-domain gets a `description:`-only `index.md` stub created before `fab memory-index` runs, and the generator round-trips it. Placement follows the memory-tree shape SHOULD guidance (~5–12 files/folder, depth ≤3, sub-domain at a ≥8-file cluster; `_shared`/`_unsorted` width-exempt).

### Backfill Mode

Distinct from generate mode (which *creates* files from source-code gaps), backfill *adds frontmatter to existing* files. Properties:

- **Independent re-scan, no caller manifest**: backfill walks `docs/memory/` itself to find topic files (non-`index.md` `.md`) missing `description:` — it receives no file list from its caller. This holds for both the direct-user form (`/docs-hydrate-memory backfill`) and the reorg-dispatched form (reorg's prompt names the operation, "backfill this tree"). The loose seam between the two independently-invocable skills. Because there is no caller manifest, **backfill takes no extra arguments** — any positional argument after the `backfill` keyword is rejected (`backfill takes no arguments — it re-scans docs/memory/ itself.`); the reorg-dispatch form never supplies extras.
- **Synthesis, curated-row-preferring**: for each file, synthesize a one-line `description:` from the file's own content (Overview / first section / H1); where an existing curated index row maps file-by-file, prefer the curated text.
- **Body-preserving**: writes the FKF frontmatter (`type: memory` + `description:`) as the leading frontmatter block and preserves the body byte-for-byte — never edits content; in particular it does **NOT** strip an existing `## Changelog` body (the strip of the 20 existing per-file changelogs is FKF migration Change 4, not backfill).
- **Idempotent**: files already carrying `description:` are skipped (so a second pass is a no-op); `type: memory` is stamped only when the frontmatter is added for the first time.
- **Index stubs**: creates missing domain/sub-domain `description:`-only `index.md` stubs (stub-before-index, per the Index Ownership model).
- **Caller-aware regen**: when dispatched by reorg (defer-regen signal in the prompt), it does NOT run `fab memory-index` (reorg runs it once at the end); when invoked directly, it runs `fab memory-index` as the final step like the other modes.
- **Scope**: pure frontmatter operation — does NOT detect/relocate tombstone rows, flatten groupings, move files, or strip existing `## Changelog` bodies; those structural concerns belong to `/docs-reorg-memory` (and the changelog strip to FKF Change 4).

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
│  ├─ Write: docs/memory/{domain}/{file}.md (FKF frontmatter: type: memory + description:;
│  │         no ## Changelog; bundle-relative /... links)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index --check (refuse-before-regen: refuse on exit 2; no-op on
│           born-compatible trees) → fab memory-index (regenerates root, domain, sub-domain indexes)
│
├── Generate Mode (folders or no args) ──────────────────
│  ├─ Glob/Read: scan codebase
│  ├─ (interactive: present gap report)
│  ├─ Write: docs/memory/{domain}/{file}.md (FKF frontmatter: type: memory + description:;
│  │         no ## Changelog; bundle-relative /... links)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index --check (refuse-before-regen: refuse on exit 2; no-op on
│           born-compatible trees) → fab memory-index (regenerates root, domain, sub-domain indexes)
│
└── Backfill Mode (backfill keyword / reorg dispatch) ───
   ├─ Glob/Read: re-scan docs/memory/ itself (no caller manifest) for topic files missing description:
   ├─ Read: each file's own content (+ matching curated index row if any)
   ├─ Edit: prepend FKF frontmatter (type: memory + description:; body byte-preserved, existing
   │        ## Changelog NOT stripped); skip files already having description: (idempotent)
   ├─ Write: missing domain/sub-domain index.md stubs (description: only)
   └─ Bash: fab memory-index --check (refuse-before-regen, no-op on born-compatible) →
            fab memory-index — ONLY when invoked directly (reorg-dispatched ⇒ defer regen to reorg)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase |
| Write/Edit | New memory files (with FKF frontmatter — `type: memory` + `description:`, no `## Changelog`, bundle-relative `/...` links), `description:`-only index stubs, and backfilled leading FKF frontmatter on existing files (body preserved, existing `## Changelog` not stripped) |
| Bash | `fab memory-index --check` refuse-before-regen guard (refuse on exit 2; no-op on born-compatible trees) before each regen; `fab memory-index` to regenerate indexes (skipped in backfill mode when reorg-dispatched) |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning; backfill's independent `docs/memory/` re-scan |

### Sub-agents

None. (Backfill mode is itself dispatched as a general-purpose sub-agent by `/docs-reorg-memory` during its compatibility orchestration, but `/docs-hydrate-memory` spawns no sub-agents of its own.)
