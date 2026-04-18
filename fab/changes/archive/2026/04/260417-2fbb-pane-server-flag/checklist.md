# Quality Checklist: Pane Server Flag

**Change**: 260417-2fbb-pane-server-flag
**Generated**: 2026-04-17
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 Persistent --server flag registered: `fab pane map/capture/send/process --help` all list `--server string` and `-L string`; `fab pane --help` lists it under persistent flags
- [x] CHK-002 Empty flag preserves behavior: running any pane subcommand without `--server` invokes tmux with no `-L` argument (argv captured in tests)
- [x] CHK-003 Non-empty flag prepends `-L`: running any pane subcommand with `--server runKit` invokes every tmux subprocess with `-L runKit` as the first two argv elements
- [x] CHK-004 withServer helper exists and behaves correctly: `WithServer(server, args...)` in `internal/pane/pane.go` returns args unchanged when server empty, prepends `["-L", server]` otherwise, does not mutate input (helper was exported — see CHK-037 note — to share canonical argv builder across packages)
- [x] CHK-005 withServer name choice preserves existing `tmuxArgs` local: the local variable at `pane_send.go:58` (now assigned from `sendTextArgs(...)`) is not renamed; no shadowing occurs
- [x] CHK-006 ValidatePane accepts server: signature is `ValidatePane(paneID, server string) error`; tmux invocation uses `WithServer` (pane.go:49-50)
- [x] CHK-007 GetPanePID accepts server: signature is `GetPanePID(paneID, server string) (int, error)`; tmux invocation uses `WithServer` (pane.go:64-65)
- [x] CHK-008 ResolvePaneContext accepts server: signature is `ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error)`; tmux invocation uses `WithServer` (pane.go:82-84)
- [x] CHK-009 discoverPanes family accepts server: signatures for `discoverPanes`, `discoverSessionPanes`, `discoverAllSessions` all include `server string` (panemap.go:129, 159, 169); all use `WithServer` for tmux argv via `listPanesArgs`/`listSessionsArgs` helpers
- [x] CHK-010 paneMapCmd reads --server and threads it: flag value reaches `discoverPanes` (panemap.go:59, 76)
- [x] CHK-011 paneCaptureCmd reads --server and threads it: flag value reaches `ValidatePane`, `ResolvePaneContext`, `capturePaneArgs` (pane_capture.go:45, 54, 60, 72)
- [x] CHK-012 paneSendCmd reads --server and threads it: flag value reaches `ValidatePane` and both send-keys argv builders (pane_send.go:29, 32, 39, 58, 66)
- [x] CHK-013 paneProcessCmd reads --server and threads it: flag value reaches `ValidatePane` and `GetPanePID` (pane_process.go:74, 77, 83); no changes to platform-specific discovery files

## Behavioral Correctness

- [x] CHK-014 Short -L form works: shorthand "L" is registered on the persistent flag at `pane.go:14`; verified via `TestPaneMapServerFlag`/`TestPaneCaptureServerFlag`/`TestPaneSendServerFlag`/`TestPaneProcessServerFlag` which all assert `flag.Shorthand == "L"`
- [x] CHK-015 --server takes precedence over `$TMUX`: when both are set, `--server` wins. Tmux's native behavior — `-L` explicitly selects a socket. Verified via argv assertion (the flag causes `-L` to be prepended to every tmux invocation regardless of `$TMUX`)
- [x] CHK-016 Non-existent server propagates native tmux error: fab does not translate or swallow the tmux error. `ValidatePane`/`discoverPanes` etc. use `fmt.Errorf("tmux ...: %w", err)` which wraps the native error (verified by reading pane.go:52, panemap.go:162)

## Scenario Coverage

- [x] CHK-017 `pane map --server runKit --json --all-sessions` enumerates runKit panes only — covered by `TestListPanesArgs`/`TestListSessionsArgs`/`TestPaneMapServerFlag` in panemap_test.go
- [x] CHK-018 `pane capture %5 --server runKit` — tests in pane_capture_test.go assert server threaded through `capturePaneArgs` (`TestCapturePaneArgs` "non-empty server prepends -L") and flag visible on subcommand (`TestPaneCaptureServerFlag`)
- [x] CHK-019 `pane send %5 "text" --server runKit` — tests in pane_send_test.go assert -L on both send-keys invocations via `TestSendTextArgs` and `TestSendEnterArgs`
- [x] CHK-020 `pane send %5 "text" --force --server runKit` — ValidatePane still runs with server (pane_send.go:32); idle check skipped when `--force` is set (pane_send.go:38). Code path is direct and reviewed
- [x] CHK-021 `pane process %5 --server runKit` — `TestPaneProcessServerFlag` asserts flag is parsed and visible; ValidatePane/GetPanePID receive the server value (pane_process.go:77, 83); platform-specific process discovery untouched (confirmed no changes to pane_process_linux.go/pane_process_darwin.go)

