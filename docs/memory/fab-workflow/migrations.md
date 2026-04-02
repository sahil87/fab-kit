# Migrations

**Domain**: fab-workflow

## Overview

The migration system lets kit releases ship step-by-step instructions that an LLM agent can follow to bring a project's `fab/` files in sync with the kit engine they run on. Migrations handle evolving `config.yaml` schemas, `.status.yaml` formats, naming conventions, and other project-level artifacts that live outside `src/kit/`.

## Requirements

### Dual-Version Model

Two VERSION files track the relationship between the installed engine and the project's file format:

- **`$(fab kit-path)/VERSION`** — engine version (ships inside `.kit/`, replaced on each `fab-upgrade.sh` run)
- **`fab/.kit-migration-version`** — local project version (lives outside `.kit/`, NOT replaced on upgrades; renamed from `fab/.kit-migration-version`)

Both files contain a bare semver string (`MAJOR.MINOR.PATCH`), no prefix, no trailing content.

### Migration Directory

`$(fab kit-path)/migrations/` ships with the kit and contains migration instruction files. The directory exists even if empty for the first release (`.gitkeep`).

### Migration File Format

Migration files are named `{FROM}-to-{TO}.md` where FROM and TO are full semver strings. A migration applies when `FROM <= fab/.kit-migration-version < TO`.

Each migration file follows this structure:

```markdown
# Migration: {FROM} to {TO}

## Summary
{What changed and why migration is needed.}

## Pre-check
{Conditions to verify before applying.}

## Changes
{Ordered list of changes to apply.}

## Verification
{Steps to confirm migration succeeded.}
```

Migration files are pure markdown instructions (Constitution I — Pure Prompt Play). They contain no executable scripts — an LLM agent reads and applies them.

### Range-Based Applicability

Migration ranges are determined by the release author, not by version bump type. Any release (patch, minor, or major) can ship a migration file if it changes project-level files. Wide-range migrations (e.g., `0.2.0-to-0.4.0.md`) cover multiple intermediate releases.

Migration file ranges MUST NOT overlap. `/fab-setup migrations` validates this before applying.

### `/fab-setup migrations` Subcommand

The migration runner, now a subcommand of `/fab-setup` (previously the standalone `/fab-update` skill). It:

1. Compares `fab/.kit-migration-version` to `$(fab kit-path)/VERSION`
2. Scans `$(fab kit-path)/migrations/` for applicable files
3. Validates non-overlapping ranges
4. Applies migrations sequentially (sorted by FROM ascending)
5. Updates `fab/.kit-migration-version` after each successful migration

**Discovery algorithm**:
1. Find first migration where `FROM <= current < TO` → apply, set current = TO
2. If no match but a later migration exists with `FROM > current` → skip to that FROM (log skip)
3. If no match and no later migrations → set `fab/.kit-migration-version` to engine version

**Failure handling**: stops immediately on failure, `fab/.kit-migration-version` reflects last successful migration, suggests re-running `/fab-setup migrations`.

### Two-Step Update Flow

`fab upgrade` (shim subcommand) handles the mechanical `.kit/` swap. `/fab-setup migrations` (skill subcommand) handles intelligent migration execution. They are separate operations — the shim handles download/swap (no LLM needed), the skill handles reading and applying instructions (LLM needed).

### Brew-Install Migration

A migration file for the transition to the system shim model. The migration:

1. **Prerequisite gate**: Verify `fab` (system shim) is on PATH. If not, instruct: `"Install fab-kit first: brew tap wvrdz/tap && brew install fab-kit"`
2. **Add `fab_version`**: Write `fab_version: "{version}"` to `fab/project/config.yaml` (set to the current `$(fab kit-path)/VERSION`)
3. **Clean `.envrc`**: Remove the `PATH_add src/kit/bin` line if present
4. **Clean `fab-go binary at `**: Remove `fab`, `fab-go`, `wt`, `idea` — only `.gitkeep` remains

**Scenarios**:
- Migration on existing repo — adds `fab_version`, cleans `.envrc`, removes binaries; subsequent `fab` invocations work via system shim
- Migration without shim installed — stops at prerequisite gate with install instructions

### Version Drift Detection

- **`fab upgrade`**: prints drift reminder when `fab/.kit-migration-version` < engine after upgrade; prints init guidance if `fab/.kit-migration-version` missing
- **`/fab-status`**: displays `⚠ Version drift: local {X}, engine {Y} — run /fab-setup migrations` when versions differ
- **`release.sh`**: warns when no migration targets the new release version; warns on overlapping migration ranges

### `fab/.kit-migration-version` Creation

Handled by `fab-sync.sh` during structural bootstrap:

