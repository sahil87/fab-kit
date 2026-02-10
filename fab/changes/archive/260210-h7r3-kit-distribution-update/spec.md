# Spec: Distribution & Update System for fab/.kit

**Change**: 260210-h7r3-kit-distribution-update
**Created**: 2026-02-10
**Affected docs**: `fab/docs/fab-workflow/distribution.md` (new), `fab/docs/fab-workflow/kit-architecture.md` (modified), `fab/docs/fab-workflow/init.md` (modified)

## Distribution: Bootstrap

### Requirement: One-Liner Bootstrap

New projects SHALL be bootstrappable via a single curl command that downloads the latest `kit.tar.gz` from GitHub Releases and extracts it into `fab/.kit/`.

The bootstrap command SHALL be:
```
curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

After extraction, the user MUST run `fab/.kit/scripts/fab-setup.sh` to create directories, symlinks, and skeleton files.

#### Scenario: Bootstrap a new project
- **GIVEN** a project with no `fab/` directory
- **WHEN** the user runs `mkdir -p fab && curl -sL https://github.com/wvrdz/fab-kit/releases/latest/download/kit.tar.gz | tar xz -C fab/`
- **THEN** `fab/.kit/` SHALL exist with all skills, templates, scripts, and VERSION file
- **AND** no other `fab/` files SHALL be created (no `config.yaml`, `constitution.md`, etc.)

#### Scenario: Bootstrap with existing fab/ directory
- **GIVEN** a project with an existing `fab/` directory (possibly containing `config.yaml`, `docs/`, etc.)
- **WHEN** the user runs the curl bootstrap command
- **THEN** `fab/.kit/` SHALL be created or replaced
- **AND** existing files outside `.kit/` (e.g., `config.yaml`, `constitution.md`, `docs/`, `changes/`) SHALL NOT be affected

### Requirement: Manual Copy Still Works

The existing `cp -r` distribution method SHALL continue to work. The bootstrap one-liner is an additive convenience, not a replacement.

#### Scenario: Manual copy bootstrap
- **GIVEN** a user with a local clone of the fab-kit repo
- **WHEN** the user runs `cp -r /path/to/fab-kit/fab/.kit fab/.kit`
- **THEN** the result SHALL be identical to using the curl bootstrap

## Distribution: Update

### Requirement: Update Script

`fab/.kit/scripts/fab-update.sh` SHALL download the latest `kit.tar.gz` from GitHub Releases, extract it to replace the current `fab/.kit/` contents, display the version change, and re-run `fab-setup.sh` to repair symlinks.

#### Scenario: Update to a newer version
- **GIVEN** a project with `fab/.kit/VERSION` containing "0.1.0"
- **WHEN** the user runs `bash fab/.kit/scripts/fab-update.sh`
- **AND** the latest release contains VERSION "0.2.0"
- **THEN** `fab/.kit/` contents SHALL be replaced with the release contents
- **AND** the script SHALL display the version change (e.g., "0.1.0 → 0.2.0")
- **AND** `fab/.kit/scripts/fab-setup.sh` SHALL be re-run automatically
- **AND** files outside `.kit/` SHALL NOT be affected

#### Scenario: Already up to date
- **GIVEN** a project with `fab/.kit/VERSION` matching the latest release version
- **WHEN** the user runs `fab-update.sh`
- **THEN** the script SHALL inform the user they are already on the latest version
- **AND** no files SHALL be modified

#### Scenario: Update with no network access
- **GIVEN** a project with `fab/.kit/` installed
- **WHEN** the user runs `fab-update.sh` without network access
- **THEN** the script SHALL exit with a non-zero code
- **AND** SHALL display an error message indicating the download failed
- **AND** the existing `fab/.kit/` SHALL remain unchanged

### Requirement: Update Preserves Project Files

`fab-update.sh` MUST NOT modify any files outside of `fab/.kit/`. Specifically, the following SHALL be preserved:
- `fab/config.yaml`
- `fab/constitution.md`
- `fab/docs/`
- `fab/specs/`
- `fab/changes/`
- `fab/current`

#### Scenario: Project files survive update
- **GIVEN** a project with customized `fab/config.yaml`, populated `fab/docs/`, and active changes in `fab/changes/`
- **WHEN** the user runs `fab-update.sh`
- **THEN** all files outside `fab/.kit/` SHALL remain unchanged
- **AND** only `fab/.kit/` contents SHALL be replaced

### Requirement: gh CLI as Primary Download Tool

`fab-update.sh` SHALL use the `gh` CLI (`gh release download`) as the primary method to download the release asset.
<!-- assumed: gh CLI as primary tool — standard GitHub CLI, widely available; curl fallback deferred to future enhancement -->

