# Intake: Pane Server Flag

**Change**: 260417-2fbb-pane-server-flag
**Created**: 2026-04-17
**Status**: Draft

## Origin

Surfaced while diagnosing a run-kit bug: the sidebar "Pane" panel in run-kit's web UI was not rendering the `fab` and `agt` lines for any window, even after recent run-kit fixes (#147, #153) that were intended to restore that enrichment. Tracing the data flow showed the Go backend (`rk serve`) does call `fab pane map --json --all-sessions` as a subprocess, but the subprocess returns entries for the wrong tmux server — not the one the sidebar is displaying.

The run-kit daemon runs inside a tmux session named `rk-daemon` (so its `TMUX` env var points at `/tmp/tmux-1001/rk-daemon,…`), while the user's sessions it is inspecting live on a different socket (`/tmp/tmux-1001/runKit`). `fab pane map` invokes `tmux list-sessions` / `list-panes` with no `-L` flag, so tmux uses `$TMUX` inherited from fab's caller. Result: fab enumerates panes from the wrong server, every key lookup misses, and all enrichment fields come back empty.

Verified by reproducing locally:

```bash
# wrong socket (server's TMUX) → returns only rk-daemon's "rk:0" pane
TMUX=/tmp/tmux-1001/rk-daemon,…  fab pane map --json --all-sessions
# right socket (my interactive shell's TMUX) → returns the runKit panes with change/stage/agent_state populated
TMUX=/tmp/tmux-1001/runKit,…     fab pane map --json --all-sessions
```

`fab pane map --help` exposes `--session <name>` (tmux session-name filter) and `--all-sessions` but **no** flag to select the tmux server / socket label. Callers that don't share a socket with the inspected sessions have no clean way to disambiguate.

## Why

**Problem.** All four `fab pane` subcommands (`map`, `capture`, `send`, `process`) shell out to `tmux` without ever passing `-L <label>`. They implicitly rely on `$TMUX` inheritance (when run inside a tmux session) or fall back to tmux's default socket. This is fine for human users typing `fab pane …` from an attached tmux pane, but it breaks any *programmatic* caller that:

1. runs as a long-lived daemon launched inside a different tmux instance than the one being queried, or
2. runs from a non-tmux context and needs to inspect a specific, named tmux server, or
3. needs to orchestrate panes across multiple tmux servers on the same host.

The motivating concrete case is run-kit (case 1): the HTTP server already threads a `server` parameter through every one of its own tmux subprocess calls (`tmux -L <server> list-sessions`, etc. — see `app/backend/internal/tmux/tmux.go`), because it knows which socket a given `/api/sessions?server=<name>` request is about. When it delegates to `fab pane map`, it has no way to pass that socket through, so fab queries whichever socket happens to be in `$TMUX`.

**Consequence of no fix.**

- run-kit's sidebar fab/agt enrichment is permanently broken in the production Homebrew install, because `rk serve` is daemonized via tmux (`tmux -L rk-daemon new-session -d -s rk -n serve …`). The "fix" shipped in run-kit #153 runs the subprocess but cannot possibly receive correct data. The only reason the fix appeared to work in dev was that `just dev` launches `rk` in the developer's own interactive tmux session, which is the same socket as the one being inspected — a coincidence of the dev loop that masks the bug entirely.
- Any future integration that wants to drive `fab pane` from outside its target tmux server (CI, cross-host monitoring, a parent process that fans out across multiple servers) has the same problem.
- Workarounds available to callers today are all bad: (a) set `TMUX=socket,pid,pane_id` — requires the caller to know the session/pane IDs of panes it is trying to discover (circular); (b) set `TMUX_TMPDIR` — only helps for default-named sockets under standard tmpdir, not for named sockets like `runKit`; (c) unset `TMUX` and hope the default socket matches — it usually doesn't.

**Why expose `--server` on fab over alternatives.**

1. *Caller sets `$TMUX` before spawning fab* — rejected. The `$TMUX` value that tmux honors is `socket_path,pid,pane_id`, not a raw socket path. Callers that want "the tmux server named X" don't have a pid/pane_id to inject. Even when they do, it's fragile: `$TMUX` means "you are currently attached to this pane", not "please query this socket". Some internal tmux code paths behave differently when `$TMUX` is set (e.g., refusing nested `attach`). Abusing it as a socket selector is a misuse.
2. *Caller sets `$TMUX_TMPDIR`* — rejected. Selects a directory, not a specific socket. Only works if the target socket is named `default` and lives in a dedicated tmpdir. Doesn't address named sockets at all.
3. *fab inspects `$TMUX_SOCKET` / custom env var* — rejected. Adds a hidden coupling via env vars and would need to be documented as a fab-specific convention. A CLI flag is more discoverable (`fab pane map --help`) and easier to plumb through in subprocess-style callers that already build argv slices.
4. *Expose `--server <name>` (matching `tmux -L <name>`)* — accepted. Matches tmux's own naming. Trivially threaded through: every internal `exec.Command("tmux", …)` call in the pane subcommands prepends `-L <server>` when the flag is set. Zero effect on existing callers who don't pass the flag.

