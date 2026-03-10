# Intake: Build wt Go Binary

**Change**: 260305-jug9-3-build-wt-go-binary
**Created**: 2026-03-05
**Status**: Draft

## Origin

> 3-build-wt-go-binary: Build the wt Go binary — port wt-common.sh shared library and all 7 wt-* commands (wt-create, wt-list, wt-open, wt-delete, wt-pr, wt-init, wt-status) into subcommands of a single `wt` binary. Separate binary from `fab` — different concern domain (git worktree operations vs YAML/status management).

Phase 2 of the Go migration. Depends on Phase 1 (fab binary) being complete. The wt scripts are not a performance bottleneck (git operations dominate), but the rewrite provides code quality benefits: proper argument parsing, testability, richer output (JSON mode), and elimination of the 578-line wt-common.sh sourced by every command.

## Why

1. **Code quality**: wt-common.sh (578 lines) is sourced by every wt-* script — a large shared library parsed on every invocation. Go eliminates this repeated parse cost and provides proper modularity.

2. **Testability**: The wt scripts have complex interactive flows (menus, fzf integration, rollback handling in wt-delete). Go makes these testable with proper unit tests rather than bats integration tests.

3. **Consistency**: After Phase 1, the lib/ scripts are Go. Having wt scripts remain as bash creates two development paradigms. A unified Go codebase reduces cognitive load.

4. **Future capabilities**: Go enables features hard to do in bash — structured JSON output for machine consumption, parallel git operations, proper error handling with context.

## What Changes

### Go Module (same module as fab binary)

The wt binary lives in the same Go module at `src/fab-go/` but builds a separate binary:

```
src/fab-go/
├── cmd/
│   ├── fab/
│   │   └── main.go          # fab binary (from Phase 1)
│   └── wt/
│       └── main.go          # wt binary
├── internal/
│   ├── ... (existing from Phase 1)
│   └── worktree/            # wt shared library
│       ├── worktree.go      # git worktree operations
│       ├── names.go         # memorable name generation
│       ├── stash.go         # stash/rollback logic
│       ├── menu.go          # interactive TUI (menus, fzf)
│       └── worktree_test.go
```

### Port: wt-common.sh → `internal/worktree/`

Port the 578-line shared library covering:

- Git worktree discovery and path resolution (`git worktree list --porcelain`)
- Memorable random name generation (adjective-noun pairs)
- Stash management and rollback logic
- Interactive menu rendering (numbered lists, fzf when available)
- Git branch operations (create, delete, fetch, push)
- Repository root detection and sibling worktree directory convention (`<repo>.worktrees/`)

### Port: wt-create → `wt create`

Create a git worktree with opinionated defaults:
- Worktrees as siblings in `<repo>.worktrees/`
- Memorable random names (unless `--name` provided)
- Auto-creates branch matching worktree name
- Optional `--non-interactive` mode (outputs path to stdout)
- Opens worktree after creation (delegates to `wt open`)

### Port: wt-list → `wt list`

List all git worktrees for the current repository:
- Tabular output with name, branch, path, status
- Optional `--path <name>` to output single worktree path (for shell scripting)
- Optional JSON output mode

### Port: wt-open → `wt open`

Open a worktree in editor, terminal, or file manager:
- Detects available tools (VS Code, Cursor, Terminal, Finder/Nautilus)
- Interactive menu for tool selection
- Supports `--editor`, `--terminal`, `--files` flags for non-interactive use

### Port: wt-delete → `wt delete`

Delete a worktree with safety features:
- Interactive confirmation with rollback handling
- Detects uncommitted changes, offers stash
- Optional branch cleanup after worktree removal
- "All" option to delete multiple worktrees
- Complex interactive flow — the most involved port

### Port: wt-pr → `wt pr`

Create a worktree for reviewing a GitHub PR:
- `wt pr <number>` fetches PR branch via `gh pr view`
- Creates worktree checked out to PR branch
- Opens worktree after creation

### Port: wt-init → `wt init` and wt-status → `wt status`

- `wt init`: Run the init script for a worktree (sources `.envrc`, runs setup)
- `wt status`: Show worktree status with fab change stage info (calls `fab status` internally)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document wt Go binary alongside fab binary

## Impact

- **Source**: New `src/fab-go/cmd/wt/` and `src/fab-go/internal/worktree/` (~1,500-2,000 lines estimated)
- **Existing wt scripts**: Unchanged in this change — parity testing and switchover is the next change
- **gh CLI dependency**: wt-pr requires `gh` — the Go binary will shell out to `gh` (same as the bash version)

## Open Questions

- None — design decisions settled in preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Separate `wt` binary (not merged into `fab`) | Discussed — different concern domains, matches current package separation | S:90 R:85 A:90 D:90 |
| 2 | Certain | Same Go module as fab (`src/fab-go/`) | Shared internal packages possible (e.g., resolve), single go.mod | S:85 R:85 A:85 D:90 |
| 3 | Certain | wt-common.sh → internal/worktree/ package | Discussed — shared library becomes proper Go package | S:85 R:85 A:90 D:95 |
| 4 | Confident | Shell out to `gh` for GitHub operations | Rewriting gh API calls in Go adds complexity for marginal benefit. gh handles auth, pagination, rate limiting | S:75 R:85 A:80 D:75 |
| 5 | Confident | Preserve interactive TUI (menus, fzf) | Users rely on interactive flows. Go TUI libraries (bubbletea, survey) can replicate this. Alternative: drop interactivity in favor of flags-only | S:70 R:70 A:75 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
