# Intake: Eliminate Symlinks, Distribute Packages via Kit

**Change**: 260218-cif4-eliminate-symlinks-distribute-packages
**Created**: 2026-02-18
**Status**: Draft

## Origin

> User initiated after investigating the convoluted symlink pattern connecting `src/lib/*/` test directories to `fab/.kit/scripts/lib/` production scripts. With new packages (idea, wt) needing distribution, the current approach doesn't scale. A detailed plan was developed collaboratively, accounting for the bootstrap constraint (fab-kit uses fab to develop itself, so `fab/.kit/` must remain the working runtime and source of truth).

One-shot with extensive codebase exploration. Key decisions:
1. `fab/.kit/` stays as source of truth (bootstrap constraint rules out moving source to `src/`)
2. Symlinks are eliminated by having tests reference production scripts via repo-root-relative paths
3. Package production code moves into `fab/.kit/packages/` so `kit.tar.gz` distributes them automatically
4. A new `env-packages.sh` script handles PATH setup, sourced by both `.envrc` and `rc-init.sh`

## Why

**Problem**: 5 symlinks in `src/lib/*/` point into `fab/.kit/scripts/lib/` so that bats tests can locate their production scripts. Each new testable script requires a new symlink. With packages (`idea`, `wt`) also needing distribution, the pattern compounds — packages currently live entirely in `src/packages/` (not distributed) while kit scripts live in `fab/.kit/` (distributed), creating two separate distribution stories.

**Consequence if not fixed**: Every new distributable component requires a new symlink, and packages remain outside the release tarball. Users can't get `idea` or `wt` via `fab-upgrade.sh`.

**Why this approach**: The bootstrap constraint (fab-kit develops itself using fab) means `fab/.kit/` must be a working runtime, not just a source tree. This rules out the "move source to `src/`" pattern. Instead, we fix the test-side coupling (drop symlinks, use direct paths) and bring packages into the distributable tree.

## What Changes

### 1. Delete 5 symlinks in `src/lib/`

Remove these symlinks (they exist solely so tests can `readlink -f` to find scripts):

```
src/lib/stageman/stageman.sh       → ../../../fab/.kit/scripts/lib/stageman.sh
src/lib/changeman/changeman.sh     → ../../../fab/.kit/scripts/lib/changeman.sh
src/lib/calc-score/calc-score.sh   → ../../../fab/.kit/scripts/lib/calc-score.sh
src/lib/preflight/preflight.sh     → ../../../fab/.kit/scripts/lib/preflight.sh
src/lib/sync-workspace/fab-sync.sh → ../../../fab/.kit/scripts/fab-sync.sh
```

### 2. Update 3 test preambles to use direct paths

Tests for stageman, changeman, and calc-score currently resolve the symlink:

```bash
# Before:
SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
STAGEMAN="$(readlink -f "$SCRIPT_DIR/stageman.sh")"

# After:
REPO_ROOT="$(cd "$(dirname "$BATS_TEST_FILENAME")/../../.." && pwd)"
STAGEMAN="$REPO_ROOT/fab/.kit/scripts/lib/stageman.sh"
```

**Files to change**:
- `src/lib/stageman/test.bats` (lines 7-8: `SCRIPT_DIR` → `REPO_ROOT`, `STAGEMAN` assignment)
- `src/lib/changeman/test.bats` (lines 7-8: same pattern, `CHANGEMAN` variable)
- `src/lib/calc-score/test.bats` (lines 8-9: same pattern, `CALC_SCORE` variable)

**No changes needed**: `src/lib/preflight/test.bats` already uses `PROJECT_ROOT` and copies scripts into a tmpdir. `src/lib/sync-workspace/test.bats` already uses `REPO_SRC_ROOT` with a direct path.

### 3. Move package production code to `fab/.kit/packages/`

Use `git mv` to preserve history:

```
src/packages/idea/bin/idea    →  fab/.kit/packages/idea/bin/idea
src/packages/wt/bin/*         →  fab/.kit/packages/wt/bin/  (wt-create, wt-delete, wt-init, wt-list, wt-open)
src/packages/wt/lib/*         →  fab/.kit/packages/wt/lib/  (wt-common.sh)
```

**What stays in `src/packages/`** (not distributed):
- `src/packages/idea/tests/` — idea test suite
- `src/packages/wt/tests/` — wt test suite (including fixtures)
- `src/packages/tests/libs/` — bats submodules (bats, bats-assert, bats-support, bats-file)
- `src/packages/setup.sh` — dev setup script (initializes submodules)
- `src/packages/rc-init.sh` — updated to delegate to `env-packages.sh`

The `wt` binaries use relative sourcing (`source "$SCRIPT_DIR/../lib/wt-common.sh"`), so `bin/` and `lib/` must remain siblings — moving the whole subtree preserves this.

### 4. Update package test setup to find moved binaries

**`src/packages/idea/tests/setup_suite.bash`** (line 11):
```bash
# Before:
export PATH="${BATS_TEST_DIRNAME}/../bin:$PATH"

# After:
REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
export PATH="$REPO_ROOT/fab/.kit/packages/idea/bin:$PATH"
```

