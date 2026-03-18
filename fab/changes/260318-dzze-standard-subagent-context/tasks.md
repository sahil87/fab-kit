# Tasks: Standard Subagent Context Template

**Change**: 260318-dzze-standard-subagent-context
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add "Standard Subagent Context" subsection and update dispatch pattern in `fab/.kit/skills/_preamble.md` — insert after the existing 5-item dispatch pattern list (line 188), before the `general-purpose` subagents note (line 190). Add item 6 to the numbered list referencing the new subsection. The subsection lists the 5 `fab/project/**` files and specifies the "read these files first" instruction pattern. Include nested dispatch requirement.
- [x] T002 Simplify `fab-continue.md` Review Behavior context list — replace the ad-hoc `fab/project/**` file list at line 153 with a reference to `_preamble.md § Standard Subagent Context`. Keep change-specific context (spec.md, tasks.md, checklist.md, source files, memory files) listed explicitly.

## Phase 2: Specs

- [x] T003 [P] Create `docs/specs/skills/SPEC-preamble.md` — document the `_preamble.md` internal partial's flow diagram, tool usage table, and note the new Standard Subagent Context subsection. Follow the format of existing SPEC files (e.g., `SPEC-fab-continue.md`).
- [x] T004 [P] Update `docs/specs/skills/SPEC-fab-continue.md` — replace inline `fab/project/**` file names in the review sub-agent box with "standard subagent context + change-specific files".

---

## Execution Order

- T001 blocks T002 (T002 references the preamble subsection created in T001)
- T003 and T004 are independent of each other, can run in parallel after T001-T002
