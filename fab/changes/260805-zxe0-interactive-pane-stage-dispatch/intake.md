# Intake: Interactive Pane Stage Dispatch (Third Adapter)

**Change**: 260805-zxe0-interactive-pane-stage-dispatch
**Created**: 2026-08-05

## Origin

> Third dispatch adapter: run a CLI-dispatched pipeline stage in a tmux pane the user can watch and steer, built on `_cli-agents` primitives

Conversational origin (`/fab-discuss` session, 2026-08-05) — Phase 2 of a two-phase plan. Phase 1 is `260805-nvad-cli-agents-helper-provider-spawn` (the `_cli-agents` helper + provider-addressable spawn), drafted in the same session. **This change depends on nvad shipping first** — it consumes nvad's spawn/deliver/peek/await primitives.

The user asked whether pane dispatch could be bundled into Phase 1; the decision was to split: this change is Go + a cross-cutting spec amendment with genuine design decisions, and bundling would have tripled Phase 1's mirror-sweep class. Drafted now (unactivated) to capture the three design decisions while fresh; refine via `/fab-clarify` before activation.

## Why

1. **The pain point**: CLI-dispatched stages (the `dispatch=` path — codex/gemini or headless claude) are black boxes. Watching is partially solved (`fab dispatch logs --tail` in a pane the user opens), but **steering is impossible**: the worker is a detached headless process (`SysProcAttr{Setsid:true}`), not a session anyone can talk to. Native Agent-tool dispatch has peek-and-converse richness; the CLI adapter has none. This asymmetry is the single largest richness loss when running pipeline stages cross-provider.

2. **The consequence of not fixing it**: "use codex for the next four stages" stays a fire-and-forget experience — a stuck or misdirected worker can only be killed and re-run, never nudged. Users avoid the CLI adapter for anything they care about, which keeps cross-provider stages second-class.

3. **Why this approach**: an interactive tmux-pane worker is the one universal interface every agent CLI supports natively. Running the stage's worker *interactively in a pane* — prompt delivered via the `_cli-agents` send discipline, completion detected via the existing result-file contract — recovers watch AND steer with no per-provider integration work. The five-state dispatch machine and the `{stage}-result.yaml` contract already exist; this adapter reuses them with a remapped detection mechanism.

## What Changes

### 1. `fab dispatch` gains a pane mode

`fab dispatch start <change> <stage> --pane` (flag name final at apply; see Assumptions) runs the stage worker **interactively in a tmux pane/window** instead of as a detached headless process:

