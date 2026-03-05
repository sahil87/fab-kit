# Intake: Worktree Status Command

**Change**: 260305-7zq4-worktree-status-command
**Created**: 2026-03-05
**Status**: Draft

## Origin

> Backlog item [7zq4]: "A command line to show the status (status = stage + state) of every worktree"

One-shot request from backlog. The user runs multiple worktrees in parallel (one per fab change) and needs a quick overview of where each worktree stands in the fab pipeline.

## Why

1. **Problem**: When running the assembly-line pattern with multiple worktrees, there is no single command that shows the fab pipeline status per worktree. `wt-list` shows git status (dirty/unpushed), `changeman.sh list` shows fab changes (stage/state), but neither bridges the two — the user must mentally map worktree names to change folders.

2. **Consequence**: Without this, the user must `cd` into each worktree and run `/fab-status` or inspect `fab/current` manually. With 4-8 concurrent worktrees, this is tedious and error-prone.

3. **Approach**: Add a `--status` flag to the existing `wt-list` command (or a new `wt-status` command) that reads each worktree's `fab/current` and `fab/changes/*/.status.yaml` to display the fab stage and state alongside git info.

## What Changes

### New `wt-status` command in `fab/.kit/packages/wt/bin/`

A new command `wt-status` that iterates over all worktrees and, for each one:

1. Reads `fab/current` in the worktree to find the active change (if any)
2. Reads `.status.yaml` for that change to extract stage and state
3. Displays a formatted table with columns:
   - **Worktree name** (basename of worktree path, with `*` for current)
   - **Change** (the active change's ID + slug, or "none")
   - **Stage** (intake/spec/tasks/apply/review/hydrate/ship/review-pr)
   - **State** (pending/active/ready/done/failed)

Example output:
```
Worktrees for: fab-kit
Location: /home/user/code/fab-kit.worktrees

* 7zq4           260305-7zq4-worktree-status    intake   ready
  8ooz           260305-8ooz-persist-confidence  review   active
  swift-fox      260302-a3b1-fix-calc-score      hydrate  done
  (main)         (no active change)
```

### Reuse of existing infrastructure

- Source `wt-common.sh` for `wt_get_repo_context`, `wt_list_worktrees`, color constants
- Use `statusman.sh display-stage` for consistent stage/state extraction (same as `changeman.sh list`)
- Read `fab/current` two-line format (line 1 = ID, line 2 = folder name)

### Integration with `wt-list`

Consider whether to:
- (A) Add `--status` flag to existing `wt-list` — shows fab status alongside git status
- (B) Create a standalone `wt-status` command — dedicated to fab pipeline view

Option B is recommended: `wt-list` is a generic git worktree tool that works outside fab projects. `wt-status` is fab-specific and belongs as a separate command. This follows the existing pattern where each `wt-*` command has a focused responsibility.

### Edge cases

- **Worktree with no `fab/` directory**: Display "(no fab)" or skip silently
- **Worktree with `fab/` but no active change**: Display "(no active change)"
- **Worktree with stale `fab/current`** (points to deleted change): Display "(stale pointer)"
- **Main repo**: Treated like any other worktree — reads its own `fab/current`

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Add wt-status to the wt package command inventory

## Impact

- **Code**: New file `fab/.kit/packages/wt/bin/wt-status` (single shell script)
- **Tests**: New test file `src/packages/wt/tests/wt-status.bats`
- **Docs**: `docs/specs/packages.md` will need a row in the wt commands table (hydrate stage)
- **No breaking changes**: Existing `wt-list` behavior is unchanged

## Open Questions

- None — the scope is clear and the implementation pattern is well-established by other `wt-*` commands.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New standalone `wt-status` command (not a flag on `wt-list`) | `wt-list` is generic git tooling; fab-specific status is a separate concern. Follows existing wt-* single-responsibility pattern. | S:70 R:90 A:90 D:85 |
| 2 | Certain | Use `statusman.sh display-stage` for stage/state extraction | This is the authoritative API used by `changeman.sh list` — consistent and maintained. | S:80 R:95 A:95 D:95 |
| 3 | Certain | Read `fab/current` two-line format for active change resolution | This is the documented format used by all fab scripts. | S:90 R:95 A:95 D:95 |
| 4 | Confident | Human-readable formatted output as default (no `--json` initially) | Backlog says "show the status" — implies human-readable. JSON can be added later if needed. | S:60 R:90 A:80 D:75 |
| 5 | Certain | Source `wt-common.sh` for shared infrastructure | All `wt-*` commands follow this pattern. | S:90 R:95 A:95 D:95 |
| 6 | Confident | Show change folder name (truncated to ID + slug) rather than full `YYMMDD-XXXX-slug` | Full folder names are long; the ID is the primary identifier. But truncation policy is a display choice. | S:55 R:90 A:70 D:65 |
| 7 | Certain | Handle edge cases gracefully (no fab dir, no active change, stale pointer) | Robustness is a constitutional requirement (idempotent operations). | S:75 R:85 A:90 D:90 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