- **New project** (no `config.yaml`): copies engine version from `$(fab kit-path)/VERSION`
- **Existing project** (has `config.yaml`, no `fab/.kit-migration-version`): writes `0.1.0` (base version) so `/fab-setup migrations` runs all migrations
- **Already exists**: preserves existing value

## Design Decisions

### Range-Based Migration Applicability
**Decision**: Migration files define a FROM-TO version range. A migration applies when `FROM <= fab/.kit-migration-version < TO`. Any release can ship a migration file. The release author decides — the system does not impose rules based on bump type.
**Why**: Avoids hardcoding assumptions about which version types need migrations. Allows sparse migration files (no empty placeholders). Supports wide-range migrations covering multiple intermediate releases.
**Rejected**: Minor-only stepping (forced empty migration files), exact-version chaining (unbroken linked list, maintenance burden).

### Two-Step Update Flow
**Decision**: `fab upgrade` (shim subcommand) handles mechanical swap; `/fab-setup migrations` (skill subcommand) handles intelligent migration.
**Why**: Migrations are LLM instruction files. The shim handles download/cache/swap (no LLM needed); the skill handles reading and applying instructions (LLM needed). Preserves Constitution I.
**Rejected**: Single combined script — would require embedding LLM invocation in shell or making migration files executable (violates pure prompt play).

### Warning-Only Release Validation
**Decision**: `release.sh` warns but does not block releases without a migration file targeting the new version.
**Why**: Not every release changes project-level files. Blocking would create friction with empty boilerplate migration files.
**Rejected**: Hard block — too restrictive.

### Existing Projects Get Base Version
**Decision**: `fab-sync.sh` assigns `0.1.0` to existing projects (detected via `config.yaml` presence) so `/fab-setup migrations` applies all needed migrations from the beginning.
**Why**: Existing projects predate the migration system. Starting from `0.1.0` ensures the full migration chain runs. New projects get the engine version since their config is freshly generated.
**Rejected**: Assigning engine version to all — would skip needed migrations for existing projects.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Ships migration for existing users: verify cache populated, inline hooks in `.claude/settings.local.json` (replace `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh` with `fab hook <subcommand>`), remove `fab/.kit/` from project, clean `PATH_add fab/.kit/scripts` from `.envrc`, clean `fab/.kit/bin/*` from `.gitignore`. `$(fab kit-path)/VERSION` is now the engine version source (read from exe-sibling kit in cache). |
| 260402-0ak9-remove-sync-version-file | 2026-04-02 | Added migration `0.45.1-to-0.46.0.md` for orphaned `fab/.kit-sync-version` cleanup. Migration deletes the obsolete sync stamp file (staleness detection now uses `$(fab kit-path)/VERSION` vs `config.yaml fab_version`). Handles missing file gracefully. |
| 260401-46hw-brew-install-system-shim | 2026-04-02 | Added brew-install migration for transition to system shim model. Prerequisite gate: verifies `fab` system binary on PATH. Adds `fab_version` field to `config.yaml`. Cleans `.envrc` (removes `PATH_add src/kit/bin`). Cleans `fab-go binary at ` (removes `fab`, `fab-go`, `wt`, `idea`). Updated Two-Step Update Flow to reference `fab upgrade` replacing `fab-upgrade.sh`. |
| 260312-9r3t-pr-change-metadata | 2026-03-12 | Added migration `0.34.0-to-0.37.0.md` for discoverability of new `linear_workspace` config field. Migration checks if `fab/project/config.yaml` already has `linear_workspace` — if so, skips. Otherwise adds a commented-out `# linear_workspace: "your-workspace"` line under the `project:` block. Does not change behavior — surfaces the new option to existing users during `/fab-setup migrations`. |
| 260307-x2tx-status-symlink-pointer | 2026-03-07 | Replaced `fab/current` pointer file with `.fab-status.yaml` symlink at repo root. Added `id` field to `.status.yaml`. Updated resolution, switch, rename, pane-map, hooks, and dispatch. Migration `0.32.0-to-0.34.0` covers conversion. |
| 260226-koj1-version-staleness-warning | 2026-02-26 | Renamed `fab/project/VERSION` → `fab/.kit-migration-version` throughout. Added `0.20.0-to-0.21.0.md` migration for the rename. Updated dual-version model description. |
| 260218-5isu-fix-docs-consistency-drift | 2026-02-18 | Replaced stale `/fab-update` → `/fab-setup migrations` in `/fab-status` version drift display message |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | `/fab-update` absorbed into `/fab-setup migrations` subcommand; `lib/sync-workspace.sh` → `fab-sync.sh`; updated design decision wording |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` in VERSION creation and design decision references |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Initial creation — migration system, dual-version model, `/fab-setup migrations` skill, version drift detection, `fab/VERSION` creation |
