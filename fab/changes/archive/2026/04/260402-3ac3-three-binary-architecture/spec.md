# Spec: Three-Binary Architecture

**Change**: 260402-3ac3-three-binary-architecture
**Created**: 2026-04-02
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Removing `fab/.kit/` from repos — this change prepares for that future but does not implement it; sync still reads from in-repo `fab/.kit/`
- Changing fab-go subcommands or behavior — fab-go source and module are untouched
- Changing wt or idea binaries — their build and distribution are unchanged
- Multi-agent sync format changes — skill deployment formats (copy vs symlink per agent) are ported as-is

## Binary Architecture

### Requirement: Three-Binary Split

The system SHALL provide three distinct binaries: `fab` (router), `fab-kit` (workspace lifecycle), and `fab-go` (workflow engine). Each binary SHALL be independently executable with its own `--help` / `-h` flag.

#### Scenario: Independent invocation
- **GIVEN** all three binaries are installed
- **WHEN** a user runs `fab-kit -h`
- **THEN** fab-kit displays its own help (init, upgrade, sync) without invoking fab-go

#### Scenario: Independent invocation of fab-go
- **GIVEN** fab-go is available in the version cache
- **WHEN** a user runs `fab-go -h`
- **THEN** fab-go displays its own help (resolve, status, preflight, etc.) without invoking fab or fab-kit

### Requirement: Router Negative-Match Dispatch

The `fab` router SHALL maintain a static allowlist of fab-kit commands: `init`, `upgrade`, `sync`, `--version`, `-v`, `--help`, `-h`, `help`. Commands matching this list SHALL be dispatched to `fab-kit` via `syscall.Exec`. All other commands SHALL be dispatched to the version-resolved `fab-go` via `syscall.Exec`.

#### Scenario: fab-kit command routing
- **GIVEN** a user is in any directory
- **WHEN** they run `fab init`
- **THEN** the router execs `fab-kit init` with all arguments passed through

#### Scenario: fab-go command routing
- **GIVEN** a user is in a fab-managed repo with `fab_version: "0.45.0"` in config.yaml
- **WHEN** they run `fab status 3ac3`
- **THEN** the router reads config.yaml, ensures v0.45.0 fab-go is cached, and execs `fab-go status 3ac3`

#### Scenario: Not in a fab-managed repo, workflow command
- **GIVEN** no `fab/project/config.yaml` exists in the directory hierarchy
- **WHEN** a user runs `fab status`
- **THEN** the router exits non-zero with: "Not in a fab-managed repo. Run 'fab init' to set one up."

### Requirement: Composed Help

`fab help` (and `fab --help`) SHALL compose help output from both sub-binaries. Workspace commands (from fab-kit) SHALL always be shown. Workflow commands (from fab-go) SHALL be shown only when inside a fab-managed repo.

#### Scenario: Help inside a repo
- **GIVEN** `fab/project/config.yaml` exists with a valid `fab_version`
- **WHEN** a user runs `fab help`
- **THEN** output shows workspace commands (init, upgrade, sync) and workflow commands (resolve, status, preflight, etc.) in labeled groups

#### Scenario: Help outside a repo
- **GIVEN** no `fab/project/config.yaml` exists
- **WHEN** a user runs `fab help`
- **THEN** output shows workspace commands only

## Source Layout

### Requirement: Single Go Module, Two Binaries

The `fab` router and `fab-kit` binaries SHALL share a single Go module at `src/go/fab-kit/`. The module SHALL contain two `cmd/` entries: `cmd/fab/main.go` (router) and `cmd/fab-kit/main.go` (workspace lifecycle). Both SHALL share `internal/` packages for cache, download, and config resolution.

#### Scenario: Module structure
- **GIVEN** the source directory `src/go/fab-kit/`
- **WHEN** a developer inspects the layout
- **THEN** they see `cmd/fab/`, `cmd/fab-kit/`, and `internal/` as siblings under one `go.mod`

