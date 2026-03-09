# Spec: CI Releases with Justfile

**Change**: 260307-ma7o-1-ci-releases-justfile
**Created**: 2026-03-09
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Rust backend cross-compilation — deferred to steps 3-4 of the migration plan
- Changes to `fab-upgrade.sh` or bootstrap one-liner — they consume GitHub Releases as before
- Adding `just` to `fab-sync.sh` prerequisites — `just` is dev-only, not needed by end users

## Build Recipes: Justfile

### Requirement: Local Go Build

The `justfile` at repo root SHALL provide a `build-go` recipe that compiles the Go binary for the current platform and places it at `fab/.kit/bin/fab-go`. The build SHALL use `CGO_ENABLED=0` for a static binary.

#### Scenario: Developer builds locally
- **GIVEN** the Go toolchain is installed
- **WHEN** the developer runs `just build-go`
- **THEN** a static Go binary is compiled at `fab/.kit/bin/fab-go` for the current platform

### Requirement: Cross-Compile for Target

The `justfile` SHALL provide a `build-go-target` recipe accepting `os` and `arch` parameters that cross-compiles the Go binary to `.release-build/fab-{os}-{arch}` (relative to repo root). The build SHALL use `CGO_ENABLED=0` with `GOOS` and `GOARCH` set to the specified values.

#### Scenario: Cross-compile for a specific platform
- **GIVEN** the Go toolchain is installed
- **WHEN** the developer runs `just build-go-target darwin arm64`
- **THEN** a static Go binary is compiled at `.release-build/fab-darwin-arm64`

### Requirement: Cross-Compile All Targets

The `justfile` SHALL provide a `build-go-all` recipe that cross-compiles the Go binary for all 4 release targets: `darwin/arm64`, `darwin/amd64`, `linux/arm64`, `linux/amd64`. It SHALL invoke `build-go-target` for each platform.

#### Scenario: Build all release targets
- **GIVEN** the Go toolchain is installed
- **WHEN** the developer runs `just build-go-all`
- **THEN** 4 binaries are produced in `.release-build/`: `fab-darwin-arm64`, `fab-darwin-amd64`, `fab-linux-arm64`, `fab-linux-amd64`

### Requirement: Package Kit Archives

The `justfile` SHALL provide a `package-kit` recipe that creates 5 tar.gz archives in the repo root:
1. `kit.tar.gz` — `.kit/` contents excluding any locally-built Go binary
2. `kit-{os}-{arch}.tar.gz` (4 archives) — `.kit/` contents plus the cross-compiled Go binary at `.kit/bin/fab-go`

The recipe SHALL:
- Create per-platform staging directories in `.release-build/`
- Copy `fab/.kit/` into each staging area
- Place the cross-compiled binary at `.kit/bin/fab-go` with executable permissions
- Produce archives rooted at `.kit/` (matching existing archive structure)
- Use `COPYFILE_DISABLE=1` to suppress macOS extended attributes

#### Scenario: Package after successful cross-compilation
- **GIVEN** `just build-go-all` has completed successfully
- **WHEN** the developer runs `just package-kit`
- **THEN** 5 archives are created in the repo root
- **AND** each per-platform archive contains `.kit/bin/fab-go`
- **AND** the generic `kit.tar.gz` does NOT contain `.kit/bin/fab-go`

#### Scenario: Package without prior build
- **GIVEN** `.release-build/` does not exist or lacks expected binaries
- **WHEN** the developer runs `just package-kit`
- **THEN** the recipe fails with an error

### Requirement: Clean Build Artifacts

The `justfile` SHALL provide a `clean` recipe that removes the `.release-build/` directory and all `kit*.tar.gz` files from the repo root.

#### Scenario: Clean up after build
- **GIVEN** build artifacts exist
- **WHEN** the developer runs `just clean`
- **THEN** `.release-build/` is removed
- **AND** `kit.tar.gz` and `kit-*.tar.gz` are removed from the repo root

## CI Workflow: GitHub Actions Release

### Requirement: Tag-Triggered Workflow

`.github/workflows/release.yml` SHALL trigger on push of tags matching `v*`.

#### Scenario: Tag push triggers workflow
- **GIVEN** a commit is tagged `v0.35.0`
- **WHEN** the tag is pushed
- **THEN** the release workflow runs

#### Scenario: Non-tag push does not trigger
- **GIVEN** a commit is pushed without a `v*` tag
- **WHEN** the push completes
- **THEN** no release workflow runs

### Requirement: Single Runner with Just Recipes

The workflow SHALL run on a single `ubuntu-latest` runner and use the same `just` recipes as local development:
1. `just build-go-all` for cross-compilation
2. `just package-kit` for archive creation

No CI-only build scripts or logic.

#### Scenario: Full CI release
- **GIVEN** the workflow is triggered by tag `v0.35.0`
- **WHEN** all steps complete
- **THEN** a GitHub Release is created for `v0.35.0` with 5 archives attached
- **AND** release notes are auto-generated

### Requirement: Workflow Steps

The workflow SHALL execute these steps in order:
1. Checkout the repository
2. Set up Go toolchain (via `actions/setup-go`)
3. Install `just` command runner
4. Run `just build-go-all`
5. Run `just package-kit`
6. Create GitHub Release via `gh release create` with all 5 archives and auto-generated notes

The workflow SHALL set `permissions: contents: write` for release creation. The `GITHUB_TOKEN` is used implicitly by `gh`.

#### Scenario: Workflow has required permissions
- **GIVEN** the workflow triggers
- **WHEN** the `gh release create` step runs
- **THEN** it succeeds because `contents: write` permission is set

## Release Script: fab-release.sh

### Requirement: Simplified Release Script

