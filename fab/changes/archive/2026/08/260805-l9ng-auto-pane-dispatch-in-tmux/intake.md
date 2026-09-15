# Intake: Auto Pane Dispatch Inside tmux

**Change**: 260805-l9ng-auto-pane-dispatch-in-tmux
**Created**: 2026-08-05

## Origin

> Can we make it so that if we detect that we are in run-kit (if we are in a TMUX session), we default to panes. If not, we default to headless or native.

Conversational origin (`/fab-discuss` session, 2026-08-05) — a follow-up to Phase 2 (`260805-zxe0-interactive-pane-stage-dispatch`, PR #524), which shipped the pane adapter with headless as the unconditional CLI-adapter default and `--pane` as per-invocation opt-in. The user wants the default to be environment-sensitive: inside a tmux session (the run-kit context), CLI-dispatched stages should land in watchable panes without being asked; outside tmux, headless stays the default. Native Agent-tool dispatch is unaffected either way.

**Ordering**: activate only after zxe0 (PR #524) merges — this change edits the exact code and doc surfaces zxe0 ships.

## Why

1. **The pain point**: after zxe0, watchability is opt-in per invocation — inside a run-kit tmux session (where a pane is exactly what the user wants and can see) a plain "use codex for apply" still yields a detached headless process, and the user must remember to say "watchable" to get the pane. The environment already carries the answer (`$TMUX` set = the user is attached somewhere a pane would be visible), so asking is ceremony.

2. **The consequence of not fixing it**: the pane adapter under-delivers in its primary habitat. Run-kit users get headless black boxes by default and either forget the flag or wrap it in aliases; the "richness" zxe0 built stays behind a flag nobody remembers.

3. **Why this approach**: an `auto` default resolved at `fab dispatch start` (Go) is a single enforcement point covering both skill-driven and manual invocations, needs zero dispatch-seam changes in skills (docs only), and keeps both explicit escapes (`--pane` to force, new `--headless` to opt out for unattended runs living inside tmux tabs). `$TMUX` is the correct signal for *defaulting* — it means "a pane opened without `-L` lands on the server the user is attached to" — and is distinct from zxe0's `ServerReachable` probe, which stays the *validation* step once pane mode is chosen.

## What Changes

### 1. `fab dispatch start` mode selection becomes `auto`

New selection logic in `runDispatchStart` (or its prologue), replacing the current "headless unless `--pane`":

1. **Explicit `--pane`** → pane mode (unchanged semantics: `ServerReachable` probe, hard error and nothing written when unreachable).
2. **Explicit `--headless`** (new flag) → headless mode.
3. **Explicit `--timeout`** (`Flags().Changed`) → headless mode (a timeout is only meaningful there). `--pane` + `--timeout` remains the existing usage error; `--headless` + `--timeout` composes.
4. **No explicit mode signal** → **auto**: `$TMUX` set in the environment → pane mode; unset → headless.
   - Auto-selected pane mode targets the **current** server (tmux commands inherit `$TMUX`; no `-L` is passed unless `--server` was given). `--server` continues to force the named socket and — carried over from zxe0 — implies pane-capability but is NOT itself a mode selector; under auto with `$TMUX` unset, `--server <name>` + no mode flag selects pane mode against that socket (the flag's only purpose is pane targeting; treat it as a pane signal — see Assumptions).
   - **Auto-pane failure is a soft fallback, not an error**: if auto selects pane but `ServerReachable` fails (e.g. `$TMUX` is stale — inherited from a killed server), fall back to headless with a one-line stderr notice (`pane auto-selection: tmux unreachable, falling back to headless`). Explicit `--pane` keeps the hard error.
5. `--pane` + `--headless` together → usage error (`Flags().Changed`-based, matching the repo's mutual-exclusion precedent).

The `dispatched %s/%s (%s)` output line already names the mode's identity (pid/pgid vs pane); add the selection source when auto fired (e.g. `(pane %%3, auto: tmux)`) so a surprising mode is explainable from output — mirroring the compliance-visibility principle.

### 2. Docs: the CLI-adapter mode default is now environment-sensitive

- `src/kit/skills/_cli-fab.md` § fab dispatch — the mode-selection ladder above (explicit > timeout > auto), the `--headless` flag, the soft-fallback rule, the `$TMUX`-inheritance note.
- `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch — the "headless is the default" statement becomes "mode defaults to auto (pane inside tmux, headless outside); pass `--headless` for unattended runs, `--pane` to force". The orchestrator's polling loop is mode-independent (result-file detection) and needs no change.
- `docs/specs/harness-adapters.md` — the pane-adapter section's mode-selection paragraph.
- SPEC mirrors of the touched skills (`SPEC-_cli-fab.md`, `SPEC-_preamble.md`) and aggregate specs only where they restate "headless default" (grep-verify; zxe0 just rewrote these surfaces, so the sweep should be small).

### 3. Tests

- Unit: selection ladder table (each explicit flag, timeout-implies-headless, auto with/without `$TMUX` via `t.Setenv`, `--pane --headless` usage error, `--server`-implies-pane under auto).
- Soft-fallback: `$TMUX` set to a dead socket → headless fallback with the stderr notice, dispatch record is headless-shaped.
- Integration (real tmux, following zxe0's ephemeral-socket pattern): auto inside tmux opens a window on the current server.

### Non-goals

- No config knob (`dispatch.default_mode`) in v1 — the env default + two explicit flags cover it; a knob is additive later if real friction appears.
- No change to native Agent-tool dispatch, the dispatch seam's `dispatch=` branch, adapter selection (which provider carries a `dispatch_command`), the five-state/three-state machines, or the result-file contract.
- No run-kit-specific detection beyond `$TMUX` (rk presence adds nothing: the pane lands in tmux either way).

## Affected Memory

- `runtime/dispatch.md`: (modify) mode-selection ladder, `--headless` flag, auto default, soft fallback, `$TMUX` semantics vs the `ServerReachable` probe
- `_shared/context-loading.md`: (modify) CLI-Adapter Dispatch mirror — the default-mode sentence
- `pipeline/execution-skills.md`: (modify) only if it restates the headless default (grep-verify)

## Impact

- `src/go/fab/`: `cmd/fab/dispatch_start.go` (+ flag), `internal/dispatch` only if selection helpers land there; `_test.go` files
- `src/kit/skills/`: `_cli-fab.md`, `_preamble.md`
- `docs/specs/`: `harness-adapters.md`, SPEC mirrors as swept
- **Depends on**: zxe0 / PR #524 merged (edits surfaces zxe0 ships). No migration (behavioral default change on a transient-state command; no user-data restructuring). Orthogonal to j3cm (provider config) — either order works between the two, but both after #524.

## Open Questions

*(none — the direction and both escape hatches were stated by the user or follow directly; residuals are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Detection = `$TMUX` env presence at `fab dispatch start` time | User-stated ("if we are in a TMUX session"); it is also exactly the condition under which an un-targeted pane is visible to the user | S:90 R:80 A:90 D:90 |
| 2 | Certain | Selection lives in Go at `fab dispatch start` (single enforcement point); skills get doc updates only | Covers manual + skill-driven invocations identically; the dispatch seam is untouched | S:80 R:80 A:90 D:85 |
| 3 | Confident | New `--headless` flag as the explicit opt-out; `--pane`+`--headless` is a usage error | Unattended runs inside tmux tabs need an escape; mutual-exclusion matches repo precedent | S:70 R:85 A:85 D:80 |
| 4 | Confident | Explicit `--timeout` selects headless under auto (rather than erroring inside tmux); `--pane`+`--timeout` stays an error | A timeout only has meaning headless; erroring would break existing scripted invocations that never mention panes | S:65 R:80 A:85 D:75 |
| 5 | Confident | Auto-pane soft-falls-back to headless (stderr notice) when tmux is unreachable; explicit `--pane` keeps the hard error | A stale `$TMUX` must not break unattended dispatches; explicitness keeps its guarantee <!-- assumed: soft fallback for auto vs hard error for explicit — asymmetry mirrors auto-defaults elsewhere --> | S:60 R:80 A:80 D:70 |
| 6 | Confident | `--server <name>` under auto is a pane signal (selects pane mode against that socket even with `$TMUX` unset) | The flag exists solely for pane targeting; naming a socket while meaning headless is incoherent | S:55 R:80 A:80 D:70 |
| 7 | Confident | No config knob in v1 | Env default + two explicit flags cover the matrix; a knob is additive later | S:65 R:90 A:80 D:75 |
| 8 | Certain | Activate only after PR #524 merges | Edits the exact code/doc surfaces zxe0 ships; sequencing stated in Origin | S:85 R:80 A:90 D:90 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
