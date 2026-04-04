# Spec: Remove Sync Version File

**Change**: 260402-0ak9-remove-sync-version-file
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Renaming `fab_version` to `kit_version` in config.yaml — separate change with migration implications
- Moving `fab/.kit-migration-version` into config.yaml — separate concern
- Changing any behavior of `fab upgrade` or `fab init` version-setting logic

## Sync: Remove `writeSyncVersionStamp`

### Requirement: sync SHALL NOT write `.kit-sync-version`

The `writeSyncVersionStamp()` function in `src/go/fab-kit/internal/sync.go` and its call site SHALL be removed. `fab-kit sync` SHALL NOT create, update, or reference `fab/.kit-sync-version`.

#### Scenario: Sync completes without writing stamp file

- **GIVEN** a project with `fab/.kit/VERSION` at `0.45.1`
- **WHEN** `fab-kit sync` runs to completion
- **THEN** no `fab/.kit-sync-version` file is created or modified
- **AND** all other sync behavior (skill deployment, scaffold, direnv) is unchanged

#### Scenario: Existing `.kit-sync-version` is left as orphan

- **GIVEN** a project with an existing `fab/.kit-sync-version` file from a prior version
- **WHEN** `fab-kit sync` runs
- **THEN** the orphaned file is NOT deleted (cleanup is deferred to migration)
- **AND** sync completes normally

## Preflight: Replace Staleness Check

### Requirement: `checkSyncStaleness` SHALL compare `fab/.kit/VERSION` against `fab_version` from `config.yaml`

The `checkSyncStaleness()` function in `src/go/fab/internal/preflight/preflight.go` SHALL read `fab_version` from `fab/project/config.yaml` instead of reading `fab/.kit-sync-version`. The comparison logic:

1. Read `fab/.kit/VERSION` → `kitVersion`
2. Read `fab_version` from `fab/project/config.yaml` → `configVersion`
3. If both are non-empty and differ → emit non-blocking stderr warning
4. If either cannot be read → silently skip (no warning, no error)

The function SHALL remain non-blocking — it MUST NOT return an error or alter the exit code.

#### Scenario: Versions match — no warning

- **GIVEN** `fab/.kit/VERSION` contains `0.46.0`
- **AND** `fab/project/config.yaml` has `fab_version: 0.46.0`
- **WHEN** preflight runs
- **THEN** no staleness warning is emitted to stderr

#### Scenario: Versions differ — warning emitted

- **GIVEN** `fab/.kit/VERSION` contains `0.46.0`
- **AND** `fab/project/config.yaml` has `fab_version: 0.45.1`
- **WHEN** preflight runs
- **THEN** stderr receives: `⚠ Skills may be out of sync — run fab sync to refresh (engine 0.46.0, project 0.45.1)`
- **AND** preflight continues normally (exit 0 if no other errors)

#### Scenario: VERSION file missing — no warning

- **GIVEN** `fab/.kit/VERSION` does not exist
- **WHEN** preflight runs
- **THEN** no staleness warning is emitted

#### Scenario: config.yaml unreadable — no warning

- **GIVEN** `fab/.kit/VERSION` contains `0.46.0`
- **AND** `fab/project/config.yaml` cannot be read or has no `fab_version` field
- **WHEN** preflight runs
- **THEN** no staleness warning is emitted (silently skipped)

### Requirement: Warning message SHALL use "project" label

The warning message SHALL read:
```
⚠ Skills may be out of sync — run fab sync to refresh (engine {kitVersion}, project {configVersion})
```

The label changes from `last synced` to `project` to reflect the new comparison source.

#### Scenario: Warning message format

- **GIVEN** `fab/.kit/VERSION` = `0.46.0` and `fab_version` = `0.45.1`
- **WHEN** the staleness warning fires
- **THEN** message is `⚠ Skills may be out of sync — run fab sync to refresh (engine 0.46.0, project 0.45.1)`

## Scaffold: Remove from `.gitignore` templates

### Requirement: `.gitignore` scaffold SHALL NOT include `.kit-sync-version`

The `fab/.kit/scaffold/fragment-.gitignore` template SHALL NOT contain a `fab/.kit-sync-version` line. The project-level `.gitignore` SHALL also have this line removed.

#### Scenario: New project scaffold

- **GIVEN** a user runs `fab init` on a new project
- **WHEN** the `.gitignore` is scaffolded from `fragment-.gitignore`
- **THEN** the generated `.gitignore` does NOT contain `fab/.kit-sync-version`

