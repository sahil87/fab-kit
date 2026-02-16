# Migrations

**Domain**: fab-workflow

## Overview

The migration system lets kit releases ship step-by-step instructions that an LLM agent can follow to bring a project's `fab/` files in sync with the kit engine they run on. Migrations handle evolving `config.yaml` schemas, `.status.yaml` formats, naming conventions, and other project-level artifacts that live outside `fab/.kit/`.

## Requirements

### Dual-Version Model

Two VERSION files track the relationship between the installed engine and the project's file format:

- **`fab/.kit/VERSION`** — engine version (ships inside `.kit/`, replaced on each `fab-upgrade.sh` run)
- **`fab/VERSION`** — local project version (lives outside `.kit/`, NOT replaced on upgrades)

Both files contain a bare semver string (`MAJOR.MINOR.PATCH`), no prefix, no trailing content.

### Migration Directory

`fab/.kit/migrations/` ships with the kit and contains migration instruction files. The directory exists even if empty for the first release (`.gitkeep`).

### Migration File Format

Migration files are named `{FROM}-to-{TO}.md` where FROM and TO are full semver strings. A migration applies when `FROM <= fab/VERSION < TO`.

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

1. Compares `fab/VERSION` to `fab/.kit/VERSION`
2. Scans `fab/.kit/migrations/` for applicable files
3. Validates non-overlapping ranges
4. Applies migrations sequentially (sorted by FROM ascending)
5. Updates `fab/VERSION` after each successful migration

**Discovery algorithm**:
1. Find first migration where `FROM <= current < TO` → apply, set current = TO
2. If no match but a later migration exists with `FROM > current` → skip to that FROM (log skip)
3. If no match and no later migrations → set `fab/VERSION` to engine version

**Failure handling**: stops immediately on failure, `fab/VERSION` reflects last successful migration, suggests re-running `/fab-setup migrations`.

### Two-Step Update Flow

`fab-upgrade.sh` handles the mechanical `.kit/` swap. `/fab-setup migrations` (subcommand) handles intelligent migration execution. They are separate operations — the shell script doesn't need an LLM, the skill does.

### Version Drift Detection

- **`fab-upgrade.sh`**: prints drift reminder when `fab/VERSION` < engine after upgrade; prints init guidance if `fab/VERSION` missing
- **`/fab-status`**: displays `⚠ Version drift: local {X}, engine {Y} — run /fab-update` when versions differ
- **`fab-release.sh`**: warns when no migration targets the new release version; warns on overlapping migration ranges

### `fab/VERSION` Creation

Handled by `fab-sync.sh` during structural bootstrap:

- **New project** (no `config.yaml`): copies engine version from `fab/.kit/VERSION`
- **Existing project** (has `config.yaml`, no `fab/VERSION`): writes `0.1.0` (base version) so `/fab-setup migrations` runs all migrations
- **Already exists**: preserves existing value

## Design Decisions

### Range-Based Migration Applicability
**Decision**: Migration files define a FROM-TO version range. A migration applies when `FROM <= fab/VERSION < TO`. Any release can ship a migration file. The release author decides — the system does not impose rules based on bump type.
**Why**: Avoids hardcoding assumptions about which version types need migrations. Allows sparse migration files (no empty placeholders). Supports wide-range migrations covering multiple intermediate releases.
**Rejected**: Minor-only stepping (forced empty migration files), exact-version chaining (unbroken linked list, maintenance burden).

### Two-Step Update Flow
**Decision**: `fab-upgrade.sh` (script) handles mechanical swap; `/fab-setup migrations` (subcommand) handles intelligent migration.
**Why**: Migrations are LLM instruction files. The shell script handles download/swap (no LLM needed); the skill handles reading and applying instructions (LLM needed). Preserves Constitution I.
**Rejected**: Single combined script — would require embedding LLM invocation in shell or making migration files executable (violates pure prompt play).

### Warning-Only Release Validation
**Decision**: `fab-release.sh` warns but does not block releases without a migration file targeting the new version.
**Why**: Not every release changes project-level files. Blocking would create friction with empty boilerplate migration files.
**Rejected**: Hard block — too restrictive.

### Existing Projects Get Base Version
**Decision**: `fab-sync.sh` assigns `0.1.0` to existing projects (detected via `config.yaml` presence) so `/fab-setup migrations` applies all needed migrations from the beginning.
**Why**: Existing projects predate the migration system. Starting from `0.1.0` ensures the full migration chain runs. New projects get the engine version since their config is freshly generated.
**Rejected**: Assigning engine version to all — would skip needed migrations for existing projects.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | `/fab-update` absorbed into `/fab-setup migrations` subcommand; `lib/sync-workspace.sh` → `fab-sync.sh`; updated design decision wording |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` in VERSION creation and design decision references |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Initial creation — migration system, dual-version model, `/fab-setup migrations` skill, version drift detection, `fab/VERSION` creation |
