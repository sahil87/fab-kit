# Intake: Fix pane_ready_test.go killing the host tmux server from inside a pane

**Change**: 260812-kgam-pane-ready-test-host-kill
**Created**: 2026-08-12

## Origin

`/fab-new` one-shot with a fully-specified fix prompt (root cause, required fix items 1–5, exact verification recipe, deliverable). The prompt was written after a live incident and pre-pins the two judgment calls: the empty-vs-unset `$TMUX` ambiguity MUST be verified empirically, and the destructive repro MUST run only against a disposable `-L fixcheck` server. Zero SRAD questions were asked — the input left none open.

> # Fix: pane_ready_test.go kills the host tmux server when run inside a tmux pane
>
> ## Context
>
> Repo: `~/code/sahil87/fab-kit`. On 2026-08-12 10:33 IST, running `go test ./cmd/fab/` from a Claude agent pane on the `fabKit1` tmux server **killed the entire fabKit1 server**, taking ~8 live agent sessions down. This is a recurring death vector (previously seen 2026-08-05 from an agent one-liner); this instance is committed code.
>
> ## Root cause
>
> `src/go/fab/cmd/fab/pane_ready_test.go`, subtest **"server null on the default socket"** (~line 149–182):
>
> - It tests that `fab pane ready <pane> --json` reports `server: null` when the pane lives on the **default socket** (no `-L`), so the test deliberately uses no `-L` on any tmux invocation.
> - Isolation is attempted via `t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "default"))`.
> - **This isolation is insufficient inside a tmux pane.** tmux socket resolution is `-L`/`-S` > `$TMUX` > `TMUX_TMPDIR`. When the test process runs inside a pane, `$TMUX` is inherited and silently wins: the bare `tmux new-session -d -s s` lands on the HOST server, and the `t.Cleanup(func() { tmux("kill-server") })` (~line 165) kills the host — the test's own process, every sibling agent, and (collaterally) leaks other tests' scratch servers because their cleanups never run.
>
> The other tmux-backed tests in the package (`pane_send_test.go`, `dispatch_*_test.go`, `panemap_test.go`, `internal/pane/pane_test.go`) are safe — their helpers pass explicit `-L <name>` or verified `-S <path>` on every call.
>
> ## Required fix
>
> 1. In the "server null on the default socket" subtest, neutralize `$TMUX` (and `$TMUX_PANE`) for **both** the test's own `tmux` helper **and** the command under test — note the command under test shells out to tmux itself and resolves the socket from the same process env, which is why the test already uses `t.Setenv` for `TMUX_TMPDIR` rather than per-command env. `t.Setenv("TMUX", "")` + `t.Setenv("TMUX_PANE", "")` is the expected shape, BUT verify that tmux treats an empty (as opposed to unset) `TMUX` as absent — check tmux's behavior empirically (see Verification). If empty is not equivalent to unset, build an explicit filtered environment instead (e.g. `os.Environ()` minus `TMUX`/`TMUX_PANE`, applied via `cmd.Env` in the helper AND plumbed to the command under test) — do not weaken the test's intent.
> 2. **Keep the test's semantics**: it must still exercise the default socket (no `-L` in the tmux args or the `fab pane ready` invocation), because the point is the `toNullable` null mapping for the empty `--server` value.
> 3. Add a short comment on WHY the env is scrubbed: `$TMUX` overrides `TMUX_TMPDIR`, so without this the test kills the host server when run from inside a pane.
> 4. **Audit the rest of the test tree** for the same latent pattern: any test that invokes tmux (directly or via the code under test) relying on TMUX_TMPDIR-only isolation without `-L`/`-S` and without scrubbing `$TMUX`. Fix any you find the same way. Suggested sweep: `grep -rn 'TMUX_TMPDIR' src/go --include='*_test.go'` and inspect each hit's tmux invocations for missing `-L`/`-S`.
> 5. Consider (and implement if cheap) defense-in-depth: the shared `tmuxSocketDir` helper (or a new shared guard) could scrub `TMUX`/`TMUX_PANE` for every test that opts into private-socket isolation, so future tests can't reintroduce this.
>
> ## Verification (do this exactly — the pre-fix failure mode is destructive)
>
> Never run the unfixed test inside a live/valuable tmux server. Use a disposable one:
>
> 1. Baseline sanity outside tmux: `cd src/go/fab && go test ./cmd/fab/ -run TestPaneReady -v` — all pass.
> 2. Destructive-repro check, safely: start a throwaway server and run the test inside it, then confirm the throwaway server SURVIVES:
>    ```sh
>    tmux -L fixcheck new-session -d -s guard 'sleep 600'
>    tmux -L fixcheck new-window -t guard -c "$PWD" \
>      'cd src/go/fab && go test ./cmd/fab/ -run TestPaneReady -v; echo EXIT=$?; sleep 120'
>    # wait for the test, then:
>    tmux -L fixcheck list-sessions   # must still work — server alive, session guard present
>    tmux -L fixcheck kill-server     # cleanup, explicitly scoped
>    ```
>    (Pre-fix, the inner test run would kill the fixcheck server itself — that is the repro. Post-fix it must survive with the test passing.)
> 3. Confirm no scratch servers/sockets leaked: check `ls /tmp/tmux-$(id -u)/` and `ps -ef | grep 'tmux -L fabtest'` before/after — no new entries.
> 4. Run the full package once more outside tmux: `go test ./cmd/fab/`.
>
> ## Deliverable
>
> A minimal diff to the test file(s) + the audit result (list of other sites checked, even if clean). Follow the repo's normal fab change workflow if instructed by the project's conventions; otherwise a plain branch + PR is fine.

