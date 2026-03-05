# Tasks: Stage Transition Logging

**Change**: 260305-4szh-stage-transition-logging
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add `transition` subcommand to `fab/.kit/scripts/lib/logman.sh` — new case branch with signature `transition <change> <stage> <action> [from] [reason] [driver]`, JSON construction with conditional `from`/`reason`/`driver` fields, change resolution via `resolve_change_dir()`
- [x] T002 Extend `_apply_metrics_side_effect` in `fab/.kit/scripts/lib/statusman.sh` — add `from` and `reason` parameters (positions 5 and 6), emit `logman.sh transition` call in the `active` case after `iterations` increment, using `enter` when `iterations==1` and `re-entry` when `iterations>1`, derive change folder from tmpfile parent directory, best-effort call (`2>/dev/null || true`)
- [x] T003 Extend `event_start` and `event_reset` in `fab/.kit/scripts/lib/statusman.sh` — accept optional `[from] [reason]` parameters, propagate to `_apply_metrics_side_effect`. Update CLI dispatch for `start` (accept 3-6 args) and `reset` (accept 3-6 args)
- [x] T004 Propagate empty `from`/`reason` in `event_finish` in `fab/.kit/scripts/lib/statusman.sh` — the auto-activate call to `_apply_metrics_side_effect` for the next stage passes empty `from` and `reason` (forward flow, always `enter`)
- [x] T005 Update `show_help()` in `fab/.kit/scripts/lib/logman.sh` — add `transition` subcommand to USAGE, SUBCOMMANDS, and EXAMPLES sections
- [x] T006 Update `show_help()` in `fab/.kit/scripts/lib/statusman.sh` — add `[from] [reason]` to `start` and `reset` CLI signatures in the help text

## Phase 2: Documentation

- [x] T007 [P] Update `fab/.kit/skills/_scripts.md` — add `logman.sh transition` to subcommands list, add entry to callers table (`statusman.sh _apply_metrics_side_effect` → `logman.sh transition`), add `[from] [reason]` to `start`/`reset` in the statusman key subcommands table
- [x] T008 [P] Update `docs/memory/fab-workflow/change-lifecycle.md` — add `stage-transition` as 4th event type in Event History section (with enter and re-entry examples), update "Three event types" → "Four event types", add `iterations` semantics clarification to `stage_metrics` description, add canonical review result values note to review event documentation
- [x] T009 [P] Update `docs/memory/fab-workflow/kit-scripts.md` — add `logman.sh transition` to subcommands list, add callers table entry for `statusman.sh` → `logman.sh transition`, document `iterations` semantics and relationship to transition `action` field, add canonical review result values note to `.history.jsonl` format section

---

## Execution Order

- T001 and T002 are independent (logman vs statusman), but T003 and T004 depend on T002 (they modify the same functions that T002 extends)
- T005 depends on T001 (help text reflects new subcommand)
- T006 depends on T003 (help text reflects new CLI parameters)
- T007, T008, T009 are independent of each other and can run after T001-T006
