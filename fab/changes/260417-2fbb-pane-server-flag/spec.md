# Spec: Pane Server Flag

**Change**: 260417-2fbb-pane-server-flag
**Created**: 2026-04-17
**Affected memory**: `docs/memory/fab-workflow/pane-commands.md` (new), `docs/memory/fab-workflow/kit-architecture.md` (trim)

## Non-Goals

- `--socket-path` / `-S` flag (full path selector) — `-L <name>` covers the motivating case; can be added later if callers need it.
- Env-var alternative (e.g., `FAB_TMUX_SERVER`) — CLI flag is sufficient for programmatic callers; env vars add hidden coupling.
- Changes to non-pane fab subcommands — the underlying bug (implicit tmux socket inheritance) is scoped to pane subcommands.
- run-kit-side integration — threading `--server` through `rk serve`'s `fetchPaneMap` is a downstream follow-up.
- Multi-server orchestration sugar — this change adds the primitive (`-L` passthrough); composition across multiple servers is left to callers.

## pane-cli: Command-Level Flag

### Requirement: Persistent --server flag on pane parent command
The `pane` parent cobra command SHALL register a persistent string flag `--server` with short form `-L`, visible on all subcommands (`map`, `capture`, `send`, `process`). Default value MUST be the empty string. Help text SHALL read: `Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket.`

#### Scenario: Help lists --server on every subcommand
- **GIVEN** the user runs `fab pane map --help` (or `capture`, `send`, `process`)
- **WHEN** the help output is inspected
- **THEN** the output includes `--server string` and `-L string` under Flags
- **AND** the description mentions "tmux socket label"

#### Scenario: Flag registered once at parent level
- **GIVEN** the user runs `fab pane --help`
- **WHEN** the persistent flag section is inspected
- **THEN** `--server` appears in Global/Persistent Flags
- **AND** no subcommand re-registers it locally

### Requirement: Flag default preserves existing behavior
When `--server` is absent or empty, every `exec.Command("tmux", ...)` invocation in the pane call tree MUST run with no `-L` argument. The resulting tmux process MUST inherit socket selection from `$TMUX` or fall back to tmux's default socket, identical to pre-change behavior.

#### Scenario: No flag, no -L
- **GIVEN** the user runs `fab pane map` inside an attached tmux pane with `$TMUX=/tmp/tmux-1001/default,...`
- **WHEN** fab invokes tmux subprocesses
- **THEN** every `exec.Command` has argv `["tmux", "list-panes", ...]` with no `-L`
- **AND** tmux connects to `/tmp/tmux-1001/default`

#### Scenario: Existing scripted callers unaffected
- **GIVEN** a subprocess-style caller invokes `fab pane map --json --all-sessions` without `--server`
- **WHEN** fab executes
- **THEN** argv of every spawned tmux process begins with `tmux list-panes` or `tmux list-sessions` — no `-L` prepended
- **AND** exit code, JSON schema, and text output match pre-change behavior byte-for-byte

### Requirement: Flag set prepends -L verbatim
When `--server` is non-empty, every `exec.Command("tmux", ...)` in the pane call tree MUST be invoked with `-L <value>` prepended to the argument list, preserving the remaining arguments in order. fab SHALL NOT inspect, validate, or normalize the server name — it is passed through to tmux verbatim.

#### Scenario: pane map with --server runKit
- **GIVEN** the user runs `fab pane map --json --all-sessions --server runKit`
- **WHEN** fab enumerates sessions and panes
- **THEN** the `list-sessions` subprocess runs as `tmux -L runKit list-sessions -F '#{session_name}'`
- **AND** each subsequent `list-panes` runs as `tmux -L runKit list-panes -s -F ...`
- **AND** the returned pane set matches the runKit server only

#### Scenario: Short -L form accepted
- **GIVEN** the user runs `fab pane map -L runKit`
- **WHEN** cobra parses the flag
- **THEN** the effective value of `--server` is `runKit`
- **AND** behavior is identical to `--server runKit`

