# Distribution

**Domain**: fab-workflow

## Overview

How `src/kit/` is distributed to new and existing projects. Covers the Homebrew distribution model (three-binary architecture: `fab` router + `fab-kit` workspace lifecycle + standalone utilities), the bootstrap process (getting `.kit/` into a project for the first time — primary method is `brew install fab-kit` + `fab init`), the update mechanism (`fab upgrade` replaces the old `fab-upgrade.sh`), the release workflow (version management via `release.sh`, build recipes via `justfile`, CI orchestration via `.github/workflows/release.yml` — producing per-platform archives with Go binaries), and the repo rename from `docs-sddr` to `fab-kit`.

## Requirements

### Homebrew Distribution

#### Homebrew Formula

A Homebrew formula named `fab-kit` SHALL be published to the `wvrdz/homebrew-tap` tap. The formula SHALL install four binaries to the system PATH: `fab` (router/dispatcher), `fab-kit` (workspace lifecycle), `wt` (worktree management), and `idea` (backlog management). Users add the tap via `brew tap wvrdz/tap`.

**Scenarios**:
- Fresh install (`brew tap wvrdz/tap && brew install fab-kit`) — installs `fab`, `fab-kit`, `wt`, and `idea` to the Homebrew bin directory; all respond to `--version`
- Upgrade via Homebrew (`brew upgrade fab-kit`) — updates the router, fab-kit, `wt`, and `idea` to latest formula version; per-version cache is unaffected

#### Router Architecture (System `fab` Binary)

The system `fab` binary acts as a router using negative-match dispatch. It maintains a static allowlist of fab-kit commands (`init`, `upgrade`, `sync`, `--version`, `-v`, `--help`, `-h`, `help`). Commands matching this list are dispatched to `fab-kit` via `syscall.Exec`. All other commands are dispatched to the version-resolved `fab-go`.

For fab-go dispatch, the router SHALL:

1. Walk up from CWD to find `fab/project/config.yaml`
2. Read `fab_version` from `config.yaml` (e.g., `fab_version: "0.43.0"`)
3. Check the local cache for the matching `fab-go` binary at `~/.fab-kit/versions/{version}/fab-go`
4. If not cached, download the release from GitHub (`wvrdz/fab-kit` releases) and cache it
5. Exec the cached `fab-go` with full argument passthrough

`fab help` composes help from both sub-binaries: workspace commands (from fab-kit) are always shown; workflow commands (from fab-go) are shown only inside a fab-managed repo.

**Scenarios**:
- fab-kit command dispatch — `fab init`, `fab sync`, `fab upgrade` are routed to `fab-kit` with all args passed through
- Normal fab-go dispatch — router reads `fab_version`, resolves cached `fab-go`, execs with all args passed through
- Version not cached — router auto-fetches from GitHub releases, caches binary + `.kit/` content, then dispatches
- No network during auto-fetch — exits non-zero with version and network hint
- `config.yaml` found but `fab_version` absent — exits with: `"No fab_version in config.yaml. Run 'fab init' to set one."`
- Not in a fab-managed repo, fab-kit command — `fab init`, `fab --version`, `fab --help` dispatched to `fab-kit` (works without config.yaml)
- Not in a fab-managed repo, workflow command — exits with: `"Not in a fab-managed repo. Run 'fab init' to set one up."`

#### Cache Layout

The router and fab-kit store versioned artifacts at `~/.fab-kit/versions/{version}/`. Each version directory contains:

- `fab-go` — the Go backend binary for the current platform
- `kit/` — full `.kit/` content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

Multiple versions coexist independently. No automatic cache eviction — users manage cleanup manually.

**Scenarios**:
- Cache structure after auto-fetch — `~/.fab-kit/versions/0.43.0/fab-go` exists and is executable; `kit/VERSION` contains `0.43.0`; `kit/skills/` contains skill files
- Multiple versions coexist — repos using different `fab_version` values dispatch to separate cached binaries

### Bootstrap

#### Primary Method: `brew install fab-kit` + `fab init`

The primary bootstrap path for new projects is:
```
brew tap wvrdz/tap && brew install fab-kit
cd <repo>
fab init
```

`fab init` (a fab-kit subcommand, routed via the `fab` router, not dispatched to fab-go) SHALL:
1. Resolve the latest release version from GitHub
2. Ensure the version is cached (download if not)
3. Copy `~/.fab-kit/versions/{latest}/kit/` to the repo's `src/kit/`
4. Set `fab_version: "{latest}"` in `fab/project/config.yaml` (creating the file if needed)
5. Call `Sync()` directly (the same logic as `fab-kit sync`) to deploy skills and set up the workspace

