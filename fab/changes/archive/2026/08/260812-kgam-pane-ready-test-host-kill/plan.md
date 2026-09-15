# Plan: Fix pane_ready_test.go killing the host tmux server from inside a pane

**Change**: 260812-kgam-pane-ready-test-host-kill
**Intake**: `intake.md`

## Requirements

### Testing: tmux isolation in the default-socket pane-ready subtest

#### R1: Process-level `$TMUX`/`$TMUX_PANE` scrub, semantics unchanged
The `"server null on the default socket"` subtest (`src/go/fab/cmd/fab/pane_ready_test.go:150-182`) MUST neutralize `$TMUX` and `$TMUX_PANE` at **process level** — covering both the test's local `tmux` closure and the command under test (`fab pane ready` shells out to tmux itself and resolves the socket from inherited process env). The subtest MUST keep its default-socket semantics: no `-L`/`-S` in the tmux args or the `fab pane ready` invocation (the empty `--server` value exercising `toNullable`'s null mapping is the test's purpose). A short comment MUST state WHY: `$TMUX` outranks `TMUX_TMPDIR` in tmux socket resolution (`-L`/`-S` > `$TMUX` > `TMUX_TMPDIR`), so without the scrub the test kills the host server when run from inside a pane.

- **GIVEN** the test process runs inside a tmux pane (inherited `$TMUX`)
- **WHEN** the subtest starts its bare `tmux new-session -d -s s`
- **THEN** the server binds under the private `TMUX_TMPDIR`, never the host socket
- **AND** `fab pane ready <pane> --json` still reports `server: null`

