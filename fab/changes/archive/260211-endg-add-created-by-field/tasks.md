# Tasks: Add `created_by` Attribution to Changes

**Change**: 260211-endg-add-created-by-field
**Spec**: `spec.md`
**Proposal**: `proposal.md`

## Phase 1: Template & Schema

- [x] T001 Add `created_by: {CREATED_BY}` field to `fab/.kit/templates/status.yaml` immediately after the `created:` line
- [x] T002 [P] Update `.status.yaml` template in `fab/specs/templates.md` — add `created_by` field after `created:` with field notes explaining its behavior (write-once, auto-detected from git, fallback to "unknown")

## Phase 2: Core Implementation

- [x] T003 [P] Update `/fab-new` skill in `fab/.kit/skills/fab-new.md` — add `created_by` to the `.status.yaml` initialization block in Step 3, and add instruction to populate from `git config user.name` with fallback to `"unknown"`
- [x] T004 [P] Update `/fab-discuss` skill in `fab/.kit/skills/fab-discuss.md` — add `created_by` to the `.status.yaml` initialization block in Step 6 (new change mode), with same git config population and fallback

## Phase 3: Display

- [x] T005 Update `fab/.kit/scripts/fab-status.sh` — parse `created_by` from `.status.yaml` and display as `Created by: {value}` between the `Change:` and `Branch:` lines. Omit the line entirely when the field is missing.

---

## Execution Order

- T001 and T002 are independent (parallel)
- T003 and T004 are independent (parallel), but should follow T001 (template defines the canonical field)
- T005 is independent of T003/T004
