---
type: memory
description: "`docs/specs/` — pre-implementation specs, specs-vs-memory distinction, bootstrap/context integration; per-skill Flow skeletons live in `docs/specs/skills.md` (one per skill section, behavioral partials under § Skill Helpers); no specs-index generator — hand-rewritten index, no frontmatter backfill (no-symmetry note)"
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

### Per-Skill Flow Skeletons (in `skills.md`)

Each user-invocable skill's section in `docs/specs/skills.md` carries a condensed structural skeleton — the Flow diagram plus Tools and Sub-agents one-liners where they add information beyond the section's prose. Behavioral partials (`_preamble`, `_generation`, `_intake`, `_pipeline`, `_review`, `_srad`) carry theirs under § Skill Helpers § Partial Flow Skeletons; the pure-reference partials (`_cli-fab`, `_cli-external`, `_cli-agents`) carry none. Adding a skill folds its skeleton into its `skills.md` section (§ New Skill Checklist).

- **`docs/specs/hooks.md` is a top-level spec** — it corresponds to no skill source (there is no `src/kit/skills/hooks.md`), so it lives at the specs root. (d9rs)

### Bootstrap Integration

`/fab-setup` creates `docs/specs/index.md` during structural bootstrap (after memory/index.md). The creation is idempotent — if the file already exists, setup skips it with a status message.

## Design Decisions

### Flow Skeletons Live in `skills.md`, Not Per-File Mirrors
**Decision**: The structural quick-reference for each skill (Flow diagram, Tools, Sub-agents) lives folded into that skill's `docs/specs/skills.md` section; there is no per-skill mirror tree under `docs/specs/`, and no constitution rule keeping one in sync.
**Why**: The mirror tree's main output had become work about itself — the only review comments on its condensing change were mirror meta-consistency issues, and an earlier skills-review audit found a dozen drift bugs nobody had noticed, evidence the mirrors were not read operationally. The residual cost was the machinery (a constitution clause, a review must-fix rule, sweep classes, checklist items), paid on every structural skill change.
**Rejected**: Keeping condensed per-file mirrors (self-referential review load); moving skeletons to `docs/memory/` (specs are the human-curated design home — `skills.md` is the existing aggregate); dropping the skeletons entirely.
*Introduced by*: 260811-rehi-delete-spec-mirror-tree

### Context Loading Integration

`docs/specs/index.md` is included in the "Always Load" context layer in `_preamble.md`, alongside `config.yaml`, `constitution.md`, and `docs/memory/index.md`. This gives every skill baseline awareness of the specs landscape.
