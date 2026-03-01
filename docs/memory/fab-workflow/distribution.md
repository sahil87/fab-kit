# Distribution

**Domain**: fab-workflow

## Overview

How `fab/.kit/` is distributed to new and existing projects. Covers the bootstrap process (getting `.kit/` into a project for the first time), the update mechanism (pulling new versions into an existing project), the release workflow (packaging and publishing new versions), and the repo rename from `docs-sddr` to `fab-kit`.

## Requirements

### Bootstrap

#### One-Liner Bootstrap

New projects SHALL be bootstrappable via a single curl command that downloads the latest `kit.tar.gz` from GitHub Releases and extracts it into `fab/.kit/`:

```
mkdir -p fab
curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

After extraction, the user MUST run `fab/.kit/scripts/fab-sync.sh` to validate prerequisites (`yq`, `jq`, `gh`, `direnv`, `bats`), create directories (`changes/`, `memory/`, `specs/`), skeleton files (copied from `scaffold/memory-index.md` and `scaffold/specs-index.md`), deploy skills (copies with model templating for Claude Code, symlinks for OpenCode, copies for Codex), `.envrc` entries (from `scaffold/envrc`, line-ensuring), and `.gitignore` entries (from `scaffold/gitignore-entries`). The bootstrap only provides `.kit/` — no `config.yaml`, `constitution.md`, or other project files.

**Scenarios**:
- Bootstrap a new project (no `fab/` directory) — creates `fab/.kit/` with all skills, templates, scripts, and VERSION file; running `fab-sync.sh` first validates prerequisites (yq, jq, gh, direnv, bats), then creates `changes/`, `memory/index.md`, `specs/index.md`, skill deployments (copies/symlinks per platform), `.envrc` (line-ensuring from scaffold), and `.gitignore` entry
- Bootstrap with existing `fab/` directory — creates or replaces `fab/.kit/`; existing files outside `.kit/` (config.yaml, constitution.md, memory/, specs/, changes/) are NOT affected

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work. The bootstrap one-liner is an additive convenience, not a replacement.

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### Update Script (`fab-upgrade.sh`)

`fab/.kit/scripts/fab-upgrade.sh` SHALL download the latest `kit.tar.gz` from GitHub Releases, extract it to replace the current `fab/.kit/` contents, display the version change, and re-run `fab-sync.sh` to repair directories and skill deployments.

The script accepts an optional positional argument — a release tag (e.g., `v0.24.0`) — to download a specific version instead of latest. The tag is passed as-is to `gh release download "$tag"`, with no normalization or `v`-prefix stripping.

**Scenarios**:
- Update to a newer version — replaces `.kit/` contents, displays version change (e.g., "0.1.0 → 0.2.0"), re-runs `fab-sync.sh`, checks for version drift and prints `/fab-setup migrations` reminder if needed, preserves all files outside `.kit/`
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

`src/scripts/fab-release.sh` SHALL package `fab/.kit/` into a `kit.tar.gz` archive, bump the VERSION file, commit the version change, and create a GitHub Release with `kit.tar.gz` as an attached asset.

The script accepts a required bump type argument (`patch`, `minor`, or `major`) and an optional `--no-latest` flag. Arguments are position-independent — `fab-release.sh patch --no-latest` and `fab-release.sh --no-latest patch` are equivalent. Unknown flags produce an error.

The script pushes to the current branch (via `git branch --show-current`) rather than hardcoded `main`. On `main`, behavior is identical to before. On a release branch (e.g., `release/0.25`), commits and tags are pushed to that branch.

When `--no-latest` is passed, `gh release create` is invoked with `--latest=false`, preventing the release from becoming the "latest" release on GitHub. This enables backport releases for older version series without breaking `fab-upgrade.sh` (no args) for users on the current release line.

After bumping VERSION, the script validates the migration chain: warns if no migration file targets the new version (reminder for release authors), and warns if overlapping migration ranges are detected. These are warnings only — they do not block the release.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.1.0" → "0.1.1"), creates `kit.tar.gz`, commits VERSION bump, pushes to current branch, creates GitHub Release marked as latest
- Minor release (`fab-release.sh minor`) — bumps minor version (e.g., "0.1.1" → "0.2.0")
- Major release (`fab-release.sh major`) — bumps major version (e.g., "0.2.0" → "1.0.0")
- Backport release (`fab-release.sh patch --no-latest`) — bumps version, pushes to current branch (not main), creates GitHub Release with `--latest=false`, prints "Note: This release was NOT marked as 'latest'."
- Backport workflow — `git checkout -b release/0.25 v0.25.1`, cherry-pick fixes, `fab-release.sh patch --no-latest` bumps 0.25.1→0.25.2, pushes to `release/0.25`, tags `v0.25.2`
- Missing bump type with flags (`fab-release.sh --no-latest`) — displays usage and exits non-zero
- Invalid bump argument — exits with error message listing valid options
- Unknown flag (`fab-release.sh patch --unknown`) — exits with error listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Release Archive Contents

`kit.tar.gz` SHALL contain only the `fab/.kit/` directory contents. All paths are rooted at `.kit/` (e.g., `.kit/VERSION`, `.kit/skills/fab-new.md`, `.kit/packages/idea/bin/idea`). No project-specific files (config.yaml, constitution.md, memory/, specs/, changes/) are included. Package production code (idea, wt) is included under `.kit/packages/` and delivered to downstream projects on upgrade.

### Repo Rename

The repository SHALL be renamed from `docs-sddr` to `fab-kit` to reflect its role as the canonical source for `fab/.kit/`. GitHub auto-redirects handle existing URLs and clones.

**Scenarios**:
- Old URLs (`github.com/wvrdz/docs-sddr`) redirect to `github.com/wvrdz/fab-kit`
- Existing clones with old remote URL continue to work via redirect

## Design Decisions

<!-- No design decisions to document for this change. -->

## Changelog

| Change | Date | Summary |
|--------|------|---------|
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