`src/scripts/fab-release.sh` SHALL be simplified to handle only version management and git operations:
- Read current version from `fab/.kit/VERSION`
- Compute new version based on bump type (patch/minor/major)
- Write new version to `fab/.kit/VERSION`
- Validate migration chain (warnings only)
- Commit VERSION change with message `release: v{version}`
- Create git tag `v{version}`
- Push commit and tag to current branch

The script SHALL retain argument parsing for the bump type argument (patch/minor/major).

#### Scenario: Standard release
- **GIVEN** VERSION contains `0.34.0` and working tree is clean
- **WHEN** the developer runs `fab-release.sh patch`
- **THEN** VERSION is updated to `0.34.1`
- **AND** a commit `release: v0.34.1` is created
- **AND** tag `v0.34.1` is created and pushed
- **AND** CI takes over from the tag push

#### Scenario: Minor release
- **GIVEN** VERSION contains `0.34.1`
- **WHEN** `fab-release.sh minor` is run
- **THEN** VERSION becomes `0.35.0`, committed and tagged as `v0.35.0`

#### Scenario: Backport release
- **GIVEN** the developer is on branch `release/0.34` with VERSION at `0.34.1`
- **WHEN** they run `fab-release.sh patch`
- **THEN** VERSION is bumped to `0.34.2`, committed, and pushed to `release/0.34`
- **AND** tag `v0.34.2` is pushed, triggering CI

### Requirement: Retained Pre-flight Checks

The script SHALL check:
- Working tree is clean (error if dirty)
- `fab/.kit/VERSION` exists (error if missing)

The script SHALL NOT check for `gh` CLI or Go toolchain (no longer needed).

#### Scenario: Dirty working tree
- **GIVEN** uncommitted changes exist
- **WHEN** the developer runs `fab-release.sh patch`
- **THEN** the script exits with error: "Working tree not clean. Commit or stash changes first."

#### Scenario: No arguments
- **GIVEN** the script is invoked with no arguments
- **WHEN** it runs
- **THEN** it displays usage and exits 0

## Deprecated Requirements

### `--no-latest` Flag in fab-release.sh

**Reason**: The `--no-latest` flag was used to pass `--latest=false` to `gh release create`. Since release creation is now handled by CI, and GitHub automatically determines "latest" status based on semver ordering (highest version = latest), the flag is no longer needed.

**Migration**: No action required. GitHub's default semver-based ordering handles the common backport case. For edge cases, use `gh release edit $TAG --latest=false` after CI creates the release.

### Go Toolchain Check in fab-release.sh

**Reason**: The release script no longer cross-compiles Go binaries — that's handled by `just build-go-all` in CI.

**Migration**: N/A — developers who run `just build-go-all` locally will get the error from Go itself.

### `gh` CLI Check in fab-release.sh

**Reason**: The release script no longer creates GitHub Releases — that's handled by CI.

**Migration**: N/A

### Cross-Compilation, Archive Packaging, and Release Creation in fab-release.sh

**Reason**: Cross-compilation moved to `justfile` (`build-go-target`, `build-go-all`). Archive packaging moved to `justfile` (`package-kit`). GitHub Release creation moved to `.github/workflows/release.yml`.

**Migration**: Use `just build-go-all && just package-kit` locally, or let CI handle everything via tag push.

## Design Decisions

1. **GitHub's semver-based "latest" replaces `--no-latest`**: When a release is created via `gh release create`, GitHub determines the "latest" release based on semver ordering. Backport releases for older version series (e.g., `v0.34.2` when `v0.35.0` exists) will not be marked as latest.
   - *Why*: Simpler workflow — no flag to remember, no CI mechanism to pass it through
   - *Rejected*: Passing `--no-latest` via commit message convention or workflow dispatch input — adds complexity for an edge case GitHub already handles

2. **Auto-generated release notes**: The CI workflow uses `gh release create --generate-notes` instead of manually constructing a changelog from `git log --oneline`.
   - *Why*: Richer content (PR titles, contributor info), less maintenance, consistent formatting
   - *Rejected*: Manually constructing changelog (current approach in fab-release.sh) — produces less informative notes

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `just` as task runner | Confirmed from intake #1 — user explicitly chose `just` | S:95 R:85 A:90 D:95 |
| 2 | Certain | Single Linux runner for Go cross-compilation | Confirmed from intake #2 — Go's CGO_ENABLED=0 produces correct binaries on any host | S:90 R:80 A:95 D:90 |
| 3 | Certain | Same 4 platform targets | Confirmed from intake #3 — no change to existing targets | S:95 R:90 A:95 D:95 |
| 4 | Certain | Tag push (`v*`) triggers CI release | Confirmed from intake #4 — standard GitHub Actions trigger | S:90 R:75 A:85 D:90 |
| 5 | Certain | `just` not added to fab-sync.sh prerequisites | Confirmed from intake #5 — dev-only tool, end users don't need it | S:85 R:90 A:90 D:90 |
| 6 | Certain | Packaging logic extracted to justfile | Confirmed from intake #6 — same staging/archive behavior | S:80 R:85 A:85 D:85 |
| 7 | Confident | CI uses `gh release create --generate-notes` | Upgraded from intake #7 — auto-generated notes richer than manual changelog | S:75 R:80 A:80 D:75 |
| 8 | Confident | `--no-latest` flag removed from fab-release.sh | GitHub's default semver ordering handles backport "latest" status correctly | S:70 R:75 A:80 D:75 |
| 9 | Confident | `clean` recipe added to justfile | Not in intake but logically needed for local dev artifact cleanup | S:65 R:90 A:85 D:85 |
| 10 | Certain | Go and gh CLI checks removed from fab-release.sh | Script no longer uses these tools — build/release delegated to justfile and CI | S:90 R:85 A:90 D:90 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
