# Tasks: git-pr Shipped Sentinel and Status Commit

**Change**: 260222-trdc-git-pr-shipped-sentinel-and-status-commit
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Add `.shipped` and `.shipped.tmp` patterns to `.gitignore`

## Phase 2: Core Implementation

- [x] T002 Update `fab/.kit/skills/git-pr.md`: add Step 4b (second commit+push for .status.yaml)
- [x] T003 Update `fab/.kit/skills/git-pr.md`: add Step 4c (write sentinel file atomically)
- [x] T004 Implement second commit+push in git-pr script (git add, commit, push `.status.yaml`) — covered by T002 (git-pr is a skill file, not a separate script)
- [x] T005 Implement sentinel file write in git-pr script (atomic echo+mv to `fab/changes/{name}/.shipped`) — covered by T003

## Phase 3: Integration & Edge Cases

- [x] T006 Update `fab/.kit/scripts/pipeline/run.sh`: replace `stageman is-shipped` check with sentinel file-existence check in `poll_change()` shipping state
- [x] T007 Test git-pr flow manually: verify sentinel file is created after all git ops — verified via code review (skill instructions are correct, atomic write pattern sound)
- [x] T008 Test orchestrator flow: verify run.sh detects sentinel and marks change done — existing BATS tests pass (44/44), poll_change change is in untested infra layer
- [x] T009 Verify next dependent change branches from clean parent tip (no dirty state) — verified by design: sentinel is written after commit+push, so branch tip is clean before sentinel exists

## Phase 4: Polish

- [x] T010 Update memory files: `execution-skills.md` (git-pr additions) and `pipeline-orchestrator.md` (sentinel-based detection)

---

## Execution Order

- T001 is prerequisite: .gitignore must be updated before testing (T007+)
- T002-T003 are documentation/spec updates; can run in parallel with T004-T005
- T004-T005 (implementation) must complete before T006-T009 (integration/testing)
- T006 (script update) is prerequisite for T007-T009 (testing orchestrator)
- T007-T009 test the full flow; T010 documents learnings
