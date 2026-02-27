# Tasks: Fix Kit Scripts

**Change**: 260227-yobi-fix-kit-scripts
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Script Fixes

- [x] T001 [P] Migrate history commands in `fab/.kit/scripts/lib/stageman.sh`: update CLI dispatch for `log-command`, `log-confidence`, `log-review` to call `resolve_change_arg`. Update `log_command()`, `log_confidence()`, `log_review()` to accept `.status.yaml` path and derive change dir via `dirname`. Update doc comments. Remove `resolve_change_dir()` (lines 924-936).
- [x] T002 [P] Fix spec gate in `fab/.kit/scripts/lib/calc-score.sh`: rewrite the spec gate path (lines 174-181) to parse `spec.md` inline and compute score on the fly, mirroring the intake gate pattern (lines 137-171). Remove the `grep '^ *score:'` reads from `.status.yaml`.

## Phase 2: Internal Caller Updates

- [x] T003 Update `fab/.kit/scripts/lib/calc-score.sh` line 329: change `"$change_dir"` to `"$status_file"` in the `log-confidence` call.
- [x] T004 [P] Update `fab/.kit/scripts/lib/changeman.sh` line 398: change `"$changes_dir/$folder_name"` to `"$folder_name"` in the `log-command` call. Line 486: change `"$changes_dir/$new_name"` to `"$new_name"`.
- [x] T005 [P] Update `show_help()` in `fab/.kit/scripts/lib/stageman.sh`: change history command signatures from `<change_dir>` to `<change>`, update examples to use change IDs.

## Phase 3: Skill Prompt & Doc Updates

- [x] T006 [P] Update `fab/.kit/skills/_generation.md` lines 95-97: change `<file>` placeholders to `<change>` in Checklist Generation Procedure stageman calls.
- [x] T007 [P] Update history command examples in `fab/.kit/skills/fab-ff.md`: replace `<change_dir>` with `<change>` in `log-command` and `log-review` references.
- [x] T008 [P] Update history command examples in `fab/.kit/skills/fab-fff.md`: replace `<change_dir>` with `<change>` in `log-command` and `log-review` references.
- [x] T009 [P] Update history command examples in `fab/.kit/skills/fab-continue.md`: replace `<change_dir>` with `<change>` in `log-command` and `log-review` references. Also fix `<file>` in `set-checklist` calls.
- [x] T010 [P] Update history command examples in `fab/.kit/skills/fab-clarify.md`: replace `<change_dir>` with `<change>` in `log-command` reference.

## Phase 4: New Files

- [x] T011 Create `fab/.kit/skills/_scripts.md`: `<change>` argument convention (accepted forms), per-script summaries (changeman, stageman, calc-score, preflight), stage transition side effects table, common error messages.
- [x] T012 Add `_scripts.md` reference to `fab/.kit/skills/_preamble.md` always-load section: one-liner instruction to also read `fab/.kit/skills/_scripts.md`.

## Phase 5: Memory

- [x] T013 [P] Create `docs/memory/fab-workflow/kit-scripts.md`: deep reference covering internal functions, state machine, `.history.jsonl` schema, design rationale.
- [x] T014 [P] Update `docs/memory/fab-workflow/index.md`: add `kit-scripts` entry.

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003 depends on T001 (history commands must be migrated before updating caller)
- T004 depends on T001 (same reason)
- T005 depends on T001 (help text should reflect new signatures)
- T006-T010 are independent of each other and of T001/T002
- T011 depends on T001/T002 being done (documents the post-fix conventions)
- T012 depends on T011 (must exist before referencing)
- T013, T014 can run in parallel after T011