#### Scenario: Non-existent server name
- **GIVEN** the user runs `fab pane map --server nonexistent`
- **WHEN** the first tmux invocation runs
- **THEN** tmux exits non-zero with its native error (e.g., `no server running on /tmp/tmux-1001/nonexistent`)
- **AND** fab propagates the error message to stderr
- **AND** fab does NOT attempt to re-create or translate tmux's error

#### Scenario: --server with special characters
- **GIVEN** a user sets `--server "my-socket"` or `--server "socket_1"`
- **WHEN** fab invokes tmux
- **THEN** the name is passed verbatim via `exec.Command` argv (not a shell) so quoting and escaping are not required
- **AND** tmux receives the exact string as its socket label

## pane-internal: Shared Helpers

### Requirement: withServer argv-building helper
A new unexported helper SHALL be added to `src/go/fab/internal/pane/pane.go` with signature `func withServer(server string, args ...string) []string`. The helper MUST return `args` unchanged when `server == ""`, and MUST return `append([]string{"-L", server}, args...)` when `server != ""`. The helper MUST NOT perform any validation, escaping, or normalization of either argument.

#### Scenario: Empty server returns args unchanged
- **GIVEN** `server == ""`
- **WHEN** `withServer("", "list-panes", "-a")` is called
- **THEN** the return value is `["list-panes", "-a"]`
- **AND** no new slice allocation is required by correctness (may still happen; not asserted)

#### Scenario: Non-empty server prepends -L
- **GIVEN** `server == "runKit"`
- **WHEN** `withServer("runKit", "list-panes", "-a")` is called
- **THEN** the return value is `["-L", "runKit", "list-panes", "-a"]`
- **AND** the original args slice is not mutated (verified by invoking with a shared slice twice)

#### Scenario: No args with non-empty server
- **GIVEN** `server == "runKit"` and no additional args
- **WHEN** `withServer("runKit")` is called
- **THEN** the return value is `["-L", "runKit"]`

### Requirement: Name collision avoidance
The helper SHALL be named `withServer`, not `tmuxArgs`, because `tmuxArgs` is already a local variable in `src/go/fab/cmd/fab/pane_send.go` at line 57. Renaming the local variable is not required — the helper's distinct name resolves the collision at introduction time.

#### Scenario: pane_send local variable remains
- **GIVEN** the post-change source of `pane_send.go`
- **WHEN** the file is inspected at line ~57
- **THEN** the local variable `tmuxArgs := []string{"send-keys", "-t", paneID, "-l", text}` (or equivalent) is preserved by name <!-- clarified: fixed missing opening quote on "-t" string literal; verified against src/go/fab/cmd/fab/pane_send.go line 57 -->

- **AND** the new helper `withServer` is called elsewhere without shadowing

### Requirement: ValidatePane accepts server
`ValidatePane` in `src/go/fab/internal/pane/pane.go` SHALL change signature from `ValidatePane(paneID string) error` to `ValidatePane(paneID, server string) error`. The `tmux list-panes -a -F '#{pane_id}'` subprocess SHALL be invoked with argv built via `withServer(server, "list-panes", "-a", "-F", "#{pane_id}")`.

#### Scenario: ValidatePane with empty server
- **GIVEN** `ValidatePane("%5", "")` is called
- **WHEN** the tmux subprocess runs
- **THEN** argv is `["tmux", "list-panes", "-a", "-F", "#{pane_id}"]`

#### Scenario: ValidatePane with non-empty server
- **GIVEN** `ValidatePane("%5", "runKit")` is called
- **WHEN** the tmux subprocess runs
- **THEN** argv is `["tmux", "-L", "runKit", "list-panes", "-a", "-F", "#{pane_id}"]`

### Requirement: GetPanePID accepts server
`GetPanePID` in `src/go/fab/internal/pane/pane.go` SHALL change signature from `GetPanePID(paneID string) (int, error)` to `GetPanePID(paneID, server string) (int, error)`. The `tmux display-message -t <pane> -p '#{pane_pid}'` subprocess SHALL be invoked with argv built via `withServer(server, "display-message", "-t", paneID, "-p", "#{pane_pid}")`.

