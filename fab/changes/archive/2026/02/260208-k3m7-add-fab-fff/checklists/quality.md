# Quality Checklist: Add fab-fff Full-Pipeline Command with Confidence Gating

**Change**: 260208-k3m7-add-fab-fff
**Generated**: 2026-02-08
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Confidence fields in `.status.yaml`: `confidence` block with certain, confident, tentative, unresolved, score fields exists and is populated by `/fab-new`
- [x] CHK-002 Confidence score formula: score computed correctly per formula (0 if unresolved > 0, else max(0, 5.0 - 0.1*confident - 1.0*tentative))
- [x] CHK-003 Confidence schema in `_context.md`: schema, formula, gate threshold, and lifecycle documented in `_context.md`
- [x] CHK-004 Status template: `fab/.kit/templates/status.yaml` exists with confidence block initialized to zeros and score 5.0
- [x] CHK-005 Manual skills recompute: `/fab-new`, `/fab-continue`, `/fab-clarify` all recompute confidence after invocation
- [x] CHK-006 Autonomous skills skip recompute: `/fab-ff` and `/fab-fff` do not update the confidence block
- [x] CHK-007 fab-fff pipeline: `/fab-fff` chains fab-ff → fab-apply → fab-review → fab-archive in sequence
- [x] CHK-008 Confidence gate: `/fab-fff` aborts when `confidence.score < 3.0` with actionable message
- [x] CHK-009 fab-fff review bail: `/fab-fff` stops immediately on review failure without interactive rework menu
- [x] CHK-010 fab-fff resumability: re-invoking `/fab-fff` skips done stages and resumes from first incomplete
- [x] CHK-011 fab-fff skill registration: `.claude/skills/fab-fff/SKILL.md` symlink exists and resolves correctly
- [x] CHK-012 Preflight confidence: `fab-preflight.sh` emits confidence block in YAML output
- [x] CHK-013 --auto removal: `/fab-ff` no longer accepts or documents `--auto` flag
- [x] CHK-014 auto-guess marker removal: no skill produces `<!-- auto-guess -->` markers
- [x] CHK-015 auto-guess references removed from `_context.md`: artifact markers table and autonomy table updated
- [x] CHK-016 auto-guess scanning removed from `/fab-clarify`: neither suggest nor auto mode scans for auto-guess markers
- [x] CHK-017 auto-guess soft gate removed from `/fab-apply`: no marker scan before implementation

## Behavioral Correctness

- [x] CHK-018 fab-ff default mode unchanged: frontload questions, interleaved auto-clarify, bail on blockers all work as before
- [x] CHK-019 fab-clarify still scans `<!-- assumed -->` markers: Tentative assumption scanning unaffected by auto-guess removal

## Removal Verification

- [x] CHK-020 No `--auto` references remain in `fab-ff.md`: search confirms zero occurrences
- [x] CHK-021 No `auto-guess` references remain in `_context.md`, `fab-clarify.md`, `fab-apply.md`: search confirms zero occurrences
- [x] CHK-022 No `fab-ff --auto` references remain in centralized docs: search across `fab/docs/` confirms zero occurrences. Note: `auto-guess` appeared in `context-loading.md` (not in Affected Docs) — fixed during review. Historical references in changelog entries and rejected-alternative notes are acceptable.

## Scenario Coverage

- [x] CHK-023 Scenario: confidence below threshold — `/fab-fff` aborts with score and threshold in message
- [x] CHK-024 Scenario: confidence at threshold (3.0) — `/fab-fff` passes gate and proceeds
- [x] CHK-025 Scenario: all decisions Certain — score is 5.0
- [x] CHK-026 Scenario: unresolved > 0 — score is 0.0
- [x] CHK-027 Scenario: fab-ff bails during fab-fff — pipeline stops, reports blocking issues
- [x] CHK-028 Scenario: resume after interruption — fab-fff skips done stages

## Edge Cases & Error Handling

- [x] CHK-029 Missing confidence block in `.status.yaml` (legacy change): preflight emits zeros + 5.0; fab-fff treats missing block as score 0 (refuses to run)
- [x] CHK-030 Review failure during fab-fff: pipeline stops with failure details and actionable suggestion

## Documentation Accuracy

- [x] CHK-031 `planning-skills.md` documents `/fab-fff` and removes `/fab-ff --auto`
- [x] CHK-032 `change-lifecycle.md` includes confidence fields in `.status.yaml` schema
- [x] CHK-033 `clarify.md` removes auto-guess references and documents confidence recomputation
- [x] CHK-034 `execution-skills.md` removes auto-guess soft gate from `/fab-apply`

## Cross References

- [x] CHK-035 Next Steps table in `_context.md` includes `/fab-fff` entry
- [x] CHK-036 Skill-Specific Autonomy Levels table has `/fab-fff` column replacing `fab-ff --auto`

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
