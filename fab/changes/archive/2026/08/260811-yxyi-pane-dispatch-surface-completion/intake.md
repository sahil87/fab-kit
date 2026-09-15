# Intake: Pane/Dispatch Surface Completion

**Change**: 260811-yxyi-pane-dispatch-surface-completion
**Created**: 2026-08-11

## Origin

> pane/dispatch surface completion — additive: fab pane kill, fab pane await, server field in fab dispatch status --json, --json on fab pane open/ready/deliver, pane process exit-code alignment to the 2/3 scheme

Conversational — emerged from a `/fab-discuss` session (2026-08-11) consolidating the agent-talking surface (`fab pane` + `fab dispatch`) and evaluating whether to extract it into a separate binary. The extraction was rejected; instead the session identified six cleanup/completion items and grouped them into two changes by the additive-vs-contract-change line:

- **This change (additive only)**: the five items below. No existing caller's behavior changes.
- **Backlog `[answ]` (contract change, deliberately excluded)**: `fab pane send --answer` mode — it changes a refusal contract and rewires operator/`_cli-agents` skill prose, so it needs its own focused review.

Key decisions from the discussion: bundle the five additive items into ONE change because they share a single mechanical sweep class (`pane-commands.md`, `dispatch.md`, `_cli-fab.md`, tests) — paying that sweep once; `fab pane await` is the SRAD-heaviest item and splits out only if intake scoring comes back weak (it did not). All five gaps were re-verified on the rebased tree (post `cebbc156`) before drafting.

## Why

