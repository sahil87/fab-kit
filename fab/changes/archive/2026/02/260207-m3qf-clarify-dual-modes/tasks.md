# Tasks: fab-clarify Dual Modes + fab-ff Clarify Checkpoints

**Change**: 260207-m3qf-clarify-dual-modes
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 Back up current skill files and read existing content in full: `fab/.kit/skills/fab-clarify.md`, `fab/.kit/skills/fab-ff.md`, `fab/.kit/skills/_context.md`

## Phase 2: Core Implementation

- [x] T002 Rewrite `fab/.kit/skills/fab-clarify.md` — restructure into dual-mode skill: add Suggest Mode section (stage-scoped taxonomy scan, structured question format with recommendation + options table, one-question-at-a-time flow, max 5 questions cap, incremental artifact updates after each answer, early termination on "done"/"good"/"no more", clarifications audit trail under `## Clarifications > ### Session {date}`, coverage summary table at completion)
- [x] T003 Add Auto Mode section to `fab/.kit/skills/fab-clarify.md` — preserve current autonomous behavior as internal mode, define machine-readable result format (`{resolved, blocking, non_blocking}` counts), document that mode is selected by call context (user invocation = suggest, fab-ff internal call = auto), no `--suggest`/`--auto` flags
- [x] T004 Rewrite `fab/.kit/skills/fab-ff.md` — add interleaved auto-clarify pipeline to default mode: `spec → auto-clarify → plan-decision → auto-clarify → tasks → auto-clarify`, add bail-on-blocking logic (stop, report blocking issues, suggest `/fab:clarify` then `/fab:ff` to resume), ensure resumability (skip stages already `done` on re-invocation)
- [x] T005 Add `--auto` flag support to `fab/.kit/skills/fab-ff.md` — same pipeline as default but never stops for blockers, makes best-guess decisions, marks guesses with `<!-- auto-guess: {description} -->` markers, warns user in output listing all auto-guesses made

## Phase 3: Integration & Edge Cases

- [x] T006 Update `fab/.kit/skills/_context.md` — add `/fab:ff --auto` row to the Next Steps Lookup Table (same next step as `/fab:ff`: `Next: /fab:apply`)
- [x] T007 Verify cross-references between fab-clarify and fab-ff skill files: auto-clarify result format consumed by fab-ff matches what fab-clarify auto mode produces; bail/resume flow references correct stage names and `.status.yaml` fields
- [x] T008 Verify edge cases in fab-clarify suggest mode: taxonomy scan on artifact with zero gaps produces "No gaps found" output; early termination after 0 answered questions produces valid coverage summary; multiple `/fab:clarify` sessions accumulate audit trail entries (don't overwrite)

## Phase 4: Polish

- [x] T009 Create centralized doc `fab/docs/fab-workflow/clarify.md` — document the dual-mode fab-clarify skill (suggest mode behavior, auto mode contract, taxonomy scan, structured questions, coverage report, audit trail) following existing doc conventions in `fab/docs/fab-workflow/`
- [x] T010 Update `fab/docs/fab-workflow/index.md` — add `clarify` entry to the domain index table

---

## Execution Order

- T001 blocks all subsequent tasks (need current file content as baseline)
- T002 and T003 are sequential (T003 extends the file T002 restructures)
- T004 depends on T003 (fab-ff references auto mode result format defined in T003)
- T005 depends on T004 (extends the default mode pipeline with --auto variant)
- T006 is independent of T002-T005, can run after T001
- T007 depends on T002-T005 (cross-reference check needs both files complete)
- T008 depends on T002-T003 (edge cases are in fab-clarify)
- T009 depends on T002-T005 (doc describes final skill behavior)
- T010 depends on T009 (index references the new doc)
