# Plan: Retire Operator /loop — Cron Entry as Sole Cadence

**Change**: 260909-6t88-retire-loop-cron-cadence
**Intake**: `intake.md`

## Requirements

### Operator Skill: Clock ownership (§4 rewrite)

#### R1: Cron entry documented as sole cadence for every provider
`src/kit/skills/fab-operator.md` §4 MUST document the rk cron operator-tick entry as the operator's sole cadence, provider-neutrally: the entry's shape (schedule `backoff min: 60s, max: 30m` anchored on operator idle with the anchor-join, `wake_on: agent-state-change` debounce `10s`, `suppress_while: [operator-loop-fresh, nothing-tracked]`, `target: role=operator`, `payload: "operator tick"`, `deliver: immediate`, `if_absent: respawn`, `pinned: true`), its ownership (`rk operator` seeds it idempotently; the skill never creates or mutates it — `rk cron` verbs are the user's), and how the union predicate covers the retired behaviors (`wake_on` ≻ the 90s tightened cadence; `nothing-tracked` ≻ stop-when-empty; backoff ≻ fixed 3m; `if_absent: respawn` + role-target fire-time resolution ≻ session-bound liveness).

- **GIVEN** any operator provider (Claude, codex, gemini, …) on a box where `rk agent setup` has run
- **WHEN** the operator is running with tracked state
- **THEN** §4 describes ticks as cron-delivered for that provider, with no provider-specific clock as primary
- **AND** §4 names run-kit's `docs/specs/cron.md` as the entry's design authority

#### R2: /loop machinery retired
§4 MUST NOT retain the `/loop` start/stop lifecycle paragraph, the **Adaptive cadence** block (3m/90s, relax-back, the one-loop invariant as cadence mechanics, the autopilot-composition cadence note), the **Loop Prompt** literals, or the dynamic-mode wakeup-prompt paragraph — except inside the single fallback paragraph (R5).

- **GIVEN** the rewritten §4
- **WHEN** grepping `src/kit/skills/fab-operator.md` for `/loop 3m`, `/loop 90s`, `Adaptive cadence`, `90s`
- **THEN** matches exist only inside R5's fallback paragraph (and none for `Adaptive cadence`/`90s` at all)

#### R3: Bare `operator tick` payload rule retained, re-grounded
The bare-text `operator tick` rule and its token-economics rationale (a slash-command payload macro-expands the ~21k-token skill on every firing) MUST survive, re-grounded to bind the cron entry's `payload`, any manually typed tick, and the fallback loop's prompt.

- **GIVEN** the rewritten §4
- **WHEN** a reader asks what a tick's text may be
- **THEN** the answer is the bare `operator tick` — never `/fab-operator` or any slash command — with the rationale intact

#### R4: Post-Compaction Reload retained, minus loop re-establishment
The Post-Compaction Reload procedure MUST survive with its trigger and one-shot `/fab-operator` reload unchanged, dropping only the loop re-establishment step; it SHOULD note that cron-delivered ticks keep arriving regardless of session health, so the reload trigger is guaranteed to eventually fire.

- **GIVEN** a compacted/resumed operator session with tracked state
- **WHEN** the next cron-delivered tick arrives and §4 Tick Behavior is not in context
- **THEN** the documented procedure is still: run `/fab-operator` once, treat the tick as consumed, resume bare ticks

#### R5: Claude-only degraded fallback
§4 MUST carry one short fallback paragraph: when the cron entry cannot exist (rk absent, or installed rk predating `rk cron` — probe `command -v rk` + `rk cron list` failing), a Claude Code operator MAY run `/loop 3m "operator tick"` as the fallback clock (bare-prompt rule unchanged); a non-Claude operator has no automatic cadence in that state. The fallback MUST NOT run alongside a live entry.

- **GIVEN** a box without rk (or with a pre-cron rk)
- **WHEN** a Claude operator starts with tracked state
- **THEN** the ready line's degraded form (R9) offers the `/loop 3m "operator tick"` literal
- **AND** with a live cron entry present, no `/loop` is started

#### R6: Tick step 7 (Loop lifecycle) removed
Tick Behavior's per-tick clock-management step MUST be removed; quiescence is the entry's `nothing-tracked` guard and cadence adaptation is the union predicate. A one-line lifecycle note MUST remain where step 7 was so existing cross-references (§2 Init, §6 merge-sequence run-condition) keep a target.

- **GIVEN** the rewritten Tick Behavior list
- **WHEN** a tick completes
- **THEN** no step instructs starting, stopping, or re-establishing any loop

#### R7: Idle message drops computed next-tick
The Idle Message MUST no longer render a computed `next:` time (the backoff rung is rk-derived, not skill-known); it shows the current time and the last tick (`fab operator time` may still supply `now:`; the `--interval`-based next-tick derivation is dropped).

- **GIVEN** a quiet moment between ticks
- **WHEN** the operator renders its idle message
- **THEN** it contains current time + last-tick reference and no predicted next-tick time

### Operator Skill: Startup (§2 Init)

#### R8: Init step 4 becomes clock verification
§2 Init step 4 MUST replace "start the single loop" with a fail-silent clock check: gated on `command -v rk`, run `rk cron list` and look for the operator-tick entry; absence or rk-absence is a degraded state, never an error.

- **GIVEN** operator startup on a box with rk + cron
- **WHEN** Init runs
- **THEN** step 4 verifies the entry exists (seeded by `rk operator`) and reports its state; no loop is started

#### R9: Ready-line copy
§2 Init step 5's ready line MUST report clock status instead of a loop literal — an active form naming the entry and its predicate, and a degraded form carrying the seed instruction (`rk operator`) plus the Claude-only fallback literal. The copy-never-compose rule survives: the degraded line's literal is what the agent copies.

- **GIVEN** startup with a live entry
- **WHEN** the ready line prints
- **THEN** it reads like `Operator ready. Clock: rk cron "operator tick" (backoff 60s–30m, wakes on agent-state-change)`
- **AND** with no entry it reads like `Operator ready. Clock: none — run rk operator to seed the cron entry, or (Claude Code only) start the fallback: /loop 3m "operator tick"`

### Operator Skill: Retirement sweep

#### R10: Cadence claims outside §4 updated
Every loop/cadence claim outside §4 MUST be updated to the cron posture: §1 header prose ("monitors progress via `/loop`. The loop is the heart of the operator"), the §1 principles "Survive compaction" row, §5's tightened-cadence trigger sentence and "block the loop" phrasing, §6's "a merge sequence in progress is by itself a loop run-condition" sentence (rewritten to document that the seeded entry's `nothing-tracked` guard does NOT yet cover an open merge-sequence note — a named run-kit follow-up, no fab-side workaround), §8's `Loop interval` and `Waiting/menu heartbeat` rows (removed), and §9's `Uses /loop?` row (replaced by a `Cadence` row).

- **GIVEN** the finished skill file
- **WHEN** reading any section other than §4
- **THEN** no text instructs running `/loop` or describes a 3m/90s skill-owned cadence
- **AND** §6 names the `nothing-tracked`/merge-sequence gap and its run-kit follow-up

### Helper and docs

#### R11: `_cli-external.md` § /loop shrinks to a fallback note
`src/kit/skills/_cli-external.md` § /loop MUST shrink to a fallback-scoped note — invocation syntax, the one-loop-at-a-time rule, the self-paced-mode bare-prompt rule, and a pointer to `fab-operator.md` §4's fallback paragraph as the sole policy owner (owner-or-pointer: no restated policy). The frontmatter description and Contents list MUST reflect the reduced role.

- **GIVEN** the edited helper
- **WHEN** an operator skill loads it
- **THEN** § /loop reads as fallback documentation pointing at §4 for policy

#### R12: Pulse-plan supersession banner
`fab/plans/sahil/26-09-03-operator-pulse-plan.md` MUST gain a top banner marking the whole two-clock design superseded by run-kit's `docs/specs/cron.md` + `fab/plans/sahil/26-09-06-cron-clock-plan.md` (Clock B half already superseded there; this change retires Clock A), noting what carried forward (OS-timer fallback + reboot analysis; guarded-delivery/TOCTOU mechanics into run-kit's injection engine; the sidecar state-dir precedent).

- **GIVEN** the plan doc
- **WHEN** opened
- **THEN** the banner is the first thing after the title and names both superseding documents

#### R13: `docs/specs/skills.md` fab-operator section updated
The § `/fab-operator` flow-skeleton line ("runs a continuous /loop cycle…") and the Tools line's `Skill (/loop)` entry MUST update to the cron-delivered-tick posture. `docs/specs/operator.md`'s v-history rows stay untouched (historical record).

- **GIVEN** the spec's fab-operator section
- **WHEN** read after this change
- **THEN** it describes cron-delivered ticks, not a continuous /loop cycle

### Non-Goals

- No Go changes (fab binary untouched; `fab operator time`/`tick-start` keep their flags) and no run-kit changes (the `nothing-tracked` guard extension is a flagged follow-up, not this change).
- No edits to append-only memory logs or to `docs/specs/operator.md` version history.
- No change to the tick procedure itself (snapshot/auto-nudge/watches/autopilot/removals steps stay; only the clock-management step goes).

### Design Decisions

#### Cron entry as the sole cadence, provider-neutral
**Decision**: `fab-operator.md` documents the rk cron operator-tick entry's union predicate as the only cadence for every provider; the skill owns no clock.
**Why**: `/loop` is Claude Code-only — for non-Claude operators the entry is the only possible clock; for Claude it eliminates the loop-death incident class and the dual-clock arrangement. `wake_on` strictly beats the 90s tightened cadence on pickup latency.
**Rejected**: Keeping the dual-clock arrangement (transitional by design; the `operator-loop-fresh` guard exists to make this migration safe, not to be permanent). Per-provider cadence sections (the entry is provider-agnostic substrate).
*Introduced by*: 260909-6t88-retire-loop-cron-cadence

#### Claude-only /loop fallback instead of hard-requiring rk
**Decision**: When the cron entry cannot exist (rk absent / pre-cron rk), a Claude operator may run `/loop 3m "operator tick"` as the fallback clock; never alongside a live entry.
**Why**: fab's rk doctrine is optional + fail-silent (`_preamble.md` § Run-Kit); the skill degrades on every other rk seam rather than erroring.
**Rejected**: Hard-requiring rk (wt-gate style) — contradicts the skill-wide rk-absent degradation posture; leaving no fallback — breaks rk-less Claude installs for no gain.
*Introduced by*: 260909-6t88-retire-loop-cron-cadence

#### Document the `nothing-tracked`/merge-sequence gap, no workaround
**Decision**: §6 documents that the seeded entry's `nothing-tracked` guard omits the open-merge-sequence run-condition, flagged as a run-kit follow-up.
**Why**: Guard semantics are run-kit's to extend; user confirmed (intake Clarifications 2026-09-09).
**Rejected**: Fab-side mitigation (dummy monitored entry or temporary fallback loop — monkey-patching); blocking C10 on the run-kit fix (merge-all is usually attended; small risk window).
*Introduced by*: 260909-6t88-retire-loop-cron-cadence

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §4: replace "The Loop" heartbeat/Adaptive-cadence/Loop-Prompt content with the cron-entry documentation (entry shape, ownership, union-predicate coverage map), the re-grounded bare-payload rule, the trimmed Post-Compaction Reload, and the Claude-only fallback paragraph <!-- R1, R2, R3, R4, R5 -->
- [x] T002 In `src/kit/skills/fab-operator.md` §4 Tick Behavior: remove step 7's clock management (leave the one-line lifecycle note), and simplify the Idle Message to current + last-tick time (drop the `--interval` next-tick derivation) <!-- R6, R7 -->
- [x] T003 In `src/kit/skills/fab-operator.md` §2 Init: replace step 4 with the gated `rk cron list` clock verification and rewrite step 5's ready-line copy (active + degraded forms, copy-never-compose preserved) <!-- R8, R9 -->
- [x] T004 Retirement sweep in `src/kit/skills/fab-operator.md`: §1 header prose + principles row, §5 trigger/`block the loop` wording, §6 merge-sequence run-condition sentence (document the `nothing-tracked` gap + run-kit follow-up), §8 remove the two cadence rows, §9 replace `Uses /loop?` with a `Cadence` row <!-- R10 -->

### Phase 3: Integration & Edge Cases

- [x] T005 [P] Shrink `src/kit/skills/_cli-external.md` § /loop to the fallback-scoped note (syntax + one-loop rule + bare-prompt rule + policy pointer to fab-operator §4); update frontmatter description and Contents <!-- R11 -->
- [x] T006 [P] Add the supersession banner to `fab/plans/sahil/26-09-03-operator-pulse-plan.md` (superseded by run-kit cron spec + plan; carried-forward notes) <!-- R12 -->
- [x] T007 [P] Update `docs/specs/skills.md` § /fab-operator: flow-skeleton line and Tools line to cron-delivered ticks <!-- R13 -->

### Phase 4: Polish

- [x] T008 Verification sweep: grep repo-wide (excluding `docs/memory/**/log*.md`, `docs/specs/findings/`, `fab/changes/`) for `/loop 3m "operator tick"`, `/loop 90s`, `Adaptive cadence`, `Waiting/menu heartbeat`, `operator tick` loop-context claims; confirm every remaining match is either §4's fallback paragraph, an append-only log, or a historical finding/plan record <!-- R2, R10 -->

## Execution Order

- T001 → T002 → T003 → T004 (same file, sequential)
- T005–T007 independent, parallel-safe
- T008 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: §4 documents the operator-tick entry (full shape incl. both suppress guards and `if_absent: respawn`), rk ownership, and the union-predicate coverage map, provider-neutrally, citing run-kit's cron spec
- [x] A-002 R8: §2 Init step 4 is a `command -v rk`-gated, fail-silent `rk cron list` clock verification
- [x] A-003 R9: the ready line has active and degraded forms per the R9 scenarios; degraded form carries `rk operator` seed instruction + the Claude fallback literal
- [x] A-004 R11: `_cli-external.md` § /loop is fallback-scoped with a policy pointer; frontmatter description + Contents updated
- [x] A-005 R12: the pulse plan opens with the supersession banner naming both superseding run-kit documents and the carried-forward items
- [x] A-006 R13: `docs/specs/skills.md` fab-operator flow + Tools lines describe cron-delivered ticks; `docs/specs/operator.md` v-history untouched

### Behavioral Correctness

- [x] A-007 R5: the fallback paragraph permits `/loop 3m "operator tick"` only for Claude Code and only when the entry cannot exist; explicitly never alongside a live entry
- [x] A-008 R7: the idle message shows current + last-tick time with no computed next-tick; no `--interval`-based derivation remains in the skill
- [x] A-009 R3: the bare `operator tick` payload rule + token-economics rationale survive, binding cron payload, manual ticks, and the fallback prompt
- [x] A-010 R4: Post-Compaction Reload keeps trigger + one-shot reload + tick-consumed semantics, has no loop re-establishment step, and notes cron ticks keep arriving

### Removal Verification

- [x] A-011 R2: no `Adaptive cadence` heading/text, no `90s` cadence, no `/loop 90s` literal anywhere in `fab-operator.md`; `/loop 3m "operator tick"` appears only in the fallback paragraph and the degraded ready line
- [x] A-012 R6: Tick Behavior contains no clock-management step; the one-line lifecycle note is present and §2/§6 cross-references resolve
- [x] A-013 R10: §8 no longer has `Loop interval` or `Waiting/menu heartbeat` rows; §9 has a `Cadence` row and no `Uses /loop?` row

### Scenario Coverage

- [x] A-014 R10: §6's rewritten sentence documents the `nothing-tracked`/merge-sequence gap and names it a run-kit follow-up (per the intake clarification)
- [x] A-015 R2, R10: T008's grep sweep ran and every remaining loop-literal match is fallback text, an append-only log, or a historical record

### Code Quality

- [x] A-016 Canonical source only: all skill edits under `src/kit/skills/`, none under `.agents/skills/` or `.claude/skills/`
- [x] A-017 Owner-or-pointer: `_cli-external.md` points at `fab-operator.md` §4 for fallback policy without restating it; no owned rule is both stated and pointed at
- [x] A-018 Sibling sweep: aggregate specs (`skills.md`; `glossary.md`/`architecture.md` checked — no loop claims found at plan time) and `kit-architecture.md`'s `_cli-external` description rows checked against the new § /loop role (memory rewrite itself is hydrate's)
- [x] A-019 Pattern consistency: new §4 prose matches the skill's existing section/register conventions (owned-rule statements, pointer style, literal blocks)
- [x] A-020 No unnecessary duplication: the cron entry's schema/semantics are summarized with authority pointed at run-kit's `docs/specs/cron.md`, not restated exhaustively

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `fab operator time --interval` flag + its `next:` output path (`src/go/fab/cmd/fab/operator*.go`; documented in `src/kit/skills/_cli-fab.md:1332`) — its only skill consumer (the §4 Idle Message next-tick render) was removed by this change; the flag is retained deliberately per plan Non-Goals (no Go changes), but nothing in the kit now passes it — future cleanup may retire it from the binary, docs, and tests together.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Exact ready-line strings as drafted in R9's scenarios (subject to line-length/register polish at apply) | Intake row 4 confirmed the shape; the literals are the natural rendering | S:95 R:85 A:75 D:70 |
| 2 | Confident | `fab operator time` remains referenced only for `now:` (or dropped if the tick header suffices); its `--interval` flag is simply no longer passed by the skill | No Go change needed; flag removal from the binary is out of scope (Non-Goals) | S:80 R:85 A:80 D:75 |
| 3 | Certain | Append-only logs (`docs/memory/**/log*.md`) and `docs/specs/findings/` are excluded from the retirement sweep | Logs record history; FKF present-truth applies to topic files only | S:90 R:90 A:95 D:90 |

3 assumptions (1 certain, 2 confident, 0 tentative).
