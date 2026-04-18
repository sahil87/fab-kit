# Tasks: Pane Capture -l Flag Bug

**Change**: 260405-iujc-pane-capture-l-flag-bug
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 2: Core Implementation

- [x] T001 Update `src/kit/skills/_cli-fab.md`: in the `fab pane capture` flags table, change the `-l` row to show both forms (`` `-l`, `--lines` ``) and update description to remove the internal implementation detail
- [x] T002 Run `fab sync` to propagate the canonical skill update to `.claude/skills/_cli-fab/SKILL.md`

---

## Execution Order

- T001 must complete before T002 (sync deploys the updated source)

---

## Clarifications

### Session 2026-04-05 (auto)

| # | Action | Detail |
|---|--------|--------|
| — | Synopsis line scope | `fab pane capture <pane> [-l N]` usage synopsis also shows only `-l`; spec requirements scope the fix to the flag TABLE only (no requirement to update the synopsis). Out of scope per spec Non-Goals and requirement wording. <!-- clarified: synopsis update is out of spec scope — flag table is the authoritative reference --> |
