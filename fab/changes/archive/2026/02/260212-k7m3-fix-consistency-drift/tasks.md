# Tasks: Fix consistency drift between design, docs, and implementation

**Change**: 260212-k7m3-fix-consistency-drift
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Implementation Files

- [x] T001 Fix `fab/.kit/templates/brief.md` — change header from `# Proposal:` to `# Brief:`, change DEFERRED timing from "before tasks" to "during spec"
- [x] T002 [P] Fix `fab/.kit/skills/_generation.md` — change "proposal" to "brief" in Spec Generation step 2 (line 18) and Tasks Generation step 2 (line 39)

## Phase 2: Design Specs

- [x] T003 Fix `fab/design/architecture.md` — change slug word count "2-4" to "2-6" in folder naming table (line 74), update Git Integration section to reference `/fab-switch` instead of `/fab-new` (lines 307+), remove `stage:` field from both .status.yaml examples (lines 151, 169), add `created_by` field to both examples
- [x] T004 [P] Fix `fab/design/glossary.md` — change "2–4 word slug" to "2–6 word slug" in Folder name format entry (line 116)
- [x] T005 Fix `fab/design/templates.md` — add `## Origin` section to brief template between metadata block and `## Why`, add archive index maintenance documentation

## Phase 3: Centralized Docs

- [x] T006 [P] Fix `fab/docs/fab-workflow/index.md` — change "docs and specs" to "docs and design" in backfill entry (line 18)
- [x] T007 [P] Fix `fab/docs/fab-workflow/planning-skills.md` — replace "all zeros" with accurate phrasing distinguishing count defaults (zero) from score default (5.0) (line 74)

---

## Execution Order

- All tasks are independent — no cross-task dependencies
- Within Phase 1, T001 and T002 touch different files and can run in parallel
- Within Phase 2, T003 and T005 are independent but T003 is larger; T004 is parallel
- Within Phase 3, T006 and T007 touch different files and can run in parallel
