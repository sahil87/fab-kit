# Tasks: Fix calc-score.sh Short-Form Path References

**Change**: 260218-hpzb-fix-calc-score-path-refs
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Replace 2 short-form `calc-score.sh` references in `fab/.kit/skills/fab-ff.md` (lines 14 and 28) with full path `fab/.kit/scripts/lib/calc-score.sh`
- [x] T002 [P] Replace 3 short-form `calc-score.sh` references in `fab/.kit/skills/_context.md` (lines 151, 279, 283) with full path `fab/.kit/scripts/lib/calc-score.sh`

## Phase 2: Verification

- [x] T003 Verify no remaining short-form `calc-score.sh` backtick references exist in `fab/.kit/skills/` — grep for backtick-enclosed patterns that don't include the full path (also found and fixed 1 additional occurrence in `_generation.md:50`)

---

## Execution Order

- T001 and T002 are independent (`[P]`)
- T003 depends on T001 and T002