**`src/packages/wt/tests/setup_suite.bash`** — add explicit bin PATH (currently relies on ambient PATH from rc-init.sh):
```bash
# Add after line 11 (fixtures PATH):
REPO_ROOT="$(cd "${BATS_TEST_DIRNAME}/../../.." && pwd)"
export PATH="$REPO_ROOT/fab/.kit/packages/wt/bin:$PATH"
```

Tmpdir paths (`${BATS_TEST_DIRNAME}/../.tmp`) are relative to the tests/ directory which hasn't moved — no change needed.

### 5. New script: `fab/.kit/scripts/env-packages.sh`

Single script that adds all package bins to PATH, sourced by both `.envrc` and `rc-init.sh`:

```bash
#!/usr/bin/env bash
# Add all fab-kit package bin directories to PATH
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KIT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
for d in "$KIT_DIR"/packages/*/bin; do
  [ -d "$d" ] && export PATH="$d:$PATH"
done
```

### 6. Add scaffold envrc entry

**`fab/.kit/scaffold/envrc`** — add one line so user projects get package bins on PATH via direnv:
```
source fab/.kit/scripts/env-packages.sh
```

This line gets merged into projects' `.envrc` by `3-sync-workspace.sh` (line-ensuring merge).

### 7. Update `rc-init.sh` to delegate

**`src/packages/rc-init.sh`** — simplify to source the kit script:
```bash
#!/usr/bin/env sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
source "$SCRIPT_DIR/../../fab/.kit/scripts/env-packages.sh"
```

### 8. Update comments in production scripts

The code works unchanged (`readlink -f "$0"` resolves to itself when there's no symlink), but comments referencing the symlink pattern need updating:

- `fab/.kit/scripts/lib/stageman.sh` line 15: remove "src/lib/stageman/ symlink" mention
- `fab/.kit/scripts/lib/changeman.sh` line 16: remove "symlinks" mention
- `fab/.kit/scripts/fab-upgrade.sh` line 96: "repair symlinks" → "repair directories"
- `fab/.kit/scripts/fab-help.sh` line 138: remove "symlinks" from description

### 9. Update README.md

- Line ~302: package location `src/packages/` → `fab/.kit/packages/`
- Line ~314: update `rc-init.sh` usage/path or note it delegates to `env-packages.sh`
- Line ~324: `setup.sh` path stays (still in `src/packages/`)

### 10. Files requiring no changes

- **`justfile`**: `src/packages/*/tests` glob still matches (tests didn't move)
- **`fab-release.sh`**: `tar czf kit.tar.gz -C fab .kit` now includes packages automatically
- **`fab-upgrade.sh`** (code logic): replaces `fab/.kit/` atomically — packages get updated automatically
- **`.gitignore`**, **`.gitmodules`**: unchanged

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update directory structure, remove symlink references, document `fab/.kit/packages/` convention
- `fab-workflow/distribution`: (modify) Document package distribution via kit tarball, `env-packages.sh` script

## Impact

- **Distribution**: `kit.tar.gz` grows to include `packages/` — `idea` and `wt` become available to all fab users after upgrade
- **User PATH**: Projects running `fab-sync.sh` after this change will get `env-packages.sh` sourced via `.envrc`, adding package bins to PATH
- **Developer workflow**: Test symlinks eliminated — one less thing to manage when adding new testable scripts
- **Existing projects**: The `.envrc` line-merge in `3-sync-workspace.sh` will add the new `source fab/.kit/scripts/env-packages.sh` line on next sync
- **Release size**: `kit.tar.gz` grows by ~65KB (idea binary + wt binaries + wt-common.sh)

## Open Questions

None — the plan was developed through collaborative exploration and addresses all identified concerns.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab/.kit/` remains source of truth | Bootstrap constraint: fab-kit uses fab to develop itself, so the kit directory must be a working runtime | S:95 R:15 A:95 D:95 |
| 2 | Certain | Packages move to `fab/.kit/packages/` | Directly follows from keeping `fab/.kit/` as the distributable unit — packages need to be in the tarball | S:90 R:80 A:90 D:90 |
| 3 | Certain | Tests stay in `src/packages/*/tests/` and `src/lib/*/` | Tests are dev-only artifacts, never distributed — keeping them in `src/` is consistent with current convention | S:90 R:85 A:90 D:95 |
| 4 | Certain | Use `env-packages.sh` sourced from `.envrc` for PATH setup | User requested a script-based approach rather than inline loops in envrc. Self-contained, works with both direnv and shell rc sourcing | S:85 R:90 A:85 D:85 |
| 5 | Confident | wt tests should explicitly add bin/ to PATH in setup_suite | Currently relies on ambient PATH which is fragile. Making it explicit matches the idea test pattern and ensures `just test-packages` works in clean environments | S:70 R:90 A:80 D:85 |
| 6 | Certain | `fab-release.sh` needs no changes | It already packages all of `fab/.kit/` — adding `packages/` underneath automatically includes them | S:95 R:90 A:95 D:95 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
