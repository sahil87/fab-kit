---
description: "`docs/specs/` directory — pre-implementation specs, distinction from memory, bootstrap and context integration, per-skill SPEC mirror coverage + naming policy (`SPEC-{source-filename}.md`; `_cli-fab`/`_cli-external` excluded — uliv)"
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

### Per-Skill SPEC Mirrors (`docs/specs/skills/`)

`docs/specs/skills/` holds one flow-diagram SPEC per skill source file (Summary + Flow + supporting tables, cross-referencing the skill source as canonical). The constitution requires every `src/kit/skills/*.md` edit to update its corresponding `SPEC-*.md` mirror.

- **Naming**: mechanical `SPEC-{source-filename}.md` — partials keep their leading underscore (`SPEC-_review.md`, `SPEC-_preamble.md`, `SPEC-_generation.md`). The former outlier `SPEC-preamble.md` was renamed to `SPEC-_preamble.md` in uliv; the live reference in [`_shared/context-loading.md`](../_shared/context-loading.md) was updated, while historical changelog rows keep the old name.
- **Coverage**: every user-invocable skill and every behavioral partial carries a SPEC. uliv closed the coverage gap with four new files: `SPEC-internal-consistency-check.md`, `SPEC-internal-retrospect.md`, `SPEC-internal-skill-optimize.md`, and `SPEC-_generation.md`.
- **Exclusion policy**: the pure-reference partials `_cli-fab.md` and `_cli-external.md` carry **no** SPEC — their content mirrors the CLI surface rather than defining behavior, and the constitution already forces `_cli-fab.md` updates on every CLI change (a SPEC would be a third copy of the same tables). The policy and the naming convention are documented in `docs/specs/skills.md` § New Skill Checklist (the SPEC-mirror item) — the single home for both, alongside the checklist's other integration points (frontmatter fields, preamble-read line, `helpers:` declaration, `Next:` line, Error Handling + Key Properties tables, skills.md mapping row, fabhelp.go help grouping).

### Bootstrap Integration

`/fab-setup` creates `docs/specs/index.md` during structural bootstrap (after memory/index.md). The creation is idempotent — if the file already exists, setup skips it with a status message.

### Context Loading Integration

`docs/specs/index.md` is included in the "Always Load" context layer in `_preamble.md`, alongside `config.yaml`, `constitution.md`, and `docs/memory/index.md`. This gives every skill baseline awareness of the specs landscape.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260611-uliv-skills-staleness-sweep-frontmatter-fixes | 2026-06-11 | Added the "Per-Skill SPEC Mirrors" section (H): mechanical `SPEC-{source-filename}.md` naming with partials keeping the leading underscore (`SPEC-preamble.md` renamed to `SPEC-_preamble.md`; live ref in `_shared/context-loading.md` updated, historical changelog rows exempt); four new SPECs created (`internal-consistency-check`, `internal-retrospect`, `internal-skill-optimize`, `_generation`); explicit exclusion policy for the pure-reference partials `_cli-fab`/`_cli-external` (CLI mirrors, not behavior — a SPEC would be a third copy), documented in `docs/specs/skills.md` § New Skill Checklist together with the naming convention. |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-init` → `/fab-setup` in bootstrap integration reference |
| 260214-m3v8-relocate-docs-dev-scripts | 2026-02-14 | Updated all path references from `fab/specs/` to `docs/specs/` |
| 260211-r3k8-simplify-planning-stages | 2026-02-11 | Consistent design terminology, updated from specs-index references |
| 260209-h3v7-fab-backfill | 2026-02-09 | Updated Human-Curated Ownership section to reference `/docs-hydrate-specs` as assisted reverse-hydration |
| 260207-bb1q-add-specs-index | 2026-02-07 | Initial creation — added `docs/specs/` directory, design index with boilerplate, bootstrap and context loading integration |
