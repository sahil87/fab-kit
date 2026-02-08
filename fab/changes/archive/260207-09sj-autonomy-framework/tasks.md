# Tasks: Add SRAD Autonomy Framework to Planning Skills

**Change**: 260207-09sj-autonomy-framework
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Foundation

<!-- SRAD framework in _context.md — everything else depends on this -->

- [x] T001 Add SRAD scoring table, confidence grades definition, and critical rule to `fab/.kit/skills/_context.md` — new section after the existing "Next Steps Convention" section
- [x] T002 Add 2-3 worked examples to the SRAD section in `fab/.kit/skills/_context.md` demonstrating how the four dimensions produce a confidence grade
- [x] T003 Add `<!-- assumed: ... -->` marker convention documentation to `fab/.kit/skills/_context.md` alongside the existing `<!-- auto-guess: ... -->` references
- [x] T004 Add Assumptions Summary Block format specification to `fab/.kit/skills/_context.md` — the standard `## Assumptions` table format that all skills use

## Phase 2: Core Skill Updates

<!-- Update each skill file. These are independent since each targets a different file. -->

- [x] T005 [P] Update `fab/.kit/skills/fab-new.md` — replace the current Step 7 (Clarifying Questions) with SRAD-based question selection (up to 3 Unresolved decisions with lowest R+A)
- [x] T006 [P] Update `fab/.kit/skills/fab-new.md` — change Step 4 (Git Integration) to auto-create branch when on main/master instead of prompting. Preserve existing behavior for `--branch` and feature branch cases
- [x] T007 [P] Update `fab/.kit/skills/fab-new.md` — add Assumptions Summary to output section and add requirement to persist `## Assumptions` section in generated `proposal.md`
- [x] T008 [P] Update `fab/.kit/skills/fab-continue.md` — add SRAD-based question selection (1-2 per stage, up to 3), [NEEDS CLARIFICATION] count in output, Key Decisions block after plan generation, and Assumptions summary in output + artifact
- [x] T009 [P] Update `fab/.kit/skills/fab-ff.md` — add clarification that `--auto` skips frontloaded questions entirely; add plan-skip reasoning to output; add cumulative Assumptions summary; add per-artifact `## Assumptions` persistence
- [x] T010 [P] Update `fab/.kit/skills/fab-clarify.md` — add `<!-- assumed: ... -->` to the taxonomy scan in both suggest and auto modes; add question format guidance for assumed markers (current assumption as recommendation)

## Phase 3: Integration

<!-- Cross-cutting updates that wire the framework into adjacent skills -->

- [x] T011 Update `fab/.kit/skills/fab-apply.md` — add soft gate that checks for `<!-- auto-guess: ... -->` markers in `tasks.md` before implementation, warns with count, and prompts "continue? (y/n)"

## Phase 4: Documentation

<!-- Update centralized docs to reflect the changes -->

- [x] T012 [P] Update `fab/docs/fab-workflow/planning-skills.md` — add SRAD framework references, autonomy level definitions, and new design decision entries
- [x] T013 [P] Update `fab/docs/fab-workflow/clarify.md` — add `<!-- assumed: ... -->` marker scanning to requirements and taxonomy scan categories
- [x] T014 [P] Update `fab/docs/fab-workflow/context-loading.md` — add SRAD protocol to shared context conventions

---

## Execution Order

- T001 blocks T002, T003, T004 (SRAD table must exist before examples and conventions reference it)
- T004 blocks T005-T010 (Assumptions Summary format must be defined before skills reference it)
- T005-T010 are independent [P] — each modifies a different skill file
- T011 depends on T009 (fab-apply gate references the auto-guess markers defined in fab-ff updates)
- T012-T014 are independent [P] and can run after T005-T011 (docs reflect the skill changes)
