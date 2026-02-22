# Intake: wt-create stderr & wt-list flags

**Change**: 260222-s101-wt-create-stderr-wt-list-flags
**Created**: 2026-02-22
**Status**: Draft

## Origin

> Fix wt-create output contract for script consumers: (1) In --non-interactive mode, redirect all human-friendly messages to stderr so only the worktree path goes to stdout — eliminates the fragile `tail -1` pattern. (2) Add status/json/path flags to wt-list.

Discussion-based. Prior conversation explored all 6 wt commands, identified 10 DX improvements, user narrowed to items 2 and 5. The `--non-interactive` implies porcelain approach was chosen over separate `--porcelain`/`--quiet` flags — the user owns both the scripts and the commands, and there's no real use case for "non-interactive but human-readable stdout."

## Why

**Problem 1 — wt-create output contract is fragile**: Three batch consumers (`batch-fab-new-backlog.sh`, `batch-fab-switch-change.sh`, `pipeline/dispatch.sh`) capture wt-create's path via `| tail -1`. This works only because `echo "$WT_PATH"` happens to be the last line. But `wt_print_success`, init script output, and open-app messages all share stdout — any change to message ordering or content breaks all three callers silently. The `--reuse` codepath already does it right (messages to stderr, path to stdout).

**Problem 2 — wt-list is too bare for scripting and monitoring**: Currently shows name, branch, path in a formatted table — useful for humans but not for scripts or status checks. There's no way to get a single worktree's path by name (needed for a future `wt-cd`), no machine-readable output, and no at-a-glance dirty/unpushed state. The building blocks exist in `wt-common.sh` (`wt_has_uncommitted_changes`, `wt_get_unpushed_count`) but aren't surfaced.

## What Changes

### 1. wt-create: `--non-interactive` implies porcelain output

When `--non-interactive` is set:
- `wt_print_success` output → stderr
- `wt_run_worktree_setup` output (init script) → stderr
- `wt-open` invocation output → stderr (already skipped in non-interactive, but defensive)
- `echo "$WT_PATH"` remains on stdout as the sole output
- The `--reuse` codepath already follows this pattern — no changes needed there

Callers can then simplify from:
```bash
wt_path=$(wt-create --non-interactive ... | tail -1)
```
to:
```bash
wt_path=$(wt-create --non-interactive ...)
```

Update all three batch callers to drop `| tail -1`.

### 2. wt-list: `--path <name>` flag

New flag that outputs just the absolute path for a named worktree (by basename match). Outputs nothing and exits non-zero if not found. Enables:
```bash
cd "$(wt-list --path swift-fox)"
```

### 3. wt-list: `--json` flag

Outputs worktree data as a JSON array. Each entry includes `name`, `branch`, `path`, `is_main`, `is_current`, `dirty` (bool), `unpushed` (int). Uses `jq` for formatting (already a project prerequisite).

### 4. wt-list: status column in default view

Add a status indicator to the formatted table output:
- `*` for dirty (uncommitted changes or untracked files)
- `↑N` for N unpushed commits
- Clean worktrees show nothing

Example:
```
  swift-fox      wt/swift-fox   *  ↑2   /path/to/worktrees/swift-fox
  calm-owl       feature/auth          /path/to/worktrees/calm-owl
```

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update wt package description — new flags, output contract change

## Impact

- **wt-create**: Output behavior changes in `--non-interactive` mode only. Interactive mode unchanged.
- **wt-list**: Additive — new flags, enhanced default view. No breaking changes.
- **batch-fab-new-backlog.sh**: Remove `| tail -1` (simplification).
- **batch-fab-switch-change.sh**: Remove `| tail -1` (simplification).
- **pipeline/dispatch.sh**: Remove `| tail -1` (simplification).
- **wt-pr**: Same `--non-interactive` flag exists — same stderr redirect should apply for consistency.
- **Test suites**: `wt-create.bats` and `wt-list.bats` need new test cases.

## Open Questions

None — design was resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--non-interactive` implies porcelain (no separate flag) | Discussed — user explicitly chose this over `--porcelain` and `--quiet` alternatives | S:95 R:90 A:95 D:95 |
| 2 | Certain | Apply same stderr redirect to `wt-pr --non-interactive` | Consistency — wt-pr shares the same flag and output pattern | S:80 R:95 A:90 D:95 |
| 3 | Certain | Use `jq` for `--json` formatting | Already a project prerequisite (validated by `sync/1-prerequisites.sh`) | S:85 R:95 A:95 D:95 |
| 4 | Confident | Status column uses `*` for dirty and `↑N` for unpushed | Common git convention (git prompt, lazygit). Could use other symbols but these are well-established | S:70 R:95 A:80 D:75 |
| 5 | Confident | `--path` exits non-zero when worktree not found | Standard CLI convention for lookup commands. Alternative: output empty string | S:75 R:90 A:85 D:80 |
| 6 | Confident | `--json` includes dirty/unpushed fields | Requires iterating worktrees and checking git state — slight perf cost but essential for monitoring use case | S:70 R:85 A:80 D:80 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
