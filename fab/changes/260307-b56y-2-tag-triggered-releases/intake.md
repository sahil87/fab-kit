# Intake: Tag-Triggered Releases

**Change**: 260307-b56y-2-tag-triggered-releases
**Created**: 2026-03-07
**Status**: Draft

## Origin

> Slim `fab-release.sh` to just version bump + commit + tag + push. The GitHub Actions workflow (created in change 1) handles building, packaging, and creating the GitHub Release. This is step 2 of the 4-part CI migration plan.

Discussion context: User asked "What would be the build trigger step then? A tag? And fab-release.sh changing to just increment tag to trigger build?" — confirmed tag push as the trigger, with `fab-release.sh` becoming a thin local script.

Depends on: `260307-ma7o-1-ci-releases-justfile` (justfile and CI workflow must exist first).

## Why

After change 1 extracts build recipes into a `justfile` and creates the CI workflow, `fab-release.sh` still contains the `gh release create` call and archive upload logic. This change completes the separation: the local script handles *what version* to release (human decision), CI handles *how* to build and publish (automation).

This decoupling means:
1. Releases are reproducible — the CI workflow is the single source of truth for what gets published
2. No local toolchain requirements beyond `git` and basic shell — developers don't need Go or `just` installed to cut a release
3. Failed releases can be retried by re-pushing the tag (or deleting and re-creating it), without re-running local scripts

## What Changes

### Modified: `src/scripts/fab-release.sh`

The script shrinks to its core responsibility — version management and tagging:

1. Parse bump type argument (`patch`, `minor`, `major`)
2. Read current version from `fab/.kit/VERSION`
3. Compute new version
4. Run migration chain validation (unchanged — this is a pre-release safety check)
5. Write new version to `fab/.kit/VERSION`
6. `git add fab/.kit/VERSION && git commit -m "release: v{version}"`
7. `git tag v{version}`
8. `git push origin HEAD:{branch} v{version}`
9. Print summary: "Tagged v{version} — CI will create the release."

Removed:
- `command -v go` check (not needed locally anymore)
- `command -v gh` check (CI handles `gh release create`)
- Cross-compilation loop (moved to justfile in change 1)
- Archive packaging (moved to justfile in change 1)
- `gh release create` call (moved to CI in change 1)
- `.release-build/` directory management
- Archive file cleanup
- `--no-latest` flag handling (moved to CI workflow — can be a workflow input or separate tag pattern)

The `--no-latest` flag needs a new mechanism. Options:
- CI workflow checks if the tag matches a pattern (e.g., tags on non-main branches get `--latest=false`)
- CI workflow has a manual dispatch input for `--no-latest`
- Convention: tags pushed from `release/*` branches automatically get `--latest=false`

### Modified: `.github/workflows/release.yml` (from change 1)

Add the `--no-latest` logic. The workflow detects whether the tag was pushed from main:
- If the push ref is on main: `--latest` (default)
- If on a release branch: `--latest=false`

Also add release notes generation from the git log (same logic currently in `fab-release.sh`).

## Affected Memory

- `fab-workflow/distribution`: (modify) Update release section — `fab-release.sh` no longer creates GitHub Releases directly; it bumps version, commits, tags, and pushes. CI handles build/package/release. Document the `--no-latest` mechanism for backport releases.

## Impact

- **`src/scripts/fab-release.sh`**: Reduces from ~275 lines to ~80 lines
- **`.github/workflows/release.yml`**: Gains release notes generation and `--no-latest` logic
- **Developer workflow**: `fab-release.sh patch` → wait for CI → release appears on GitHub. No local Go/just/gh needed.
- **Backport workflow**: Needs a new mechanism for `--no-latest` (see Open Questions)

## Open Questions

- How should backport releases (`--no-latest`) be signaled to CI? Branch-based detection (release/* branches) seems cleanest.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tag push triggers CI build and release | Discussed — user confirmed tag-based trigger | S:95 R:80 A:90 D:95 |
| 2 | Certain | fab-release.sh retains version bump + migration validation + commit + tag + push | Discussed — these are the local-only responsibilities | S:90 R:85 A:90 D:90 |
| 3 | Certain | fab-release.sh removes all build/package/release logic | Discussed — CI takes over these responsibilities | S:90 R:75 A:85 D:90 |
| 4 | Confident | Release notes generated from git log in CI (same logic as current fab-release.sh) | Existing pattern, just moved to CI context | S:75 R:85 A:80 D:80 |
| 5 | Confident | Backport `--no-latest` detection via branch name (release/* → --latest=false) | Convention-based, clean, no extra flags needed. Other options exist | S:60 R:80 A:70 D:60 |
| 6 | Confident | No local Go/just/gh required to cut a release after this change | Only git needed — CI has all toolchains | S:80 R:85 A:85 D:85 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
