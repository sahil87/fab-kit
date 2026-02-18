# Spec: Eliminate Symlinks, Distribute Packages via Kit

**Change**: 260218-cif4-eliminate-symlinks-distribute-packages
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Moving production source code out of `fab/.kit/` — the bootstrap constraint requires `fab/.kit/` to be a working runtime
- Changing the `fab-release.sh` packaging logic — `tar czf kit.tar.gz -C fab .kit` already captures everything under `fab/.kit/`
- Modifying test framework or test runner configuration — bats infrastructure stays as-is
- Changing `.gitmodules` or bats submodule locations — they remain in `src/packages/tests/libs/`

## Test Path Resolution: Eliminate Symlinks

### Requirement: Delete test-to-production symlinks

All 5 symlinks in `src/lib/` that point into `fab/.kit/` SHALL be deleted. These symlinks exist solely to let bats tests locate production scripts via `readlink -f`. They are: `src/lib/stageman/stageman.sh`, `src/lib/changeman/changeman.sh`, `src/lib/calc-score/calc-score.sh`, `src/lib/preflight/preflight.sh`, `src/lib/sync-workspace/fab-sync.sh`.

#### Scenario: Symlinks removed from working tree

- **GIVEN** the 5 symlinks exist in `src/lib/*/`
- **WHEN** the change is applied
- **THEN** all 5 symlinks are deleted (via `git rm`)
- **AND** no broken references remain in `src/lib/`

### Requirement: Update test preambles to use repo-root-relative paths

Tests for `stageman`, `changeman`, and `calc-score` SHALL resolve their production script via a repo-root-relative path instead of `readlink -f` on a symlink.

The new preamble pattern SHALL be:

```bash
REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
SCRIPT_VAR="$REPO_ROOT/fab/.kit/scripts/lib/{script}.sh"
```

This replaces:
```bash
SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
SCRIPT_VAR="$(readlink -f "$SCRIPT_DIR/{script}.sh")"
```

#### Scenario: stageman test preamble updated

- **GIVEN** `src/lib/stageman/test.bats` uses `SCRIPT_DIR` and `readlink -f` (lines 7-8)
- **WHEN** the preamble is replaced with the repo-root pattern
- **THEN** `STAGEMAN` points to `$REPO_ROOT/fab/.kit/scripts/lib/stageman.sh`
- **AND** all 53 existing tests pass without modification

#### Scenario: changeman test preamble updated

- **GIVEN** `src/lib/changeman/test.bats` uses `SCRIPT_DIR` and `readlink -f` (lines 7-8)
- **WHEN** the preamble is replaced with the repo-root pattern
- **THEN** `CHANGEMAN` points to `$REPO_ROOT/fab/.kit/scripts/lib/changeman.sh`
- **AND** all existing tests pass without modification

#### Scenario: calc-score test preamble updated

- **GIVEN** `src/lib/calc-score/test.bats` uses `SCRIPT_DIR` and `readlink -f` (lines 8-9)
- **WHEN** the preamble is replaced with the repo-root pattern
- **THEN** `CALC_SCORE` points to `$REPO_ROOT/fab/.kit/scripts/lib/calc-score.sh`
- **AND** all existing tests pass without modification

### Requirement: Preflight and sync-workspace tests require no changes

`src/lib/preflight/test.bats` and `src/lib/sync-workspace/test.bats` SHALL NOT be modified — they already use direct path patterns (`PROJECT_ROOT`/`REPO_SRC_ROOT` and tmpdir copies respectively).

#### Scenario: Preflight tests unaffected

- **GIVEN** `src/lib/preflight/test.bats` uses `PROJECT_ROOT` with direct paths
- **WHEN** the preflight symlink is deleted
- **THEN** all preflight tests continue to pass (they copy scripts into tmpdir, not `readlink`)

#### Scenario: Sync-workspace tests unaffected

- **GIVEN** `src/lib/sync-workspace/test.bats` uses `REPO_SRC_ROOT` with direct paths
- **WHEN** the sync-workspace symlink is deleted
- **THEN** all sync-workspace tests continue to pass

## Package Distribution: Move to Kit

### Requirement: Move package production code to `fab/.kit/packages/`