#### Scenario: gh CLI available
- **GIVEN** `gh` is installed and authenticated
- **WHEN** `fab-update.sh` runs
- **THEN** it SHALL use `gh release download` to fetch `kit.tar.gz`

#### Scenario: gh CLI not available
- **GIVEN** `gh` is not installed or not in PATH
- **WHEN** `fab-update.sh` runs
- **THEN** it SHALL exit with an error message: "gh CLI not found. Install it from https://cli.github.com/"
- **AND** the existing `.kit/` SHALL remain unchanged

### Requirement: Atomic Update

`fab-update.sh` SHALL use an atomic update strategy to prevent corruption if interrupted mid-extraction. The script SHALL extract `kit.tar.gz` to a temporary directory, verify the extraction succeeded, then replace the existing `fab/.kit/` via `rm -rf` and `mv`.
<!-- clarified: Atomic update via temp directory — user selected over direct extraction and backup strategies -->

#### Scenario: Update interrupted during download
- **GIVEN** a project with a working `fab/.kit/`
- **WHEN** `fab-update.sh` is interrupted during the download step
- **THEN** the existing `fab/.kit/` SHALL remain unchanged

#### Scenario: Update interrupted during extraction
- **GIVEN** a project with a working `fab/.kit/`
- **WHEN** `fab-update.sh` is interrupted during extraction to the temp directory
- **THEN** the existing `fab/.kit/` SHALL remain unchanged
- **AND** the temp directory SHALL be cleaned up on next run

#### Scenario: Extraction verification fails
- **GIVEN** `kit.tar.gz` was downloaded but extraction to the temp directory produced an incomplete result
- **WHEN** the script verifies the extraction (e.g., checks for `VERSION` file)
- **THEN** the script SHALL abort without replacing `fab/.kit/`
- **AND** SHALL display an error: "Extraction verification failed. Existing .kit/ unchanged."

### Requirement: Symlink Repair After Update

After extracting the new `.kit/` contents, `fab-update.sh` SHALL re-run `fab/.kit/scripts/fab-setup.sh` to ensure all agent symlinks point to the updated skill files.

#### Scenario: Symlinks repaired after update
- **GIVEN** an update has replaced `fab/.kit/` contents
- **WHEN** `fab-setup.sh` runs as part of the update
- **THEN** all symlinks in `.claude/skills/`, `.opencode/commands/`, and `.agents/skills/` SHALL point to the new skill files in `fab/.kit/skills/`

## Distribution: Release

### Requirement: Release Script

`fab/.kit/scripts/fab-release.sh` SHALL package `fab/.kit/` into a `kit.tar.gz` archive, bump the VERSION file, commit the version change, and create a GitHub Release with `kit.tar.gz` as an attached asset.

`fab-release.sh` SHALL determine the target GitHub repository by inferring from `git remote get-url origin`. It SHALL NOT hardcode any repository name. This ensures the script works for forks and renamed repos.
<!-- clarified: Repo determined from git remote origin — user selected over hardcoded name and env var approaches -->

`fab-release.sh` SHALL accept an optional argument specifying the bump type: `patch` (default), `minor`, or `major`. If no argument is provided, the script SHALL default to a patch increment.
<!-- clarified: Version bump type is configurable via optional argument — user selected over patch-only and interactive prompt -->

#### Scenario: Create a release (default patch)
- **GIVEN** the user is in the fab-kit repo on the main branch with a clean working tree
- **WHEN** the user runs `bash fab/.kit/scripts/fab-release.sh`
- **THEN** the script SHALL:
  1. Bump the patch version in `fab/.kit/VERSION` (e.g., "0.1.0" → "0.1.1")
  2. Create `kit.tar.gz` containing only the contents of `fab/.kit/`
  3. Commit the VERSION bump
  4. Create a GitHub Release via `gh release create` with the `kit.tar.gz` asset
- **AND** the release tag SHALL match the new VERSION (e.g., `v0.2.0`)

#### Scenario: Create a minor release
- **GIVEN** the user is in the fab-kit repo on the main branch with a clean working tree
- **AND** `fab/.kit/VERSION` contains "0.1.1"
- **WHEN** the user runs `bash fab/.kit/scripts/fab-release.sh minor`
- **THEN** the script SHALL bump the minor version (e.g., "0.1.1" → "0.2.0")
- **AND** create a GitHub Release tagged `v0.2.0`

#### Scenario: Create a major release
- **GIVEN** the user is in the fab-kit repo on the main branch with a clean working tree
- **AND** `fab/.kit/VERSION` contains "0.2.0"
- **WHEN** the user runs `bash fab/.kit/scripts/fab-release.sh major`
- **THEN** the script SHALL bump the major version (e.g., "0.2.0" → "1.0.0")
- **AND** create a GitHub Release tagged `v1.0.0`

