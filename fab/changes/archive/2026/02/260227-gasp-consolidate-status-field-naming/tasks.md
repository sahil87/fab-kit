# Tasks: Consolidate Status Field Naming

**Change**: 260227-gasp-consolidate-status-field-naming
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Update `fab/.kit/templates/status.yaml`: replace `issue_id: null` with `issues: []`, replace `shipped: []` with `prs: []`
- [x] T002 [P] Update `fab/.kit/schemas/workflow.yaml`: replace `shipped:` section with `issues:` and `prs:` declarations

## Phase 2: Core Implementation

- [x] T003 Replace `ship_url()` and `is_shipped()` in `fab/.kit/scripts/lib/stageman.sh` with four new functions: `add_issue()`, `get_issues()`, `add_pr()`, `get_prs()`. Update section header from `# Shipped Tracking` to `# Issues & PRs`
- [x] T004 Update stageman CLI routes in `fab/.kit/scripts/lib/stageman.sh`: remove `ship` and `is-shipped` case branches, add `add-issue`, `get-issues`, `add-pr`, `get-prs` branches
- [x] T005 Update stageman help text in `fab/.kit/scripts/lib/stageman.sh`: remove `ship`/`is-shipped` entries, add `add-issue`/`get-issues`/`add-pr`/`get-prs` entries

## Phase 3: Integration & Edge Cases

- [x] T006 [P] Update `fab/.kit/skills/fab-new.md` Step 3: replace raw `yq -i '.issue_id = "DEV-988"'` with `stageman.sh add-issue` call
- [x] T007 [P] Update `fab/.kit/skills/git-pr.md`: Step 1 reads issues via `stageman.sh get-issues`, Step 3c joins with space for PR title, Step 4a uses `stageman.sh add-pr` instead of `ship`, Step 4b commit message updated
- [x] T008 [P] Update `fab/.kit/skills/git-pr.md`: rename `.shipped` sentinel to `.pr-done` in Step 4c
- [x] T009 [P] Update `fab/.kit/scripts/pipeline/run.sh`: rename `.shipped` sentinel variable and check to `.pr-done`, update log message and comment
- [x] T010 [P] Update `fab/.kit/skills/fab-archive.md` Step 1: reference `.pr-done` instead of `.shipped`
- [x] T011 [P] Update `.gitignore` (repo root) and `fab/.kit/scaffold/fragment-.gitignore`: change `fab/changes/**/.shipped` to `fab/changes/**/.pr-done`

## Phase 4: Polish

- [x] T012 Update `docs/specs/naming.md`: replace `issue_id` references with `issues` in PR title pattern and backlog entry pattern
- [x] T013 Create migration file `fab/.kit/migrations/0.22.0-to-0.24.0.md`: migrate active changes' `.status.yaml` (issue_id → issues[], shipped → prs[]), rename `.shipped` → `.pr-done` sentinels, bump version
- [x] T014 Run stageman test suite (`src/lib/stageman/test.bats`) and update tests for renamed functions/CLI routes

---

## Execution Order

- T001, T002 are independent setup (parallel)
- T003 blocks T004 and T005 (functions must exist before CLI routes and help text)
- T006–T011 are independent skill/script updates (parallel), but depend on T003–T005
- T012, T013 are documentation/migration, independent of each other
- T014 depends on T003–T005 (tests validate the new implementation)
