# Intake: Add --reuse flag to wt-create

**Change**: 260222-6ldg-wt-create-reuse-flag
**Created**: 2026-02-22
**Status**: Draft

## Origin

> Add --reuse flag to wt-create that returns existing worktree path instead of erroring on name collision. Update batch-fab-switch-change.sh and dispatch.sh to pass --reuse. Add/update wt-create tests.

Preceded by a `/fab-discuss` session that traced all worktree/branch collision points across the codebase. The discussion identified 6 collision points, classified them as hard blockers vs soft failures, and concluded that `batch-fab-switch-change.sh` and `dispatch.sh` need idempotent worktree creation while `batch-fab-new-backlog.sh` and interactive use should keep the hard error.

User decision: `--reuse` should return the existing worktree path blindly (no branch verification) for now.

## Why

1. **Problem**: `batch-fab-switch-change.sh` fails when a worktree already exists from a previous run. The batch script creates worktrees via `wt-create --non-interactive --worktree-name "$match" "$branch_name"`, which errors on name collision (exit 1). This means re-running the batch script after a partial failure or interruption requires manually cleaning up worktrees first.

2. **Consequence**: Pipeline resumability is broken. If `batch-fab-switch-change` is interrupted mid-run, the user must manually `wt-delete` each already-created worktree before retrying. `dispatch.sh` works around this with a bespoke `wt_get_worktree_path_by_name` check before calling `wt-create`, but that check only covers registered git worktrees — orphaned directories still cause failures.

3. **Approach**: Add `--reuse` to `wt-create` itself so the idempotency behavior lives in one place. Callers that need resume semantics pass `--reuse`; callers that need collision detection (fresh creation) don't. This is cleaner than each caller implementing its own pre-check.

## What Changes

### 1. New `--reuse` flag in `wt-create`

**File**: `fab/.kit/packages/wt/bin/wt-create`

Add a `--reuse` flag to the argument parser. When `--reuse` is set and the worktree name collides with an existing directory at `$WT_WORKTREES_DIR/$final_name`:

- Skip worktree creation entirely
- Set `WT_PATH` to the existing worktree path
- Skip init script execution (worktree already initialized)
- Skip app opening
- Print the existing worktree path as the last line of output (same contract as normal creation)

When `--reuse` is set but there is no collision, proceed with normal creation (the flag is a no-op).

`--reuse` requires `--worktree-name` — it only makes sense when the caller specifies an exact name. If `--reuse` is passed without `--worktree-name`, error: "--reuse requires --worktree-name".

### 2. Update `batch-fab-switch-change.sh` to pass `--reuse`

**File**: `fab/.kit/scripts/batch-fab-switch-change.sh` (line 123)

Change:
```bash
wt_path=$(wt-create --non-interactive --worktree-name "$match" "$branch_name" | tail -1)
```
To:
```bash
wt_path=$(wt-create --non-interactive --reuse --worktree-name "$match" "$branch_name" | tail -1)
```

### 3. Update `dispatch.sh` to pass `--reuse` and remove bespoke check

**File**: `fab/.kit/scripts/pipeline/dispatch.sh`

In `create_worktree()`, remove the `wt_get_worktree_path_by_name` pre-check (lines 94-99) and add `--reuse` to the `wt-create` call (line 112):

```bash
wt_path=$(wt-create --non-interactive --reuse --worktree-open skip --worktree-name "$CHANGE_ID" "$CHANGE_BRANCH" | tail -1)
```

This simplifies `create_worktree()` from a two-step check-then-create to a single idempotent call.

### 4. New test cases in `wt-create.bats`

**File**: `src/packages/wt/tests/wt-create.bats`

Add a new section `# --reuse Flag Tests` with these cases:

- `--reuse returns existing worktree path on collision` — create worktree, call again with `--reuse`, assert success and same path
- `--reuse creates normally when no collision` — call with `--reuse` on fresh name, assert normal creation
- `--reuse skips init script on collision` — create worktree, add init script, call again with `--reuse`, assert init marker not created a second time
- `--reuse requires --worktree-name` — call `--reuse` without `--worktree-name`, assert failure
- `--reuse prints path as last line` — verify output contract matches normal creation

Also add an edge case in `edge-cases.bats`:

- `--reuse with orphaned directory (not registered as git worktree)` — create dir manually at worktree path, call with `--reuse`, assert it returns the path (blind reuse, no verification)

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the `--reuse` flag as part of wt-create's CLI interface

## Impact

- **`wt-create`**: New optional flag, fully backward-compatible. No change to default behavior.
- **`batch-fab-switch-change.sh`**: Becomes idempotent — safe to re-run after interruption.
- **`dispatch.sh`**: Simplified — removes 6 lines of bespoke worktree reuse logic in favor of the flag.
- **`batch-fab-new-backlog.sh`**: No change — keeps hard error on collision.
- **Interactive `wt-create`**: No change — `--reuse` is opt-in.

## Open Questions

- None. Scope, behavior, and callsite decisions were resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--reuse` returns path blindly without branch verification | Discussed — user explicitly chose blind reuse for simplicity | S:95 R:80 A:95 D:95 |
| 2 | Certain | `--reuse` requires `--worktree-name` | Reuse only makes sense with an explicit name; random names wouldn't collide intentionally | S:90 R:90 A:90 D:90 |
| 3 | Certain | `batch-fab-switch-change.sh` and `dispatch.sh` get `--reuse`; `batch-fab-new-backlog.sh` does not | Discussed — resume scripts need idempotency, fresh-creation scripts need collision detection | S:95 R:85 A:95 D:95 |
| 4 | Certain | `--reuse` skips init script and app opening on collision | Worktree already exists and was previously initialized; re-running init would be surprising | S:85 R:85 A:85 D:90 |
| 5 | Confident | Remove bespoke reuse check from `dispatch.sh` | The `wt_get_worktree_path_by_name` pre-check becomes redundant with `--reuse` in `wt-create` | S:80 R:70 A:80 D:85 |
| 6 | Confident | Test for orphaned directory edge case | Directory may exist without git worktree registration; `--reuse` should handle this since it's blind | S:75 R:75 A:80 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