#### Scenario: Building both binaries
- **GIVEN** the developer is in `src/go/fab-kit/`
- **WHEN** they run `go build ./cmd/fab` and `go build ./cmd/fab-kit`
- **THEN** two separate binaries are produced, each importing from the shared `internal/` package

### Requirement: Rename src/go/shim/ to src/go/fab-kit/

The existing shim source directory SHALL be renamed from `src/go/shim/` to `src/go/fab-kit/`. The Go module path in `go.mod` SHALL be updated accordingly. All internal package imports SHALL be updated to reflect the new module path.

#### Scenario: Source rename
- **GIVEN** the current source at `src/go/shim/`
- **WHEN** the rename is applied
- **THEN** `src/go/fab-kit/go.mod` declares the new module path and all `import` statements reference it

## fab-kit sync

### Requirement: Absorb fab-sync.sh into fab-kit sync

`fab-kit sync` SHALL replace `fab-sync.sh` and `fab/.kit/sync/2-sync-workspace.sh` as a clean cut — the shell scripts SHALL be removed. The Go implementation SHALL replicate all sync behavior.

#### Scenario: Clean replacement
- **GIVEN** a project previously using `fab-sync.sh`
- **WHEN** a user runs `fab sync` (or `fab-kit sync` directly)
- **THEN** the workspace is fully synced: directories created, scaffold applied, skills deployed, stale files cleaned, version stamp written
- **AND** `fab-sync.sh` and `fab/.kit/sync/2-sync-workspace.sh` do not exist

### Requirement: Directory Scaffolding

`fab-kit sync` SHALL create required directories if they do not exist: `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/`. Each SHALL contain a `.gitkeep` file in `fab/changes/` and `fab/changes/archive/`.

#### Scenario: Fresh repo
- **GIVEN** a repo with `fab/.kit/` but no `fab/changes/` directory
- **WHEN** `fab-kit sync` runs
- **THEN** `fab/changes/`, `fab/changes/archive/`, `docs/memory/`, `docs/specs/` are created with appropriate `.gitkeep` files

### Requirement: Scaffold Tree-Walk

`fab-kit sync` SHALL process all files under `fab/.kit/scaffold/` using the overlay tree convention (file paths relative to `scaffold/` mirror their destinations relative to repo root). Strategy dispatch SHALL be based on filename:

- `fragment-` prefix + `.json` extension → JSON merge (merge `permissions.allow` arrays, dedup)
- `fragment-` prefix + other extension → line-ensure merge (append non-duplicate, non-comment lines)
- No `fragment-` prefix → copy-if-absent (skip if destination exists)

#### Scenario: Fragment JSON merge
- **GIVEN** `fab/.kit/scaffold/.claude/fragment-settings.local.json` contains permissions entries
- **AND** `.claude/settings.local.json` already exists with some permissions
- **WHEN** `fab-kit sync` runs
- **THEN** the permissions.allow arrays are merged (union, no duplicates)

#### Scenario: Copy-if-absent
- **GIVEN** `fab/.kit/scaffold/docs/memory/index.md` exists
- **AND** `docs/memory/index.md` does NOT exist
- **WHEN** `fab-kit sync` runs
- **THEN** the file is copied to `docs/memory/index.md`

#### Scenario: Copy-if-absent skip
- **GIVEN** `docs/memory/index.md` already exists
- **WHEN** `fab-kit sync` runs
- **THEN** the existing file is NOT overwritten

### Requirement: Multi-Agent Skill Deployment

`fab-kit sync` SHALL deploy skill files from `fab/.kit/skills/` to agent-specific directories. Deployment SHALL be conditional — each agent's CLI command is checked via PATH lookup before syncing. The `FAB_AGENTS` environment variable (space-separated list) MAY override PATH detection.

Deployment formats per agent:

