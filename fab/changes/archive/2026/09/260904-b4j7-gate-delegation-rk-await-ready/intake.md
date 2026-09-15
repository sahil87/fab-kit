# Intake: Gate Delegation to rk mux await --ready

**Change**: 260904-b4j7-gate-delegation-rk-await-ready
**Created**: 2026-09-04

## Origin

One-shot `/fab-new` invocation (agent-messaging execution plan, Part D — the fab-kit part). User's raw input:

> Agent-messaging Part D: gate delegation (gap 5). Delegate the dispatch gates classification half to rk mux await --ready in fab-kit internal dispatch/pane readiness gate. run-kit shipped Part B (PR #835, released in v3.19.9, brew-installable) which added a sentinel echo classifier to rk mux await --ready: state-present -> ready %N (state); sentinel echo at input box, no existing state -> ready %N (echo); no echo on a settled non-blank screen -> parked %N (exit 0) with a screen snippet on stderr; booting never returns, only ready/parked/gone/timeout(running). Update fab-kits pane readiness gate (see docs/specs/agent-messaging.md commit c36cfbd8 on run-kit main, section Spawn and trust walls -- the readiness standard, and section Gaps from current state item 5) so the mechanical classification (state-present / sentinel echo / parked) delegates to rk mux await --ready (command -v rk gated, fail-open to the existing raw-tmux fallback when rk is absent or too old to have --ready sentinel support) -- fab keeps only the judgment rounds (deciding what a parked wall wants) and the rk-less raw-tmux fallback path. See docs/specs/harness-adapters.md and _cli-fab.md pane readiness gate documentation for the current fab-side implementation to replace. This fab-kit part intrinsically includes: the standards audit (shll standards) and any help-dump/skill-doc updates the delegation touches.

Upstream design of record: run-kit `docs/specs/agent-messaging.md` @ c36cfbd8 (verified locally — § Spawn and trust walls, § Surface and naming item "fab consumes, never reimplements", § Gaps item 5, § Execution plan row D "gate delegation | fab-kit | Gap 5 … keeps judgment rounds and the rk-less fallback | B *(released)*"). Dependency satisfied: run-kit Part B merged as PR #835 (b32e1aaf) and released in v3.19.9; the locally installed brew `rk` is v3.19.9 and its `rk mux await --help` already documents the sentinel `--ready` contract (`ready %N (state)` / `ready %N (echo)` / `parked %N` + stderr snippet).

Key findings from intake-time code reading (this session):

1. **fab's classifier lives in Go**, `src/go/fab/internal/pane/gate.go` (`Gate.Probe`), bound by three consumers: `fab pane ready`, `fab dispatch ready`, and `Gate.Deliver`'s readiness precondition (both `fab pane deliver` and `fab dispatch deliver`).
2. **rk's `AwaitReady` (run-kit `app/backend/internal/inject/ready.go`) has NO shell-foreground takeover precondition** — by design ("terminals vs agents are one standard"): a cooked-mode shell's echo of the `#rk-ready-probe` sentinel classifies `ready (echo)`. fab's gate added exactly that precondition as the 57mp false-ready fix (cooked-tty echo before TUI takeover). The delegation must therefore keep fab's shell-foreground check as a pre-rk guard and delegate only the post-takeover classification.
3. **Precedent for Go-side rk delegation exists**: `fab pane map` shells out to `rk mux panes --json` (`src/go/fab/cmd/fab/pane_map.go`, `rkPanesRunner`).
4. **Capability discrimination matters**: `--ready` existed before Part B with the weaker capture-settle classifier (false-fire hazard: a settled trust dialog reported `ready (settled)`). "rk present" is not enough; fab must detect *sentinel-capable* `--ready`. Memory lesson: verify the binary, not the version string — a help-text probe (`rk mux await --help` mentions `parked`) discriminates correctly; a bare version compare can lie (bottle-predates-source).
5. fab-kit has its own help-dump test (`src/go/fab/cmd/fab/help_dump_test.go`) — help-text edits must update it.

## Why

1. **Problem**: fab-kit maintains its own pane readiness classifier (sentinel typing, echo counting, wrap-tolerant squeeze, stability captures) in parallel with run-kit's now-shipped equivalent. The agent-messaging spec fixes the target ownership: classification is mechanical and rk-owned; "fab consumes, never reimplements" — fab's own pane copies are for the rk-less arm only. Two live implementations of the same probe drift (different sentinels, different settle heuristics, different wrap tolerance) and every future hardening (e.g. Part A's pane-mode guard, named-buffer bracketed paste, novelty counting) lands only on the rk side.
2. **What fab gains concretely**: rk's classifier is *better* on rk-equipped machines — (a) a **state-present fast path** (`ready %N (state)`): an instrumented agent whose hooks fired is classified touch-nothing, where fab today always types a sentinel into it; (b) delivery through rk rides the single inject engine (sanitize → named-buffer bracketed paste → novelty echo probe), hardened by Part A's `#{pane_in_mode}` guard; (c) `booting` never returns from rk — it blocks through boot churn, so one call replaces fab's re-probe polling for the common case.
3. **If we don't**: fab's copy stays the weaker sibling (no state fast path, no bracketed paste), the spec's Gap 5 stays open, and the two classifiers diverge further with each run-kit messaging change (Parts A–C already shipped or in flight).
4. **Why this approach** (in-binary delegation with fail-open fallback, not a skill-side rewrite): the gate's callers are Go verbs (`fab dispatch ready`, `fab pane ready`, the deliver precondition) with a frozen report contract the orchestrator skills branch on; delegating inside `Gate.Probe` upgrades all three consumers at once while every documented verb contract, exit code, and skill-side wiring (`_preamble.md` § The pane readiness gate — budget, judgment rounds, escalation) stays intact. The judgment half was never in the binary and does not move.

## What Changes

### 1. `internal/pane`: rk-delegated classification inside `Gate.Probe`

`Gate.Probe` (src/go/fab/internal/pane/gate.go) becomes a two-arm classifier:

1. **Takeover precondition stays fab-side, ahead of both arms** (57mp guard, unchanged): read `#{pane_current_command}`; while a shell owns the pane, report `booting` with nothing typed and **do not invoke rk**. Rationale: rk's `AwaitReady` deliberately classifies a cooked-shell echo as `ready (echo)` (terminals-are-one-standard); for a dispatch pane that signal is exactly the 57mp false-ready. The precondition is a read-only tmux query fab already makes; it is not part of the "state-present / sentinel echo / parked" classification the spec delegates.
2. **rk arm** (preferred): when rk is available and sentinel-capable (§3), run `rk mux await --ready <pane> [-L <server>] --timeout <bounded>` and map its report words onto fab's frozen `Readiness` contract:

   | rk outcome | fab report |
   |------------|------------|
   | `ready %N (state)` / `ready %N (echo)` (exit 0) | `ready` |
   | `parked %N` (exit 0, snippet on stderr) | `parked` + snippet (rk's stderr snippet, re-emitted per fab's existing `--- last 20 lines ---` convention) |
   | `running` (timeout, exit 0) | `booting` + fab's own capture snippet (rk's `running` line carries none) |
   | `gone` (exit 1) | fab's existing dead-pane error path (same refusal the identity check produces — not a classification) |
   | any other exit/parse outcome | **fail-open**: fall through to the raw-tmux arm for this probe (stderr warning) |

   The rk timeout is a bounded internal constant (mechanics, not policy — the `settleDelay`/`stabilityDelay` precedent; no flag, no config field). `booting` remains a reachable report (precondition + timeout mapping), so the skill-side gate loop and its 5-consecutive-booting allowance stay valid unchanged.
3. **raw-tmux arm** (fallback, byte-identical to today's classifier): sentinel type → echo count → stability captures → `C-u`. Reached when rk is absent (`exec.LookPath`), present but not sentinel-capable, or an rk invocation fails unexpectedly. Fail-open and warn-once semantics; never a hard error solely because rk is missing (mirrors the `_preamble.md` rk fail-silent rule).

`Gate.Deliver`'s readiness precondition inherits the delegation automatically (it calls `Probe`). **The delivery choreography itself does NOT delegate to `rk mux send`** — that is not Gap 5 and is out of scope (Non-Goals).

### 2. Capability probe: sentinel-capable `--ready` detection

A helper (in `internal/pane`, near the delegation) answers "is this rk's `--ready` the Part B sentinel classifier?" by probing the binary, not the version string: run `rk mux await --help` and require the `--ready` flag help to mention the sentinel contract (the token `parked` — absent from the pre-Part-B capture-settle help, present in v3.19.9's: "…else a sentinel echo probe: echo = ready, no echo = parked"). Cache the answer per process (one probe per fab invocation at most). `command -v rk` equivalent is `exec.LookPath("rk")` (the `rkPanesRunner` precedent).

### 3. Docs: specs, `_cli-fab.md`, `_preamble.md`

- **`docs/specs/harness-adapters.md`** (§3, "The readiness gate stands between `open` and `deliver`…" bullet, and the § fab dispatch ready contract language): amend the classifier MUSTs from "it types a sentinel…" to the two-arm shape — mechanical classification delegates to `rk mux await --ready` when a sentinel-capable rk is on PATH (fail-open), the raw-tmux classifier is the rk-less arm, the takeover precondition and the no-dialog-table / answers-nothing / non-`ready`-carries-snippet MUSTs bind both arms. Same explicit-amendment route the spec header already uses.
- **`src/kit/skills/_cli-fab.md`** § fab pane ready + § fab dispatch ready (and the § fab dispatch overview line naming the gate): document the delegation, the capability probe, the fail-open rule, and the unchanged report/exit contract. Constitution: CLI behavior change ⇒ `_cli-fab.md` + tests.
- **`src/kit/skills/_preamble.md`** § The pane readiness gate: wiring is unchanged (open → ready loop → deliver, budgets, judgment rounds, escalation); at most a one-line note that classification is rk-delegated when available. Owner-or-pointer: mechanics stay owned by `_cli-fab.md`/spec, so keep this minimal.
- Help text for `fab pane ready` / `fab dispatch ready` (if reworded) ⇒ regenerate `help_dump_test.go` fixtures.

### 4. Standards audit + skill docs (intrinsic scope)

- Run `shll standards`, re-fetch and check the standards governing CLI surface/help output against the changed help text (constitution § Toolkit Standards; the shll contract is re-fetched live, never trusted from memory).
- Sweep skill docs that restate the gate's classifier mechanics (`_cli-agents.md` § Pre-Send Validation pointer, `fab-operator.md` if it names the classifier) — sibling-sweep the phrase class "types a sentinel" / "echo-and-stability probe" repo-wide before finishing apply.

### 5. Tests

- `gate_test.go`: the raw-tmux arm's tests stay (fallback must remain byte-identical); new table tests for the rk-arm mapping (report-word parse → Readiness, stderr snippet passthrough, timeout→booting, gone→error, malformed output→fallback) against a scripted runner fake (the `PaneIO` testability pattern); capability-probe tests (help text with/without `parked`, missing rk).
- Help-dump fixtures if help text changes.

## Affected Memory

- `runtime/pane-commands.md`: (modify) `fab pane ready` classifier is now two-arm — rk-delegated (sentinel-capable `rk mux await --ready`, fail-open) over the raw-tmux fallback; report/exit contract unchanged
- `runtime/dispatch.md`: (modify) `fab dispatch ready`'s gate binding inherits the delegation; the state-present fast path and rk-arm mapping table
- `runtime/agent-primitives.md`: (modify) the readiness-gate row of the rk-gated/raw-fallback pattern now includes classification (send gate precedent already documented there)

## Impact

- **Code**: `src/go/fab/internal/pane/gate.go` (+ a small rk runner/capability helper, likely a new file in the same package), `gate_test.go`, possibly `pane_ready.go`/`dispatch_ready.go` help strings, `help_dump_test.go` fixtures.
- **Specs**: `docs/specs/harness-adapters.md` (§3 readiness-gate bullet + §fab dispatch ready language).
- **Kit skills**: `src/kit/skills/_cli-fab.md` (two § ready sections + dispatch overview), `src/kit/skills/_preamble.md` (one-line note at most), sibling sweep across `_cli-agents.md`/operator prose.
- **Memory**: three `runtime/` files above.
- **Runtime behavior**: on rk-equipped machines the gate gains the state-present touch-nothing path and rk's hardened inject engine; on rk-less machines behavior is byte-identical. No migration (no user-data restructuring). Report words, exit codes, JSON shapes, and the skill-side gate wiring are all unchanged.
- **Release**: MINOR (new behavior, no breaking contract change).

## Open Questions

- None — the run-kit spec fixes the target semantics; remaining choices are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly Gap 5: the gate's classification half only. `Gate.Deliver`'s typing choreography does NOT delegate to `rk mux send`; judgment rounds, budgets, escalation stay skill-side unchanged | Spec § Gaps item 5 and Execution-plan row D say so verbatim; user's input repeats it | S:95 R:85 A:95 D:95 |
| 2 | Confident | Delegation lives in Go inside `Gate.Probe` (`internal/pane`), so `fab pane ready`, `fab dispatch ready`, and both delivers' readiness precondition inherit it; every verb's report/exit/JSON contract is frozen | User points at "fab-kit internal dispatch/pane readiness gate … implementation to replace"; the three consumers share one classifier, and skills branch on the frozen report words — a skill-side rewrite would fork the contract | S:75 R:60 A:85 D:80 |
| 3 | Confident | The shell-foreground takeover precondition (57mp) stays fab-side, evaluated BEFORE the rk arm; rk is never invoked while a shell owns the pane | Verified in run-kit source: `inject.AwaitReady` has no takeover guard and deliberately classifies cooked-shell echo as `ready (echo)` — correct for rk's terminals-too standard, a false-ready regression for fab's dispatch gate. The precondition is not in the spec's delegated list (state-present / sentinel echo / parked) | S:60 R:55 A:90 D:85 |
| 4 | Confident | Sentinel-capability detection is a help-text probe (`rk mux await --help` output mentions `parked`), cached per process — not an `rk --version` compare | "Too old to have --ready sentinel support" must be discriminated from pre-Part-B capture-settle `--ready`; memory lesson: verify the binary, not the version string (bottle-predates-source); `rk skill` capability-probe precedent in `_preamble.md` | S:70 R:80 A:80 D:70 |
| 5 | Confident | Fail-open scope: rk absent, not sentinel-capable, or an invocation failing unexpectedly ⇒ raw-tmux arm with a stderr warning; rk's `gone` maps to fab's dead-pane error (NOT fallback); `parked`/`ready`/`running` are honored, never re-classified by the fallback arm | User: "fail-open to the existing raw-tmux fallback when rk is absent or too old"; a classified outcome is an answer, not a failure — re-running the raw arm after rk answered would double-type sentinels | S:80 R:75 A:80 D:75 |
| 6 | Confident | fab's `ready` verbs keep their probe posture: the rk call rides a bounded internal timeout constant (no flag, no config — the gate-timings precedent) with rk's `running` mapped to `booting`, so the skill-side loop (boot re-probes, 5-consecutive-booting allowance) needs no rewiring | rk's await blocks through boot churn ("booting never returns"), fab's verb contract documents `booting` as a reachable report; mapping timeout→booting preserves every documented contract while letting one rk call absorb most boot churn | S:55 R:75 A:75 D:65 |
| 7 | Tentative | The bounded rk timeout starts at ~20s per probe call (one rk await absorbs normal boot churn; 5 booting reports ≈ 100s before parked-treatment) | Value is mechanics, judged at apply against the existing settle constants and the gate wiring's re-probe cadence; easily tuned later | S:40 R:90 A:60 D:55 |
| 8 | Confident | fab's report words stay exactly `ready`/`booting`/`parked` — rk's `(state)`/`(echo)` discriminator is not surfaced as a new contract token (at most an informational stderr/suffix detail decided at apply) | Report-word contract is frozen and skills branch on it; adding tokens is a contract change Gap 5 doesn't ask for | S:65 R:70 A:80 D:70 |
| 9 | Certain | Dependency gate satisfied: Part B is released (PR #835 → v3.19.9, brew-installable; installed rk verified v3.19.9 with the sentinel `--ready` help) | Execution-plan row D gates on "B *(released)*"; verified live this session | S:95 R:90 A:95 D:95 |
| 10 | Certain | Intrinsic scope includes the shll standards audit (re-fetch live) and help-dump/skill-doc updates the delegation touches | User's input states it; constitution § Toolkit Standards + the CLI ⇒ `_cli-fab.md`+tests rule | S:90 R:85 A:90 D:90 |

10 assumptions (3 certain, 6 confident, 1 tentative, 0 unresolved).
