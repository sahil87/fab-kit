# Intake: Port wt Tests & Cleanup Legacy

**Change**: 260310-8m3k-port-wt-tests-cleanup-legacy
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Check src/packages/wt/ — these are test cases written for the older shell-based implementation of wt. Port them to the new Go implementation in src/go/wt/. Then check if src/tests, src/packages/ are still needed. Also check if .gitmodules is still needed.

## Why

The project recently extracted `wt` from a shell-based implementation (`src/packages/wt/`) into a standalone Go binary (`src/go/wt/`). The shell-based tests in `src/packages/wt/tests/` (13 bats test files covering init, create, delete, list, open, pr, edge cases, integration) encode valuable behavioral expectations that should carry forward. Without porting these, the Go implementation lacks equivalent coverage for the behaviors the bats tests verified.

Additionally, the shell infrastructure that supported the old implementation — `src/packages/` (rc-init.sh, the wt shell package), `src/tests/` (empty except for bats libs), and `.gitmodules` (bats submodule references) — is now dead weight. Removing them reduces repo clutter and eliminates the bats submodule dependency entirely.

## What Changes

### 1. Port wt bats tests to Go tests

Review each bats test file in `src/packages/wt/tests/` and port the behavioral expectations to Go tests in `src/go/wt/`:

| Bats file | Scope |
|-----------|-------|
| `wt-init.bats` | `wt init` — directory setup, config creation |
| `wt-create.bats` | `wt create` — worktree creation, branch naming, validation |
| `wt-delete.bats` | `wt delete` — worktree removal, cleanup |
| `wt-list.bats` | `wt list` — output formatting, filtering |
| `wt-open.bats` | `wt open` — worktree switching/opening |
| `wt-pr.bats` | `wt pr` — PR creation integration |
| `wt-common.bats` | Shared utility behavior |
| `edge-cases.bats` | Error handling, boundary conditions |
| `integration.bats` | End-to-end workflows |

Existing Go tests (`errors_test.go`, `names_test.go`, `rollback_test.go`, `context_test.go`, `git_test.go`) cover internal packages. The ported tests should target the `cmd/` layer or add integration-level coverage as appropriate.

### 2. Remove `src/packages/`

The directory contains:
- `rc-init.sh` — shell env setup that delegates to `fab/.kit/scripts/lib/env-packages.sh` (no longer needed since wt is now a Go binary)
- `wt/` — old shell wt package with `.tmp/` (test artifacts) and `tests/` (the bats tests being ported)
- `tests/` — a `libs/` directory containing only the bats git submodules

All of this is legacy shell infrastructure superseded by the Go rewrite.

### 3. Remove `src/tests/`

Contains only `libs/` which holds the bats submodule checkouts. No longer needed.

### 4. Remove `.gitmodules`

Currently references 4 bats-related submodules (`bats-core`, `bats-support`, `bats-assert`, `bats-file`). All live under `src/packages/tests/libs/`. Once `src/packages/` is removed, `.gitmodules` has no remaining entries and should be deleted.
<!-- assumed: No other submodules exist — .gitmodules contains only bats entries -->

### 5. Clean up any remaining references

Check for references to `src/packages/`, `src/tests/`, or bats in CI configs, scripts, or documentation that need updating.

## Affected Memory

- `fab-workflow/distribution`: (modify) May reference src/packages in distribution notes

## Impact

- **Test coverage**: Behavioral expectations move from bats to Go — net positive for maintainability
- **Repo structure**: `src/packages/`, `src/tests/`, `.gitmodules` removed — cleaner layout
- **Git submodules**: Eliminated entirely — simpler cloning and CI
- **Developer workflow**: No more bats dependency for running wt tests; `go test ./...` covers everything

## Open Questions

- Are there any CI pipelines that invoke the bats tests directly?
- Does `wt pr` have a Go implementation yet? (only `init`, `create`, `delete`, `list`, `open` seen in `src/go/wt/cmd/`)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Port tests to Go test files in src/go/wt/ | Go module already has test infrastructure; natural target | S:90 R:90 A:95 D:95 |
| 2 | Certain | Remove src/packages/ entirely | All contents are legacy shell infrastructure superseded by Go rewrite | S:85 R:85 A:90 D:95 |
| 3 | Certain | Remove src/tests/ entirely | Contains only bats submodule libs, no longer needed | S:85 R:85 A:90 D:95 |
| 4 | Certain | Remove .gitmodules | Only contains bats submodule refs; confirmed by reading the file | S:95 R:85 A:95 D:95 |
| 5 | Confident | Tests target cmd/ layer or integration level | Existing Go tests cover internal/; bats tests are behavioral/CLI-level | S:75 R:85 A:80 D:75 |
| 6 | Confident | No CI references to bats tests | No CI config files observed, but not exhaustively searched | S:60 R:75 A:70 D:80 |
| 7 | Certain | wt pr not yet implemented in Go | Only init, create, delete, list, open found in src/go/wt/cmd/ | S:90 R:90 A:90 D:95 |
| 8 | Certain | .gitmodules has no non-bats entries | File contents confirmed — only 4 bats submodule entries | S:95 R:90 A:95 D:95 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
