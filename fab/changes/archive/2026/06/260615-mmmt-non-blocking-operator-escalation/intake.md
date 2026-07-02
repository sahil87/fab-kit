# Intake: Non-Blocking Operator Escalation

**Change**: 260615-mmmt-non-blocking-operator-escalation
**Created**: 2026-06-15

## Origin

> Backlog `[mmmt]` (from the `idea` backlog, not `fab/backlog.md`): "Non-blocking operator
> question-handling (feat, src/kit/skills/fab-operator/): Today a managed agent's interactive menu
> can freeze the ENTIRE operator loop. … Fix: make escalation NON-BLOCKING — operator posts the
> question out-of-band and KEEPS TICKING … tighten heartbeat … shorten the strategic auto-default …
> auto-pick-and-notify … Notification channel (PRIMARY = ntfy.sh)."

**Interaction mode**: conversational. Invoked as `/fab-new mmmt`. The literal arg `mmmt` was not in
`fab/backlog.md` (the Step-0 check looked there and missed); the user supplied the entry from the
`idea --main ls` backlog. The change ID `mmmt` was preserved via `fab change new --change-id mmmt`.

**Key decisions reached in this conversation** (each encoded as a graded assumption below):

- **Scope split (decisive).** The notification *channel* implicates **run-kit**, a separate repo.
  Investigation found run-kit has a PWA manifest (`app/frontend/public/manifest.json`), an SSE
  channel (`app/backend/api/sse.go`, route `GET /api/sessions/stream`), and a Go/chi server — but
  **no service worker, no Web Push/VAPID, no `rk notify` command**. "run-kit push" is therefore
  net-new feature work. The user chose to **split**: `mmmt` (fab-kit) does the operator-side logic
  now with the notification *send* abstracted behind a single shell command (ntfy.sh as the working
  default today); run-kit Web Push is a **separate change in the run-kit repo**, not a blocker.
- **Auto-default threshold: KEEP 30m.** The user explicitly chose *not* to touch the strategic
  idle auto-default value or its config surface in this change (it stays hardcoded at 30m). The
  backlog floated shortening it; that tuning is deferred to a separate change.
- **Auto-pick-and-notify (not notify-and-wait).** For strategic menus where the operator has a
  defensible recommendation, auto-pick immediately + notify + keep ticking (reversible at PR
  review). Reserve true parking only for menus with no defensible default.
- **Notification channel default = `rk notify` (run-kit Web Push)** — UPDATED 2026-06-16: run-kit
  Web Push shipped (backlog `[xd9r]`, released in `rk v2.3.2`), so the intended eventual channel is
  now the real default. `rk notify <message> [--title …]` delivers a real background mobile/desktop
  Web Push to every subscribed device, is fail-silent by contract (exits 0 / prints nothing on any
  error, so it can never stall the loop), and routes through infra the user already runs (no
  third-party service, no world-readable topic). It is the user's stated preference.
  **Documented alternatives** (when `rk` is absent — the skill MUST check `command -v rk` first per
  `_preamble.md` § Run-Kit, and fall back): **ntfy.sh** with a required high-entropy topic (research
  confirmed it as #1 for curl-from-shell / headless / aggregator; caveat: public topics are
  world-readable — the topic name is the only secret); Discord webhook (no-account, searchable
  history); the `PushNotification` harness tool (zero-infra, personal push not a shared feed); Slack
  MCP (searchable, but §7 warns interactively-authed MCP may be absent headless).

## Why

**The problem (pain point).** The operator's coordination loop is a single-threaded `/loop 3m`
heartbeat (`fab-operator.md` §4). When a managed agent surfaces a *strategic* menu the operator
cannot auto-answer (§5 rule 4), the operator "escalates to user." In practice escalation parks the
operator's turn waiting for a human answer, so **no tick fires meanwhile** — and because one loop
serves *every* monitored change across every repo and session (§8, one operator per server), a
single strategic question on one change **freezes the entire queue**. Every other change stops
advancing until the human returns. This was observed live during FKF autopilot: change `bmzo` hit
one Unresolved SRAD decision and the operator parked the whole queue.

