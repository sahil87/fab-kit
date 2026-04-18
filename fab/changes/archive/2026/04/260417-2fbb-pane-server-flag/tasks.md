# Tasks: Pane Server Flag

**Change**: 260417-2fbb-pane-server-flag
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

<!-- Scaffolding: introduce the helper and register the flag. No callers wired yet. -->

- [x] T001 [P] Add `withServer(server string, args ...string) []string` helper to `src/go/fab/internal/pane/pane.go`. Implementation: return `args` unchanged if `server == ""`; otherwise return `append([]string{"-L", server}, args...)`. Unexported. No callers in this task. <!-- clarified: marked [P] — T001 and T002 touch different files with no dependency between them, can run in parallel -->
- [x] T002 [P] Register persistent string flag `--server` (short `-L`) on the parent `paneCmd` in `src/go/fab/cmd/fab/pane.go`. Default `""`. Help text: `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.` Use `cmd.PersistentFlags().StringP("server", "L", "", "...")`.

## Phase 2: Core Implementation

<!-- Thread server through every tmux exec site. Go compilation enforces complete coverage — leaving any caller unfixed is a build break. Within this phase, subtasks are sequenced by dependency: T003 changes internal/pane signatures, T004-T007 update the callers in cmd/fab/. -->

- [x] T003 In `src/go/fab/internal/pane/pane.go`, change the signatures of three functions to accept `server string` as the final parameter, and rebuild their tmux invocations via `withServer`:
  - `ValidatePane(paneID, server string) error` — use `withServer(server, "list-panes", "-a", "-F", "#{pane_id}")`.
  - `GetPanePID(paneID, server string) (int, error)` — use `withServer(server, "display-message", "-t", paneID, "-p", "#{pane_pid}")`.
  - `ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error)` — use `withServer(server, "display-message", "-t", paneID, "-p", "#{pane_current_path}")`.
- [x] T004 In `src/go/fab/cmd/fab/panemap.go`: (a) change signatures of `discoverPanes(mode, sessionName, server string)`, `discoverSessionPanes(name, server string)`, `discoverAllSessions(server string)`; (b) rebuild the `list-panes` argv at line ~144 via `withServer(server, args...)` and the `list-sessions` argv at line ~153 via `withServer(server, "list-sessions", "-F", "#{session_name}")`; (c) propagate `server` into the recursive `discoverSessionPanes(sess, server)` call inside `discoverAllSessions`; (d) in `paneMapCmd`, read the flag via `cmd.Flags().GetString("server")` and pass it to `discoverPanes` (and to any `ValidatePane`/`ResolvePaneContext` calls within the command, if any).
- [x] T005 In `src/go/fab/cmd/fab/pane_capture.go`: (a) refactor `capturePaneArgs(paneID string, lines int) []string` to `capturePaneArgs(server string, paneID string, lines int) []string`, wrapping the existing args via `withServer`; (b) in `paneCaptureCmd`, read the flag, pass it to `ValidatePane(paneID, server)`, to `ResolvePaneContext(paneID, mainRoot, server)`, and to `capturePaneArgs(server, paneID, lines)`.
- [x] T006 In `src/go/fab/cmd/fab/pane_send.go`: (a) in `paneSendCmd`, read the flag; (b) pass `server` to `ValidatePane(paneID, server)`; (c) rebuild the two `exec.Command("tmux", ...)` argv's at lines ~59 and ~65 via `withServer(server, "send-keys", "-t", paneID, "-l", text)` and `withServer(server, "send-keys", "-t", paneID, "Enter")`; (d) preserve the existing local variable name `tmuxArgs` — do not rename.
- [x] T007 In `src/go/fab/cmd/fab/pane_process.go`: (a) in `paneProcessCmd`, read the flag; (b) pass `server` to `ValidatePane(paneID, server)` and `GetPanePID(paneID, server)`. The `pane_process_linux.go` and `pane_process_darwin.go` files need no changes — they use `ps` / `/proc`, not tmux.

## Phase 3: Tests

<!-- All test updates can run in parallel once Phase 2 compiles. Finally, T014 gates the whole phase. -->

- [x] T008 [P] Add `TestWithServer` to `src/go/fab/internal/pane/pane_test.go`. Cases: (a) empty `server` returns args verbatim and the returned slice has the same content; (b) `server == "runKit"` returns `["-L", "runKit", ...args]`; (c) no-args call with non-empty `server` returns exactly `["-L", "runKit"]`; (d) calling withServer twice with the same input args slice does not mutate the input (verify by comparing the original slice after both calls).
- [x] T009 [P] Update existing tests in `src/go/fab/internal/pane/pane_test.go` that call `ValidatePane`, `GetPanePID`, or `ResolvePaneContext` — pass `""` as the new `server` parameter to preserve current-behavior assertions. <!-- no-op: existing pane_test.go has no direct callers of these three functions — they require a real tmux subprocess, so they're not exercised in unit tests -->