#### Scenario: GetPanePID threads server through
- **GIVEN** `GetPanePID("%5", "runKit")` is called
- **WHEN** the tmux subprocess runs
- **THEN** argv is `["tmux", "-L", "runKit", "display-message", "-t", "%5", "-p", "#{pane_pid}"]`

### Requirement: ResolvePaneContext accepts server
`ResolvePaneContext` in `src/go/fab/internal/pane/pane.go` SHALL change signature from `ResolvePaneContext(paneID, mainRoot string) (*PaneContext, error)` to `ResolvePaneContext(paneID, mainRoot, server string) (*PaneContext, error)`. The `tmux display-message -t <pane> -p '#{pane_current_path}'` subprocess SHALL be invoked with argv built via `withServer(server, "display-message", "-t", paneID, "-p", "#{pane_current_path}")`.

#### Scenario: ResolvePaneContext threads server through
- **GIVEN** `ResolvePaneContext("%5", "/repo", "runKit")` is called
- **WHEN** the tmux subprocess runs
- **THEN** argv is `["tmux", "-L", "runKit", "display-message", "-t", "%5", "-p", "#{pane_current_path}"]`

### Requirement: discoverPanes accepts server
`discoverPanes`, `discoverSessionPanes`, and `discoverAllSessions` in `src/go/fab/cmd/fab/panemap.go` SHALL accept an additional `server string` parameter. Both `list-panes` and `list-sessions` tmux subprocesses SHALL be invoked with argv built via `withServer`.

#### Scenario: discoverAllSessions with server
- **GIVEN** `discoverAllSessions("runKit")` is called
- **WHEN** the first tmux subprocess runs
- **THEN** argv is `["tmux", "-L", "runKit", "list-sessions", "-F", "#{session_name}"]`
- **AND** subsequent `list-panes -t <sess>` invocations also prepend `-L runKit`

#### Scenario: discoverSessionPanes with empty server
- **GIVEN** `discoverSessionPanes("main", "")` is called
- **WHEN** the tmux subprocess runs
- **THEN** argv is `["tmux", "list-panes", "-s", "-F", "<format>", "-t", "main"]` (no `-L`)

## pane-subcommands: Per-Subcommand Wiring

### Requirement: pane map reads --server
`paneMapCmd` in `src/go/fab/cmd/fab/panemap.go` SHALL read the `--server` flag via `cmd.Flags().GetString("server")` and pass the value to `discoverPanes`. No other pane-map logic changes.

#### Scenario: Sidebar-style caller targets runKit
- **GIVEN** a parent process invokes `fab pane map --json --all-sessions --server runKit`
- **WHEN** the command runs from a context where `$TMUX` points to a different socket (e.g., `rk-daemon`)
- **THEN** the returned JSON array contains panes from the `runKit` server only
- **AND** each pane's `change`, `stage`, `agent_state` fields are resolved correctly via per-pane CWD (independent of the caller's socket)

### Requirement: pane capture reads --server
`paneCaptureCmd` in `src/go/fab/cmd/fab/pane_capture.go` SHALL read the `--server` flag and pass it to `ValidatePane(paneID, server)`, to `ResolvePaneContext(paneID, mainRoot, server)`, and to the `tmux capture-pane` invocation (currently at line 106, via `capturePaneArgs`). `capturePaneArgs` SHALL be refactored to accept `server` and use `withServer`.

#### Scenario: Capture a pane on named server
- **GIVEN** `fab pane capture %5 --json --server runKit`
- **WHEN** the command runs
- **THEN** pane validation, context resolution, and capture all run against the `runKit` server
- **AND** the JSON output's `change`/`stage`/`agent_state` fields reflect the pane's actual fab context (resolved from pane CWD, independent of socket)

### Requirement: pane send reads --server
`paneSendCmd` in `src/go/fab/cmd/fab/pane_send.go` SHALL read the `--server` flag and pass it to `ValidatePane(paneID, server)` and to both `tmux send-keys` invocations (the main `send-keys -t <pane> -l <text>` and the trailing `send-keys -t <pane> Enter`). The existing local variable `tmuxArgs` SHALL be retained by name; the helper is called separately.

