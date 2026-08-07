# Intake: fab dispatch wait — event-driven blocking wait replaces the 30s poll loop

**Change**: 260806-mkfj-dispatch-wait-event-driven
**Created**: 2026-08-06

## Origin

Conversational — a `/fab-discuss` session reviewing a live two-pane screenshot of the CLI-adapter dispatch wiring in action, followed by `/fab-draft`.

> This is how conversation happens between main agent and worker agent (subagent) if done via panes. Its a polling mechanism. Is there a better way to do this? Something more akin to a push mechanism?

The screenshot showed the orchestrator agent burning an inference turn every ~30s — run `fab dispatch status`, think "Watching continues", re-arm a background poll-loop command — while the pane worker executed a long apply stage. Key decisions from the discussion:

- The **native Agent-tool adapter already is push** (the harness re-invokes the orchestrator when a background subagent completes). The polling cost is specific to the CLI/pane adapter, because `fab dispatch` sits outside the harness.
- The harness offers a seam to convert the CLI path back to push: a **background Bash command re-invokes the agent when it exits**. The orchestrator in the screenshot already half-used it — it wrapped a *poll loop* in a background command ("Poll apply dispatch status every 30s (10 polls, then report)"). Replacing the poll loop with a *blocking wait* makes it a genuine push: zero turns until something happens.
- Agreed design: a new `fab dispatch wait <change> <stage> [--timeout N]` blocking verb + skill wiring that runs it as a background command; the bounded `--timeout` is not a poll interval but the **peek-on-suspicion cadence** (the recovery policy survives intact).
- **Rejected**: the worker (or a completion hook) pushing into the orchestrator's pane via tmux send-keys. Send-keys delivery is documented-flaky (the printed-prompt trap — see `operator-staged-command-is-printed-text` lore), it forks the cross-adapter contract (native workers have no such channel), and the pipeline's verb set deliberately excludes typing at agents (`_preamble.md` § Recovery policy: "The pipeline NEVER sends keys to a worker" — the same reasoning applies in the reverse direction).
- Noted in passing: the *worker's* own until-grep-sleep loop (waiting on an e2e suite in a background shell) is fine as-is — a background shell burns no inference; only agent-turn polling is expensive.

## Why

