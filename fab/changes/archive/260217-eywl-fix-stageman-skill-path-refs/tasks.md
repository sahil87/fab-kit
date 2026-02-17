# Tasks: Fix Stageman Skill Path References

**Change**: 260217-eywl-fix-stageman-skill-path-refs
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Path Replacements

- [x] T001 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/fab-continue.md` (10 occurrences)
- [x] T002 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/fab-ff.md` (8 occurrences)
- [x] T003 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/fab-fff.md` (9 occurrences)
- [x] T004 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/fab-clarify.md` (1 occurrence)
- [x] T005 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/_generation.md` (3 occurrences)
- [x] T006 [P] Replace all `lib/stageman.sh` with `fab/.kit/scripts/lib/stageman.sh` in `fab/.kit/skills/fab-status.md` (1 occurrence)

## Phase 2: Preflight Short-Form Fixes

- [x] T007 [P] Replace short-form `lib/preflight.sh` with `fab/.kit/scripts/lib/preflight.sh` in `fab/.kit/skills/_context.md` line 21 (1 occurrence — preserve existing full-form on line 27)
- [x] T008 [P] Replace short-form `lib/preflight.sh` with `fab/.kit/scripts/lib/preflight.sh` in `fab/.kit/skills/fab-status.md` line 41 (1 occurrence — preserve existing full-form on line 38)

## Phase 3: Verification

- [x] T009 Verify no remaining short-form `lib/stageman.sh` references exist in `fab/.kit/skills/`
- [x] T010 Verify no remaining short-form `lib/preflight.sh` references exist in `fab/.kit/skills/` (excluding intentional short-form prose references)
- [x] T011 Verify no files under `fab/.kit/scripts/` were modified

---

## Execution Order

- T001–T006 are all [P] and independent — can run in parallel
- T007–T008 are [P] and independent of each other
- T009–T011 depend on all prior tasks completing