#### Scenario: Send to idle pane on named server
- **GIVEN** pane `%5` on server `runKit` is validated idle via `.fab-runtime.yaml`
- **WHEN** the user runs `fab pane send %5 "hello" --server runKit`
- **THEN** `ValidatePane("%5", "runKit")` succeeds
- **AND** both send-keys subprocesses run with `-L runKit`
- **AND** the pane receives the text followed by Enter

#### Scenario: --force still validates pane existence on named server
- **GIVEN** the user runs `fab pane send %5 "text" --force --server runKit`
- **WHEN** the command starts
- **THEN** `ValidatePane("%5", "runKit")` is still called
- **AND** the idle check is skipped per `--force`
- **AND** if pane `%5` doesn't exist on `runKit`, the command exits 1 with `Error: pane %5 not found`

### Requirement: pane process reads --server
`paneProcessCmd` in `src/go/fab/cmd/fab/pane_process.go` SHALL read the `--server` flag and pass it to `ValidatePane(paneID, server)` and `GetPanePID(paneID, server)`. Platform-specific process discovery (`ps` on darwin, `/proc` on linux) is independent of tmux server selection — no changes to `pane_process_linux.go` or `pane_process_darwin.go`.

#### Scenario: Process tree from named server's pane
- **GIVEN** the user runs `fab pane process %5 --json --server runKit`
- **WHEN** the command runs
- **THEN** pane validation and shell-PID resolution run against `runKit`
- **AND** the subsequent process tree traversal uses the shell PID returned by `runKit`'s tmux
- **AND** the `/proc` or `ps` walk is socket-independent (operates on the OS process table)

## pane-semantics: Invariants

### Requirement: --server precedence
When both `$TMUX` and `--server` are set, `--server` SHALL take precedence. This matches tmux's own behavior: `tmux -L <label>` explicitly selects a socket, overriding any inherited selection.

#### Scenario: Caller in rk-daemon targets runKit
- **GIVEN** `$TMUX=/tmp/tmux-1001/rk-daemon,12345,0` and `fab pane map --server runKit`
- **WHEN** tmux subprocesses run
- **THEN** every subprocess receives `-L runKit` argv
- **AND** the inspected panes belong to `runKit`, not `rk-daemon`

### Requirement: Pane ID scope
Pane IDs (e.g., `%5`) are per-server, not globally unique. When `--server <S>` is passed with a pane ID argument, the ID is interpreted in the context of server `<S>`. Callers are responsible for pairing the correct pane ID with the correct server.

#### Scenario: Same pane ID on two servers
- **GIVEN** pane `%3` exists on both `runKit` and `rk-daemon` (different tmux processes)
- **WHEN** the user runs `fab pane capture %3 --server runKit`
- **THEN** fab captures content from `runKit`'s pane `%3`, not `rk-daemon`'s
- **AND** running `fab pane capture %3 --server rk-daemon` returns content from the other pane

### Requirement: --server does not affect non-tmux operations
File reads (`.fab-runtime.yaml`, `.status.yaml`), git-worktree detection, and OS-level process discovery MUST NOT be influenced by the `--server` value. These operations key off the pane's CWD or the folder name, not the tmux server.

#### Scenario: Runtime state resolves from worktree, not socket
- **GIVEN** `fab pane send %5 "text" --server runKit` where `%5`'s CWD is `/repo/worktrees/alpha`
- **WHEN** fab resolves the agent idle state
- **THEN** it reads `/repo/worktrees/alpha/.fab-runtime.yaml`
- **AND** the server name `runKit` is not used as a lookup key anywhere in the runtime file read

## docs: Documentation Updates

### Requirement: _cli-fab.md flag documentation
`src/kit/skills/_cli-fab.md` SHALL document the persistent `--server` / `-L` flag under each of the four pane subcommand sections (`fab pane map`, `fab pane capture`, `fab pane send`, `fab pane process`). Each entry SHALL include: flag syntax, type (`string`), default (`""` — inherit `$TMUX`), and a one-line description linking to tmux's `-L` convention.

