# Tasks: Standardize Tmux Tab Naming for Spawned Agents

**Change**: 260328-iqt8-standardize-tmux-tab-naming
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Update generic "Open agent tab" example in `fab/.kit/skills/fab-operator7.md` — change `tmux new-window -n "fab-<id>"` to `tmux new-window -n "⚡<wt>"` in the Spawning an Agent section (~line 305)
- [x] T002 [P] Update "From existing change" spawn path in `fab/.kit/skills/fab-operator7.md` — change `tmux new-window -n "fab-<id>"` to `tmux new-window -n "⚡<wt>"` (~line 365)
- [x] T003 [P] Update "From raw text" spawn path in `fab/.kit/skills/fab-operator7.md` — change `tmux new-window -n "fab-<wt>"` to `tmux new-window -n "⚡<wt>"` (~line 373)
- [x] T004 [P] Update "From backlog ID or Linear issue" spawn path in `fab/.kit/skills/fab-operator7.md` — change `tmux new-window -n "fab-<id>"` to `tmux new-window -n "⚡<wt>"` (~line 382)

## Phase 2: Memory Update

- [x] T005 Update `docs/memory/fab-workflow/execution-skills.md` — add entry documenting the tab naming convention change from `fab-<id>` to `⚡<wt>`

---

## Execution Order

- T001–T004 are independent and parallel (all edit the same file at different locations)
- T005 depends on T001–T004 (documents the completed change)