## Migration: Cleanup orphaned file

### Requirement: A migration SHALL instruct removal of `fab/.kit-sync-version`

A migration file SHALL be created in `fab/.kit/migrations/` to clean up orphaned `fab/.kit-sync-version` files from existing projects.

#### Scenario: Migration removes orphaned file

- **GIVEN** a project with an existing `fab/.kit-sync-version` file
- **WHEN** the user runs `/fab-setup migrations`
- **THEN** the agent deletes `fab/.kit-sync-version`
- **AND** the agent prints confirmation

#### Scenario: Migration handles missing file

- **GIVEN** a project with no `fab/.kit-sync-version` file
- **WHEN** the user runs `/fab-setup migrations`
- **THEN** the agent prints that the file was already absent and skips

## Documentation: Update memory files

### Requirement: Memory files SHALL remove all `.kit-sync-version` references

The following memory files SHALL be updated to remove or replace references to `.kit-sync-version`:

- **`docs/memory/fab-workflow/kit-architecture.md`**: Remove `.kit-sync-version` from version tracking inventory. Update preserved/replaced file lists. Remove sync stamp from directory overview. Update the changelog entry for `260226-koj1-version-staleness-warning`.
- **`docs/memory/fab-workflow/preflight.md`**: Update validation check 1b to describe the new comparison (VERSION vs config.yaml fab_version). Update the `260226-koj1-version-staleness-warning` changelog entry.
- **`docs/memory/fab-workflow/distribution.md`**: Remove `.kit-sync-version` from preserved files list. Update sync staleness detection section to describe the new mechanism.

#### Scenario: Version tracking documentation accuracy

- **GIVEN** the memory files have been updated
- **WHEN** a reader consults `kit-architecture.md` for version file inventory
- **THEN** only three version locations are listed: `fab/.kit/VERSION`, `config.yaml fab_version`, `fab/.kit-migration-version`

## Deprecated Requirements

### `writeSyncVersionStamp` function
**Reason**: Redundant — staleness detection achieved by comparing `fab/.kit/VERSION` against `config.yaml fab_version`
**Migration**: Function and call site deleted from `sync.go`

### `fab/.kit-sync-version` file
**Reason**: No longer written or read by any code path
**Migration**: Orphaned copies cleaned up via migration file

## Design Decisions

1. **Inline YAML parse in preflight over shared config package**: The `checkSyncStaleness` function needs to read one field (`fab_version`) from config.yaml. The preflight package (`src/go/fab/`) and the config package (`src/go/fab-kit/`) are in separate Go modules/binaries. Rather than creating cross-binary shared code, use a minimal inline YAML parse (the `fab` binary already has `gopkg.in/yaml.v3` as a transitive dependency via `statusfile`).
   - *Why*: Keeps the change minimal and avoids coupling binaries.
   - *Rejected*: Sharing `readFabVersion` across binaries — requires module restructuring for one function.

2. **Unified warning message for both mismatch cases**: The old code had two messages ("Skills out of sync" when stamp exists but differs, "Skills may be out of sync" when stamp missing). The new code uses a single message pattern since there's only one comparison.
   - *Why*: Simpler code, clearer message.
   - *Rejected*: Keeping separate messages — no distinct state to differentiate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `.kit-sync-version` file entirely | Confirmed from intake #1 — user explicitly chose this consolidation | S:95 R:85 A:90 D:95 |
| 2 | Certain | Replace staleness check with VERSION vs config.yaml fab_version | Confirmed from intake #2 — equivalent detection, user confirmed approach | S:90 R:80 A:85 D:90 |
| 3 | Certain | Keep scope to sync-version removal only | Confirmed from intake #3 — user explicitly scoped | S:95 R:95 A:95 D:95 |
| 4 | Confident | Ship migration for orphaned file cleanup | Confirmed from intake #4 — migration policy consistency | S:60 R:90 A:75 D:80 |
| 5 | Confident | Warning uses "project" label instead of "last synced" | Confirmed from intake #5 — follows naturally from new source | S:65 R:90 A:80 D:85 |
| 6 | Confident | Inline YAML parse rather than shared config package | Codebase shows separate Go modules for fab vs fab-kit; cross-module sharing would be disproportionate to the need | S:70 R:85 A:80 D:75 |
| 7 | Confident | Single warning message instead of two variants | Old dual-message pattern distinguished "stamp missing" from "stamp differs" — new check has only one comparison path | S:75 R:90 A:85 D:90 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