## Why

**The pain point.** `src/go/fab/cmd/fab/pane_ready_test.go`'s subtest `"server null on the default socket"` (lines 150–182, inside `TestPaneReady_JSON`) deliberately uses no `-L` anywhere — that is the point of the test (the empty `--server` value must map to JSON `null` via `toNullable`). Its only isolation is `t.Setenv("TMUX_TMPDIR", tmuxSocketDir(t, "default"))` (line 157). tmux socket resolution is `-L`/`-S` > `$TMUX` > `TMUX_TMPDIR`, so when the test process runs **inside a tmux pane**, the inherited `$TMUX` silently wins: the bare `tmux new-session -d -s s` (line 162) lands on the HOST server, and the `t.Cleanup(func() { tmux("kill-server") })` (line 165) kills the host. On 2026-08-12 10:33 IST this took down the `fabKit1` server and ~8 live agent sessions; it is a recurring death vector (previously 2026-08-05 from an agent one-liner) — this instance is committed code, so any agent running `go test ./cmd/fab/` from a pane re-triggers it.

**Consequence of not fixing.** Every future in-pane run of the package (a completely normal thing for agent workers to do — dispatch pane workers live in tmux by design) kills the operator's entire tmux server: the test's own process, all sibling agents, and — collaterally — leaks other tests' scratch servers because their cleanups never run.

**Why this approach.** Verified at intake, the repo already contains BOTH halves of the answer, applied inconsistently:

- Change `260721-0j0t-fix-tmux-test-socket-leak` established per-test-private `TMUX_TMPDIR` via process-level `t.Setenv` (process-level because the code under test shells out to tmux itself and resolves the socket from inherited process env) — but never addressed `$TMUX` precedence.
- `dispatch_start_test.go` later encoded a full "recorded repo discipline" for unscoped default-socket tests (comments at lines 512–518 and 708–724): private `TMUX_TMPDIR` + `$TMUX` must be empty (there enforced by **refusal**: `t.Fatalf` if `$TMUX` is set, lines 529–532/732–735) + `os.Stat`-verify the private socket **before** registering any destructive cleanup + scope `kill-server` with explicit `-S <verified path>`.

The pane_ready subtest predates/misses that discipline entirely. The fix chosen here is **scrubbing** `$TMUX`/`$TMUX_PANE` rather than the dispatch_start-style refusal, because scrubbing keeps the test *running and passing* inside panes (refusal would merely convert the kill into a red test), and the prompt explicitly requires it. The one spot a naive scrub can silently not work — whether tmux treats an **empty** `$TMUX` as absent, versus only an **unset** one — is pinned as a mandatory empirical check before the shape is finalized.

## What Changes

Test-only change — no production `.go` behavior changes, no CLI signature changes (so no `_cli-fab.md` update), no migration.

### 1. Scrub `$TMUX`/`$TMUX_PANE` in the "server null on the default socket" subtest (the core fix)

In `src/go/fab/cmd/fab/pane_ready_test.go` lines 150–182, neutralize `$TMUX` and `$TMUX_PANE` at **process level** — it must cover both the test's local `tmux` closure AND the command under test (`runPaneCmd` → `fab pane ready`, which shells out to tmux itself and resolves the socket from the same process env; this is exactly why the existing `TMUX_TMPDIR` isolation already uses `t.Setenv` rather than per-command `cmd.Env`).

Expected shape, gated on the empirical check below:

```go
// $TMUX overrides TMUX_TMPDIR in tmux's socket resolution (-L/-S > $TMUX >
// TMUX_TMPDIR). Without this scrub, running the test from inside a tmux pane
// lands the bare new-session on the HOST server — and the kill-server cleanup
// below kills the host.
t.Setenv("TMUX", "")
t.Setenv("TMUX_PANE", "")
```

