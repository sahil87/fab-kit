# Intake: Extract Idea into Standalone Go Binary

**Change**: 260310-y8o6-extract-idea-binary
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Extract `idea` from a `fab` subcommand into its own standalone Go binary, reorganize the justfile so `build-go` and `test-go` handle all 3 Go binaries (fab, wt, idea), separate Rust recipes from aggregate targets, and update the release workflow.

Initiated via `/fab-discuss` conversation. User identified that `idea` should be a standalone binary like `wt`, not a subcommand of `fab`. Discussion also covered justfile reorganization to unify Go build/test recipes and decouple Rust from aggregate targets.

## Why

The `idea` subcommand was embedded inside the `fab` binary as `fab idea ...`. This created an implicit dependency — users who only want backlog management must have the full `fab` binary. Extracting `idea` into its own binary (`idea add`, `idea list`, etc.) matches the pattern already established by `wt` and enables independent distribution. The justfile had separate, duplicated recipes for `fab` and `wt` builds, and Rust was mixed into aggregate targets (`test`, `build-all`) despite being an optional/separate concern.

## What Changes

### Standalone `idea` binary

- Create `src/fab-go/cmd/idea/main.go` — cobra root command with all 7 subcommands (add, list, show, done, reopen, edit, rm)
- Move cobra CLI wiring from `src/fab-go/cmd/fab/idea.go` into the new entry point
- Remove `ideaCmd()` registration from `src/fab-go/cmd/fab/main.go`
- Delete `src/fab-go/cmd/fab/idea.go`
- Business logic in `internal/idea/` is unchanged — only the CLI layer moves

### Justfile reorganization

- `test` recipe: remove `test-rust` (Rust tests run standalone via `just test-rust`)
- `build-go`: build all 3 Go binaries (fab → `fab/.kit/bin/fab-go`, wt → `fab/.kit/bin/wt`, idea → `fab/.kit/bin/idea`)
- New `_build-go-binary` helper recipe for DRY cross-compilation
- `build-go-target os arch`: cross-compile all 3 binaries for a single platform
- `build-go-all`: cross-compile all 3 for all 4 release targets
- `build-all`: only calls `build-go-all` (no `build-rust-all`)
- Remove separate `build-wt`, `build-wt-target`, `build-wt-all` recipes
- `package-kit`: verify and stage all 3 Go binaries, exclude `idea` from generic archive

### Release workflow

- `.github/workflows/release.yml`: change build step from `just build-go-all` to `just build-all`

## Affected Memory

- `fab-workflow/distribution`: (modify) Update to reflect 3 Go binaries in release packaging

## Impact

- `src/fab-go/cmd/fab/` — idea subcommand removed
- `src/fab-go/cmd/idea/` — new binary entry point
- `justfile` — full rewrite of build/test section
- `.github/workflows/release.yml` — build step updated
- `fab/.kit/bin/` — `idea` binary added alongside `fab-go` and `wt`

## Open Questions

None — all decisions were resolved during `/fab-discuss`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | idea becomes standalone binary, not fab subcommand | Discussed — user explicitly chose clean break, no alias | S:95 R:80 A:90 D:95 |
| 2 | Certain | Rust recipes separated from test/build-all aggregates | Discussed — user confirmed Rust should be standalone | S:90 R:85 A:85 D:90 |
| 3 | Certain | build-go/test-go handle all 3 Go binaries uniformly | Discussed — user confirmed unified Go recipes | S:90 R:80 A:85 D:90 |
| 4 | Certain | package-kit includes idea alongside fab-go and wt | Discussed — user confirmed idea should be packaged | S:85 R:75 A:80 D:90 |
| 5 | Certain | No `fab idea` alias support | Discussed — user said no alias needed | S:95 R:90 A:90 D:95 |
| 6 | Confident | internal/idea/ business logic unchanged | Clean separation — only CLI wiring moves | S:80 R:90 A:85 D:85 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
