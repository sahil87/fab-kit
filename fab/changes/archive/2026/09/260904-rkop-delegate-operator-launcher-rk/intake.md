# Intake: Delegate Operator Launcher to rk

**Change**: 260904-rkop-delegate-operator-launcher-rk
**Created**: 2026-09-05

## Origin

> Backlog `[rkop]` (2026-09-03): Delegate the bare 'fab operator' LAUNCHER to 'rk operator' when rk is present (capability-probed, fail-open to today's launcher when absent — fab stays standalone-installable). run-kit is growing 'rk operator' (change 260903 rk-operator-launcher, stacked on run-kit PR #806): server-wide singleton probed by @rk_win_role=operator first / window name second, role option stamped at create, launcher via 'fab agent operator --print', kickoff '/fab-operator' TYPED via the inject.DeliverWhenReady composite (positional kickoff is claude-only — kimi parses it as a subcommand and exits; same class as run-kit PR #801's tutorial break). fab side when it lands in a RELEASED rk: bare 'fab operator' probes 'command -v rk' + capability (rk operator exists), execs 'rk operator' passing --workers through; keeps the current launcher as the rk-absent fallback; help text points at rk operator; the operator SUBCOMMAND family (tick-start/autopilot/enroll/state/note/watch/branch-map) stays in fab — choreography, not substrate (cli-layering.md delegation rules 1-2; gate on released-not-merged per the execution-plan convention).