| Agent | CLI command | Target path | Method |
|-------|-----------|-------------|--------|
| Claude Code | `claude` | `.claude/skills/{name}/SKILL.md` | Copy |
| OpenCode | `opencode` | `.opencode/commands/{name}.md` | Relative symlink |
| Codex | `codex` | `.agents/skills/{name}/SKILL.md` | Copy |
| Gemini CLI | `gemini` | `.gemini/skills/{name}/SKILL.md` | Copy |

#### Scenario: Claude Code skill deployment
- **GIVEN** `claude` is found on PATH
- **WHEN** `fab-kit sync` runs
- **THEN** each `*.md` file in `fab/.kit/skills/` is copied to `.claude/skills/{stem}/SKILL.md`

#### Scenario: Agent not installed
- **GIVEN** `opencode` is NOT on PATH and `FAB_AGENTS` is not set
- **WHEN** `fab-kit sync` runs
- **THEN** OpenCode skill deployment is skipped with an informational message

#### Scenario: FAB_AGENTS override
- **GIVEN** `FAB_AGENTS=claude` is set
- **WHEN** `fab-kit sync` runs
- **THEN** only Claude Code skills are deployed, regardless of other agents on PATH

### Requirement: Stale Skill Cleanup

`fab-kit sync` SHALL remove skill entries from agent directories that are NOT present in the canonical `fab/.kit/skills/` directory. This ensures skills removed from the kit are cleaned up from all agents.

#### Scenario: Removed skill
- **GIVEN** `.claude/skills/old-skill/SKILL.md` exists but `fab/.kit/skills/old-skill.md` does not
- **WHEN** `fab-kit sync` runs
- **THEN** `.claude/skills/old-skill/` is removed

### Requirement: Version Stamp

`fab-kit sync` SHALL write `fab/.kit-sync-version` with the current `fab/.kit/VERSION` content after successful sync. This enables staleness detection by preflight.

#### Scenario: Stamp written
- **GIVEN** `fab/.kit/VERSION` contains `0.45.0`
- **WHEN** `fab-kit sync` completes successfully
- **THEN** `fab/.kit-sync-version` contains `0.45.0`

### Requirement: Project-Level Sync Scripts

`fab-kit sync` SHALL discover and execute shell scripts in `fab/sync/*.sh` (sorted order) after completing kit-level sync. If `fab/sync/` does not exist, this step SHALL be skipped silently. Script failures SHALL halt the sync pipeline.

#### Scenario: Project sync scripts
- **GIVEN** `fab/sync/custom.sh` exists and is executable
- **WHEN** `fab-kit sync` runs
- **THEN** kit-level sync completes first, then `custom.sh` is executed
- **AND** if `custom.sh` exits non-zero, sync halts with an error

### Requirement: Prerequisites Check

`fab-kit sync` SHALL validate prerequisites before performing sync operations, replicating the behavior of `sync/1-prerequisites.sh` (which delegates to `fab-doctor.sh`). Required tools: `git`, `bash`, `yq` (v4+), `jq`, `gh`, `direnv`.

#### Scenario: Missing prerequisite
- **GIVEN** `yq` is not installed
- **WHEN** `fab-kit sync` runs
- **THEN** sync halts with an actionable error message listing the missing tool

## Shell Script Removal

### Requirement: Remove Absorbed Shell Scripts

The following shell scripts SHALL be removed as part of the clean cut:

- `fab/.kit/scripts/fab-sync.sh` — replaced by `fab-kit sync`
- `fab/.kit/sync/2-sync-workspace.sh` — logic absorbed into `fab-kit sync`
- `fab/.kit/sync/1-prerequisites.sh` — prerequisites check absorbed into `fab-kit sync`
- `fab/.kit/sync/3-direnv.sh` — `direnv allow` absorbed into `fab-kit sync`

