# Tasks: Restructure config.yaml

**Change**: 260218-bb93-restructure-config-yaml
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Create Scaffold Templates

- [x] T001 [P] Create scaffold template `fab/.kit/scaffold/context.md` — markdown structure with placeholder guidance for tech stack, conventions, architecture, and monorepo sections
- [x] T002 [P] Create scaffold template `fab/.kit/scaffold/code-quality.md` — markdown structure with `## Principles`, `## Anti-Patterns`, `## Test Strategy` sections and example content matching the current scaffold's commented-out `code_quality:` examples

## Phase 2: Update Config Files

- [x] T003 Update `fab/.kit/scaffold/config.yaml` — remove `context:`, `stages:`, `code_quality:` sections; add `model_tiers:` section with defaults and comments; update header comment to list all companion files
- [x] T004 Update `fab/config.yaml` (this project) — remove `context:`, `stages:`, `code_quality:` sections; add `model_tiers:` section with this project's defaults; update header comment
- [x] T005 Create `fab/context.md` for this project — extract the current `context:` value from `fab/config.yaml` into markdown format

## Phase 3: Update Scripts

- [x] T006 Delete `fab/.kit/model-tiers.yaml`
- [x] T007 Update `fab/.kit/sync/2-sync-workspace.sh` — remove model-tiers.yaml pre-flight check (lines 20-23), change `yaml_value` to read from `fab/config.yaml` `model_tiers.fast.claude`, remove override logic, add hardcoded `haiku` fallback when config.yaml has no `model_tiers` section or doesn't exist

## Phase 4: Update Skill Files

- [x] T008 [P] Update `fab/.kit/skills/_context.md` — add `fab/context.md` and `fab/code-quality.md` to Always Load layer (Section 1), mark both as optional (no error if missing), update file count and exception notes
- [x] T009 [P] Update `fab/.kit/skills/fab-continue.md` — change all `code_quality` references from config.yaml to `fab/code-quality.md` in apply stage (pattern extraction) and review stage (code quality check)
- [x] T010 [P] Update `fab/.kit/skills/_generation.md` — change Checklist Generation Procedure to read `code_quality` from `fab/code-quality.md` instead of `fab/config.yaml`
- [x] T011 Update `fab/.kit/skills/fab-setup.md` — remove `stages`, `code_quality`, `context` from valid config sections list; add `model_tiers`; add bootstrap steps for `fab/context.md` and `fab/code-quality.md` creation; update config menu numbering

## Phase 5: Migration and Finalization

- [x] T012 Create migration file `fab/.kit/migrations/0.7.0-to-0.8.0.md` — handles extracting context to context.md, code_quality to code-quality.md, removing stages, adding model_tiers, cleaning config.yaml
- [x] T013 Bump `fab/.kit/VERSION` from `0.7.0` to `0.8.0`

---

## Execution Order

- T001, T002 are independent (Phase 1 parallelizable)
- T003 must complete before T004 (scaffold informs project config structure)
- T005 depends on reading the current `context:` value from config before T004 removes it — **execute T005 before T004**
- T006 blocks T007 (delete file before updating references to it)
- T008, T009, T010 are independent (Phase 4 parallelizable)
- T011 depends on understanding T003 changes (sections list matches scaffold)
- T012 depends on all prior phases (migration validates the final state)
- T013 is the final task
