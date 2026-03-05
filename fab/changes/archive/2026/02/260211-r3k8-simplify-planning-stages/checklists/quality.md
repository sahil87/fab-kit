# Quality Checklist: Simplify Planning Stages

**Change**: 260211-r3k8-simplify-planning-stages
**Generated**: 2026-02-11
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 6-Stage Pipeline: pipeline is brief → spec → tasks → apply → review → archive everywhere
- [x] CHK-002 Stage ID Renames: no remaining instances of `proposal` (as stage ID), `specs` (as stage ID), or `plan` (as stage ID) in active skill files, templates, or scripts
- [x] CHK-003 Plan Removal: no `plan.md` template, no Plan Generation Procedure in `_generation.md`, no plan-skip logic in `/fab-continue` or `/fab-ff`
- [x] CHK-004 Spec Absorbs Design Decisions: Spec Generation Procedure includes optional `## Design Decisions` guidance
- [x] CHK-005 /fab-new Lands on Brief: produces `brief.md`, sets `progress.brief: done`
- [x] CHK-006 /fab-discuss Dual Output: new change mode produces both `brief.md` and `spec.md`, marks both done
- [x] CHK-007 Directory Rename: `fab/specs/` renamed to `fab/design/`, all references updated
- [x] CHK-008 Template Rename: `proposal.md` → `brief.md`, `plan.md` deleted
- [x] CHK-009 Config Stages: `config.yaml` stages block uses new IDs, no plan entry
- [x] CHK-010 Status Template: `.status.yaml` template uses new progress keys

## Behavioral Correctness

- [x] CHK-011 /fab-continue After Spec: proceeds directly to tasks, no plan-skip prompt
- [x] CHK-012 /fab-ff Pipeline: generates spec → tasks (no plan step)
- [x] CHK-013 /fab-clarify Stages: accepts brief, spec, tasks; rejects proposal, specs, plan
- [x] CHK-014 /fab-continue Reset: accepts spec, tasks; rejects brief with correct message
- [x] CHK-015 Stage Numbering: /fab-status shows (N/6), correct number for each stage

## Removal Verification

- [x] CHK-016 Plan Template: `fab/.kit/templates/plan.md` does not exist
- [x] CHK-017 Plan Generation Procedure: no "Plan Generation" section in `_generation.md`
- [x] CHK-018 Plan Skip Logic: no "skip plan" or "plan warranted" logic in `/fab-continue` or `/fab-ff`
- [x] CHK-019 Old Specs Directory: `fab/specs/` directory no longer exists (renamed to `fab/design/`)

## Scenario Coverage

- [x] CHK-020 Config Stage Definitions scenario: stage IDs, requires chains, no plan entry
- [x] CHK-021 .status.yaml Progress Keys scenario: new keys, no old keys
- [x] CHK-022 /fab-discuss New Change Mode scenario: both artifacts produced, both stages done
- [x] CHK-023 Constitution References scenario: all `fab/design/` references
- [x] CHK-024 No Backward Compatibility scenario: no migration logic

## Documentation Accuracy

- [x] CHK-025 Planning-skills doc: reflects 3 planning stages, no plan references
- [x] CHK-026 Change-lifecycle doc: 6 stages, updated state vocabulary
- [x] CHK-027 Configuration doc: updated stages schema
- [x] CHK-028 Templates doc: brief.md section, no plan.md section
- [x] CHK-029 Kit-architecture doc: correct directory listing, correct template listing
- [x] CHK-030 Design-index doc: consistent "design" terminology

## Cross References

- [x] CHK-031 _context.md: all stage names updated, no stale references
- [x] CHK-032 fab/docs/index.md: correct doc listings after rename
- [x] CHK-033 fab/design/index.md: cross-reference to fab/docs/index.md valid
- [x] CHK-034 All centralized docs: no remaining `fab/specs/` references (except in archived changes)

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-archive`
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
- Archived changes (`fab/changes/archive/`) are historical records and excluded from this change
