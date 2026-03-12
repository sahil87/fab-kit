# Tasks: PR Change Metadata

**Change**: 260312-9r3t-pr-change-metadata
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `linear_workspace` field to `fab/project/config.yaml` under the `project:` block as `linear_workspace: "weaver-ai"`

## Phase 2: Core Implementation

- [x] T002 Update `fab/.kit/skills/git-pr.md` Step 3c to add "Change" section generation — read `.status.yaml` `id` and `name`, read `linear_workspace` from config, format issue links, construct the Change table above Stats
- [x] T003 [P] Create migration file `fab/.kit/migrations/0.34.0-to-0.37.0.md` — check for existing `linear_workspace`, add commented-out field under `project:` block, print explanatory note

## Phase 3: Integration & Edge Cases

- [x] T004 Update `docs/specs/skills/SPEC-git-pr.md` to reflect the new "Change" section in the flow diagram and add `config.yaml` (linear_workspace) to the tools/reads table

---

## Execution Order

- T001 is independent setup
- T002 and T003 are parallelizable (different files)
- T004 depends on T002 (needs final skill content to document accurately)
