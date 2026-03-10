# Intake: Split Go Modules

**Change**: 260310-1rjf-split-go-modules
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Split the monolithic `src/go/fab/` into three independent Go modules (`src/go/fab/`, `src/go/wt/`, `src/go/idea/`) with zero shared code between them.

Initiated via `/fab-discuss` conversation. User identified that the single Go module containing all 3 binaries was confusing — you can't tell which binary owns which `internal/` package, tests are scattered inconsistently, and the dependency graph between `wt` and `fab` internal packages was unnecessarily coupled. Discussion explored three proposals (group internal by binary, separate modules, minimal fix). User chose separate modules (Proposal B).

Key design decision during discussion: `wt list` should NOT show fab-specific state (change, stage). Instead, `fab status show` should call the `wt` binary via `wt list --json` and resolve fab state itself. This eliminates the coupling where `internal/worktree` imported `internal/status` and `internal/statusfile`.

## Why

The current `src/go/fab/` module contains three independent binaries (fab, wt, idea) in a single Go module. This creates several problems:

1. **Ownership ambiguity** — `internal/` has 14 flat packages. You can't tell from the directory tree which binary owns which package (e.g., does `internal/hooklib` belong to `fab` or `wt`?).
2. **Unnecessary coupling** — `internal/worktree` imports `internal/status` and `internal/statusfile` solely to display fab pipeline state in `wt list`. This pulls in a transitive chain (status → config, hooks, log, resolve) that has nothing to do with worktree management.
3. **Inconsistent test placement** — `fab` has cmd-level tests, `wt` and `idea` have none. Tests for a binary are split between `cmd/` and `internal/` with no clear convention.
4. **Build coupling** — all 3 binaries share one `go.mod`/`go.sum`, so dependency changes to one affect all.

Splitting into independent modules makes ownership unambiguous, eliminates coupling, and lets each binary evolve independently.

## What Changes

### Decouple wt from fab internals

Remove fab-specific state from the worktree package:

- Remove `internal/status` and `internal/statusfile` imports from `internal/worktree/worktree.go`
- Remove `Change`, `Stage`, `State` fields from the `Info` struct
- Delete `resolveFabState()` function
- Update `FormatHuman`/`FormatAllHuman` to show name + branch only (no fab state)

Make `fab status show` call the `wt` binary instead of importing the worktree package:

- Find `wt` binary in the same directory as the running `fab` binary via `os.Executable()`
- Call `wt list --json`, parse the JSON output into a local struct
- Resolve fab state locally (read `.status.yaml` via existing `status`/`statusfile` packages)
- Format output with fab-specific columns (change, stage, state)
- Handle `wt` binary not found gracefully

### Create `src/go/wt/` module

```
src/go/wt/
  go.mod                    module github.com/wvrdz/fab-kit/src/go/wt
  cmd/
    main.go, create.go, delete.go, init.go, list.go, open.go
  internal/
    worktree/
      apps.go, context.go, context_test.go, crud.go, errors.go, errors_test.go,
      git.go, git_test.go, menu.go, names.go, names_test.go, platform.go,
      rollback.go, rollback_test.go, stash.go, worktree.go
```

All import paths updated from `github.com/wvrdz/fab-kit/src/go/fab/internal/worktree` to `github.com/wvrdz/fab-kit/src/go/wt/internal/worktree`.

### Create `src/go/idea/` module

```
src/go/idea/
  go.mod                    module github.com/wvrdz/fab-kit/src/go/idea
  cmd/
    main.go, resolve.go, add.go, list.go, show.go, done.go, reopen.go, edit.go, rm.go
  internal/
    idea/
      idea.go, idea_test.go
```

Split the monolithic `cmd/idea/main.go` (305 lines) into per-subcommand files matching the pattern used by `cmd/wt/` and `cmd/fab/`. All import paths updated to `github.com/wvrdz/fab-kit/src/go/idea/internal/idea`.

### Clean up `src/go/fab/`

- Delete `cmd/wt/` (moved to `wt-go`)
- Delete `cmd/idea/` (moved to `idea-go`)
- Delete `internal/worktree/` (moved to `wt-go`)
- Delete `internal/idea/` (moved to `idea-go`)

### Delete old shell packages

- Delete `src/packages/idea/` (old bats tests — Go coverage is sufficient)
- Delete `fab/.kit/packages/idea/` (old shell script — replaced by Go binary)

### Update justfile

- `test-go`: run `go test` in all 3 module directories
- `build-go`: build from each module's own path (`src/go/fab`, `src/go/wt`, `src/go/idea`)
- `_build-go-binary`: accept `src_dir` and `cmd_path` parameters
- `build-go-target`: cross-compile all 3 using per-module paths
- Remove `go_src` variable (no longer a single source directory)

### Update `_scripts.md`

- Remove `fab idea` section — `idea` is now a standalone binary, not a fab subcommand
- Note that `fab status show` calls the `wt` binary

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document 3-module Go structure
- `fab-workflow/distribution`: (modify) Update binary paths for wt-go and idea-go modules

## Impact

- `src/go/fab/` — significant cleanup (remove wt, idea, worktree code)
- `src/go/wt/` — new Go module (moved from fab-go)
- `src/go/idea/` — new Go module (moved from fab-go)
- `justfile` — build/test paths updated for 3 modules
- `fab/.kit/skills/_scripts.md` — remove `fab idea` documentation
- `src/packages/idea/` — deleted (old shell tests)
- `fab/.kit/packages/idea/` — deleted (old shell script)

## Open Questions

None — all decisions were resolved during `/fab-discuss`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 3 separate Go modules, not grouped internal or shared module | Discussed — user chose Proposal B after seeing all 3 options | S:95 R:70 A:90 D:95 |
| 2 | Certain | wt list drops fab-specific state (change, stage, state) | Discussed — user said "let wt-list not show fab specific things" | S:95 R:75 A:90 D:95 |
| 3 | Certain | fab status show calls wt binary via JSON, not import | Discussed — user approved calling binary over inlining or importing | S:90 R:80 A:85 D:90 |
| 4 | Certain | Delete old shell idea package and tests | Discussed — user approved after seeing Go test coverage audit (42/55 covered, missing 13 are CLI-layer) | S:90 R:75 A:85 D:90 |
| 5 | Certain | Keep old shell wt bats tests (separate change later) | Discussed — user said "we will create a new change for this later" | S:95 R:85 A:90 D:95 |
| 6 | Certain | Split cmd/idea/main.go into per-subcommand files | Discussed — matches cmd/wt/ and cmd/fab/ patterns | S:85 R:90 A:85 D:90 |
| 7 | Confident | Each module gets its own go.mod with cobra dependency | Standard Go practice — cobra is the only shared dep, trivial to duplicate | S:75 R:90 A:90 D:85 |
| 8 | Confident | No shared code between modules after decoupling | Dependency analysis confirmed: idea has zero cross-deps, wt→fab coupling is only 2 function calls (resolveFabState) which get removed | S:80 R:80 A:85 D:85 |

8 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
