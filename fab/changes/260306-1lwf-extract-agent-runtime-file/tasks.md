# Tasks: Extract Agent Runtime to Gitignored File

**Change**: 260306-1lwf-extract-agent-runtime-file
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `.fab-runtime.yaml` to `.gitignore`

## Phase 2: Core Implementation

- [x] T002 [P] Update `fab/.kit/hooks/on-stop.sh` to write `agent.idle_since` to `.fab-runtime.yaml` instead of `.status.yaml`
- [x] T003 [P] Update `fab/.kit/hooks/on-session-start.sh` to clear `agent` block from `.fab-runtime.yaml` instead of `.status.yaml`

## Phase 3: Documentation

- [x] T004 [P] Update `docs/memory/fab-workflow/schemas.md` — rewrite "Ephemeral Runtime State" section to document `.fab-runtime.yaml` location and keyed structure
- [x] T005 [P] Update `docs/memory/fab-workflow/pipeline-orchestrator.md` — update "Agent idle signal" paragraph to reference `.fab-runtime.yaml`

---

## Execution Order

- T001 is independent setup
- T002 and T003 are independent of each other (different files)
- T004 and T005 are independent of each other (different files)
- All phases are sequential: T001 → T002/T003 → T004/T005
