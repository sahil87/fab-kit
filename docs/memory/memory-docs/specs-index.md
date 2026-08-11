---
type: memory
description: "`docs/specs/` — pre-implementation specs, specs-vs-memory distinction, bootstrap/context integration, per-skill SPEC mirrors as structural quick-reference (Flow + Tools + Sub-agents) with mechanical `SPEC-{source-filename}.md` naming, complete coverage, and a narrowed sync rule (mirror updates only on flow / tool / sub-agent changes); mirrors are reserved paths for `docs-reorg-specs`; no specs-index generator — hand-rewritten index, no frontmatter backfill (no-symmetry note)"
---
# Specs Index

**Domain**: memory-docs

## Overview

`docs/specs/index.md` is the centralized index for pre-implementation specifications. It complements `docs/memory/index.md` (post-implementation truth) by providing a persistent home for design intent — what was planned, the "why" behind features.

## Requirements

### Specs vs Memory Distinction

Spec files are pre-implementation artifacts — what you planned. They capture conceptual design intent, high-level decisions, and the "why" behind features. Memory files are post-implementation artifacts — what actually happened, the authoritative source of truth for system behavior.

- `docs/specs/index.md` boilerplate clearly states spec files are pre-implementation / planning artifacts
- `docs/memory/index.md` boilerplate clearly states memory files are post-implementation / authoritative truth
- Both index files cross-reference each other with relative links

### Flat Structure

The specs index does not prescribe a domain-based directory hierarchy. Spec files may be organized by the human in any structure they choose. The index simply lists what exists.

### Human-Curated Ownership

Spec files are written and maintained by humans. No automated tooling creates or enforces structure in `docs/specs/`. `/docs-hydrate-specs` provides assisted reverse-hydration — it identifies structural gaps between memory and specs and proposes concise additions, but every insertion requires explicit user confirmation. Spec files remain human-curated.

### No Compatibility/Backfill Step for Specs (5ewp)

Specs are **out of scope** for the pre-fab-kit compatibility/frontmatter backfill that `/docs-reorg-memory` orchestrates (detect missing `description:` frontmatter → dispatch `/docs-hydrate-memory` backfill). The asymmetry is verified and intentional:

- **No specs-index generator** — there is no counterpart to `fab memory-index` (`internal/memoryindex`) for specs. A spec missing `description:` frontmatter breaks nothing downstream, because nothing generates the specs index from frontmatter.
- **Hand-rewritten index** — `docs/specs/index.md` is rewritten by hand (`docs-reorg-specs` Step 5), not regenerated. There is no compatibility contract for a backfill to satisfy.
- **Constitution VI** keeps specs human-curated, pre-implementation design intent. Adding a specs backfill would invent a non-problem and push specs toward the generated-index model the constitution rejects.

`docs-reorg-specs.md` carries an explicit one-line **no-symmetry note** stating all of the above, so a future contributor does not "fix the asymmetry" by adding a specs backfill step.

### Per-Skill SPEC Mirrors (`docs/specs/skills/`)

`docs/specs/skills/` holds one structural quick-reference SPEC per skill source file: title, a short (2–3 sentence) header, the Flow diagram, the Tools table, and the Sub-agents section — cross-referencing the skill source as canonical. The constitution requires a `src/kit/skills/*.md` change to update its corresponding `SPEC-*.md` mirror only when the change alters the skill's flow, tool usage, or sub-agent structure; prose-only skill edits do not trigger a mirror update.

- **Naming**: mechanical `SPEC-{source-filename}.md` — partials keep their leading underscore (`SPEC-_review.md`, `SPEC-_preamble.md`, `SPEC-_generation.md`).
- **Coverage is complete — no exclusions**: every `src/kit/skills/*.md` file carries a SPEC mirror, covering every user-invocable skill and every partial. That includes the behavioral partials (`_cli-agents.md` defines *procedures* — spawn / pre-send validation / delivery probe / peek / await) and the pure-reference ones (`_cli-fab.md`, `_cli-external.md`), whose mirrors index an external command surface rather than defining behavior. The coverage rule and the naming convention are documented in `docs/specs/skills.md` § New Skill Checklist (the SPEC-mirror item) — the single home for both, alongside the checklist's other integration points (frontmatter fields, preamble-read line, `helpers:` declaration, `Next:` line, Error Handling + Key Properties tables, skills.md mapping row, fabhelp.go help grouping).
- **Reserved paths for spec reorganization (d9rs)**: `docs-reorg-specs` treats the mirrors as constitution-pinned reserved paths — their names derive mechanically from their sources and the constitution pins the skill-edit ⇒ mirror-update rule (on structural changes), so the skill never proposes renaming, moving, merging, or splitting them (a Migration Map row targeting a reserved path is invalid). They may be *read* for theme analysis; the skill's Step 1 also recurses into `docs/specs/` subfolders (e.g., `skills/`, `findings/`).
- **`docs/specs/hooks.md` is a top-level spec, not a mirror** — it mirrors no skill source (there is no `src/kit/skills/hooks.md`), so it lives at the specs root rather than under `skills/`. (d9rs)

### Bootstrap Integration

`/fab-setup` creates `docs/specs/index.md` during structural bootstrap (after memory/index.md). The creation is idempotent — if the file already exists, setup skips it with a status message.

## Design Decisions

### SPEC Mirrors Are Reserved Paths in Spec Reorganization
**Decision**: `docs-reorg-specs` exempts `docs/specs/skills/SPEC-*.md` from reorganization — read for theme analysis only; any Migration Map row targeting a reserved path is invalid.
**Why**: Mirror names derive mechanically from their `src/kit/skills/` sources (`SPEC-{source-filename}.md`) and the constitution pins the skill-edit ⇒ mirror-update rule (triggered by flow, tool-usage, or sub-agent structure changes). A reorg that renamed, moved, merged, or split a mirror would break the mechanical naming contract and orphan the constitution rule — the mirror set's structure is owned by the source tree, not by theme analysis.
**Rejected**: Allowing migrations with link rewriting — the naming convention itself is the contract, not just the inbound links; rewriting links would preserve navigation while still breaking the source↔mirror correspondence.
*Introduced by*: 260612-d9rs-docs-reality-sweep

### Context Loading Integration

`docs/specs/index.md` is included in the "Always Load" context layer in `_preamble.md`, alongside `config.yaml`, `constitution.md`, and `docs/memory/index.md`. This gives every skill baseline awareness of the specs landscape.