Scripts NOT removed (remain as-is):
- `fab/.kit/sync/5-sync-hooks.sh` — delegates to `fab hook sync` (fab-go), separate concern
- `fab/.kit/scripts/fab-doctor.sh` — standalone prerequisite checker, used by `/fab-setup` independently
- All batch scripts, launcher scripts, and lib/ utilities

#### Scenario: Scripts removed
- **GIVEN** the change is applied
- **WHEN** a user inspects `fab/.kit/scripts/` and `fab/.kit/sync/`
- **THEN** `fab-sync.sh`, `1-prerequisites.sh`, `2-sync-workspace.sh`, and `3-direnv.sh` are absent
- **AND** `5-sync-hooks.sh`, `fab-doctor.sh`, `fab-help.sh`, and batch scripts remain

## Build System

### Requirement: Five-Binary Build

The build system SHALL produce 5 binaries: `fab` (router), `fab-kit` (workspace lifecycle), `fab-go` (workflow engine), `wt` (worktree management), `idea` (backlog management).

#### Scenario: Build all targets
- **GIVEN** a developer runs `just build-all`
- **THEN** 20 binaries are produced (5 binaries × 4 platforms: darwin/arm64, darwin/amd64, linux/arm64, linux/amd64)

#### Scenario: Local dev build
- **GIVEN** a developer runs `just build`
- **THEN** all 5 binaries are compiled for the current platform

### Requirement: Updated Packaging

Brew archives SHALL include 4 binaries: `fab`, `fab-kit`, `wt`, `idea`. Kit archives SHALL continue to include `fab-go` + kit content (unchanged).

#### Scenario: Brew archive contents
- **GIVEN** `just package-brew` runs
- **THEN** each per-platform brew archive contains `fab`, `fab-kit`, `wt`, `idea`

### Requirement: Updated Homebrew Formula

The Homebrew formula SHALL install 4 binaries: `fab`, `fab-kit`, `wt`, `idea`. The formula test SHALL verify all 4 respond to `--version`.

#### Scenario: Brew install
- **GIVEN** a user runs `brew install fab-kit`
- **THEN** `fab`, `fab-kit`, `wt`, `idea` are installed to the Homebrew bin directory
- **AND** `fab --version`, `fab-kit --version`, `wt --version`, and `idea --version` all succeed

### Requirement: Updated CI Workflow

`.github/workflows/release.yml` SHALL cross-compile all 5 binaries, produce updated brew archives (4 binaries), and update the Homebrew formula to install 4 binaries.

#### Scenario: Release workflow
- **GIVEN** a `v*` tag is pushed
- **WHEN** CI runs
- **THEN** 5 binaries are cross-compiled per platform, brew archives contain 4 binaries, kit archives contain fab-go + kit content, and the Homebrew tap formula is updated

## Deprecated Requirements

### fab-sync.sh as Orchestrator

**Reason**: `fab-sync.sh` and the `fab/.kit/sync/{1,2,3}-*.sh` scripts are replaced by `fab-kit sync` (Go binary). Clean cut — no transition period.
**Migration**: `fab sync` (via router) or `fab-kit sync` (direct) replaces all invocations of `fab-sync.sh`.

### Two-Binary Shim Architecture

**Reason**: The shim is split into `fab` (router) and `fab-kit` (workspace lifecycle) for testability and separation of concerns.
**Migration**: The system `fab` binary becomes the router; `fab-kit` is a new system binary installed alongside it.

## Design Decisions

1. **Single Go module for fab + fab-kit**: Both binaries live in `src/go/fab-kit/` with two `cmd/` entries sharing `internal/`. This avoids Go workspace complexity and keeps cache/download code importable by both without duplication.
   - *Why*: Both binaries need `EnsureCached()`, `CachedKitDir()`, `Download()`, and `ResolveConfig()`. A shared `internal/` package is the standard Go pattern for this.
   - *Rejected*: Separate Go modules (requires Go workspaces or a published shared module), code duplication (maintenance burden).