`fab-kit sync` follows a 6-step pipeline, resolving all kit content from the system cache (`~/.fab-kit/versions/{version}/kit/`) rather than `src/kit/` in the repo: (1) validates prerequisites (`git`, `bash`, `yq` v4+, `direnv`), (2) version guard (ensures `fab_version` <= system `fab-kit` version, auto-runs `fab update` if behind), (3) ensures cache (downloads if needed), (4) workspace scaffolding from cache (directories, scaffold tree-walk with fragment merges and copy-if-absent, skill deployment to detected agents, hook sync, version stamp, legacy cleanup), (5) direnv allow, (6) project-level `fab/sync/*.sh` scripts. Supports `--shim` (steps 1-5 only) and `--project` (step 6 only) flags; mutually exclusive.

**Scenarios**:
- Init in a new repo (no `fab/` directory) — `src/kit/` populated from cache; `config.yaml` created with `fab_version` set to latest; `fab-sync.sh` runs
- Init in a repo with existing `fab/` but no `fab_version` — `fab_version` added to existing `config.yaml`; `src/kit/` updated from cache; existing project files NOT overwritten

#### Legacy One-Liner Bootstrap

The curl-based one-liner bootstrap continues to work for environments where Homebrew is not available:

**With compiled backend** (auto-detects platform via `uname`):
```
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
mkdir -p fab; curl -sL "https://github.com/{repo}/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

Where `{repo}` is `sahil87/fab-kit`. After extraction, the user runs `src/kit/scripts/fab-sync.sh` to complete workspace setup.

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work, given the system `fab` binary is installed (`brew install fab-kit`). The system binary provides version-aware execution; `src/kit/` provides content (skills, templates, configuration).

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### `fab upgrade` (Shim Subcommand)

`fab upgrade [version]` is a fab-kit subcommand (routed via the `fab` router to `fab-kit`, not dispatched to `fab-go`) that replaces the former `src/kit/scripts/fab-upgrade.sh`. It SHALL:

1. Resolve the target version — latest release if no argument, or the explicit version (e.g., `fab upgrade 0.44.0`)
2. Download the release to cache if not already present (binary + `.kit/` content)
3. Copy `~/.fab-kit/versions/{version}/kit/` to the repo's `src/kit/` (atomic swap: extract to temp, verify, then replace)
4. Update `fab_version` in `fab/project/config.yaml` to the new version
5. Call `Sync()` directly (the same logic as `fab-kit sync`) to deploy skills to agent directories
6. Display version change and migration reminder if needed

**Scenarios**:
- Upgrade to latest — downloads new version to cache, replaces `src/kit/`, updates `fab_version` in config, runs sync, displays "Updated: 0.43.0 → 0.44.0"
- Upgrade to specific version (`fab upgrade 0.42.1`) — downloads to cache, replaces `src/kit/` and updates `fab_version`
- Already up to date — displays "Already on the latest version (0.43.0). No update needed.", no files modified
- Migration reminder — when `fab/.kit-migration-version` is behind the new version and a migration exists, output includes a reminder to run `/fab-setup migrations`
- No network access — exits non-zero with error message, existing `.kit/` unchanged

#### Update Preserves Project Files

`fab upgrade` MUST NOT modify any files outside of `src/kit/` and `fab/project/config.yaml` (version bump only). Preserved: `fab/project/constitution.md`, `fab/.kit-migration-version`, `docs/memory/`, `docs/specs/`, `fab/changes/`, `.fab-status.yaml`.

#### Deprecated: `fab-upgrade.sh`

`src/kit/scripts/fab-upgrade.sh` has been removed. Use `fab upgrade` instead.

### Sync Staleness Detection

Preflight compares `$(fab kit-path)/VERSION` against `fab_version` in `fab/project/config.yaml` and emits a non-blocking stderr warning when they differ:

- `⚠ Skills may be out of sync — run fab sync to refresh (engine X, project Y)`

If either value is unreadable or empty, the check is silently skipped. This detects stale local skill deployments when a developer pulls new `src/kit/` source via git but hasn't re-run `fab sync` (since `.claude/`, `.agents/`, `.opencode/` are gitignored and not updated by git pull).

#### Atomic Update

`fab upgrade` SHALL use an atomic update strategy: extract cached content to a temporary directory, verify the extraction succeeded (checks for VERSION file), then replace the existing `src/kit/` via `rm -rf` and `mv`. This prevents corruption if interrupted mid-extraction.

**Scenarios**:
- Interrupted during download — existing `.kit/` unchanged
- Interrupted during extraction to temp dir — existing `.kit/` unchanged, temp dir cleaned up on next run
- Extraction verification fails — aborts without replacing `.kit/`, displays error

#### Skill Deployment Repair After Update

After copying new `.kit/` contents, `fab upgrade` SHALL call `Sync()` directly (the same logic as `fab-kit sync`) to ensure all skill deployments are up to date: copies refreshed (`.claude/skills/`, `.agents/skills/`), symlinks valid (`.opencode/commands/`), and stale agent files cleaned up (`.claude/agents/`).

### Release

Release is split across three components: `release.sh` handles version management and git operations, a `justfile` at repo root provides locally-replicable build recipes, and `.github/workflows/release.yml` orchestrates CI. The key principle: CI uses the exact same `just` commands a developer runs locally — no CI-only build logic.

#### Release Script (`release.sh`)

`scripts/release.sh` handles version bumping, migration validation, and git commit/tag/push. It does NOT cross-compile, package archives, or create GitHub Releases — those responsibilities moved to the justfile and CI workflow.

The script accepts a bump type argument (`patch`, `minor`, or `major`) that is required to perform a release. When invoked with no arguments, the script displays usage and exits successfully. Unknown arguments produce an error.

The script pushes to the current branch (via `git branch --show-current`) rather than hardcoded `main`. On `main`, behavior is identical to before. On a release branch (e.g., `release/0.25`), commits and tags are pushed to that branch. The tag push triggers CI to handle cross-compilation, packaging, and GitHub Release creation.

After bumping VERSION, the script validates the migration chain: warns if no migration file targets the new version (reminder for release authors), and warns if overlapping migration ranges are detected. These are warnings only — they do not block the release.

Pre-flight checks: clean working tree (error if dirty), `$(fab kit-path)/VERSION` exists (error if missing). The script does NOT check for `gh` CLI or Go toolchain — those are no longer needed locally for releasing.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.34.0" → "0.34.1"), commits VERSION bump with message `release: v0.34.1`, creates tag `v0.34.1`, pushes commit and tag to current branch; CI takes over from the tag push
- Minor release (`release.sh minor`) — bumps minor version (e.g., "0.34.1" → "0.35.0")
- Major release (`release.sh major`) — bumps major version (e.g., "0.35.0" → "1.0.0")
- Backport release — on branch `release/0.34`, `release.sh patch` bumps 0.34.1→0.34.2, pushes to `release/0.34`, tags `v0.34.2`; CI creates the release, and GitHub's semver ordering ensures the backport is not marked as "latest"
- Backport workflow — `git checkout -b release/0.34 v0.34.1`, cherry-pick fixes, `release.sh patch` bumps and pushes to `release/0.34`, CI handles the rest
- Invalid bump argument — exits with error message listing valid options
- Unknown argument — exits with error listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Build Recipes (`justfile`)

The `justfile` at repo root provides locally-replicable build recipes using [just](https://github.com/casey/just). These same recipes are invoked by CI.

**Development recipes**:
- **`build`** — compiles all five binaries (`fab` router, `fab-kit`, `fab-go`, `idea`, `wt`) for the current platform using `CGO_ENABLED=0`
- **`test`** — runs all unit tests across all Go modules
- **`test-v`** — runs all unit tests (verbose)
- **`doctor`** — checks prerequisites and environment health

**Release recipes** (all output goes to `dist/`):
- **`release [bump]`** — bumps VERSION (default: patch), commits, tags, and pushes; CI handles the rest
- **`dist-kit`** — assembles `dist/kit/` from `src/kit/` (single copy, reused by packaging)
- **`build-target os arch`** — cross-compiles all five binaries for a specific platform into `dist/bin/{name}-{os}-{arch}`
- **`build-all`** — cross-compiles for all 4 release targets (`darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`), producing 20 binaries total (5 per platform)
- **`package-kit`** — creates 4 per-platform `dist/kit-{os}-{arch}.tar.gz` (kit content + `fab-go` only). Archives are rooted at `.kit/`.
- **`package-brew`** — creates 4 per-platform `dist/brew-{os}-{arch}.tar.gz` (`fab`, `fab-kit`, `wt`, `idea`)
- **`release-notes [tag]`** — generates `dist/release-notes.md` with commit-level changelog
- **`brew-formula [tag]`** — generates `dist/fab-kit.rb` from template with SHA256 hashes
- **`dist`** — full pipeline: `dist-kit` + `build-all` + `package-kit` + `package-brew`
- **`clean`** — removes `dist/`

**Five Go binaries**:

| Binary | Source | Distribution |
|--------|--------|-------------|
| `fab` (router) | `src/go/fab-kit/cmd/fab/` | Homebrew formula |
| `fab-kit` | `src/go/fab-kit/cmd/fab-kit/` | Homebrew formula |
| `fab-go` | `src/go/fab/` | Per-version cache via GitHub releases |
| `wt` | `src/go/wt/` | Homebrew formula |
| `idea` | `src/go/idea/` | Homebrew formula |

**Scenarios**:
- Local dev build (`just build`) — compiles all five binaries for current platform
- Cross-compile for a single target (`just build-target darwin arm64`) — produces 5 binaries in `dist/bin/`
- Build all targets (`just build-all`) — produces 20 binaries in `dist/bin/` (5 per platform x 4 platforms)
- Full pipeline (`just dist`) — assembles kit, builds all, packages all into `dist/`
- Package without prior build (`just package-kit`) — fails with error directing to run prerequisite steps first
- Clean up (`just clean`) — removes `dist/`

#### CI Workflow (`.github/workflows/release.yml`)

`.github/workflows/release.yml` is a GitHub Actions workflow triggered on push of tags matching `v*`. It runs on a single `ubuntu-latest` runner and uses the same `just` recipes as local development.

Workflow steps:
1. Checkout repository (`actions/checkout@v4`)
2. Set up Go toolchain (`actions/setup-go@v5`, Go 1.22)
3. Install `just` command runner (`extractions/setup-just@v2`)
4. Run `just build-all` (cross-compiles all 20 targets: 5 binaries x 4 platforms)
5. Run `just package-kit` (creates 5 archives with `fab-go` per platform — router, fab-kit, wt, idea are Homebrew-distributed)
6. Create GitHub Release via `gh release create` with all 5 archives and commit-level changelog (minor releases cumulate all commits since the previous minor; patch releases show commits since the previous release)

The workflow sets `permissions: contents: write` for release creation. `GITHUB_TOKEN` is used implicitly by `gh`.

GitHub determines "latest" release status based on semver ordering — backport releases for older version series (e.g., `v0.34.2` when `v0.35.0` exists) are not marked as latest automatically. For edge cases, use `gh release edit $TAG --latest=false` after CI creates the release.

**Scenarios**:
- Tag push triggers workflow — push of `v0.35.0` tag triggers the release workflow
- Non-tag push does not trigger — regular commits pushed without a `v*` tag do not run the workflow
- Full CI release — tag `v0.35.0` triggers workflow, which cross-compiles all Go binaries (20 total: fab router, fab-kit, fab-go, idea, wt x 4 platforms), packages 5 archives (`fab-go` per platform), and creates a GitHub Release with commit-level changelog
- Backport release via CI — tag `v0.34.2` triggers workflow; GitHub's semver ordering ensures it is not marked as "latest" since `v0.35.0` exists

#### Release Archive Contents

Each release produces per-platform archives structured for the router/fab-kit to download and cache. Per-platform archives (`kit-{os}-{arch}.tar.gz`) contain:
- `.kit/bin/fab-go` — the versioned Go backend binary
- `.kit/` — all content (skills, templates, scripts, hooks, migrations, scaffold, VERSION)

The router (or fab-kit) extracts `fab-go` to `~/.fab-kit/versions/{version}/fab-go` and the rest to `~/.fab-kit/versions/{version}/kit/`.

Per-platform archives:
- **`kit-darwin-arm64.tar.gz`** — Content + `fab-go` compiled for macOS Apple Silicon.
- **`kit-darwin-amd64.tar.gz`** — Content + `fab-go` compiled for macOS Intel.
- **`kit-linux-arm64.tar.gz`** — Content + `fab-go` compiled for Linux ARM64 (musl, fully static).
- **`kit-linux-amd64.tar.gz`** — Content + `fab-go` compiled for Linux x86-64 (musl, fully static).
- **`kit.tar.gz`** — Generic archive containing content only (no binary). Serves as a fallback for unsupported platforms.

No project-specific files (config.yaml, constitution.md, memory/, specs/, changes/) are included in any archive. Package production code (idea only) is included under `.kit/packages/`, hook scripts under `.kit/hooks/` — all delivered to downstream projects on upgrade. `src/kit/sync/` contains only `.gitkeep` (all sync scripts absorbed into `fab-kit` Go binary). `idea` is a standalone system binary (installed via Homebrew, not per-repo); the shell package at `.kit/packages/idea/bin/idea` is retained for rollback safety and generic-archive users. Skill files are included in all archives and deployed to agents by `fab-kit sync`. `fab-go binary at ` contains only `.gitkeep` — no binaries are shipped in the repo.

**Binary distribution split**: The router (`fab`), `fab-kit`, `wt`, and `idea` are Homebrew-only (version-independent, system-level). Only `fab-go` is version-coupled and lives in the per-version cache.

### Deprecated: Backend Override Mechanism

The `FAB_BACKEND` env var and `.fab-backend` file mechanism has been removed. The Go backend is the only backend. The system shim dispatches to `fab-go` directly — no override needed. References to `FAB_BACKEND` and `.fab-backend` should be removed from scripts and documentation.

### Repo Rename

The repository SHALL be renamed from `docs-sddr` to `fab-kit` to reflect its role as the canonical source for `src/kit/`. GitHub auto-redirects handle existing URLs and clones.

**Scenarios**:
- Old URLs (`github.com/sahil87/docs-sddr`) redirect to the current repo URL
- Existing clones with old remote URL continue to work via redirect

## Design Decisions

- **CI/local parity via justfile (260307-ma7o-1)**: Build recipes live in the `justfile` so CI and local development use identical commands (`just build-all`, `just package-kit`). No CI-only build scripts or logic. This makes CI behavior fully reproducible locally.
- **Three-way release split (260307-ma7o-1)**: `release.sh` owns version/tag/push, `justfile` owns build/package, `.github/workflows/release.yml` owns orchestration. Each component has a single responsibility and can be tested independently.
- **GitHub semver ordering replaces `--no-latest` (260307-ma7o-1)**: GitHub automatically determines "latest" release based on semver. Backport releases (e.g., `v0.34.2` when `v0.35.0` exists) are not marked latest. The `--no-latest` flag was removed from `release.sh` — no flag to remember, no CI mechanism to pass it through. For edge cases, `gh release edit` can be used post-creation.
- **Commit-level release notes with minor cumulation**: CI generates release notes from `git log --oneline` with linked commit SHAs. Minor releases (x.y.0) cumulate all commits since the previous minor tag, giving a complete picture of the release cycle. Patch releases show commits since the previous release only. Major releases use the same patch-style diff (manual curation expected for milestone releases).
- **Homebrew distribution with three-binary architecture (260401-46hw, 260402-3ac3)**: The system `fab` binary is a router installed via `brew install fab-kit`. It dispatches workspace commands to `fab-kit` and workflow commands to the version-resolved `fab-go`. `fab-kit` owns workspace lifecycle (init, upgrade, sync). This decouples binary distribution from the repo — `src/kit/` holds content only, the binaries manage execution. Rejected: binary-in-repo (redundant when router manages versions), `fab self-update` (don't reinvent the package manager), two-binary shim model (untestable, blurred concerns).
- **`fab upgrade` as fab-kit subcommand (260401-46hw, 260402-3ac3)**: `fab-kit` handles upgrade directly, replacing `fab-upgrade.sh`. `fab-kit` already has download/cache logic — upgrade is a natural extension. Rejected: keeping `fab-upgrade.sh` alongside `fab-kit` (duplication of download logic).
- **Cache stores binary + content (260401-46hw)**: Each cached version includes both `fab-go` and the full `.kit/` content. `fab upgrade` needs the content to populate the repo's `src/kit/`. Rejected: binary-only cache (would need separate download for content).
- **Formula name `fab-kit`, binary name `fab` (260401-46hw)**: Homebrew formula uses `fab-kit` to avoid collision with Python Fabric's `fab` formula, while the installed binary is `fab`. Rejected: `fab` as formula name (collides with Fabric).
- **~~Backend override via env var + file (260307-bmp3-3)~~**: *Deprecated* — removed with the shim model. Go is the only backend; the shim dispatches to `fab-go` directly.

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260402-gnx5-relocate-kit-to-system-cache | 2026-04-02 | Kit content no longer copied to user projects — served entirely from system cache at `~/.fab-kit/versions/<version>/kit/`. `fab init` and `fab upgrade` no longer create `fab/.kit/` in projects. Source repo layout: `fab/.kit/` renamed to `src/kit/`. Build scripts (`justfile`, `release.sh`) updated to read from `src/kit/`. `.gitignore` cleaned of `fab/.kit/` entries. `fab kit-path` command added for agent-agnostic kit path resolution. Bootstrap one-liner and manual copy references updated. |
| 260402-ktbg-sync-from-cache | 2026-04-02 | Rewrote `fab-kit sync` to resolve kit content from system cache (`~/.fab-kit/versions/{version}/kit/`) instead of `src/kit/` in the repo. 6-step pipeline: prerequisites, version guard, ensure cache, scaffolding, direnv, project scripts. Added `--shim` (steps 1-5) and `--project` (step 6) flags. Absorbed hook sync into step 4 (replicated hooklib in `fab-kit` internal package). Removed `5-sync-hooks.sh`. Fixed `fragment-.envrc` (`fab-kit sync` -> `fab sync`). Updated prerequisites (removed jq, gh — no longer needed by sync). Updated release archive description (sync/ now empty). |
| 260326-p4ki-allow-idea-shorthand | 2026-03-26 | Restored bare `idea "text"` shorthand (equivalent to `idea add "text"`). Added `RunE` with `cobra.ArbitraryArgs` to root command in `src/go/idea/cmd/main.go`. Multiple args joined with space. Empty text returns error. Persistent flags (`--main`, `--file`) work with shorthand. Updated `_cli-external.md` and `docs/specs/packages.md`. |
| 260320-9tqo-fix-idea-docs-main-flag | 2026-03-20 | Corrected `idea` documentation: moved Backlog section from `_cli-fab.md` to `_cli-external.md`, fixed invocation from `fab idea` to standalone `fab-go binary at idea`. Added `--main` persistent flag — default now uses current worktree (`--show-toplevel`), `--main` opts into main worktree (`--git-common-dir`). Renamed `GitRepoRoot()` to `MainRepoRoot()`, added `WorktreeRoot()`. Updated `_cli-external.md` frontmatter and `docs/specs/packages.md`. |
| 260312-96nf-remove-rust-implementation | 2026-03-12 | Removed all Rust references from distribution docs. Removed Rust recipes from build recipes section, Rust CI steps (toolchain, Zig, cargo-zigbuild), Rust from archive descriptions (3→2 binaries per platform, 12→8 total). Updated backend override to Go-only. Removed "Transition Period: Dual Backends" section. Updated bootstrap descriptions, packaging scenarios, and CI workflow steps. Removed cargo-zigbuild design decision. |
| 260310-8m3k-port-wt-tests-cleanup-legacy | 2026-03-10 | Removed `src/packages/` (legacy shell wt package and bats tests), `src/tests/` (bats submodule libs), and `.gitmodules` (bats submodule refs only). Ported 73 behavioral tests from bats to Go in `src/go/wt/cmd/*_test.go`. Removed `bats` from prerequisites description (already absent from actual sync scripts). Removed `test-setup` and `test-packages` justfile targets and their backing scripts (`scripts/just/test-setup.sh`, `test-packages.sh`). |
| 260310-qbiq-go-wt-binary | 2026-03-10 | Per-platform archives now include `wt` binary at `.kit/bin/wt` alongside `fab-go` and `fab-rust` (3 binaries per platform, 12 total cross-compiled). Added justfile recipes: `build-wt`, `build-wt-target`, `build-wt-all`. Updated `build-all` to include wt. Updated `package-kit` to verify and include wt binary. `src/kit/packages/wt/` removed — wt is a binary, not a shell package. `env-packages.sh` already adds `$KIT_DIR/bin` to PATH — no change needed for wt binary availability. |
| 260310-pl72-port-idea-to-go | 2026-03-10 | `idea` is now available as `fab idea` via the Go binary (in per-platform archives), in addition to the shell package at `.kit/packages/idea/bin/idea`. Both coexist — shell package retained for rollback safety and generic-archive users. |
| 260307-buf0-4-rust-ci-build | 2026-03-10 | Releases now ship both Go and Rust binaries. Added Rust cross-compilation recipes to justfile (`build-rust-target`, `build-rust-all`, `build-all`, `_rust-target`). Updated `package-kit` to include both `fab-go` and `fab-rust` in per-platform archives and exclude both from generic archive. CI workflow updated with Rust toolchain (`dtolnay/rust-toolchain`), Zig (`pip install ziglang`), `cargo-zigbuild`, and cached tool installations. `build-go-all` → `build-all` in CI. Linux Rust targets use musl for fully static binaries. |
| 260307-bmp3-3-rust-binary-port | 2026-03-10 | Added backend override mechanism (`FAB_BACKEND` env var, `.fab-backend` file) to dispatcher for switching between Rust and Go backends. Documented transition period where both binaries coexist — Rust preferred by default, Go shipped in release archives, Rust built locally via `just build-rust`. CI/release for Rust deferred. Updated bootstrap one-liners to reference both backends. |
| 260307-ma7o-1-ci-releases-justfile | 2026-03-09 | Split release workflow into three components: `release.sh` simplified to version bump + git commit/tag/push only (~60 lines, removed ~200 lines of build/package/release logic). New `justfile` at repo root provides build recipes (`build-go`, `build-go-target`, `build-go-all`, `package-kit`, `clean`) replicable locally and in CI. New `.github/workflows/release.yml` triggered on `v*` tag push — uses `just` recipes on single `ubuntu-latest` runner, creates GitHub Release with auto-generated notes. Removed `--no-latest` flag (GitHub's semver ordering handles backport "latest" status). Removed Go toolchain and `gh` CLI checks from release script. |
| 260306-qkov-operator1-skill | 2026-03-07 | Noted that `fab-operator1.md` ships as part of the kit skills directory in all archives — no new distribution mechanics, just another skill file deployed by `fab-sync.sh`. |
| 260305-u8t9-clean-break-go-only | 2026-03-05 | Updated generic archive (shell-only) scenario: no longer provides a working `fab` command — Go binary is required. Shell script fallback removed from dispatcher. |
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added `$(fab kit-path)/hooks/` as a new distributed directory (hook scripts shipped with kit). Updated bootstrap description to mention hook registration via `5-sync-hooks.sh`. Updated release archive contents to note hooks and sync scripts alongside packages. |
| 260305-g0uq-2-ship-fab-go-binary | 2026-03-05 | Ship fab Go binary: release now produces 5 archives (generic `kit.tar.gz` + 4 per-platform `kit-{os}-{arch}.tar.gz` with Go binary at `.kit/bin/fab`). `release.sh` cross-compiles via `CGO_ENABLED=0` for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64. `fab-upgrade.sh` detects platform via `uname -s`/`uname -m`, downloads platform archive with fallback to generic. README bootstrap one-liner is now platform-aware. Shell scripts in `lib/` have shim layer that delegates to Go binary when present. Skills updated via `_cli-fab.md` (renamed from `_scripts.md`) to invoke `fab-go binary at fab` as primary calling convention. |
| 260305-bhd6-1-build-fab-go-binary | 2026-03-05 | Go binary (`src/go/fab/`) built — ports all lib/ scripts to single `fab` binary. No distribution changes in this change — binary inclusion in kit.tar.gz and per-platform archives are deferred to a future change. Go toolchain required only for building from source, not for end users. |
| 260303-l6nk-gemini-cli-agent-aware-sync | 2026-03-04 | Added Gemini CLI as 4th supported agent. Updated bootstrap/sync descriptions to reflect conditional agent deployment (skills deployed only when agent's CLI found in PATH). Four agents: Claude Code (copies), OpenCode (symlinks), Codex (copies), Gemini CLI (copies). |
| 260301-08pa-version-pinned-upgrade-and-release | 2026-03-02 | Added version-pinned upgrade (`fab-upgrade.sh v0.24.0`) with tag-aware messaging. Added backport release support to `release.sh`: push to current branch instead of hardcoded `main`, `--no-latest` flag for `gh release create --latest=false`, position-independent argument parsing. |
| 260402-0ak9-remove-sync-version-file | 2026-04-02 | Removed `fab/.kit-sync-version` from preserved files list. Sync staleness detection now compares `$(fab kit-path)/VERSION` against `fab_version` in `config.yaml` (single warning message). |
| 260226-koj1-version-staleness-warning | 2026-02-26 | Added sync staleness detection (preflight stderr warning). Renamed `fab/project/VERSION` → `fab/.kit-migration-version`. Updated preserved files list in upgrade section. |
| 260224-v40o-wt-drop-prefix-and-dotworktrees | 2026-02-25 | wt package: dropped `wt/` branch prefix from exploratory worktrees (branch = worktree name directly). Switched worktree home directory from `<repo>-worktrees` to `<repo>.worktrees` (GitLens convention). Updated `wt-create` help text. No migration for existing worktrees. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | `env-packages.sh` moved from `scripts/` to `scripts/lib/` — now sourced from `src/kit/scripts/lib/env-packages.sh` in both `scaffold/fragment-.envrc` and `src/packages/rc-init.sh` |
| 260219-d2y2-copy-template-skills-drop-agents | 2026-02-19 | Updated references from symlinks to copies for Claude Code skills. Renamed "Symlink Repair After Update" to "Skill Deployment Repair After Update". Updated bootstrap and upgrade descriptions to reflect copy-with-template deployment |
| 260218-cif4-eliminate-symlinks-distribute-packages | 2026-02-18 | Package production code (idea, wt) now distributed via `kit.tar.gz` under `.kit/packages/`. Updated release archive contents description. Updated `fab-upgrade.sh` description (symlinks → directories and agents). Added `env-packages.sh` for centralized PATH setup, sourced by `scaffold/envrc` (direnv) and `src/packages/rc-init.sh` (shell rc). |
| 260217-zkah-readme-quickstart-prereqs-check | 2026-02-18 | Added prerequisites validation to `fab-sync.sh` pipeline (via `sync/1-prerequisites.sh`). Updated bootstrap description to mention prerequisites check. Restructured README Quick Start: folded Initialize and Updating under Install as sub-sections. |
| 260216-ymvx-DEV-1043-envrc-line-sync | 2026-02-16 | Updated `.envrc` references from symlink to line-ensuring: bootstrap description now says "`.envrc` entries (from `scaffold/envrc`, line-ensuring)"; scenario updated to note line-ensuring from scaffold |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | `lib/sync-workspace.sh` → `fab-sync.sh` (promoted to `scripts/`); `/fab-init` → `/fab-setup`; `/fab-update` → `/fab-setup migrations` |
| 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests | 2026-02-16 | Renamed `init-scaffold.sh` → `sync-workspace.sh` throughout (bootstrap description, update script references, symlink repair) |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added version drift scenarios to update section; added `fab/VERSION` to preserved files list; added migration chain validation to release section |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated bootstrap description to mention `fab-sync.sh` reads from `scaffold/` files for index templates, envrc, and gitignore entries |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` throughout; moved `release.sh` from `src/kit/scripts/` to `scripts/` (dev-only, not shipped in kit) |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed script references: `fab-setup.sh` → `_init_scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Updated bootstrap description to include `docs/specs/` directory and `design/index.md` in `fab-sync.sh` output |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Initial creation — bootstrap, update, release, and repo rename requirements |
| 260401-46hw-brew-install-system-shim | 2026-04-02 | Homebrew distribution model: system `fab` shim installed via `brew install fab-kit` (formula at `wvrdz/homebrew-tap`). Shim reads `fab_version` from `config.yaml`, resolves cached `fab-go` at `~/.fab-kit/versions/`, auto-fetches on miss. `fab init` bootstraps new repos (primary method replaces curl one-liner). `fab upgrade` replaces `fab-upgrade.sh` (shim subcommand). `wt` and `idea` become system-only Homebrew binaries. `fab-go binary at ` emptied (binary-free repo). Backend override mechanism (`FAB_BACKEND`, `.fab-backend`) removed. Sync pipeline: `4-get-fab-binary.sh` removed, `5-sync-hooks.sh` calls `fab hook sync` (system shim), `.envrc` scaffold removes `PATH_add src/kit/bin`. Release archives restructured for shim cache extraction (`fab-go` + `kit/` content). 4 Go binaries: `fab` (shim, Homebrew), `fab-go` (per-version cache), `wt` (Homebrew), `idea` (Homebrew). |
| 260401-ixzv-org-migrate-mit-license | 2026-04-02 | Migrated GitHub org references from wvrdz to sahil87. License changed from PolyForm Internal Use to MIT (root LICENSE). |
| 260402-3ac3-three-binary-architecture | 2026-04-02 | Three-binary architecture: Homebrew formula installs 4 binaries (`fab`, `fab-kit`, `wt`, `idea`). Shim section renamed to "Router Architecture" — `fab` uses negative-match dispatch to `fab-kit` or `fab-go`. Build produces 5 binaries (20 cross-compiled). Binary table updated: `fab` (router) from `src/go/fab-kit/cmd/fab/`, `fab-kit` from `src/go/fab-kit/cmd/fab-kit/`. `fab-sync.sh` references replaced with `fab-kit sync` / `fab sync`. `init` and `upgrade` call `Sync()` directly instead of `fab-sync.sh`. Updated design decisions (Homebrew distribution, fab upgrade). |
