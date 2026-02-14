# Tasks: Archive Restore Mode

**Change**: 260214-v7k3-archive-restore-mode
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 Add restore mode section to `fab/.kit/skills/fab-archive.md` — add `## Restore Mode` with arguments (`restore <change-name> [--switch]`), pre-flight (scan `fab/changes/archive/` for matching folders), and step-by-step behavior (move folder, remove index entry, optional pointer update)
- [x] T002 Add restore name resolution logic to `fab/.kit/skills/fab-archive.md` — document case-insensitive substring matching against `fab/changes/archive/` folder names, with exact/single/ambiguous/no-match flows (mirrors `/fab-switch` resolution pattern)

## Phase 2: Edge Cases & Output

- [x] T003 Add idempotent restore behavior to `fab/.kit/skills/fab-archive.md` — document detection of folder already in `fab/changes/` (skip move, complete remaining steps), consistent with archive resumability pattern
- [x] T004 Add restore output format to `fab/.kit/skills/fab-archive.md` — structured summary (Moved/Index/Pointer lines) and `Next:` suggestion per `_context.md` convention
- [x] T005 Add restore error handling to `fab/.kit/skills/fab-archive.md` — no archived changes, no match, ambiguous match, archive folder missing

## Phase 3: Integration

- [x] T006 Update `fab/.kit/skills/fab-archive.md` top-level sections — update Arguments to include `restore` subcommand and `--switch` flag, update Purpose to mention restore, update Key Properties table
- [x] T007 Update the `/fab-archive` argument/syntax line at the top of `fab/.kit/skills/fab-archive.md` to reflect new `restore` subcommand: `/fab-archive [<change-name>] | restore <change-name> [--switch]`

---

## Execution Order

- T001 blocks T003 (idempotent behavior references restore steps)
- T001 blocks T004 (output format references step outcomes)
- T001 blocks T005 (error handling references restore flow)
- T006 and T007 depend on T001-T005 (integration updates reference all restore content)
- T002 is independent of T003-T005, can run alongside them after T001
