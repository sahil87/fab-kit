---
type: memory
description: "Migration system — dual-version model, migration file format, binary-owned discovery (`fab migrations-status [--json]` / `DiscoverMigrations`), `/fab-setup migrations` subcommand (delegates discovery, applies LLM-driven), brew-install migration, `1.8.0-to-1.9.0` migration (tasks-stage collapse + plan.md), `1.9.1-to-1.9.2` migration (`true_impact_exclude` config field), `1.9.7-to-1.10.0` migration (spec-stage collapse, four-state spec.md→plan.md case table), `2.1.6-to-2.2.0` migration (drops dead `stage_directives` + defensive `model_tiers`; preserves `stage_hooks`), `2.2.0-to-2.3.0` migration (fully-commented `agent.tiers` reference block, comment-sentinel idempotency), `2.5.5-to-2.6.0` migration (freeze-on-write `log.md` re-baseline — `fab memory-index --rebuild` + commit, `--rebuild` binary pre-check, no `fab/`/`.status.yaml` change), version drift detection (`upgrade-repo` mechanical detection + silent self-stamp + TTY-gated styled reminder), `fab/.kit-migration-version` creation"
---
# Migrations

**Domain**: distribution

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

Migration file ranges MUST NOT overlap. Overlap detection is owned by the binary (`fab migrations-status`), which surfaces conflicting filename pairs in its `overlaps` field; both `/fab-setup migrations` and `fab upgrade-repo` refuse to apply (or stamp) when `overlaps` is non-empty.

### Binary-Owned Discovery — `fab migrations-status`

Discovery (the scan/parse/validate-non-overlap/sort + the applicability walk) lives in the `fab-kit` binary, implemented in `src/go/fab-kit/internal/migrations.go`:

- `parseMigrationFilename(name)` — matches `{FROM}-to-{TO}.md`, parses both parts as semver, returns false for non-matching names (`.gitkeep`, `README.md`, malformed).
- `DiscoverMigrations(migrationsDir, local, engine)` — scans the dir, detects overlaps (`A.From < B.To && B.From < A.To`), sorts by FROM ascending, walks the discovery loop, and returns a `DiscoverResult{Local, Engine, Applicable, GapSkips, Overlaps}`. It reuses the existing `parseSemver`/`compareSemver` helpers in `semver.go` (split out of `sync.go` in 260612-tb6f; no new semver dependency). The convenience predicate "migrations needed" is `len(Applicable) > 0`.

