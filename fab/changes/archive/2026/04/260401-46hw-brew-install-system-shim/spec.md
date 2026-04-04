# Spec: Brew Install System Shim

**Change**: 260401-46hw-brew-install-system-shim
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/configuration.md`, `docs/memory/fab-workflow/migrations.md`

## Non-Goals

- Full removal of `fab/.kit/` from repos — future scope; this change removes binaries only
- `fab self-update` — rely on `brew upgrade fab-kit`
- Automatic cache eviction — manual cleanup only; a cleanup command can be added later
- Linux package managers (apt, yum) — Homebrew only for initial distribution

## Homebrew Distribution

### Requirement: Homebrew Formula

A Homebrew formula named `fab-kit` SHALL be published to the `wvrdz/homebrew-tap` tap. The formula SHALL install three binaries to the system PATH: `fab` (shim), `wt` (worktree management), and `idea` (backlog management).

#### Scenario: Fresh install
- **GIVEN** the user has Homebrew installed and the tap is configured
- **WHEN** the user runs `brew install fab-kit`
- **THEN** `fab`, `wt`, and `idea` are installed to the Homebrew bin directory (e.g., `/usr/local/bin/`)
- **AND** all three binaries are executable and respond to `--version`

#### Scenario: Upgrade via Homebrew
- **GIVEN** the user has `fab-kit` installed via Homebrew
- **WHEN** the user runs `brew upgrade fab-kit`
- **THEN** the shim (`fab`), `wt`, and `idea` are updated to the latest formula version
- **AND** the per-version cache is unaffected (cached `fab-go` binaries remain)

### Requirement: Tap Configuration

The Homebrew tap SHALL be hosted at `wvrdz/homebrew-tap` (org-level). Users add it via `brew tap wvrdz/tap`.

#### Scenario: Tap setup
- **GIVEN** the user has Homebrew installed
- **WHEN** the user runs `brew tap wvrdz/tap && brew install fab-kit`
- **THEN** the formula is fetched from `wvrdz/homebrew-tap` and all three binaries are installed

## Shim Architecture

### Requirement: Version-Aware Dispatch

The system `fab` binary SHALL act as a version-aware shim. On every invocation, it SHALL:

1. Walk up from CWD to find `fab/project/config.yaml`
2. Read `fab_version` from `config.yaml`
3. Resolve the matching `fab-go` binary from the local cache
4. Exec the cached binary with full argument passthrough

#### Scenario: Normal dispatch in a fab-managed repo
- **GIVEN** CWD is inside a repo with `fab/project/config.yaml` containing `fab_version: "0.43.0"`
- **AND** `~/.fab-kit/versions/0.43.0/fab-go` exists in cache
- **WHEN** the user runs `fab status current-stage 46hw`
- **THEN** the shim execs `~/.fab-kit/versions/0.43.0/fab-go status current-stage 46hw`
- **AND** all arguments are passed through unmodified

#### Scenario: Version not yet cached — auto-fetch
- **GIVEN** CWD is inside a repo with `fab_version: "0.44.0"`
- **AND** `~/.fab-kit/versions/0.44.0/` does not exist
- **WHEN** the user runs any `fab` command
- **THEN** the shim downloads the `v0.44.0` release from GitHub (`wvrdz/fab-kit`)
- **AND** extracts `fab-go` and `kit/` content to `~/.fab-kit/versions/0.44.0/`
- **AND** then dispatches to the newly cached `fab-go`

#### Scenario: No network during auto-fetch
- **GIVEN** `fab_version: "0.44.0"` is not cached
- **AND** the network is unavailable
- **WHEN** the user runs any `fab` command
- **THEN** the shim exits non-zero with an error message including the version and a hint to check network

#### Scenario: `config.yaml` found but `fab_version` absent
- **GIVEN** `fab/project/config.yaml` exists but has no `fab_version` field
- **WHEN** the user runs any `fab` command
- **THEN** the shim exits with error: `"No fab_version in config.yaml. Run 'fab init' to set one."`

#### Scenario: Not in a fab-managed repo — non-repo commands
- **GIVEN** CWD has no `fab/project/config.yaml` in any parent directory
- **WHEN** the user runs `fab init`, `fab --version`, or `fab --help`
- **THEN** the shim handles the command directly (no dispatch to cached binary)

#### Scenario: Not in a fab-managed repo — repo commands
- **GIVEN** CWD has no `fab/project/config.yaml` in any parent directory
- **WHEN** the user runs `fab resolve` or any repo-requiring command
- **THEN** the shim exits with error: `"Not in a fab-managed repo. Run 'fab init' to set one up."`

### Requirement: Cache Layout

The shim SHALL store versioned artifacts at `~/.fab-kit/versions/{version}/`. Each version directory SHALL contain:

- `fab-go` — the Go backend binary for the current platform
- `kit/` — the full `.kit/` content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

#### Scenario: Cache structure after auto-fetch
- **GIVEN** version `0.43.0` is fetched for the first time
- **WHEN** the download and extraction completes
- **THEN** `~/.fab-kit/versions/0.43.0/fab-go` exists and is executable
- **AND** `~/.fab-kit/versions/0.43.0/kit/VERSION` contains `0.43.0`
- **AND** `~/.fab-kit/versions/0.43.0/kit/skills/` contains the skill files for that version

#### Scenario: Multiple versions coexist
- **GIVEN** repos A and B use `fab_version: "0.43.0"` and `fab_version: "0.42.1"` respectively
- **WHEN** the user switches between repos
- **THEN** the shim dispatches to the correct cached binary for each repo
- **AND** both version directories coexist independently in the cache

### Requirement: No Automatic Cache Eviction

The shim SHALL NOT automatically evict cached versions. Users manage cache cleanup manually (e.g., `rm -rf ~/.fab-kit/versions/0.40.0/`).

#### Scenario: Old versions accumulate
- **GIVEN** the cache contains versions 0.40.0 through 0.43.0
- **WHEN** no cleanup is performed
- **THEN** all versions remain in cache indefinitely

## Shim Subcommands

### Requirement: `fab init`

The shim SHALL handle `fab init` directly (no dispatch to cached binary). `fab init` SHALL:

1. Resolve the latest release version from GitHub
2. Ensure the version is cached (download if not)
3. Copy `~/.fab-kit/versions/{latest}/kit/` → repo's `fab/.kit/`
4. Set `fab_version: "{latest}"` in `fab/project/config.yaml` (creating the file if needed)
5. Run `fab-sync.sh` to deploy skills and set up the workspace

#### Scenario: Init in a new repo
- **GIVEN** CWD is a git repo with no `fab/` directory
- **WHEN** the user runs `fab init`
- **THEN** `fab/.kit/` is populated from the cache
- **AND** `fab/project/config.yaml` is created with `fab_version` set to the latest release
- **AND** `fab-sync.sh` runs to complete workspace setup

#### Scenario: Init in a repo with existing `fab/` but no `fab_version`
- **GIVEN** `fab/project/config.yaml` exists but has no `fab_version` field
- **WHEN** the user runs `fab init`
- **THEN** `fab_version` is added to the existing `config.yaml`
- **AND** `fab/.kit/` is updated from the cache
- **AND** existing project files (`config.yaml` content, `constitution.md`, `changes/`, `memory/`, `specs/`) are NOT overwritten

### Requirement: `fab upgrade`

The shim SHALL handle `fab upgrade [version]` directly (no dispatch to cached binary). `fab upgrade` SHALL:

1. Resolve the target version — latest if no argument, or the explicit version
2. Download to cache if not already present
3. Copy `~/.fab-kit/versions/{version}/kit/` → repo's `fab/.kit/` (atomic swap: extract to temp, verify, then replace)
4. Update `fab_version` in `fab/project/config.yaml` to the new version
5. Run `fab-sync.sh` to deploy skills to agent directories
6. Display version change and migration reminder if needed

`fab upgrade` replaces `fab/.kit/scripts/fab-upgrade.sh`, which SHALL be removed.

#### Scenario: Upgrade to latest
- **GIVEN** the repo has `fab_version: "0.43.0"`
- **WHEN** the user runs `fab upgrade` and the latest release is `0.44.0`
- **THEN** version `0.44.0` is downloaded to cache (if not present)
- **AND** `fab/.kit/` is replaced with content from cache
- **AND** `fab_version` in `config.yaml` is updated to `"0.44.0"`
- **AND** `fab-sync.sh` runs
- **AND** output shows `"Updated: 0.43.0 → 0.44.0"`

#### Scenario: Upgrade to specific version
- **GIVEN** the repo has `fab_version: "0.43.0"`
- **WHEN** the user runs `fab upgrade 0.42.1`
- **THEN** version `0.42.1` is downloaded to cache (if not present)
- **AND** `fab/.kit/` is replaced and `fab_version` updated to `"0.42.1"`

#### Scenario: Already up to date
- **GIVEN** the repo has `fab_version: "0.43.0"` and cache has `0.43.0`
- **WHEN** the user runs `fab upgrade` and the latest release is `0.43.0`
- **THEN** output shows `"Already on the latest version (0.43.0). No update needed."`
- **AND** no files are modified

#### Scenario: Migration reminder after upgrade
- **GIVEN** `fab/.kit-migration-version` is `0.43.0` and the upgrade installed `0.44.0`
- **AND** a migration file exists for `0.43.0-to-0.44.0.md`
- **WHEN** the upgrade completes
- **THEN** output includes `"⚠ Run /fab-setup migrations to update project files (0.43.0 → 0.44.0)"`

## Binary-Free Repo Model

### Requirement: No Binaries in Repo

`fab/.kit/bin/` SHALL contain no binaries. The shell dispatcher (`fab`), Go backend (`fab-go`), worktree manager (`wt`), and backlog manager (`idea`) SHALL all be removed. Only `.gitkeep` remains.

#### Scenario: Clean .kit/bin/
- **GIVEN** the change is applied
- **WHEN** inspecting `fab/.kit/bin/`
- **THEN** only `.gitkeep` exists — no `fab`, `fab-go`, `wt`, or `idea`

### Requirement: Skill Invocation Path Change

All skill files (`fab/.kit/skills/*.md`), the CLI reference (`_cli-fab.md`), and hook scripts (`fab/.kit/hooks/on-*.sh`) SHALL invoke the CLI as `fab <command>` instead of `fab/.kit/bin/fab <command>`.

Affected files:
- `_preamble.md`, `_cli-fab.md`, `_generation.md`
- `fab-new.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`
- `fab-clarify.md`, `fab-switch.md`, `fab-status.md`, `fab-archive.md`
- `fab-setup.md`, `fab-operator7.md`
- `git-branch.md`, `git-pr.md`, `git-pr-review.md`
- All `fab/.kit/hooks/on-*.sh` scripts

#### Scenario: Skill invokes CLI
- **GIVEN** a skill file contains `fab resolve --folder`
- **WHEN** Claude Code reads the skill and executes the command
- **THEN** the system shim resolves the version and dispatches to the cached `fab-go`

#### Scenario: Hook script invokes CLI
- **GIVEN** `fab/.kit/hooks/on-post-tool-use.sh` contains `fab hook artifact-write`
- **WHEN** the hook fires
- **THEN** the system shim dispatches to the cached `fab-go`

### Requirement: `wt` and `idea` System-Only

`wt` and `idea` SHALL be distributed exclusively through the Homebrew formula. They SHALL NOT appear in `fab/.kit/bin/` or in per-version cache directories. They are version-independent standalone utilities.

#### Scenario: `wt` invocation after change
- **GIVEN** `fab-kit` is installed via Homebrew
- **WHEN** the user runs `wt create`
- **THEN** the system-installed `wt` handles the command directly (no version resolution)

### Requirement: `fab-upgrade.sh` Removal

`fab/.kit/scripts/fab-upgrade.sh` SHALL be removed from the kit. The `fab upgrade` shim subcommand replaces its functionality.

#### Scenario: Upgrade after change
- **GIVEN** the new kit version is deployed
- **WHEN** the user wants to upgrade
- **THEN** they run `fab upgrade` (not `fab-upgrade.sh`)

## Sync Pipeline

### Requirement: Remove `4-get-fab-binary.sh`

`fab/.kit/sync/4-get-fab-binary.sh` SHALL be removed. Binary provisioning is handled by the shim's auto-fetch mechanism, not by the sync pipeline.

#### Scenario: Sync pipeline runs without binary download
- **GIVEN** `fab-sync.sh` is invoked
- **WHEN** the sync scripts in `fab/.kit/sync/` are executed in order
- **THEN** step 4 (binary download) is skipped (script does not exist)
- **AND** all other sync steps run normally

### Requirement: Update `5-sync-hooks.sh`

`fab/.kit/sync/5-sync-hooks.sh` SHALL invoke `fab hook sync` instead of `$kit_dir/bin/fab hook sync`. The system shim is on PATH via Homebrew.

#### Scenario: Hook sync after upgrade
- **GIVEN** `fab-sync.sh` runs after `fab upgrade`
- **WHEN** `5-sync-hooks.sh` executes
- **THEN** it calls `fab hook sync` (system shim)
- **AND** the shim dispatches to the cached `fab-go` (version already cached by `fab upgrade`)

### Requirement: Update `1-prerequisites.sh`

`fab-doctor.sh` SHALL add a check for the `fab` system binary on PATH. This is a hard prerequisite.

#### Scenario: Doctor check — fab present
- **GIVEN** `fab-kit` is installed via Homebrew
- **WHEN** `fab-doctor.sh` runs
- **THEN** the `fab` check passes with the shim version

#### Scenario: Doctor check — fab missing
- **GIVEN** the system `fab` binary is not on PATH
- **WHEN** `fab-doctor.sh` runs
- **THEN** the check fails with: `"fab — not found"` and hint: `"Install: brew install fab-kit"`

### Requirement: Update `.envrc` Scaffold

`fab/.kit/scaffold/fragment-.envrc` SHALL remove the `PATH_add fab/.kit/bin` line. The system shim is on PATH via Homebrew. `PATH_add fab/.kit/scripts` SHALL be retained.

#### Scenario: New project .envrc
- **GIVEN** a new project runs `fab-sync.sh`
- **WHEN** the `.envrc` is created from the scaffold
- **THEN** it contains `PATH_add fab/.kit/scripts` but NOT `PATH_add fab/.kit/bin`

## Configuration

### Requirement: `fab_version` Field

`fab/project/config.yaml` SHALL support a new optional field `fab_version` (string) at the top level. When present, the system shim uses it for version resolution. Format: bare semver string (e.g., `"0.43.0"`).

#### Scenario: Config with fab_version
- **GIVEN** `config.yaml` contains `fab_version: "0.43.0"`
- **WHEN** the shim reads it
- **THEN** it resolves and dispatches to the `0.43.0` cached binary

## Constitution

### Requirement: Amend Principle V

Constitution Principle V SHALL be amended from:

> *"The `fab/.kit/` directory MUST work in any project via `cp -r`."*

To:

> *"The `fab/.kit/` directory MUST work in any project via `cp -r`, given the system `fab` binary is installed (`brew install fab-kit`). The system binary provides version-aware execution; `fab/.kit/` provides content (skills, templates, configuration)."*

#### Scenario: Constitution reflects new model
- **GIVEN** the change is applied
- **WHEN** reading `fab/project/constitution.md`
- **THEN** Principle V reflects the system shim prerequisite

## Migration

### Requirement: Migration File for Existing Repos

A migration file SHALL be created at `fab/.kit/migrations/{FROM}-to-{TO}.md` covering the transition to the shim model. The migration SHALL:

1. **Prerequisite gate**: Verify `fab` is on PATH (the system shim). If not, instruct: `"Install fab-kit first: brew tap wvrdz/tap && brew install fab-kit"`
2. **Add `fab_version`**: Write `fab_version: "{version}"` to `fab/project/config.yaml` (set to the current `fab/.kit/VERSION`)
3. **Clean `.envrc`**: Remove the `PATH_add fab/.kit/bin` line if present
4. **Clean `fab/.kit/bin/`**: Remove `fab`, `fab-go`, `wt`, `idea` — only `.gitkeep` remains

#### Scenario: Migration on existing repo
- **GIVEN** an existing repo with binaries in `fab/.kit/bin/` and no `fab_version` in config
- **WHEN** the user runs `/fab-setup migrations` after upgrading
- **THEN** the migration adds `fab_version`, cleans `.envrc`, and removes binaries
- **AND** subsequent `fab` invocations work via the system shim

#### Scenario: Migration without shim installed
- **GIVEN** an existing repo being migrated
- **AND** `fab` (system shim) is NOT on PATH
- **WHEN** the migration runs
- **THEN** the migration stops at the prerequisite gate with install instructions

## Build & Release

### Requirement: Four Go Binaries

The build system SHALL produce 4 Go binaries:

| Binary | Source | Distribution |
|--------|--------|-------------|
| `fab` (shim) | `src/go/shim/` | Homebrew formula |
| `fab-go` | `src/go/fab/` | Per-version cache via GitHub releases |
| `wt` | `src/go/wt/` | Homebrew formula |
| `idea` | `src/go/idea/` | Homebrew formula |

#### Scenario: Local build
- **GIVEN** the developer runs `just build`
- **WHEN** compilation completes
- **THEN** `fab` (shim), `fab-go`, `wt`, and `idea` binaries are produced

### Requirement: Release Archive Structure

Release archives SHALL be structured for the shim to download and cache. Per-platform archives (`kit-{os}-{arch}.tar.gz`) SHALL contain:
- `.kit/bin/fab-go` — the versioned Go backend binary
- `.kit/` — all content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

The shim extracts `fab-go` to `~/.fab-kit/versions/{version}/fab-go` and the rest to `~/.fab-kit/versions/{version}/kit/`.

#### Scenario: Shim downloads and caches a release
- **GIVEN** `fab_version: "0.44.0"` and version is not cached
- **WHEN** the shim auto-fetches
- **THEN** it downloads `kit-{os}-{arch}.tar.gz` for the current platform
- **AND** extracts `fab-go` to `~/.fab-kit/versions/0.44.0/fab-go`
- **AND** extracts `.kit/` content to `~/.fab-kit/versions/0.44.0/kit/`

## Deprecated Requirements

### Backend Override Mechanism

**Reason**: The `FAB_BACKEND` env var and `.fab-backend` file mechanism is no longer needed. The Go backend is the only backend. The shim dispatches to `fab-go` directly.
**Migration**: Remove references to `FAB_BACKEND` and `.fab-backend` from documentation and scripts. No user action needed.

### Shell Dispatcher at `fab/.kit/bin/fab`

**Reason**: The 1KB shell script that delegated to `fab-go` is replaced by the system shim. The shim provides the same dispatching with added version resolution.
**Migration**: Remove the file. Skills already updated to call `fab` instead of `fab/.kit/bin/fab`.

## Design Decisions

1. **Binary-free repo, system-managed execution**: The repo holds content only (markdown, yaml); the system shim resolves and runs the correct binary from cache.
   - *Why*: Eliminates binary duplication across repos, removes platform-specific artifacts from version control, simplifies upgrades to a version bump in config.yaml.
   - *Rejected*: Binary in repo (PR #287 model) — redundant when the shim already manages versions.

2. **Cache stores binary + content**: Each cached version includes both `fab-go` and the full `.kit/` content, not just the binary.
   - *Why*: `fab upgrade` needs the content to populate the repo's `fab/.kit/`. Storing both enables a single download for both dispatch and upgrade.
   - *Rejected*: Binary-only cache — would still need a separate download mechanism for content during upgrade.

3. **`fab upgrade` as shim subcommand**: The shim handles upgrade directly, replacing the `fab-upgrade.sh` shell script.
   - *Why*: The shim already has download/cache logic. Upgrade is a natural extension. Eliminates a separate script with duplicated download logic.
   - *Rejected*: Keep `fab-upgrade.sh` alongside the shim — duplication of download logic, confusing which to use.

4. **Formula name `fab-kit`, binary name `fab`**: Homebrew formula uses `fab-kit` to avoid collision with Python Fabric's `fab` formula, while the installed binary is still `fab`.
   - *Why*: Users type `fab` (short), but Homebrew needs a unique formula name.
   - *Rejected*: `fab` as formula name — collides with Fabric.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Homebrew formula name is `fab-kit`, binary name is `fab` | Confirmed from intake #1 — user explicitly chose this split | S:95 R:90 A:95 D:95 |
| 2 | Certain | `wt` and `idea` are system-only, not per-repo | Confirmed from intake #2 — standalone utilities, version-independent | S:95 R:80 A:90 D:95 |
| 3 | Certain | No Go binary in repo — `fab/.kit/bin/` is empty | Confirmed from intake #3 — user proposed and confirmed | S:95 R:70 A:90 D:90 |
| 4 | Certain | Version pinned per-repo via `config.yaml` `fab_version` field | Confirmed from intake #4 — version-manager pattern | S:90 R:70 A:85 D:85 |
| 5 | Certain | Skills call `fab` not `fab/.kit/bin/fab` | Confirmed from intake #5 — direct consequence of binary-free repo | S:95 R:65 A:95 D:95 |
| 6 | Certain | Constitution Principle V amended to require system shim | Confirmed from intake #6 — user confirmed | S:95 R:60 A:90 D:90 |
| 7 | Confident | Cache lives at `~/.fab-kit/versions/` | Confirmed from intake #7 — standard location; `~/.cache/fab-kit/` also viable | S:60 R:90 A:70 D:60 |
| 8 | Confident | Shim downloads from GitHub releases | Confirmed from intake #8 — natural fit with existing repo | S:65 R:85 A:75 D:70 |
| 9 | Certain | Homebrew tap at `wvrdz/homebrew-tap` | Confirmed from intake #9 — org-level tap | S:95 R:90 A:50 D:50 |
| 10 | Certain | Error when `fab_version` absent from `config.yaml` | Confirmed from intake #10 — strict mode with actionable message | S:95 R:70 A:50 D:45 |
| 11 | Certain | `fab init` in scope | Confirmed from intake #11 — primary use case for the shim | S:95 R:70 A:80 D:90 |
| 12 | Certain | No automatic cache eviction | Confirmed from intake #12 — manual cleanup only | S:95 R:90 A:80 D:90 |
| 13 | Certain | No `fab self-update` — rely on `brew upgrade fab-kit` | Confirmed from intake #13 | S:95 R:90 A:85 D:90 |
| 14 | Certain | Full `.kit/` removal is future scope | Confirmed from intake #14 — this change is binary removal only | S:95 R:85 A:95 D:95 |
| 15 | Certain | Cache stores `fab-go` + full `.kit/` content per version | Confirmed from intake #15 — content needed for `fab upgrade` | S:95 R:80 A:90 D:90 |
| 16 | Certain | `fab upgrade` replaces `fab-upgrade.sh` | Confirmed from intake #16 — shim handles full lifecycle | S:95 R:75 A:90 D:90 |
| 17 | Certain | 4 Go binaries: `fab` (shim), `fab-go`, `wt`, `idea` | Confirmed from intake #17 — shell dispatcher removed | S:95 R:85 A:95 D:95 |
| 18 | Certain | Sync: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` calls `fab hook sync` | Confirmed from intake #18 | S:95 R:80 A:90 D:95 |
| 19 | Certain | `.envrc` scaffold removes `PATH_add fab/.kit/bin` | Confirmed from intake #19 | S:95 R:85 A:95 D:95 |
| 20 | Certain | Migration required for existing repos | Confirmed from intake #20 — adds `fab_version`, cleans envrc and bin dir | S:95 R:65 A:85 D:90 |
| 21 | Certain | Backend override mechanism (`FAB_BACKEND`, `.fab-backend`) deprecated | Spec-level: Go is the only backend; override mechanism is dead code | S:90 R:85 A:95 D:95 |
| 22 | Confident | Shim extracts platform archive, splits into `fab-go` + `kit/` in cache | Spec-level: reuses existing per-platform archive format; shim handles extraction | S:70 R:80 A:80 D:75 |

22 assumptions (19 certain, 3 confident, 0 tentative, 0 unresolved).
