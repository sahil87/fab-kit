# Intake: Provider-Generic Pane Verbs — Extract open/ready/deliver Primitives from Dispatch

**Change**: 260810-1lah-provider-generic-pane-verbs
**Created**: 2026-08-10

## Origin

Conversational (`/fab-discuss` session 2026-08-09/10). The user asked:

> Also take a look at the subcommands available in fab to be able to communicate with "pane" mode subagents — should we make these commands more generic / composable so you are able to run tests like these more easily?

The question was grounded by a live kimi probe run in the same session, which had to be **hand-rolled with raw tmux** (`tmux split-window` → `sleep` → `capture-pane` → `send-keys`) because every piece of the needed choreography is locked behind the wrong address. The user accepted the layering recommendation and sequenced this change **after** `260810-ki9v-kimi-pane-enablement` merges (it relocates the gate code that change edits).

## Why

1. **Pain point** (all three observed live, same session):
   - `fab dispatch open/ready/deliver` are **change+stage-addressed**: without an active change there is no way to spawn a provider's TUI, classify its readiness, or perform a verified delivery — even though those three capabilities (the readiness classifier, the wall-answering gate, the wrap-tolerant echo-verify) are exactly what an ad-hoc provider probe, a warm-up, or an operator interaction needs.
   - `fab pane send`'s idle validation requires run-kit's `@rk_agent_state` pane option, which only rk-managed claude panes set — against any foreign-agent pane (kimi, agy, codex) it **always refuses** and the caller must always pass `--force`, i.e. the validation never validates in precisely the multi-provider scenarios where validation matters.
   - The valuable verification logic (`countWrapped`/`squeeze`, the `ready`/`booting`/`parked` classifier, `ProbeReadiness`) lives inside `internal/dispatch/gate.go`, reachable only through a dispatch record.
2. **Consequence of not fixing**: every provider probe (the rpsr rule mandates one before shipping any `interactive_command`) is re-hand-rolled with raw tmux; the operator skill has no sanctioned primitives for driving foreign-agent panes; echo-verify logic risks growing a second, divergent copy.
3. **Why layering over ad-hoc flags**: giving dispatch verbs a `--no-record`/`--pane-id` bypass would muddy the dispatch state machine and its five-state contract. Extracting pane-addressed primitives keeps dispatch's contract byte-identical (thin bindings over the primitives plus record bookkeeping) while making the primitives independently usable. Accepted tradeoff (discussed explicitly): the `_preamble` "pipeline never sends keys to a worker" discipline is behavioral, not mechanical — generic verbs make bypass easier, and that is acceptable because probes and operator work already need the escape hatch.

## What Changes

### 1. New pane-addressed, provider-generic verbs (`fab pane` family)

