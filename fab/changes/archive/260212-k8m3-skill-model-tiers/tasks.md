# Tasks: Provider-Agnostic Model Tiers for Fab Skills

**Change**: 260212-k8m3-skill-model-tiers
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Setup

- [x] T001 Create mapping file at `fab/.kit/model-tiers.yaml` with `fast` and `capable` tier definitions, Claude Code model entries, and commented placeholders for OpenCode/Codex

## Phase 2: Core Implementation

- [x] T002 [P] Add `model_tier: fast` to frontmatter in `fab/.kit/skills/fab-help.md` (after `description` field)
- [x] T003 [P] Add `model_tier: fast` to frontmatter in `fab/.kit/skills/fab-status.md` (after `description` field)
- [x] T004 [P] Add `model_tier: fast` to frontmatter in `fab/.kit/skills/fab-switch.md` (after `description` field)
- [x] T005 [P] Add `model_tier: fast` to frontmatter in `fab/.kit/skills/fab-init.md` (after `description` field)
- [x] T006 Update `fab/.kit/scripts/fab-setup.sh`: add YAML frontmatter parsing function to extract `model_tier` from skill files, load `fab/.kit/model-tiers.yaml`, merge with optional `model_tiers:` section from `fab/config.yaml`, and generate agent files in `.claude/agents/` (and `.agents/skills/` for Codex) for fast-tier skills with translated `model:` field

## Phase 3: Integration & Edge Cases

- [x] T007 Add strict error handling to `fab/.kit/scripts/fab-setup.sh`: exit non-zero with descriptive stderr messages for missing `model-tiers.yaml`, unrecognized `model_tier` values, missing platform mappings, and unparseable frontmatter
- [x] T008 Run `fab/.kit/scripts/fab-setup.sh` and verify: (a) fast-tier skills have both skill symlinks and agent files, (b) capable skills have symlinks only and no agent files, (c) agent file frontmatter contains `model: haiku` not `model_tier: fast`

## Phase 4: Documentation

- [x] T009 [P] Create `fab/docs/fab-workflow/model-tiers.md`: tier naming scheme, selection criteria table, full skill classification audit, mapping file format, how to add providers, how to select tiers for new skills
- [x] T010 [P] Update `fab/docs/fab-workflow/kit-architecture.md`: add `model-tiers.yaml` to directory structure listing, document dual deployment strategy, update "Agent Integration via Symlinks" section
- [x] T011 [P] Update `fab/docs/fab-workflow/templates.md`: document `model_tier` frontmatter field with valid values and default behavior
- [x] T012 Update `fab/docs/fab-workflow/index.md`: add `model-tiers` entry to the domain docs table

---

## Execution Order

- T001 blocks T006 (mapping file must exist before setup script reads it)
- T002-T005 block T006 (frontmatter must be tagged before setup script parses it)
- T006 blocks T007 (core logic before error handling refinement)
- T007 blocks T008 (error handling complete before integration test)
- T009-T011 are independent of each other but depend on T008 (docs reflect verified behavior)
- T012 depends on T009 (index references the new doc)
