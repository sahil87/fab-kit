# Spec: Sync From Cache

**Change**: 260402-ktbg-sync-from-cache
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/setup.md`

## Non-Goals

- Removing `fab/.kit/` from git — separate follow-up change
- Changing `fab init` or `fab upgrade` behavior — they continue copying kit to `fab/.kit/`
- Creating a symlink `fab/.kit → cache` — deferred to follow-up

## Sync Pipeline: Cache-Based Resolution

### Requirement: Sync SHALL resolve kit content from the cache

`fab sync` (via `fab-kit sync`) SHALL resolve all kit content from `~/.fab-kit/versions/{version}/kit/` (or `~/.fab-kit/local-versions/{version}/kit/` when present), where `{version}` is the `fab_version` field from `fab/project/config.yaml`. The existing `CachedKitDir(version)` function SHALL be used for resolution, preserving local-versions priority.

The `fab/.kit/` directory in the repo is NOT read by `fab sync` after this change. It continues to exist (populated by `fab init`/`fab upgrade`) but sync treats the cache as the sole source.

#### Scenario: Normal sync from remote cache
- **GIVEN** `fab/project/config.yaml` contains `fab_version: "0.44.10"`
- **AND** `~/.fab-kit/versions/0.44.10/kit/` exists
- **WHEN** `fab sync` is executed
- **THEN** all kit content (scaffold, skills, hooks) is read from `~/.fab-kit/versions/0.44.10/kit/`
- **AND** `fab/.kit/` in the repo is not read

#### Scenario: Local-versions takes priority
- **GIVEN** both `~/.fab-kit/local-versions/0.44.10/kit/` and `~/.fab-kit/versions/0.44.10/kit/` exist
- **WHEN** `fab sync` is executed
- **THEN** content is read from `~/.fab-kit/local-versions/0.44.10/kit/`

#### Scenario: Config missing fab_version
- **GIVEN** `fab/project/config.yaml` exists but has no `fab_version` field
- **WHEN** `fab sync` is executed
- **THEN** sync exits with error: `"No fab_version in config.yaml. Run 'fab init' to set one."`

#### Scenario: Config missing entirely
- **GIVEN** `fab/project/config.yaml` does not exist
- **WHEN** `fab sync` is executed
- **THEN** sync exits with error: `"Not in a fab-managed repo. Run 'fab init' to set one up."`

### Requirement: Sync SHALL follow a 6-step pipeline

The sync pipeline SHALL execute these steps in order:

1. **Prerequisites check** — validate required tools
2. **Version guard** — ensure `fab_version` <= system `fab-kit` version
3. **Ensure cache** — download if needed
4. **Workspace scaffolding** — directories, scaffold tree-walk, skill deployment, hook sync, version stamp, legacy cleanup
5. **Direnv allow** — `direnv allow` if `.envrc` exists
6. **Project sync scripts** — run `fab/sync/*.sh` in sorted order

#### Scenario: Full sync (no flags)
- **GIVEN** a fab-managed repo with valid config
- **WHEN** `fab sync` is executed without flags
- **THEN** all 6 steps execute in order

## Step 2: Version Guard

### Requirement: Sync SHALL verify system version compatibility

`fab sync` SHALL compare the project's `fab_version` (from `config.yaml`) against the system `fab-kit` binary's embedded version. If `fab_version` is greater than the system version, sync SHALL attempt `fab update` to upgrade the system binaries. If the update fails or the version remains insufficient, sync SHALL exit with an error.

The comparison SHALL use semver ordering (major.minor.patch).

#### Scenario: System version is sufficient
- **GIVEN** `fab_version` is `"0.44.10"` and system `fab-kit` version is `"0.44.10"`
- **WHEN** version guard runs
- **THEN** guard passes, sync continues

#### Scenario: System version is newer
- **GIVEN** `fab_version` is `"0.44.9"` and system `fab-kit` version is `"0.45.0"`
- **WHEN** version guard runs
- **THEN** guard passes, sync continues

#### Scenario: System version is too old
- **GIVEN** `fab_version` is `"0.45.0"` and system `fab-kit` version is `"0.44.10"`
- **WHEN** version guard runs
- **THEN** sync runs `fab update` to upgrade the system binaries
- **AND** if update succeeds and the new version >= `fab_version`, sync continues
- **AND** if update fails or version remains insufficient, sync exits with: `"System fab-kit v{system} is older than project fab_version {project}. Run 'fab update' manually."`

## Step 3: Ensure Cache

### Requirement: Sync SHALL ensure the needed version is cached

`fab sync` SHALL call `EnsureCached(fab_version)` to guarantee the version directory exists at `~/.fab-kit/`. If the version is not cached and cannot be downloaded, sync SHALL exit with the download error.

#### Scenario: Version already cached
- **GIVEN** `~/.fab-kit/versions/0.44.10/` exists with `fab-go` and `kit/`
- **WHEN** ensure cache runs
- **THEN** returns immediately, no download

#### Scenario: Version not cached, download succeeds
- **GIVEN** `~/.fab-kit/versions/0.44.10/` does not exist
- **AND** network is available
- **WHEN** ensure cache runs
- **THEN** version is downloaded from GitHub releases
- **AND** `~/.fab-kit/versions/0.44.10/kit/` is populated

#### Scenario: Download fails
- **GIVEN** version is not cached and network is unavailable
- **WHEN** ensure cache runs
- **THEN** sync exits with error including version and network hint

## Step 4: Workspace Scaffolding

### Requirement: Scaffolding SHALL read from the cached kit directory

All scaffolding operations — directory creation, scaffold tree-walk, skill deployment, hook sync, version stamp, and legacy cleanup — SHALL use the cached kit directory (returned by `CachedKitDir(fab_version)`) as the source for all kit content.

#### Scenario: Scaffold tree-walk from cache
- **GIVEN** cached kit at `~/.fab-kit/versions/0.44.10/kit/scaffold/`
- **WHEN** scaffold tree-walk runs
- **THEN** fragment files and copy-if-absent files are processed from the cache path
- **AND** results are written to the repo root (same destinations as before)

#### Scenario: Skill deployment from cache
- **GIVEN** cached kit at `~/.fab-kit/versions/0.44.10/kit/skills/`
- **WHEN** skill deployment runs
- **THEN** skills are copied/symlinked from the cache path to agent directories
- **AND** OpenCode symlinks point back to `../../fab/.kit/skills/` (unchanged — the repo copy still exists)

### Requirement: Hook sync SHALL be absorbed into workspace scaffolding

The hook registration logic currently in `hooklib.Sync()` (in the `fab` binary's `internal/hooklib` package) SHALL be replicated in the `fab-kit` binary's `internal` package. The `5-sync-hooks.sh` script SHALL be deleted.

Hook sync SHALL run as part of step 4, after skill deployment. It SHALL:
1. Discover `on-*.sh` scripts in `{cachedKitDir}/hooks/`
2. Map scripts to Claude Code events using the same mapping table as `hooklib.DefaultMappings`
3. Merge entries into `.claude/settings.local.json` (idempotent, same deduplication logic)
4. Support path migration (relative → `$CLAUDE_PROJECT_DIR` absolute)

#### Scenario: Hook sync during scaffolding
- **GIVEN** cached kit contains `hooks/on-session-start.sh`, `hooks/on-stop.sh`, `hooks/on-user-prompt.sh`, `hooks/on-artifact-write.sh`
- **WHEN** workspace scaffolding runs
- **THEN** 5 hook entries are merged into `.claude/settings.local.json` (4 scripts, `on-artifact-write.sh` maps to both Write and Edit matchers)
- **AND** no shell script is executed (hook sync is pure Go)

#### Scenario: Hook sync is idempotent
- **GIVEN** `.claude/settings.local.json` already has all hook entries
- **WHEN** workspace scaffolding runs again
- **THEN** output reports "hooks: OK", no file changes

## Command Flags

### Requirement: Sync SHALL support `--shim` and `--project` flags

`fab sync` SHALL accept two mutually exclusive flags:
- `--shim` — execute steps 1-5 only (prerequisites, version guard, ensure cache, scaffolding, direnv)
- `--project` — execute step 6 only (project sync scripts)

When neither flag is provided, all steps 1-6 execute. The flags are mutually exclusive — providing both SHALL produce an error.

#### Scenario: --shim flag
- **GIVEN** a fab-managed repo
- **WHEN** `fab sync --shim` is executed
- **THEN** steps 1-5 run (prerequisites through direnv)
- **AND** step 6 (project sync scripts) is skipped

#### Scenario: --project flag
- **GIVEN** a fab-managed repo
- **WHEN** `fab sync --project` is executed
- **THEN** steps 1-5 are skipped
- **AND** step 6 runs (project sync scripts from `fab/sync/*.sh`)

#### Scenario: Both flags provided
- **GIVEN** a fab-managed repo
- **WHEN** `fab sync --shim --project` is executed
- **THEN** sync exits with error: `"--shim and --project are mutually exclusive"`

## Scaffold Fix: fragment-.envrc

### Requirement: fragment-.envrc SHALL use `fab sync`

`fab/.kit/scaffold/fragment-.envrc` SHALL reference `fab sync` instead of `fab-kit sync` in the `WORKTREE_INIT_SCRIPT` variable.

#### Scenario: New repo gets correct envrc
- **GIVEN** a new repo being initialized
- **WHEN** scaffold tree-walk processes `fragment-.envrc`
- **THEN** `.envrc` contains `export WORKTREE_INIT_SCRIPT="fab sync"`

## Migration: Fix .envrc in Existing Repos

### Requirement: A migration SHALL fix `fab-kit sync` references in `.envrc`

A migration file SHALL be created that replaces `fab-kit sync` with `fab sync` in the project's `.envrc`. The version range SHALL span from the current version to the release version (determined at release time).

#### Scenario: .envrc has old reference
- **GIVEN** `.envrc` contains `export WORKTREE_INIT_SCRIPT="fab-kit sync"`
- **WHEN** the migration runs
- **THEN** the line is changed to `export WORKTREE_INIT_SCRIPT="fab sync"`

#### Scenario: .envrc already correct
- **GIVEN** `.envrc` contains `export WORKTREE_INIT_SCRIPT="fab sync"`
- **WHEN** the migration runs
- **THEN** no changes are made, migration prints "already correct"

## Deprecated Requirements

### `5-sync-hooks.sh` kit-level sync script
**Reason**: Hook sync absorbed into `fab-kit sync` Go code (step 4).
**Migration**: Delete `fab/.kit/sync/5-sync-hooks.sh`. Hook registration happens automatically during workspace scaffolding.

## Design Decisions

1. **Replicate hooklib rather than share**: The hooklib sync logic (~100 lines of Go) is replicated in `fab-kit/internal/` rather than creating a shared module.
   - *Why*: The two Go modules (`src/go/fab-kit` and `src/go/fab`) are separate with independent `go.mod` files. Creating a shared module adds complexity (workspace, import paths) for ~100 lines of self-contained logic. The hooklib in `fab` continues to exist for `fab hook sync` CLI usage; `fab-kit` gets its own copy for internal sync use.
   - *Rejected*: Go workspace with shared module — over-engineering for one small function. Also rejected: having `fab-kit` shell out to `fab hook sync` — defeats the purpose of absorption, adds a process spawn, and reintroduces the `fab` binary dependency.

2. **Version guard auto-updates system binaries**: When the system version is too old, sync attempts `fab update` rather than just failing.
   - *Why*: Reduces friction — the user gets a working sync without manual intervention. The update is via Homebrew (already the installation method), so it's safe and expected.
   - *Rejected*: Fail with instructions only — adds an unnecessary manual step for a common scenario (project upgraded, system not yet).

3. **`--shim` / `--project` flag names**: Using domain terms rather than `--steps-1-5` / `--step-6`.
   - *Why*: Maps to the conceptual split (workspace lifecycle vs project customization). More memorable and self-documenting.
   - *Rejected*: `--no-project` / `--only-project` — double negatives are confusing.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sync reads from `~/.fab-kit/` cache | Confirmed from intake #1 — user explicitly decided | S:95 R:60 A:90 D:95 |
| 2 | Certain | 6-step pipeline ordering | Confirmed from intake #2 — user provided exact ordering | S:95 R:70 A:90 D:95 |
| 3 | Certain | `--shim` and `--project` flags | Confirmed from intake #3 — user specified the split | S:90 R:80 A:85 D:90 |
| 4 | Certain | Hook sync absorbed, `5-sync-hooks.sh` removed | Confirmed from intake #4 — hooks are scaffolding | S:90 R:65 A:85 D:90 |
| 5 | Certain | `fab-kit` executes project sync scripts directly | Confirmed from intake #5 | S:95 R:80 A:90 D:95 |
| 6 | Certain | Fix fragment-.envrc `fab-kit sync` → `fab sync` | Confirmed from intake #6 | S:95 R:85 A:95 D:95 |
| 7 | Certain | Migration for existing .envrc files | Confirmed from intake #7 | S:90 R:75 A:85 D:90 |
| 8 | Certain | No symlink — `fab/.kit/` remains copied directory | Confirmed from intake #8 — deferred to follow-up | S:95 R:85 A:90 D:95 |
| 9 | Certain | `fab init`/`fab upgrade` behavior unchanged | Confirmed from intake #13 | S:95 R:85 A:90 D:95 |
| 10 | Certain | Replicate hooklib in fab-kit rather than shared module | Codebase analysis — separate go.mod files make sharing complex; ~100 lines of self-contained logic | S:85 R:70 A:90 D:80 |
| 11 | Certain | Version guard uses semver comparison of embedded version vs fab_version | Confirmed from intake #14, verified — both use same VERSION at build time | S:90 R:75 A:90 D:85 |
| 12 | Confident | Version guard attempts auto-update before failing | Reasonable UX — reduces friction; but user may not want auto-update in all contexts | S:70 R:65 A:75 D:70 |
| 13 | Confident | OpenCode symlinks continue pointing to `../../fab/.kit/skills/` | Repo copy still exists (fab init/upgrade maintain it); changing symlink targets is follow-up scope | S:75 R:70 A:80 D:75 |
| 14 | Certain | Prerequisites list: git, bash, yq v4+, direnv (jq and gh removed) | Intake step 1 listed git, bash, yq v4+, direnv — jq was used by old hook sync (now Go), gh only needed for download | S:80 R:80 A:85 D:80 |

14 assumptions (12 certain, 2 confident, 0 tentative, 0 unresolved).
