# Intake: Retire Operator /loop — Cron Entry as Sole Cadence

**Change**: 260909-6t88-retire-loop-cron-cadence
**Created**: 2026-09-09

## Origin

> Wave 4 change C10 "Replacement posture" from run-kit's `fab/plans/sahil/26-09-06-cron-clock-plan.md` (this is the cross-repo fab-kit item in that plan — read its Wave 4 row and the 2026-09-09 off-plan-narrowing note, both in the run-kit repo checkout, for exact scope). Scope: fab-operator.md §4 rewrite: retire /loop + Adaptive cadence in favor of the cron entry's union predicate; ready-line copy; pulse-plan supersession cleanup. Per the narrowing note, draft this provider-neutral: /loop is a Claude Code feature, so for a non-Claude operator the cron entry is not a replacement but the only clock — document the cron entry as the sole cadence for every provider, retiring the loop only for Claude.

Invoked via `/fab-new` (one-shot; the description carries the design authority pointers). Both source documents were read from the run-kit checkout at `/home/sahil/code/sahil87/run-kit`:

- **The plan's Wave 4 row (C10)**: "`fab-operator.md` §4 rewrite: retire `/loop` + Adaptive cadence in favor of the cron entry's union predicate; ready-line copy; pulse-plan supersession cleanup. Only after C4's gate has held in daily use." Sequencing: "C10 last, gated on lived experience, not tests."
- **The 2026-09-09 off-plan-narrowing note**: "C10 should be drafted provider-neutral: `/loop` is a Claude Code feature, so for a non-Claude operator the cron entry is not a replacement but the only clock — the rewrite retires the loop for Claude and documents the cron entry as the sole cadence for every provider." Substrate that makes this true: run-kit #878 (agent-neutral `@rk_pane_agent_state` hooks for Codex/Gemini/Copilot/Kimi/OpenCode/Antigravity — backoff anchor, `wake_on`, and `when-idle` gating work for non-Claude operators) and #876 (provider-aware skill-invocation prefix — the cron respawner renders `/fab-operator` through `RenderSkillRef`).
- **Design authority**: run-kit `docs/specs/cron.md` — the union predicate, the seeded operator-tick entry shape, suppress guards, and the P3 replacement-posture paragraph ("the operator stops running `/loop`, and the entry's union predicate (`backoff` + `wake_on`) replaces §4 Adaptive cadence, eliminating the dual-clock arrangement and the loop-death incident class entirely").
- **Plan status**: Waves 1–3 merged (C1 #855 … C7 #864). The Wave 2 GATE's four checks have no recorded run; the plan asks for a recorded pass or explicit waiver before C10, whose own gate is "the backstop holding in daily use."

## Why

1. **The pain point**: the operator's cadence lives in a `/loop` inside its own session — monitoring liveness depends on the monitored session (the incident class that motivated the whole cron plan: a Ctrl-C killed the loop silently and three monitored workers ran unwatched for hours). Since run-kit Waves 1–3 merged, a durable, session-independent clock exists: `rk operator` idempotently seeds a pinned operator-tick cron entry (backoff `60s→30m` anchored on operator idle, `wake_on: agent-state-change` debounced, `suppress_while: [operator-loop-fresh, nothing-tracked]`, `target: role=operator`, `payload: "operator tick"`, `if_absent: respawn`). Today the two clocks coexist — the `operator-loop-fresh` guard keeps the cron silent while the loop is healthy. That dual-clock arrangement was explicitly transitional.
2. **If we don't finish it**: the skill keeps teaching a Claude-only clock as primary. A non-Claude operator (codex/gemini/etc.) cannot run `/loop` at all — for it the skill's §4 is not merely stale but *wrong*: the cron entry is its only clock. And every Claude operator keeps paying the loop-death risk plus the cognitive overhead of two clocks, adaptive-cadence re-establishment, and the one-loop invariant.
3. **Why this approach**: collapse to one clock, owned by rk. The union predicate strictly dominates Adaptive cadence: the `90s` tightened heartbeat existed to bound waiting-agent pickup latency, and `wake_on: agent-state-change` (10s debounce) beats it — a `waiting` flip fires a tick within seconds, not within a 90s poll. The `nothing-tracked` suppress guard replaces the skill's own stop-the-loop bookkeeping. The backoff ladder (60s→30m anchored on operator idle) replaces the fixed `3m`. Provider-neutral framing per the narrowing note: document the entry as the sole cadence for **every** provider; the `/loop` retirement is the Claude-specific consequence, not the headline.

## What Changes

All skill edits target the canonical sources under `src/kit/skills/` (never the deployed copies — `fab/project/context.md`).

### 1. `fab-operator.md` §4 rewrite ("The Loop" → the cron clock)

Rewrite §4 so cadence is **not owned by the skill at all** — it is rk-owned substrate the skill *documents and verifies*, exactly like `@rk_pane_agent_state`:

- **Retire**: the `/loop` start/stop lifecycle paragraph (the four run-conditions as *loop* conditions), the entire **Adaptive cadence** block (`3m` normal / `90s` tightened, the relax-back rule, the one-loop invariant as cadence mechanics, the autopilot-composition cadence note), the **Loop Prompt** literals (`/loop 3m "operator tick"` / `/loop 90s "operator tick"`), and the dynamic-mode wakeup-prompt paragraph.
- **Replace with** a section documenting the operator-tick cron entry as the sole cadence for every provider: the entry's shape (backoff `min: 60s, max: 30m` anchored on operator idle with the anchor-join, `wake_on: agent-state-change` debounce `10s`, `suppress_while: [operator-loop-fresh, nothing-tracked]`, `target: role=operator`, `payload: "operator tick"`, `deliver: immediate`, `if_absent: respawn`, `pinned: true`), who owns it (`rk operator` seeds it idempotently; the skill never creates or mutates it — `rk cron` verbs are the user's), and how the union predicate covers the old behaviors: `wake_on` ≻ the 90s tightened cadence; `nothing-tracked` ≻ the stop-when-empty rule; backoff ≻ the fixed 3m; `if_absent: respawn` + role-target fire-time resolution ≻ session-bound liveness.
- **Keep, re-grounded**: the bare `operator tick` payload contract and its token-economics rationale (a slash-command payload macro-expands ~21k tokens per firing — this rule now binds the cron entry's `payload` and any manually typed tick, not a loop prompt); **Post-Compaction Reload** (strengthened: ticks keep arriving from the cron regardless of session health, so the reload trigger — a tick arrives with no Tick Behavior in context — is now guaranteed to fire eventually; the procedure drops its "loop re-establishment" step); the Operator State File section (unchanged — `last_tick_at`/`tick_count` written by `tick-start` are what the `operator-loop-fresh` guard and the dashboard staleness read).
- **Claude-only degraded fallback** (one short paragraph, not a co-equal clock): when the cron entry cannot exist — rk absent, or an installed rk predating `rk cron` (probe: `command -v rk` + `rk cron list` failing) — a Claude Code operator MAY run `/loop 3m "operator tick"` as the explicit fallback clock, bare-prompt rule unchanged; a non-Claude operator has no automatic cadence in that state and the ready line says so. Never both: the fallback runs only when the entry does not exist (the dual-clock arrangement is what this change retires). <!-- assumed: keep /loop as rk-absent Claude fallback rather than hard-requiring rk — matches the skill-wide fail-silent rk-optional doctrine (_preamble § Run-Kit); hard-require rejected as contradicting the existing rk-absent degradation posture throughout the skill -->
- **Tick Behavior step 7** ("Loop lifecycle") is deleted/replaced: no per-tick clock management remains. Quiescence is the entry's `nothing-tracked` guard; cadence adaptation is the backoff + `wake_on`. Keep a one-line lifecycle note where step 7 was (the tick list may renumber) so the old cross-references (§6 merge-sequence run-condition, §2 Init) have a target.
- **Idle Message**: `fab operator time --interval {interval}` can no longer predict the next tick (the backoff rung is rk-derived, not skill-known). Simplify the idle message to current time + last-tick time (from the tick header), dropping the computed `next:`; exact copy decided at apply. <!-- assumed: drop the next-tick prediction rather than shell out to `rk cron list --json` per idle render — cheaper, and next-fire is a UI concern (C5–C7 surfaces it) -->

### 2. Ready-line copy (§2 Init steps 4–5)

Step 4 (start the loop) becomes a **clock verification**: gated on `command -v rk`, check the operator-tick entry exists (`rk cron list`, fail-silent). Step 5's ready line replaces the loop literal with clock status — target shape (exact copy at apply):

```
Operator ready. Clock: rk cron "operator tick" (backoff 60s–30m, wakes on agent-state-change)
Operator ready. Clock: none — run `rk operator` to seed the cron entry, or (Claude Code only) start the fallback: /loop 3m "operator tick"
```

The copy-never-compose rule survives: whatever literal the degraded line carries is the one the agent copies.

### 3. Retirement sweep across `fab-operator.md`

Every cadence/loop claim outside §4, updated to the cron posture:

- §1 header prose: "monitors progress via `/loop`. The loop is the heart of the operator" → cron-delivered ticks are the heartbeat.
- §1 principles table "Survive compaction" row: reload wording keeps working, minus loop re-establishment.
- §5: "Strategic handling MUST NOT block the loop" and the tightened-cadence trigger sentence (the `waiting` state now feeds `wake_on`, not a 90s interval); "keeps ticking" phrasing stays valid.
- §6 Auto-Merge Choreography: "a merge sequence in progress is by itself a loop run-condition (§4)" — rewrite: an in-progress merge sequence still needs ticks, but the seeded entry's `nothing-tracked` guard covers only `monitored`/`watches`/`autopilot`; see Open Questions (run-kit coordination) — the skill text flags this as the one condition the guard does not yet know.
- §8 Settings: remove the `Loop interval | 3m` and `Waiting/menu heartbeat | 90s` rows (cadence tuning is `rk cron` territory now); keep the rest.
- §9 Key Properties: the `Uses /loop?` row becomes a `Cadence` row — rk cron operator-tick entry (union predicate), Claude-only `/loop` fallback when the entry cannot exist, bare `operator tick` payload rule.

### 4. `_cli-external.md` § /loop shrink

The section (and the frontmatter description + Contents entry) currently presents `/loop` as the operator's clock. Shrink it to a fallback-scoped note: invocation syntax, the one-loop-at-a-time rule, self-paced mode's bare-prompt rule, and a pointer to `fab-operator.md` §4's fallback paragraph as the sole policy owner. <!-- assumed: shrink rather than delete — the fallback still needs its syntax documented somewhere operator skills load -->

### 5. Pulse-plan supersession cleanup

`fab/plans/sahil/26-09-03-operator-pulse-plan.md`: add a supersession banner at the top. run-kit's cron plan already superseded the Clock B half (the `fab operator pulse` verb family — never built); this change retires Clock A (the `/loop`), so the whole two-clock design is superseded by run-kit's `docs/specs/cron.md` + `fab/plans/sahil/26-09-06-cron-clock-plan.md`. Note what carried forward: the OS-timer fallback and reboot analysis (into the cron plan), the guarded-delivery/TOCTOU mechanics (into run-kit's injection engine), the sidecar precedent (the cron state file location). Grep confirms no other fab-kit file (src/, docs/) references the pulse — the banner is the whole cleanup.

### 6. Spec touch-up

`docs/specs/skills.md` § `/fab-operator`: the flow-skeleton line "runs a continuous /loop cycle" and the Tools line's `Skill (/loop)` entry update to the cron-delivered tick posture. `docs/specs/operator.md`'s version-history row ("v4 | `/loop`-driven monitoring…") is historical and stays. Human-authored spec prose edited deliberately as part of the change (Constitution VI permits authored edits; only auto-generation is prohibited).

## Affected Memory

- `runtime/operator`: (modify) Rewrite the `/loop` lifecycle block (the mmmt/ioku/4q3l paragraph) to present truth — cron entry as sole cadence, union predicate, Claude-only fallback, reload procedure minus loop re-establishment; update the §8 settings mirror rows; add a Design Decision (loop retirement in favor of the rk cron union predicate; provider-neutral rationale; rejected: dual-clock retention, hard-requiring rk); reconcile the existing mmmt/ioku DD prose where it states loop facts in the present tense.

(Indexes regenerated via `fab docs-index` at hydrate.)

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (§1, §2 Init, §4 wholesale, §5 wording, §6 one sentence, §8 two rows, §9 one row), `src/kit/skills/_cli-external.md` (frontmatter description, Contents, § /loop), `fab/plans/sahil/26-09-03-operator-pulse-plan.md` (banner), `docs/specs/skills.md` (two lines), `docs/memory/runtime/operator.md` + regenerated indexes (hydrate).
- **No Go changes**: `fab operator time`, `tick-start`, and the state-file verbs are untouched (the skill stops passing `--interval`-derived next-tick copy but the command remains for other uses); `rk cron` is run-kit's, already shipped.
- **Cross-repo dependency**: purely consuming — Waves 1–3 are merged in run-kit; nothing here blocks on unmerged run-kit work. One coordination gap flagged (merge-sequence run-condition, below).
- **Behavior contract**: the operator's documented clock changes for every provider; Claude operators stop being told to run `/loop`. Existing running operators pick this up on their next `/fab-operator` reload; the seeded entry's `operator-loop-fresh` guard makes the transition safe in both directions (a not-yet-reloaded operator still running its loop keeps the cron suppressed).

## Open Questions

- **Wave 2 GATE evidence**: the plan requires a recorded pass (or explicit waiver) of the C4 gate's four checks before C10, and C10's own gate is "the backstop holding in daily use." Drafting and implementing the skill rewrite now is user-sanctioned (this invocation); the gate evidence should be recorded — or explicitly waived — before this change ships/merges. Track as a ship-time acceptance item.
- **`nothing-tracked` vs. the merge-sequence run-condition**: the skill's old loop ran on four conditions (monitored, autopilot, watches, in-progress merge sequence); the seeded entry's `nothing-tracked` guard covers only the first three (run-kit spec: "`monitored`, `watches`, and `autopilot` all empty"). A merge sequence in progress with everything else empty would get its ticks suppressed. Needs a run-kit follow-up (extend the guard to open merge-sequence coordination notes, or read them under `monitored`-adjacent state); this change documents the gap in the rewritten §6 sentence rather than papering over it.
- **Idle-message next-fire**: is dropping the predicted `next:` acceptable, or should the idle message read next-fire from `rk cron list --json` when available? (Assumed: drop — see marker in What Changes.)

## Clarifications

### Session 2026-09-09

| Q | A |
|---|---|
| How should C10 handle the `nothing-tracked`-guard/merge-sequence run-condition gap? | Document the gap in the rewritten §6 sentence and flag a run-kit follow-up to extend the guard — no fab-side workaround, no blocking on run-kit (user confirmed the recommendation; row 10 re-graded). |

### Session 2026-09-09 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 3 | Confirmed | — |
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 9 | Confirmed | — |
| 12 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill-prose-only change; no Go edits in fab-kit, no run-kit edits | The plan row is a `fab-operator.md` rewrite; all cron machinery shipped in run-kit Waves 1–3 | S:85 R:80 A:90 D:85 |
| 2 | Certain | Provider-neutral framing: the cron entry is documented as the sole cadence for every provider; `/loop` retirement is the Claude-specific consequence | Verbatim from the 2026-09-09 narrowing note, backed by run-kit #878/#876 substrate | S:95 R:85 A:90 D:90 |
| 3 | Confident | Keep `/loop 3m "operator tick"` as a Claude-only degraded fallback used only when the cron entry cannot exist (rk absent / pre-cron rk); never alongside a live entry | Clarified — user confirmed | S:95 R:70 A:65 D:55 |
| 4 | Confident | Ready line reports clock status from a gated `rk cron list` check, replacing the loop literal; degraded form carries the seed instruction + Claude fallback literal | Clarified — user confirmed; exact copy decided at apply | S:95 R:85 A:70 D:60 |
| 5 | Confident | Tick step 7 (Loop lifecycle) is deleted; quiescence and cadence adaptation are owned by the entry's guards/predicate; a one-line note keeps cross-references anchored | Clarified — user confirmed | S:95 R:80 A:75 D:65 |
| 6 | Confident | Idle message drops the computed next-tick (`fab operator time --interval` can't know the backoff rung); shows current + last-tick time | Clarified — user confirmed; next-fire is a run-kit UI concern (C5–C7) | S:95 R:85 A:55 D:45 |
| 7 | Confident | `_cli-external.md` § /loop shrinks to a fallback-scoped note (syntax + one-loop rule + policy pointer) rather than deletion | Clarified — user confirmed | S:95 R:80 A:60 D:50 |
| 8 | Certain | Pulse-plan cleanup = a supersession banner on `26-09-03-operator-pulse-plan.md` only; no other fab-kit references exist | `grep -ri pulse src/ docs/` returns nothing; the cron plan header already declares the partial supersession this completes | S:70 R:90 A:85 D:75 |
| 9 | Confident | Wave-2-gate evidence is a ship-time acceptance item, not an intake blocker | Clarified — user confirmed | S:95 R:75 A:60 D:65 |
| 10 | Confident | The merge-sequence/`nothing-tracked` gap is documented in the rewritten §6 sentence and flagged as a run-kit follow-up — not worked around in fab-kit | Clarified — user confirmed (document + flag run-kit follow-up; no fab-side workaround, no blocking on run-kit) | S:95 R:60 A:50 D:40 |
| 11 | Certain | Affected memory is `runtime/operator` (modify) only | The loop lifecycle, settings mirror, and DDs all live there; no other memory file states cadence facts | S:80 R:85 A:90 D:85 |
| 12 | Confident | `docs/specs/skills.md`'s fab-operator flow skeleton is edited in this change; `docs/specs/operator.md` v-history stays | Clarified — user confirmed | S:95 R:80 A:70 D:65 |

12 assumptions (4 certain, 8 confident, 0 tentative, 0 unresolved).
