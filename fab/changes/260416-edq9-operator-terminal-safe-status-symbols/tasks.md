# Tasks: Operator Terminal-Safe Status Symbols

**Change**: 260416-edq9-operator-terminal-safe-status-symbols
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Replace emoji health indicators in status frame example code block in `src/kit/skills/fab-operator.md` (lines 178-184): 🟢→●, 🟡→◌, 🔴→✗
- [x] T002 Replace emoji in change health legend line in `src/kit/skills/fab-operator.md` (line 203): 🟢→●, 🟡→◌, 🔴→✗
- [x] T003 Replace emoji in watch health legend line in `src/kit/skills/fab-operator.md` (line 205): 🟢→●, 🟡→◌, 🔴→✗, ⏸→–
- [x] T004 Update Health column description from "Status emoji" to "Status indicator" in `src/kit/skills/fab-operator.md` (line 196)
- [x] T005 Replace emoji in autopilot queue progress reference in `src/kit/skills/fab-operator.md` (line 420): 🟢/🟡→●/◌

## Phase 2: Spec Update

- [x] T006 Update `docs/specs/skills/SPEC-fab-operator.md` to reflect new status symbols wherever old emoji are referenced (no emoji found — spec uses abstract descriptions, no changes needed)

---

## Execution Order

All T001-T005 are in the same file and can execute sequentially in a single pass. T006 is independent.
