# Intake: CI Releases with Justfile

**Change**: 260307-ma7o-1-ci-releases-justfile
**Created**: 2026-03-07
**Status**: Draft

## Origin

> Move Go cross-compilation and release packaging from `fab-release.sh` into a `justfile` with locally-replicable recipes. This is step 1 of a 4-part plan to move to CI-based releases and eventually add a Rust backend. The key principle: CI should use commands that are completely replicable locally.

Discussion context: The user explored switching from Go to Rust for the fab binary. The first step agreed upon was extracting build logic into a `justfile` so that local and CI builds use identical commands. Cross-compilation will use a single Linux runner (Go's `CGO_ENABLED=0 GOOS/GOARCH` makes this trivial).

## Why

The current `fab-release.sh` (~275 lines) conflates three concerns: version bumping, cross-compilation, and GitHub Release creation. This makes it impossible to run just the build step locally for testing, and CI can't reuse the same commands. Extracting build recipes into a `justfile` enables:

1. Local developers to run `just build-go` to build for their platform or `just build-go-all` to replicate the full CI matrix
2. CI workflows to call the exact same `just` recipes — no drift between local and CI behavior
3. Clean separation: `justfile` owns build/package, `fab-release.sh` owns version/tag/push, GitHub Actions owns orchestration

If we don't do this, adding Rust as a second backend (steps 3-4) would mean duplicating cross-compilation logic in both the release script and CI, with no local replication path.

## What Changes

### New: `justfile` at repo root

A `justfile` (consumed by [just](https://github.com/casey/just)) with these recipes:

```just
# Build Go binary for the current platform (local dev)
build-go:
    CGO_ENABLED=0 go build -C src/go/fab -o ../../fab/.kit/bin/fab-go ./cmd/fab

# Cross-compile Go binary for a specific target
build-go-target os arch:
    CGO_ENABLED=0 GOOS={{os}} GOARCH={{arch}} go build -C src/go/fab -o .release-build/fab-{{os}}-{{arch}} ./cmd/fab

# Cross-compile Go binary for all release targets
build-go-all:
    just build-go-target darwin arm64
    just build-go-target darwin amd64
    just build-go-target linux arm64
    just build-go-target linux amd64

# Package kit archives for release (generic + per-platform)
package-kit:
    # Creates kit.tar.gz (no binary) + kit-{os}-{arch}.tar.gz (with binary) for each platform
    # Logic extracted from fab-release.sh lines 182-206
```

The `package-kit` recipe extracts the tar packaging logic currently in `fab-release.sh` (creating staging directories, copying `.kit/`, placing the binary, tar'ing). It produces 5 archives in the repo root: `kit.tar.gz` + 4 platform-specific `kit-{os}-{arch}.tar.gz`.

### New: `.github/workflows/release.yml`

GitHub Actions workflow triggered on `v*` tag push:

```yaml
on:
  push:
    tags: ['v*']
```

Single Linux runner (`ubuntu-latest`). Steps:
1. Checkout code
2. Install Go toolchain
3. Install `just`
4. Run `just build-go-all` (cross-compiles all 4 targets)
5. Run `just package-kit` (creates 5 archives)
6. Create GitHub Release via `gh release create` with all 5 archives, auto-generated changelog

The workflow uses the same `just` commands a developer would run locally. No CI-only build logic.

### Modified: `src/scripts/fab-release.sh`

Shrinks to ~60 lines. Retains:
- Version bump logic (read VERSION, compute new version, write)
- Migration chain validation (pre-release check)
- Git commit + tag + push

Removes:
- Go toolchain check (`command -v go`)
- Cross-compilation loop (moved to `justfile`)
- Archive packaging (moved to `justfile`)
- `gh release create` (moved to GitHub Actions)
- `.release-build/` cleanup (moved to `justfile` / CI)

The new flow: `fab-release.sh patch` → bumps VERSION → commits → tags → pushes → CI takes over.

### New: `.github/workflows/` directory

This is the first GitHub Actions workflow in the project. The directory needs to be created.

## Affected Memory

- `fab-workflow/distribution`: (modify) Update release section to document justfile recipes, CI workflow, and the new split between fab-release.sh (version/tag) and CI (build/package/release)

## Impact

- **`src/scripts/fab-release.sh`**: Major refactor — remove ~200 lines of build/package/release logic
- **`fab/.kit/`**: No changes — the kit directory and its contents are unaffected
- **Build toolchain**: Adds `just` as a dev dependency (single binary, no runtime). Go toolchain still required for building from source
- **`fab-upgrade.sh`**: Unaffected — still downloads from GitHub Releases as before
- **Bootstrap one-liner**: Unaffected — still downloads platform archives from GitHub Releases

## Open Questions

- Should `just` be added to the prerequisites check in `fab-sync.sh`? Probably not — it's a dev-only tool, not needed by end users of fab-kit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `just` as task runner | Discussed — user explicitly chose `just` for locally-replicable CI commands | S:95 R:85 A:90 D:95 |
| 2 | Certain | Single Linux runner for Go cross-compilation | Discussed — Go's CGO_ENABLED=0 produces identical binaries regardless of host OS | S:90 R:80 A:95 D:90 |
| 3 | Certain | Same 4 platform targets (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64) | Existing targets from current fab-release.sh, no change requested | S:95 R:90 A:95 D:95 |
| 4 | Certain | Tag push (`v*`) triggers CI release | Discussed — user asked about tag-based triggers | S:90 R:75 A:85 D:90 |
| 5 | Confident | `just` not added to fab-sync.sh prerequisites | Dev-only tool, not needed by end users. End users download pre-built archives | S:70 R:90 A:80 D:85 |
| 6 | Confident | `package-kit` recipe handles staging dir creation and cleanup | Extracted from existing fab-release.sh logic, same behavior | S:75 R:85 A:85 D:80 |
| 7 | Confident | CI workflow uses `gh release create` with auto-generated changelog | Existing pattern from fab-release.sh, moved to CI context | S:75 R:80 A:80 D:75 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
