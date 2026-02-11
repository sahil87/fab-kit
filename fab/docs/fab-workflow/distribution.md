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

After extraction, the user MUST run `fab/.kit/scripts/fab-setup.sh` to create directories, symlinks, and skeleton files. The bootstrap only provides `.kit/` — no `config.yaml`, `constitution.md`, or other project files.

**Scenarios**:
- Bootstrap a new project (no `fab/` directory) — creates `fab/.kit/` with all skills, templates, scripts, and VERSION file; no other `fab/` files created
- Bootstrap with existing `fab/` directory — creates or replaces `fab/.kit/`; existing files outside `.kit/` (config.yaml, constitution.md, docs/, design/, changes/) are NOT affected

#### Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work. The bootstrap one-liner is an additive convenience, not a replacement.

**Scenario**: Manual copy (`cp -r /path/to/fab-kit/fab/.kit fab/.kit`) produces an identical result to the curl bootstrap.

### Update

#### Update Script (`fab-update.sh`)

`fab/.kit/scripts/fab-update.sh` SHALL download the latest `kit.tar.gz` from GitHub Releases, extract it to replace the current `fab/.kit/` contents, display the version change, and re-run `fab-setup.sh` to repair symlinks.

**Scenarios**:
- Update to a newer version — replaces `.kit/` contents, displays version change (e.g., "0.1.0 → 0.2.0"), re-runs `fab-setup.sh`, preserves all files outside `.kit/`
- Already up to date — informs user, no files modified
- No network access — exits non-zero with error message, existing `.kit/` unchanged

#### Update Preserves Project Files

`fab-update.sh` MUST NOT modify any files outside of `fab/.kit/`. Preserved: `fab/config.yaml`, `fab/constitution.md`, `fab/docs/`, `fab/design/`, `fab/changes/`, `fab/current`.

#### gh CLI as Primary Download Tool

`fab-update.sh` SHALL use `gh release download` as the primary method to download the release asset. If `gh` is not installed, the script exits with an error directing the user to install it. Curl fallback is deferred to a future enhancement.

#### Atomic Update

`fab-update.sh` SHALL use an atomic update strategy: extract `kit.tar.gz` to a temporary directory, verify the extraction succeeded (checks for VERSION file), then replace the existing `fab/.kit/` via `rm -rf` and `mv`. This prevents corruption if interrupted mid-extraction.

**Scenarios**:
- Interrupted during download — existing `.kit/` unchanged
- Interrupted during extraction to temp dir — existing `.kit/` unchanged, temp dir cleaned up on next run
- Extraction verification fails — aborts without replacing `.kit/`, displays error

#### Symlink Repair After Update

After extracting the new `.kit/` contents, `fab-update.sh` SHALL re-run `fab-setup.sh` to ensure all agent symlinks (`.claude/skills/`, `.opencode/commands/`, `.agents/skills/`) point to the updated skill files.

### Release

#### Release Script (`fab-release.sh`)

`fab/.kit/scripts/fab-release.sh` SHALL package `fab/.kit/` into a `kit.tar.gz` archive, bump the VERSION file, commit the version change, and create a GitHub Release with `kit.tar.gz` as an attached asset.

The script accepts an optional argument specifying the bump type: `patch` (default), `minor`, or `major`.

The script infers the target GitHub repository from `git remote get-url origin`. It does not hardcode any repository name, ensuring it works for forks and renamed repos.

**Scenarios**:
- Default patch release — bumps patch version (e.g., "0.1.0" → "0.1.1"), creates `kit.tar.gz`, commits VERSION bump, creates GitHub Release
- Minor release (`fab-release.sh minor`) — bumps minor version (e.g., "0.1.1" → "0.2.0")
- Major release (`fab-release.sh major`) — bumps major version (e.g., "0.2.0" → "1.0.0")
- Invalid bump argument — exits with error message listing valid options
- No git remote configured — exits with error
- Dirty working tree — aborts with error directing user to commit or stash

#### Release Archive Contents

`kit.tar.gz` SHALL contain only the `fab/.kit/` directory contents. All paths are rooted at `.kit/` (e.g., `.kit/VERSION`, `.kit/skills/fab-new.md`). No project-specific files (config.yaml, constitution.md, docs/, design/, changes/) are included.

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
| 260210-h7r3-kit-distribution-update | 2026-02-10 | Initial creation — bootstrap, update, release, and repo rename requirements |