#### Scenario: Skill context lists --server
- **GIVEN** a skill load of `_cli-fab.md`
- **WHEN** the pane subcommand reference is inspected
- **THEN** each subcommand's Flags table includes a `--server` row
- **AND** the parent `fab pane` section includes an explanation of the persistent-flag placement

### Requirement: Memory file pane-commands.md
`docs/memory/fab-workflow/pane-commands.md` SHALL be created during the hydrate stage. It SHALL document the four pane subcommands, the `--server` / `-L` flag, the motivating run-kit daemon case (where `rk serve` runs inside an `rk-daemon` tmux session and inspects panes on the `runKit` socket), and the default-inheritance behavior when the flag is absent.

### Requirement: Memory index registration
`docs/memory/fab-workflow/index.md` SHALL be updated during hydrate to register `pane-commands.md` as a new entry in the fab-workflow domain table, with a one-line description and the current date.

### Requirement: kit-architecture.md trim
`docs/memory/fab-workflow/kit-architecture.md` SHALL be trimmed during hydrate: the per-subcommand detail currently on lines ~308–312 SHALL be replaced with a high-level statement that `fab pane` groups four subcommands whose detailed behavior lives in `pane-commands.md`. The `fabGoNoConfigArgs` allowlist note SHALL be retained.

## test: Testing Requirements

### Requirement: withServer unit tests
A test for `withServer` SHALL be added to `src/go/fab/internal/pane/pane_test.go` covering: (a) empty server returns args verbatim, (b) non-empty server prepends `-L <server>`, (c) the original args slice is not mutated across calls, (d) no-args call with non-empty server returns `["-L", server]`.

### Requirement: Subcommand flag passthrough tests
Each of `panemap_test.go`, `pane_capture_test.go`, `pane_send_test.go`, `pane_process_test.go` SHALL include a test verifying that when a test double captures the argv of tmux subprocesses, the `--server <value>` CLI flag causes `-L <value>` to appear at positions `[0,1]` of the argv, and absence of the flag causes no `-L` to appear. Tests SHALL NOT require a live tmux server.

### Requirement: Signature-change compilation
All existing call sites of `ValidatePane`, `GetPanePID`, `ResolvePaneContext`, `discoverPanes`, `discoverSessionPanes`, `discoverAllSessions` SHALL be updated to pass the new `server` parameter. The Go compiler enforces this via compilation errors — `go build ./...` and `go test ./...` MUST pass after the change.

#### Scenario: Build is clean
- **GIVEN** the post-change tree
- **WHEN** `just test` or `go test ./...` runs from the repo root
- **THEN** all tests pass
- **AND** no stale callers remain with the old signatures

## Design Decisions

1. **Persistent flag on the parent `pane` command, not per-subcommand**:
   - *Why*: Cobra idiom for a flag that applies uniformly to every subcommand in a group. One registration point, one help-text location, zero chance of per-subcommand drift.
   - *Rejected*: Per-subcommand registration — four copies of the same flag declaration, four places to update if the description changes.

2. **`withServer` helper in `internal/pane/pane.go`, not a new `internal/tmuxutil/` package**:
   - *Why*: Scope is exactly one helper for one flag. A new package would be a future-looking abstraction with no second caller to justify it. The pane package already owns all tmux-invocation code; the helper sits next to its users.
   - *Rejected*: `internal/tmuxutil/` package — proportionality concern. Can be promoted later if other fab subcommand groups need tmux-invocation helpers (none do currently).

3. **Helper name `withServer`, not `tmuxArgs`**:
   - *Why*: `tmuxArgs` is already a local variable in `pane_send.go:57`. Introducing a function of the same name would shadow or collide, and renaming the local variable creates unnecessary churn outside the flag's scope.
   - *Rejected*: `tmuxArgs` with local rename — couples this change to an unrelated identifier rename. `prependServer` / `withL` / `tmuxCmdArgs` — `withServer` reads most naturally at call sites: `exec.Command("tmux", withServer(server, "list-panes", "-a")...)`.

