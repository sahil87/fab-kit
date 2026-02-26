# Tasks: Drop Fast Model Tier

**Change**: 260226-85rg-drop-fast-model-tier
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Remove Frontmatter

- [x] T001 [P] Remove `model_tier: fast` line from `fab/.kit/skills/fab-switch.md`
- [x] T002 [P] Remove `model_tier: fast` line from `fab/.kit/skills/fab-help.md`
- [x] T003 [P] Remove `model_tier: fast` line from `fab/.kit/skills/fab-status.md`
- [x] T004 [P] Remove `model_tier: fast` line from `fab/.kit/skills/fab-setup.md`
- [x] T005 [P] Remove `model_tier: fast` line from `fab/.kit/skills/git-branch.md`

## Phase 2: Remove Config and Scaffold

- [x] T006 [P] Remove `model_tiers:` section (and its nested keys) from `fab/project/config.yaml`
- [x] T007 [P] Remove `model_tiers:` section (and its comment + nested keys) from `fab/.kit/scaffold/fab/project/config.yaml`

## Phase 3: Simplify Sync Script

- [x] T008 Remove section 3b (classify skills by model tier: `fast_skills` array, frontmatter parsing loop, validation) from `fab/.kit/sync/2-sync-workspace.sh` (lines 248-274)
- [x] T009 Remove section 3c (resolve fast-tier model: `claude_fast_model` variable, config lookup, fallback) from `fab/.kit/sync/2-sync-workspace.sh` (lines 413-423)
- [x] T010 Replace conditional Claude Code deployment block (lines 426-431) with plain `sync_agent_skills "Claude Code" "$repo_root/.claude/skills" "directory" "copy"`
- [x] T011 Remove `yaml_value` helper function from `fab/.kit/sync/2-sync-workspace.sh` (lines 34-52)

## Phase 4: Update Tests and References

- [x] T012 Update `setup()` fixture in `src/lib/sync-workspace/test.bats` — change `fab-status.md` to use plain frontmatter without `model_tier: fast` (lines 70-78)
- [x] T013 Remove test `fast-tier skill copy has model: instead of model_tier:` from `src/lib/sync-workspace/test.bats`
- [x] T014 Update test `capable-tier skill copy preserves content without model override` to verify plain copy for all skills
- [x] T015 Remove test `uses haiku fallback when config.yaml has no model_tiers` from `src/lib/sync-workspace/test.bats`
- [x] T016 Remove test `reads model_tiers from config.yaml when present` from `src/lib/sync-workspace/test.bats`
- [x] T017 Update `internal-skill-optimize.md` — remove `model_tier` from frontmatter preservation list (line 46)
- [x] T018 Run `bats src/lib/sync-workspace/test.bats` and verify all tests pass

## Phase 5: Migration

- [x] T019 Create migration `fab/.kit/migrations/0.21.0-to-0.22.0.md` — remove `model_tiers:` from existing projects' `config.yaml`
- [x] T020 Bump `fab/.kit/VERSION` to `0.22.0`

---

## Execution Order

- T001-T005 are independent (parallel)
- T006-T007 are independent (parallel)
- T008 before T009-T011 (section 3b removal first, then 3c and deployment)
- T012 before T018 (fixture update before test run)
- T013-T016 before T018 (test removals before verification)
- T018 is the final verification step
