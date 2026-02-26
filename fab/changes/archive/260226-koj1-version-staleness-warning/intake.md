# Intake: Version Staleness Warning

**Change**: 260226-koj1-version-staleness-warning
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Add a mechanism to warn users that they are on an older version of the skills. Maybe add a VERSION to the skills folder?
>
> Clarification: The problem is stale local skill deployments, not project-vs-engine version drift. `.claude/`, `.agents/`, `.opencode/` are gitignored — when Developer A upgrades and commits, Developer B pulls the new `fab/.kit/` source via git but their local skill copies remain stale until they run `fab-sync.sh`. There is currently no warning about this.
>
> Decision: `.kit-sync-version` is gitignored (per-developer local state). Also rename `fab/project/VERSION` → `fab/.kit-migration-version` (better name for the migration tracking file).

## Why

The skill deployment directories (`.claude/skills/`, `.agents/skills/`, `.opencode/commands/`) are **gitignored** — they're local copies deployed by `fab-sync.sh`. When one developer upgrades `fab/.kit/` and commits, every other developer who pulls gets the new kit source but continues running stale skill copies. There is no mechanism to detect or warn about this.

Without a fix, developers unknowingly run outdated skills that may reference non-existent scripts, use deprecated templates, or miss new pipeline behavior. The mismatch is silent and can cause confusing failures that look like bugs rather than staleness.

The fix is a lightweight sync-staleness check: `fab-sync.sh` stamps the kit version it last synced, and a check at skill-invocation time compares the stamp against the current `fab/.kit/VERSION`.

## What Changes

### Sync version stamp (`.kit-sync-version`)

`fab-sync.sh` (specifically `2-sync-workspace.sh`) writes a `.kit-sync-version` marker file after successful skill deployment. The file contains the `fab/.kit/VERSION` value that was active at sync time.

Location: `fab/.kit-sync-version` — inside `fab/` but outside `.kit/` so it survives atomic kit replacement. **Gitignored** — this is per-developer local state (each developer's stamp reflects when *they* last ran `fab-sync.sh`).

Content: bare semver string matching `fab/.kit/VERSION` at sync time (e.g., `0.20.0`).

### Staleness detection

A check compares `fab/.kit/VERSION` against `fab/.kit-sync-version`. If they differ (or stamp is missing), the deployed skills are stale:

```
⚠ Skills out of sync — run fab-sync.sh to refresh (engine 0.20.0, last synced 0.19.0)
```

or if stamp is missing:

```
⚠ Skills may be out of sync — run fab-sync.sh to refresh
```

### Integration point

Preflight (`lib/preflight.sh`) — runs on every pipeline skill, emits to stderr (non-blocking). Reaches the most users. All pipeline skills already surface preflight stderr.

### Rename `fab/project/VERSION` → `fab/.kit-migration-version`

Rename the migration tracking file from `fab/project/VERSION` to `fab/.kit-migration-version`. This file tracks how far migrations have been run and **is committed** (shared state). The new name is clearer about its purpose and parallels the new `.kit-sync-version` file.

All references to `fab/project/VERSION` must be updated:
- `2-sync-workspace.sh` — creates the file on first setup
- `fab-upgrade.sh` — reads it for drift detection
- `/fab-setup migrations` — reads and writes it during migration runs
- `/fab-status` — reads it for version drift display
- `fab-release.sh` — references it in the release workflow
- `docs/memory/fab-workflow/migrations.md` — documents the dual-version model

## Affected Memory

- `fab-workflow/distribution`: (modify) Document sync version stamp and staleness detection
- `fab-workflow/preflight`: (modify) Add staleness check behavior
- `fab-workflow/kit-architecture`: (modify) Document `.kit-sync-version` marker file
- `fab-workflow/migrations`: (modify) Rename `fab/project/VERSION` → `fab/.kit-migration-version`

## Impact

- **`2-sync-workspace.sh`** — writes `.kit-sync-version` at end of sync; updates `fab/project/VERSION` → `fab/.kit-migration-version` references
- **`lib/preflight.sh`** — adds ~10 lines for staleness check with stderr warning
- **`fab-upgrade.sh`** — updates `fab/project/VERSION` → `fab/.kit-migration-version` references
- **`/fab-setup migrations`** (skill) — updates version file references
- **`/fab-status`** (skill) — updates version file references
- **`fab-release.sh`** — updates references
- **Scaffold gitignore** — adds `.kit-sync-version` entry
- **Skills using preflight** — automatically inherit the warning with zero changes
- **No breaking changes** — warning is non-blocking; rename can include a migration or compat shim

## Open Questions

*None — all questions resolved.*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Problem is stale local skill copies, not project-vs-engine drift | User clarified — `.claude/`, `.agents/` are gitignored; git pull doesn't refresh them | S:95 R:95 A:95 D:95 |
| 2 | Certain | `fab-sync.sh` is the sync mechanism | It's the only thing that deploys skills to agent directories — the stamp must come from it | S:90 R:95 A:95 D:95 |
| 3 | Certain | Warning is non-blocking | Blocking on staleness would break workflows; Constitution III (idempotent operations) | S:85 R:95 A:90 D:95 |
| 4 | Certain | `.kit-sync-version` is gitignored | User decided — per-developer local state; each developer's stamp reflects their own last sync | S:95 R:90 A:95 D:95 |
| 5 | Certain | Rename `fab/project/VERSION` → `fab/.kit-migration-version` | User decided — better name, clearer purpose, parallels `.kit-sync-version` | S:95 R:80 A:95 D:95 |
| 6 | Confident | Stamp file contains the kit VERSION at sync time | Simple, deterministic — if stamp != current VERSION, skills are stale | S:80 R:90 A:85 D:85 |
| 7 | Confident | Preflight is the primary integration point | All pipeline skills run preflight; stderr warning surfaces without agent cooperation | S:75 R:85 A:90 D:80 |
| 8 | Confident | Stamp location is `fab/.kit-sync-version` | Outside `.kit/` (survives atomic replacement), inside `fab/` (colocated with kit concerns) | S:75 R:85 A:80 D:80 |

8 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