1. **Pain point**: The provider-generic pane layer (1lah's `fab pane open`/`ready`/`deliver`) is deliberately usable outside fab pipelines, but its lifecycle is incomplete and its outputs are script-hostile:
   - There is no generic **kill** — `pane.KillPane` exists (`src/go/fab/internal/pane/create.go:308`) but has no verb, so the operator's removal paths and probe cleanups fall back to raw `tmux kill-pane`, bypassing the family's validated 2/3 exit-code contract.
   - There is no generic **await** — the two honest completion signals (`@rk_agent_state` flipping to `idle`; a contract file appearing) are today pollable only by hand-rolled loops (the operator's heartbeat, `_cli-agents.md`'s prose-level "artifact-oriented await").
   - `fab dispatch status --json` omits the tmux **socket** — `_preamble.md` § CLI-Adapter Dispatch documents the workaround explicitly ("obtain the exact socket-included command from `fab dispatch logs` … because `status --json` does not carry the socket").
   - `fab pane open`/`ready`/`deliver` have no `--json` while the rest of the family (`map`, `capture`, `window-name`) does — scripts must parse `opened pane %N (provider …)` and multi-line readiness reports.
   - `fab pane process` is the one family member still exiting flat 1 on a missing pane (`pane_process.go` routes through main's ERROR formatter), so the family's "2 = pane missing / 3 = other tmux failure" contract carries an asterisk.
2. **Consequence of not fixing**: skills and external consumers (operator, run-kit) keep routing around the binary with raw tmux and fragile text parsing — the exact drift class the gated verbs exist to prevent — and every future consumer re-learns the socket-lookup workaround.
3. **Why this approach**: complete the existing surface rather than extract a new binary (rejected in the discussion: the completion signal is convention-bound, run-kit owns adjacent conventions, the tmux choreography is a depreciating workaround for missing provider APIs, and the toolkit already suffers version-skew pain). All five items are additive, so they bundle into one mechanical review pass.

## What Changes

### 1. New verb: `fab pane kill <pane>`

Expose the existing `pane.KillPane(paneID, server)` helper as a ninth `fab pane` subcommand.

- **Source**: new `src/go/fab/cmd/fab/pane_kill.go`, registered on `paneCmd` (inherits the persistent `--server`/`-L` flag).
- **Validation**: `pane.ValidatePane` first — missing pane → `Error: pane <id> not found`, **exit 2** (in-handler, `*PaneNotFoundError` via `errors.As`); any other tmux failure → **exit 3**. Same scheme as `capture`/`send`/`ready`/`deliver`/`window-name`.
- **Output**: `killed <pane>` on success (plus `server: <name>` line when non-default, matching `open`'s form).
- Generic and record-free: no dispatch-record interaction, no `.fab-dispatch/` state. `fab dispatch kill` (record-keyed, ungated recovery) is unaffected and remains the pipeline's kill.

### 2. New verb: `fab pane await <pane> [--file <path>] [--timeout <secs>]`

The generic blocking wait — the completion primitive the record-free layer lacks. Blocks until **any** waitable signal fires, then prints a one-word report and exits 0 (report string is the discriminator, the `fab dispatch wait` precedent):

| Report | Fired by |
|--------|----------|
| `idle` | the pane's `@rk_agent_state` resolves to `idle` (same parser as `map`/`send`/`capture`) |
| `file` | the `--file <path>` file exists |
| `running` | `--timeout` expired with neither signal — **exit 0** (timeout bounds the observer, not the worker) |

- **Signals compose as OR**: with both `--file` and an instrumented pane, whichever fires first wins.
- An uninstrumented pane (agent state unknown/absent) with **no `--file`** is an immediate error — there is nothing observable to wait on, and blocking until timeout would report `running` while meaning "I was never watching anything". <!-- assumed: error-immediately posture for unwaitable panes — cleaner than a guaranteed-useless block; alternative (warn and wait on state anyway, in case instrumentation appears mid-wait) rejected as a silent no-op trap -->
- **Cadence/bounds**: internal ~2s re-derive tick and `--timeout` default 300, mirroring `dispatch wait`'s observer conventions. <!-- assumed: default 300s and ~2s tick copied from the dispatch wait precedent — no independent signal for a different default -->
- **Errors**: pane missing → exit 2; other tmux failure → exit 3; unwaitable (no signal to watch) → exit 1 via RunE. A pane that **dies mid-wait** reports `gone` and exits 2 (the wait cannot complete and the caller must branch on cause).
- Non-goals: no `fab dispatch await` binding (`fab dispatch wait` already owns the record-keyed side); no operator-skill rewiring onto `await` in this change (follow-up once the verb is proven).

### 3. `server` field in `fab dispatch status --json`

Add `Server string \`json:"server,omitempty"\`` to `dispatchStatusJSON` (`src/go/fab/cmd/fab/dispatch_status.go:40`), sourced from the record's `Server` field. Additive shape — existing consumers ignore unknown keys (the `repo`/`window_id`/`pr_url` precedent). Retires the documented logs-lookup workaround; `_preamble.md`'s "obtain the socket from `fab dispatch logs`" sentence and the equivalent claim in `docs/memory/runtime/dispatch.md` are updated in the same change. Human output unchanged.

### 4. `--json` on `fab pane open`, `ready`, `deliver`

Each gains a `--json` bool emitting a single JSON object on stdout (always an object, including for every classification — the `window-name` precedent):

```json
// open:    {"pane": "%12", "provider": "claude", "server": null}
// ready:   {"state": "parked", "pane": "%12", "server": null, "snippet": "…last 20 lines…"}
// deliver: {"pane": "%12", "source": "prompt", "path": ".fab-dispatch/yxyi/apply-prompt.md"}
```

- `ready --json`: `state` ∈ `{ready, booting, parked}`; `snippet` is the same trailing-blank-trimmed 20-line capture the text report carries (empty string when empty); all three states still exit 0.
- `deliver --json`: `source` ∈ `{prompt, text}`; `path` present only for `prompt`. Emitted only on verified delivery (failures keep the existing stderr + non-zero contract).
- Nullability via the existing `toNullable`-style contract: empty server → `null`.
- Plain-text output remains byte-identical to today in all three verbs.
- Scope: the `fab pane` primitives only — the `fab dispatch open/ready/deliver` bindings keep their text-only output (their consumer is the skill wiring, which reads the report words).

### 5. `fab pane process` exit-code alignment

Adopt the family scheme in `pane_process.go`: pane missing → **exit 2** (in-handler, via the same `*PaneNotFoundError` branch `capture`/`send` use), any other tmux validation failure → **exit 3**. Today it returns a flat exit 1 through main's formatter. The `map` subcommand is deliberately untouched (multi-pane discovery, no single target pane to be "missing"). `pane_exitcode_test.go` gains `process` rows; the memory note documenting process as the exception is updated.

### Cross-cutting

- **`_cli-fab.md`** (constitution CLI constraint): § fab pane gains `kill` + `await` entries and the `--json` flags; § fab dispatch status documents the `server` JSON field; the pane exit-code table drops the `process` exception.
- **Tests**: new `pane_kill`/`pane_await` coverage (pure decision halves table-tested; integration against a real tmux server following the `set-option -p` writer-simulation precedent), exit-code rows for `process`, JSON-shape tests for the three verbs and dispatch status.
- **No skill flow changes** — no `SPEC-*.md` mirror updates required under constitution v1.5.0 (mirrors trigger only on flow/tool/sub-agent changes). Any `_cli-agents.md` mention of `await` is prose-only.

## Affected Memory

- `runtime/pane-commands`: (modify) add `kill` + `await` subcommand sections (nine→ten verbs), `--json` on open/ready/deliver, drop the process exit-code exception note
- `runtime/dispatch`: (modify) `status --json` field table gains `server`; remove the socket-from-logs workaround claim
- `runtime/agent-primitives`: (modify) the "artifact-oriented await" prose gains the first-class `fab pane await` reference

## Impact

- **Go**: `cmd/fab/pane_kill.go` (new), `cmd/fab/pane_await.go` (new), `pane_ready.go`/`pane_open.go`/`pane_deliver.go` (`--json`), `dispatch_status.go` (field), `pane_process.go` (exit codes), `pane.go` (subcommand registration); possibly a small `internal/pane` await/tick helper. Tests alongside each (`pane_exitcode_test.go` + new files).
- **Kit prose**: `src/kit/skills/_cli-fab.md`; `_preamble.md` workaround sentence (§ CLI-Adapter Dispatch pane-output bullet).
- **Docs**: the three runtime memory files at hydrate.
- **Consumers unaffected**: operator skill keeps working unchanged (adopting `kill`/`await` is a follow-up); run-kit ignores unknown JSON keys.
- **No migration**: no user-data restructuring; all additions are additive surface.

## Open Questions

- None — the two Tentative rows below are marked inline and reviewable via `/fab-clarify`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the five additive items; `send --answer` excluded to backlog `[answ]` | Discussed — user chose the two-change grouping on the additive-vs-contract line | S:95 R:90 A:95 D:95 |
| 2 | Certain | `pane kill` exposes existing `KillPane` with the family 2/3 exit scheme + `--server` | Family precedent pins every choice; helper already exists | S:90 R:85 A:95 D:95 |
| 3 | Confident | `await` signal set = OR of `@rk_agent_state` idle and `--file` existence | Discussed — the two honest signals named in the design conversation; OR is the only composition that serves both instrumented and contract-based flows | S:70 R:75 A:70 D:65 |
| 4 | Tentative | `await` on an uninstrumented pane with no `--file` errors immediately rather than blocking to timeout | A block that watches nothing is a silent no-op trap; but warn-and-wait is defensible — marked inline | S:45 R:60 A:40 D:35 |
| 5 | Certain | `await` timeout expiry exits 0 printing `running`; report string is the discriminator | `fab dispatch wait` precedent is explicit and documented | S:80 R:85 A:85 D:80 |
| 6 | Tentative | `await --timeout` defaults to 300s with a ~2s internal tick | Copied from the dispatch-wait observer conventions; no independent signal for this verb's own default — marked inline | S:40 R:60 A:40 D:35 |
| 7 | Confident | JSON shapes as specified (`state`/`pane`/`server`/`snippet` for ready; `source`/`path` for deliver; always-an-object) | `window-name`/`map`/`capture` JSON precedents; exact key names are cheap to adjust at review | S:70 R:85 A:75 D:60 |
| 8 | Certain | `process` adopts exit 2/3 via the `*PaneNotFoundError` branch; `map` stays untouched | User picked the item; mechanism is the family's existing typed-error path | S:85 R:90 A:90 D:90 |
| 9 | Certain | `status --json` gains `server` (omitempty), sourced from the record; human output unchanged | Additive-JSON-field precedent (`repo`, `window_id`, `pr_url`); workaround is documented and retired in-change | S:85 R:90 A:90 D:85 |
| 10 | Certain | No SPEC-mirror updates: no skill flow/tool/sub-agent changes; `_cli-fab.md` + tests are the mandatory sweep | Constitution v1.5.0 (amended 2026-08-11) narrowed the mirror rule; CLI constraint unchanged | S:80 R:85 A:90 D:85 |
| 11 | Confident | `await` stays generic-only (no dispatch binding); operator rewiring onto kill/await deferred | `fab dispatch wait` already owns the record-keyed side; rewiring belongs with `[answ]`'s skill pass or later | S:75 R:85 A:80 D:75 |

11 assumptions (6 certain, 3 confident, 2 tentative, 0 unresolved).