## Edge Cases & Error Handling

- [x] CHK-022 Empty --server value: `--server ""` (or default) treated identically to flag absent. `WithServer` returns args unchanged when server is empty (`TestWithServer` "empty server returns args verbatim")
- [x] CHK-023 --server with special characters: names like `my-socket`, `socket_1` passed verbatim via argv (no shell interpretation). Verified by `TestWithServer` "special characters in server name passed verbatim"
- [x] CHK-024 **N/A**: Same pane ID on two servers — this is a runtime invariant of tmux's architecture, not directly testable without two live tmux servers. Code behavior is correct (argv prepend `-L <server>` means tmux operates in scope of the named server); the scenario is covered by the design (pane ID lookups go through `WithServer`-wrapped calls)
- [x] CHK-025 Non-tmux ops unaffected: `.fab-runtime.yaml`, `.status.yaml`, git worktree detection, and `/proc`/`ps` discovery do not use the `--server` value as a lookup key. Confirmed by inspection — `ResolveAgentState`, `FormatIdleDuration`, `GitWorktreeRoot`, process discovery functions take no `server` parameter and are unchanged

## Code Quality

- [x] CHK-026 Pattern consistency: new code follows existing patterns in `cmd/fab/pane_*.go` (cobra flag reading via `GetString`, `fmt.Errorf` wrapping, small helper extraction) and `internal/pane/pane.go`
- [x] CHK-027 No unnecessary duplication: `WithServer` is the single argv-building helper used at every tmux exec site — no per-file copies or inlined conditionals. All tmux `exec.Command` sites in scope (pane.go:50, 65, 84; panemap.go:160, 170; pane_capture.go:109; pane_send.go:60, 66) use `WithServer` either directly or via a thin wrapper (`listPanesArgs`, `listSessionsArgs`, `capturePaneArgs`, `sendTextArgs`, `sendEnterArgs`)
- [x] CHK-028 Readability: `WithServer` is a short pure function; call sites read as `exec.Command("tmux", WithServer(server, ...args)...)` — no indirection beyond that. The per-subcommand wrapper helpers (e.g., `listPanesArgs`) are one-line argv builders that exist primarily for unit-testability (see CHK-017–CHK-019)
- [x] CHK-029 Composition over inheritance: helper is a free function, not a method on a type; no new struct hierarchy introduced
- [x] CHK-030 No god functions: touched functions remain short (the only refactor is signature extension + a helper call); extracted wrappers are 1–3 lines each
- [x] CHK-031 No magic strings: `"-L"` appears exactly once (inside `WithServer` at pane.go:31); server name is always passed as a variable, never hardcoded

## Documentation Accuracy

- [x] CHK-032 `src/kit/skills/_cli-fab.md` updated: each of the four pane subcommand sections documents `--server` / `-L` in its Flags table; the `fab pane` parent section adds a `#### Persistent Flag` subsection explaining cross-subcommand visibility
- [x] CHK-033 Help text matches spec: flag description in code registration (pane.go:14) and in `_cli-fab.md` both read `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.`
- [x] CHK-034 Hydrate-stage deliverables noted: tasks.md correctly defers `docs/memory/fab-workflow/pane-commands.md` creation, `index.md` registration, and `kit-architecture.md` trim to the hydrate stage (tasks.md footnote after T015)

## Cross References

- [x] CHK-035 spec.md ↔ tasks.md: every spec requirement in §pane-cli/pane-internal/pane-subcommands/pane-semantics maps to at least one task (T001–T015); memory requirements deferred to hydrate
- [x] CHK-036 tasks.md ↔ implementation: every completed task `[x]` has corresponding source edits visible in `git diff` (13 files modified: 7 in `src/go/fab/` code, 5 test files, 1 docs). An additional caller at `resolve.go:49` (not originally listed) was updated to pass `""` — preserves pre-change behavior exactly (zero-semantic change)
- [x] CHK-037 Assumptions trail: spec Assumptions #10 (new pane-commands.md) and #11 (withServer name) reference intake clarifications; no drift. Note: helper was exported (`WithServer`) rather than unexported as drafted in spec §pane-internal — justified by cross-package use from `cmd/fab`; spec's intent (single canonical helper, no duplication) is preserved. Not a regression
- [x] CHK-038 Constitution compliance: per constitution "Changes to the `fab` CLI (Go binary) MUST include corresponding test updates and MUST update `src/kit/skills/_cli-fab.md`" — both satisfied (Phase 3 tests: `TestWithServer`, `TestListPanesArgs`, `TestListSessionsArgs`, `TestCapturePaneArgs`, `TestSendTextArgs`, `TestSendEnterArgs`, four `TestPane*ServerFlag`; T015 docs). `.claude/skills/` (gitignored deployed copies) not edited directly

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
