# Plan: Operator Spawn-Session Inference

**Change**: 260904-cx52-operator-spawn-session-inference
**Intake**: `intake.md`

## Requirements

### Operator Skill: Spawn Target-Session Inference

#### R1: Hard exclusion of rk-infrastructure sessions and the operator's own session
§6 step 2 MUST state a hard exclusion rule: `_rk-*`-prefixed sessions (run-kit infrastructure — `_rk-ctl` the control anchor, `_rk-pin-*` board pin-sessions, `_rk-operator` the operator session; reserved constants in run-kit's `internal/tmux/tmux.go`) and the operator's own session are **never** spawn targets. The rule SHALL strengthen — not replace — the existing "the ambient session is never an implicit target" prohibition, which stays, along with the shell-escaped `-t '<session>:'` requirement and step 7's escaping rule.

- **GIVEN** a tmux server holding `_rk-ctl`, `_rk-operator` (the operator's session), and `fabKit`
- **WHEN** the operator establishes a spawn target session
- **THEN** `_rk-ctl`, `_rk-operator`, and the operator's own session are excluded from candidacy before any evidence is weighed
- **AND** the ambient-session prohibition and `-t` escaping requirements remain verbatim

#### R2: Evidence-ordered inference replaces the rigid rung ladder
§6 step 2 SHALL direct the operator to decide the target session from the evidence it already holds, in strength order: (1) **monitored agents for the target repo** — today's rungs (a)/(b), preserving the existing multi-session majority rule and most-recently-enrolled tie-break; (2) **pane-map repo affinity** — the session holding panes whose worktrees belong to the target repo, read from the same `fab pane map --all-sessions` / tick-snapshot data the operator already fetches, with the same majority/tie rules; (3) **structural dominance** — exactly one plausible user-facing work session remains after R1's exclusions. Attached state and window count MAY support an announced choice but MUST NOT silently decide on their own.

- **GIVEN** a cold-started operator (empty monitored set) on a server where `fabKit` holds 13 panes whose worktrees belong to the target repo
- **WHEN** a spawn establishes its target session
- **THEN** tier-2 evidence decides `fabKit` without asking the user
- **GIVEN** a cold start where the target repo has no panes anywhere and exactly one non-excluded session exists
- **WHEN** a spawn establishes its target session
- **THEN** tier-3 structural dominance decides that session without asking

#### R3: Default-and-announce posture with §8 auto-set
On an evidence-backed inference the operator SHALL act: spawn into the chosen session, announce the chosen session **and its reason** in the spawn output, and auto-set it as the §8 "Spawn target session" setting so later spawns stay consistent. The existing "spawn into session {name}" override MUST remain the user's correction path.

- **GIVEN** an inference decided at any evidence tier
- **WHEN** the spawn proceeds
- **THEN** the spawn output names the session and the evidence that chose it, and the §8 setting is set to it
- **AND** a later "spawn into session X" from the user overrides it

#### R4: Ask or escalate only when genuinely torn
The ask (attended) / §5 escalation (unattended) SHALL survive only for the genuinely-torn case: two or more plausible candidates with no repo affinity separating them. An evidence-backed decision is a derivation, not a guess — unattended (watch/autopilot) spawns decide on evidence too. The "never guess, never fall back to ambient" rule survives with "guess" meaning **decide without evidence**.

- **GIVEN** an unattended autopilot spawn on a cold-started operator with decisive tier-2 evidence
- **WHEN** the spawn establishes its target session
- **THEN** it proceeds without a §5 escalation
- **GIVEN** two non-excluded sessions each holding panes of the target repo in equal measure with no tie-break signal
- **WHEN** a spawn establishes its target session
- **THEN** an attended spawn asks the user once; an unattended spawn escalates via §5

#### R5: §8 settings-table row reflects the inference
The §8 "Spawn target session" row (currently `derived (§6 step 2 ladder)`) SHALL be updated to reflect the evidence-ordered inference and the auto-set behavior: the setting is set both by a cold-start/torn answer and by each announced inference. Session-lifetime scoping (reset on compaction/`/clear`/restart) is unchanged.

- **GIVEN** the §8 Settings table
- **WHEN** the change lands
- **THEN** the row's Default cell describes the §6 step 2 evidence-ordered inference and notes the auto-set-on-announce behavior, and the override phrase is unchanged

#### R6: Sibling sweep — no stale restatement survives
Per `fab/project/code-quality.md` § Sibling Sweeps, a repo-wide grep for restatements of the cold-start/ladder behavior MUST be run before finishing apply, and every occurrence in the class updated. Known sites from the intake-time grep: `src/kit/skills/fab-operator.md` ~516 (the primary edit), ~624 (§ Working a Change cross-reference), ~703 (autopilot spawn step), ~850 (§8 row). Verified pointer-only (re-verify): `docs/specs/skills.md`, `docs/memory/runtime/agent-primitives.md:33`, `src/kit/skills/_cli-agents.md`. `docs/memory/runtime/operator.md` (line ~94 prose + the z597 DD block) is hydrate-owned, not an apply edit.

- **GIVEN** the completed edits
- **WHEN** grepping repo-wide for the old ladder wording (rung labels, "Cold start — ask the user once", "fallback ladder") outside `docs/memory/` and `fab/changes/`
- **THEN** no stale restatement of the old rung-(d) ask-first behavior remains in `src/kit/`

### Non-Goals

- No run-kit changes — exposing a session-role / user-facing-session query from rk is a follow-up backlog idea, outside this change.
- No persistent `operator.spawn_session` config setting — rejected as an ephemeral-identity staleness trap.
- No Go code changes, no migration, no `.claude/skills/` edits (deployed copies).

### Design Decisions

#### Fab Adopts run-kit's `_rk-*` Reserved-Infrastructure Prefix
**Decision**: fab's operator skill formally treats the `_rk-*` session-name prefix as run-kit's reserved-infrastructure namespace and excludes such sessions from spawn-target candidacy.
**Why**: The prefix is stable, structural, and already load-bearing in run-kit (`PinSessionPrefix`, `ControlAnchorSessionName`, `OperatorSessionName` constants in `internal/tmux/tmux.go`); excluding it removes the noise sessions that make cold-start inference ambiguous. Precedent exists — fab already consumes run-kit's `@rk_pane_agent_state` pane-option convention as a data contract.
**Rejected**: A blanket `_*` hidden-session rule (broader than what rk reserves; not discussed and rk owns the substrate). Waiting for rk to expose a session-role query (right long-term home, but cross-repo — deferred to a run-kit backlog idea).
*Introduced by*: 260904-cx52-operator-spawn-session-inference

## Tasks

### Phase 1: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §6 step 2: add the hard exclusion rule (`_rk-*` + own session, MUST NOT), recast the rungs as evidence-ordered inference (monitored agents → pane-map repo affinity → structural dominance; attached/window-count announce-support only), add the default-and-announce posture with §8 auto-set, and narrow the ask/§5-escalation to the genuinely-torn case with the reworded never-guess rule <!-- R1, R2, R3, R4 -->
- [x] T002 Update the §8 Settings table "Spawn target session" row in `src/kit/skills/fab-operator.md` (~850) to describe the evidence-ordered inference default and the auto-set-on-announce behavior <!-- R5 -->
- [x] T003 Sweep: re-verify and reconcile the internal cross-references at `src/kit/skills/fab-operator.md` ~624 and ~703, then repo-wide grep (old rung wording, "Cold start", "fallback ladder", §8 row phrase; user-facing string literals included) and update every stale occurrence in `src/kit/`; confirm `docs/specs/skills.md`, `docs/memory/runtime/agent-primitives.md`, and `src/kit/skills/_cli-agents.md` remain pointer-only <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: §6 step 2 states the MUST NOT exclusion naming the `_rk-*` prefix (with `_rk-ctl` / `_rk-pin-*` / `_rk-operator`) and the operator's own session, and keeps the ambient-never-implicit-target prohibition and `-t` escaping requirements verbatim
- [x] A-002 R2: §6 step 2 orders the evidence monitored-agents → pane-map repo affinity → structural dominance, preserves the existing majority/tie rules at tiers 1–2, and marks attached state/window count as announce-support only
- [x] A-003 R3: §6 step 2 directs announce-with-reason in the spawn output and auto-sets the §8 setting on every announced inference; the "spawn into session {name}" override is referenced
- [x] A-004 R4: the ask/§5-escalation path is scoped to the genuinely-torn case; unattended spawns decide on evidence; "never guess" is reworded to decide-without-evidence
- [x] A-005 R5: the §8 "Spawn target session" row reflects the inference default and auto-set behavior with scoping unchanged

### Behavioral Correctness

- [x] A-006 R2: the intake's live reproduction (operator in `_rk-operator`, work in `fabKit` with 13 target-repo panes) resolves to `fabKit` without an ask under the new prose, traceable through the written tiers

### Scenario Coverage

- [x] A-007 R4: both R4 scenarios are covered by the prose — evidence-backed unattended spawns proceed; torn cases ask (attended) or escalate (unattended)

### Edge Cases & Error Handling

- [x] A-008 R2: zero-candidate cold start (target repo has no panes anywhere, multiple non-excluded sessions) falls through to the torn-case ask/escalation rather than silently picking

### Removal Verification

- [x] A-009 R6: repo-wide grep shows no stale restatement of the old 4-rung ask-first ladder in `src/kit/`; the ~624/~703 cross-references are consistent with the new step 2

### Code Quality

- [x] A-010 Pattern consistency: the new §6 step 2 prose matches the skill's existing voice, structure, and cross-reference conventions
- [x] A-011 Canonical source only: all edits land in `src/kit/skills/fab-operator.md`; no `.claude/skills/` files touched
- [x] A-012 Owner-or-pointer: no rule owned elsewhere is restated alongside a pointer; the exclusion rule's run-kit facts are stated as fab's adopted convention, not as a copy of rk's docs

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- `docs/memory/runtime/operator.md` updates (line ~94 spawn prose + the z597 DD block) are hydrate-owned — listed in the intake's Affected Memory, deliberately not an apply task.

## Deletion Candidates

- None — this change rewrites the old rung-ladder prose of `fab-operator.md` §6 step 2 in place; it makes no other code or prose redundant. (`docs/memory/runtime/operator.md`'s old-ladder restatement becomes stale, but that is planned hydrate work, not a discovered redundancy.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | §6 step 2 keeps a lettered/tiered list structure (exclusion preamble + ordered evidence tiers + posture + torn-case rule) rather than free prose — mirrors the current rung ladder's scannability and the skill's list-heavy voice | Intake assumption 11 delegated presentation to apply; structure choice is typographic, highly reversible | S:60 R:90 A:80 D:60 |
| 2 | Confident | The memory-file edit is excluded from `## Tasks` and left to hydrate — the intake lists it under Affected Memory and the pipeline's hydrate stage owns memory rewrites | Pipeline convention (Constitution II; hydrate owns docs/memory) | S:75 R:85 A:85 D:80 |
| 3 | Confident | The §8 setting slots as tier (c) — below repo-scoped live evidence (monitored agents, pane-map affinity), above structural dominance | Preserves the old ladder's live-evidence-over-setting precedence (a/b above c) in the multi-repo model; intake assumption 10 makes the setting an auto-set consistency anchor, not a trump card | S:65 R:85 A:80 D:70 |

3 assumptions (0 certain, 3 confident, 0 tentative).
