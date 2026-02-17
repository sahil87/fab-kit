# Tasks: Split Stage Display from Routing

**Change**: 260218-95xn-split-stage-display-from-routing
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `get_display_stage` function to `fab/.kit/scripts/lib/stageman.sh` — implement two-tier fallback: (1) first `active` stage, (2) last `done` stage, (3) fallback to `intake` with `pending`. Output `stage:state` format.
- [x] T002 Add `display-stage` CLI subcommand to `fab/.kit/scripts/lib/stageman.sh` — wire up in CLI dispatch section, validate arg count, call `get_display_stage`, add to help text.

## Phase 2: Integration

- [x] T003 Update `fab/.kit/scripts/lib/preflight.sh` — add `display_stage` and `display_state` fields to YAML output by calling `stageman.sh display-stage` and parsing the `stage:state` response.
- [x] T004 Update `fab/.kit/scripts/lib/changeman.sh` `cmd_switch` function — replace single `Stage:` line with two-line format using `display-stage` for display and `current-stage` for routing. Update `next_command` usage to produce `{stage} (via {command})` format.

## Phase 3: Skill Documentation

- [x] T005 [P] Update `fab/.kit/skills/fab-status.md` — document that the skill uses `display_stage`/`display_state` from preflight for the Stage line and derives the Next line with the new format.
- [x] T006 [P] Update `fab/.kit/skills/fab-switch.md` — update the canonical output format in the Output section to reflect the new two-line Stage/Next format from changeman.

---

## Execution Order

- T001 blocks T002 (function must exist before CLI wiring)
- T002 blocks T003 (CLI command must exist before preflight calls it)
- T003 and T004 are independent (changeman calls stageman directly, not via preflight)
- T005 and T006 are independent, parallelizable after T003/T004
