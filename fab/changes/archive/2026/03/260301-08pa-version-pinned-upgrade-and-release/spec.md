# Spec: Version-Pinned Upgrade and Release

**Change**: 260301-08pa-version-pinned-upgrade-and-release
**Created**: 2026-03-02
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Version comparison or downgrade warnings — the upgrade script downloads whatever tag is requested, no directional checks
- Curl fallback for `gh` — deferred to a future enhancement (existing decision)
- Backport branch management automation — the backport workflow is manual (checkout, cherry-pick, release)

## Upgrade: Version-Pinned Download

### Requirement: Accept optional release tag argument

`fab/.kit/scripts/fab-upgrade.sh` SHALL accept an optional positional argument `$1` as a release tag. When provided, the script SHALL download that specific tagged release instead of "latest". When omitted, behavior SHALL be identical to the current implementation (downloads latest).

#### Scenario: Download a specific tagged release
- **GIVEN** `fab/.kit/` is installed at version 0.25.1
- **WHEN** the user runs `fab-upgrade.sh v0.24.0`
- **THEN** the script downloads `kit.tar.gz` from the `v0.24.0` release
- **AND** replaces `fab/.kit/` with the downloaded version
- **AND** displays "Updating: 0.25.1 → 0.24.0"
- **AND** re-runs `fab-sync.sh`

#### Scenario: Download latest (no argument)
- **GIVEN** `fab/.kit/` is installed at version 0.25.0
- **WHEN** the user runs `fab-upgrade.sh` with no arguments
- **THEN** the script downloads `kit.tar.gz` from the latest release (existing behavior)

#### Scenario: Tag not found
- **GIVEN** the user runs `fab-upgrade.sh v99.99.99`
- **WHEN** `gh release download` fails
- **THEN** the script displays: `ERROR: Failed to download kit.tar.gz for tag 'v99.99.99' from $repo.`
- **AND** displays: `Check that the tag exists: gh release view v99.99.99 --repo $repo`
- **AND** exits non-zero

#### Scenario: Already on the requested tag
- **GIVEN** `fab/.kit/` is installed at version 0.24.0
- **WHEN** the user runs `fab-upgrade.sh v0.24.0`
- **THEN** the script displays: `Already on v0.24.0 (0.24.0). No update needed.`
- **AND** exits 0 without modifying `fab/.kit/`

## Release: Backport Support

### Requirement: Push to current branch

`src/scripts/fab-release.sh` SHALL push to the current branch (as reported by `git branch --show-current`) instead of the hardcoded `main` branch. On `main`, behavior is identical to today. On a release branch (e.g., `release/0.25`), commits and tags are pushed to that branch.

#### Scenario: Release from main
- **GIVEN** the user is on the `main` branch
- **WHEN** the user runs `fab-release.sh patch`
- **THEN** the script pushes `HEAD:main` and the tag (identical to current behavior)

#### Scenario: Release from a release branch
- **GIVEN** the user is on branch `release/0.25`
- **WHEN** the user runs `fab-release.sh patch --no-latest`
- **THEN** the script pushes `HEAD:release/0.25` and the tag
- **AND** does NOT push to `main`

### Requirement: --no-latest flag

`src/scripts/fab-release.sh` SHALL accept an optional `--no-latest` flag that prevents the release from being marked as "latest" on GitHub. The flag is position-independent — it MAY appear before or after the bump type argument.

#### Scenario: Normal release (no flag)
- **GIVEN** the user runs `fab-release.sh patch`
- **WHEN** the release is created
- **THEN** `gh release create` is invoked without `--latest=false`
- **AND** the release becomes the "latest" release on GitHub (default behavior)

#### Scenario: Backport release with --no-latest
- **GIVEN** the user runs `fab-release.sh patch --no-latest`
- **WHEN** the release is created
- **THEN** `gh release create` is invoked with `--latest=false`
- **AND** the completion output includes: `Note: This release was NOT marked as "latest".`

#### Scenario: Full backport workflow
- **GIVEN** the latest release is v0.26.0 on `main`
- **WHEN** the maintainer runs:
  ```
  git checkout -b release/0.25 v0.25.1
  git cherry-pick <commit>
  fab-release.sh patch --no-latest
  ```
- **THEN** VERSION is bumped from 0.25.1 to 0.25.2
- **AND** the commit and tag `v0.25.2` are pushed to `release/0.25`
- **AND** the GitHub release `v0.25.2` is created with `--latest=false`
- **AND** `fab-upgrade.sh` (no args) for other users still resolves to v0.26.0

### Requirement: Argument parsing rework

The release script SHALL identify the first non-flag argument as the bump type (required) and scan all arguments for `--no-latest`. The bump type validation (patch/minor/major) SHALL remain unchanged. Unknown flags SHALL produce an error.

#### Scenario: Invalid flag
- **GIVEN** the user runs `fab-release.sh patch --unknown`
- **WHEN** argument parsing encounters `--unknown`
- **THEN** the script displays an error listing valid options
- **AND** exits non-zero

#### Scenario: Missing bump type
- **GIVEN** the user runs `fab-release.sh --no-latest`
- **WHEN** argument parsing runs
- **THEN** `--no-latest` is NOT treated as the bump type
- **AND** the script displays the usage message (bump type is required)

## Documentation: README Update

### Requirement: Version-pinned upgrade in Quick Start

The README's "Updating from a previous version" section SHALL include the version-pinned usage example alongside the existing upgrade line.

#### Scenario: README documents both upgrade modes
- **GIVEN** a user reads the Quick Start section
- **WHEN** they reach the "Updating" subsection
- **THEN** they see both `fab-upgrade.sh` (latest) and `fab-upgrade.sh v0.24.0` (specific version)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Accept full tag name as-is, no normalization | Confirmed from intake #1 — user explicitly said "keep it same as the tag version" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Use `--no-latest` flag name for release script | Confirmed from intake #2 — matches `gh release create --latest=false` convention | S:80 R:85 A:85 D:80 |
| 3 | Certain | No version comparison logic for downgrades | Confirmed from intake #3 — download proceeds regardless of direction | S:80 R:90 A:80 D:85 |
| 4 | Confident | `--no-latest` as flag rather than separate subcommand | Confirmed from intake #4 — additive to existing interface | S:70 R:85 A:75 D:70 |
| 5 | Certain | Bump type still required with `--no-latest` | Confirmed from intake #5 — backport releases still need version bumping | S:85 R:85 A:90 D:90 |
| 6 | Certain | Push to current branch instead of hardcoded `main` | Confirmed from intake #6 — makes backport workflow possible | S:90 R:80 A:90 D:90 |
| 7 | Certain | `--no-latest` is position-independent in argument list | Scanning remaining args after bump type; no reason to enforce position | S:75 R:90 A:85 D:85 |
| 8 | Certain | Unknown flags produce an error | Defensive — prevents silent misuse (e.g., typo `--no-latset`) | S:80 R:90 A:90 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
