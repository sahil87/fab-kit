# Tasks: Formalize Assumptions Tables & Fix Scoring Pipeline

**Change**: 260214-m3w7-formalize-assumptions-scoring
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Foundation — Templates & Rules

- [x] T001 [P] Add formalized `## Assumptions` section to `fab/.kit/templates/brief.md` — 5-column table header (`| # | Grade | Decision | Rationale | Scores |`), HTML comment explaining state-transfer purpose, placeholder row, summary line template
- [x] T002 [P] Add formalized `## Assumptions` section to `fab/.kit/templates/spec.md` — 5-column table header, HTML comment noting this is the sole `calc-score.sh` scoring source, placeholder row, summary line template
- [x] T003 [P] Update `fab/.kit/skills/_context.md` — (a) Confidence Grades table: Certain→"Noted in Assumptions summary", Unresolved→"Asked as question AND noted in Assumptions summary"; (b) Assumptions Summary Block rules: replace "Only include Confident and Tentative" with "Include all four grades"; (c) Summary line format: `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved)`; (d) Update the Scores column description from optional to required

## Phase 2: Core Implementation — Script & Skill Prompts

- [x] T004 Fix `fab/.kit/scripts/lib/calc-score.sh` — (a) Remove `brief_file` variable and `parse_assumptions "$brief_file"` call; (b) Fix AWK column index from `cols[4]` to `cols[6]`; (c) Remove `has_scores` detection logic — always extract `cols[6]`, always output `grade|scores` format; (d) Add `unresolved)` case to grade counting switch, remove hardcoded `unresolved=0`; (e) Remove implicit Certain carry-forward block (lines 179-195); (f) Replace `total_certain` with `table_certain` directly; (g) Ensure empty `cols[6]` gracefully skips dimension aggregation (transition compatibility)
- [x] T005 [P] Update `fab/.kit/skills/fab-new.md` — Step 5 item 8 (Assumptions section): specify all four SRAD grades with required Scores column; add guidance that brief is sole context for downstream stages
- [x] T006 [P] Update `fab/.kit/skills/_generation.md` — Spec Generation Procedure Step 6: spec agent reads `brief.md`'s Assumptions as starting point (confirm/upgrade/override), adds new assumptions, includes all four grades with required Scores
- [x] T007 [P] Update `fab/.kit/skills/fab-ff.md` — Step 1: track all four SRAD grades in cumulative Assumptions summary

## Phase 3: Memory Updates

- [x] T008 [P] Update `docs/memory/fab-workflow/planning-skills.md` — (a) calc-score invocation note: "spec Assumptions table" not "brief + spec"; (b) SRAD/Assumptions references: all four grades; (c) Assumptions summary format: 4-grade summary line; (d) Add changelog entry
- [x] T009 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — (a) confidence field description: computed from spec.md only; (b) Add changelog entry

---

## Execution Order

- T001, T002, T003 are independent (parallel)
- T004 is independent of T001-T003 (different file, no content dependency)
- T005, T006, T007 are independent of each other and of T004 (parallel)
- T008, T009 depend on all prior tasks being conceptually stable but can execute in parallel with each other