**The consequence of not fixing it.** The operator's entire value proposition is "take work off the
user's hands" and "automate the routine" (§1). A coordination layer that any one agent can freeze is
fragile precisely when it is most needed — running many agents unattended. Detection latency
compounds it: the operator only notices a menu on its next `/loop` tick (up to ~3m, longer if the
heartbeat was widened), so even after the user answers elsewhere, progress can stall.

**Why this approach over alternatives.** Making escalation **non-blocking** (post out-of-band, keep
ticking) is the minimal structural fix: it removes the single-question-freezes-everything failure
mode without changing the operator's single-loop architecture. We pair it with (a) an **adaptive
heartbeat** that tightens to 60–90s whenever any monitored agent is sitting on a menu — bounding
worst-case detection without paying that cadence cost when idle — and (b) **auto-pick-and-notify**
for strategic menus with a defensible recommendation, so the operator keeps the queue moving and the
human reviews the choice at PR-review time (the existing safety net per §1). The notification
*send* is abstracted behind one shell command so the channel can evolve (ntfy.sh today → `rk notify`
later) without touching the escalation logic.

## What Changes

All changes are to **`src/kit/skills/fab-operator.md`** (the authoritative skill) with a
**synchronous SPEC mirror** to **`docs/specs/skills/SPEC-fab-operator.md`** (Constitution rule).
The touched sections are §4 (The Loop — tick behavior + adaptive cadence), §5 (Auto-Nudge —
escalation path), and §8 (Configuration — settings). Naming note: the backlog wrote the path as
`src/kit/skills/fab-operator/` (a directory) but the skill is a single flat file
`src/kit/skills/fab-operator.md`.

### 1. Non-blocking strategic escalation (§5 Auto-Nudge)

Today, §5 rule 4 "Strategic → escalate to user" parks the loop. Change escalation so it **never ends
the operator's turn**:

- When the answer model (§5) decides a menu is Strategic, the operator **does not block**. It:
  1. Either **auto-picks** its recommended option and sends it (see change 3 below), OR — only when
     there is no defensible default — **leaves the prompt open** for the user.
  2. **Sends a notification out-of-band** via the notify command (change 4) describing the change,
     the question summary, and (if auto-picked) the option taken.
  3. **Continues ticking** — the loop proceeds to the next monitored change in the same tick. Other
     changes keep advancing; a strategic question on one change no longer freezes the queue.
