# Tasks: Extract Logman from Preflight

**Change**: 260302-9fnn-extract-logman-from-preflight
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Script Changes

- [x] T001 Redesign logman.sh `command` subcommand — flip arg order to `command <cmd> [change] [args]`, add optional change resolution via `fab/current` with silent exit 0 on failure, update arg count validation, update help text (`fab/.kit/scripts/lib/logman.sh`)
- [x] T002 Update logman test suite — flip all `command` test invocations to new arg order, add tests for: cmd-only invocation with active change via `fab/current`, cmd-only with no `fab/current` (silent exit 0), cmd-only with stale `fab/current` (silent exit 0), explicit change that doesn't resolve (exit 1), new arg count validation (1 arg = cmd only is valid, 0 args = error) (`src/lib/logman/test.bats`)
- [x] T003 Remove `--driver` flag from preflight.sh — delete `LOGMAN` variable, `--driver` arg parsing block, and step 6 logman call; keep all validation and YAML output unchanged (`fab/.kit/scripts/lib/preflight.sh`)

## Phase 2: Caller Updates

- [x] T004 Update changeman.sh logman `command` calls — flip arg order in `new` subcommand (`logman command "fab-new" "$folder_name" "$log_args"`) and `rename` subcommand (`logman command "changeman-rename" "$new_name" ...`) (`fab/.kit/scripts/lib/changeman.sh`)
- [x] T005 Run changeman and preflight test suites to verify no regressions (`src/lib/changeman/test.bats`, `src/lib/preflight/test.bats`)

## Phase 3: Skill & Documentation Updates

- [x] T006 Add logman call step to `_preamble.md` §2 Change Context — new step after "Parse stdout YAML" (step 3), before "Load artifacts" (step 4): instruct agents to call `logman.sh command "<skill-name>" "<name>" 2>/dev/null || true` using the resolved name from preflight output (`fab/.kit/skills/_preamble.md`)
- [x] T007 [P] Add logman call instructions to exempt skill files — add a best-effort logman call to each of: `fab-new.md` (after folder creation, with change name), `fab-switch.md`, `fab-setup.md`, `fab-discuss.md`, `fab-help.md` (all without change arg) (`fab/.kit/skills/fab-new.md`, `fab/.kit/skills/fab-switch.md`, `fab/.kit/skills/fab-setup.md`, `fab/.kit/skills/fab-discuss.md`, `fab/.kit/skills/fab-help.md`)
- [x] T008 [P] Update `_scripts.md` — update logman `command` signature and examples, remove `--driver` from preflight docs, update logman callers table (remove `preflight.sh --driver` row, add skills as direct callers), update call graph text, replace "Skills never call logman.sh directly" with "Skills call logman.sh command directly" (`fab/.kit/skills/_scripts.md`)

---

## Execution Order

- T001 blocks T002 (tests need new implementation)
- T001 blocks T004 (changeman calls depend on new logman signature)
- T003 blocks T005 (preflight tests need --driver removed)
- T006, T007, T008 are independent of each other but depend on T001 and T003
