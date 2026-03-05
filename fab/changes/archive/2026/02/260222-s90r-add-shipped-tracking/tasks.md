# Tasks: Add Shipped Tracking

**Change**: 260222-s90r-add-shipped-tracking
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 [P] Add `shipped: []` field to `fab/.kit/templates/status.yaml` between `stage_metrics` and `last_updated`
- [x] T002 [P] Add `shipped` documentation section to `fab/.kit/schemas/workflow.yaml`

## Phase 2: Core Implementation

- [x] T003 Add `ship_url` function to `fab/.kit/scripts/lib/stageman.sh` — append URL to `shipped` array with deduplication, create key if missing, atomic write, update `last_updated`
- [x] T004 Add `is_shipped` function to `fab/.kit/scripts/lib/stageman.sh` — exit 0 if `shipped` array has >= 1 entry, exit 1 otherwise, no stdout
- [x] T005 Add `ship` and `is-shipped` CLI dispatch entries and help text to `fab/.kit/scripts/lib/stageman.sh`

## Phase 3: Integration

- [x] T006 Update `/git-pr` skill (`fab/.kit/skills/git-pr.md`) — add Step 4 to resolve active change via `changeman.sh resolve` and call `stageman.sh ship` with PR URL after creation; graceful skip on failure
- [x] T007 [P] Update `_preamble.md` state table (`fab/.kit/skills/_preamble.md`) — change hydrate row to route to `/git-pr` as default, `/fab-archive` as alternative
- [x] T008 [P] Update `default_command` in `fab/.kit/scripts/lib/changeman.sh` — map `hydrate` to `/git-pr` instead of `/fab-archive`

## Phase 4: Tests

- [x] T009 Add bats tests for `ship` and `is-shipped` subcommands in `src/lib/stageman/test.bats`

---

## Execution Order

- T003 and T004 are independent (both add functions to stageman.sh)
- T005 depends on T003 and T004 (dispatch calls the functions)
- T006 depends on T005 (git-pr calls stageman ship via CLI)
- T007 and T008 are independent of each other and of T003-T006
- T009 depends on T005 (tests exercise the CLI interface)
