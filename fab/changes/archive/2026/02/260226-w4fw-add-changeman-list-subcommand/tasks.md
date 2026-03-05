# Tasks: Add changeman.sh list Subcommand

**Change**: 260226-w4fw-add-changeman-list-subcommand
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `cmd_list` function to `fab/.kit/scripts/lib/changeman.sh` — scan `fab/changes/` excluding `archive/`, call `stageman.sh display-stage` per change, output `name:display_stage:display_state` per line. Handle missing `.status.yaml` with `name:unknown:unknown` + stderr warning. Handle missing `fab/changes/` with stderr error and exit 1. Support `--archive` flag to scan `fab/changes/archive/` instead.

## Phase 2: Integration

- [x] T002 Add `list` to CLI dispatch in `fab/.kit/scripts/lib/changeman.sh` — add case in the dispatch block and update `show_help()` with usage line and subcommand description.

## Phase 3: Testing

- [x] T003 Add tests for `changeman.sh list` in the existing test infrastructure — cover: multiple changes, single change, no changes, missing `.status.yaml`, `--archive` flag, missing `fab/changes/` directory.

---

## Execution Order

- T001 blocks T002 (function must exist before dispatch wires it)
- T001 blocks T003 (function must exist before testing)
