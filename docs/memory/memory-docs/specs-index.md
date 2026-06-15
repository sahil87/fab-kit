---
description: "`docs/specs/` directory — pre-implementation specs, distinction from memory, bootstrap and context integration, per-skill SPEC mirror coverage + naming policy (`SPEC-{source-filename}.md`; `_cli-fab`/`_cli-external` excluded — uliv); mirrors are reserved paths for `docs-reorg-specs` (d9rs); specs are out of scope for compatibility/frontmatter backfill — no specs-index generator, hand-rewritten index, `docs-reorg-specs` carries an explicit no-symmetry note (5ewp)"
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

`docs/specs/skills/` holds one flow-diagram SPEC per skill source file (Summary + Flow + supporting tables, cross-referencing the skill source as canonical). The constitution requires every `src/kit/skills/*.md` edit to update its corresponding `SPEC-*.md` mirror.

- **Naming**: mechanical `SPEC-{source-filename}.md` — partials keep their leading underscore (`SPEC-_review.md`, `SPEC-_preamble.md`, `SPEC-_generation.md`). The former outlier `SPEC-preamble.md` was renamed to `SPEC-_preamble.md` in uliv; the live reference in [`_shared/context-loading.md`](../_shared/context-loading.md) was updated, while historical changelog rows keep the old name.
- **Coverage**: every user-invocable skill and every behavioral partial carries a SPEC. uliv closed the coverage gap with four new files: `SPEC-internal-consistency-check.md`, `SPEC-internal-retrospect.md`, `SPEC-internal-skill-optimize.md`, and `SPEC-_generation.md`.
- **Exclusion policy**: the pure-reference partials `_cli-fab.md` and `_cli-external.md` carry **no** SPEC — their content mirrors the CLI surface rather than defining behavior, and the constitution already forces `_cli-fab.md` updates on every CLI change (a SPEC would be a third copy of the same tables). The policy and the naming convention are documented in `docs/specs/skills.md` § New Skill Checklist (the SPEC-mirror item) — the single home for both, alongside the checklist's other integration points (frontmatter fields, preamble-read line, `helpers:` declaration, `Next:` line, Error Handling + Key Properties tables, skills.md mapping row, fabhelp.go help grouping).
- **Reserved paths for spec reorganization (d9rs)**: `docs-reorg-specs` treats the mirrors as constitution-pinned reserved paths — their names derive mechanically from their sources and the constitution requires every skill edit to update its mirror, so the skill never proposes renaming, moving, merging, or splitting them (a Migration Map row targeting a reserved path is invalid). They may be *read* for theme analysis; the skill's Step 1 also now recurses into `docs/specs/` subfolders (e.g., `skills/`, `findings/`).
- **d9rs resync**: the docs-reality sweep resynced ~23 drifted mirrors against their post-batch sources (Theme 7c — e.g., SPEC-fab-operator's rejected Decision 2 recorded as current, SPEC-fab-continue writing a removed "Spec" artifact, SPEC-fab-clarify's removed `[target-artifact]` flow, SPEC-_preamble's misquoted opening instruction) and rewrote `SPEC-hooks.md` as-shipped (Go `fab hook` handlers; no deleted shell scripts, no shipped-behavior-as-proposal).

### Bootstrap Integration

`/fab-setup` creates `docs/specs/index.md` during structural bootstrap (after memory/index.md). The creation is idempotent — if the file already exists, setup skips it with a status message.

## Design Decisions

### SPEC Mirrors Are Reserved Paths in Spec Reorganization
**Decision**: `docs-reorg-specs` exempts `docs/specs/skills/SPEC-*.md` from reorganization — read for theme analysis only; any Migration Map row targeting a reserved path is invalid.
**Why**: Mirror names derive mechanically from their `src/kit/skills/` sources (`SPEC-{source-filename}.md`) and the constitution pins the skill-edit ⇒ mirror-update rule. A reorg that renamed, moved, merged, or split a mirror would break the mechanical naming contract and orphan the constitution rule — the mirror set's structure is owned by the source tree, not by theme analysis.
**Rejected**: Allowing migrations with link rewriting — the naming convention itself is the contract, not just the inbound links; rewriting links would preserve navigation while still breaking the source↔mirror correspondence.
*Introduced by*: 260612-d9rs-docs-reality-sweep

### Context Loading Integration

`docs/specs/index.md` is included in the "Always Load" context layer in `_preamble.md`, alongside `config.yaml`, `constitution.md`, and `docs/memory/index.md`. This gives every skill baseline awareness of the specs landscape.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260614-5ewp-reorg-memory-backfill-orchestration | 2026-06-14 | **No compatibility/backfill step for specs** recorded (new Requirements subsection): specs are out of scope for the pre-fab-kit frontmatter backfill `/docs-reorg-memory` orchestrates — verified asymmetry (no specs-index generator, hand-rewritten index, Constitution VI human-curated). `docs-reorg-specs.md` now carries an explicit one-line no-symmetry note so a future contributor does not "fix the asymmetry" by adding a specs backfill. |
| 260612-d9rs-docs-reality-sweep | 2026-06-12 | **Reserved-path exemption** (skills-audit batch 5/5): `docs-reorg-specs` never proposes renaming/moving/merging/splitting the constitution-pinned `docs/specs/skills/SPEC-*.md` mirrors (read-only for theme analysis; reserved-path Migration Map rows invalid) and its Step 1 now recurses into `docs/specs/` subfolders. New "SPEC Mirrors Are Reserved Paths in Spec Reorganization" design decision. **Mirror resync recorded**: ~23 drifted mirrors resynced against post-batch sources (Theme 7c) and `SPEC-hooks.md` rewritten as-shipped (Go `fab hook` system; no deleted shell scripts, UserPromptSubmit registered). |
| 260611-uliv-skills-staleness-sweep-frontmatter-fixes | 2026-06-11 | Added the "Per-Skill SPEC Mirrors" section (H): mechanical `SPEC-{source-filename}.md` naming with partials keeping the leading underscore (`SPEC-preamble.md` renamed to `SPEC-_preamble.md`; live ref in `_shared/context-loading.md` updated, historical changelog rows exempt); four new SPECs created (`internal-consistency-check`, `internal-retrospect`, `internal-skill-optimize`, `_generation`); explicit exclusion policy for the pure-reference partials `_cli-fab`/`_cli-external` (CLI mirrors, not behavior — a SPEC would be a third copy), documented in `docs/specs/skills.md` § New Skill Checklist together with the naming convention. |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` in bootstrap integration reference |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated all path references from `fab/specs/` to `docs/specs/` |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Consistent design terminology, updated from specs-index references |
| 260209-h3v7-fab-backfill | 2026-02-09 | Updated Human-Curated Ownership section to reference `/docs-hydrate-specs` as assisted reverse-hydration |
| 260207-bb1q-add-specs-index | 2026-02-07 | Initial creation — added `docs/specs/` directory, design index with boilerplate, bootstrap and context loading integration |
