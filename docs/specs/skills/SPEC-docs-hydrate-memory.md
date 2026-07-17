# docs-hydrate-memory

## Summary

Hydrates `docs/memory/` from external sources (URLs, .md files), generates from codebase analysis, or **backfills** an existing tree's missing FKF frontmatter. Ingest mode fetches/reads sources and creates memory files. Generate mode scans for undocumented areas interactively. **Backfill mode** (third mode — `backfill` keyword or `/docs-reorg-memory` dispatch) re-scans `docs/memory/` for topic files lacking `description:` frontmatter and adds the FKF frontmatter (`type: memory` + `description:`) body-preserving (only prepends/edits leading frontmatter), migrating a pre-fab-kit hand-curated tree to the convention `fab memory-index` depends on. All three modes author to the FKF contract — the shipped normative extract at `$(fab kit-path)/reference/fkf.md` (260616-frlo; mirror of the dev-repo design doc `docs/specs/fkf.md`) — by reading the canonical memory-file shape from the shipped template `$(fab kit-path)/templates/memory.md` (260616-2fm8; the single source of truth, read on demand the same way `_generation.md`/`_intake.md` read `$(fab kit-path)/templates/intake.md`) rather than an inlined shape block: generate and ingest modes fill the template's full skeleton, and backfill takes the template's **frontmatter shape only** (it stays body-preserving — see Backfill Mode). New/updated/backfilled memory files carry the FKF frontmatter pair — `type: memory` (constant, §3.1) plus a curated `description:` one-liner (§3.2); an ingest **merge into an existing/legacy file missing `type: memory` stamps the constant in** so the merged file is FKF-conforming (§2/§3.1 require it on every memory file, stamped by every memory writer — not only on creation); ingest/generate files carry **no `## Changelog` section** (§3.3 — change history lives in the per-folder generated `log.md`, §6); memory↔memory cross-links use the bundle-relative `/...` form (§7). As of **260715-xu0k** all three modes state the `description:` **500-character one-liner cap** (§3.2 — a routing signal, not a summary; detail belongs in the body, `fab memory-index` warns over the cap) at each authoring step (ingest Step 3/4, generate Step 3, backfill Step 2), and the regen steps carry the **never-hand-merge pointer** (a generated `docs/memory/**/index.md`/`log.md` conflict is resolved by fixing topic files + re-running `fab memory-index` and taking its output wholesale — FKF §5). Indexes are regenerated mechanically by `fab memory-index` (the single index writer — never hand-edited). **Refuse-before-regen guard**: every `fab memory-index` regen step is preceded by `fab memory-index --check`; on **exit 2** (destructive loss — curated description→`—`, dropped tombstone row, or flattened grouping) the skill refuses to regenerate and points to `/docs-reorg-memory` — the orchestrator for all three tier-2 categories, which relocates tombstone rows itself and dispatches this skill's backfill mode for the descriptions (backfill alone does not relocate tombstones). As the primary pre-fab-kit-tree entry point, this protects the *first* regen of a legacy tree. The guard is a **no-op on born-compatible fab-kit trees** (provably never exit 2) — it is not dead code; it only fires on a pre-fab-kit tree reached via ingest/generate before backfill. (Backfill mode only *adds* frontmatter, never destroys, so by the time *it* regenerates the guard is already satisfied.) The `description:` frontmatter is the single hand-curated index field: a new domain or sub-domain gets a `description:`-only `index.md` stub created before `fab memory-index` runs, and the generator round-trips it. Placement follows the memory-tree shape SHOULD guidance (~5–12 files/folder, depth ≤3, sub-domain at a ≥8-file cluster; `_shared`/`_unsorted` width-exempt).

### Backfill Mode

Distinct from generate mode (which *creates* files from source-code gaps), backfill *adds frontmatter to existing* files. Properties:

