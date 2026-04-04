# Spec: Relocate Kit to System Cache

**Change**: 260402-gnx5-relocate-kit-to-system-cache
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/setup.md`, `docs/memory/fab-workflow/preflight.md`, `docs/memory/fab-workflow/migrations.md`, `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/configuration.md`

## Non-Goals

- Changing how `fab sync` deploys skills to `.claude/skills/` — skill deployment mechanism is unchanged
- Changing the cache layout at `~/.fab-kit/versions/<version>/` — it already has the right structure
- Modifying the `fab` router dispatch logic — it already resolves `fab-go` from cache
- Updating change archive documents — historical references to `fab/.kit/` in completed changes are left as-is

## Kit Path Resolution

### Requirement: fab-go SHALL resolve kit via exe-sibling

`fab-go` SHALL resolve its kit directory by finding the `kit/` directory sibling to its own executable. This replaces all `filepath.Join(fabRoot, ".kit", ...)` patterns in the `fab-go` codebase.

A shared utility `internal/kitpath/kitpath.go` SHALL provide:
- `KitDir() (string, error)` — resolves `os.Executable()`, evaluates symlinks, returns `filepath.Join(dir, "kit")`

All `fab-go` internal packages that currently construct `fab/.kit/` paths SHALL use `kitpath.KitDir()` instead.

#### Scenario: fab-go resolves kit from cache
- **GIVEN** `fab-go` is installed at `~/.fab-kit/versions/0.47.0/fab-go`
- **AND** kit content exists at `~/.fab-kit/versions/0.47.0/kit/`
- **WHEN** `fab-go` calls `kitpath.KitDir()`
- **THEN** it returns `~/.fab-kit/versions/0.47.0/kit`

#### Scenario: fab-go run via symlink
- **GIVEN** `fab-go` is symlinked from another location
- **WHEN** `kitpath.KitDir()` is called
- **THEN** it evaluates symlinks before resolving the sibling, returning the real kit path

#### Scenario: kit directory missing
- **GIVEN** `fab-go` exists but `kit/` sibling does not
- **WHEN** `kitpath.KitDir()` is called
- **THEN** it returns an error (the caller decides how to handle it)

### Requirement: fab-kit SHALL resolve kit via version from config

`fab-kit` (system binary, not in the versions directory) SHALL resolve kit content by reading `fab_version` from `fab/project/config.yaml` and looking up `~/.fab-kit/versions/{version}/kit/`. This is the same version resolution the `fab` router already performs.

For commands that don't require `config.yaml` (e.g., `fab init`), `fab-kit` SHALL resolve the latest cached version or download it.

#### Scenario: fab-kit sync resolves kit from config version
- **GIVEN** `fab/project/config.yaml` contains `fab_version: "0.47.0"`
- **AND** `~/.fab-kit/versions/0.47.0/kit/` exists
- **WHEN** `fab-kit sync` runs
- **THEN** it reads skills from `~/.fab-kit/versions/0.47.0/kit/skills/` for deployment

#### Scenario: fab init resolves latest version
- **GIVEN** no `config.yaml` exists (new project)
- **WHEN** `fab init` runs
- **THEN** it resolves the latest cached (or downloaded) version and uses that kit

## fab kit-path Command

### Requirement: fab SHALL expose a kit-path subcommand

A new `fab kit-path` command SHALL output the absolute path to the resolved kit directory. It SHALL be a `fab-go` subcommand (routed via the `fab` router like all workflow commands).

The command:
- Reads `fab_version` from `fab/project/config.yaml`
- Resolves `~/.fab-kit/versions/{version}/kit/`
- Prints the absolute path to stdout (no trailing newline, no decoration)
- Exits 0 on success, non-zero with error on stderr if resolution fails

#### Scenario: Agent resolves kit path
- **GIVEN** `fab/project/config.yaml` contains `fab_version: "0.47.0"`
- **AND** `~/.fab-kit/versions/0.47.0/kit/` exists
- **WHEN** `fab kit-path` is executed
- **THEN** stdout contains `/home/user/.fab-kit/versions/0.47.0/kit`
- **AND** exit code is 0

#### Scenario: Kit path used by skill to read template
- **GIVEN** an agent is executing `/fab-new`
- **WHEN** the agent needs to read the intake template
- **THEN** it runs `fab kit-path` and reads `{output}/templates/intake.md`

#### Scenario: Version not cached
- **GIVEN** `fab_version: "0.47.0"` but `~/.fab-kit/versions/0.47.0/` does not exist
- **WHEN** `fab kit-path` is executed
- **THEN** it exits non-zero with an error suggesting `fab sync` or `fab upgrade`

## Go Binary Path Updates

### Requirement: change.New SHALL read template from kit cache

`src/go/fab/internal/change/change.go` line 67 currently reads:
```go
templatePath := filepath.Join(fabRoot, ".kit", "templates", "status.yaml")
```
This SHALL be replaced with `kitpath.KitDir()` resolution:
```go
kitDir, err := kitpath.KitDir()
templatePath := filepath.Join(kitDir, "templates", "status.yaml")
```

#### Scenario: Create new change reads template from cache
- **GIVEN** `fab-go` runs from `~/.fab-kit/versions/0.47.0/fab-go`
- **WHEN** `fab change new --slug my-change` is executed
- **THEN** the status.yaml template is read from `~/.fab-kit/versions/0.47.0/kit/templates/status.yaml`

### Requirement: Preflight SHALL read VERSION from kit cache

`src/go/fab/internal/preflight/preflight.go` `checkSyncStaleness()` currently reads:
```go
versionFile := filepath.Join(fabRoot, ".kit", "VERSION")
```
This SHALL resolve VERSION from the exe-sibling kit directory.

#### Scenario: Preflight staleness check uses cache VERSION
- **GIVEN** `fab-go` 0.47.0 runs from cache
- **AND** `config.yaml` has `fab_version: "0.47.0"`
- **WHEN** preflight runs the staleness check
- **THEN** it reads VERSION from `~/.fab-kit/versions/0.47.0/kit/VERSION`
- **AND** compares against `fab_version` in config (same version = no warning)

### Requirement: fabhelp SHALL scan skills from kit cache

`src/go/fab/cmd/fab/fabhelp.go` line 67 currently constructs:
```go
kitDir := filepath.Join(fabRoot, ".kit")
```
This SHALL use `kitpath.KitDir()`.

#### Scenario: fab fab-help scans skills from cache
- **GIVEN** `fab-go` runs from cache
- **WHEN** `fab fab-help` is executed
- **THEN** it scans `~/.fab-kit/versions/{version}/kit/skills/*.md` for frontmatter

## Hook Inlining

### Requirement: Hook sync SHALL register inline fab commands

`hooklib.Sync()` currently constructs hook commands as:
```go
cmd := `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/` + m.Script
```

This SHALL be replaced with inline `fab hook` commands. The `DefaultMappings` table SHALL map directly to `fab hook <subcommand>` commands instead of shell script paths:

| Event | Current command | New command |
|-------|----------------|-------------|
| SessionStart | `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-session-start.sh` | `fab hook session-start` |
| Stop | `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-stop.sh` | `fab hook stop` |
| UserPromptSubmit | `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-user-prompt.sh` | `fab hook user-prompt` |
| PostToolUse (Write) | `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-artifact-write.sh` | `fab hook artifact-write` |
| PostToolUse (Edit) | `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-artifact-write.sh` | `fab hook artifact-write` |

The `hooksDir` parameter to `Sync()` is no longer needed for script discovery — the mappings are hardcoded. `Sync()` SHALL still accept the `settingsPath` parameter.

#### Scenario: Fresh project hook sync
- **GIVEN** a new project with no `.claude/settings.local.json`
- **WHEN** `fab hook sync` runs
- **THEN** settings.local.json is created with hook entries using `fab hook <subcommand>` commands
- **AND** no `fab/.kit/hooks/` directory is referenced

#### Scenario: Existing project with old-style hooks
- **GIVEN** `.claude/settings.local.json` contains `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-stop.sh`
- **WHEN** `fab hook sync` runs
- **THEN** the old command is replaced with `fab hook stop`

### Requirement: Hook shell scripts SHALL be removed from kit

The `hooks/` directory SHALL be removed from `src/kit/` (the relocated kit source). Hook scripts (`on-session-start.sh`, `on-stop.sh`, `on-user-prompt.sh`, `on-artifact-write.sh`) are no longer needed since hooks are inlined.

#### Scenario: Kit archive has no hooks directory
- **GIVEN** a release is built from `src/kit/`
- **WHEN** the kit archive is packaged
- **THEN** it does not contain a `hooks/` directory

## Eliminate kit.conf

### Requirement: build-type feature SHALL be removed

The test-build guard in `_preamble.md` (Section "Test-Build Guard") that reads `fab/.kit/kit.conf` and stops on `build-type=test` SHALL be removed entirely. No replacement mechanism is needed.

#### Scenario: Skill no longer checks kit.conf
- **GIVEN** a skill invocation
- **WHEN** the preamble is loaded
- **THEN** it does NOT read `kit.conf` or check `build-type`
- **AND** proceeds directly to context loading

### Requirement: repo SHALL be hardcoded in Go binary

The `repo` field currently read from `kit.conf` SHALL be replaced with a Go constant:

```go
const defaultRepo = "sahil87/fab-kit"
```

All code that previously read `repo` from `kit.conf` SHALL use this constant instead.

#### Scenario: Download resolves repo from constant
- **GIVEN** `fab-kit` needs to download a release
- **WHEN** it constructs the GitHub API URL
- **THEN** it uses the hardcoded `sahil87/fab-kit` constant

## Skill File Updates

### Requirement: Skills SHALL reference templates via fab kit-path

All skill files that reference `fab/.kit/templates/` SHALL be updated to use `$(fab kit-path)/templates/`. This applies to:

- `_generation.md` — intake, spec, tasks, checklist template reads
- `_preamble.md` — remove test-build guard, remove `kit.conf` reference
- `_cli-fab.md` — update any `fab/.kit/` references in command documentation
- `fab-setup.md` — migration discovery via `$(fab kit-path)/migrations/`
- `fab-help.md` — if it references kit paths

#### Scenario: Agent reads intake template via kit-path
- **GIVEN** an agent is generating an intake
- **WHEN** it follows `_generation.md` instructions
- **THEN** it runs `fab kit-path` and reads `{output}/templates/intake.md`

### Requirement: Skills SHALL reference shared skills from .claude/skills/

Skill files that reference `fab/.kit/skills/_preamble.md` etc. SHALL be updated to reference them from their deployed location. Since `fab sync` deploys all skills (including `_` prefixed ones) to `.claude/skills/`, the preamble instruction at the top of each skill changes from:

```
Read `fab/.kit/skills/_preamble.md` first
```

to the equivalent deployed path. The `_preamble.md` file itself and other `_` prefixed files continue to be deployed to `.claude/skills/` via `fab sync`.

#### Scenario: Skill reads preamble from deployed location
- **GIVEN** `fab sync` has deployed skills to `.claude/skills/`
- **WHEN** a skill is invoked
- **THEN** it reads the preamble from the deployed `.claude/skills/` location
- **AND** does not reference `fab/.kit/`

## Source Repo Layout

### Requirement: fab/.kit/ SHALL be moved to src/kit/ in the source repo

The kit content directory SHALL be relocated from `fab/.kit/` to `src/kit/` in the fab-kit development repository. The directory structure is preserved — only the parent path changes.

`kit.conf` is NOT included in the move (it is eliminated entirely).
`hooks/` directory is NOT included in the move (hook scripts are eliminated).

#### Scenario: Source repo layout after move
- **GIVEN** the fab-kit development repo
- **WHEN** the move is complete
- **THEN** `src/kit/skills/`, `src/kit/templates/`, `src/kit/migrations/`, `src/kit/scaffold/`, `src/kit/schemas/`, `src/kit/VERSION` exist
- **AND** `fab/.kit/` does not exist
- **AND** `src/kit/hooks/` does not exist
- **AND** `src/kit/kit.conf` does not exist

## Build and Release

### Requirement: Build scripts SHALL reference src/kit/

| File | Current | New |
|------|---------|-----|
| `justfile` | `cat fab/.kit/VERSION` | `cat src/kit/VERSION` |
| `justfile` | `rsync -a --delete --exclude='bin/' fab/.kit/ ...` | `rsync -a --delete src/kit/ ...` |
| `justfile` | `cp -a fab/.kit/. dist/kit/` | `cp -a src/kit/. dist/kit/` |
| `scripts/release.sh` | `kit_dir="$repo_root/fab/.kit"` | `kit_dir="$repo_root/src/kit"` |
| `.github/copilot-code-review.yml` | `fab/.kit/**` | `src/kit/**` |

#### Scenario: Release packages from src/kit/
- **GIVEN** the release script runs
- **WHEN** it assembles the kit archive
- **THEN** it reads from `src/kit/` (not `fab/.kit/`)

### Requirement: .gitignore SHALL be cleaned

Remove entries that reference `fab/.kit/`:
- `fab/.kit/bin/*`
- `!fab/.kit/bin/.gitkeep`

#### Scenario: Clean gitignore
- **GIVEN** `.gitignore` has been updated
- **THEN** it contains no `fab/.kit/` entries

## fab-kit Sync and Init Updates

### Requirement: fab init SHALL NOT copy kit to project

`fab-kit init` currently copies `~/.fab-kit/versions/{latest}/kit/` to `fab/.kit/` in the project. After this change, `fab init` SHALL:
1. Resolve the latest version (download if needed)
2. Set `fab_version` in `fab/project/config.yaml`
3. Run scaffold (directories, fragment merges, copy-if-absent) from the cache kit
4. Deploy skills to `.claude/skills/` via sync
5. NOT create `fab/.kit/` in the project

#### Scenario: Init creates project without fab/.kit/
- **GIVEN** a new repository
- **WHEN** `fab init` runs
- **THEN** `fab/project/`, `fab/changes/`, `.claude/skills/` are created
- **AND** `fab/.kit/` does NOT exist in the project

### Requirement: fab upgrade SHALL NOT copy kit to project

`fab upgrade` currently copies kit from cache to `fab/.kit/`. After this change, `fab upgrade` SHALL:
1. Resolve target version (download if needed)
2. Update `fab_version` in `config.yaml`
3. Re-run sync (deploy updated skills)
4. NOT touch `fab/.kit/` (it doesn't exist in the project)

#### Scenario: Upgrade updates config and re-syncs skills
- **GIVEN** a project at version 0.46.0
- **WHEN** `fab upgrade 0.47.0` runs
- **THEN** `config.yaml` is updated to `fab_version: "0.47.0"`
- **AND** skills are re-deployed from `~/.fab-kit/versions/0.47.0/kit/skills/`
- **AND** no `fab/.kit/` directory exists in the project

## Preflight Staleness

### Requirement: Preflight staleness SHALL compare cache VERSION vs config

The staleness check (already updated by #307) compares `fab/.kit/VERSION` vs `config.yaml`'s `fab_version`. After this change, the VERSION source moves to the exe-sibling kit in cache. The comparison logic is unchanged.

#### Scenario: No staleness when versions match
- **GIVEN** `fab-go` 0.47.0 runs from cache with `kit/VERSION` = "0.47.0"
- **AND** `config.yaml` has `fab_version: "0.47.0"`
- **WHEN** preflight runs
- **THEN** no staleness warning is emitted

## User-Facing Messages

### Requirement: CLI messages SHALL reflect cache-based model

Update user-facing strings:

| Binary | Current message | New message |
|--------|----------------|-------------|
| `fab-kit` | `"Upgrade fab/.kit/ to a specific or latest version"` | `"Upgrade to a specific or latest version"` |
| `fab-kit` | `"Populating fab/.kit/..."` | `"Setting up project..."` |
| `fab-kit` | `"Updating fab/.kit/..."` | `"Upgrading to {version}..."` |

#### Scenario: Init shows updated message
- **WHEN** `fab init` runs
- **THEN** it prints `"Setting up project..."` (not `"Populating fab/.kit/..."`)

## Migration

### Requirement: Ship migration for existing users

A migration file SHALL be shipped (versioned appropriately, e.g., `0.46.0-to-0.47.0.md`) that:

1. **Prerequisite**: Verify `~/.fab-kit/versions/{version}/kit/` exists (where version = `fab_version` from config)
2. **Inline hooks**: Update `.claude/settings.local.json` — replace `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh` commands with `fab hook <subcommand>` inline commands
3. **Remove fab/.kit/**: Delete the `fab/.kit/` directory from the project
4. **Clean .envrc**: Remove `PATH_add fab/.kit/scripts` line if present
5. **Clean .gitignore**: Remove `fab/.kit/bin/*` and `!fab/.kit/bin/.gitkeep` entries

#### Scenario: Migration on existing project
- **GIVEN** a project with `fab/.kit/` and `fab_version: "0.47.0"` in config
- **AND** `~/.fab-kit/versions/0.47.0/kit/` exists
- **WHEN** `/fab-setup migrations` runs the migration
- **THEN** hooks are inlined in settings.local.json
- **AND** `fab/.kit/` is deleted
- **AND** `.envrc` and `.gitignore` are cleaned

#### Scenario: Migration without cache
- **GIVEN** a project with `fab_version: "0.47.0"` but no cache at that version
- **WHEN** the migration runs
- **THEN** it stops at the prerequisite with guidance: `"Cache not found. Run 'fab sync' first."`

## Deprecated Requirements

### Hook Shell Scripts
**Reason**: Replaced by inline `fab hook <subcommand>` commands registered directly in settings.local.json.
**Migration**: `fab hook sync` now registers inline commands. Migration file updates existing registrations.

### kit.conf
**Reason**: `build-type` feature removed; `repo` hardcoded in binary.
**Migration**: File is simply deleted as part of the source repo move (not included in `src/kit/`).

### fab/.kit/ in User Projects
**Reason**: Kit content served from system cache at `~/.fab-kit/versions/<version>/kit/`.
**Migration**: Migration file removes `fab/.kit/` after verifying cache is populated.

## Design Decisions

1. **Exe-sibling resolution over environment variable**: Using `os.Executable()` to find kit is self-contained — no env var to set, no config to read. The binary and its content are always co-located.
   - *Why*: Zero-configuration path resolution. Works in any execution context (direct, symlinked, worktree).
   - *Rejected*: `FAB_KIT_DIR` env var — requires setup, easy to misconfigure, doesn't survive across execution contexts.

2. **Standalone `fab kit-path` over preflight YAML field**: Not all skills run preflight (fab-new, fab-discuss, fab-help, fab-setup). A standalone command works universally.
   - *Why*: Agent-agnostic, works for any agent that can run a shell command. No coupling to preflight.
   - *Rejected*: Preflight YAML field — insufficient coverage (not all skills run preflight), would require fab sync changes.

3. **Inline hooks over cache-path hooks**: `fab hook <subcommand>` is simpler than resolving cache paths for shell scripts. Eliminates the hooks directory entirely.
   - *Why*: Hook scripts were single-line wrappers around `fab hook`. Removing the indirection simplifies the architecture.
   - *Rejected*: Resolving hook scripts from cache path — adds complexity for no benefit when the scripts are trivial.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Kit content co-located with fab-go at `~/.fab-kit/versions/<version>/kit/` | Confirmed from intake #1 — user specified exe-sibling resolution | S:95 R:60 A:95 D:95 |
| 2 | Certain | Source directory renamed from `fab/.kit/` to `src/kit/` in dev repo | Confirmed from intake #2 — user specified explicitly | S:95 R:70 A:90 D:95 |
| 3 | Certain | `kit.conf` eliminated entirely (build-type removed, repo hardcoded) | Confirmed from intake #3 — user confirmed both eliminations | S:95 R:80 A:90 D:95 |
| 4 | Certain | Templates accessed via `fab kit-path`, not synced to project | Confirmed from intake #4 — user chose cache access for agent-agnosticism | S:90 R:70 A:85 D:90 |
| 5 | Certain | User projects will no longer contain `fab/.kit/` | Confirmed from intake #5 — explicit goal | S:95 R:50 A:90 D:95 |
| 6 | Certain | `build-type` feature removed entirely | Confirmed from intake #6 | S:90 R:85 A:90 D:95 |
| 7 | Certain | `repo` hardcoded as Go constant | Confirmed from intake #7 | S:85 R:80 A:85 D:90 |
| 8 | Certain | Shared `kitpath.KitDir()` utility using `os.Executable()` + sibling resolution | Confirmed from intake #8 — standard Go pattern | S:90 R:80 A:85 D:85 |
| 9 | Certain | `fab-kit` resolves kit via version from config.yaml | Confirmed from intake #9 — system binary, not in versions dir | S:85 R:70 A:80 D:80 |
| 10 | Certain | Skill reference files deployed to `.claude/skills/` via `fab sync` | Confirmed from intake #10 — no change to current behavior | S:90 R:75 A:80 D:85 |
| 11 | Certain | Hook scripts replaced with inline `fab hook <subcommand>` commands | Confirmed from intake #11 — user chose inline | S:90 R:75 A:85 D:90 |
| 12 | Certain | No sync staleness stamp file needed | Confirmed from intake #12 — #307 already handles via VERSION vs config | S:95 R:90 A:95 D:95 |
| 13 | Certain | `fab kit-path` as standalone command only | Confirmed from intake #13 — not all skills run preflight | S:90 R:85 A:85 D:90 |
| 14 | Certain | Migration removes `fab/.kit/` automatically with cache verification | Confirmed from intake #14 — standard migration behavior | S:85 R:60 A:75 D:80 |

14 assumptions (14 certain, 0 confident, 0 tentative, 0 unresolved).
