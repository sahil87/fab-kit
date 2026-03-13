# Tasks: Pane Map JSON Session Flags

**Change**: 260313-wrt4-pane-map-json-session-flags
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Struct & Discovery Refactor

- [x] T001 Extend `paneEntry` struct with `session string` and `index int` fields in `src/go/fab/cmd/fab/panemap.go`
- [x] T002 Extend `paneRow` struct with `session string` and `windowIndex int` fields in `src/go/fab/cmd/fab/panemap.go`
- [x] T003 Update `discoverPanes()` to accept a session targeting mode (default/named/all-sessions), extend tmux format string to include `#{session_name}` and `#{window_index}`, parse the new fields into `paneEntry` in `src/go/fab/cmd/fab/panemap.go`
- [x] T004 Update `resolvePane()` to populate `session` and `windowIndex` in `paneRow` from the extended `paneEntry` in `src/go/fab/cmd/fab/panemap.go`

## Phase 2: Flags & Command Wiring

- [x] T005 Add `--json`, `--session`, and `--all-sessions` Cobra flags to `paneMapCmd()` in `src/go/fab/cmd/fab/panemap.go`
- [x] T006 Update `runPaneMap()` to validate mutual exclusion of `--session` and `--all-sessions`, and move `$TMUX` guard to only apply when neither flag is set, in `src/go/fab/cmd/fab/panemap.go`
- [x] T007 Update `runPaneMap()` to pass session targeting mode to `discoverPanes()` — default uses current session, `--session` targets by name, `--all-sessions` enumerates all sessions in `src/go/fab/cmd/fab/panemap.go`

## Phase 3: Output Modes

- [x] T008 Add `printPaneJSON()` function that marshals `[]paneRow` to JSON array with correct field names (`session`, `window_index`, `pane`, `tab`, `worktree`, `change`, `stage`, `agent_state`, `agent_idle_duration`) and null semantics in `src/go/fab/cmd/fab/panemap.go`
- [x] T009 Update `printPaneTable()` to include `WinIdx` column between `Pane` and `Tab`, and conditionally include `Session` column as first column when `--all-sessions` is used, in `src/go/fab/cmd/fab/panemap.go`
- [x] T010 Wire `--json` flag in `runPaneMap()` to route to `printPaneJSON()` vs `printPaneTable()` in `src/go/fab/cmd/fab/panemap.go`

## Phase 4: Tests & Documentation

- [x] T011 Add tests for `discoverPanes` format string parsing, `printPaneJSON` null semantics, `printPaneTable` with new columns, and mutual exclusion validation in `src/go/fab/cmd/fab/panemap_test.go`
- [x] T012 Update `fab pane-map` section in `fab/.kit/skills/_scripts.md` with new flags (`--json`, `--session`, `--all-sessions`) and JSON output schema

---

## Execution Order

- T001, T002 are independent (struct changes)
- T003 depends on T001 (uses extended paneEntry)
- T004 depends on T002, T003 (uses both structs)
- T005, T006, T007 depend on T003 (flags wire into discovery)
- T008 depends on T002, T004 (uses paneRow)
- T009 depends on T002, T004 (uses paneRow)
- T010 depends on T005, T008, T009 (wires flag to output)
- T011 depends on T003–T010 (tests new functionality)
- T012 depends on T005 (documents flags)
