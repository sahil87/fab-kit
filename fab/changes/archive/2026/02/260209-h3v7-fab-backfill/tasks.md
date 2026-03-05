# Tasks: Add `/fab-backfill` Command

**Change**: 260209-h3v7-fab-backfill
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 [P] Create skill file `fab/.kit/skills/fab-backfill.md` with full skill definition — purpose, arguments, context loading, gap detection algorithm, per-gap confirmation flow, output format, error handling, and key properties table
- [x] T002 [P] Create symlink `.claude/skills/fab-backfill.md` → `../../fab/.kit/skills/fab-backfill.md`

## Phase 2: Core Implementation

- [x] T003 Update `fab/.kit/scripts/fab-help.sh` to include `/fab-backfill` in the skill catalog output

## Phase 3: Doc Updates

- [x] T004 [P] Update `fab/docs/fab-workflow/specs-index.md` — replace "reverse-hydration is a future consideration" in the Human-Curated Ownership section with a reference to `/fab-backfill` as the assisted reverse-hydration mechanism
- [x] T005 [P] Update `fab/specs/skills.md` — add a `/fab-backfill` section documenting the skill's purpose, arguments, context loading, and behavior summary (concise, matching existing skill entry style)
- [x] T006 [P] Update `fab/specs/index.md` — no change needed (specs index lists files, not individual skills; `skills.md` already exists)

## Execution Order

- T001 and T002 are independent (parallel)
- T003 depends on T001 (needs the skill to exist for reference)
- T004, T005 are independent (parallel), can run alongside T003
- T006 is a no-op verification — confirm no index update is needed
