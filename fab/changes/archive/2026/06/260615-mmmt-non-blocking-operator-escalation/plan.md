# Plan: Non-Blocking Operator Escalation

**Change**: 260615-mmmt-non-blocking-operator-escalation
**Intake**: `intake.md`

## Requirements

<!-- This is a skill-behavior change. The two artifacts are the authoritative skill
     src/kit/skills/fab-operator.md and its synchronous SPEC mirror
     docs/specs/skills/SPEC-fab-operator.md (Constitution: "Changes to skill files
     MUST update the corresponding SPEC-*.md file"). Every behavioral edit in the
     skill has a matching summary edit in the SPEC. -->

### Operator: Non-Blocking Strategic Escalation

#### R1: Strategic escalation never ends the operator's turn
The operator's §5 Auto-Nudge Strategic path SHALL NOT block the loop. When the answer model classifies a menu as Strategic, the operator MUST take a non-blocking action (auto-pick or leave-open per R3), send a notification out-of-band (per R4), and continue ticking to the next monitored change in the same tick.

- **GIVEN** the operator's monitoring loop is ticking and several changes are monitored across repos
- **WHEN** one monitored agent surfaces a Strategic menu the operator cannot auto-answer routinely
- **THEN** the operator resolves that agent's prompt non-blockingly (auto-pick or leave-open), fires a notification, and proceeds to the next monitored change in the SAME tick — no `/loop` tick is skipped and no other change's advancement is frozen
- **AND** the user's eventual answer (typed into the pane or guided by the notification) is picked up on a later tick via the existing §5 re-capture/re-detection path — no new pickup mechanism is added

#### R2: The 30m Idle Auto-Default watchdog is unchanged
The §5 Idle Auto-Default (30 minutes, hardcoded) SHALL remain the watchdog for a left-open Strategic prompt (the no-defensible-default case). Its threshold, its rule-6 scope exclusion, and its answer-selection priority MUST be untouched by this change.

- **GIVEN** a Strategic prompt was left open because no defensible default existed (R3)
- **WHEN** the prompt stays idle for 30 minutes
- **THEN** the existing Idle Auto-Default fires exactly as before — same 30m threshold (still hardcoded, no new config surface), same answer-selection priority (stated default else `1`), same rule-6 exclusion
- **AND** auto-picked Strategic prompts (R3) are already resolved, so the watchdog has nothing to act on for them

### Operator: Auto-Pick-and-Notify for Strategic Menus

#### R3: Defensible-recommendation strategic menus are auto-picked; otherwise left open
Refine §5 answer-model rule 4's Strategic branch. When the operator has a defensible recommendation (LLM judgment over the terminal capture using the signals §5 already lists — option text, distinctness, surrounding context, reversibility), it SHALL auto-pick that option, send it, notify, and keep ticking. Only when there is no defensible default SHALL it leave the prompt open, notify, and keep ticking. The Routine branch (`→ 1`) and rule 6 ("cannot determine keystrokes", still excluded from any auto-pick/auto-default) are unchanged.

- **GIVEN** a Strategic menu with options whose tradeoffs the operator can defensibly rank from the capture
- **WHEN** the operator classifies it Strategic with a defensible recommendation
- **THEN** the operator auto-picks the recommended option, sends it (after the §5 re-capture-before-send guard), fires a notification naming the option taken, and continues ticking — the PR review stage (§1) is the reversal point
- **GIVEN** a Strategic menu with no defensible default
- **WHEN** the operator classifies it Strategic
- **THEN** the operator leaves the prompt open, fires a notification, continues ticking, and the 30m Idle Auto-Default (R2) remains the backstop
- **AND** rule 6 escalations are never auto-picked or auto-defaulted (sending a guess would emit nonsense into the pane)

### Operator: Notification Send Abstraction

#### R4: Notification send is one fail-silent shell command, default `rk notify`
The notification mechanism SHALL be a single out-of-band shell send. The default channel MUST be `rk notify` — a run-kit external contract (`rk notify <message> [--title string]`, run-kit Web Push, released in `rk v2.3.2`) — with the send gated on `command -v rk` and fail-silent per `_preamble.md` § Run-Kit Reference (which documents the gate and the fail-silent discipline, not the `notify` subcommand itself). When `rk` is absent, the operator MUST fall back to the first available documented alternative (ntfy.sh with a required high-entropy topic, Discord webhook, the `PushNotification` harness tool, or Slack MCP). All notify sends MUST fail silently — a send that cannot be delivered logs one line and the loop keeps ticking; it MUST NOT crash or stall the loop.

- **GIVEN** the operator needs to notify the user about a strategic question (auto-picked or left-open)
- **WHEN** `rk` is on PATH (`command -v rk` succeeds)
- **THEN** the operator sends `rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"` — fail-silent by `rk` contract
- **GIVEN** `rk` is NOT on PATH
- **WHEN** the operator needs to notify
- **THEN** it falls back to the first available documented alternative, and that fallback send is itself fail-silent (a missing curl/tool/server logs one line and the loop continues)
- **AND** the documented alternatives are named so the channel can be swapped conversationally: ntfy.sh (high-entropy topic REQUIRED — public topics are world-readable, never put secrets in bodies), Discord webhook, `PushNotification` harness tool, Slack MCP (with the inline headless-absence caveat — an interactively-authed MCP may be absent in headless/cron runs, so it cannot be a headless default)

### Operator: Adaptive Heartbeat on Interactive Menu

#### R5: Loop cadence tightens to 90s when any monitored agent is on a menu
The §4 loop SHALL adapt its cadence. Whenever any monitored agent is detected sitting on an interactive menu (input-waiting per §5 Question Detection), the operator MUST tighten the heartbeat to 90s; when no monitored agent is menu-waiting, it relaxes back to the normal 3m. The "one loop at a time" invariant (`_cli-external.md` § /loop) MUST be preserved — adapting cadence means re-establishing the single loop at the new interval, not running two loops. Autopilot's own cadence override (default 2m) composes unchanged. The §4 Idle Message reflects whatever interval is currently active.

- **GIVEN** the operator is ticking at the normal 3m cadence
- **WHEN** a tick detects any monitored agent input-waiting on an interactive menu
- **THEN** the operator re-establishes the single monitoring loop at 90s (e.g., restart `/loop 90s "operator tick"`) — never a second concurrent loop
- **GIVEN** the operator is ticking at the tightened 90s cadence
- **WHEN** a tick detects no monitored agent menu-waiting
- **THEN** the operator relaxes the single loop back to 3m
- **AND** the §4 Idle Message (`Waiting for next tick … next tick: HH:MM`) shows the nearer next-tick time, since it already takes `--interval {interval}`
- **AND** when autopilot is driving, its own cadence governs — the menu-tightening applies to the monitoring loop

### Operator: Settings

#### R6: New session-scoped §8 settings for menu-heartbeat and notify-channel
The §8 Settings table SHALL gain two session-scoped natural-language-override rows: a menu-detected-heartbeat row (default 90s) and a notify-channel row (default `rk`, auto-fallback when `rk` absent). These match the existing loop-interval / stuck-threshold rows. The operator-state-file schema MUST NOT change. The strategic auto-default threshold MUST stay hardcoded at 30m with no new setting.

- **GIVEN** the §8 Settings table with its existing two rows
- **WHEN** this change adds settings
- **THEN** two rows are added — `Menu-detected heartbeat` (default `90s`, override "tighten to {N}s when an agent is on a menu") and `Notify channel` (default `rk`, override "notify via ntfy topic {topic}" / "notify via discord {url}" / "notify via push") — both session-scoped (reset on `/clear`), and the operator-state-file schema (§4) is unchanged
- **AND** no row is added for the strategic auto-default threshold (explicitly out of scope — stays hardcoded at 30m)

### Operator: SPEC Mirror Synchrony

#### R7: SPEC mirror summaries match every skill edit
Per the Constitution, every behavioral edit to `src/kit/skills/fab-operator.md` SHALL have a matching summary edit in `docs/specs/skills/SPEC-fab-operator.md` (§4 Monitoring System, §5 Auto-Nudge, §8 Configuration summaries; and the Resolved Design Decisions list where a new decision is recorded).

- **GIVEN** the skill file is edited for R1–R6
- **WHEN** the change is complete
- **THEN** the SPEC mirror's §4/§5/§8 summaries reflect the non-blocking escalation, adaptive heartbeat, auto-pick-and-notify, notify-send abstraction, and the two new settings — the skill and SPEC agree on every behavioral point

### Non-Goals

- No Go code changes, no new `fab` CLI verbs, no `.status.yaml` schema change, no operator-state-file schema change — this is a skill-behavior change in two markdown files only.
- The strategic auto-default threshold is NOT changed and NO setting is added for it (stays hardcoded at 30m).
- run-kit Web Push itself (`rk notify`, service worker + VAPID) is NOT built here — it shipped separately in run-kit (`rk v2.3.2`, backlog `[xd9r]`); `mmmt` only consumes it as the default channel.
- The menu-detected heartbeat tightens only the monitoring loop, not autopilot's 2m cadence (intake Open Question; conservative default — flagged for review).

### Design Decisions

1. **Non-blocking escalation reuses the existing `/loop` + re-capture architecture**: post out-of-band + keep ticking + pick up the async answer on a later tick — *Why*: minimal structural fix that removes the single-question-freezes-everything failure mode without changing the single-loop architecture — *Rejected*: a second concurrent loop or a separate watcher thread (violates the one-loop invariant and adds infrastructure).
2. **Auto-pick-and-notify (not notify-and-wait) for defensible-recommendation menus**: auto-pick + notify + keep ticking, reversible at PR review — *Why*: matches §1 "PR review is the safety net"; keeps the queue moving — *Rejected*: parking every strategic menu (the freeze this change removes).
3. **`rk notify` default, gated on `command -v rk`, fail-silent**: abstract the send behind one shell command — *Why*: real background push, fail-silent by contract, routes through infra the user already runs, no world-readable topic — *Rejected*: hardcoding ntfy.sh as default (world-readable topic; now a fallback) and committing fab-kit to extra deps.
4. **90s tightened cadence (calmer end of the backlog's 60–90s range)**: — *Why*: sub-2-minute pickup while limiting capture-pane churn — *Rejected*: 60s (more churn for marginal latency gain).

## Tasks

### Phase 1: Skill edits — src/kit/skills/fab-operator.md

- [x] T001 Edit §4 The Loop / Tick Behavior / Idle Message in `src/kit/skills/fab-operator.md`: add adaptive cadence (tighten the single monitoring loop to 90s when any monitored agent is input-waiting on an interactive menu, relax to 3m otherwise; preserve the one-loop invariant by re-establishing the single loop at the new interval; autopilot's 2m cadence composes unchanged); note the Idle Message reflects the currently-active interval via its existing `--interval {interval}`. <!-- R5 -->
- [x] T002 Edit §5 Auto-Nudge in `src/kit/skills/fab-operator.md`: (a) Answer Model rule 4 Strategic branch → split into "defensible recommendation → auto-pick + send + notify + keep ticking" vs "no defensible default → leave open + notify + keep ticking"; keep Routine (`→ 1`) and rule 6 unchanged; (b) rework the escalation path so it never ends the operator's turn (post out-of-band, keep ticking, pick up async answer on a later tick via existing re-capture); (c) state the §5 Idle Auto-Default stays the backstop for left-open prompts, unchanged; (d) add the notification-send command block (default `rk notify`, `command -v rk` gate, documented fallbacks: ntfy.sh high-entropy / Discord / PushNotification / Slack MCP; all fail-silent); (e) update the Logging block to add the two new line shapes (`auto-picked strategic … · notified`, `strategic … left open · notified. Please respond.`) alongside the existing two. <!-- R1 --> <!-- R3 --> <!-- R4 -->
- [x] T003 Edit §8 Configuration Settings table in `src/kit/skills/fab-operator.md`: add the `Menu-detected heartbeat` (90s) and `Notify channel` (`rk`, auto-fallback) rows; keep "session-scoped" note; do NOT add a strategic-auto-default-threshold row; confirm the §4 state-file schema note remains "unchanged". Also confirm the §9 `Uses /loop?` property remains accurate (3m default; adaptive note optional). <!-- R6 -->

### Phase 2: SPEC mirror — docs/specs/skills/SPEC-fab-operator.md

- [x] T004 Mirror the skill edits into `docs/specs/skills/SPEC-fab-operator.md`: (a) §4 Monitoring System summary — adaptive cadence (90s on menu-waiting, relax to 3m, one-loop invariant preserved); (b) §5 Auto-Nudge summary + the dedicated Auto-Nudge section — non-blocking strategic escalation, auto-pick-and-notify vs leave-open, notify-send abstraction (default `rk notify`, gate, fallbacks, fail-silent), and the new log line shapes; (c) §8 Configuration summary — the two new session-scoped settings + auto-default-stays-30m note; (d) add a Resolved Design Decisions entry recording the non-blocking-escalation + auto-pick-and-notify + `rk notify`-default decisions for this change. Keep the SPEC's existing prose style and section structure. <!-- R7 -->

### Phase 3: Verification

- [x] T005 Sanity-check internal consistency: skill ↔ SPEC mirror agree on §4/§5/§8 behavior; the six §5 Logging bullets (5 answer-line shapes + a notify line) match between skill §5 Logging and SPEC §5; the 30m auto-default is described as unchanged in both; no operator-state-file schema field was added; no Go/CLI/status-schema change was introduced. <!-- R7 --> <!-- R2 -->

## Execution Order

- T001, T002, T003 edit disjoint sections of the same file and may proceed in any order, but are written sequentially to avoid edit-collision; do them before T004.
- T004 mirrors T001–T003 into the SPEC; run after the skill edits land.
- T005 verifies after all edits.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: §5's Strategic path in `fab-operator.md` no longer "escalates to user" by parking — it posts out-of-band, keeps ticking, and picks up the async answer on a later tick via the existing re-capture path.
- [ ] A-002 R2: The §5 Idle Auto-Default is present and unchanged — 30m hardcoded, rule-6 excluded, answer-priority (stated default else `1`) intact; no new config surface for it.
- [ ] A-003 R3: §5 rule 4 Strategic branch splits into auto-pick-and-notify (defensible recommendation) vs leave-open-and-notify (no defensible default); Routine and rule 6 unchanged.
- [ ] A-004 R4: The notification send is one shell command, default `rk notify` gated on `command -v rk`, with ntfy.sh/Discord/PushNotification/Slack documented as fallbacks; all sends documented fail-silent.
- [ ] A-005 R5: §4 documents the adaptive heartbeat — 90s when any monitored agent is menu-waiting, 3m otherwise, one-loop invariant preserved, autopilot 2m composes unchanged, Idle Message reflects the active interval.
- [ ] A-006 R6: §8 Settings gains the two session-scoped rows (menu-heartbeat 90s; notify channel `rk`); operator-state-file schema unchanged; no auto-default-threshold setting added.
- [ ] A-007 R7: `docs/specs/skills/SPEC-fab-operator.md` §4/§5/§8 summaries (and a new Resolved Design Decisions entry) reflect every skill edit; skill and SPEC agree.

### Behavioral Correctness

- [ ] A-008 R1: A strategic question on one monitored change does not stall advancement of other monitored changes within the same tick (the freeze failure mode is removed in the prose).
- [ ] A-009 R4: A notification that cannot be delivered (rk/server unreachable, no subscriptions, missing curl/tool) is documented to log one line and let the loop keep ticking — never crash or stall.

### Scenario Coverage

- [ ] A-010 R3: The §5 Logging block carries all six bullets — 5 answer-line shapes (`auto-answered`, NEW `auto-picked strategic … · notified`, NEW `strategic … left open · notified. Please respond.`, `can't determine answer …` rule-6, `auto-defaulted after 30m idle …`) plus a fail-silent `notify failed ({channel}). Continuing.` line.

### Edge Cases & Error Handling

- [ ] A-011 R5: The adaptive cadence is described as re-establishing the single loop (not spawning a second), so the `_cli-external.md` one-loop invariant holds.

### Code Quality

- [ ] A-012 Pattern consistency: The new prose matches the existing `fab-operator.md`/`SPEC-fab-operator.md` style — section structure, the §4 render-rule discipline, RFC-2119 phrasing in the SPEC, and the skill ↔ SPEC lockstep mirror convention.
- [ ] A-013 No unnecessary duplication: The notify-send abstraction is stated once in §5 and referenced (not re-inlined) where §8 names the setting; the `command -v rk` gate reuses the `_preamble.md` § Run-Kit discipline rather than redefining it.

### Documentation Accuracy

- [ ] A-014 Documentation accuracy: All file paths, version strings (`rk v2.3.2`), section numbers, and the 30m/90s/3m/2m values in both files are accurate and mutually consistent (config `checklist.extra_categories`).

### Cross-References

- [ ] A-015 Cross-references: References to `_preamble.md` § Run-Kit Reference (scoped to the `command -v rk` gate + fail-silent discipline only — NOT the `rk notify` command shape, which is the run-kit external contract), `_cli-external.md` § /loop, §1 "PR review is the safety net", and the §5 re-capture path resolve to real, current sections; the Slack-MCP headless-absence caveat is stated inline (no dangling §7 pointer) (config `checklist.extra_categories`).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- No Go tests exist for a skill-only change. Verification is the internal-consistency check (T005) plus the doc-accuracy / cross-reference acceptance categories drawn from `fab/project/config.yaml` `checklist.extra_categories`. Memory impact (`runtime/operator.md`) is confirmed at hydrate, not apply.

## Assumptions

<!-- Carried forward from intake.md (the design decisions reached conversationally) and
     graded for apply. Three grades only (Certain/Confident/Tentative) — apply decides
     and records. The intake's 12-row Assumptions table is the source; these are the
     apply-relevant subset that shaped requirement/task generation. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edits land in `src/kit/skills/fab-operator.md` (single flat file) + synchronous SPEC mirror `docs/specs/skills/SPEC-fab-operator.md` | Verified on disk; Constitution mandates synchronous SPEC mirrors for skill edits | S:90 R:85 A:95 D:95 |
| 2 | Certain | Strategic auto-default stays hardcoded at 30m — no value change, no new config surface | Intake decision (user: "keep 30m, just non-blocking"); explicitly out of scope | S:95 R:80 A:95 D:95 |
| 3 | Certain | Non-blocking escalation = post out-of-band + keep ticking + pick up async answer on a later tick; reuses existing `/loop` + re-capture, no new mechanism | Backlog's core ask; the existing architecture already supports it | S:90 R:75 A:90 D:90 |
| 4 | Confident | Strategic + defensible recommendation → auto-pick-and-notify (not parked); reversible at PR review | Intake decision; matches §1 "PR review is the safety net"; reuses §5 LLM-judgment signals | S:85 R:70 A:85 D:80 |
| 5 | Confident | Notify default = `rk notify`, gated on `command -v rk`; ntfy.sh / Discord / PushNotification / Slack are fallbacks; channel abstracted behind one shell send | run-kit Web Push shipped (`rk v2.3.2`); user-preferred channel; fail-silent by contract; ntfy.sh strongest no-`rk` fallback | S:90 R:80 A:90 D:80 |
| 6 | Confident | Adaptive heartbeat tightens to 90s when any monitored agent is on a menu, relaxes to 3m; preserves one-loop invariant by re-establishing the single loop | Backlog stated 60–90s; 90s = calmer end (less capture churn); one-loop rule respected | S:80 R:80 A:80 D:70 |
| 7 | Confident | All notify sends fail silently — never crash/stall the loop on a send failure | Mirrors `_preamble.md` § Run-Kit "fail silently" discipline; a coordination loop must not die on a notification error | S:80 R:85 A:90 D:85 |
| 8 | Confident | New §8 settings (menu-heartbeat, notify-channel) are session-scoped NL overrides; operator-state-file schema unchanged | Matches existing §8 rows (loop-interval, stuck-threshold); keeps the state-file schema minimal | S:80 R:80 A:85 D:80 |
| 9 | Tentative | Menu-detected heartbeat tightens only the monitoring loop, not autopilot's 2m cadence | Conservative default; autopilot has its own cadence model; flagged as an intake Open Question for review | S:55 R:70 A:60 D:55 |
| 10 | Tentative | When `rk` is absent, fallback channel is "first available documented alternative" (auto-detect), not an explicit session setting | Low-stakes, reversible; intake Open Question left to apply; the `Notify channel` setting still allows explicit conversational override | S:60 R:80 A:60 D:60 |

10 assumptions (3 certain, 5 confident, 2 tentative).
