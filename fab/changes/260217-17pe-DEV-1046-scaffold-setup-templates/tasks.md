# Tasks: Scaffold Setup Templates

**Change**: 260217-17pe-DEV-1046-scaffold-setup-templates
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Create `fab/.kit/scaffold/config.yaml` — scaffold template with all config sections, `{PLACEHOLDER}` values for user-provided fields, actual defaults for everything else. Include full inline comments matching current `fab/config.yaml` header style
- [x] T002 [P] Create `fab/.kit/scaffold/constitution.md` — minimal skeleton with `# {Project Name} Constitution`, `## Core Principles` (single placeholder principle `### I. {Principle Name}`), `## Additional Constraints`, `## Governance` with `{DATE}` placeholders

## Phase 2: Core Implementation

- [x] T003 Update `fab/.kit/skills/fab-setup.md` Config Create Mode (lines ~186-259) — replace inline YAML block with instruction to read `fab/.kit/scaffold/config.yaml`, substitute placeholders with user-provided values, and write to `fab/config.yaml`
- [x] T004 Update `fab/.kit/skills/fab-setup.md` Constitution Create Mode (lines ~332-356) — replace inline markdown block with instruction to read `fab/.kit/scaffold/constitution.md`, generate principles from project context filling the skeleton, and write to `fab/constitution.md`

## Phase 3: Integration

- [x] T005 [P] Update `fab/.kit/skills/fab-setup.md` step 1c (lines ~70-80) — replace inline memory-index.md template with instruction to read from `fab/.kit/scaffold/memory-index.md` and write to `docs/memory/index.md`
- [x] T006 [P] Update `fab/.kit/skills/fab-setup.md` step 1d (lines ~84-104) — replace inline specs-index.md template with instruction to read from `fab/.kit/scaffold/specs-index.md` and write to `docs/specs/index.md`

---

## Execution Order

- T001 and T002 are independent — can run in parallel
- T003 depends on T001 (config scaffold must exist to reference it)
- T004 depends on T002 (constitution scaffold must exist to reference it)
- T005 and T006 are independent of T001-T004 (scaffold files already exist for indexes)
