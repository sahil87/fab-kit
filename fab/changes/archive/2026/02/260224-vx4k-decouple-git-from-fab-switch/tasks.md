# Tasks: Decouple Git from Fab Switch

**Change**: 260224-vx4k-decouple-git-from-fab-switch
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Remove Git from fab-switch

- [x] T001 Strip git branch logic from `fab/.kit/scripts/lib/changeman.sh` `cmd_switch()` — remove config reading for `git.enabled`/`git.branch_prefix` (lines 192-203), git repo detection (lines 210-212), branch checkout/create block (lines 214-231), `branch_status` variable, and `Branch:` output line (line 251-253)
- [x] T002 Update `fab/.kit/skills/fab-switch.md` — remove `## Branch Integration` section (lines 68-79), remove `--branch <name>` and `--no-branch-change` from arguments (lines 17-18), remove `--blank --branch` combination from deactivation flow (line 54), update Key Properties (`Modifies git state? → No`), remove git error handling row, add `/git-branch` hint logic for when `git.enabled` is true

## Phase 2: New /git-branch Command

- [x] T003 Create `fab/.kit/skills/git-branch.md` — new skill file with frontmatter (`name: git-branch`, `model_tier: fast`, `allowed-tools: Bash(git:*)`), arguments (`[change-name]` optional), behavior (read config → check git.enabled → resolve change → derive branch name → context-dependent prompts → checkout/create), output format, error handling, key properties

## Phase 3: Integration

- [x] T004 [P] Update `fab/.kit/skills/git-pr.md` — enhance Step 2 branch guard to suggest `/git-branch` when active change exists, add new Step 1b for branch mismatch nudge (resolve active change via `changeman.sh resolve`, compare branch name with/without prefix, show non-blocking note if mismatch, proceed with PR workflow)
- [x] T005 [P] Update `fab/.kit/scripts/pipeline/dispatch.sh` line 217 — change `"/fab-switch $CHANGE_ID --no-branch-change"` to `"/fab-switch $CHANGE_ID"`
- [x] T006 [P] Update `docs/specs/architecture.md` Git Integration section (lines 324-347) — reflect that `/fab-switch` no longer handles branches, document `/git-branch` as the branch management command, update the options table
- [x] T007 [P] Update `docs/specs/skills.md` — add `/git-branch` skill entry with description, arguments, behavior summary, and key properties

---

## Execution Order

- T001 blocks T002 (changeman.sh change must land before skill references it)
- T003 is independent of T001-T002 (new file, no dependencies)
- T004-T007 are all independent of each other (different files, no cross-dependencies)
- T004-T007 depend on T001-T003 being complete (need to know final behavior for docs/integration)