- **Independent re-scan, no caller manifest**: backfill walks `docs/memory/` itself to find topic files (non-`index.md` `.md`) missing `description:` — it receives no file list from its caller. This holds for both the direct-user form (`/docs-hydrate-memory backfill`) and the reorg-dispatched form (reorg's prompt names the operation, "backfill this tree"). The loose seam between the two independently-invocable skills. Because there is no caller manifest, **backfill takes no extra arguments** — any positional argument after the `backfill` keyword is rejected (`backfill takes no arguments — it re-scans docs/memory/ itself.`); the reorg-dispatch form never supplies extras.
- **Synthesis, curated-row-preferring**: for each file, synthesize a one-line `description:` from the file's own content (Overview / first section / H1); where an existing curated index row maps file-by-file, prefer the curated text.
- **Body-preserving**: writes the FKF frontmatter (`type: memory` + `description:`) as the leading frontmatter block — taking the **frontmatter shape only** from `$(fab kit-path)/templates/memory.md`, never the template's body skeleton — and preserves the existing body byte-for-byte; never edits content; in particular it does **NOT** strip an existing `## Changelog` body (the strip of the 20 existing per-file changelogs is FKF migration Change 4, not backfill).
- **Idempotent**: files already carrying `description:` are skipped (so a second pass is a no-op); `type: memory` is stamped only when the frontmatter is added for the first time.
- **Index stubs**: creates missing domain/sub-domain `description:`-only `index.md` stubs (stub-before-index, per the Index Ownership model).
- **Caller-aware regen**: when dispatched by reorg (defer-regen signal in the prompt), it does NOT run `fab memory-index` (reorg runs it once at the end); when invoked directly, it runs `fab memory-index` as the final step like the other modes.
- **Scope**: pure frontmatter operation — does NOT detect/relocate tombstone rows, flatten groupings, move files, or strip existing `## Changelog` bodies; those structural concerns belong to `/docs-reorg-memory` (and the changelog strip to FKF Change 4).

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts (the `templates/memory.md` read + FKF frontmatter + no-`## Changelog` rule + shape bounds now stated once in ingest Step 3 and referenced from generate/backfill; refuse-before-regen guard and arg-classification reject strings compressed to pointers) and a `## Contents` TOC added; no behavioral change (Flow / Tools / Sub-agents unchanged).

**Present-truth authoring** (260717-3plm): all authoring paths now follow the FKF present-truth body-style rule (§3.3, amended by this change) and the no-change-ids-in-`description:` clarification (§3.2). Ingest's **merge into an existing file** (Step 3 item 4) rewrites the affected section to **current truth** rather than appending a change-keyed delta — superseded statements are removed, not narrated (no "renamed X→Y in {id}", "was `old.value`"); body provenance is citation-only (trailing `(change-id)` / `*Introduced by*`). Every authored `description:` (ingest create/merge, generate, backfill synthesis) is **free of change-ids** — a routing signal, not a provenance record; provenance citations live in the body. Generate/ingest bodies are written in present tense (no transition narration). Backfill stays body-preserving (it authors only frontmatter, so the change-id-free description rule is the only present-truth rule it applies). No Flow/Tools/Sub-agents change.

## Flow

```
User invokes /docs-hydrate-memory [sources...|folders...|backfill]   (or /docs-reorg-memory dispatch)
│
├─ Read: _preamble.md (Context Loading override: skips the always-load layer entirely)
├─ Pre-flight: docs/memory/ and index.md must exist
│
├── Ingest Mode (URLs or .md files) ─────────────────────
│  ├─ WebFetch/Read: source files
│  ├─ Read: $(fab kit-path)/templates/memory.md (canonical shape, on demand)
│  ├─ (identify domains and topics)
│  ├─ Write: docs/memory/{domain}/{file}.md (from template: FKF frontmatter
│  │         type: memory + description:; no ## Changelog; bundle-relative /... links)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index --check (refuse-before-regen: refuse on exit 2; no-op on
│           born-compatible trees) → fab memory-index (regenerates root, domain, sub-domain indexes)
│
├── Generate Mode (folders or no args) ──────────────────
│  ├─ Glob/Read: scan codebase
│  ├─ Read: $(fab kit-path)/templates/memory.md (canonical shape, on demand)
│  ├─ (interactive: present gap report)
│  ├─ Write: docs/memory/{domain}/{file}.md (from template: FKF frontmatter
│  │         type: memory + description:; no ## Changelog; bundle-relative /... links)
│  ├─ Write: new domain/sub-domain index.md stubs (description: only, before memory-index)
│  └─ Bash: fab memory-index --check (refuse-before-regen: refuse on exit 2; no-op on
│           born-compatible trees) → fab memory-index (regenerates root, domain, sub-domain indexes)
│
└── Backfill Mode (backfill keyword / reorg dispatch) ───
   ├─ Glob/Read: re-scan docs/memory/ itself (no caller manifest) for topic files missing description:
   ├─ Read: each file's own content (+ matching curated index row if any)
   ├─ Read: $(fab kit-path)/templates/memory.md — FRONTMATTER SHAPE ONLY (not the body skeleton)
   ├─ Edit: prepend FKF frontmatter (type: memory + description:; body byte-preserved, existing
   │        ## Changelog NOT stripped, template body skeleton NOT imposed); skip files already
   │        having description: (idempotent)
   ├─ Write: missing domain/sub-domain index.md stubs (description: only)
   └─ Bash: fab memory-index --check (refuse-before-regen, no-op on born-compatible) →
            fab memory-index — ONLY when invoked directly (reorg-dispatched ⇒ defer regen to reorg)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Sources, existing memory files, codebase, `$(fab kit-path)/templates/memory.md` (canonical memory-file shape — full skeleton for generate/ingest, frontmatter shape only for backfill) |
| Write/Edit | New memory files filled from `$(fab kit-path)/templates/memory.md` (FKF frontmatter — `type: memory` + `description:`, no `## Changelog`, bundle-relative `/...` links), `description:`-only index stubs, and backfilled leading FKF frontmatter on existing files (frontmatter shape only from the template; body preserved, existing `## Changelog` not stripped) |
| Bash | `fab memory-index --check` refuse-before-regen guard (refuse on exit 2; no-op on born-compatible trees) before each regen; `fab memory-index` to regenerate indexes (skipped in backfill mode when reorg-dispatched) |
| WebFetch | Fetch URL sources |
| Glob/Grep | Codebase scanning; backfill's independent `docs/memory/` re-scan |

### Sub-agents

None. (Backfill mode is itself dispatched as a general-purpose sub-agent by `/docs-reorg-memory` during its compatibility orchestration, but `/docs-hydrate-memory` spawns no sub-agents of its own.)
