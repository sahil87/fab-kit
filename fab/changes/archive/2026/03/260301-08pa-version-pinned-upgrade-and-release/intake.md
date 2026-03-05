# Intake: Version-Pinned Upgrade and Release

**Change**: 260301-08pa-version-pinned-upgrade-and-release
**Created**: 2026-03-01
**Status**: Draft

## Origin

> Add optional version argument to fab-upgrade.sh that accepts a release tag (e.g. v0.24.0) to download a specific version instead of latest. Also update the release process to support publishing patch releases for older version series without marking them as "latest" (using gh release create --latest=false).

Conversational `/fab-discuss` session preceded this intake. Key discussion points:
- `gh release download` already accepts an optional `[<tag>]` positional argument — minimal change to upgrade script
- GitHub marks the most recently *created* release as "latest" by default, which breaks older-series patch releases
- `gh release create --latest=false` is the mechanism to publish a release without changing the "latest" pointer
- User confirmed: accept the full tag name as-is (e.g. `v0.24.0`), no normalization or `v`-prefix stripping

## Why

1. **Version pinning**: Users on an older release series (e.g. v0.25.x) cannot currently pull a specific patch for their series — `fab-upgrade.sh` always downloads "latest", which may be a different major/minor line (e.g. v0.26.x). This forces an unwanted major upgrade or leaves them stuck.

2. **Older-series patch releases**: The release script (`fab-release.sh`) has no mechanism to publish a patch for an older series without that release becoming the new "latest". This means publishing v0.25.5 after v0.26.0 exists would break `fab-upgrade.sh` (no args) for everyone on the v0.26.x line.

3. **If we don't fix it**: Maintainers cannot safely backport fixes to older release lines, and users cannot pin to a known-good version during migration windows.

## What Changes

### 1. `fab/.kit/scripts/fab-upgrade.sh` — Accept optional tag argument

The upgrade script accepts an optional positional argument `$1` as a release tag:

```bash
fab-upgrade.sh           # downloads latest (existing behavior)
fab-upgrade.sh v0.24.0   # downloads the specific tagged release
```

When a tag is provided:
- Pass it as the first positional arg to `gh release download "$tag" --repo ...`
- Error message on download failure includes the tag and a hint: `Check that the tag exists: gh release view $tag --repo $repo`
- "Already up to date" message says `Already on $tag ($current_version)` instead of `Already on the latest version`

When no tag is provided — behavior is identical to today.

### 2. `src/scripts/fab-release.sh` — Push to current branch + `--no-latest` flag

Two changes to enable backport releases:

**a) Push to current branch instead of hardcoded `main`**

Replace the hardcoded `HEAD:main` push with the current branch:

```bash
# before (hardcoded)
git push git@github.com:"$repo".git HEAD:main "$tag"

# after (dynamic)
branch=$(git branch --show-current)
git push git@github.com:"$repo".git HEAD:"$branch" "$tag"
```

This is what makes the backport workflow actually work — without it, `--no-latest` alone isn't enough because the script would push backport commits onto `main`. On `main`, behavior is identical to today.

**b) Add `--no-latest` flag**

Add an optional `--no-latest` flag so maintainers can publish releases for older series:

```bash
fab-release.sh patch                # normal release, marked as "latest" (default)
fab-release.sh patch --no-latest    # backport release, NOT marked as "latest"
```

When `--no-latest` is passed:
- Add `--latest=false` to the `gh release create` invocation
- Display a note in the completion output: `Note: This release was NOT marked as "latest".`

**Argument parsing rework**: The script currently uses a simple `case` on `$1` for the bump type. This needs a small rework to handle both the positional bump-type arg and the `--no-latest` flag. Parse `$1` as bump type (required), then scan remaining args for `--no-latest`.

**Intended backport workflow**:

```bash
git checkout -b release/0.25 v0.25.1    # branch from the old tag
git cherry-pick <commit>                 # apply fixes
fab-release.sh patch --no-latest         # bumps 0.25.1→0.25.2, pushes to release/0.25, tags v0.25.2
```

The changelog also works correctly — `git describe --tags --abbrev=0` on the backport branch resolves to `v0.25.1`, so `git log v0.25.1..HEAD` only includes the cherry-picked commits.

### 3. `README.md` — Document version-pinned upgrade

Add the pinned-version usage example alongside the existing upgrade line in the Quick Start section.

### 4. `docs/memory/fab-workflow/distribution.md` — Update memory

Update the distribution memory to document:
- Version-pinned upgrade scenarios (specific tag, downgrade, tag not found)
- Backport release scenario (`--no-latest` flag, push to current branch, intended workflow)

## Affected Memory

- `fab-workflow/distribution`: (modify) Add version-pinned upgrade and backport release scenarios

## Impact

- `fab/.kit/scripts/fab-upgrade.sh` — shipped in kit, affects all downstream users
- `src/scripts/fab-release.sh` — dev-only, not shipped in kit
- `README.md` — documentation
- `docs/memory/fab-workflow/distribution.md` — memory file

No breaking changes — all modifications are additive (new optional arguments with backward-compatible defaults).

## Open Questions

- None — the approach was discussed and resolved in the preceding conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Accept full tag name as-is, no normalization | Discussed — user explicitly said "keep it same as the tag version" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use `--no-latest` flag name for release script | Convention matches `gh release create --latest=false`; clear, discoverable | S:80 R:85 A:85 D:80 |
| 3 | Certain | No version comparison logic for downgrades | Download proceeds regardless of direction; "already up to date" only triggers on exact match | S:80 R:90 A:80 D:85 |
| 4 | Confident | `--no-latest` as a flag rather than a separate subcommand | Additive to existing `patch|minor|major` interface; single boolean concern doesn't warrant a new subcommand | S:70 R:85 A:75 D:70 |
| 5 | Certain | Bump type still required with `--no-latest` | Backport releases still need version bumping; the flag only controls the "latest" marker | S:85 R:85 A:90 D:90 |
| 6 | Certain | Push to current branch instead of hardcoded `main` | Discussed — user proposed this; makes backport workflow possible without special-casing | S:90 R:80 A:90 D:90 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
