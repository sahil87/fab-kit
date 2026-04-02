# Intake: Remove Sync Version File

**Change**: 260402-0ak9-remove-sync-version-file
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Discussion session (`/fab-discuss`) explored the four version files in the project: `fab/.kit/VERSION`, `config.yaml fab_version`, `fab/.kit-migration-version`, and `fab/.kit-sync-version`. Analysis showed that `.kit-sync-version` is redundant — its sole purpose (detecting stale skill deployments) can be achieved by comparing the two files that already exist: `fab/.kit/VERSION` vs `config.yaml fab_version`. User agreed to consolidate from 4 to 3 version locations, keeping scope tight to just removing the sync-version file.

## Why

Version state is currently spread across four locations, which is confusing to understand and maintain. The `.kit-sync-version` file exists only so `preflight.go` can warn "skills may be out of sync" by comparing it against `fab/.kit/VERSION`. But this comparison is equivalent to checking whether `fab/.kit/VERSION` differs from `config.yaml fab_version` — a divergence that already indicates "kit upgraded but project not synced." Removing `.kit-sync-version` eliminates the hardest-to-explain version file without losing any detection capability.

The only edge case `.kit-sync-version` catches that the replacement doesn't is a partially-completed sync (crash mid-deploy). This is better handled by sync being idempotent (which it already is) than by a version stamp.

## What Changes

### 1. Remove `writeSyncVersionStamp()` from sync.go

In `src/go/fab-kit/internal/sync.go`:
- Delete the `writeSyncVersionStamp()` function
- Remove the call site that invokes it at the end of the sync process

### 2. Update staleness check in preflight.go

In `src/go/fab/internal/preflight/preflight.go`, update `checkSyncStaleness()`:
- **Before**: reads `fab/.kit/VERSION` and `fab/.kit-sync-version`, warns if they differ
- **After**: reads `fab/.kit/VERSION` and `fab_version` from `fab/project/config.yaml`, warns if they differ
- Warning message stays similar: "Skills may be out of sync — run `fab sync` to refresh (engine {kit_version}, project {config_version})"

### 3. Remove from .gitignore and scaffold

- Remove the `fab/.kit-sync-version` line from `.gitignore`
- Remove it from `fab/.kit/scaffold/fragment-.gitignore` (the scaffold template for new projects)

### 4. Update documentation

Update references in these memory files:
- `docs/memory/fab-workflow/kit-architecture.md`
- `docs/memory/fab-workflow/preflight.md`
- `docs/memory/fab-workflow/distribution.md`

### 5. Migration for cleanup

Ship a migration file in `fab/.kit/migrations/` that instructs removal of the orphaned `fab/.kit-sync-version` file from existing projects. Since the file is gitignored and local-only, this is a lightweight cleanup — not blocking.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Remove `.kit-sync-version` from the version file inventory
- `fab-workflow/preflight`: (modify) Update staleness check description to reflect new comparison
- `fab-workflow/distribution`: (modify) Remove `.kit-sync-version` from distributed file list if mentioned

## Impact

- **Go source**: `sync.go` (fab-kit binary), `preflight.go` (fab-go binary) — both need recompilation
- **Existing users**: Orphaned `.kit-sync-version` files remain harmlessly until migration runs
- **New projects**: Will never see `.kit-sync-version` — cleaner mental model from day one
- **Preflight behavior**: Warning still fires on version drift, just uses a different comparison source

## Open Questions

- None — the discussion session resolved all design questions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `.kit-sync-version` file entirely | Discussed — user explicitly chose this consolidation | S:95 R:85 A:90 D:95 |
| 2 | Certain | Replace staleness check with VERSION vs config.yaml fab_version | Discussed — equivalent detection, user confirmed approach | S:90 R:80 A:85 D:90 |
| 3 | Certain | Keep scope to sync-version removal only (no rename of fab_version, no migration-version move) | Discussed — user explicitly scoped to tight change | S:95 R:95 A:95 D:95 |
| 4 | Confident | Ship a migration to clean up orphaned file | Context.md requires migrations for user data restructuring; `.kit-sync-version` is local-only and gitignored so impact is minimal, but consistency with migration policy favors including one | S:60 R:90 A:75 D:80 |
| 5 | Confident | Warning message format stays similar but shows "project {version}" instead of "last synced {version}" | Preflight already emits non-blocking warnings; wording change follows naturally from the new comparison source | S:65 R:90 A:80 D:85 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
