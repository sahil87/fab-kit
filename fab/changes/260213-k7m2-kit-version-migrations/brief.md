# Brief: Kit Version Migrations

**Change**: 260213-k7m2-kit-version-migrations
**Created**: 2026-02-13
**Status**: Draft

## Origin

> We need the concept of migrations to gain flexibility with config.yaml, status.yaml, conventions etc, across different releases of fabkit. Migrations can be simple instruction files for agents to move the project's fab implementation from one minor version to the next.
> Kit engine version (the engine that is installed): fab/.kit/VERSION
> Kit local version: fab/VERSION (new file, should be setup by fab-init)
>
> The fab-update command should work at moving the local version, one minor version at a time to the kit's version. The steps for the conversion can reside in fab/.kit/migrations/ (distributed with the code).

## Why

Today `fab-update.sh` atomically replaces `fab/.kit/` with the latest release, but there's no mechanism to evolve the project-level files (`config.yaml`, `.status.yaml` format, conventions in `constitution.md`, etc.) that live *outside* `.kit/`. When a new kit version introduces schema changes, new required fields, or renamed conventions, projects are silently left on stale formats with no guidance on how to catch up.

A migration system lets each kit release ship step-by-step instructions that an agent (or human) can follow to bring a project's fab implementation from one minor version to the next, keeping project files in sync with the engine they run on.

## What Changes

- **New file `fab/VERSION`** — tracks the local project's kit version (the version its config, status, and conventions were written for). Created by `/fab-init`, updated by `/fab-update` after migrations complete.
- **New directory `fab/.kit/migrations/`** — ships with the kit. Contains markdown migration instruction files, one per minor version transition (e.g., `0.1-to-0.2.md`).
- **Modified `/fab-init`** — creates `fab/VERSION` on first init, copying the value from `fab/.kit/VERSION`. Existing projects without `fab/VERSION` get it on next `/fab-init` re-run.
- **New `/fab-update` skill** — compares `fab/VERSION` (local) to `fab/.kit/VERSION` (engine), then runs each intervening migration one minor version at a time. Each migration is an LLM instruction file — the skill reads it and executes the steps. Updates `fab/VERSION` after each successful migration step. Separate from `fab-update.sh`, which handles the mechanical `.kit/` swap.
<!-- assumed: patch versions skip migrations — only minor+ version bumps get migration files -->
- **Migration file format** — each migration is a markdown file with structured agent instructions: what to check, what to change, what to verify. Follows the "pure prompt play" principle (Constitution I) — the LLM interprets and applies them, no scripts needed.

## Affected Docs

### New Docs
- `fab-workflow/migrations`: Migration system — version tracking, migration file format, update workflow, authoring guide

### Modified Docs
- `fab-workflow/kit-architecture`: Add `fab/.kit/migrations/` to directory structure, update version tracking section to cover dual-version model (engine vs local), update `fab-update.sh` description
- `fab-workflow/init`: Document `fab/VERSION` creation during `/fab-init`
- `fab-workflow/distribution`: Add migration awareness to the update flow description

### Removed Docs
(none)

## Impact

- **New `/fab-update` skill (e.g., `fab/.kit/skills/fab-update.md`)** — the migration runner; reads migration instructions and applies them as an LLM
- **`fab/.kit/scripts/fab-update.sh`** — unchanged in core purpose (download + swap), but the user workflow becomes: run the script first, then invoke `/fab-update` to apply migrations
- **`fab/.kit/skills/fab-init.md`** — needs to create `fab/VERSION` on init and re-init
- **`fab/.kit/scripts/fab-setup.sh`** — may need to create `fab/.kit/migrations/` directory
- **`fab/.kit/scripts/fab-release.sh`** — should validate that migration files exist for any minor version bump (optional safeguard)
- **`fab/.kit/scripts/fab-status.sh`** — could display version drift warning when `fab/VERSION` < `fab/.kit/VERSION`
- **All projects using fab-kit** — need to run `/fab-update` (or `/fab-init`) once to get the initial `fab/VERSION` file

## Open Questions

(none — all decisions resolved via SRAD)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Migration files named by minor version range (e.g., `0.1-to-0.2.md`) | Natural convention matching semver; user said "one minor version at a time" |
| 2 | Confident | Two-step update flow: `fab-update.sh` (script, mechanical swap) then `/fab-update` (skill, LLM-driven migrations) | Migrations are agent instruction files — they need an LLM to interpret. Shell script handles download/swap; skill handles intelligent migration execution. |
| 3 | Tentative | Patch versions don't need migration files — only minor+ bumps trigger migrations | User specifically mentioned "minor version" transitions; patch updates typically only change .kit/ internals without schema/convention changes |

3 assumptions made (2 confident, 1 tentative). Run /fab-clarify to review.
