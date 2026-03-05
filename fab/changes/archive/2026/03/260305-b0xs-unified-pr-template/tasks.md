# Tasks: Unified PR Template

**Change**: 260305-b0xs-unified-pr-template
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Read the current `/git-pr` skill source at `fab/.kit/skills/git-pr.md` and identify the Tier 1 / Tier 2 branching logic in Step 3c, the title derivation branching in Step 3c step 2, and the PR Type Reference table

## Phase 2: Core Implementation

- [x] T002 Replace Step 3c step 2 title derivation — remove the type-based branching ("Fab-linked" vs "Lightweight"), unify to: if `changeman.sh resolve` succeeds AND `intake.md` exists → use intake heading; otherwise → use commit subject. Keep `{pr_title}` format unchanged in `fab/.kit/skills/git-pr.md`
- [x] T003 Replace Step 3c step 3 body generation — remove Tier 1 / Tier 2 branching, implement single template with conditional population: (a) Summary section from intake Why or commits/diff, (b) Changes section from intake subsections (omitted when no fab change), (c) Stats table with Type/Confidence/Checklist/Tasks/Review columns using `—` for unavailable fields, (d) Pipeline progress line with intake/spec as hyperlinks in `fab/.kit/skills/git-pr.md`
- [x] T004 Clean up PR Type Reference table at bottom of `fab/.kit/skills/git-pr.md` — remove "Fab Pipeline?" and "Template Tier" columns, keep only Type and Description

## Phase 3: Integration & Edge Cases

- [x] T005 Verify graceful degradation path in the unified template: when `changeman.sh resolve` fails, Stats table shows only Type populated with `—` for all other columns, Changes section omitted, Pipeline line omitted, no "housekeeping change" footer in `fab/.kit/skills/git-pr.md`
- [x] T006 Sync deployed copy: run `bash fab/.kit/scripts/fab-sync.sh` to propagate changes from `fab/.kit/skills/git-pr.md` to `.claude/skills/git-pr.md`

## Phase 4: Polish

- [x] T007 Update `docs/memory/fab-workflow/execution-skills.md` — revise the "Two-Tier PR Templates with Type Resolution" design decision entry to reflect unified template, update the PR type system paragraph, add changelog entry for this change

---

## Execution Order

- T001 blocks T002, T003, T004
- T002 and T003 are sequential (both modify Step 3c, T002 is title, T003 is body)
- T004 is independent of T002/T003 (different section of file)
- T005 depends on T003 (verifies the template logic)
- T006 depends on T002, T003, T004 (all source changes complete before sync)
- T007 is independent of implementation tasks