4. **Pass the server verbatim to tmux without fab-side validation**:
   - *Why*: tmux owns the semantics of socket labels. Any pre-validation in fab (e.g., "does this socket exist?") would duplicate tmux's own error handling and introduce race conditions (socket created/destroyed between check and use). Propagating tmux's native error is both simpler and more accurate.
   - *Rejected*: Pre-check via `tmux -L <server> has-session` or similar — adds an extra subprocess, and fab would still need to handle the real tmux error from the actual command anyway.

5. **Uniform `-L` prepend at every tmux exec site, not a shared "tmux runner" abstraction**:
   - *Why*: The project has only a handful of tmux invocations in the pane subcommands. A dedicated runner type (e.g., `type TmuxClient struct { server string }` with methods `ListPanes()`, `DisplayMessage()`, etc.) would be a large refactor for a one-flag change. The `withServer` helper gives 95% of the benefit with a line of code.
   - *Rejected*: `TmuxClient` type — over-engineering for current needs; can be extracted later if tmux-helper surface grows.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name `--server` with short `-L`, matching tmux convention | Confirmed from intake #1 | S:95 R:85 A:95 D:90 |
| 2 | Certain | Empty flag default preserves current behavior | Confirmed from intake #2 | S:95 R:95 A:95 D:95 |
| 3 | Certain | Scope is all four `pane` subcommands | Confirmed from intake #3 | S:90 R:80 A:90 D:90 |
| 4 | Certain | Registered as PersistentFlags on parent `paneCmd` | Confirmed from intake #4 | S:90 R:90 A:95 D:90 |
| 5 | Certain | Helper lives in `internal/pane/pane.go` | Confirmed from intake #5 + clarified intake #11 | S:95 R:80 A:85 D:80 |
| 6 | Certain | Change type is `feat` (additive CLI capability) | Confirmed from intake #6; `.status.yaml` set to `feat` | S:90 R:90 A:90 D:80 |
| 7 | Certain | No `-S` / `--socket-path` in first cut | Confirmed from intake #7; declared as non-goal | S:95 R:85 A:75 D:75 |
| 8 | Certain | No env-var alternative in first cut | Confirmed from intake #8; declared as non-goal | S:95 R:80 A:75 D:75 |
| 9 | Certain | Pure-function unit test only; no live-tmux integration test | Confirmed from intake #9 | S:95 R:85 A:80 D:75 |
| 10 | Certain | Primary memory file is new `fab-workflow/pane-commands.md`; `kit-architecture.md` is trimmed | Confirmed from intake clarification #10 | S:95 R:85 A:60 D:55 |
| 11 | Certain | Helper named `withServer`, avoiding collision with existing `tmuxArgs` local | Confirmed from intake clarification #11 | S:95 R:80 A:60 D:55 |
| 12 | Certain | `--server` takes precedence over inherited `$TMUX` | Tmux's native behavior — `-L` is an explicit selector; fab just passes it through | S:90 R:85 A:95 D:90 |
| 13 | Certain | Pane IDs (`%N`) are per-server, not globally unique | Tmux invariant — pane IDs are allocated per-server | S:95 R:90 A:95 D:95 |
| 14 | Certain | No fab-side validation of the server name | Tmux owns socket semantics; propagate its native error | S:90 R:85 A:90 D:85 |
| 15 | Certain | Non-tmux operations (file reads, git, process discovery) unaffected by `--server` | These operations key off CWD / folder name, not socket | S:95 R:85 A:95 D:95 |
| 16 | Certain | `_cli-fab.md` must be updated per constitution rule on fab CLI changes | Constitution: "Changes to the `fab` CLI ... MUST update `src/kit/skills/_cli-fab.md`" | S:95 R:90 A:95 D:95 |
| 17 | Certain | Subcommand test doubles use argv-capture, not live tmux | Matches existing test strategy in `pane_*_test.go` | S:85 R:85 A:80 D:80 |
| 18 | Confident | `capturePaneArgs` in pane_capture.go is refactored to accept `server` (same shape as withServer call path) | Reasonable default; alternative is inlining — equivalent in effect | S:75 R:85 A:80 D:80 |

18 assumptions (17 certain, 1 confident, 0 tentative, 0 unresolved).