Package production binaries and libraries SHALL be relocated from `src/packages/` to `fab/.kit/packages/` using `git mv` to preserve history. The directory structure within each package SHALL be preserved (bin/ and lib/ remain siblings).

Files to move:
- `src/packages/idea/bin/idea` → `fab/.kit/packages/idea/bin/idea`
- `src/packages/wt/bin/*` (wt-create, wt-delete, wt-init, wt-list, wt-open) → `fab/.kit/packages/wt/bin/`
- `src/packages/wt/lib/wt-common.sh` → `fab/.kit/packages/wt/lib/wt-common.sh`

#### Scenario: Package binaries distributed via kit tarball

- **GIVEN** package binaries are moved to `fab/.kit/packages/`
- **WHEN** `fab-release.sh` runs `tar czf kit.tar.gz -C fab .kit`
- **THEN** the tarball includes `idea` and `wt` binaries under `.kit/packages/`
- **AND** `fab-upgrade.sh` delivers them to downstream projects

#### Scenario: wt relative sourcing preserved

- **GIVEN** wt binaries use `source "$SCRIPT_DIR/../lib/wt-common.sh"`
- **WHEN** the entire `wt/bin/` and `wt/lib/` subtree is moved as a unit
- **THEN** relative sourcing continues to work (bin/ and lib/ remain siblings)

### Requirement: Test suites stay in `src/packages/`

Test directories, bats submodules, `setup.sh`, and `rc-init.sh` SHALL remain in `src/packages/`. Tests are dev-only artifacts and are not distributed.

Files that stay:
- `src/packages/idea/tests/`
- `src/packages/wt/tests/` (including fixtures)
- `src/packages/tests/libs/` (bats submodules)
- `src/packages/setup.sh`
- `src/packages/rc-init.sh`

#### Scenario: Test glob in justfile still works

- **GIVEN** the justfile uses `src/packages/*/tests` glob pattern
- **WHEN** only production code (bin/, lib/) is moved out of `src/packages/`
- **THEN** the glob still matches `src/packages/idea/tests` and `src/packages/wt/tests`

### Requirement: Update package test setup to find moved binaries

Package test setup files SHALL use repo-root-relative paths to locate binaries in their new `fab/.kit/packages/` location.

#### Scenario: idea test setup updated

- **GIVEN** `src/packages/idea/tests/setup_suite.bash` adds `${BATS_TEST_DIRNAME}/../bin` to PATH (line 11)
- **WHEN** the PATH line is updated to use `$REPO_ROOT/fab/.kit/packages/idea/bin` (REPO_ROOT via `../../../..` — 4 levels up from tests/)
- **THEN** idea tests find the `idea` binary via the new path
- **AND** all idea tests pass

#### Scenario: wt test setup gets explicit bin PATH

- **GIVEN** `src/packages/wt/tests/setup_suite.bash` relies on ambient PATH for wt binaries (no explicit bin addition)
- **WHEN** an explicit `$REPO_ROOT/fab/.kit/packages/wt/bin` PATH entry is added
- **THEN** wt tests find wt binaries regardless of ambient PATH
- **AND** all wt tests pass

## PATH Setup: env-packages.sh

### Requirement: Create env-packages.sh script

A new script `fab/.kit/scripts/env-packages.sh` SHALL be created that adds all package `bin/` directories to PATH. It SHALL iterate `fab/.kit/packages/*/bin` and export each existing directory to PATH.

```bash
#!/usr/bin/env bash
# Add all fab-kit package bin directories to PATH
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
for d in "$KIT_DIR"/packages/*/bin; do
  [ -d "$d" ] && export PATH="$d:$PATH"
done
```

#### Scenario: env-packages.sh adds package bins to PATH

- **GIVEN** `fab/.kit/packages/idea/bin/` and `fab/.kit/packages/wt/bin/` exist
- **WHEN** `env-packages.sh` is sourced
- **THEN** both directories are added to PATH
- **AND** `idea`, `wt-create`, `wt-delete`, `wt-init`, `wt-list`, `wt-open` are executable from PATH

#### Scenario: env-packages.sh handles no packages gracefully

- **GIVEN** no `fab/.kit/packages/*/bin` directories exist
- **WHEN** `env-packages.sh` is sourced
- **THEN** PATH is unchanged and no errors are emitted

