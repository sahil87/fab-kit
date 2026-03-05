# Distribution

**Domain**: fab-workflow

## Overview

How `fab/.kit/` is distributed to new and existing projects. Covers the bootstrap process (getting `.kit/` into a project for the first time), the update mechanism (pulling new versions into an existing project), the release workflow (packaging and publishing new versions — including per-platform archives with Go binary), and the repo rename from `docs-sddr` to `fab-kit`.

## Requirements

### Bootstrap

#### One-Liner Bootstrap

New projects SHALL be bootstrappable via a single curl command that downloads a kit archive from GitHub Releases and extracts it into `fab/.kit/`.

**With Go binary** (recommended — auto-detects platform via `uname`):
```
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac
mkdir -p fab; curl -sL "https://github.com/{repo}/releases/latest/download/kit-${os}-${arch}.tar.gz" | tar xz -C fab/
```

**Generic** (shell scripts only, no binary):
```
mkdir -p fab
gh release download --repo {repo} --pattern 'kit.tar.gz' --output - | tar xz -C fab/
```

Where `{repo}` is the `repo` value from `fab/.kit/kit.conf` (e.g. `wvrdz/fab-kit`). The platform-aware one-liner detects OS and architecture at runtime (`uname -s`, `uname -m`) and downloads the matching `kit-{os}-{arch}.tar.gz` archive which includes the pre-compiled Go binary at `fab/.kit/bin/fab`.

