# Brief: Archive Restore Mode

**Change**: 260214-v7k3-archive-restore-mode
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Add a "restore" mode to `/fab-archive` that moves an archived change back to the active changes folder. Does not switch to the restored change unless explicitly specified. Removes the entry from `archive/index.md`.

## Why

Archiving is currently a one-way operation. If a user needs to revisit or continue work on an archived change, there's no built-in way to restore it — they'd have to manually move the folder and clean up the index. A restore mode makes the archive workflow fully reversible, consistent with Constitution Principle III (Idempotent Operations).

## What Changes

- Add `restore` subcommand to `/fab-archive`: `/fab-archive restore <change-name>`
- Move `fab/changes/archive/{name}/` back to `fab/changes/{name}/`
- Remove the corresponding entry from `fab/changes/archive/index.md`
- Do **not** switch to the restored change by default (no `fab/current` modification)
- Support optional `--switch` flag (or intent detection) to activate after restore
- Preserve all artifacts (`.status.yaml`, `brief.md`, `spec.md`, `tasks.md`, etc.) as-is — no status reset

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the restore subcommand under `/fab-archive`
- `fab-workflow/change-lifecycle`: (modify) Add restore as a lifecycle transition (archived → active)

## Impact

- **Skill file**: `fab/.kit/skills/fab-archive.md` — add restore mode section with its own behavior, arguments, pre-flight, and output format
- **Archive index**: `fab/changes/archive/index.md` — restore removes entries
- **No script changes**: Restore is pure skill-level logic (move folder, edit index), no new shell scripts needed

## Open Questions

(None — requirements fully specified by user input.)

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Preserve .status.yaml and all artifacts as-is during restore | User wants to resume work; resetting state would lose progress context |

1 assumption made (1 confident, 0 tentative). Run /fab-clarify to review.
