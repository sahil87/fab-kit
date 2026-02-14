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

After extraction, the user MUST run `fab/.kit/scripts/_init_scaffold.sh` to create directories (`changes/`, `memory/`, `specs/`), skeleton files (copied from `scaffold/memory-index.md` and `scaffold/specs-index.md`), symlinks, `.envrc` (symlink to `scaffold/envrc`), and `.gitignore` entries (from `scaffold/gitignore-entries`). The bootstrap only provides `.kit/` — no `config.yaml`, `constitution.md`, or other project files.

**Scenarios**:
- Bootstrap a new project (no `fab/` directory) — creates `fab/.kit/` with all skills, templates, scripts, and VERSION file; running `_init_scaffold.sh` then creates `changes/`, `memory/index.md`, `specs/index.md`, symlinks, `.envrc`, and `.gitignore` entry
- Bootstrap with existing `fab/` directory — creates or replaces `fab/.kit/`; existing files outside `.kit/` (config.yaml, constitution.md, memory/, specs/, changes/) are NOT affected

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work. The bootstrap one-liner is an additive convenience, not a replacement.

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### Update Script (`fab-upgrade.sh`)

`fab/.kit/scripts/fab-upgrade.sh` SHALL download the latest `kit.tar.gz` from GitHub Releases, extract it to replace the current `fab/.kit/` contents, display the version change, and re-run `_init_scaffold.sh` to repair symlinks.

**Scenarios**:
- Update to a newer version — replaces `.kit/` contents, displays version change (e.g., "0.1.0 → 0.2.0"), re-runs `_init_scaffold.sh`, checks for version drift and prints `/fab-update` reminder if needed, preserves all files outside `.kit/`
- Already up to date — informs user, no files modified
- No network access — exits non-zero with error message, existing `.kit/` unchanged
- `fab/VERSION` missing after upgrade — prints guidance to run `/fab-init` then `/fab-update`
- `fab/VERSION` behind new engine version — prints reminder to run `/fab-update` to apply migrations

#### Update Preserves Project Files

`fab-upgrade.sh` MUST NOT modify any files outside of `fab/.kit/`. Preserved: `fab/config.yaml`, `fab/constitution.md`, `fab/VERSION`, `fab/memory/`, `fab/specs/`, `fab/changes/`, `fab/current`.

#### gh CLI as Primary Download Tool

`fab-upgrade.sh` SHALL use `gh release download` as the primary method to download the release asset. If `gh` is not installed, the script exits with an error directing the user to install it. Curl fallback is deferred to a future enhancement.

#### Atomic Update

`fab-upgrade.sh` SHALL use an atomic update strategy: extract `kit.tar.gz` to a temporary directory, verify the extraction succeeded (checks for VERSION file), then replace the existing `fab/.kit/` via `rm -rf` and `mv`. This prevents corruption if interrupted mid-extraction.

**Scenarios**:
- Interrupted during download — existing `.kit/` unchanged
- Interrupted during extraction to temp dir — existing `.kit/` unchanged, temp dir cleaned up on next run
- Extraction verification fails — aborts without replacing `.kit/`, displays error

#### Symlink Repair After Update

After extracting the new `.kit/` contents, `fab-upgrade.sh` SHALL re-run `_init_scaffold.sh` to ensure all agent symlinks (`.claude/skills/`, `.opencode/commands/`, `.agents/skills/`) point to the updated skill files.

### Release

#### Release Script (`fab-release.sh`)

`fab/.kit/scripts/fab-release.sh` SHALL package `fab/.kit/` into a `kit.tar.gz` archive, bump the VERSION file, commit the version change, and create a GitHub Release with `kit.tar.gz` as an attached asset.

The script accepts an optional argument specifying the bump type: `patch` (default), `minor`, or `major`.

After bumping VERSION, the script validates the migration chain: warns if no migration file targets the new version (reminder for release authors), and warns if overlapping migration ranges are detected. These are warnings only — they do not block the release.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.1.0" → "0.1.1"), creates `kit.tar.gz`, commits VERSION bump, creates GitHub Release
- Minor release (`fab-release.sh minor`) — bumps minor version (e.g., "0.1.1" → "0.2.0")
- Major release (`fab-release.sh major`) — bumps major version (e.g., "0.2.0" → "1.0.0")
- Invalid bump argument — exits with error message listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Release Archive Contents

`kit.tar.gz` SHALL contain only the `fab/.kit/` directory contents. All paths are rooted at `.kit/` (e.g., `.kit/VERSION`, `.kit/skills/fab-new.md`). No project-specific files (config.yaml, constitution.md, memory/, specs/, changes/) are included.

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
| 260213-k7m2-kit-version-migrations | 2026-02-14 | Added version drift scenarios to update section; added `fab/VERSION` to preserved files list; added migration chain validation to release section |
| 260213-3njv-scaffold-dir | 2026-02-13 | Updated bootstrap description to mention `_init_scaffold.sh` reads from `scaffold/` files for index templates, envrc, and gitignore entries |
| 260213-iq2l-rename-setup-scripts | 2026-02-13 | Renamed script references: `fab-setup.sh` → `_init_scaffold.sh`, `fab-update.sh` → `fab-upgrade.sh` |
| 260212-emcb-clarify-fab-setup | 2026-02-12 | Updated bootstrap description to include `fab/specs/` directory and `design/index.md` in `_init_scaffold.sh` output |
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Initial creation — bootstrap, update, release, and repo rename requirements |
