# Tasks: Redefine fab-ff and fab-fff Scope

**Change**: 260215-237b-DEV-1027-redefine-ff-fff-scope
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill Rewrites

<!-- Primary deliverables — rewrite both skill files. Independent of each other. -->

- [x] T001 [P] Rewrite `fab/.kit/skills/fab-fff.md` — full pipeline command with no confidence gate, frontloaded questions, interleaved auto-clarify, interactive rework on review failure. Update frontmatter description. Symlinked from `.claude/skills/fab-fff/SKILL.md`.
- [x] T002 [P] Rewrite `fab/.kit/skills/fab-ff.md` — fast-forward-from-spec command with confidence gate (`calc-score.sh --check-gate`), no frontloaded questions, minimal auto-clarify (tasks only), bail on review failure. Update frontmatter description. Symlinked from `.claude/skills/fab-ff/SKILL.md`.

## Phase 2: Context Updates

<!-- Depends on Phase 1 content being finalized. -->

- [x] T003 Update `fab/.kit/skills/_context.md` — Next Steps lookup table (swap /fab-ff and /fab-fff bail/interactive entries), Skill-Specific Autonomy Levels table (swap postures), Confidence Scoring gate reference (change from /fab-fff to /fab-ff)

## Phase 3: Memory File Updates

<!-- Depends on Phase 1 content being finalized. All three are independent. -->

- [x] T004 [P] Update `docs/memory/fab-workflow/planning-skills.md` — rewrite `/fab-ff` and `/fab-fff` requirement sections, update overview paragraph, update/add design decisions, add changelog entry
- [x] T005 [P] Update `docs/memory/fab-workflow/execution-skills.md` — update pipeline invocation note in Overview to reflect new behavioral differentiation, add changelog entry
- [x] T006 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — update "Full pipeline path" description to reference `/fab-fff` as ungated full pipeline and `/fab-ff` as confidence-gated from-spec command, add changelog entry

---

## Execution Order

- T001, T002 are independent (parallel)
- T003 depends on T001, T002 (references final skill content)
- T004, T005, T006 are independent of each other but depend on T001, T002