`fab migrations-status [--json]` exposes this as a queryable command (registered in the router's `fabKitArgs` allowlist so it routes to `fab-kit`). It resolves `fab/.kit-migration-version` (local) and the engine `VERSION` from the cached kit, runs `DiscoverMigrations`, and reports the result. Human output lists local/engine, the ordered applicable list (or "No migrations apply."), gap-skips, and overlaps. `--json` emits `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}`. Exit code is `0` on any clean query — including the no-op case AND the overlap case (overlap is surfaced via the `overlaps` field); non-zero only on a genuine error (missing version file, unreadable dir). The command is read-only — it never writes `fab/.kit-migration-version`.

### `/fab-setup migrations` Subcommand

The migration runner, a subcommand of `/fab-setup` (previously the standalone `/fab-update` skill). It:

1. Runs `fab migrations-status --json` (binary-owned discovery — no manual scan/parse/validate/sort in skill prose)
2. STOPs and reports if `overlaps` is non-empty
3. Surfaces any `gap_skips` lines, then applies each file in `applicable` sequentially (FROM ascending, already chained by the binary)
4. Reads each migration file and executes its Pre-check/Changes/Verification (application stays LLM-driven per Constitution I)
5. Writes the migration's `TO` to `fab/.kit-migration-version` after each successful migration

Only *discovery* moved into the binary; *application* of each migration instruction file remains an LLM activity in the skill.

**Discovery loop** (now implemented in `DiscoverMigrations`):
1. Find first migration where `FROM <= current < TO` → append to `Applicable`, set current = TO
2. If no match but a later migration exists with `FROM > current` → record a gap-skip, advance current to that FROM
3. If no match and no later migrations → done (empty `Applicable` = no-op; `fab upgrade-repo` self-stamps the engine version in this case)

**Failure handling**: stops immediately on failure, `fab/.kit-migration-version` reflects last successful migration, suggests re-running `/fab-setup migrations`.

### Two-Step Update Flow

`fab upgrade-repo` (shim subcommand) handles the mechanical `.kit/` swap. `/fab-setup migrations` (skill subcommand) handles intelligent migration execution. They are separate operations — the shim handles download/swap (no LLM needed), the skill handles reading and applying instructions (LLM needed).

### Brew-Install Migration

A migration file for the transition to the system shim model. The migration:

1. **Prerequisite gate**: Verify `fab` (system shim) is on PATH. If not, instruct: `"Install fab-kit first: brew tap sahil87/tap && brew install fab-kit"`
2. **Add `fab_version`**: Write `fab_version: "{version}"` to `fab/project/config.yaml` (set to the current `$(fab kit-path)/VERSION`)
3. **Clean `.envrc`**: Remove the `PATH_add src/kit/bin` line if present
4. **Clean `fab-go binary at `**: Remove `fab`, `fab-go`, `wt`, `idea` — only `.gitkeep` remains

**Scenarios**:
- Migration on existing repo — adds `fab_version`, cleans `.envrc`, removes binaries; subsequent `fab` invocations work via system shim
- Migration without shim installed — stops at prerequisite gate with install instructions

### `2.5.5-to-2.6.0` Re-Baseline Migration (freeze-on-write `log.md` — tayp)

`src/kit/migrations/2.5.5-to-2.6.0.md` transitions existing projects onto **freeze-on-write** `log.md` generation (the schema/code side is in [pipeline/schemas.md](/pipeline/schemas.md) § Freeze-on-Write `log.md` Generation; the normative spec is [fkf.md](../../specs/fkf.md) §6.4). As of 2.6.0 `fab memory-index` reads the existing `log.md` back and appends-only rather than re-projecting from scratch, so it is deterministic across git history rewrites. Existing projects carry `log.md` files generated under the old pure-projection model — already stale relative to live git after any squash-merge — so the first freeze-on-write run would freeze the stale-but-committed lines. The migration **IS the fix**: a one-time `fab memory-index --rebuild` (the destructive re-projection — clean baseline from current git) followed by a commit; from that commit on, every `fab memory-index` run is append-only stable.

- **Binary pre-check (the upgrade-ordering guard).** This is the precedent for a migration whose Pre-check gates on a **binary capability**, not just project-file state. The standard upgrade ordering MUST be respected — the new binary lands first (`brew upgrade fab-kit`), *then* `/fab-setup migrations` applies — because applying with an older binary would fail on the unknown `--rebuild` flag. The Pre-check probes `fab memory-index --help` for `--rebuild`; if absent it **aborts with no partial rewrite** (`Aborted: this migration needs fab ≥ 2.6.0 (the --rebuild flag). Upgrade the binary first: brew upgrade fab-kit.`). A project with no `docs/memory/` directory skips it entirely.
- **No `fab/` data change.** Unlike the `.status.yaml`-schema migrations above (`1.8.0-to-1.9.0`, `1.9.7-to-1.10.0`, `2.4.2-to-2.5.0`), this migration ships **no `.status.yaml` schema change and no `fab/` data change** — it only regenerates `docs/memory/` `log.md` files (and the indexes, which `--rebuild` also rewrites) and commits them. The re-baseline commit is the **last** churn the repo sees from the non-determinism issue.
- **Idempotent (Constitution III).** Re-running `--rebuild` + commit on an already-clean tree is a no-op diff (nothing to commit), and the `--rebuild` pre-check still passes. After the baseline, `fab memory-index --check` exits **0 or 1, never 2** (a freshly re-projected tree is provably never destructive-loss).
- **Version bump.** `src/kit/VERSION` is bumped to `2.6.0` (the migration's target version) — a behavior change to a shipped CLI warrants a minor bump, matching the catalog's `2.4.2-to-2.5.0` / `2.2.0-to-2.3.0` feature-migration convention.

### Version Drift Detection

- **`fab upgrade-repo`**: after sync, runs `DiscoverMigrations` against the target version's cached `migrations/` dir and the current `fab/.kit-migration-version` (mechanical relevance check, not string inequality). Three terminal cases:
  - **Overlap** → warns naming the conflicting files + "Run '/fab-setup migrations' to resolve."; does NOT stamp.
  - **Applicable non-empty** → prints `Run '/fab-setup migrations' to update project files ({LOCAL} -> {TARGET})`, styled bold+yellow (`\033[1;33m…\033[0m`) when `os.Stdout` is a character device and plain when piped/redirected (TTY detection is dependency-free via `os.ModeCharDevice`); does NOT stamp (the skill owns the write after applying).
  - **Applicable empty (no overlap)** → silently writes the target version to `fab/.kit-migration-version` (no migration line printed), stopping the drift that occurred when only the skill ever advanced the local version.
  - **`fab/.kit-migration-version` missing** → preserves the existing init-guidance behavior.
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
**Decision**: `fab upgrade-repo` (shim subcommand) handles mechanical swap; `/fab-setup migrations` (skill subcommand) handles intelligent migration.
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
