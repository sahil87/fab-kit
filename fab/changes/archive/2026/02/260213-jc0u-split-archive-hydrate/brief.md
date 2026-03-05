# Brief: Split Archive into Hydrate Stage and fab-archive Command

**Change**: 260213-jc0u-split-archive-hydrate
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Split archive into hydrate stage and fab-archive command. Add a new "hydrate" stage to the pipeline (between review and archive) that handles final validation, concurrent change check, and doc hydration into fab/docs/. This becomes the terminal stage for ff/fff — they stop here and leave the change folder in fab/changes/. Remove "archive" from the progress map entirely. Create a new standalone fab-archive skill whose only job is moving fab/changes/{name}/ to fab/changes/archive/{name}/, updating archive index, marking backlog done, and clearing fab/current. It requires hydrate: done as a guard. Update fab-continue, fab-ff, fab-fff, _context.md, and any templates accordingly.

## Why

The current archive stage bundles two distinct concerns: (1) doc hydration — integrating learnings from spec.md into centralized fab/docs/ — and (2) folder housekeeping — moving the change to archive, updating indexes, clearing pointers. These have different triggers: hydration is the logical completion of the pipeline (the agent's work is done), while archiving is a manual housekeeping action the user performs when they're ready to clean up. Splitting them lets ff/fff stop at the meaningful work boundary and leaves archiving as an explicit user action.

## What Changes

- **New pipeline stage `hydrate`** — inserted between `review` and the end of the pipeline. Handles: final validation (all tasks + checklist [x]), concurrent change check, and doc hydration into `fab/docs/`.
- **Remove `archive` from progress map** — the 6-stage pipeline (`brief → spec → tasks → apply → review → archive`) becomes a 5-stage pipeline (`brief → spec → tasks → apply → review → hydrate`). Archive is no longer a tracked stage.
- **New standalone `/fab-archive` skill** — a lightweight command whose only job is moving `fab/changes/{name}/` → `fab/changes/archive/{name}/`, updating `fab/changes/archive/index.md`, marking the backlog item done, and clearing `fab/current`. Requires `hydrate: done` as a guard.
- **Update `/fab-continue`** — replace archive behavior with hydrate behavior as the terminal stage. Add hydrate to stage guard table, context loading, and output templates. Remove archive from the stage progression.
- **Update `/fab-ff`** — terminal step becomes hydrate (Step 8). Remove archive step.
- **Update `/fab-fff`** — terminal step becomes hydrate (Step 4). Remove archive step.
- **Update `_context.md`** — next steps convention table, stage references.
- **Update `config.yaml`** — replace `archive` stage with `hydrate` in the stages list.
- **Update `status.yaml` template** — replace `archive: pending` with `hydrate: pending`.

## Affected Docs

### New Docs
- None (fab-archive skill file is implementation, not a centralized doc)

### Modified Docs
- `fab-workflow/change-lifecycle`: Pipeline stage sequence changes from 6 to 5+archive
- `fab-workflow/execution-skills`: Archive behavior section needs rewriting — split into hydrate (pipeline) and archive (standalone)
- `fab-workflow/templates`: status.yaml template changes
- `fab-workflow/configuration`: stages list in config.yaml changes
- `fab-workflow/schemas`: .status.yaml progress map keys change

### Removed Docs
- None

## Impact

- **Skill files**: `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `_context.md` — all reference the 6-stage pipeline and archive behavior
- **New skill file**: `fab-archive.md` — new standalone skill
- **Templates**: `fab/.kit/templates/status.yaml` — progress map keys
- **Config**: `fab/config.yaml` — stages list
- **Existing changes**: Changes already in archive with `archive: done` in their `.status.yaml` are unaffected. Active changes with `archive: pending` will need the key renamed to `hydrate: pending` (or handled gracefully).
- **fab-status, fab-help**: May reference the stage list — need audit

## Open Questions

- None — design was discussed and agreed before brief creation.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Existing archived changes keep their old status format | They're already archived — no migration needed for historical data |
| 2 | Confident | fab-status and fab-help will need minor updates to reflect the new stage list | Logical consequence of changing the pipeline, but scope not yet audited |
| 3 | Confident | The status.yaml template replaces `archive: pending` with `hydrate: pending` rather than adding both | User explicitly said to remove archive from the progress map |
| 4 | Confident | fab-archive does not need a progress map entry — it's a one-shot action | User agreed with this recommendation |

4 assumptions made (4 confident, 0 tentative).