- **`fab pane open --provider <name> [--role <role>] [-c <dir>]`** — resolve the provider's `interactive_command` (+ role/profile fills via the standard precedence; `--role` optional, default role's fill otherwise), spawn the pane, print the pane id (and socket when non-default). **No dispatch record, no `.fab-dispatch/` state.** Plain split in the current window by default — the worker-column carving/placement logic stays dispatch-owned (placement is pipeline policy, not a primitive). Missing `interactive_command` is a hard error naming the provider (mirroring `fab dispatch open`'s explicit-pane posture: no descent).
- **`fab pane ready <pane>`** — the existing gate classifier addressed by pane id: reports `ready` / `booting` / `parked` with the capture snippet. Same sentinel-probe mechanics (type sentinel, C-u clear) relocated from `gate.go`; the sentinel probe types into the target pane, so the command documents that it is a *probe with side effects on the input buffer* (cleared before return) and is not run against panes the caller doesn't own.
- **`fab pane deliver <pane> (--prompt-file <path> | --text <string>)`** — the verified typed-delivery choreography addressed by pane id: C-u → type → wrap-tolerant echo-verify → Enter → confirm, with the same bounded retry as dispatch's deliver. `--prompt-file` types the pointer line (dispatch parity); `--text` types the literal text (probe/operator use).

### 2. `fab dispatch open/ready/deliver` become thin bindings

Identical external contract (flags, output forms, exit codes, `opened …`/`delivered …` lines, readiness reports carrying pane+socket+snippet): they resolve change+stage → call the pane-layer primitives → perform record bookkeeping (`.fab-dispatch/{id}/` state, result-file clearing on re-delivery, refuse-if-running). This is a relocation refactor of `gate.go`/`pane_mode.go` internals into `internal/pane` (or a new `internal/pane`-adjacent package if import cycles force it), not a behavior change; existing dispatch tests must pass unmodified except for import/package moves.

### 3. Fix `fab pane send`'s foreign-pane validation posture

Unknown `@rk_agent_state` (option absent/unparseable) stops being a refusal: proceed with a warning (`agent state unknown — sending anyway`), keeping the refusal only for a *parseable* non-idle state. `--force` retains its skip-everything meaning. This ends the always-`--force` ritual against foreign-agent panes while preserving the guard where the convention actually reports state.

### 4. Documentation and skill wiring

- `src/kit/skills/_cli-fab.md` — new verb reference (§ fab pane) + dispatch § notes the layering.
- `src/kit/skills/_cli-agents.md` — the probe recipes (rpsr pre-ship probe, first-run wall discovery) rewritten onto the new 3-command flow: `fab pane open --provider X` → `fab pane ready` → `fab pane deliver --text`.
- `_preamble.md` § pane readiness gate / CLI-Adapter Dispatch — unchanged contracts, but the carve-out prose may reference the primitives; verify no owned-rule duplication (owner-or-pointer convention).
- `docs/specs/harness-adapters.md` — records the layering (primitives vs dispatch bindings); SPEC mirrors for every touched skill file per the sweep class.

## Affected Memory

- `runtime/pane-commands.md`: (modify) new `open`/`ready`/`deliver` verbs, send's unknown-state posture change, exit-code scheme extension if any.
- `runtime/dispatch.md`: (modify) dispatch verbs as thin bindings over the pane primitives; contract unchanged.
- `runtime/agent-primitives.md`: (modify) pre-send gate posture; the probe recipe moves to the new verbs.

## Impact

- `src/go/fab/internal/dispatch/gate.go`, `pane_mode.go` → logic relocation into `internal/pane` (+ cmd wiring for three new `fab pane` subcommands, `src/go/fab/cmd/fab/`).
- `internal/spawn` / `internal/agent` — `fab pane open` reuses the existing interactive-command resolution (`fab agent --provider` machinery / `WithProfile` substitution); no new resolution logic.
- Tests: gate/pane tests move with the code; new cmd-level tests for the three verbs; pane-family exit-code scheme (2 = pane missing / 3 = tmux failure) extended consistently.
- Skill files + SPEC mirrors + `docs/specs/harness-adapters.md` (sweep class).
- **Hard sequencing**: branch after `260810-ki9v-kimi-pane-enablement` merges — it edits `squeeze` in the exact code this change relocates; extracting first would force a semantic (not mechanical) conflict on the verifier.

## Open Questions

- (none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Layer as pane-addressed primitives + dispatch thin bindings (not ad-hoc dispatch flags) | Discussed — user accepted the recommendation and its stated tradeoff explicitly | S:90 R:70 A:85 D:85 |
| 2 | Certain | Dispatch verbs' external contract stays byte-identical | Stated design constraint of the layering; existing tests enforce it | S:85 R:75 A:90 D:90 |
| 3 | Confident | `fab pane open` does a plain split; worker-column placement stays dispatch-owned | Placement is pipeline policy; primitives stay minimal — clear front-runner, not user-confirmed | S:60 R:75 A:80 D:70 |
| 4 | Confident | `pane send` unknown-state → warn-and-proceed (refusal only for parseable non-idle) | Discussed as item 5 of the proposal; preserves the guard where the convention reports | S:70 R:80 A:80 D:75 |
| 5 | Confident | `deliver` supports both `--prompt-file` (pointer parity) and `--text` (probe use) | Probe use case demonstrated live; dispatch parity required for the binding | S:65 R:80 A:80 D:75 |
| 6 | Confident | Relocated code lands in `internal/pane` (vs a new sibling package) | Existing shared-helper home; import-cycle risk unassessed — apply decides after reading the graph | S:50 R:70 A:60 D:55 |
| 7 | Confident | `fab pane ready`'s typed sentinel probe is acceptable against arbitrary user-named panes (documented side effect, buffer cleared) | Same mechanics dispatch already uses pre-delivery; generic addressing widens exposure — flag in review | S:50 R:60 A:65 D:55 |

7 assumptions (2 certain, 5 confident, 0 tentative, 0 unresolved).
