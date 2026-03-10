# Intake: Go wt Binary

**Change**: 260310-qbiq-go-wt-binary
**Created**: 2026-03-10
**Status**: Draft

## Origin

> Consolidate wt-* shell scripts into a single `wt` Go binary. The wt binary lives in the same Go module as fab (`src/go/fab/`) but builds a separate binary. Uses cobra subcommands (wt create, wt list, wt open, wt delete, wt init) mirroring the current wt-* script functionality. `wt pr` excluded (overlaps with `/git-pr`). Separate binary from fab — wt should work in any git repo, not just fab-initialized projects.

Initiated via `/fab-discuss` conversation. The user chose Go to match the existing `fab-go` binary and leverage the shared `internal/` packages. Chose **option B** (separate binary) over merging wt into fab as a subcommand.

This change supersedes the earlier Go-based wt binary intakes (`260305-jug9-3-build-wt-go-binary`, `260305-k8ds-4-ship-wt-go-binary`) and backlog items `wt01`–`wt10`.

## Why

1. **Single toolchain**: The `fab` binary is already Go (`src/go/fab/`, cobra + yaml.v3). The wt binary lives in the same module, shares `go.mod`, and reuses existing `internal/` packages.

2. **Shared code**: `wt` needs repo root detection, worktree path conventions, and config reading — already implemented in `src/go/fab/internal/`. No reimplementation needed.

3. **Code quality**: The 6 wt-* shell scripts total ~2,800 lines plus a 578-line shared library (`wt-common.sh`). Go provides proper argument parsing (cobra), testability, structured error handling, and eliminates repeated `source wt-common.sh` parse overhead.

4. **Distribution simplicity**: Constitution mandates single-binary utilities. Both `fab` and `wt` binaries build from the same Go module and ship together in per-platform archives.

## What Changes

### Go Module Structure

The wt binary lives in the same Go module at `src/go/fab/` as a separate `cmd/` entry:

```
src/go/fab/
├── cmd/
│   ├── fab/
│   │   └── main.go          # fab binary (existing)
│   └── wt/
│       └── main.go          # wt binary (new)
├── internal/
│   ├── ... (existing packages)
│   └── worktree/            # wt shared library (new)
│       ├── worktree.go      # git worktree operations
│       ├── names.go         # memorable name generation
│       ├── stash.go         # stash/rollback logic
│       ├── menu.go          # interactive TUI (menus, fzf)
│       └── worktree_test.go
```

### Port: wt-common.sh → `internal/worktree/`

The 578-line shared library maps to:

| Bash function group | Go location |
|---------------------|-------------|
| Repo detection (`wt_get_repo_context`, `wt_validate_git_repo`) | `internal/worktree/` (or reuse existing resolve) |
| Random name generation (`wt_generate_random_name`, adjective/noun lists) | `internal/worktree/names.go` |
| Rollback stack (`wt_register_rollback`, `wt_rollback`) | `internal/worktree/rollback.go` |
| Stash operations (`wt_stash_create`, `wt_stash_apply`) | `internal/worktree/stash.go` |
| Branch validation/detection | `internal/worktree/git.go` |
| Menu helper (`wt_show_menu`) | `internal/worktree/menu.go` |
| Change detection (`wt_has_uncommitted_changes`, etc.) | `internal/worktree/git.go` |
| OS detection (`wt_detect_os`, `wt_is_tmux_session`) | `internal/worktree/platform.go` |
| Worktree CRUD (`wt_create_worktree`, `wt_list_worktrees`) | `internal/worktree/worktree.go` |

### Port: 5 wt-* commands → cobra subcommands

> **Excluded**: `wt pr` (wt-pr, 307 lines) — overlaps with `/git-pr` territory. Not ported.

```
wt create    # wt-create (328 lines)
wt list      # wt-list (264 lines)
wt open      # wt-open (538 lines)
wt delete    # wt-delete (680 lines)
wt init      # wt-init (105 lines)
```

Each subcommand preserves the current script's behavior:
- Same argument semantics and flags
- Same interactive flows (menus, confirmations, fzf integration)
- Same output format (for script compatibility)
- Same exit codes (WT_EXIT_SUCCESS=0, WT_EXIT_GENERAL_ERROR=1, etc.)

