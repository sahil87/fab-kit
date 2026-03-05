# Proposal: Add Reading Order Guide and Documentation Map

**Change**: 260210-q7m3-reading-order-doc-map
**Created**: 2026-02-10
**Status**: Draft

## Why

A newcomer to this project faces 50+ markdown files across 5 directories with no guidance on where to start or what to read next. The existing index files (`fab/docs/index.md`, `fab/specs/index.md`) are flat tables without audience-specific paths or prerequisite relationships. Backlog item [wr10] identifies this as a concrete gap: "Currently a newcomer faces ~20+ documentation files with no guidance on sequence."

## What Changes

- **Modified `README.md`** — expand the existing Documentation section into a full Documentation Map providing:
  - Three audience-specific reading paths (new user, contributor, spec reader)
  - A complete document inventory grouped by category (Getting Started, Concepts, Reference, Internals/Research)
  - Brief descriptions of what each document covers
  - Explicit prerequisite relationships shown per-path (not exhaustive pairwise)
  - Prominent links to the glossary (`fab/specs/glossary.md`)
  <!-- clarified: doc map merged into README.md per user preference — no standalone file -->
- **Modified `fab/specs/index.md`** — add a "Start here" note pointing to the README Documentation Map for newcomers
- **Modified `fab/docs/index.md`** — add a "Start here" note pointing to the README Documentation Map for newcomers

## Affected Docs

### New Docs
(none)

### Modified Docs
- Root `README.md`: Expand Documentation section into full Documentation Map with audience paths, inventory, prerequisites, and glossary links
- `fab/specs/index.md`: Add newcomer orientation note pointing to README
- `fab/docs/index.md`: Add newcomer orientation note pointing to README

### Removed Docs
(none)

## Impact

- Documentation only — no code, scripts, or skill files are modified
- Entry-point files (README, both index.md) gain navigation pointers
- The glossary at `fab/specs/glossary.md` becomes a more explicitly referenced resource
- The `references/` directory is excluded from the doc map — it is self-contained with its own READMEs
  <!-- clarified: references/ excluded from doc map per user preference — self-contained -->
- Addresses backlog item [wr10]

## Open Questions

(none — all decisions resolved via SRAD analysis)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | ~~Confident~~ Resolved | ~~Standalone file~~ → Merged into README.md | User preference: single entry point, no extra file |
| 2 | Confident | Prerequisites shown per-audience-path rather than exhaustive pairwise graph | Per-path is more actionable for readers; pairwise graph would be hard to maintain and visually overwhelming |
| 3 | ~~Tentative~~ Resolved | Glossary linking scoped to entry-point docs only | User confirmed: entry-point docs (README, index files) only; full term audit is a separate effort |

3 assumptions made (2 confident, 1 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-10

- **Q**: Where should the Documentation Map live? (`doc/` doesn't exist, actual dirs are `references/` and `fab/`)
  **A**: Merge into README.md — single entry point, no extra file
- **Q**: Should the doc map cover the `references/` directory (22 research analysis files)?
  **A**: Exclude — `references/` is self-contained with its own READMEs
- **Q**: Glossary linking scope — entry-point docs only, or audit all 50 files?
  **A**: Accepted recommendation: entry-point docs only (README, index files)
