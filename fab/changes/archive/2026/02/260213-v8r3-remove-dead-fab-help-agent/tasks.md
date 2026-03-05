# Tasks: Remove Dead fab-help Agent File

**Change**: 260213-v8r3-remove-dead-fab-help-agent
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Implementation

- [x] T001 [P] Delete `.claude/agents/fab-help.md` — remove the unused agent definition file
- [x] T002 [P] Update `fab/docs/fab-workflow/kit-architecture.md` — remove the `.claude/agents/fab-help.md    # Generated with model: haiku` line from the "Model Tier Agent Files (Dual Deployment)" code block (line 104)

---

## Execution Order

- T001 and T002 are independent — can run in parallel
