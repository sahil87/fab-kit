# Tasks: Window Prefix Primitives and Done-Marker on Removal

**Change**: 260423-rxu3-window-prefix-primitives
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Setup

- [x] T001 Create `src/go/fab/cmd/fab/pane_window_name.go` with `paneWindowNameCmd()` returning a cobra group whose `Use` is `window-name`, `Short` is `Window-name prefix operations`, with two child subcommands (`ensureCmd`, `replaceCmd`) registered via `cmd.AddCommand(...)`. The child `RunE` functions may be stub-empty at this task; core logic lands in Phase 2.
- [x] T002 Wire `paneWindowNameCmd()` into `paneCmd()` in `src/go/fab/cmd/fab/pane.go` by appending it to the existing `cmd.AddCommand(paneMapCmd(), paneCaptureCmd(), paneSendCmd(), paneProcessCmd())` block. Update the parent command's `Long` description from `"Tmux pane operations: map, capture, send, process"` to `"Tmux pane operations: map, capture, send, process, window-name"`.

## Phase 2: Core Implementation

- [x] T003 Add a shared helper `ReadWindowName(paneID, server string) (string, error)` to `src/go/fab/internal/pane/pane.go` that wraps `tmux display-message -p -t <pane> '#W'` via the existing `WithServer` argv builder. Return the trimmed window name and any exec error. Follow the style of the existing `GetPanePID` helper.
- [x] T004 Implement the `ensure-prefix` subcommand `RunE` in `pane_window_name.go`: accept `<pane>` and `<char>` positional args (cobra `ExactArgs(2)`); short-circuit with exit 1 + `tmux not running` on unset `$TMUX`; call `ReadWindowName`; compare with `strings.HasPrefix(name, char)`; on mismatch call `exec.Command("tmux", pane.WithServer(server, "rename-window", "-t", paneID, char+name)...).Run()`. On success print `renamed: <old> -> <new>` to stdout, exit 0. On no-op, print nothing, exit 0. Plumb the `--server` persistent flag via `cmd.Flags().GetString("server")`.
- [x] T005 Implement the `replace-prefix` subcommand `RunE` in `pane_window_name.go`: accept `<pane>`, `<from>`, `<to>` positional args (cobra `ExactArgs(3)`); validate `<from>` non-empty (exit 3 with usage message on stderr if empty); short-circuit tmux-unset to exit 1; call `ReadWindowName`; if `strings.HasPrefix(name, from)` is false, no-op with exit 0; otherwise compute `newName := to + strings.TrimPrefix(name, from)` and call `tmux rename-window -t <pane> <newName>`. Match the print and exit semantics of T004.
- [x] T006 Add a `mapTmuxError(err error, stderrBytes []byte) int` helper in `pane_window_name.go` (or in `internal/pane/` if a second caller wants it) that maps: `ValidatePane` pane-not-found → exit 2 with tmux's stderr propagated (or `pane <id> not found` fallback); any other tmux failure → exit 3 with tmux's stderr propagated. Wire both `ensure-prefix` and `replace-prefix` through this on any non-nil exec error. Call `pane.ValidatePane(paneID, server)` at the top of each RunE so the pane-existence check happens before `ReadWindowName` and maps cleanly to exit 2.
- [x] T007 [P] Add a `--json` bool flag to both subcommands via `cmd.Flags().Bool("json", false, "Emit structured JSON output")`. When set, emit `{"pane": "<pane>", "old": "<old>", "new": "<new>", "action": "renamed"|"noop"}` via `encoding/json` to stdout instead of the plain `renamed: ...` line. Factor the output choice into a small `emitResult(pane, old, new, action string, asJSON bool)` helper in `pane_window_name.go` shared between both verbs.

## Phase 3: Integration & Edge Cases

