# Tasks: Operator7 Direct fab-new Spawn

**Change**: 260326-13ro-operator7-direct-fab-new-spawn
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Replace "From raw text" subsection in `fab/.kit/skills/fab-operator7.md` §6 — remove the 2-step `idea add` flow (lines 342-344) and replace with the direct spawn sequence: worktree → resolve deps → spawn with `/fab-new <description>` → enroll → completion
- [x] T002 Update the explanatory paragraph (lines 346) in `fab/.kit/skills/fab-operator7.md` — remove `idea add` reference, explain that `/fab-new` handles traceability directly via its intake Origin section

## Phase 2: Verification

- [x] T003 Verify no other references to `idea add` remain in `fab/.kit/skills/fab-operator7.md` after edits

---

## Execution Order

- T001 and T002 are adjacent edits in the same file, execute sequentially
- T003 runs after T001 and T002