#### Scenario: Invalid bump argument
- **GIVEN** the user runs `fab-release.sh foo`
- **WHEN** the argument is not one of `patch`, `minor`, or `major`
- **THEN** the script SHALL exit with an error: "Invalid bump type 'foo'. Use: patch, minor, or major."

#### Scenario: Repo inferred from git remote
- **GIVEN** `git remote get-url origin` returns `git@github.com:wvrdz/fab-kit.git`
- **WHEN** `fab-release.sh` creates a release
- **THEN** the release SHALL be created on the `wvrdz/fab-kit` repository

#### Scenario: No git remote configured
- **GIVEN** the repository has no `origin` remote
- **WHEN** the user runs `fab-release.sh`
- **THEN** the script SHALL exit with an error: "No origin remote found. Set a git remote to use fab-release.sh."

#### Scenario: Release with dirty working tree
- **GIVEN** the user has uncommitted changes
- **WHEN** the user runs `fab-release.sh`
- **THEN** the script SHALL abort with an error: "Working tree not clean. Commit or stash changes first."

### Requirement: Release Archive Contents

`kit.tar.gz` SHALL contain only the `fab/.kit/` directory contents. It MUST NOT include `config.yaml`, `constitution.md`, `docs/`, `specs/`, `changes/`, or any other project-specific files.
<!-- assumed: tar.gz extracts to .kit/ path — ensures tar xz -C fab/ produces fab/.kit/ correctly -->

#### Scenario: Archive contains only .kit/
- **GIVEN** a generated `kit.tar.gz`
- **WHEN** its contents are listed
- **THEN** all paths SHALL be rooted at `.kit/` (e.g., `.kit/VERSION`, `.kit/skills/fab-new.md`)
- **AND** no paths outside `.kit/` SHALL be present

## Distribution: Repo Rename

### Requirement: Repo Rename

The repository SHALL be renamed from `docs-sddr` to `fab-kit` to reflect its role as the canonical source for `fab/.kit/`.
<!-- assumed: Repo rename docs-sddr → fab-kit — user confirmed; GitHub auto-redirects handle existing links -->

#### Scenario: Old URLs redirect
- **GIVEN** the repo has been renamed from `docs-sddr` to `fab-kit`
- **WHEN** a user or script accesses `github.com/wvrdz/docs-sddr`
- **THEN** GitHub SHALL redirect to `github.com/wvrdz/fab-kit`

#### Scenario: Existing clones continue to work
- **GIVEN** a local clone with remote URL `github.com/wvrdz/docs-sddr`
- **WHEN** the user runs `git fetch` or `git pull`
- **THEN** git SHALL follow the redirect and succeed

## Distribution: README

### Requirement: README Update

`README.md` SHALL be updated to document the distribution system, including:
- Bootstrap one-liner with explanation
- Update instructions (`fab-update.sh` usage)
- Release workflow (`fab-release.sh` usage)
- Version checking

#### Scenario: README contains bootstrap instructions
- **GIVEN** the updated README.md
- **WHEN** a new user reads it
- **THEN** they SHALL find the curl one-liner for bootstrapping
- **AND** instructions to run `fab-setup.sh` after extraction
- **AND** instructions to run `/fab-init` for config/constitution generation

## Deprecated Requirements

None.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | gh CLI as primary download tool in fab-update.sh | Proposal specifies gh; standard GitHub CLI widely available; curl fallback deferred |
| 2 | Confident | tar.gz extracts to `.kit/` path prefix | Ensures `tar xz -C fab/` produces correct `fab/.kit/` structure; standard tar convention |
| 3 | Confident | Version display is simple before/after comparison | Proposal says "shows version diff"; semver string comparison is sufficient |
| 4 | Confident | Repo rename from docs-sddr to fab-kit | User confirmed in proposal; GitHub auto-redirects handle existing URLs and clones |

4 assumptions made (4 confident, 0 tentative). Run /fab-clarify to review.

## Clarifications

### Session 2026-02-10

- **Q**: Should fab-update.sh use an atomic update strategy (extract to temp dir, then replace) to prevent corruption if interrupted mid-extraction?
  **A**: Atomic update via temp directory — extract to temp, verify, swap.
- **Q**: Should fab-release.sh accept an argument to control the version bump type (patch/minor/major)?
  **A**: Optional argument `[patch|minor|major]`, defaults to `patch`.
- **Q**: How should fab-release.sh determine the target GitHub repository?
  **A**: Infer from `git remote get-url origin` — zero config, works for forks.
