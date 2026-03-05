# Tasks: Consistency Fixes from 260214 Audit

**Change**: 260214-eikh-consistency-fixes
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Spec File Corrections

<!-- All spec file edits are independent — one task per file. Each task handles ALL changes for that file. -->

- [x] T001 [P] Fix `fab/specs/skills.md` — cf01 (archive→hydrate stage refs), cf02 (/fab-hydrate→/docs-hydrate-memory ~12 occurrences), cf04 (TEMPLATES.md→templates.md in line 199), cf06 (retitle "Archive Behavior" heading at line 443), cf07 (add `/docs-reorg-memory` and `/docs-reorg-specs` sections), cf11 (remove `/fab-init-config`, `/fab-init-constitution`, `/fab-init-validate` sections at lines 119-150)
- [x] T002 [P] Fix `fab/specs/glossary.md` — cf01 (archive→hydrate in stage list line 23, Stage 6 entry line 36, Change entry line 13, pointer file entry line 19), cf02 (/fab-hydrate→/docs-hydrate-memory lines 12, 17, 45), cf14 (add `/docs-reorg-memory` and `/docs-reorg-specs` skill entries), cf15 (expand Hydration definition line 17 to cover dual-mode)
- [x] T003 [P] Fix `fab/specs/architecture.md` — cf02 (/fab-hydrate→/docs-hydrate-memory lines 392, 400, 419), cf04 (SKILLS.md→skills.md and fix anchor at line 400), cf05 (fab-hydrate.md→docs-hydrate-memory.md in directory listing line 24 and symlink example lines 361-362), cf08 (add batch scripts documentation section), cf09 (add internal skills to `.kit/skills/` directory listing)
- [x] T004 [P] Fix `fab/specs/overview.md` — cf01 (stage 6 "Archive"→"Hydrate" in stage table line 124), cf02 (/fab-hydrate→/docs-hydrate-memory lines 71, 75, 78, 81, 162)
- [x] T005 [P] Fix `fab/specs/user-flow.md` — cf02 (/fab-hydrate→/docs-hydrate-memory in flowchart node line 78)

## Phase 2: Schema & Memory Index Fixes

- [x] T006 Fix `fab/.kit/schemas/workflow.yaml` — cf03 (change `commands: [fab-apply]` to `commands: [fab-continue]` at line 90, change `commands: [fab-review]` to `commands: [fab-continue]` at line 100)
- [x] T007 Fix `fab/memory/index.md` — cf12 (remove `hydrate-design` and `design-index` from fab-workflow domain's memory file list at line 14)

---

## Execution Order

- All Phase 1 tasks (T001-T005) are independent and parallelizable
- Phase 2 tasks (T006-T007) are independent of each other and of Phase 1
- No blocking dependencies between any tasks
