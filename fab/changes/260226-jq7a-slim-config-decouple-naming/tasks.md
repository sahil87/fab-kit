# Tasks: Slim Config & Decouple Naming

**Change**: 260226-jq7a-slim-config-decouple-naming
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup — Templates & Config

- [ ] T001 Update `fab/.kit/templates/status.yaml` — add `issue_id: null` after `change_type`
- [ ] T002 [P] Update `fab/.kit/scaffold/fab/project/config.yaml` — remove `git`, `naming`; rename `rules` → `stage_directives` with all 6 stage placeholders; reduce comments
- [ ] T003 [P] Update `fab/project/config.yaml` (this project's config) — same structural changes as T002, preserving project-specific values

## Phase 2: Core — Skill Updates

- [ ] T004 Update `fab/.kit/skills/git-branch.md` — remove Step 1 `git.enabled` gate, remove `git.branch_prefix` from Step 4, simplify to: check git repo → resolve change → branch = change name
- [ ] T005 [P] Update `fab/.kit/skills/git-pr.md` — add `issue_id` read from `.status.yaml`, include in PR title when non-null; remove `git.branch_prefix` read
- [ ] T006 [P] Update `fab/.kit/skills/fab-new.md` — Step 1: generate slug without Linear ID prefix; after Step 3: write `issue_id` to `.status.yaml` if Linear ticket detected
- [ ] T007 [P] Update `fab/.kit/skills/fab-switch.md` — remove `git.enabled` conditional from tip line; always show `/git-branch` hint
- [ ] T008 [P] Update `fab/.kit/skills/fab-status.md` — remove `git.enabled` conditional; always show current branch

## Phase 3: Integration — Migration, Spec, References

- [ ] T009 Create `fab/.kit/migrations/0.10.0-to-0.20.0.md` — migration for removing `git`, `naming`, renaming `rules` → `stage_directives`, stripping verbose comments
- [ ] T010 [P] Create `docs/specs/naming.md` — five naming conventions with pattern, example, encoding location
- [ ] T011 [P] Update `docs/specs/index.md` — add `naming.md` row
- [ ] T012 [P] Grep for remaining references to `config.rules`, `git.enabled`, `git.branch_prefix`, `naming.format` across all skill files and update to `stage_directives` or remove

## Phase 4: Polish — Memory Updates

- [ ] T013 Update `docs/memory/fab-workflow/configuration.md` — remove `git` and `naming` schema docs, rename `rules` → `stage_directives`, update lifecycle menu, update relationship section
- [ ] T014 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — simplify git integration section (always enabled, no prefix config), update `/git-branch` and `/fab-switch` descriptions
- [ ] T015 [P] Update `docs/memory/fab-workflow/templates.md` — document `issue_id` field in `.status.yaml` initial state section
- [ ] T016 [P] Update `docs/specs/skills.md` — update `/git-branch` and `/git-pr` skill specs
- [ ] T017 [P] Update `docs/specs/architecture.md` — remove git config references

---

## Execution Order

- T001, T002, T003 are independent (Phase 1 parallelizable)
- T004-T008 are independent of each other but depend on Phase 1 config being settled
- T009-T012 depend on skill changes being finalized (Phase 2)
- T013-T017 are independent of each other (Phase 4 parallelizable)
