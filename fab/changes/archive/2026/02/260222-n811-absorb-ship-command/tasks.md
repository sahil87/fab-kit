# Tasks: Absorb Ship Command

**Change**: 260222-n811-absorb-ship-command
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create skill file `fab/.kit/skills/git-pr.md` with YAML frontmatter (`name: git-pr`, `description`, `model_tier: fast`, `allowed-tools: Bash(git:*), Bash(gh:*)`)

## Phase 2: Core Implementation

- [x] T002 Write the `/git-pr` skill prompt body in `fab/.kit/skills/git-pr.md` — state gathering, branch guard, commit (git add -A + message generation), push, PR creation (gh pr create --fill), progress output format, error handling, skip logic for already-done steps
- [x] T003 Update `fab/.kit/scripts/pipeline/run.sh` lines 366-367 — replace `"/changes:ship pr"` with `"/git-pr"` and update log message from `"Sending /changes:ship pr"` to `"Sending /git-pr"`

## Phase 3: Integration & Edge Cases

- [x] T004 Add `git-pr` to the `skill_to_group` mapping in `fab/.kit/scripts/fab-help.sh` so it appears under an appropriate group in help output

---

## Execution Order

- T001 blocks T002 (frontmatter must exist before body)
- T003 is independent
- T004 is independent
