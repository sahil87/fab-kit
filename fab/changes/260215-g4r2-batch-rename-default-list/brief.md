# Brief: Batch Script Rename and Default List Behavior

**Change**: 260215-g4r2-batch-rename-default-list
**Created**: 2026-02-15
**Status**: Draft

## Origin

> Change the three batch scripts in `fab/.kit/scripts/` so that (1) calling with no arguments shows `--list` output instead of help, with `-h`/`--help` for help, and (2) rename from `batch-*` to `batch-fab-*`.

## Why

The current no-arg behavior (showing help) is less useful than showing the list of available items. Users invoking a batch script with no args most likely want to see what's available, not read usage docs. The rename to `batch-fab-*` aligns with the `fab-` prefix convention for user-facing scripts and avoids ambiguity in repos that might have other batch scripts.

## What Changes

- **Default no-arg behavior**: All three batch scripts (`batch-new-backlog.sh`, `batch-switch-change.sh`, `batch-archive-change.sh`) will run `--list` when invoked with no arguments instead of showing usage text
- **Help via flags only**: `-h` and `--help` will continue to show usage text (no behavioral change)
- **Script rename**: `batch-*` → `batch-fab-*` (three files)
- **Usage text update**: Script names in usage/examples text updated to match new filenames
- **Documentation update**: Memory and specs files updated with new names and naming pattern

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update directory tree listing, batch scripts section naming pattern and descriptions

## Impact

- **Scripts**: `fab/.kit/scripts/batch-new-backlog.sh`, `batch-switch-change.sh`, `batch-archive-change.sh` — renamed and behavior modified
- **Docs**: `docs/memory/fab-workflow/kit-architecture.md` — directory tree and batch scripts section
- **Specs**: `docs/specs/architecture.md` — prefix convention table and batch scripts table
- **No downstream dependencies**: These scripts are standalone entry points invoked manually from the terminal; no other scripts source or call them

## Open Questions

None — the scope is well-defined and the implementation approach is clear.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg triggers `set -- --list` fallthrough | The `--list` case already exists in each script's case statement; `set --` is standard bash for rewriting positional params | S:95 R:95 A:95 D:95 |
| 2 | Certain | Rename pattern is `batch-fab-{verb}-{entity}.sh` | User explicitly specified the pattern; examples given | S:95 R:90 A:90 D:95 |
| 3 | Certain | `batch-fab-` stays as separate row in prefix convention table | User explicitly chose this when asked | S:95 R:95 A:95 D:95 |
| 4 | Confident | Archive change records not updated | Archive records are historical snapshots; updating them would rewrite history unnecessarily | S:70 R:90 A:85 D:80 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