## What Changes

All changes are confined to the four `fab pane` subcommands and their shared helpers. No new dependencies, no new config, no breaking changes.

### 1. Add `--server` flag to each `pane` subcommand

Register a persistent string flag on the `pane` parent cobra command so it applies to every subcommand:

```go
// In the pane command's init (src/go/fab/cmd/fab/panemap.go or the parent pane.go that owns the cobra Command group)
paneCmd.PersistentFlags().StringP("server", "L", "", "Target tmux socket label (passed as `tmux -L <name>`). Defaults to $TMUX / tmux default socket.")
```

Short flag `-L` mirrors tmux's own convention. Empty default preserves current behavior (inherit `$TMUX` / default socket).

Each subcommand's `RunE` reads the flag value (`cmd.Flags().GetString("server")`) and plumbs it down to the tmux-invoking helpers.

### 2. Thread `server` through internal tmux helpers

Current helpers hardcode `exec.Command("tmux", …)`. Refactor them to accept a `server string` parameter and prepend `-L <server>` when non-empty.

**`src/go/fab/cmd/fab/panemap.go`**:

- `discoverPanes(mode sessionMode, sessionName string)` → `discoverPanes(mode sessionMode, sessionName, server string)`
- `discoverSessionPanes(name string)` → `discoverSessionPanes(name, server string)` — at line 140, build `args := []string{"list-panes", "-s", "-F", tmuxPaneFormat}` and prepend `-L <server>` if set.
- `discoverAllSessions()` → `discoverAllSessions(server string)` — at line 153, same treatment for the `list-sessions` call.

**`src/go/fab/internal/pane/pane.go`** (shared with `capture`, `send`, `process`):

- `ValidatePane(paneID string)` → `ValidatePane(paneID, server string)` — line 33, `tmux list-panes -a`.
- `ResolvePanePID(paneID string)` → `ResolvePanePID(paneID, server string)` — line 47, `tmux display-message`.
- `ResolvePanePath(paneID string)` → `ResolvePanePath(paneID, server string)` — line 63, same.
- Any other `exec.Command("tmux", …)` sites in this package.

**`src/go/fab/cmd/fab/pane_capture.go`, `pane_send.go`, `pane_process.go`, `pane_process_linux.go`, `pane_process_darwin.go`**: update every tmux invocation site to receive `server` and prepend `-L <server>` when non-empty.

A small helper lives in `internal/pane/pane.go`, named `withServer` to avoid colliding with the existing `tmuxArgs` local variable in `pane_send.go`:

```go
// withServer prepends -L <server> when server is non-empty, else returns args unchanged.
func withServer(server string, args ...string) []string {
    if server == "" {
        return args
    }
    return append([]string{"-L", server}, args...)
}
// usage: exec.Command("tmux", withServer(server, "list-sessions", "-F", "#{session_name}")...)
```

### 3. Behavior when the flag is absent

Unchanged. Every helper's default path is exactly today's path: `exec.Command("tmux", <args>...)` with no `-L`. So:

- Humans running `fab pane map` from inside a tmux session continue to get the current (inherited-from-`$TMUX`) behavior.
- Scripts that don't know or care about sockets are unaffected.

### 4. Tests

Each `_test.go` for the pane subcommands gets:

- A test asserting the helper prepends `-L <server>` when `server != ""` and does not when `server == ""`. This is a pure-function test on the argv-building helper (`tmuxArgs` or equivalent), independent of actually spawning tmux.
- If the tests currently exercise the tmux subprocess path via a fake or a real socket, extend them to cover `--server` being honored.