2. **Negative-match routing**: The router knows the fab-kit command set (small, stable) and sends everything else to fab-go. This avoids maintaining a fab-go command registry in the router.
   - *Why*: fab-kit commands change rarely; fab-go commands change with every release. Negative match means the router doesn't need updating when fab-go adds subcommands.
   - *Rejected*: Positive match (router would need fab-go's command list), prefix-based routing (e.g., `fab kit sync` — changes user-facing CLI).

3. **Clean cut for sync migration**: Shell scripts are removed immediately, not deprecated with a transition period.
   - *Why*: User explicitly decided clean cut. Both implementations would need to coexist and be tested if phased, adding complexity for no benefit since this is a major version change.
   - *Rejected*: Phased migration (fab-sync.sh delegates to fab-kit sync as intermediate step).

4. **5-sync-hooks.sh retained**: The hooks sync script (`5-sync-hooks.sh`) is kept because it delegates to `fab hook sync` (a fab-go subcommand). It's a separate concern from workspace sync and belongs in the fab-go domain.
   - *Why*: Hook registration is a runtime configuration concern (`.claude/settings.local.json`), not a workspace structure concern. Moving it into `fab-kit sync` would create a dependency on fab-go from fab-kit.
   - *Rejected*: Absorbing hook sync into fab-kit (cross-binary dependency, different concern).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sync belongs in fab-kit, not fab-go | Confirmed from intake #1 — user decided sync is workspace lifecycle | S:95 R:70 A:90 D:90 |
| 2 | Certain | Three binaries: fab (router), fab-kit (lifecycle), fab-go (engine) | Confirmed from intake #2 — user proposed for testability | S:95 R:60 A:85 D:90 |
| 3 | Certain | Negative match routing in fab | Confirmed from intake #3 — user chose option 3 | S:95 R:80 A:90 D:95 |
| 4 | Certain | Each binary owns its own --help | Confirmed from intake #4 — fab composes help from both | S:90 R:85 A:85 D:90 |
| 5 | Certain | Future: no fab/.kit/ in repo, content from cache | Confirmed from intake #5 — marked as non-goal for this change but architecturally prepared for | S:90 R:50 A:80 D:85 |
| 6 | Certain | Both binaries in one Go module at src/go/fab-kit/ | Confirmed from intake #9 — single module, two cmd/ entries | S:95 R:80 A:90 D:95 |
| 7 | Certain | Source directory renamed src/go/shim/ → src/go/fab-kit/ | Confirmed from intake #10 | S:95 R:80 A:90 D:95 |
| 8 | Certain | Clean cut — remove shell scripts, no transition | User explicitly decided clean cut during discussion | S:95 R:60 A:85 D:90 |
| 9 | Confident | Build produces 5 binaries (fab, fab-kit, fab-go, wt, idea) | Confirmed from intake #6 — follows from three-binary decision | S:80 R:70 A:80 D:85 |
| 10 | Confident | Brew installs fab + fab-kit + wt + idea | Confirmed from intake #7 — fab-go stays in per-version cache | S:75 R:75 A:80 D:80 |
| 11 | Confident | Project-level fab/sync/*.sh extensibility preserved | From intake #8 — exec from Go | S:60 R:65 A:75 D:70 |
| 12 | Confident | 5-sync-hooks.sh retained (not absorbed) | Hook sync delegates to fab-go — different concern, would create cross-binary dependency | S:70 R:75 A:80 D:80 |
| 13 | Confident | fab-doctor.sh retained as standalone | Used by /fab-setup independently; only prerequisites check portion absorbed into fab-kit sync | S:65 R:80 A:75 D:75 |
| 14 | Confident | fab-kit sync runs direnv allow | Absorbs 3-direnv.sh behavior — idempotent, single-line operation | S:70 R:90 A:85 D:85 |

14 assumptions (8 certain, 6 confident, 0 tentative, 0 unresolved).
