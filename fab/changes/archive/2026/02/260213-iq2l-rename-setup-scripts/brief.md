# Brief: Rename and Reorganize Bootstrap Scripts

**Change**: 260213-iq2l-rename-setup-scripts
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Rename and reorganize fab-init, fab-setup.sh, and fab-update.sh for clarity. Specifically: (1) Rename fab-setup.sh to _fab-scaffold.sh (or move to lib/) to signal it's internal plumbing, not user-facing. (2) Rename fab-update.sh to fab-upgrade.sh to better match developer expectations. (3) Keep fab-init as-is. The goal is to reduce user-facing concepts from 3 to 2 (init + upgrade), with scaffold as an internal implementation detail both call into.

## Why

"init" and "setup" are near-synonyms — users can't tell which to run without reading source code. The current 3-concept model (init skill, setup script, update script) forces users to understand a distinction that should be an implementation detail. Renaming clarifies the mental model: two user-facing entry points (init, upgrade) backed by one internal scaffold primitive.

## What Changes

- **Rename `fab-setup.sh` to `_fab-scaffold.sh`**: The underscore prefix signals "internal, don't call directly" — consistent with `_context.md` convention already used in skills. The name "scaffold" replaces the synonymous "setup" to break confusion with "init"
- **Rename `fab-update.sh` to `fab-upgrade.sh`**: "Upgrade" better describes what it does (fetch a new kit version), matches developer vocabulary (npm upgrade, brew upgrade), and distinguishes from generic "update"
- **Update all callers**: `fab-init.md` skill, `fab-upgrade.sh` (internal call to scaffold), worktree init script (`2-rerun-fab-setup.sh` renamed to `2-rerun-fab-scaffold.sh`)
- **Update cross-references in live docs**: `fab/docs/` and `fab/design/` files that reference the old script names
- **Update README.md**: Installation and upgrade instructions reference both scripts
<!-- assumed: Underscore prefix convention (_fab-scaffold.sh) rather than lib/ subdirectory — matches existing _context.md pattern and avoids changing the scripts/ directory structure -->

## Affected Docs

### New Docs
(none)

### Modified Docs
- `fab-workflow/kit-architecture.md`: Update `fab-setup.sh` and `fab-update.sh` references to new names
- `fab-workflow/distribution.md`: Update script names in distribution/install instructions
- `fab-workflow/init.md`: Update `fab-setup.sh` references
- `fab-workflow/model-tiers.md`: Update script references if present
- `fab-workflow/hydrate.md`: Update any setup script references
- `fab-workflow/index.md`: Update script names in domain index

### Removed Docs
(none)

## Impact

- **Shell scripts** (direct renames):
  - `fab/.kit/scripts/fab-setup.sh` -> `fab/.kit/scripts/_fab-scaffold.sh`
  - `fab/.kit/scripts/fab-update.sh` -> `fab/.kit/scripts/fab-upgrade.sh`
  - `fab/.kit/worktree-init-common/2-rerun-fab-setup.sh` -> `fab/.kit/worktree-init-common/2-rerun-fab-scaffold.sh`
- **Skill files**: `fab/.kit/skills/fab-init.md` — references to `fab-setup.sh`
- **Model tiers**: `fab/.kit/model-tiers.yaml` — if it lists scripts
- **Design docs**: `fab/design/architecture.md`, `fab/design/glossary.md`
- **README.md**: Installation one-liner, upgrade instructions
- **Archived changes**: NOT touched — archives are historical records
- **Coordination**: Complements `260213-3tyk-merge-fab-init-subcommands` (that change merges init subcommands; this one renames the underlying scripts). No conflict — they touch different layers.

## Open Questions

(none — all decisions resolved via SRAD)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | `fab-upgrade.sh` over other names (e.g., `fab-update.sh` kept) | "Upgrade" matches developer vocabulary (npm/brew), user explicitly discussed this |
| 2 | Tentative | Underscore prefix (`_fab-scaffold.sh`) rather than `lib/` subdirectory | Matches existing `_context.md` convention; avoids restructuring `scripts/` directory |

2 assumptions made (1 confident, 1 tentative). Run /fab-clarify to review.
