# Brief: Rename Batch Scripts and Add Batch Archive

**Change**: 260213-v3rn-batch-commands
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Batch commands
> Currently we have fab-batch-new.sh and fab-batch-switch.sh
> Rename to batch-new-backlog.sh, batch-switch-change.sh
> Create new batch-archive-change.sh that can run fab-archive on multiple changes

## Why

The current `fab-batch-*.sh` naming is inconsistent — the `fab-` prefix is redundant since these already live in `fab/.kit/scripts/`, and the names don't clearly describe what entity they operate on (backlog items vs changes). Renaming to `batch-{verb}-{entity}.sh` makes the purpose immediately obvious. Additionally, there's no batch mechanism for archiving completed changes, requiring manual one-at-a-time `/fab-continue` invocations.

## What Changes

- **Rename** `fab-batch-new.sh` to `batch-new-backlog.sh` — same functionality, clearer name (operates on backlog IDs)
- **Rename** `fab-batch-switch.sh` to `batch-switch-change.sh` — same functionality, clearer name (operates on change names/IDs)
- **Create** `batch-archive-change.sh` — new script that opens tmux tabs with Claude Code sessions running `/fab-continue` on changes that are ready for archive (review:done stage), following the same worktree + tmux + Claude pattern as the other batch scripts
<!-- assumed: archive command is /fab-continue — archive is the next stage after review:done, and /fab-continue is the generic stage-advancement skill -->

## Affected Docs

- `fab-workflow/kit-architecture`: (modify) Update script inventory to reflect renamed files and new script

## Impact

- **Scripts**: `fab/.kit/scripts/fab-batch-new.sh`, `fab/.kit/scripts/fab-batch-switch.sh` (rename), new `batch-archive-change.sh`
- **No external references**: Old script names are only referenced in their own comments/usage text — no other files need updating
- **No API changes**: These are standalone shell scripts invoked directly from the command line

## Open Questions

None — the scope, naming, and pattern are all well-defined.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Archive via `/fab-continue` | Archive is the next stage after review:done; `/fab-continue` is the stage-advancement skill; no separate `/fab-archive` skill exists |
| 2 | Confident | Same worktree + tmux + Claude pattern | Consistent with `fab-batch-new.sh` and `fab-batch-switch.sh` |

2 assumptions made (2 confident, 0 tentative). Run /fab-clarify to review.