**Mandatory empirical gate (do this before committing to the shape):** verify that tmux treats an *empty* `$TMUX` as absent. Probe from inside a disposable pane (e.g. on the `-L fixcheck` server from the verification recipe): with `TMUX="" TMUX_TMPDIR=<private dir>`, run a bare `tmux new-session -d` and confirm the socket binds under the private dir (`os.Stat`/`ls`), not the host's. If empty is NOT equivalent to unset, do NOT weaken the test: use a real unset instead — `os.Unsetenv("TMUX")`/`os.Unsetenv("TMUX_PANE")` with the prior values restored via `t.Cleanup` (mirroring `t.Setenv`'s restore contract; the subtest never calls `t.Parallel()`, verified — `t.Setenv` at line 157 already enforces that), or the prompt's alternative of an explicitly filtered `os.Environ()` minus `TMUX`/`TMUX_PANE` plumbed to **both** the helper and the command under test.

**Semantics stay intact:** no `-L` may appear in the subtest's tmux args or the `fab pane ready` invocation — the empty `--server` value exercising the `toNullable` null mapping is the test's reason to exist.

### 2. Adopt the verified-socket cleanup discipline in the same subtest (defense-in-depth, cheap, precedented)

Mirror `dispatch_start_test.go`'s recorded discipline (lines 536–545): after starting the server, `os.Stat` the expected private socket path (`filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")`) and `t.Fatalf` if it did not bind — **before** registering the kill-server cleanup — and scope the cleanup with explicit `-S <verified path>` instead of a bare `tmux kill-server`. This makes the destructive cleanup structurally unable to reach a real server even if the env isolation ever regresses. The verified `-S` on the *cleanup only* does not touch the test's semantics: the code under test (`fab pane ready` with no `--server`) still resolves through tmux's default-socket path.

### 3. Shared-guard defense-in-depth in `tmuxSocketDir` (prompt item 5 — implement if the empirical gate allows)

There are two duplicate `tmuxSocketDir` copies: `src/go/fab/cmd/fab/pane_send_test.go:237` and `src/go/fab/internal/pane/pane_test.go:453`. If (and only if) the empirical check confirms empty-equals-unset (or a clean unset mechanism is used), add the `TMUX`/`TMUX_PANE` scrub inside **both** copies so every test that opts into private-socket isolation gets it automatically and future tests cannot reintroduce the death vector. This is harmless for the `-L`-scoped majority (`-L` outranks `$TMUX` anyway) and removes the hazard class at the root.

**Interaction to reconcile (do not skip):** `dispatch_start_test.go`'s refusal guards (lines 529–532 and 732–735, `t.Fatalf` when `$TMUX` is non-empty) run **after** their `tmuxSocketDir` calls — a shared scrub makes those refusals dead code and, deliberately, makes those tests runnable inside panes (they already carry the verified-socket `-S`-scoped cleanup, so this is safe). If the shared scrub lands, update those guards and their "SAFETY"/"recorded repo discipline" comments to reflect the new invariant (scrub-then-verify rather than refuse) instead of leaving stale prose — comment accuracy is a review must-fix class in this repo. Note `startPrivateTmuxWithPane` sets `$TMUX` *afterwards* on purpose (line 548 area — simulating a dispatcher inside a pane); the scrub must not disturb that, and it doesn't (the set happens after). If the empirical gate fails and the local-filtered-env fallback is used instead, keep the fix local to the pane_ready subtest and leave `tmuxSocketDir` and the refusal guards untouched.

### 4. Audit the test tree for the same latent pattern (deliverable includes the site list)

Sweep run at intake (`grep -rn 'TMUX_TMPDIR' src/go --include='*_test.go'`) — apply must re-verify each hit's tmux invocations and record the result in the PR:

- `cmd/fab/pane_ready_test.go:157` — **THE BUG** (no `-L`, no `$TMUX` scrub, unscoped kill-server). Fixed by this change.
- `cmd/fab/dispatch_start_test.go` (~427, 477, 528, 731) — default-socket sites; already guarded by the refusal + verified-`-S` discipline. Safe (not kill vectors); functionally unchanged unless the §3 shared scrub lands, in which case they become pane-runnable.
- All remaining hits (`dispatch_reap/restart/status/wait/kill/deliver/open_test.go`, `pane_exitcode_test.go`, `pane_send_test.go`, `panemap_test.go`, `internal/pane/pane_test.go`) — pair `TMUX_TMPDIR` with an explicit `-L <fabtest-*>` server name on every tmux call (via `newTmuxPane`/local closures). `-L` outranks `$TMUX`, so these are safe; confirm during apply that no stray unscoped call hides in them.

### Verification (follow the prompt's recipe exactly — the pre-fix failure mode is destructive)

