# Intake: Fix tmux socket leak in Go integration tests

**Change**: 260721-0j0t-fix-tmux-test-socket-leak
**Created**: 2026-07-21

## Origin

Promptless pipeline dispatch (`/fab-proceed` create-intake sub-operation, `{questioning-mode} = promptless-defer`) from a synthesized live-conversation description. The conversation diagnosed the root cause and agreed the fix mechanism before dispatch; those decisions are encoded as Certain/Confident assumptions below.

> Stop fab-kit's Go integration tests leaking tmux socket files (root cause of ~7,100 stale sockets in `/tmp/tmux-$UID`). Three integration tests start ephemeral private tmux servers with unique nano-timestamp socket names; all three run `tmux kill-server` in `t.Cleanup`, but tmux does not unlink its socket file on server exit — a stale socket is only removed when a NEW server rebinds the same path. Because every run generates a fresh unique name, nothing ever rebinds, so every clean test run leaks one socket file per test. Agreed fix: set `t.Setenv("TMUX_TMPDIR", t.TempDir())` before starting the server, so sockets land in the per-test temp dir and die with it.

## Why

**The pain point.** Three Go integration tests each start an ephemeral private tmux server via `tmux -L <unique-name>`:

- `src/go/fab/internal/pane/pane_test.go:448` — `fabtest-<nano36>` (`TestReadAgentStateOption_Integration`)
- `src/go/fab/cmd/fab/pane_send_test.go:181` — `fabtest-send-<nano36>` (`TestPaneSendGate_Integration`)
- `src/go/fab/cmd/fab/panemap_test.go:618` — `fabtest-agree-<nano36>` (`TestMapSendAgentAgreement_Integration`)

All three register `t.Cleanup(func() { _, _ = tmux("kill-server") })`, but **tmux does not unlink its socket file when the server exits** — a stale socket at `/tmp/tmux-$UID/<name>` is only removed when a *new* server rebinds the same path. Every run mints a fresh nano-timestamp name, so nothing ever rebinds, and **every clean test run leaks exactly one socket file per test** (abrupt teardown is not required for the leak). On the reporting machine this had accumulated ~7,100 stale sockets in `/tmp/tmux-$UID`.

**Consequence of not fixing.** Unbounded accumulation of dead socket files in the shared `/tmp/tmux-$UID` directory on every developer/CI machine that runs the suite — directory-listing noise, slow `tmux` server enumeration, and eventual inode/tmpfs pressure.

**Why this approach.** Setting `TMUX_TMPDIR` to a per-test temp dir moves the socket out of the shared `/tmp/tmux-$UID` entirely; the temp dir (and the socket inside it) is deleted when the test ends, regardless of how tmux exits. The mechanism MUST be `t.Setenv` (process-level env), NOT `cmd.Env` scoped to the test's local `tmux` closure: the code under test (`ReadAgentStateOption`, `fab pane send`, `fab pane map`) shells out to `tmux -L <server>` itself and locates the socket via *inherited process env* — env scoped only to the test helper's commands would strand the production code's tmux invocations on `/tmp/tmux-$UID`, splitting the test and the code-under-test onto two different servers. `t.Setenv` is legal here: none of the three tests call `t.Parallel()` (verified by grep at intake time).

## What Changes

Test-only change — no production `.go` behavior changes. Three call sites across two packages (`internal/pane`, `cmd/fab`).

### 1. Per-test private TMUX_TMPDIR (the core fix, all three tests)

In each of the three integration tests, set the process env before the first tmux command:

```go
// Private socket dir: the socket dies with the per-test temp dir
// (tmux never unlinks sockets on server exit — see change 0j0t).
t.Setenv("TMUX_TMPDIR", <per-test private dir>)
```

placed after the `exec.LookPath("tmux")` skip guard and before `server := ...` / the first `tmux(...)` call. Because `t.Setenv` mutates process env, both the test's local `tmux` closure *and* the production code's own `tmux -L <server>` invocations resolve the socket under the private dir (tmux places sockets at `$TMUX_TMPDIR/tmux-$UID/<name>`).

### 2. Socket-path length budget (macOS `sun_path` edge case — must be handled, not theoretical)

Unix socket paths are capped at ~104 bytes on macOS (`sun_path`, including terminating NUL). Measured on the reporting machine: `$TMPDIR` base is 49 chars, so `t.TempDir()` for the longest test name is roughly `/var/folders/.../T/TestReadAgentStateOption_Integration<random>/001` ≈ 98 chars; adding tmux's `/tmux-<uid>/<name>` suffix exceeds the cap. **The naive `t.Setenv("TMUX_TMPDIR", t.TempDir())` would make tmux fail to bind on macOS, and the tests' existing `t.Skipf("could not start tmux server ...")` guard would silently erase their coverage.**

Handle it with both measures agreed in conversation ("length sanity check and/or shorten the socket names" — do both):

