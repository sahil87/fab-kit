# Tasks: Add fab-fff Full-Pipeline Command with Confidence Gating

**Change**: 260208-k3m7-add-fab-fff
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Setup

- [x] T001 Create `fab/.kit/templates/status.yaml` — stamp-out template with all existing fields plus `confidence: {certain: 0, confident: 0, tentative: 0, unresolved: 0, score: 5.0}` block
- [x] T002 Create `fab/.kit/skills/fab-fff.md` — new skill file (thin wrapper: confidence gate → fab-ff → fab-apply → fab-review → fab-archive, resumable via progress map, bails on review failure)
- [x] T003 Create `.claude/skills/fab-fff/` directory with `SKILL.md` symlink → `../../../fab/.kit/skills/fab-fff.md` (skill registration)

## Phase 2: Core Implementation

- [x] T004 Update `fab/.kit/skills/_context.md` — add Confidence Scoring section (schema, formula, gate threshold, lifecycle: which skills recompute, which don't). Remove `<!-- auto-guess -->` row from Artifact Markers table. Replace `fab-ff --auto` column with `fab-fff` column in Skill-Specific Autonomy Levels table. Add `/fab-fff` to Next Steps Lookup Table. Remove `/fab-ff --auto` row from Next Steps table.
- [x] T005 [P] Update `fab/.kit/skills/fab-ff.md` — remove `--auto` mode section, remove all `<!-- auto-guess -->` references, remove full-auto output examples. Keep only default mode (frontload questions, interleaved auto-clarify, bail on blockers).
- [x] T006 [P] Update `fab/.kit/skills/fab-new.md` — add confidence score computation step after proposal generation (count SRAD grades across proposal, compute score, write `confidence` block to `.status.yaml`)
- [x] T007 [P] Update `fab/.kit/skills/fab-continue.md` — add confidence score recomputation step after each artifact generation (re-count SRAD grades across all change artifacts, recompute score, update `.status.yaml`)
- [x] T008 [P] Update `fab/.kit/skills/fab-clarify.md` — add confidence score recomputation after each suggest-mode session. Remove `<!-- auto-guess: ... -->` scanning references from both suggest and auto mode descriptions. Keep `<!-- assumed: ... -->` scanning.
- [x] T009 [P] Update `fab/.kit/skills/fab-apply.md` — remove the Auto-Guess Soft Gate section entirely (the `<!-- auto-guess -->` marker scan, warning, and y/n prompt)
- [x] T010 Update `fab/.kit/scripts/fab-preflight.sh` — extract `confidence` block from `.status.yaml` and emit it in the YAML output (fields: certain, confident, tentative, unresolved, score). Handle missing confidence block gracefully (emit zeros + score 5.0 for backwards compat with existing changes).

## Phase 3: Integration & Edge Cases

- [x] T011 Verify `/fab-fff` handles edge cases: confidence block missing from `.status.yaml` (treat as score 0 — refuse to run), review failure (bail with actionable message), interruption mid-pipeline (resumable via progress map)
- [x] T012 Verify `/fab-ff` still works correctly after `--auto` removal — default mode unchanged, no dangling references to auto-guess or --auto flag

## Phase 4: Polish

- [x] T013 [P] Update `fab/docs/fab-workflow/planning-skills.md` — add `/fab-fff` documentation, remove `/fab-ff --auto` documentation, document `/fab-continue` confidence recomputation
- [x] T014 [P] Update `fab/docs/fab-workflow/change-lifecycle.md` — add confidence fields to `.status.yaml` schema section, add fab-fff path description
- [x] T015 [P] Update `fab/docs/fab-workflow/clarify.md` — remove `<!-- auto-guess -->` references from auto mode scanning, add confidence score recomputation behavior
- [x] T016 [P] Update `fab/docs/fab-workflow/execution-skills.md` — remove Auto-Guess Soft Gate section from `/fab-apply` documentation

---

## Execution Order

- T001 (template) should complete before T006 (fab-new uses template for new changes)
- T004 (_context.md) should complete before T005-T009 (skills reference _context.md conventions)
- T005-T009 are independent of each other (different skill files)
- T010 (preflight) should complete before T011 (verification depends on preflight emitting confidence)
- T011-T012 verify integration after core implementation
- T013-T016 are independent doc updates (different files)