### Build & Release Integration

- `justfile` targets for building both binaries (`build-go` updated, new `build-wt`)
- Cross-compile both binaries in `fab-release.sh`:

```bash
for pair in "darwin/arm64" "darwin/amd64" "linux/arm64" "linux/amd64"; do
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go build -o "../../fab/.kit/bin/fab" ./cmd/fab
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go build -o "../../fab/.kit/bin/wt" ./cmd/wt
  # Package archive with both binaries
done
```

- Both binaries placed in `fab/.kit/bin/` and included in each per-platform archive

### PATH and env-packages.sh Update

Update `fab/.kit/scripts/lib/env-packages.sh` to add `fab/.kit/bin/` to PATH (for both `fab` and `wt` binaries), in addition to the existing `fab/.kit/packages/*/bin` entries.

### Remove Legacy wt Shell Scripts

Direct cutover — no shim layer. Remove all bash wt-* scripts:

- Delete `fab/.kit/packages/wt/bin/wt-create`, `wt-delete`, `wt-init`, `wt-list`, `wt-open`, `wt-pr`
- Delete `fab/.kit/packages/wt/lib/wt-common.sh`
- Remove the `fab/.kit/packages/wt/` directory entirely
- Update `env-packages.sh` PATH entries to remove `fab/.kit/packages/wt/bin`

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document wt Go binary alongside fab binary
- `fab-workflow/distribution`: (modify) Document wt binary in per-platform archives, env-packages.sh update

## Impact

- **Source**: New `src/go/fab/cmd/wt/` and `src/go/fab/internal/worktree/` (~1,200-1,700 lines estimated, wt pr excluded)
- **Test coverage**: Same or greater coverage than the shell scripts being replaced
- **Build**: `justfile`, `fab-release.sh` — updated for dual binary builds and cross-compilation
- **Release pipeline**: Both `fab` and `wt` binaries in each per-platform archive
- **env-packages.sh**: Adds `fab/.kit/bin/` to PATH
- **Existing fab binary**: Unchanged — wt is a separate cmd/ entry
- **Existing wt scripts**: All removed (including `wt-pr` — not ported, not kept)
- **Batch scripts**: `batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh` — may reference wt-* directly, need updating

## Open Questions

- None.

## Clarifications

### Session 2026-03-10 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Changed | "Exclude wt pr from scope — overlaps with /git-pr" |
| 5 | Confirmed | — |
| 6 | Changed | "No shim layer — direct cutover" |
| 8 | Confirmed | Added test coverage requirement |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go, same module as fab (`src/go/fab/`) | Discussed — shared internal packages, single go.mod, consistent toolchain | S:90 R:85 A:90 D:90 |
| 2 | Certain | Separate `wt` binary, not a `fab` subcommand | Discussed — different concern domains, wt works in any git repo without fab init | S:90 R:85 A:90 D:90 |
| 3 | Certain | wt-common.sh → `internal/worktree/` package | Shared library becomes proper Go package with testable units | S:85 R:85 A:90 D:95 |
| 4 | Certain | Exclude wt pr from scope — overlaps with /git-pr. Shell script also removed | Clarified — user changed: wt pr dropped entirely, not ported and not kept | S:95 R:85 A:80 D:75 |
| 5 | Certain | Preserve interactive TUI (menus, fzf detection) | Clarified — user confirmed | S:95 R:70 A:75 D:70 |
| 6 | Certain | No shim layer — direct cutover | Clarified — user changed: shim not needed | S:95 R:90 A:80 D:75 |
| 7 | Certain | Both binaries in same per-platform archive | Single download, same distribution path as existing fab binary | S:85 R:85 A:85 D:90 |
| 8 | Certain | `fab/.kit/bin/` added to PATH via env-packages.sh. Ensure same or more test coverage | Clarified — user confirmed with test coverage requirement | S:95 R:85 A:80 D:75 |
| 9 | Confident | Remove all legacy wt shell scripts including wt-pr | Scripts become dead code once binary is validated. wt-pr also dropped (not ported, /git-pr covers it) | S:75 R:70 A:80 D:75 |

9 assumptions (8 certain, 1 confident, 0 tentative, 0 unresolved).
