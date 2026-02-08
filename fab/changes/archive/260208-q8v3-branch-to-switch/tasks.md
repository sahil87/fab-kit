# Tasks: Move Branch Integration from fab-new to fab-switch

**Change**: 260208-q8v3-branch-to-switch
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Remove branch field from templates and scripts

- [x] T001 [P] Remove `branch: {BRANCH}` line from `fab/.kit/templates/status.yaml`
- [x] T002 [P] Remove `branch` parsing and YAML output from `fab/.kit/scripts/fab-preflight.sh`
- [x] T003 [P] Replace `branch:` field display with live `git branch --show-current` in `fab/.kit/scripts/fab-status.sh` — read `fab/config.yaml` for `git.enabled`, omit Branch line when git disabled or not in a repo

## Phase 2: Update skill definitions

- [x] T004 [P] Update `fab/.kit/skills/fab-switch.md` — add branch integration step after writing `fab/current`, add `--branch <name>` argument, update context loading to include `config.yaml` for `git.enabled` and `git.branch_prefix`, add branch-related error handling, update Key Properties table
- [x] T005 [P] Update `fab/.kit/skills/fab-new.md` — remove Step 4 (Git Integration), remove `--branch` argument, remove `branch:` from `.status.yaml` example in Step 5, add internal `/fab-switch` call after proposal generation (before Step 9: Mark Proposal Complete), update output examples to remove Branch lines, update error handling table
- [x] T006 [P] Update `fab/.kit/skills/fab-discuss.md` — remove `Note: no branch: field` comment from `.status.yaml` example in Step 6, remove `branch:` line from the YAML block
- [x] T007 [P] Update `fab/.kit/skills/fab-status.md` — update behavior description to note the script now uses live git query instead of `.status.yaml` branch field
- [x] T008 [P] Update `fab/.kit/skills/_context.md` — remove `branch` from preflight YAML fields list in "Parse stdout YAML" step, remove "0 for branch-on-main" from fab-new's interruption budget in Skill-Specific Autonomy table

## Phase 3: Update remaining skill references

- [x] T009 Scan all remaining skill files (`fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-apply.md`, `fab-review.md`, `fab-archive.md`, `fab-clarify.md`, `fab-init.md`) for references to `branch` in preflight output or `.status.yaml` and update accordingly

## Phase 4: Update config.yaml

- [x] T010 Verify `git.branch_prefix` is present in `fab/config.yaml` under the `git:` section (add if missing, since `/fab-switch` now consumes it)

---

## Execution Order

- Phase 1 tasks (T001-T003) are all independent and can run in parallel
- Phase 2 tasks (T004-T008) are all independent and can run in parallel
- Phase 3 (T009) should run after Phase 2 to avoid conflicting edits
- Phase 4 (T010) is independent