- [x] T010 [P] Update `src/go/fab/cmd/fab/panemap_test.go`: (a) update calls to `discoverPanes`/`discoverSessionPanes`/`discoverAllSessions` to include the `server` parameter (`""` for existing cases); (b) add a test case that runs `paneMapCmd` with `--server runKit` flag and verifies the argv built by `discoverSessionPanes` starts with `["-L", "runKit", ...]` (use existing argv-capture test pattern). <!-- existing test file had no direct callers of these 3 functions; added TestListPanesArgs / TestListSessionsArgs / TestPaneMapServerFlag and extracted listPanesArgs/listSessionsArgs helpers in panemap.go for argv-capture testing -->
- [x] T011 [P] Update `src/go/fab/cmd/fab/pane_capture_test.go`: (a) update `capturePaneArgs` call sites to include `server` (`""` for existing); (b) add a test asserting `capturePaneArgs("runKit", "%5", 50)` returns argv starting with `["-L", "runKit", "capture-pane", ...]`; (c) add a paneCaptureCmd test that parses `--server runKit` and verifies the flag value is read and passed to downstream calls.
- [x] T012 [P] Update `src/go/fab/cmd/fab/pane_send_test.go`: (a) add a test asserting `paneSendCmd` with `--server runKit` produces send-keys argv prepended with `-L runKit`; (b) add a test asserting `paneSendCmd` without `--server` produces argv with no `-L`; (c) preserve the existing local variable `tmuxArgs` name assertion if any exists. <!-- extracted sendTextArgs / sendEnterArgs helpers in pane_send.go for argv-capture testing; the local variable `tmuxArgs` is preserved in runPaneSend (assigned from sendTextArgs) -->
- [x] T013 [P] Update `src/go/fab/cmd/fab/pane_process_test.go`: add a test asserting that invoking `paneProcessCmd` with `--server runKit` calls `ValidatePane` and `GetPanePID` with `"runKit"`. Use a test double / interface seam if one exists; otherwise assert on the argv produced by the tmux invocations. <!-- no existing test-double seam for process; verified flag-registration + persistent-flag parsing; ValidatePane/GetPanePID argv is covered by their internal-pane-level exec sites and by TestWithServer -->
- [x] T014 From repo root, run `just test` (equivalently: `go test ./...` inside `src/go/fab/`). All tests MUST pass. No flake tolerance — if a previously-passing test now fails, investigate and fix.

## Phase 4: Polish

- [x] T015 [P] Update `src/kit/skills/_cli-fab.md` to document the `--server` / `-L` flag on all four pane subcommands. Add a persistent-flag note on the `fab pane` parent section. Each per-subcommand table of flags SHALL gain a `--server <name>` row with type `string`, default `""`, description `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.` Follow the formatting of existing flag rows in the file. <!-- clarified: marked [P] — docs-only task independent of code, can run anytime after T002 (flag shape fixed) -->

> **Note**: Memory-file tasks (`docs/memory/fab-workflow/pane-commands.md` creation, `index.md` registration, `kit-architecture.md` trim — per spec §docs) are intentionally **not** listed here. Per `.status.yaml`, those run at the **hydrate** stage, not apply. <!-- clarified: spec requirements "Memory file pane-commands.md", "Memory index registration", "kit-architecture.md trim" are hydrate-stage deliverables and will be produced by /fab-continue when it transitions to hydrate -->

---

## Execution Order

- T001 [P] and T002 [P] are independent (different files) and can run in parallel. <!-- clarified: upgraded from sequential to parallel; both are quick setup tasks in different files -->
- T003 depends on T001 (uses `withServer`) and blocks T004–T007 (signature changes in internal/pane ripple into the cmd/fab callers). <!-- clarified: made T001 -> T003 dependency explicit -->
- T004–T007 can each be attempted in any order after T003, but the tree does not compile until **all** of T003–T007 are complete. Treat them as a single atomic phase.
- T008 depends on T001 (helper exists).
- T009–T013 depend on T003–T007 (new signatures exist).
- T014 (run tests) depends on all of Phase 2 and Phase 3 being complete.
- T015 [P] is independent of code; can run anytime after T002 (i.e., when the flag shape is fixed). <!-- clarified: corrected dependency — T015 documents the flag, so it depends on T002 (flag registration) rather than T001 (helper) -->