- The user answers asynchronously (either by responding to the notification's guidance, or by typing
  directly into the agent's pane). The operator **picks up the resolution on a later tick** via its
  normal re-capture/re-detection (§5 "Sending Auto-Answers" already re-captures before any send).
- The §5 *Idle Auto-Default* (30m) is **unchanged** — it remains the watchdog for a left-open
  strategic prompt (the no-defensible-default case). Its threshold, scope exclusion (rule-6
  escalations excluded), and answer-selection priority are untouched.

**Concretely**, the §5 Logging block gains/retains three line shapes:
```
{change}: auto-answered '{summary}' → {answer}                              # routine (existing)
{change}: auto-picked strategic '{summary}' → {answer} · notified           # NEW — auto-pick-and-notify
{change}: strategic '{summary}' left open · notified. Please respond.       # NEW — no defensible default
{change}: auto-defaulted after 30m idle: '{summary}' → {answer}             # existing (unchanged)
```

### 2. Adaptive heartbeat on interactive menu (§4 The Loop)

Today the loop is a fixed `/loop 3m` (§4). Add **adaptive cadence**: whenever **any** monitored
agent is detected sitting on an interactive menu (input-waiting, per §5 Question Detection), the
operator tightens the heartbeat to a short interval to bound worst-case detection/pickup latency;
when no monitored agent is menu-waiting, it relaxes back to the normal `3m`.

- **Tightened interval: 90s** (within the backlog's stated 60–90s range; chosen as the calmer end to
  reduce capture-pane churn while still giving sub-2-minute pickup).
- The "one loop at a time" invariant (`_cli-external.md` §/loop) is preserved — adapting cadence
  means re-establishing the single loop at the new interval (e.g. restart `/loop 90s "operator
  tick"`), not running two loops.
- Autopilot's own cadence override (default 2m, per `_cli-external.md`) composes unchanged — when
  autopilot is driving, its cadence governs; the menu-tightening applies to the monitoring loop.
- The §4 *Idle Message* (`Waiting for next tick … next tick: HH:MM`) reflects whatever interval is
  currently active (it already takes `--interval {interval}`), so a tightened cadence shows the
  nearer next-tick time.

### 3. Auto-pick-and-notify for strategic menus (§5 Answer Model)

Refine §5 rule 4's Strategic branch. Today: "Strategic → escalate to user" (park). New behavior:

- **Strategic + defensible recommendation** → the operator **auto-picks** its recommended option
  (LLM judgment over the terminal capture — same signals §5 already lists: option text,
  distinctness, surrounding context, reversibility), sends it, fires a notification, and keeps
  ticking. The PR-review stage is the reversal point (§1 "The PR review stage is the safety net").
- **Strategic + no defensible default** → the operator leaves the prompt open, notifies, and keeps
  ticking. The 30m idle auto-default remains the backstop for these.
- The Routine branch (`→ 1`) is unchanged. Rule 6 ("cannot determine keystrokes") is unchanged and
  still excluded from any auto-pick/auto-default (sending a guess would emit nonsense into the pane).

### 4. Notification send, abstracted behind one command (§5 + §8)

The notification *mechanism* is a single shell send the operator runs out-of-band. **Default channel
= `rk notify` (run-kit Web Push)** — UPDATED 2026-06-16, now that run-kit Web Push shipped (`rk
v2.3.2`, backlog `[xd9r]`):

```sh
command -v rk >/dev/null 2>&1 && rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"
```

- **Why `rk notify` is the default.** It delivers a real background mobile/desktop Web Push to every
  subscribed device even when the app is closed; it is **fail-silent by contract** (exits 0 / prints
  nothing on any error — server unreachable, no subscriptions — so it can never stall the operator
  loop); it routes through infra the user already runs (no third-party service, no world-readable
  topic secret); and a single user's subscriptions form one feed across every operator on the box,
  satisfying the aggregator goal without a shared-secret topic. `rk context` surfaces it as a
  capability, so the operator discovers it at runtime.
- **`rk`-detection gate.** Per `_preamble.md` § Run-Kit Reference, the operator MUST check `command
  -v rk` before using `rk notify`. When `rk` is absent (operator running where run-kit isn't
  installed), fall back to the first available documented alternative below. The composite send is
  itself fail-silent: a notification that cannot be delivered logs one line and the loop keeps
  ticking.
- **Documented alternatives** (named so the channel can be swapped conversationally when `rk` is
  absent or the user prefers another sink):
  - **ntfy.sh** — `curl -d "{change}: {summary} ({repo})" ntfy.sh/<high-entropy-topic>` — no
    account, curl-from-shell, cross-repo aggregator, mobile push. **High-entropy topic REQUIRED**:
    public topics are world-readable to anyone who knows the name (the topic name is the only
    secret), so use a long random topic (e.g. `op-9f3a2c7e-strat`) and never put secrets in bodies.
    The strongest no-run-kit fallback.
  - **Discord webhook** — `curl -H 'Content-Type: application/json' -d '{"content":"…"}' <webhook>` —
    no account, one webhook = one channel, indefinite searchable history, mobile push.
  - **`PushNotification`** (built-in Claude Code harness tool) — zero infra, no topic secret to
    leak, headless-safe; but a *personal* push to the user's Claude apps, **not** a shared
    searchable feed. Good "just ping me" fallback.
  - **Slack MCP** (`mcp__claude_ai_Slack__slack_send_message`) — searchable channel feed, mobile
    push; caveat per §7: interactively-authed MCP may be **absent in headless/cron** runs, so it
    cannot be a headless default.
- All notify sends **fail silently** — a notification that cannot be sent (`rk`/run-kit server
  unreachable, channel down, no subscriptions, curl/tool missing) MUST NOT crash or stall the loop;
  the operator logs one line and keeps ticking. `rk notify` is already fail-silent by contract; the
  fallback path must match it. This mirrors the existing rk "fail silently" discipline
  (`_preamble.md` § Run-Kit Reference).

### 5. Settings (§8 Configuration)

Add to the §8 settings table, session-scoped (resets on `/clear`, like the existing rows), set via
natural language:

| Setting | Default | Override via natural language |
|---------|---------|------------------------------|
| Menu-detected heartbeat | 90s | "tighten to {N}s when an agent is on a menu" |
| Notify channel | `rk` (run-kit Web Push; auto-fallback when `rk` absent) | "notify via ntfy topic {topic}" / "notify via discord {url}" / "notify via push" |

The §4 operator state-file schema is **unchanged** (these are session settings, consistent with the
current loop-interval / stuck-threshold rows). The **strategic auto-default threshold stays
hardcoded at 30m** — no new setting for it (explicitly out of scope this change).

## Affected Memory

- `runtime/operator.md`: (modify) The operator runtime memory documents loop/tick behavior and the
  status-frame design history. Add the non-blocking-escalation model, adaptive heartbeat, and
  auto-pick-and-notify decision rationale. (Confirm exact path against `docs/memory/runtime/`
  during hydrate — SPEC §4 references `runtime/operator.md` for design history.)

<!-- assumed: only runtime/operator.md is affected — the change is operator-skill-internal (loop
     cadence + escalation + a notify command), touching no other domain's spec-level behavior. Memory
     impact is confirmed at hydrate against the live docs/memory/ tree. -->

## Impact

- **Authoritative skill**: `src/kit/skills/fab-operator.md` — §4 (The Loop / Tick Behavior / Idle
  Message), §5 (Auto-Nudge: Answer Model, escalation path, Logging), §8 (Configuration: Settings).
- **SPEC mirror (synchronous, Constitution rule)**: `docs/specs/skills/SPEC-fab-operator.md` —
  matching edits to §4/§5/§8 summaries (Section Structure, Auto-Nudge, Configuration blocks).
- **No Go code changes in fab-kit.** This is a skill-behavior change. No new `fab` CLI verbs, no
  `.status.yaml` schema change, no operator-state-file schema change. The notify send is an `rk
  notify` call (default) or a `curl`/harness-tool fallback — both run via Bash, which the operator
  already has. The `rk notify` capability itself was delivered in run-kit (`rk v2.3.2`), not here.
- **No new dependencies in fab-kit.** `rk notify` is an already-installed CLI (run-kit `v2.3.2`);
  ntfy.sh/Discord are reached via `curl`; the rest are existing tools (`PushNotification` harness
  tool, Slack MCP).
- **Cross-repo dependency RESOLVED (2026-06-16):** the `rk notify` channel (run-kit Web Push,
  backlog `[xd9r]`) **shipped and is released** in `rk v2.3.2`. `mmmt` now defaults to it, with
  `command -v rk` gating the call and the documented alternatives as fallback. `mmmt` was never
  blocked on it, and is not now — the dependency is satisfied.
- **Distribution**: changes ship via `fab sync` deploying the updated skill to `.claude/skills/`.

## Open Questions

- Whether the menu-detected heartbeat should also shorten *autopilot's* 2m cadence, or only the
  monitoring loop's 3m (intake assumes monitoring-loop-only; autopilot cadence composes unchanged).
- Fallback selection order when `rk` is absent — intake assumes "first available documented
  alternative"; whether to make the fallback channel an explicit session setting vs. auto-detect is
  left to apply (low-stakes, reversible).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edits land in `src/kit/skills/fab-operator.md` (single flat file) + synchronous SPEC mirror `docs/specs/skills/SPEC-fab-operator.md`; the backlog's `…/fab-operator/` directory path is wrong | Verified on disk: skill is a flat `.md`; SPEC mirror exists; Constitution mandates synchronous SPEC mirrors | S:90 R:85 A:95 D:95 |
| 2 | Certain | Strategic auto-default stays hardcoded at 30m — no value change, no new config surface | User explicitly chose "Keep 30m, just non-blocking"; removes a design reversal from scope | S:95 R:80 A:95 D:95 |
| 3 | Certain | Non-blocking escalation = operator posts out-of-band + keeps ticking + picks up async answer on a later tick | Backlog's core ask; the operator's existing `/loop` + re-capture architecture already supports it; no new mechanism needed | S:90 R:75 A:90 D:90 |
| 4 | Confident | Strategic menus with a defensible recommendation are auto-picked-and-notified (not parked); reversible at PR review | User chose "Auto-pick + notify"; matches §1 "PR review is the safety net"; reuses §5 LLM-judgment signals | S:85 R:70 A:85 D:80 |
| 5 | Confident | Notification default = `rk notify` (run-kit Web Push), gated on `command -v rk`; ntfy.sh / Discord / `PushNotification` / Slack are the fallback alternatives. Channel abstracted behind one shell send | UPDATED 2026-06-16: run-kit Web Push shipped (`rk v2.3.2`), so the user-preferred channel is now the real default — real background push, fail-silent by contract, no third-party/world-readable-topic surface. ntfy.sh stays the strongest no-`rk` fallback (research #1; high-entropy topic mitigates world-readability) | S:90 R:80 A:90 D:80 |
| 6 | Certain | run-kit Web Push (`rk notify`, service-worker + VAPID) was built as a SEPARATE run-kit change (backlog `[xd9r]`) and is now RELEASED (`rk v2.3.2`); `mmmt` consumes it as the default channel | UPDATED 2026-06-16: the split decision held (user chose "fab-kit now, run-kit separate"); run-kit delivered + released the capability; verified live via `rk notify --help` and `rk context`. Dependency satisfied, not pending | S:95 R:80 A:95 D:90 |
| 7 | Confident | Adaptive heartbeat tightens to 90s when any monitored agent is on an interactive menu; relaxes to 3m otherwise; preserves one-loop invariant | Backlog stated 60–90s; 90s chosen as calmer end to limit capture churn; §/loop one-loop rule respected by re-establishing the single loop | S:80 R:80 A:80 D:70 |
| 8 | Confident | Documented alternative channels: Discord webhook, `PushNotification` harness tool, Slack MCP (with §7 headless caveat) | Research-backed; gives swap options without committing fab-kit to extra deps; Slack's headless-absence caveat is the skill's own §7 | S:75 R:85 A:85 D:75 |
| 9 | Confident | All notify sends fail silently — never crash/stall the loop on a send failure | Mirrors existing rk "fail silently" discipline (`_preamble.md` § Run-Kit); a coordination loop must not die on a notification error | S:80 R:85 A:90 D:85 |
| 10 | Confident | New §8 settings (menu-heartbeat, notify-channel) are session-scoped NL overrides; operator-state-file schema unchanged | Matches existing §8 rows (loop-interval, stuck-threshold); avoids expanding the deliberately-minimal state-file schema | S:80 R:80 A:85 D:80 |
| 11 | Tentative | Affected memory is `runtime/operator.md` only | Change is operator-skill-internal; SPEC §4 points design history there; confirmed at hydrate against live `docs/memory/` | S:65 R:75 A:60 D:65 |
| 12 | Tentative | Menu-detected heartbeat tightens only the monitoring loop, not autopilot's 2m cadence | Conservative default; autopilot has its own cadence model; flagged as an Open Question for review | S:55 R:70 A:60 D:55 |

12 assumptions (4 certain, 6 confident, 2 tentative, 0 unresolved). Run /fab-clarify to review.
