# Tasks: Explicit Change Targeting for Workflow Commands

**Change**: 260213-w4k9-explicit-change-targeting
**Spec**: `spec.md`
**Brief**: `brief.md`

## Phase 1: Core Script Changes

- [x] T001 Modify `fab/.kit/scripts/fab-preflight.sh` — add optional `$1` override argument with case-insensitive substring matching against `fab/changes/` folders (excluding `archive/`). When `$1` is provided: scan folders, match (exact → single partial → ambiguous error → no-match error), validate the matched change directory and `.status.yaml`, emit YAML as before. When `$1` is absent: fall back to existing `fab/current` reading. Never modify `fab/current`.
- [x] T002 Modify `fab/.kit/scripts/fab-status.sh` — add optional `$1` override argument with the same matching logic as preflight. When provided, resolve the change from `$1` instead of reading `fab/current`. Display status for the targeted change.

## Phase 2: Skill File Updates

- [x] T003 [P] Update `fab/.kit/skills/fab-continue.md` — add `[change-name]` to the Arguments section. Document disambiguation: stage names (`brief`, `spec`, `tasks`, `apply`, `review`, `archive`) are recognized first; any other argument is a change-name override. Both can coexist. Update the Pre-flight Check section to pass the change-name argument to `fab-preflight.sh` when present.
- [x] T004 [P] Update `fab/.kit/skills/fab-ff.md` — add `[change-name]` to the Arguments section. Update the preflight invocation to pass the argument when present.
- [x] T005 [P] Update `fab/.kit/skills/fab-fff.md` — add `[change-name]` to the Arguments section. Update the preflight invocation to pass the argument when present.
- [x] T006 [P] Update `fab/.kit/skills/fab-clarify.md` — add `[change-name]` to the Arguments section. Update the preflight invocation to pass the argument when present.
- [x] T007 [P] Update `fab/.kit/skills/fab-status.md` — add `[change-name]` to the Arguments section. Update the status script invocation to pass the argument when present.

## Phase 3: Shared Context Documentation

- [x] T008 Update `fab/.kit/skills/_context.md` — in the "Change Context" section, update the preflight invocation pattern from `fab/.kit/scripts/fab-preflight.sh` to `fab/.kit/scripts/fab-preflight.sh [change-name]`. Add a brief note explaining the override: when a change-name is passed, preflight resolves against it instead of `fab/current`, transiently and without modifying the pointer file.

---

## Execution Order

- T001 blocks T003-T008 (script must be ready before skill files reference new argument)
- T002 blocks T007 (status script must be ready before status skill references new argument)
- T003-T007 are independent [P] tasks within Phase 2
