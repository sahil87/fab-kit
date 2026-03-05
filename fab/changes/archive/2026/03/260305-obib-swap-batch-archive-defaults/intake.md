# Intake: Swap batch-fab-archive-change defaults

**Change**: 260305-obib-swap-batch-archive-defaults
**Created**: 2026-03-05
**Status**: Draft

## Origin

> `batch-fab-archive-changes.sh: move the default behaviour to --list. Make --all the new default behaviour when no arguments are passed - as that is the most frequent action`

One-shot request. The user wants to swap the default (no-argument) behavior of `batch-fab-archive-change.sh` from `--list` to `--all`, since archiving all eligible changes is the most common use case.

## Why

Currently, running `batch-fab-archive-change.sh` with no arguments shows a list of archivable changes (`--list`). The user must then re-run with `--all` to actually archive them. Since the most frequent action is archiving all eligible changes, the no-argument default should perform the archive directly, reducing a redundant two-step invocation to one.

If unchanged, every batch archive operation requires an extra invocation, adding friction to the most common workflow.

## What Changes

### Swap default behavior in `fab/.kit/scripts/batch-fab-archive-change.sh`

The current no-argument fallback on line 82–84:

```bash
if [[ $# -eq 0 ]]; then
  set -- --list
fi
```

Changes to:

```bash
if [[ $# -eq 0 ]]; then
  set -- --all
fi
```

The `--list` flag remains available as an explicit option. The `--all` flag also remains as an explicit option for clarity. Only the zero-argument default changes.

### Update usage text

The usage/help text should reflect the new default behavior — noting that running with no arguments archives all eligible changes, and `--list` is available to preview first.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Update batch archive script documentation to reflect new default behavior

## Impact

- `fab/.kit/scripts/batch-fab-archive-change.sh` — single file change
- No API or dependency changes
- Users who relied on the old default (list on no args) will now get archive behavior — but this is the intended improvement

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only the zero-argument default changes; both `--list` and `--all` flags remain as explicit options | Directly stated in user request — swap defaults, not remove flags | S:90 R:95 A:95 D:95 |
| 2 | Certain | The usage/help text is updated to reflect the new default | Standard practice when changing CLI defaults | S:85 R:95 A:90 D:95 |
| 3 | Confident | No confirmation prompt needed before archiving when invoked with no args | Script already archives without confirmation via `--all`; changing default just removes the list step | S:70 R:75 A:80 D:80 |

3 assumptions (2 certain, 1 confident, 0 tentative, 0 unresolved).