- Resolves tier → provider exactly as today, but composes the provider's **interactive session invocation** rather than `dispatch_command` (see §3 mode selection).
- Creates the pane/window via the `internal/pane` helpers (reusing the operator's window-open pattern: `tmux new-window -n <name> -c <dir> "<cmd>"`), records pane identity in `.fab-dispatch/{id}/` state.
- Requires a reachable tmux server; `--pane` without tmux is a hard error (exit non-zero, actionable stderr) — the headless path remains the tmux-independent default, preserving dispatch.md's tmux-independence guarantee for everything except this explicit opt-in.

### 2. Prompt delivery: prompt file + pointer send

The stage prompt (same block prompt as both existing adapters, all Dispatch-Prompt Obligations intact) is written to `.fab-dispatch/{4-char-change-id}/{stage}-prompt.md`. The pane worker receives a one-line pointer — "Read `.fab-dispatch/{id}/{stage}-prompt.md` and execute it" — delivered via the `_cli-agents` pre-send validation + delivery-probe discipline. Rationale: multi-thousand-token prompts cannot ride send-keys or argv reliably (documented printed-prompt trap); the file is also a debugging artifact, consistent with `.fab-dispatch/`'s transient-comms role and its existing cleanup story (archive-time + `fab dispatch clean`, no auto-GC).

### 3. Mode selection surface

Where "this stage runs interactively" is declared. Working decision (Tentative — see Assumptions): a **per-invocation flag** on `fab dispatch start` (`--pane`), surfaced to skills as a user directive ("run review where I can watch") rather than a config field. The alternative — a third per-provider command field (e.g. `pane_command`) — was considered and deferred: providers already deliberately carry two unmerged invocation fields, and the interactive invocation is derivable (it is the provider's `session_command` — the same string `fab agent` composes), so no new config field is required in v1. If a provider's interactive grammar diverges from its `session_command`, that becomes the trigger to add the field later (data-only config addition).

Sequencer wiring: the dispatch sites' branch stays keyed on `dispatch=` presence; the pane mode is an *option within the CLI-adapter branch* (skills pass `--pane` when the user asked for a watchable stage). No change to `fab resolve-agent` output in v1.

### 4. Completion detection + state remap

An interactive worker never exits on task completion — it finishes and sits at its prompt. Detection therefore keys on the **result file**, which is already the contract's success token:

- `running` — pane alive, no `{stage}-result.yaml`.
- `done` — result file present (validated exactly as today; a review `verdict: fail` inside a `done` result remains a review outcome, not a dispatch failure).
- `orphaned` — pane dead (or tmux server gone), no result file. On the pane path, worker-crash and kill collapse into `orphaned` — there is no exit-code channel, so the headless path's `failed` state is **unreachable for pane dispatches** and `failed (no-result)` (clean exit, no result) has no pane analogue. `fab dispatch status` output remains byte-stable; pane dispatches simply never emit the two exit-code-derived states.
- Polling cadence unchanged (fixed `sleep 30`).

### 5. Steering semantics (documented contract, not code)

The user may converse with the pane worker mid-stage. The block contract is unchanged: the worker still owes `{stage}-result.yaml`, still runs the terminal `fab status refresh` epilogue, still never runs status *transition* commands — the orchestrator owns transitions. Steering is human input into the worker's context, exactly like answering a sub-agent's question; a steered worker that gets redirected away from producing a result eventually surfaces as a never-`done` dispatch the orchestrator escalates on. This is documented in the spec amendment, with the note that `fab dispatch kill` needs a pane-mode implementation (kill the pane, mark orphaned).

### 6. Spec + docs amendments (sweep class, declared up front)

- `docs/specs/harness-adapters.md` — two adapters → three (native / headless CLI / interactive pane); the shared protocol table gains the pane column (prompt-file delivery, result-file-only detection, reachable-states subset).
- `src/kit/skills/_preamble.md` — § CLI-Adapter Dispatch gains the pane-mode option; § Dispatch-Prompt Obligations notes the prompt-file delivery variant.
- `src/kit/skills/_cli-fab.md` § fab dispatch — the `--pane` flag, state-reachability notes, kill behavior (constitution: CLI change ⇒ `_cli-fab.md` + tests).
- Dispatch-site skills (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md`) — only if their carried text enumerates adapter modes (grep-verify during apply; the canonical procedure lives in `_preamble`, which they reference).
- SPEC mirrors: `SPEC-_preamble.md`, `SPEC-fab-continue.md`/`SPEC-fab-adopt.md` as swept; aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) where they restate the two-adapter model.
- Go tests: pane-mode start/status/kill state transitions (pane alive/dead × result present/absent), tmux-absent error, prompt-file write.

### Non-goals

- No change to native Agent-tool dispatch, `fab resolve-agent` output, or the providers schema (no new config field in v1 — see §3).
- No completion *notification* (polling stands; an `rk notify` line in the worker prompt remains the opt-in signal).
- No auto-GC change for `.fab-dispatch/` (prompt files ride the existing cleanup paths).
- Not a replacement for the headless CLI adapter — `--pane` is opt-in per invocation; unattended pipelines keep the tmux-independent default.

## Affected Memory

- `runtime/dispatch.md`: (modify) pane mode — prompt-file delivery, result-file-only detection, reachable-state subset, kill semantics, tmux-reachability requirement scoped to `--pane`
- `runtime/operator.md`: (modify) only if the operator's monitored-set/watch behavior references dispatch modes (grep-verify)
- `pipeline/execution-skills.md`: (modify) dispatch-seam description gains the pane option
- `_shared/context-loading.md`: (modify) § Per-Stage Model Resolution / CLI-Adapter Dispatch mirror of the `_preamble` amendment

## Impact

- `src/go/fab/`: `cmd/fab/dispatch*.go`, `internal/dispatch` (state detection), reuse of `internal/pane` + `internal/spawn`; `_test.go` files
- `src/kit/skills/`: `_preamble.md`, `_cli-fab.md`, `_cli-agents.md` (consumed, possibly cross-referenced), dispatch-site skills as swept
- `docs/specs/`: `harness-adapters.md` (primary amendment), skills SPEC mirrors, aggregate specs as swept
- **Depends on**: `260805-nvad-cli-agents-helper-provider-spawn` (must ship first — this change consumes its procedures and cross-references the helper)
- No migration (no user-data restructuring; `.fab-dispatch/` is transient and gains one new file type)

## Open Questions

- Should the pane worker's window carry the operator's `»` marker convention (making it operator-monitorable when an operator is running), or stay unmarked to avoid implying operator ownership? (Low stakes; decide at apply or via `/fab-clarify`.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Completion detection = result-file presence; pane liveness distinguishes `running` from `orphaned` | Discussed — an interactive worker never exits on completion; the result file is already the contract's success token | S:75 R:80 A:90 D:85 |
| 2 | Certain | Prompt delivery = prompt file at `.fab-dispatch/{id}/{stage}-prompt.md` + one-line pointer sent via `_cli-agents` discipline | Discussed — send-keys/argv cannot carry stage prompts (printed-prompt trap); file doubles as debug artifact | S:75 R:85 A:90 D:85 |
| 3 | Certain | Mode selection = per-invocation `--pane` flag reusing the provider's `session_command`; no new provider config field in v1 <!-- clarified: user confirmed the --pane flag over a provider config field (and over both-from-day-one) at activation, 2026-08-05 --> | Asked — user confirmed the flag; config field stays a data-only later addition if a provider's interactive grammar diverges | S:85 R:80 A:90 D:90 |
| 4 | Confident | Pane mode lives inside `fab dispatch` (a `--pane` flag), not a new command family | Sequencer wiring already branches at `fab dispatch`; a parallel command would duplicate the state machine | S:60 R:70 A:85 D:75 |
| 5 | Confident | `failed` / `failed (no-result)` are unreachable on the pane path (no exit-code channel); crash collapses into `orphaned` | Follows from decision 1; status strings stay byte-stable, states are a subset | S:55 R:75 A:80 D:70 |
| 6 | Confident | Steering is permitted and contract-neutral (worker still owes result + refresh epilogue; orchestrator owns transitions) | Discussed — steering equals human input to a worker; never-`done` escalation already covers redirect-away failure | S:65 R:75 A:80 D:75 |
| 7 | Confident | `--pane` without a reachable tmux server is a hard error; headless stays the tmux-independent default | Preserves dispatch.md's tmux-independence guarantee everywhere except the explicit opt-in | S:60 R:85 A:85 D:80 |
| 8 | Confident | Hard dependency: activate only after nvad ships (consumes `_cli-agents` procedures) | Discussed — the split's rationale; drafting order encodes it | S:80 R:70 A:85 D:80 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
