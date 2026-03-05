# Tasks: Add Code Quality Layer

**Change**: 260215-r8k3-DEV-1024-code-quality-layer
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup — Independent File Updates

- [x] T001 [P] Add Code Quality section to checklist template in `fab/.kit/templates/checklist.md` — insert `## Code Quality` section with two baseline items (pattern consistency, no unnecessary duplication) between Edge Cases and Security sections. Use `CHK-{NNN}` placeholder IDs consistent with existing template style
- [x] T002 [P] Add apply-stage and review-stage source code loading steps to `fab/.kit/skills/_context.md` — add step 4a (apply: read neighboring files for pattern extraction) and step 4b (review: re-read modified files for consistency) to Section 4 (Source Code Loading)
- [x] T003 [P] Add `code_quality` as derivation source to Checklist Generation Procedure in `fab/.kit/skills/_generation.md` — update step 4 to include `code_quality` config-derived items and the two baseline items when no config exists
- [x] T004 [P] Add `code_quality` config support to `fab/.kit/skills/fab-init.md` — add `code_quality` to valid sections list in Config Arguments, add menu item 9 ("coding standards for apply/review") with Done renumbered to 10, add commented-out `code_quality` block to Config Create Mode template

## Phase 2: Core — `fab-continue.md` Modifications

- [x] T005 Add Pattern Extraction section and expand per-task guidance in Apply Behavior in `fab/.kit/skills/fab-continue.md` — insert Pattern Extraction subsection between Preconditions and Task Execution, add code_quality config integration note, expand Task Execution step 4 to 7-step sequence
- [x] T006 Add Code Quality Validation as step 6 to Review Behavior in `fab/.kit/skills/fab-continue.md` — insert step 6 (code quality check) after existing step 5 (memory drift check) in Validation Steps, covering naming, function size, error handling, utility reuse, and config-derived checks
- [x] T007 Add optional pattern capture as step 5 to Hydrate Behavior in `fab/.kit/skills/fab-continue.md` — insert between step 4 (run stageman set-state) and existing content, noting new implementation patterns in memory Design Decisions, with skip guidance for changes that follow existing patterns

---

## Execution Order

- Phase 1 tasks (T001–T004) are fully parallel — all edit different files
- Phase 2 tasks (T005–T007) edit the same file (`fab-continue.md`) and MUST execute sequentially
- T005 before T006 (Apply before Review, matching skill section order)
- T006 before T007 (Review before Hydrate, matching skill section order)
