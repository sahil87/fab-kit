# Quality Checklist: Split Stage Display from Routing

**Change**: 260218-95xn-split-stage-display-from-routing
**Generated**: 2026-02-18
**Spec**: `spec.md`
**Re-reviewed**: 2026-02-18 (post-rework verification)

## Functional Completeness

- [x] CHK-001 `get_display_stage` function: returns first active stage, or last done, or `intake` with `pending`
- [x] CHK-002 `display-stage` CLI command: outputs `stage:state` colon-separated format
- [x] CHK-003 Preflight `display_stage` field: emits correct display stage name in YAML output
- [x] CHK-004 Preflight `display_state` field: emits correct state qualifier in YAML output
- [x] CHK-005 Changeman switch two-line format: `Stage:` shows display stage with state qualifier
- [x] CHK-006 Changeman switch two-line format: `Next:` shows routing stage with default command

## Behavioral Correctness

- [x] CHK-007 `get_current_stage` unchanged: routing logic produces same results as before
- [x] CHK-008 Existing preflight `stage` field unchanged: continues to represent routing stage

## Scenario Coverage

- [x] CHK-009 Display stage with active stage: returns the active stage (not the next pending)
- [x] CHK-010 Display stage after done, next pending: returns last done stage (not the next pending)
- [x] CHK-011 Display stage fresh change (all pending): returns `intake:pending`
- [x] CHK-012 Display stage all done: returns `hydrate:done`
- [x] CHK-013 Display stage review failed with apply active: returns `apply:active`
- [x] CHK-014 Changeman switch all-done case: `Next:` shows `/fab-archive` without stage prefix

## Edge Cases & Error Handling

- [x] CHK-015 `display-stage` with missing file: exits non-zero with error to stderr
- [x] CHK-016 `display-stage` with invalid file: handles gracefully

## Code Quality

- [x] CHK-017 Pattern consistency: new function follows naming/structure of existing stageman functions
- [x] CHK-018 No unnecessary duplication: reuses `get_progress_map` and existing helpers

## Documentation Accuracy

- [x] CHK-019 fab-status.md: documents display_stage/display_state usage and new output format
- [x] CHK-020 fab-switch.md: canonical output format updated to two-line Stage/Next

## Cross References

- [x] CHK-021 Stageman help text: `display-stage` listed in subcommands
- [x] CHK-022 Changeman: uses `$STAGEMAN display-stage` (CLI subprocess, not sourced function)

## Review Findings (Re-review)

- F-001 (must-fix): DISMISSED -- `change_dir` shell variable (preflight.sh:31) confirmed unchanged from main. The YAML output change (`fab/changes/$name` to `changes/$name`) is a separate spec-alignment fix, not related to F-001.
- F-002 (should-fix): VERIFIED FIXED -- all-done condition at changeman.sh:256 now requires three-part check: `routing_stage = "hydrate" && display_stage = "hydrate" && display_state = "done"`.
- F-003 (nice-to-have): No action needed (pre-existing design tension).

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-008 **N/A**: {reason}`
