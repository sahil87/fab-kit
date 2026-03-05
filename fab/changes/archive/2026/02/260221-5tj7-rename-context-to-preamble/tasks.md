# Tasks: Rename _context.md to _preamble.md

**Change**: 260221-5tj7-rename-context-to-preamble
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: File Rename

- [x] T001 Rename `fab/.kit/skills/_context.md` to `fab/.kit/skills/_preamble.md` via `git mv`

## Phase 2: Skill File Reference Updates

- [x] T002 [P] Update instruction line in `fab/.kit/skills/_preamble.md` (self-reference in opening blockquote)
- [x] T003 [P] Update instruction line in `fab/.kit/skills/_generation.md`
- [x] T004 [P] Update instruction line in `fab/.kit/skills/fab-new.md`
- [x] T005 [P] Update instruction line in `fab/.kit/skills/fab-continue.md`
- [x] T006 [P] Update instruction line in `fab/.kit/skills/fab-ff.md`
- [x] T007 [P] Update instruction line in `fab/.kit/skills/fab-fff.md`
- [x] T008 [P] Update instruction line in `fab/.kit/skills/fab-clarify.md`
- [x] T009 [P] Update instruction line in `fab/.kit/skills/fab-switch.md`
- [x] T010 [P] Update instruction line in `fab/.kit/skills/fab-setup.md`
- [x] T011 [P] Update instruction line in `fab/.kit/skills/fab-status.md`
- [x] T012 [P] Update instruction line in `fab/.kit/skills/fab-archive.md`
- [x] T013 [P] Update instruction line in `fab/.kit/skills/fab-discuss.md`
- [x] T014 [P] Update instruction line in `fab/.kit/skills/docs-hydrate-memory.md`
- [x] T015 [P] Update instruction line in `fab/.kit/skills/docs-hydrate-specs.md`
- [x] T016 [P] Update instruction line in `fab/.kit/skills/internal-retrospect.md`
- [x] T017 [P] Update instruction line in `fab/.kit/skills/internal-skill-optimize.md`

## Phase 3: Documentation Reference Updates

- [x] T018 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/context-loading.md`
- [x] T019 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/kit-architecture.md` (including directory tree listing)
- [x] T020 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/planning-skills.md`
- [x] T021 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/clarify.md`
- [x] T022 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/execution-skills.md`
- [x] T023 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/model-tiers.md`
- [x] T024 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/change-lifecycle.md`
- [x] T025 [P] Update all `_context.md` path references in `docs/memory/fab-workflow/specs-index.md`
- [x] T026 [P] Update all `_context.md` path references in `docs/specs/skills.md`
- [x] T027 [P] Update all `_context.md` path references in `docs/specs/glossary.md`

## Phase 4: Verification

- [x] T028 Verify no remaining references to `_context.md` in live files (excluding `fab/changes/archive/` and `fab/changes/260221-5tj7-rename-context-to-preamble/`)

---

## Execution Order

- T001 blocks all other tasks (file must exist at new path before references are updated)
- T002-T017 are all parallel (independent skill file edits)
- T018-T027 are all parallel (independent doc file edits)
- T028 depends on all previous tasks
