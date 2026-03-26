# Tasks: Unified Tick Status List

**Change**: 260326-oxgu-unified-tick-status-list
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Replace the tick status frame example block in `fab/.kit/skills/fab-operator7.md` §4 Tick Behavior (lines 173-183) with the unified list format — new header line, `[change]`/`[watch]` type prefixes, `▶` autopilot marker, consistent column layout
- [x] T002 Replace the stage/watch indicator description paragraph (line 185) with the column structure table and health emoji definitions for both changes and watches
- [x] T003 Update the §4 Tick Behavior step 1 description (line 171) to reference the unified list rendering instead of separate monitored changes + watches sections

## Phase 2: Downstream References

- [x] T004 Update §6 Autopilot references to `autopilot 1/3` in the header — replace with description of `▶` per-entry approach
- [x] T005 Update the §4 Tick Behavior step 4 description (line 189) — remove "autopilot step" as a separate tick step or reframe as "autopilot dispatch" since autopilot state is now visible per-entry

## Phase 3: Sync

- [x] T006 Run `fab/.kit/scripts/fab-sync.sh` to update the deployed copy at `.claude/skills/fab-operator7.md`

---

## Execution Order

- T001 → T002 → T003 (sequential, all modify the same §4 region)
- T004 and T005 can run after T001-T003
- T006 runs last (depends on all prior tasks)