After extraction, the user MUST run `fab/.kit/scripts/fab-sync.sh` to validate prerequisites (`yq`, `jq`, `gh`, `direnv`, `bats`), create directories (`changes/`, `memory/`, `specs/`), skeleton files (copied from `scaffold/memory-index.md` and `scaffold/specs-index.md`), deploy skills conditionally to detected agents (copies for Claude Code, Codex, and Gemini CLI; symlinks for OpenCode — only when the agent's CLI is found in PATH), `.envrc` entries (from `scaffold/envrc`, line-ensuring), `.gitignore` entries (from `scaffold/gitignore-entries`), and register Claude Code hooks from `fab/.kit/hooks/` into `.claude/settings.local.json`. The bootstrap only provides `.kit/` — no `config.yaml`, `constitution.md`, or other project files.

**Scenarios**:
- Bootstrap with Go binary (platform-aware one-liner) — downloads `kit-{os}-{arch}.tar.gz`, creates `fab/.kit/` with all skills, templates, scripts, VERSION file, and Go binary at `fab/.kit/bin/fab`; running `fab-sync.sh` first validates prerequisites (yq, jq, gh, direnv, bats), then creates `changes/`, `memory/index.md`, `specs/index.md`, skill deployments conditionally per detected agent (Claude Code, OpenCode, Codex, Gemini CLI — only when the agent's CLI command is found in PATH), `.envrc` (line-ensuring from scaffold), and `.gitignore` entry
- Bootstrap without Go binary (generic one-liner) — downloads `kit.tar.gz`, creates `fab/.kit/` with shell scripts only (no `fab/.kit/bin/fab`); shell scripts function identically, the Go binary shims fall through to bash implementations
- Bootstrap a new project (no `fab/` directory) — creates `fab/.kit/` with all skills, templates, scripts, and VERSION file; same sync behavior as above
- Bootstrap with existing `fab/` directory — creates or replaces `fab/.kit/`; existing files outside `.kit/` (config.yaml, constitution.md, memory/, specs/, changes/) are NOT affected

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work. The bootstrap one-liner is an additive convenience, not a replacement.

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### Update Script (`fab-upgrade.sh`)

`fab/.kit/scripts/fab-upgrade.sh` SHALL detect the current platform via `uname -s` (OS) and `uname -m` (architecture, with `x86_64`→`amd64` and `aarch64`→`arm64` normalization), attempt to download the platform-specific `kit-{os}-{arch}.tar.gz` from GitHub Releases, fall back to the generic `kit.tar.gz` if the platform archive is not available, extract it to replace the current `fab/.kit/` contents, display the version change, report whether the Go binary is included, and re-run `fab-sync.sh` to repair directories and skill deployments.

The script accepts an optional positional argument — a release tag (e.g., `v0.24.0`) — to download a specific version instead of latest. The tag is passed as-is to `gh release download "$tag"`, with no normalization or `v`-prefix stripping.

**Scenarios**:
- Update to a newer version (platform archive available) — downloads `kit-{os}-{arch}.tar.gz`, replaces `.kit/` contents including `bin/fab` Go binary, displays version change (e.g., "0.1.0 → 0.2.0"), reports "Go binary: included ({os}/{arch})", re-runs `fab-sync.sh`, checks for version drift and prints `/fab-setup migrations` reminder if needed, preserves all files outside `.kit/`
- Update with platform archive unavailable — falls back to generic `kit.tar.gz` with message "Platform {os}/{arch} not available, using generic archive", reports "Go binary: not included (shell scripts only)"
- Update to a specific version (`fab-upgrade.sh v0.24.0`) — downloads and installs the exact tagged release; no version comparison or downgrade warning
- Already up to date — informs user ("Already on the latest version"), no files modified
- Already on the requested tag — informs user ("Already on v0.24.0 (0.24.0)"), no files modified
- Tag not found — exits non-zero with error including the tag and a hint: `Check that the tag exists: gh release view $tag --repo $repo`
- No network access — exits non-zero with error message, existing `.kit/` unchanged
- `fab/.kit-migration-version` missing after upgrade — prints guidance to run `/fab-setup` then `/fab-setup migrations`
- `fab/.kit-migration-version` behind new engine version — prints reminder to run `/fab-setup migrations` to apply migrations

#### Update Preserves Project Files

`fab-upgrade.sh` MUST NOT modify any files outside of `fab/.kit/`. Preserved: `fab/project/config.yaml`, `fab/project/constitution.md`, `fab/.kit-migration-version`, `fab/.kit-sync-version`, `docs/memory/`, `docs/specs/`, `fab/changes/`, `fab/current`.

### Sync Staleness Detection

`fab-sync.sh` writes `fab/.kit-sync-version` after skill deployment — a gitignored stamp file containing the `fab/.kit/VERSION` value at sync time. `lib/preflight.sh` compares this stamp against the current kit VERSION and emits a non-blocking stderr warning when they differ:

- `⚠ Skills out of sync — run fab-sync.sh to refresh (engine X, last synced Y)` — when stamp is behind
- `⚠ Skills may be out of sync — run fab-sync.sh to refresh` — when stamp is missing

This detects stale local skill deployments when a developer pulls new `fab/.kit/` source via git but hasn't re-run `fab-sync.sh` (since `.claude/`, `.agents/`, `.opencode/` are gitignored and not updated by git pull).

#### gh CLI as Primary Download Tool

`fab-upgrade.sh` SHALL use `gh release download` as the primary method to download the release asset. If `gh` is not installed, the script exits with an error directing the user to install it. Curl fallback is deferred to a future enhancement.

#### Atomic Update

`fab-upgrade.sh` SHALL use an atomic update strategy: extract `kit.tar.gz` to a temporary directory, verify the extraction succeeded (checks for VERSION file), then replace the existing `fab/.kit/` via `rm -rf` and `mv`. This prevents corruption if interrupted mid-extraction.

**Scenarios**:
- Interrupted during download — existing `.kit/` unchanged
- Interrupted during extraction to temp dir — existing `.kit/` unchanged, temp dir cleaned up on next run
- Extraction verification fails — aborts without replacing `.kit/`, displays error

#### Skill Deployment Repair After Update

After extracting the new `.kit/` contents, `fab-upgrade.sh` SHALL re-run `fab-sync.sh` to ensure all skill deployments are up to date: copies refreshed (`.claude/skills/`, `.agents/skills/`), symlinks valid (`.opencode/commands/`), and stale agent files cleaned up (`.claude/agents/`).

### Release

#### Release Script (`fab-release.sh`)

`src/scripts/fab-release.sh` SHALL cross-compile the Go binary for 4 platform targets (`darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`) using `CGO_ENABLED=0 GOOS={os} GOARCH={arch} go build`, package `fab/.kit/` into 5 archives (1 generic `kit.tar.gz` without binary + 4 per-platform `kit-{os}-{arch}.tar.gz` with the compiled binary at `.kit/bin/fab`), bump the VERSION file, commit the version change, and create a GitHub Release with all 5 archives as attached assets.

The script requires the Go toolchain in addition to `gh` CLI. Cross-compilation uses `CGO_ENABLED=0` for static binaries with no system library dependencies. Build artifacts are staged in a `.release-build/` temporary directory, cleaned up after release.

The script accepts a bump type argument (`patch`, `minor`, or `major`) that is required to perform a release, and an optional `--no-latest` flag. When invoked with no arguments, the script displays usage and exits successfully; when flags are provided without a bump type, it produces an error. Arguments are position-independent — `fab-release.sh patch --no-latest` and `fab-release.sh --no-latest patch` are equivalent. Unknown flags produce an error.

The script pushes to the current branch (via `git branch --show-current`) rather than hardcoded `main`. On `main`, behavior is identical to before. On a release branch (e.g., `release/0.25`), commits and tags are pushed to that branch.

When `--no-latest` is passed, `gh release create` is invoked with `--latest=false`, preventing the release from becoming the "latest" release on GitHub. This enables backport releases for older version series without breaking `fab-upgrade.sh` (no args) for users on the current release line.

After bumping VERSION, the script validates the migration chain: warns if no migration file targets the new version (reminder for release authors), and warns if overlapping migration ranges are detected. These are warnings only — they do not block the release.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.1.0" → "0.1.1"), cross-compiles Go binary for 4 platforms, creates 5 archives (kit.tar.gz + kit-{os}-{arch}.tar.gz), commits VERSION bump, pushes to current branch, creates GitHub Release marked as latest with all 5 archives
- Minor release (`fab-release.sh minor`) — bumps minor version (e.g., "0.1.1" → "0.2.0")
- Major release (`fab-release.sh major`) — bumps major version (e.g., "0.2.0" → "1.0.0")
- Backport release (`fab-release.sh patch --no-latest`) — bumps version, pushes to current branch (not main), creates GitHub Release with `--latest=false`, prints "Note: This release was NOT marked as 'latest'."
- Backport workflow — `git checkout -b release/0.25 v0.25.1`, cherry-pick fixes, `fab-release.sh patch --no-latest` bumps 0.25.1→0.25.2, pushes to `release/0.25`, tags `v0.25.2`
- Go toolchain missing — exits with error directing user to install from https://go.dev/
- Missing bump type with flags (`fab-release.sh --no-latest`) — displays usage and exits non-zero
- Invalid bump argument — exits with error message listing valid options
- Unknown flag (`fab-release.sh patch --unknown`) — exits with error listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Release Archive Contents

Each release produces 5 archives, all rooted at `.kit/` (e.g., `.kit/VERSION`, `.kit/skills/fab-new.md`, `.kit/packages/idea/bin/idea`):

- **`kit.tar.gz`** — Generic archive containing shell scripts, skills, templates, packages, and all non-binary `.kit/` contents. No Go binary included. Serves as a fallback for unsupported platforms.
- **`kit-darwin-arm64.tar.gz`** — Generic contents + Go binary at `.kit/bin/fab` compiled for macOS Apple Silicon.
- **`kit-darwin-amd64.tar.gz`** — Generic contents + Go binary at `.kit/bin/fab` compiled for macOS Intel.
- **`kit-linux-arm64.tar.gz`** — Generic contents + Go binary at `.kit/bin/fab` compiled for Linux ARM64.
- **`kit-linux-amd64.tar.gz`** — Generic contents + Go binary at `.kit/bin/fab` compiled for Linux x86-64.

No project-specific files (config.yaml, constitution.md, memory/, specs/, changes/) are included in any archive. Package production code (idea, wt) is included under `.kit/packages/`, hook scripts under `.kit/hooks/`, and sync scripts under `.kit/sync/` — all delivered to downstream projects on upgrade. The Go binary is placed at `.kit/bin/fab` in platform archives; the `bin/` directory contains a `.gitkeep` to ensure the directory exists even in the generic archive.

### Repo Rename

The repository SHALL be renamed from `docs-sddr` to `fab-kit` to reflect its role as the canonical source for `fab/.kit/`. GitHub auto-redirects handle existing URLs and clones.

**Scenarios**:
- Old URLs (`github.com/wvrdz/docs-sddr`) redirect to the current repo URL
- Existing clones with old remote URL continue to work via redirect

## Design Decisions

<!-- No design decisions to document for this change. -->

## Changelog

| Change | Date | Summary |
|--------|------|---------|
| 260305-bs5x-orchestrator-idle-hooks | 2026-03-05 | Added `fab/.kit/hooks/` as a new distributed directory (hook scripts shipped with kit). Updated bootstrap description to mention hook registration via `5-sync-hooks.sh`. Updated release archive contents to note hooks and sync scripts alongside packages. |
| 260305-g0uq-2-ship-fab-go-binary | 2026-03-05 | Ship fab Go binary: release now produces 5 archives (generic `kit.tar.gz` + 4 per-platform `kit-{os}-{arch}.tar.gz` with Go binary at `.kit/bin/fab`). `fab-release.sh` cross-compiles via `CGO_ENABLED=0` for darwin/arm64, darwin/amd64, linux/arm64, linux/amd64. `fab-upgrade.sh` detects platform via `uname -s`/`uname -m`, downloads platform archive with fallback to generic. README bootstrap one-liner is now platform-aware. Shell scripts in `lib/` have shim layer that delegates to Go binary when present. Skills updated via `_scripts.md` to invoke `fab/.kit/bin/fab` as primary calling convention. |
| 260305-bhd6-1-build-fab-go-binary | 2026-03-05 | Go binary (`src/fab-go/`) built — ports all lib/ scripts to single `fab` binary. No distribution changes in this change — binary inclusion in kit.tar.gz and per-platform archives are deferred to a future change. Go toolchain required only for building from source, not for end users. |
| 260303-l6nk-gemini-cli-agent-aware-sync | 2026-03-04 | Added Gemini CLI as 4th supported agent. Updated bootstrap/sync descriptions to reflect conditional agent deployment (skills deployed only when agent's CLI found in PATH). Four agents: Claude Code (copies), OpenCode (symlinks), Codex (copies), Gemini CLI (copies). |
| 260301-08pa-version-pinned-upgrade-and-release | 2026-03-02 | Added version-pinned upgrade (`fab-upgrade.sh v0.24.0`) with tag-aware messaging. Added backport release support to `fab-release.sh`: push to current branch instead of hardcoded `main`, `--no-latest` flag for `gh release create --latest=false`, position-independent argument parsing. |
| 260226-koj1-version-staleness-warning | 2026-02-26 | Added sync staleness detection (`fab/.kit-sync-version` stamp, preflight stderr warning). Renamed `fab/project/VERSION` → `fab/.kit-migration-version`. Updated preserved files list in upgrade section. |
| 260224-v40o-wt-drop-prefix-and-dotworktrees | 2026-02-25 | wt package: dropped `wt/` branch prefix from exploratory worktrees (branch = worktree name directly). Switched worktree home directory from `<repo>-worktrees` to `<repo>.worktrees` (GitLens convention). Updated `wt-create` help text. No migration for existing worktrees. |
| 260221-i0z6-move-env-packages-add-fab-pipeline | 2026-02-21 | `env-packages.sh` moved from `scripts/` to `scripts/lib/` — now sourced from `fab/.kit/scripts/lib/env-packages.sh` in both `scaffold/fragment-.envrc` and `src/packages/rc-init.sh` |
| 260219-d2y2-copy-template-skills-drop-agents | 2026-02-19 | Updated references from symlinks to copies for Claude Code skills. Renamed "Symlink Repair After Update" to "Skill Deployment Repair After Update". Updated bootstrap and upgrade descriptions to reflect copy-with-template deployment |
| 260218-cif4-eliminate-symlinks-distribute-packages | 2026-02-18 | Package production code (idea, wt) now distributed via `kit.tar.gz` under `.kit/packages/`. Updated release archive contents description. Updated `fab-upgrade.sh` description (symlinks → directories and agents). Added `env-packages.sh` for centralized PATH setup, sourced by `scaffold/envrc` (direnv) and `src/packages/rc-init.sh` (shell rc). |
| 260217-zkah-readme-quickstart-prereqs-check | 2026-02-18 | Added prerequisites validation to `fab-sync.sh` pipeline (via `sync/1-prerequisites.sh`). Updated bootstrap description to mention prerequisites check. Restructured README Quick Start: folded Initialize and Updating under Install as sub-sections. |
| 260216-ymvx-DEV-1043-envrc-line-sync | 2026-02-16 | Updated `.envrc` references from symlink to line-ensuring: bootstrap description now says "`.envrc` entries (from `scaffold/envrc`, line-ensuring)"; scenario updated to note line-ensuring from scaffold |
| 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow | 2026-02-16 | `lib/sync-workspace.sh` → `fab-sync.sh` (promoted to `scripts/`); `/fab-init` → `/fab-setup`; `/fab-update` → `/fab-setup migrations` |
| 260216-b1k9-DEV-1028-rename-scaffold-add-kit-tests | 2026-02-16 | Renamed `init-scaffold.sh` → `sync-workspace.sh` throughout (bootstrap description, update script references, symlink repair) |
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added version drift scenarios to update section; added `fab/VERSION` to preserved files list; added migration chain validation to release section |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated bootstrap description to mention `fab-sync.sh` reads from `scaffold/` files for index templates, envrc, and gitignore entries |
| 260214-q7f2-reorganize-src | 2026-02-14 | Renamed `_init_scaffold.sh` → `fab-sync.sh` throughout; moved `fab-release.sh` from `fab/.kit/scripts/` to `src/scripts/` (dev-only, not shipped in kit) |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed script references: `fab-setup.sh` → `_init_scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Updated bootstrap description to include `docs/specs/` directory and `design/index.md` in `fab-sync.sh` output |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Initial creation — bootstrap, update, release, and repo rename requirements |