### Requirement: Add env-packages.sh to scaffold envrc

`fab/.kit/scaffold/envrc` SHALL include a `source fab/.kit/scripts/env-packages.sh` line. This ensures that projects running `fab-sync.sh` (which line-ensures entries from `scaffold/envrc`) will get package bins on PATH via direnv.

#### Scenario: New projects get package PATH via direnv

- **GIVEN** a project runs `fab-sync.sh` after this change
- **WHEN** `3-sync-workspace.sh` line-ensures entries from `scaffold/envrc`
- **THEN** `.envrc` contains `source fab/.kit/scripts/env-packages.sh`
- **AND** direnv loads package bins into PATH

### Requirement: Update rc-init.sh to delegate

`src/packages/rc-init.sh` SHALL be simplified to delegate to `fab/.kit/scripts/env-packages.sh` instead of iterating package bins itself.

#### Scenario: rc-init.sh delegates to env-packages.sh

- **GIVEN** `src/packages/rc-init.sh` currently iterates `$SCRIPT_DIR/*/bin` to add bins to PATH
- **WHEN** rc-init.sh is replaced with a delegation to `env-packages.sh`
- **THEN** sourcing `rc-init.sh` produces the same PATH additions as before
- **AND** PATH setup logic is centralized in one script

## Comment Updates

### Requirement: Update stale symlink references in comments

Production scripts that reference symlinks in comments SHALL have those comments updated. The code itself requires no changes — `readlink -f "$0"` resolves to the script itself when there is no symlink.

Changes:
- `fab/.kit/scripts/lib/stageman.sh` line 15: remove "src/lib/stageman/ symlink" mention
- `fab/.kit/scripts/lib/changeman.sh` line 16: remove "symlinks" mention
- `fab/.kit/scripts/fab-upgrade.sh` line 96: "repair symlinks" → "repair directories"
- `fab/.kit/scripts/fab-help.sh` line 138: remove "symlinks" from description

#### Scenario: No stale symlink references in comments

- **GIVEN** the symlinks are deleted
- **WHEN** the change is complete
- **THEN** no comment in `fab/.kit/scripts/` references the deleted `src/lib/` symlink pattern

## README Updates

### Requirement: Update README package references

`README.md` SHALL be updated to reflect the new package location and PATH setup mechanism.

- Package location reference: `src/packages/` → `fab/.kit/packages/` (for production code)
- `rc-init.sh` documentation: note delegation to `env-packages.sh`
- `setup.sh` path: unchanged (still in `src/packages/`)

#### Scenario: README reflects new package location

- **GIVEN** README.md references package binaries under `src/packages/`
- **WHEN** the location references are updated
- **THEN** README accurately describes `fab/.kit/packages/` as the location for distributed package code
- **AND** `src/packages/` is described as containing tests, setup, and dev tooling only

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab/.kit/` remains source of truth | Confirmed from intake #1 — bootstrap constraint: fab-kit uses fab to develop itself | S:95 R:15 A:95 D:95 |
| 2 | Certain | Packages move to `fab/.kit/packages/` | Confirmed from intake #2 — follows from keeping `fab/.kit/` as the distributable unit | S:90 R:80 A:90 D:90 |
| 3 | Certain | Tests stay in `src/packages/*/tests/` and `src/lib/*/` | Confirmed from intake #3 — tests are dev-only, never distributed | S:90 R:85 A:90 D:95 |
| 4 | Certain | Use `env-packages.sh` for PATH setup | Confirmed from intake #4 — self-contained, works with both direnv and rc-init.sh | S:85 R:90 A:85 D:85 |
| 5 | Confident | wt tests should explicitly add bin/ to PATH in setup_suite | Confirmed from intake #5 — ambient PATH is fragile; verified wt setup_suite has no explicit bin PATH | S:70 R:90 A:80 D:85 |
| 6 | Certain | `fab-release.sh` needs no changes | Confirmed from intake #6 — already packages all of `fab/.kit/` | S:95 R:90 A:95 D:95 |
| 7 | Certain | Use `git mv` for history preservation | Standard git practice for file moves; no alternative approach reasonable | S:90 R:85 A:95 D:95 |

7 assumptions (6 certain, 1 confident, 0 tentative, 0 unresolved).