- **Shorten the socket names.** The nano-timestamp uniqueness is unnecessary once each test owns a private `TMUX_TMPDIR` — replace `"fabtest-" + strconv.FormatInt(time.Now().UnixNano(), 36)` etc. with short fixed names (e.g. `fabtest`, `fabtest-send`, `fabtest-agree`, kept distinct for debuggability). This also removes the very mechanism that prevented socket rebinding/cleanup.
- **Length sanity check with a short-dir fallback — never a silent skip.** Compute the candidate dir (`t.TempDir()`); if `len(dir) + len("/tmux-<uid>/") + len(name)` would exceed a conservative budget (~103 bytes), fall back to a short directory such as `os.MkdirTemp("/tmp", "fabtest-")` (≈ 20 chars, always fits) registered with `t.Cleanup(func() { os.RemoveAll(dir) })` so the leak guarantee holds on the fallback path too. The guard MUST NOT let an over-long path degrade into the existing `t.Skipf` — path-length problems are detectable and fixable, unlike a missing tmux binary (which legitimately keeps its `t.Skip`).

### 3. Keep the existing `kill-server` cleanup

The `t.Cleanup(func() { _, _ = tmux("kill-server") })` in all three tests stays — it still frees the server *process* promptly; the temp-dir deletion is what now removes the socket *file*.

### 4. Helper extraction is optional

No shared helper currently exists across the two packages, and none is required. A small unexported test helper per package (one in `internal/pane`, one in `cmd/fab` serving its two tests) MAY be extracted to avoid triplicating the dir-selection/length-guard logic, but inlining at each of the three call sites is acceptable.

### Verification

- `go test ./internal/pane/ ./cmd/fab/ -run '_Integration'` from `src/go/fab` passes with tmux installed (integration paths exercised, not skipped).
- Snapshot `/tmp/tmux-$UID` before and after a test run: **zero new socket files**.
- Existing unit tests in the three files remain green.

## Affected Memory

None — test-infrastructure-only change; no spec-level behavior changes. The `runtime` domain documents `fab pane` behavior, which is untouched.

## Impact

- `src/go/fab/internal/pane/pane_test.go` — `TestReadAgentStateOption_Integration` (server setup around line 448)
- `src/go/fab/cmd/fab/pane_send_test.go` — `TestPaneSendGate_Integration` (around line 181)
- `src/go/fab/cmd/fab/panemap_test.go` — `TestMapSendAgentAgreement_Integration` (around line 618)
- No production code, no CLI signatures (no `_cli-fab.md` update needed), no skills (no SPEC mirrors), no memory updates.
- Constitution VII (Test Integrity) note: this modifies test *infrastructure* (server lifecycle), not assertions — the tests still verify the same spec conformance; no implementation code is bent to satisfy tests.

## Open Questions

None — the mechanism, constraints, and edge-case handling were resolved in the originating conversation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix mechanism is `t.Setenv("TMUX_TMPDIR", <private dir>)` (process env) set before the first tmux command — NOT `cmd.Env` on the test's local tmux closure | Discussed — explicit conversation decision with rationale: the code under test shells out to `tmux -L` itself and locates the socket via inherited process env; closure-scoped env would strand production code on `/tmp/tmux-$UID` | S:95 R:85 A:95 D:95 |
| 2 | Certain | `t.Setenv` is legal in all three tests | Verified at intake — grep shows no `t.Parallel()` in any of the three files | S:90 R:90 A:100 D:95 |
| 3 | Certain | Keep the existing `tmux kill-server` `t.Cleanup` in all three tests | Discussed — still frees the server process promptly; temp-dir deletion handles the socket file | S:90 R:95 A:95 D:90 |
| 4 | Confident | Handle the macOS `sun_path` (~104-byte) limit with BOTH measures: shorten socket names to short fixed strings (nano-timestamp uniqueness obsolete under per-test TMUX_TMPDIR) AND add a length guard that falls back to a short `os.MkdirTemp("/tmp", ...)` dir (with `t.Cleanup` removal) rather than letting `t.Skipf` silently erase coverage | Conversation directed "length sanity check and/or shorten the socket names"; doing both is the conservative reading. Fallback (vs. hard `t.Fatalf`) chosen because measurement shows `t.TempDir()` exceeds the cap on macOS for the longest test name — failing would break every macOS run; exact guard mechanics decided at apply | S:70 R:90 A:85 D:55 |
| 5 | Certain | No shared cross-package helper required; a small per-package test helper MAY be extracted, or the change inlined per call site | Discussed — explicitly marked "optional, not required" in conversation; three call sites across two packages | S:85 R:95 A:90 D:80 |
| 6 | Certain | Affected Memory is none — test-only change, no spec-level behavior change, no CLI signature change | Template rule: implementation-only changes need no memory updates; `runtime` domain documents `fab pane` behavior, which is untouched | S:80 R:90 A:90 D:85 |
| 7 | Confident | Change type is `test` (explicit override of the keyword inference, which matches `fix` at priority 1) | Taxonomy (`docs/specs/change-types.md`): `test` covers "Test additions or modifications … fix flaky test"; all modified files are `*_test.go` | S:70 R:85 A:80 D:60 |
| 8 | Confident | Non-goal: no cleanup of the already-accumulated stale sockets ships in this change | Scope framed as root-cause fix; existing `/tmp/tmux-$UID/fabtest-*` files are machine state, removable manually — not a repo artifact | S:65 R:90 A:75 D:60 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