- [x] T008 Create `src/go/fab/cmd/fab/pane_window_name_test.go` following the argv-capture pattern from `pane_send_test.go`. Factor testable argv builders (e.g., `renameArgs(server, paneID, newName) []string`) and test: (a) empty server returns bare `rename-window` argv, (b) non-empty server prepends `-L <server>`, (c) pane ID and new-name placement are correct. Add flag-existence tests for `--json` on both subcommands and for the `--server` flag inheritance (mirror `TestPaneSendServerFlag`).
- [x] T009 [P] Add parent-group wiring tests to `pane_window_name_test.go`: (a) `paneCmd()` registers a child named `window-name`, (b) `paneWindowNameCmd()` registers exactly two children named `ensure-prefix` and `replace-prefix`, (c) positional arg counts match (`ExactArgs(2)` for ensure, `ExactArgs(3)` for replace).
- [x] T010 [P] Add output-format unit tests: `emitResult` in plain mode produces `renamed: OLD -> NEW\n` for `action="renamed"` and empty string for `action="noop"`; JSON mode produces the documented object shape with stable key order and `action` matching the input. Test both verbs through these helpers without invoking real tmux.
- [x] T011 Update `src/kit/skills/fab-operator.md` §4 Monitored Set Enrollment paragraph (currently lines 161–172 around the three-line shell snippet). REPLACE the `tmux display-message` / `case` / `tmux rename-window` shell block with a single-line invocation `fab pane window-name ensure-prefix <pane> »`. PRESERVE the surrounding prose: durability claim, non-zero exit logging format `"{change}: window rename skipped ({error})."`, and the guard-semantics description (update to say the guard is now enforced by the primitive's literal prefix check).
- [x] T012 Update `src/kit/skills/fab-operator.md` §4 Monitored Set Removal paragraph (around line 174). DELETE the sentence "The window name is **not** restored on removal — the `»` prefix persists. Users who want it removed rename the window manually (`Ctrl-b ,`)." REPLACE with a paragraph describing the `replace-prefix » ›` swap invoked on every removal path, including: (a) exit 2 (pane missing) is treated as successful removal; (b) non-zero exits log `"{change}: window rename skipped ({error})."`; (c) user-renamed windows are no-oped by the primitive's guard.
- [x] T013 Update `src/kit/skills/fab-operator.md` §6 step 4 (around line 341). Rewrite the parenthetical "(Enrollment applies the §4 window-rename rule; the `»<wt>` name produced in step 3 already satisfies the idempotent prefix guard, so no duplicate rename occurs.)" so it references the new primitive explicitly: "(Enrollment calls `fab pane window-name ensure-prefix <pane> »` per §4; the `»<wt>` name produced in step 3 already satisfies the primitive's idempotent prefix check, so no duplicate rename occurs.)"
- [x] T014 Update `docs/specs/skills/SPEC-fab-operator.md` Section Structure item 4 (the "Monitoring System" one-bullet) to read: `Window-name rename on enrollment: prefix » to the tmux window name via fab pane window-name ensure-prefix (idempotent). Removal replaces » with › via fab pane window-name replace-prefix, guarded to skip user-renamed windows.` Replace the existing line verbatim; no other edits to the file.

## Phase 4: Polish

- [x] T015 Bump `src/kit/VERSION` from `1.5.1` to `1.6.0` (minor: adds a new subcommand group without breaking existing surface). No other version constants exist in the repo.
- [x] T016 Run `go build ./...` and `go test ./src/go/fab/...` from the repo root to confirm the new subcommands compile and all tests pass.

---

## Execution Order

- T001 blocks T002, T004, T005 (paneWindowNameCmd must exist before wiring or filling RunE).
- T003 blocks T004, T005 (ReadWindowName is called by both RunE implementations).
- T004 and T005 block T006 (mapTmuxError is used from both RunE call sites). T006 may also be implemented first as a pure function and called from T004/T005, in which case the block reverses.
- T007 is independent of T004/T005's happy-path logic but touches the same files — run after T005.
- T008, T009, T010 can run in parallel but all require T001–T007 done (they test the implementation).
- T011, T012, T013 edit the same markdown file (`src/kit/skills/fab-operator.md`) in different sections — safest to do sequentially to avoid merge churn even though they touch non-overlapping text.
- T014 is independent.
- T015 can run anytime after T001.
- T016 runs last — it validates the whole thing.