#### R2: Scrub mechanism gated on an empirical empty-vs-unset check
The scrub mechanism MUST be chosen by an empirical probe, not assumption: `t.Setenv("TMUX", "")` + `t.Setenv("TMUX_PANE", "")` is acceptable **only if** tmux demonstrably treats an *empty* `$TMUX` as absent. If empty is not equivalent to unset, use a real unset (`os.Unsetenv` with prior values restored via `t.Cleanup`, mirroring `t.Setenv`'s restore contract — the subtest is non-parallel) or an explicitly filtered `os.Environ()` minus `TMUX`/`TMUX_PANE` plumbed to **both** the helper and the command under test. The probe result MUST be recorded (plan `## Notes`).

- **GIVEN** a disposable tmux server and a private `TMUX_TMPDIR`
- **WHEN** a bare `tmux new-session -d` runs with `TMUX=""` (empty, set)
- **THEN** the socket binds under the private dir ⇒ `t.Setenv` shape is valid; otherwise the fallback mechanism is used instead

#### R3: Verified-socket, `-S`-scoped destructive cleanup
The subtest MUST adopt the recorded repo discipline (`dispatch_start_test.go:536-545`): after starting the server, `os.Stat` the expected private socket path (`filepath.Join(socketDir, "tmux-"+strconv.Itoa(os.Getuid()), "default")`) and fail the test if it did not bind — **before** registering the kill-server cleanup — and scope the cleanup with explicit `-S <verified path>` instead of a bare `tmux("kill-server")`. The code under test still resolves through tmux's default-socket path (the `-S` rides the *cleanup only*).

- **GIVEN** the env isolation ever regresses (e.g. a future edit reintroduces `$TMUX` leakage)
- **WHEN** the private socket did not bind at the expected path
- **THEN** the test fails before any destructive cleanup is registered, and no `kill-server` can reach a real server

#### R4: Shared-guard defense-in-depth (conditional on R2's gate)
If (and only if) R2's probe confirms a clean process-level scrub mechanism, both duplicate `tmuxSocketDir` copies (`src/go/fab/cmd/fab/pane_send_test.go:237`, `src/go/fab/internal/pane/pane_test.go:453`) SHOULD apply the `TMUX`/`TMUX_PANE` scrub so every private-socket test gets it automatically. When this lands, `dispatch_start_test.go`'s now-dead refusal guards (lines 529-532, 732-735) and their "SAFETY"/"recorded repo discipline" comments MUST be reconciled to the new invariant (scrub-then-verify rather than refuse) — no stale prose. `startPrivateTmuxWithPane`'s deliberate later `t.Setenv("TMUX", privateSocket+",1,0")` (~line 548) MUST remain undisturbed. If R2's gate fails, the fix stays local to the pane_ready subtest and these files stay untouched.

- **GIVEN** the shared scrub landed in `tmuxSocketDir`
- **WHEN** a future test opts into private-socket isolation via the helper and forgets `$TMUX`
- **THEN** the scrub has already neutralized it — the death vector cannot be reintroduced
- **AND** the dispatch_start default-socket tests become runnable inside panes (their verified-`-S` cleanups make that safe)

#### R5: Audit deliverable
Every `TMUX_TMPDIR` test site (the intake §4 sweep list) MUST be re-verified for unscoped tmux invocations and the classified site list recorded in this plan's `## Notes` and the PR body — including the clean sites.

- **GIVEN** the sweep `grep -rn 'TMUX_TMPDIR' src/go --include='*_test.go'`
- **WHEN** each hit's tmux invocations are inspected for missing `-L`/`-S`/scrub
- **THEN** each site is classified (fixed / guarded-safe / `-L`-scoped-safe) and the list is recorded

#### R6: Verification per the pinned recipe — never endanger a live server
Verification MUST follow the intake's recipe: (1) baseline with `$TMUX` absent from the test process env; (2) in-pane survival repro against a disposable `tmux -L fixcheck` server — post-fix the fixcheck server MUST survive with the test passing; (3) before/after leak check (`/tmp/tmux-$(id -u)/`, stray `fabtest` servers); (4) full `./cmd/fab/` package with `$TMUX` absent (+ `./internal/pane/` if R4 lands). The unfixed test MUST NOT run with an inherited live `$TMUX`. **This orchestrator session itself runs inside a pane on the live `fabKit1` server** — every direct `go test` invocation from this session MUST be wrapped `env -u TMUX -u TMUX_PANE`, and the with-`$TMUX` path exercised only inside the fixcheck server.

- **GIVEN** this session's own `TMUX=/tmp/tmux-1001/fabKit1,…`
- **WHEN** any `go test` touching tmux-backed tests runs from this session
- **THEN** it runs under `env -u TMUX -u TMUX_PANE` (baseline) or inside the disposable fixcheck server (in-pane repro), never bare

### Design Decisions

#### Scrub, not refuse, for the pane_ready default-socket subtest
**Decision**: Neutralize `$TMUX`/`$TMUX_PANE` (scrub) so the subtest runs and passes inside panes, layered with the verified-socket `-S`-scoped cleanup.
**Why**: Scrubbing keeps coverage alive everywhere agents actually run tests (pane workers live in tmux by design); the verified-`-S` cleanup makes the destructive call structurally unable to reach a real server even if isolation regresses.
**Rejected**: The `dispatch_start_test.go`-style refusal (`t.Fatalf` when `$TMUX` set) — it prevents the kill but converts every in-pane package run into a red test, and the incident prompt explicitly requires the scrub shape.
*Introduced by*: 260812-kgam-pane-ready-test-host-kill

### Non-Goals

- Refactoring the pre-existing `tmuxSocketDir` duplication into one shared package — minimal diff; the duplication predates this change.
- Any production (non-test) code change; any CLI surface, docs/memory, or migration work.

## Tasks

### Phase 1: Setup

- [x] T001 Empirical probe (R2 gate): on a disposable tmux server, determine whether tmux treats empty `$TMUX` as absent — with `TMUX="" TMUX_TMPDIR=<private dir>`, run a bare `tmux new-session -d` from inside a pane-like env and check where the socket binds; record the result in `## Notes` <!-- R2 -->

### Phase 2: Core Implementation

- [x] T002 Fix `src/go/fab/cmd/fab/pane_ready_test.go` "server null on the default socket": add the process-level `TMUX`/`TMUX_PANE` scrub (mechanism per T001), the why-comment, and the verified-socket `-S`-scoped kill-server cleanup (os.Stat before registering); keep no-`-L` semantics <!-- R1, R3 -->
- [x] T003 Conditional (T001-gated): add the scrub inside both `tmuxSocketDir` copies (`cmd/fab/pane_send_test.go:237`, `internal/pane/pane_test.go:453`); reconcile `dispatch_start_test.go` refusal guards + SAFETY comments to the new invariant; verify `startPrivateTmuxWithPane`'s later `$TMUX` set is undisturbed. If the gate failed, skip and document in `## Notes` <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Audit sweep: re-verify every `TMUX_TMPDIR` hit from the intake §4 list for unscoped tmux calls; record the classified site list in `## Notes` (feeds the PR body) <!-- R5 -->
- [x] T005 Verification recipe: baseline `env -u TMUX -u TMUX_PANE go test ./cmd/fab/ -run TestPaneReady -v`; fixcheck in-pane survival repro (disposable `-L fixcheck` server per intake recipe); before/after leak check; full package `env -u TMUX -u TMUX_PANE go test ./cmd/fab/` (+ `./internal/pane/` if T003 landed) <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: The subtest scrubs `$TMUX`/`$TMUX_PANE` at process level (covering helper and command under test), retains no-`-L` default-socket semantics, and carries the why-comment
- [x] A-002 R2: The empirical empty-vs-unset probe ran, its result is recorded in `## Notes`, and the implemented mechanism matches the finding
- [x] A-003 R3: `os.Stat` socket verification precedes the cleanup registration and the kill-server cleanup is scoped with `-S <verified path>`

### Behavioral Correctness

- [x] A-004 R4: Either both `tmuxSocketDir` copies carry the scrub AND `dispatch_start_test.go`'s guards/comments are reconciled (no stale SAFETY prose), or the gate-failed fallback is documented and those files are untouched
- [x] A-005 R1: Post-fix, the subtest passes when run inside a tmux server and that server survives (the fixcheck repro)

### Scenario Coverage

- [x] A-006 R5: The audit site list (every `TMUX_TMPDIR` hit, classified, clean sites included) is recorded in `## Notes` for the PR deliverable
- [x] A-007 R6: Baseline and full-package runs are green with `$TMUX` absent; leak check shows no new sockets/servers; no verification step ran the unfixed test against a live server

### Code Quality

- [x] A-008 Pattern consistency: The fix follows the in-repo recorded discipline (verified socket, `-S`-scoped destructive calls) and its comment style matches `dispatch_start_test.go`'s SAFETY comments
- [x] A-009 No unnecessary duplication: Existing helpers reused; the pre-existing `tmuxSocketDir` duplication is not expanded beyond the mirrored scrub (and not refactored — Non-Goal)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

### T001 probe result (R2 gate)

**Empty ≡ unset** on tmux 3.6a (this machine): with `TMUX=""` (empty, set) + private `TMUX_TMPDIR`, a bare `tmux new-session -d` bound the default socket under the private dir (`$dir/tmux-$UID/default`) and the live fabKit1 server saw no new session. `t.Setenv("TMUX", "")` is therefore a valid scrub; the R4 shared-scrub gate PASSED.

### T004 audit result (R5 — the PR-body site list)

Sweep: `grep -rn 'TMUX_TMPDIR' src/go --include='*_test.go'` + `exec.Command("tmux"` scan for `-L`/`-S`-less call sites.

| Site | Classification |
|------|----------------|
| `cmd/fab/pane_ready_test.go` "server null on the default socket" | **THE BUG — FIXED**: deliberate no-`-L` + previously no `$TMUX` scrub + bare `kill-server`. Now: subtest-level scrub + tmuxSocketDir scrub + `os.Stat`-verified `-S`-scoped cleanup |
| `cmd/fab/dispatch_start_test.go` `TestDispatchStart_PanePreferenceNoInteractiveCommand_Integration` + `startPrivateTmuxWithPane` (~:527, :727) | Deliberate no-`-L` default-socket sites; were guarded by refuse-if-`$TMUX`-set + verified-`-S` cleanup (never kill vectors). Refusal guards now dead code under the shared scrub → removed; SAFETY comments updated; these tests are now runnable inside panes. `startPrivateTmuxWithPane`'s deliberate later `t.Setenv("TMUX", privateSocket+",1,0")` undisturbed |
| All remaining `TMUX_TMPDIR` hits (`dispatch_{reap,restart,status,wait,kill,deliver,open}_test.go`, `pane_exitcode_test.go`, `pane_send_test.go`, `panemap_test.go`, `internal/pane/pane_test.go`) | Clean: every tmux call rides `-L <fabtest-*>` via `newTmuxPane`/local closures (`-L` outranks `$TMUX`); no unscoped calls found. All additionally hardened by the tmuxSocketDir scrub |

### T005 verification evidence (R6)

1. **Fixed subtest** passes with `$TMUX` absent (baseline) — `TestPaneReady_JSON/server_null_on_the_default_socket PASS`.
2. **Fixcheck survival repro**: `go test ./cmd/fab/ -run TestPaneReady -v` run inside a window of a disposable `tmux -L fixcheck` server → ALL TestPaneReady tests PASS (`EXIT=0`) and the fixcheck server SURVIVED (`list-sessions` answered, session `guard` present). Pre-fix this exact run killed its own server. fabKit1 untouched throughout. Fixcheck then killed explicitly scoped; its stale socket file removed (tmux never unlinks — 0j0t).
3. **Leak check**: before/after diff of `/tmp/tmux-$UID/` shows zero new `fabtest-*` sockets and zero live `fabtest` servers. (Concurrent noise: ~60 new `rk-test-*` sockets from run-kit's own test suite running on this box — not fab-kit's.)
4. **Full packages**: `env -u TMUX -u TMUX_PANE go test ./cmd/fab/` → exit 0, no failures; `go test ./internal/pane/` → ok. `gofmt`/`go build ./...`/`go vet` clean.

### Pre-existing flake found during verification (out of scope — backlog [zshf])

One earlier full-package run failed `TestPaneReady_ReadyReport`/`TestPaneReady_JSON/ready` with zsh's `zsh-newuser-install` menu drawn over the pane prompt. Attributed by stash-and-rerun on UNMODIFIED code: same failure — pre-existing, not caused by this change. Root cause: cmd/fab's `TestMain` points `HOME` at an empty temp dir (config isolation, #553), so default-shell panes start zsh with no `~/.zshrc` on zsh-default machines; the menu self-aborts when the readiness probe's keystroke hits it, making the failure timing-dependent. CI (bash default shell) is unaffected. Recorded as backlog `[zshf]`; not fixed here (unrelated root cause, scope discipline).

### Deliberate redundancy note (A-009 context)

The `$TMUX` scrub appears both inside `tmuxSocketDir` (the class guard — future tests can't reintroduce the vector) and inline in the fixed subtest (the incident site — keeps the WHY visible where the bare default-socket server is started). Two `t.Setenv` lines of overlap, each serving a distinct documented purpose; not consolidation-worthy.

## Deletion Candidates

None — the only code this change made redundant (the refuse-if-`$TMUX`-set guards at `dispatch_start_test.go:529-532` and `:732-735`) was removed by the change itself; nothing else became unused.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | "Outside tmux" verification steps are satisfied by `env -u TMUX -u TMUX_PANE` from this session — this orchestrator itself lives in a fabKit1 pane, and an env-scrubbed process tree is equivalent for tmux's socket resolution | tmux resolves from the process env; a child with `TMUX` unset is indistinguishable from an outside-tmux process for every code path this change touches | S:70 R:85 A:85 D:75 |
| 2 | Confident | The R4 shared scrub is implemented only when T001 confirms a clean mechanism; a gate failure degrades to the local-only fix without failing the pipeline | Intake §3 pre-agreed both outcomes; degradation is the documented fallback | S:75 R:80 A:80 D:70 |
| 3 | Certain | No verification step runs the unfixed test with an inherited live `$TMUX`; the destructive path is exercised only inside the disposable fixcheck server | The pre-fix failure mode kills this session's own server; the intake pins the recipe | S:95 R:90 A:90 D:95 |

3 assumptions (1 certain, 2 confident, 0 tentative).