1. Baseline outside tmux: `cd src/go/fab && go test ./cmd/fab/ -run TestPaneReady -v` — all pass.
2. Empirical empty-vs-unset probe (feeds §1's gate) — on a disposable server, never a live one.
3. Destructive-repro check against a throwaway `-L fixcheck` server per the Origin recipe: run `-run TestPaneReady` in a window on that server; post-fix the fixcheck server MUST survive (`tmux -L fixcheck list-sessions` still answers, session `guard` present) with the test passing. Clean up with the explicitly-scoped `tmux -L fixcheck kill-server`.
4. Leak check before/after: `ls /tmp/tmux-$(id -u)/` and `ps -ef | grep 'tmux -L fabtest'` — no new entries.
5. Full package outside tmux: `go test ./cmd/fab/`. If §3's shared scrub lands, also run `go test ./internal/pane/` (second `tmuxSocketDir` copy) and re-run the affected `dispatch_start` tests.

**Never run the unfixed test inside a live/valuable tmux server.** This session itself runs inside a worktree — check `$TMUX` before any bare test invocation and use the fixcheck recipe for anything in-pane.

## Affected Memory

None — test-only fix; no spec-level behavior changes (Constitution: implementation-only changes don't need memory updates). If hydrate judges the now-unified tmux-test-isolation discipline (scrub + verify + `-S`-scoped destructive cleanup) worth capturing as a durable pattern, a Design Decision note under `runtime` is the natural home, but no existing memory file makes a claim this change invalidates.

## Impact

- `src/go/fab/cmd/fab/pane_ready_test.go` — the core fix (§1, §2).
- `src/go/fab/cmd/fab/pane_send_test.go` + `src/go/fab/internal/pane/pane_test.go` — the two `tmuxSocketDir` copies, only if §3's shared scrub lands.
- `src/go/fab/cmd/fab/dispatch_start_test.go` — refusal-guard/comment reconciliation, only if §3 lands.
- No production code, no CLI surface, no docs/memory, no migration. Test scope: `go test ./cmd/fab/` (+ `./internal/pane/` if §3 lands).

## Open Questions

None — the input is exhaustively specified; the one genuine ambiguity (empty vs unset `$TMUX`) is deliberately delegated to an empirical check during apply, with both outcomes' fix shapes pre-agreed (§1).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Root cause is inherited `$TMUX` outranking `TMUX_TMPDIR` in the default-socket subtest, with the unscoped `kill-server` cleanup as the kill mechanism | Verified at intake against `pane_ready_test.go:150-182`; matches tmux's documented resolution order and the in-repo precedent comments (`dispatch_start_test.go:512-518`) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Fix shape: process-level scrub of `$TMUX`/`$TMUX_PANE`, semantics unchanged (no `-L` anywhere), why-comment added | Explicit in the prompt (items 1–3); process-level is forced by the command under test resolving env itself | S:95 R:85 A:90 D:90 |
| 3 | Confident | The empty-vs-unset `$TMUX` question gates the shape: `t.Setenv("TMUX", "")` only if tmux treats empty as absent, else real unset (`os.Unsetenv` + `t.Cleanup` restore) or filtered env | Prompt pins the empirical check as mandatory; the choice among fallback mechanisms is the agent's, both preserve intent | S:85 R:80 A:75 D:70 |
| 4 | Confident | Adopt the verified-socket `-S`-scoped kill-server cleanup in the fixed subtest (defense-in-depth) | Prompt item 5 says "implement if cheap"; it is cheap and already precedented verbatim in-repo (`dispatch_start_test.go:536-545`) | S:65 R:85 A:85 D:75 |
| 5 | Confident | Shared-guard scrub inside both `tmuxSocketDir` copies, conditional on the empirical gate, with the dispatch_start refusal guards + SAFETY comments reconciled if it lands; local-only fix as fallback | Prompt item 5 invites it; the refusal-guard interaction was found at intake and the reconciliation path is pre-agreed — stale-comment drift is a review must-fix class here | S:60 R:75 A:60 D:40 |
| 6 | Certain | Verification follows the prompt's recipe exactly: disposable `-L fixcheck` repro server, leak check, full package outside tmux; never run the unfixed test in a live server | Explicit, step-by-step in the prompt; destructive failure mode makes deviation unacceptable | S:95 R:90 A:90 D:95 |
| 7 | Confident | `dispatch_start_test.go`'s guarded default-socket sites are audit-classified "safe (refuse, not kill)" and stay functionally unchanged unless §3 lands | Read at intake: refusal guard + verified `-S` cleanup present at every site; they fail-red inside panes rather than killing, which the prompt's audit item tolerates | S:70 R:80 A:80 D:65 |
| 8 | Certain | Normal fab workflow: test-only diff on branch `260812-kgam-pane-ready-test-host-kill` + PR; no `_cli-fab.md` update (no CLI signature change), no migration, change type `fix` | Constitution constraints map deterministically: no command signature touched, no user-data restructuring | S:85 R:90 A:95 D:90 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
