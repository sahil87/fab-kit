# Tasks: Centralize Current Pointer Format

**Change**: 260302-a8ay-centralize-current-pointer
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Read/Write Path

<!-- resolve.sh and changeman.sh switch must both be updated for the format to work.
     resolve reads what changeman writes — they must agree on the two-line format. -->

- [x] T001 Update `resolve.sh` default mode to read line 2 of `fab/current` instead of `tr -d '[:space:]'` on the entire file. In `fab/.kit/scripts/lib/resolve.sh`, replace `name=$(tr -d '[:space:]' < "$current_file")` (line 85) with reading line 2 specifically and stripping whitespace. Update `src/lib/resolve/test.bats`: change test "no argument reads fab/current" (line 109) to write two-line format `printf 'a1b2\n260228-a1b2-test-change'`; change test "fab/current with trailing whitespace" (line 117) to write two-line format with whitespace on line 2. Add new test "fab/current two-line format reads folder from line 2".
- [x] T002 Update `changeman.sh switch` to write two-line format (ID on line 1, folder name on line 2). In `fab/.kit/scripts/lib/changeman.sh`, replace `printf '%s' "$resolved" > "$FAB_ROOT/current"` (line 185) with extracting the 4-char ID via `cut -d'-' -f2` and writing both lines. Update `src/lib/changeman/test.bats`: change test "switch: writes fab/current" (line 523) to assert line 1 is the 4-char ID and line 2 is the full folder name. Add new tests for line-specific assertions using `sed -n '1p'` and `sed -n '2p'`.
- [x] T003 Update `changeman.sh rename` to handle two-line `fab/current`. In `fab/.kit/scripts/lib/changeman.sh` `cmd_rename` (line 394–402): replace `current_val=$(cat "$current_file")` / `if [ "$current_val" = "$folder" ]` with reading line 2 for comparison and writing both lines (line 1 preserved, line 2 updated). Update `src/lib/changeman/test.bats`: change tests "rename updates fab/current when it points to old folder" (line 305) and "rename does not modify fab/current when it points to different change" (line 315) to use two-line format in setup and assertions. Also update "resolve: reads fab/current when no override" (line 467) and "resolve: fab/current with trailing whitespace resolves" (line 475) to write two-line format.

## Phase 2: Dependent Scripts

- [x] T004 Update `preflight.sh` to emit `id:` field in YAML output. In `fab/.kit/scripts/lib/preflight.sh`: after resolving `name`, extract the 4-char ID via `resolve.sh --id` or `echo "$name" | cut -d'-' -f2`. Add `id: $id` before the `name:` line in the YAML output (around line 120). Update `src/lib/preflight/test.bats`: change `set_current` helper (line 49) from `echo "$1"` to write two-line format (`echo "$1" | cut -d'-' -f2` on line 1, `$1` on line 2 — or for short test names without IDs, keep backward compat). Add new test "YAML output includes id field" asserting output contains `id:`.
- [x] T005 [P] Update `logman.sh` to remove direct `fab/current` read and delegate to `resolve.sh`. In `fab/.kit/scripts/lib/logman.sh` `command` subcommand (lines 72–80): replace the direct-read block (`current_file`, `tr -d`, `resolve_change_dir`) with `change_dir=$(resolve_change_dir "" 2>/dev/null) || exit 0` where the empty string triggers resolve.sh's default mode (which reads fab/current internally). Update `src/lib/logman/test.bats`: change test "command with cmd only resolves via fab/current" (line 92) to write two-line format; change test "command with cmd only and stale fab/current exits 0 silently" (line 125) to use two-line format with stale name.
- [x] T006 [P] Update `dispatch.sh` to poll `fab/current` line 1 (4-char ID) instead of full content. In `fab/.kit/scripts/pipeline/dispatch.sh` `run_pipeline` function (lines 242–249): extract the expected 4-char ID from `CHANGE_ID` via `cut -d'-' -f2`, then read line 1 of `fab/current` (via `sed -n '1p'` or `head -1`) and compare against it. Replace the `tr -d '[:space:]' < "$fab_current_file"` comparison.

## Phase 3: Skill Updates

- [x] T007 [P] Update `_preamble.md` §2 to instruct agent to use `id` field for script calls. In `fab/.kit/skills/_preamble.md`: (a) update step 3 "Parse stdout YAML" to mention the `id` field; (b) update step 4 to change `"<name>"` to `"<id>"` in the `logman.sh command` example; (c) add a note that `id` is for script invocations while `name` remains for display and path construction.
- [x] T008 [P] Update `fab-discuss` skill to use `resolve.sh` instead of reading `fab/current` directly. In `.claude/skills/fab-discuss/SKILL.md`: (a) replace "Read `fab/current`" instruction (line 41) with a bash call to `resolve.sh --folder` with silent-failure handling; (b) replace "read `fab/changes/{name}/.status.yaml`" with using the resolved name.
- [x] T009 [P] Update `fab-archive` skill to use `changeman.sh` instead of direct `fab/current` operations. In `.claude/skills/fab-archive/SKILL.md`: (a) replace Step 5 "Clear Pointer" instruction with `changeman.sh switch --blank`; (b) in restore Step 3 "Update Pointer", replace direct write with `changeman.sh switch <name>`; (c) replace any "read fab/current" instructions with `changeman.sh resolve` or `resolve.sh --folder`.

---

## Execution Order

- T001 and T002 must both complete before running resolve or changeman tests (they share the read/write format)
- T003 depends on T001 + T002 (rename reads/writes the new format)
- T004 depends on T001 (preflight calls resolve.sh which must handle two-line format)
- T005 and T006 depend on T001 (logman and dispatch read fab/current via resolve.sh or directly)
- T007, T008, T009 are independent of each other and of T004-T006 (skill file edits)
