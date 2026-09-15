# Intake: Dispatch Worker Lifecycle Supervision

**Change**: 260806-mnri-dispatch-worker-lifecycle-supervision
**Created**: 2026-08-06

## Origin

> Operator checks periodically if the agent is stuck, and nudges it forward. Would a Worktree Agent not do the same? What happens if (say) the apply agent gets stuck on a 529 error? Who is going to nudge the subagent forward? (I see that in the case of a native subagent, claude is sometimes able to kill / restart the sub agent — we might need similar affordance even when spawning subagents via a different pane)

Conversational origin (l9ng session, 2026-08-06) — a follow-up to `260805-l9ng-auto-pane-dispatch-in-tmux` (PR #525). The user's mental model: operator → worktree-agent is tier-1 communication (operator acts like a human, periodically checks for stuck agents and nudges); worktree-agent → stage worker (pane/headless CLI dispatch) is tier-2 — and tier 2 currently has **no supervision at all**. Key decisions reached in discussion:

- **Restart, not nudge, is tier 2's recovery verb.** The operator nudges because *sessions* carry irreplaceable conversational state. A *stage dispatch* is the opposite: fab checkpoints state into artifacts (plan.md task checkboxes, the result-file contract, idempotent stages), so a restarted worker resumes from the last `[x]` — kill+restart is deterministic and loses almost nothing, while send-keys nudging is flaky (the printed-prompt trap, probe-then-retype choreography) and stays operator/human territory.
- **Bounded, with escalation** — an unbounded restart loop against a provider that is 529ing platform-wide just burns tokens; the point is bounded recovery, not liveness at all costs.
- **The 529 path specifically**: worker CLIs (claude/codex) retry overload errors internally — the pane-path replacement for the harness retry native subagents enjoy. The gap is what happens *after*: a worker that exits post-retry-exhaustion (→ `orphaned`/`failed`) today just stops the pipeline; a worker that sits alive at an error banner reads `running` forever.

**Ordering**: activate only after l9ng (PR #525) merges — this change edits the exact code and doc surfaces l9ng ships.

## Why

1. **The pain point**: on the CLI-adapter path the orchestrator is a blind poll loop with no recovery affordances. The skill wiring's five-state handling says `orphaned` → "surface and stop"; a stuck-alive worker (pane parked at an error prompt; headless process wedged) is indistinguishable from one working hard — `running` forever. Native subagent dispatch has supervision built into the harness (internal retry on 5xx, death notification, cheap re-dispatch by the orchestrator); CLI/pane dispatch has the *mechanisms* (`fab dispatch kill`, `start`-over-a-dead-attempt) but no restart command that reuses the persisted prompt, and no policy telling the orchestrator to use them.

2. **The consequence of not fixing it**: this gap is about to become common. j3cm (merged, #526) makes codex/gemini live built-in providers — a one-line tier override puts any stage on CLI dispatch — and l9ng (#525) makes that a watchable pane by default inside run-kit. Every transient worker death (post-retry 529 exhaustion, OOM, an accidentally closed tmux window) then halts an unattended pipeline that one restart would have recovered; every parked worker silently burns wall-clock until a human notices.

3. **Why this approach**: a `fab dispatch restart` command (relaunch from the already-persisted `{stage}-prompt.md`) plus a bounded poll-loop recovery policy in the dispatch-seam skills. Restart composes with l9ng's mode ladder for free — mode is re-derived from the *current* environment, so restarting an orphaned pane dispatch after a tmux server death correctly soft-falls-back to headless. The alternative — an orchestrator→worker send-keys channel — was considered and rejected: it forks the cross-adapter contract (native workers have no such channel), depends on fragile TUI delivery, and duplicates the operator's job.

## What Changes

### 1. `fab dispatch restart <change> <stage>` (Go)

New subcommand that relaunches a **non-running** dispatch from the persisted prompt:

- Reads the stage prompt from `.fab-dispatch/{id}/{stage}-prompt.md` instead of stdin — the orchestrator does not need the block prompt in context (it may have been lost to compaction). A missing prompt file is a clear error ("nothing to relaunch — run `fab dispatch start` with a prompt on stdin").
- Runs the same prologue as `start`: change resolution, config + tier→provider resolution, **refuse-if-running** (the prior record's own mode's finished signal — a genuinely running dispatch refuses and points at `fab dispatch kill`), stale-file clearing, and the same **mode-selection ladder** (l9ng): same flags (`--pane`/`--headless`/`--timeout`/`--server`, same exclusions), mode re-derived from the current environment — a restart is a fresh attempt under the existing last-attempt-only semantics, using either mode regardless of the prior attempt's.
- Output and record are byte-shaped like `start`'s (same `dispatched …` line incl. the `auto:` suffix rules; no new state strings, no attempt history, no `restarted:` marker in `{stage}.yaml`).
- Implementation: refactor `runDispatchStart`'s prompt-acquisition seam so `start` (stdin) and `restart` (state-dir file) share one launch path — no duplicated tail.

### 2. Poll-loop recovery policy (skills — `_preamble.md` § CLI-Adapter Dispatch)

The five-state handling in the dispatch seam gains bounded recovery, replacing the unconditional "surface and stop" rows:

- **`orphaned` → one automatic `fab dispatch restart`** per stage dispatch. If the restarted attempt also ends without `done` (orphaned again, or failed), escalate: surface the evidence (`fab dispatch logs` / `fab pane capture` per mode), send `rk notify` (gated on `command -v rk`, fail-silent per the rk universal rule), and stop per the stage's existing failure path.
- **`failed` keeps no *automatic* restart** — a deterministic failure (bad config, real test failure, `124` timeout) would loop. The orchestrator (an agent, not a script) MAY judge a clearly-transient failure from the log tail (provider 5xx/overload signatures) worth one restart within the **same** single-restart budget; anything else stops as today. **`failed (no-result)` always escalates** — a contract violation needs eyes, not retries.
- **Peek-on-suspicion for result-less dispatches**: every 10th result-less poll (~5 min at the fixed 30s cadence), take a read-only peek — `fab pane capture [-L <server>] <pane>` (pane) or `fab dispatch logs --tail 40` (headless) — and classify three ways: (a) visibly progressing → keep polling; (b) parked at an error banner / dead-ended → `fab dispatch kill` + restart within the same budget (or escalate if spent); (c) waiting on genuine human input (a permission prompt, a question) → escalate via `rk notify` **without killing** — a human may want to answer, and steering stays human.
- **No send-keys from the pipeline, ever** — nudging/answering is the operator's and the user's affordance; the pipeline's verbs are peek (read-only), kill, restart, notify, stop.
- The restart budget lives in the **orchestrator's context** (per stage dispatch), not on disk — no attempt counter is added to `{stage}.yaml` (preserves last-attempt-only; the worst case after orchestrator context loss is one extra restart, which is acceptable).

### 3. Docs/spec/memory sweep

- `src/kit/skills/_cli-fab.md` § fab dispatch — the `restart` subcommand (signature, prompt-file source, refuse-if-running, flag set), and the family headline (`start`/`restart`/`status`/`logs`/`kill`/`clean`).
- `src/kit/skills/_preamble.md` § CLI-Adapter Dispatch — the recovery policy above (the five-state handling rows + the pane-subset row), § Dispatch-Prompt Obligations untouched (the worker contract does not change).
- `docs/specs/harness-adapters.md` — a recovery/supervision paragraph in the shared-protocol section (states unchanged; recovery is orchestrator policy over existing states).
- SPEC mirrors of touched skills (`SPEC-_cli-fab.md`, `SPEC-_preamble.md`) + grep-sweep the sibling dispatch-seam surfaces (`_pipeline.md`, `fab-continue.md`, `fab-adopt.md` and their mirrors) for restated "surface and stop" five-state rows — the l9ng sweep showed the class is wider than the literal-phrase grep.
- Memory: `runtime/dispatch.md`, `_shared/context-loading.md`, `pipeline/execution-skills.md`.

### Non-goals

- No orchestrator→worker send-keys/nudge channel (operator territory; forks the cross-adapter contract).
- No supervisor daemon, timer, or background sweep — polling remains the only clock (fab's no-magic-background-work posture).
- No on-disk attempt history or counter (last-attempt-only preserved; the budget is orchestrator context).
- No new dispatch states and no change to the five/three-state machines, the result-file contract, or the prompt obligations — restart is a fresh attempt over existing states.
- No auto-answer of worker prompts (classification (c) notifies a human; it never types into the pane).

## Affected Memory

- `runtime/dispatch.md`: (modify) the `restart` subcommand, the shared prompt-acquisition seam, recovery-policy pointers
- `_shared/context-loading.md`: (modify) CLI-Adapter Dispatch mirror — the five-state handling rows gain the bounded-recovery policy
- `pipeline/execution-skills.md`: (modify) only if it restates the poll-loop/five-state handling (grep-verify)

## Impact

- `src/go/fab/`: `cmd/fab/dispatch_restart.go` (new) + `dispatch_start.go` (prompt-seam refactor), `internal/dispatch` as needed; `_test.go` files (incl. restart-over-orphaned, refuse-while-running, missing-prompt-file, mode-re-derivation cases)
- `src/kit/skills/`: `_cli-fab.md`, `_preamble.md` (+ sibling dispatch-seam skills per grep-sweep)
- `docs/specs/`: `harness-adapters.md`, SPEC mirrors as swept
- **Depends on**: l9ng / PR #525 merged (edits surfaces l9ng ships). j3cm is already on main. No migration (new subcommand + skill policy; no user-data restructuring).

## Open Questions

*(none — the direction, the recovery verbs, and the boundedness were all settled in discussion; residuals are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Tier 2's recovery verb is kill+restart, never send-keys nudge; nudging/answering stays operator/human territory | Discussed and agreed — stage state lives in artifacts so restart is cheap; send-keys delivery is documented-flaky (printed-prompt trap); a pipeline nudge channel would fork the cross-adapter contract | S:85 R:80 A:90 D:85 |
| 2 | Certain | `fab dispatch restart` relaunches from the persisted `{stage}-prompt.md` (no stdin), refuses while running, and reuses `start`'s prologue + l9ng mode ladder (mode re-derived from the current environment) | Grounded in existing machinery: prompt persistence, last-attempt-only overwrite, and the refuse-if-running check already define every semantic; restart only adds the prompt-file source | S:80 R:85 A:90 D:85 |
| 3 | Confident | Auto-restart policy: exactly ONE automatic restart per stage dispatch, on `orphaned`; then escalate (surface evidence + gated `rk notify`) and stop per the stage's failure path | User asked for the affordance and endorsed boundedness; the specific budget of 1 is the assistant's default — mirrors "bounded recovery, not liveness at all costs" | S:70 R:85 A:80 D:70 |
| 4 | Confident | `failed` gets no automatic restart; the orchestrator MAY spend the same single budget on a clearly-transient failure (provider 5xx in the log tail); `failed (no-result)` always escalates | Deterministic failures must not loop; the orchestrator is an agent and can read logs — encoding judgment beats encoding a rule that either loops or under-recovers | S:60 R:80 A:80 D:65 |
| 5 | Confident | Peek-on-suspicion: read-only peek every 10th result-less poll (~5 min), three-way classification (progressing / parked-recoverable → kill+restart in budget / needs-human → notify, never kill, never type) | Cadence and taxonomy are the assistant's defaults over the agreed shape (peek read-only, escalate-don't-answer); 10 polls balances detection latency against capture noise | S:60 R:85 A:75 D:65 |
| 6 | Confident | Restart budget tracked in orchestrator context only — no on-disk attempt counter | Preserves last-attempt-only (no history is a shipped design decision); worst case after context loss is one extra restart, which is benign | S:55 R:80 A:80 D:70 |
| 7 | Confident | `restart` output/record is byte-shaped like `start`'s — no new state strings, no `restarted:` marker | A restart IS a fresh attempt under last-attempt-only; a marker would be a second source of truth the states don't need | S:60 R:85 A:85 D:75 |
| 8 | Certain | Activate only after PR #525 (l9ng) merges | Edits the same code/doc surfaces; sequencing stated in Origin | S:85 R:80 A:90 D:90 |

8 assumptions (3 certain, 5 confident, 0 tentative, 0 unresolved).