One-shot invocation via `/fab-new rkop`. The release gate was **verified live at intake**: the installed `rk` is v3.19.11 and `rk operator --help` resolves (run-kit change 260903-a8e4-rk-operator-launcher shipped as run-kit PR #812 and is in the released binary), so the fab side is unblocked.

## Why

1. **The pain point**: two launchers now exist for the same operator window. run-kit's `rk operator` is the richer one — it probes the singleton by `@rk_win_role=operator` (role option, name-match as fallback), stamps the role option at create so the dashboard pins the window as the server's orchestrator, and delivers the `/fab-operator` kickoff by **typing** it via `inject.DeliverWhenReady` (provider-agnostic — a positional kickoff argument is claude-only; kimi parses `/fab-operator` as a subcommand and exits). fab's `runOperator` still passes the kickoff positionally, matches the singleton by window name only, and stamps no role option.
2. **The consequence of not fixing it**: the two launchers drift. A `fab operator` launch produces an unmarked window the dashboard cannot pin, breaks the kickoff on non-claude session providers, and can violate the singleton against an rk-launched window that was renamed (rk's role-option probe would catch it; fab's name match does not). Every improvement run-kit makes to its launcher (readiness gating, verified delivery) is invisible to `fab operator` users.
3. **Why this approach**: cli-layering delegation rule 1 — the launch mechanics (window creation, role marking, readiness-gated typed kickoff) are tmux **substrate**, which rk owns; fab owns the choreography (agent-profile resolution, which rk consumes back via `fab agent operator --print` — the delegation is already bidirectional). Rule 2 makes it safe: capability-probed, fail-open to today's launcher, so fab stays standalone-installable. The alternative — porting the role-option probe and typed-kickoff composite into fab — would reimplement rk's layer (rule 1 violation) and duplicate the `inject` machinery.

## What Changes

### 1. `runOperator` delegates to `rk operator` when capable

In `src/go/fab/cmd/fab/operator.go`, `runOperator` gains a delegation branch **before** the existing launcher logic:

- **Capability probe** (side-effect-free, two parts): `exec.LookPath("rk")` succeeds AND the installed rk has the `operator` subcommand (probe via a no-side-effect invocation, e.g. `rk operator --help` exiting 0 — an rk predating the subcommand fails this and falls through). The probe itself must not create windows or touch tmux state.
- **Probe passes** → run `rk operator`, passing `--workers <v>` through when the flag was set (see §2). Inherit stdio and propagate the exit code (exec-style handover — the user sees rk's output verbatim). From this point rk owns the whole launch: its own `$TMUX` / fab-on-PATH preconditions, the role-option singleton probe, window creation + `@rk_win_role=operator` stamping, `fab agent operator --print` launcher resolution, and the typed kickoff.
- **Probe fails** (rk absent from PATH, or present without the `operator` subcommand) → **silent** fall-through to today's launcher, unchanged byte-for-byte (the fail-open path per delegation rule 2: absence degrades, never errors, no warning).
- **A runtime failure of the delegated `rk operator` is surfaced, not fallen back on** — after the probe passes, `rk operator` may already have created or switched windows; retrying with fab's launcher would risk a double launch or a duplicate window. Failure-after-probe is a real error (matching rk's own exit codes 1/2/3), distinct from absence.

The subcommand family (`tick-start`/`time`/`state`/`enroll`/`update`/`remove`/`note`/`watch`/`autopilot`/`branch-map`) is untouched — choreography stays in fab; only the bare-`fab operator` `RunE` delegates.

### 2. `--workers` pass-through

Today fab prefixes the spawn command with `FAB_AGENT_WORKERS='<v>'` verbatim (no validation). On the delegated path the flag becomes `rk operator --workers <v>`, and **rk's validation applies**: the value is restricted to letters, digits, `_` and `-`, with an invalid value a usage error (exit 2) before anything runs. This is a deliberate, documented behavior delta on the delegated path; the fallback path keeps fab's verbatim env-prefix behavior.

### 3. Help text points at rk operator

The cobra `Short`/`Long` for `fab operator` gains a line stating that when run-kit is installed the launch is delegated to `rk operator` (role-marked window, provider-agnostic typed kickoff), with today's launcher as the rk-absent fallback.

### 4. Docs + tests (CLI ⇒ docs + tests rule)

- `src/kit/skills/_cli-fab.md` § fab operator: document the delegation branch first (probe, pass-through, failure semantics), then the existing launcher text as the fallback path; note the `--workers` validation delta.
- `src/kit/skills/fab-operator.md`: the launcher sentence (~line 25 "Start via `fab operator`…") and the §9 Key Properties launcher rows note the delegation (pointer to `_cli-fab.md` § fab operator per owner-or-pointer — no restating).
- Tests in `src/go/fab/cmd/fab/operator_test.go`: probe-positive delegates with correct argv (incl. `--workers` pass-through), probe-negative falls through silently, delegated runtime failure propagates. Delegation seams (lookPath / runner) injectable package-level vars, matching the `pane map` delegation-test pattern (`rkPanesRunner`).

## Affected Memory

- `runtime/operator.md`: (modify) the launcher documentation — bare `fab operator` now delegates to `rk operator` when capable (capability-probed, fail-open on absence, surface-on-runtime-failure); today's launcher becomes the documented fallback path; Design Decision recording why absence ≠ failure in the fallback rule.

## Impact

- `src/go/fab/cmd/fab/operator.go` — delegation branch in `runOperator`, help text; `operator_test.go` — new delegation tests. Existing launcher code (`operatorSpawnCommand`, `findWindowExact`, cwd resolution) unchanged — it IS the fallback.
- `src/kit/skills/_cli-fab.md` § fab operator (canonical launcher doc), `src/kit/skills/fab-operator.md` (launcher pointer rows) — sweep class per code-quality.md § Sibling Sweeps.
- `docs/memory/runtime/operator.md` at hydrate.
- No config schema, no `.status.yaml`, no migration (no user-data restructuring).
- Release: MINOR (new delegation behavior on an existing command).
- Cross-repo dependency already satisfied: `rk operator` shipped in released rk v3.19.11 (verified via installed binary at intake).

## Open Questions

*(none — the backlog entry pins the design; remaining choices are graded assumptions below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Release gate satisfied — build against `rk operator` now | Verified live: installed rk v3.19.11 answers `rk operator --help`; run-kit PR #812 merged and released | S:90 R:90 A:95 D:95 |
| 2 | Certain | Subcommand family stays in fab; only bare `RunE` delegates | Explicit in the backlog entry (choreography vs substrate, delegation rules 1–2) | S:95 R:90 A:95 D:95 |
| 3 | Certain | `--workers` passes through as `rk operator --workers <v>`; rk's charset validation applies on the delegated path | rk's flag exists for exactly this hand-off; delta documented in `_cli-fab.md` | S:85 R:85 A:90 D:85 |
| 4 | Confident | Capability probe = `exec.LookPath("rk")` + side-effect-free subcommand check (`rk operator --help` exit 0), NOT attempt-is-the-probe | pane map's attempt-is-probe suits an idempotent read; the launcher has side effects (window creation), so the probe must be side-effect-free — falling back after a partial launch risks duplicates | S:75 R:80 A:80 D:70 |
| 5 | Confident | Runtime failure after a passing probe is surfaced, never fallen back on | Delegation rule 2 says *absence* degrades; a failed `rk operator` may have mutated tmux state — double-launch risk. rk's own exit codes (1 precondition / 2 usage / 3 subprocess) are meaningful to the user | S:75 R:75 A:85 D:75 |
| 6 | Confident | Hand-over runs `rk operator` with inherited stdio + exit-code propagation (subprocess; true `syscall.Exec` optional, decided at plan) | Backlog says "execs"; behaviorally equivalent, and an injectable subprocess seam matches the repo's delegation-test pattern | S:70 R:85 A:75 D:60 |
| 7 | Confident | fab's own `$TMUX` precheck is skipped on the delegated path — rk owns its preconditions | Double-owning the check duplicates rk's error surface; rk hard-errors identically (exit 1). Fallback path keeps fab's check unchanged | S:70 R:80 A:80 D:70 |
| 8 | Confident | Docs sweep class = `_cli-fab.md` § fab operator (owner) + `fab-operator.md` launcher rows (pointers) + `runtime/operator.md` (hydrate) | Grep-verified: these are the launcher-behavior claim sites; owner-or-pointer keeps `fab-operator.md` as pointers | S:70 R:85 A:80 D:75 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