No end-to-end tmux integration test is required (existing test strategy doesn't mandate it for similar flags).

### 5. Documentation

- Update `--help` strings — already covered by the cobra flag registration.
- Update the relevant memory file(s) — see Affected Memory.

## Affected Memory

- `fab-workflow/pane-commands`: (new) Create `docs/memory/fab-workflow/pane-commands.md` as the dedicated memory file for the four `fab pane` subcommands. Document the `--server` / `-L` flag, the default-inheritance behavior when absent, and the multi-socket use case (run-kit daemon on a separate socket from the inspected sessions). Register it in `docs/memory/fab-workflow/index.md`.
- `fab-workflow/kit-architecture`: (modify) Trim the per-subcommand detail that will now live in `pane-commands.md`; leave only the high-level statement that the `pane` subcommand group exists and is the sole `fabGoNoConfigArgs` allowlist entry. Optionally cross-link to `pane-commands.md`.

## Impact

**Code areas**:
- `src/go/fab/cmd/fab/panemap.go` — flag registration + `discoverPanes`/`discoverSessionPanes`/`discoverAllSessions`
- `src/go/fab/cmd/fab/pane_capture.go`, `pane_send.go`, `pane_process.go`, `pane_process_linux.go`, `pane_process_darwin.go`
- `src/go/fab/internal/pane/pane.go` — `ValidatePane`, `ResolvePanePID`, `ResolvePanePath`
- Test files alongside each of the above

**Not touched**:
- Any other `fab` subcommand group (`change`, `runtime`, `status`, etc.) — the scope is exactly the pane subcommands.
- run-kit itself. Once fab exposes `--server`, run-kit's `fetchPaneMap` in `app/backend/internal/sessions/sessions.go` will gain an argument `tmux -L runKit`-style; that's a run-kit-side follow-up change, not part of this intake.

**Runtime behavior**:
- Callers that don't pass `--server`: zero change.
- Callers that pass `--server runKit`: all internal `tmux` invocations inside the subcommand run with `-L runKit`.

**Dependencies / APIs**: none added.

**Release**: No behavior change for existing users. Safe to ship in the next minor version.

## Open Questions

- Should the flag also be exposed as a persistent env var (e.g., `FAB_TMUX_SERVER`) for convenience in shells, or is the CLI flag sufficient? (Leaning CLI-only; env vars add hidden coupling.)
- Should we consider `--socket-path` as an alternative to `-L` for callers that know the full path rather than the label? tmux supports both (`-L <name>` and `-S <path>`). Likely yes — cheap to add, matches tmux. Spec stage decides.

> Both questions resolved in Clarifications (Session 2026-04-17). See assumptions #7 and #8.

## Clarifications

### Session 2026-04-17

**Q (Tentative #10)**: Primary memory file to update for the `--server` flag?
**A**: Create a new `docs/memory/fab-workflow/pane-commands.md` as the dedicated memory file for the four pane subcommands. Trim `kit-architecture.md` (which currently owns the per-subcommand detail) to a high-level pointer. Register the new file in `docs/memory/fab-workflow/index.md`.

**Q (Tentative #11)**: Helper function name and location?
**A**: `withServer(server string, args ...string) []string` in `internal/pane/pane.go`. Chose `withServer` over `tmuxArgs` to avoid collision with the existing `tmuxArgs` local variable at `pane_send.go:57`.

### Session 2026-04-17 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 7 | Confirmed | "passthrough is the correct technique" — user endorsed letting `tmux -L` do the work rather than inventing a second selector |
| 8 | Confirmed | — |
| 9 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flag name is `--server` with short `-L`, matching tmux's own `-L <label>` convention | Tmux precedent; every shell user who has run `tmux -L` immediately recognizes this; discoverable via `--help` | S:90 R:85 A:95 D:90 |
| 2 | Certain | Empty flag (default) preserves today's behavior — every helper runs `exec.Command("tmux", …)` with no `-L`, exactly as now | Required for backward compatibility; any other default would break every existing caller | S:95 R:95 A:95 D:95 |
| 3 | Certain | Scope is all four `pane` subcommands (`map`, `capture`, `send`, `process`), not just `map` | The underlying bug (implicit socket inheritance) affects every subprocess-caller of any pane subcommand equally; fixing only `map` leaves the others broken for the same use case | S:90 R:80 A:90 D:90 |
| 4 | Certain | Flag is registered as `PersistentFlags` on the `pane` parent cobra command, not individually on each subcommand | Cobra idiom for shared flags across a command group; one registration, uniform UX | S:90 R:90 A:95 D:90 |
| 5 | Certain | The argv-prepend helper lives close to the pane code (either `internal/pane/pane.go` or a new `internal/tmuxutil/` package) | There is no existing shared tmux-invocation layer in fab-go; introducing one now for exactly this purpose is proportional | S:85 R:80 A:85 D:80 |
| 6 | Certain | Change type is `feat` (new CLI flag, additive capability — callers that don't pass it see no change) | Not a bug fix in fab itself: the bug is that fab *never had* a way to target a non-default socket. Adding that way is a new feature, even though it was motivated by a downstream bug | S:85 R:90 A:90 D:80 |
| 7 | Certain | No `--socket-path` / `-S` flag in the first cut; add if requested during review | Clarified — user confirmed; noted "passthrough is the correct technique" (let `tmux -L` do the work, don't invent a second selector) | S:95 R:85 A:75 D:75 |
| 8 | Certain | No env-var-based alternative (e.g., `FAB_TMUX_SERVER`) in the first cut | Clarified — user confirmed; CLI-only avoids hidden env coupling | S:95 R:80 A:75 D:75 |
| 9 | Certain | Tests cover the argv-building helper as a pure function; no end-to-end tmux integration test added | Clarified — user confirmed; matches existing `pane_*_test.go` strategy | S:95 R:85 A:80 D:75 |
| 10 | Certain | Primary memory file is a new `docs/memory/fab-workflow/pane-commands.md` (not a modification of `kit-architecture.md`) | Clarified — user chose to extract pane subcommand documentation into a dedicated memory file | S:95 R:85 A:60 D:55 |
| 11 | Certain | The helper function is named `withServer` (not `tmuxArgs`) and lives in `internal/pane/pane.go` | Clarified — user confirmed; `withServer` avoids collision with the existing `tmuxArgs` local variable at `pane_send.go:57` | S:95 R:80 A:60 D:55 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