1. **The pain**: on the CLI-adapter dispatch path (headless or pane), the orchestrator polls `fab dispatch status` every 30s (`_preamble.md` § CLI-Adapter Dispatch step 2). Every poll is a full inference turn — model wake-up, tool call, "Watching continues" narration, re-arm. A 90-minute apply stage costs ~180 idle turns of pure spin. The screenshot that prompted this change shows the orchestrator "Cooked for 23s/30s/33s" between polls — real thinking-time and tokens spent observing that nothing happened.
2. **The consequence of not fixing it**: CLI/pane dispatch — the adapter that exists precisely for long, watchable stages — is the most expensive way to run a stage, in tokens and in context-window burn (each poll turn also grows the orchestrator's transcript, accelerating compaction). As pane dispatch becomes the default inside tmux (`dispatch.watchable`), every watched pipeline run pays this tax.
3. **Why this approach**: convert long-poll to push at the existing harness seam. A blocking `fab dispatch wait` collapses N status polls into one tool call; run via a background command, the harness re-invokes the orchestrator only when the wait exits (state change or timeout). No new channel, no new state, no contract fork — the five-state machine, the result-file contract, and the bounded recovery policy are unchanged; only the *observation mechanism* changes. Alternatives rejected: send-keys push into the orchestrator (flaky, contract-forking — see Origin); leaving it as-is (the cost scales with stage duration).

## What Changes

### 1. New CLI verb: `fab dispatch wait <change> <stage> [--timeout <secs>] [--json]`

Seventh subcommand in the `fab dispatch` family (start, restart, status, logs, kill, clean → + wait). Blocks until the dispatch's state leaves `running`, then prints the state exactly as `status` does and exits 0.

- **Derivation reuse**: the wait loop re-derives state on an internal tick (~2s) using the exact pure functions `status` uses — `DeriveState` (headless) / `DerivePaneState` (pane) — via the same record loader. `wait` and `status` therefore cannot disagree about state, by construction.
- **Internal tick, not fsnotify**: the discussion floated an fsnotify watch on `{stage}-result.yaml`, but a file watcher cannot observe pid/pane death (the `orphaned` trigger), so a periodic liveness probe is needed regardless. A ~2s in-process tick over the existing derivation gives sub-2s event latency at zero new dependency and zero new derivation code. The efficiency being bought is *inference-turn* elimination, not filesystem-poll elimination — a 2s stat loop inside one Go process is free.
- **`--timeout <secs>`**: optional upper bound on the block. On expiry, `wait` prints the still-current state (necessarily `running` — any other state would have ended the wait) and exits 0. Absent → wait indefinitely. The printed state string is the discriminator: a consumer that reads `running` knows the wait timed out; every other string is a terminal state reached.
- **Exit codes**: 0 for any successfully-observed state (terminal or timeout); non-zero only for real errors (no dispatch record for the pair, unresolvable change — same error surface as `status`).
- **Already-terminal**: if the state is already non-`running` at entry, return immediately printing it (idempotent; safe to re-arm after a restart or re-run after interruption).
- **`--json`**: same object `status --json` emits (mode discriminator + the mode's identity keys), rendered through the same code path.
- **Orphan latency win**: today an orphaned worker is noticed at the next 30s poll; `wait`'s liveness tick surfaces it in ~2s, so the recovery policy's single automatic restart fires almost immediately.
- **Platform**: the wait loop itself is platform-independent (it calls the existing per-mode liveness probes, which carry their own POSIX/Windows split — `Alive` conservatively `false` on Windows, where headless dispatch already errors at `start`).

### 2. Skill wiring: replace the poll loop (`_preamble.md` § CLI-Adapter Dispatch)

Step 2 ("Poll") is rewritten from `fab dispatch status` + `sleep 30` to:

- **Preferred (push)**: run `fab dispatch wait <change> <stage> --timeout 300` as a **background command** (the harness's notify-on-exit seam — in Claude Code, Bash `run_in_background`). The orchestrator does other work or ends its turn; when the command exits, the harness re-invokes it with the printed state. Terminal state → step 3 handling, unchanged. `running` (timeout) → this wake IS the peek-on-suspicion moment: peek, classify (a/b/c), re-arm a fresh `wait`.
- **Fallback (degraded, still 10×)**: a harness without notify-on-exit background commands runs the same `wait --timeout 300` as a plain foreground blocking call — one turn per 5 minutes instead of ten.
- **`--timeout 300` preserves the existing peek cadence exactly**: the old rule was "peek every 10th result-less poll" at 30s = every 5 minutes. The timeout is the peek cadence, not a poll interval.
- **Recovery policy text updates in place**: "every 10th result-less poll" becomes "every timeout-return of `fab dispatch wait`" (same 5-minute cadence); the restart budget, the three-way peek classification, escalation, and the never-send-keys rule are all unchanged. After the single automatic restart on `orphaned`, the orchestrator re-arms `wait`.
- The pane-mode bullet ("the same fixed `sleep 30` poll applies") updates to reference the same wait wiring.
- `fab-continue.md` and `_pipeline.md` restate the poll contract in their dispatch-adapter passages — both update to the wait wiring. `fab-adopt.md` references § CLI-Adapter Dispatch without restating the cadence, so it inherits the change with no edit (verify at apply).

### 3. Documentation surfaces

- `src/kit/skills/_cli-fab.md` § fab dispatch — add the `wait` subcommand (constitution: CLI change ⇒ `_cli-fab.md` + tests).
- `docs/specs/harness-adapters.md` — the shared-protocol section that names the 30s poll cadence updates to the wait/push observation model (this spec was authored by the 3c change and is maintained through the pipeline).
- SPEC mirrors in the same change: `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_cli-fab.md` (constitution: skill change ⇒ SPEC mirror; sweep the whole mirror class per code-quality.md § Sibling & Mirror Sweeps).
- Sweep note: `docs/specs/skills/SPEC-git-pr-review.md` also contains a "sleep 30" — that is the unrelated Copilot-review wait; do NOT touch it.

### 4. Explicit non-goals

- **No send-keys / no worker-initiated push channel** — rejected above; the cross-adapter contract stays observation-only.
- **No change to `status`, `start`, `restart`, `kill`, `clean`** — `wait` is purely additive; `status` remains the one-shot probe (still used by refuse-if-running explanations, `--json` consumers, and human spot checks).
- **No new dispatch state, no record schema change, no migration** — `wait` reads what exists.
- **No backoff/cadence config** — the internal tick is a constant; the wiring's `--timeout 300` is a skill-text constant, overridable per-invocation like any flag.

## Affected Memory

- `runtime/dispatch`: (modify) add the `wait` verb to the command family (subcommand table, skill-wiring paragraph — "sleep 30 polling" → wait/push wiring, source-layout note gains `dispatch_wait.go`)
- `pipeline/execution-skills`: (modify) the dispatch-adapter passage restating the poll cadence updates to the wait wiring
- `_shared/context-loading`: (modify) the Per-Stage Model Resolution passage restating "sleep 30 poll" updates likewise

## Impact

- **Go** (`src/go/fab/`): `internal/dispatch` — new wait-loop core (reusing loader + `DeriveState`/`DerivePaneState` + liveness probes) + table tests incl. a timeout test and an already-terminal fast-path test; `cmd/fab/dispatch_wait.go` — thin cobra wiring registered in the `dispatch` parent; scope test runs to the dispatch package + cmd tests first.
- **Kit skills** (`src/kit/skills/`): `_preamble.md` (§ CLI-Adapter Dispatch steps 2–3 + Recovery policy wording + pane-mode bullet), `_pipeline.md`, `fab-continue.md`, `_cli-fab.md`.
- **Specs**: `docs/specs/harness-adapters.md`; mirrors `SPEC-_preamble.md`, `SPEC-_pipeline.md`, `SPEC-fab-continue.md`, `SPEC-_cli-fab.md`.
- **Memory** (hydrate): the three files above.
- **No user-data restructuring** → no migration file needed.
- **Version-skew**: none — kit skills and the binary ship together; the wiring text lands in the same release as the verb.

## Open Questions

*(none — all decision points resolved in discussion or graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New verb is `fab dispatch wait <change> <stage>`, additive seventh subcommand; no changes to existing verbs | Discussed — user approved drafting exactly this; family precedent makes the shape deterministic | S:90 R:85 A:95 D:90 |
| 2 | Certain | Skill wiring runs `wait` via the harness's background notify-on-exit seam (Claude Code: Bash `run_in_background`), replacing the `sleep 30` status poll | Discussed — the push seam was the core of the agreed design | S:90 R:80 A:90 D:85 |
| 3 | Confident | Internal mechanism is a ~2s in-process derivation tick, not fsnotify | Refines the discussion's fsnotify sketch: liveness (orphan detection) needs a periodic probe regardless, so a watcher adds a dependency without removing the tick; user-visible contract (blocking, ~2s latency) identical | S:70 R:85 A:90 D:75 |
| 4 | Certain | Wiring timeout is 300s, preserving the existing peek-on-suspicion cadence (10 polls × 30s = 5 min) exactly | Arithmetic from the existing recovery policy (discussion said "10 min ≈ every 10th poll" — corrected here: 10 × 30s = 5 min) | S:85 R:95 A:100 D:95 |
| 5 | Confident | Timeout expiry prints the current state (`running`) and exits 0; state string is the timeout discriminator; errors (no record) stay non-zero like `status` | Mirrors `status` semantics; the consuming skill reads stdout either way; a distinct exit code would add a second channel for information the string already carries | S:60 R:90 A:85 D:70 |
| 6 | Confident | `wait` supports `--json` via `status`'s existing render path | Parity at near-zero cost; keeps the JSON surface single-sourced | S:55 R:95 A:90 D:80 |
| 7 | Confident | Recovery policy semantics unchanged: peek classification, single restart budget, escalation, never-send-keys — only the cadence carrier changes (timeout-return replaces 10th-poll counting) | Discussed — "the recovery policy survives intact" was an explicit design constraint | S:80 R:75 A:85 D:80 |
| 8 | Confident | Foreground blocking `wait --timeout 300` documented as the fallback for harnesses without notify-on-exit background commands | Cross-harness spec (harness-adapters.md) must not assume a Claude-Code-only seam; foreground blocking is still a 10× turn reduction | S:60 R:85 A:80 D:75 |
| 9 | Tentative | Internal tick constant is 2s (not configurable) | Any 1–5s value is fine; constant chosen for sub-poll orphan latency without measurable cost; trivially adjustable later | S:50 R:95 A:70 D:60 |

9 assumptions (3 certain, 5 confident, 1 tentative).
