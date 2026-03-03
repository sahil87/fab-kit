# Tasks: Update Underscore Skill References

**Change**: 260303-6b7c-update-underscore-skill-references
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Skill File Updates

- [x] T001 [P] Update `_preamble.md` self-reference at `fab/.kit/skills/_preamble.md` line 12 — change `./fab/.kit/skills/_preamble.md` to `fab/.kit/skills/_preamble.md`
- [x] T002 [P] Update `_preamble.md` `_scripts.md` reference in §1 Always Load — change `fab/.kit/skills/_scripts.md` reference to ensure no `./` prefix
- [x] T003 [P] Update top-of-file instruction in 11 skill files (`fab-ff.md`, `fab-archive.md`, `fab-setup.md`, `fab-clarify.md`, `fab-status.md`, `fab-new.md`, `fab-continue.md`, `fab-fff.md`, `docs-hydrate-memory.md`, `fab-discuss.md`, `docs-hydrate-specs.md`) — remove `./` prefix from `./fab/.kit/skills/_preamble.md`
- [x] T004 [P] Update `fab-switch.md` line 8 — standardize path to `fab/.kit/skills/_preamble.md` (remove `./` if present, verify format matches variant)
- [x] T005 [P] Update `internal-skill-optimize.md` line 21 — ensure `fab/.kit/skills/_preamble.md` has no `./` prefix

## Phase 2: Test Updates

- [x] T006 Update stale test in `src/lib/sync-workspace/test.bats` — replace the "skips _preamble.md partial" test (line 288) with assertions that `_preamble`, `_generation`, and `_scripts` directories ARE deployed in `.claude/skills/` with valid `SKILL.md` files

## Phase 3: Memory File Updates

- [x] T007 [P] Update `docs/memory/fab-workflow/context-loading.md` — ensure any full-path references to underscore files use `fab/.kit/skills/` without `./` prefix
- [x] T008 [P] Update `docs/memory/fab-workflow/kit-architecture.md` — update full-path references and document underscore file deployment in the skills directory structure/deployment sections
- [x] T009 [P] Update `docs/memory/fab-workflow/planning-skills.md` — ensure any full-path references use `fab/.kit/skills/` without `./` prefix
- [x] T010 [P] Update `docs/memory/fab-workflow/execution-skills.md` — ensure any full-path references use `fab/.kit/skills/` without `./` prefix

---

## Execution Order

- T001–T005 are independent and parallelizable
- T006 is independent
- T007–T010 are independent and parallelizable
- No cross-phase dependencies — phases can overlap
