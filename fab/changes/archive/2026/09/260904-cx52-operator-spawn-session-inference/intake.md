# Intake: Operator Spawn-Session Inference

**Change**: 260904-cx52-operator-spawn-session-inference
**Created**: 2026-09-05

## Origin

Created via `/fab-proceed`'s promptless create-new dispatch (`{questioning-mode} = promptless-defer`) from a synthesized design-discussion description — no interactive questioning; every decision below was converged in the prior discussion and is encoded as graded assumptions.

> Operator spawn-session inference — infer the spawn target session instead of asking on cold start. `fab-operator.md` §6 Spawning an Agent step 2's target-session fallback ladder ends at rung (d) "Cold start — ask the user once"; every fresh operator session with no monitored agents hits this ask even when the answer is obvious from the pane map the operator already fetches. Replace the rigid rungs with evidence-ordered judgment over the candidate sessions, hard-excluding `_rk-*` infrastructure sessions and the operator's own session, acting default-and-announce on evidence-backed inferences, and asking (or §5-escalating, unattended) only when genuinely torn.

## Why

**The pain point.** On a cold start the monitored set is *always* empty, so rungs (a)/(b) never fire, rung (c)'s §8 "Spawn target session" setting is session-lifetime-scoped and thus also empty, and rung (d) asks the user — on every fresh operator session. Live reproduction on this machine: the tmux server held `_rk-ctl` (1 window, run-kit's hidden control anchor), `_rk-operator` (1 window, the operator itself), and `fabKit` (15 windows, attached, holding 13 fab-kit worktree/agent panes). The operator's own ask-prompt *computed* "fabKit (Recommended) — where every other fab-kit worktree/agent pane currently lives (13 windows)" — then asked anyway.

**Root cause.** Rungs (a)/(b) consult only the *monitored* set, while the pane map (`fab pane map --all-sessions`, which the operator already fetches every tick) carries a richer, always-available signal the ladder does not permit the operator to use. And because the §8 setting lives only for the operator session, the ask recurs on every cold start.

**The consequence of not fixing it.** Every cold-started operator session pays one needless interruption; worse, on **unattended** spawns (watch/autopilot ticks) the cold-start rung escalates a §5 notification — a human round-trip — for an answer the pane map already knew. This defeats the operator's autonomy posture exactly where it matters (autopilot).

**Why this approach.** A wrong spawn target is highly reversible (`move-window`; step 7's `-P -F` print confirms the landing), so the correct posture is default-and-announce on evidence, not ask-first. The alternatives were considered and rejected in discussion (see Assumptions #5): a persistent config setting is a staleness trap (session names are per-tmux-server and ephemeral); a run-kit-provided session role is conceptually the right home but a cross-repo change (deferred as a run-kit backlog idea); the status quo asks exactly when the answer is obvious.

## What Changes

All edits are **skill prose in the canonical source `src/kit/skills/fab-operator.md`** — never the `.claude/skills/` deployed copies (Constitution V; code-quality anti-pattern). No Go code changes; no migration.

### 1. §6 step 2 — hard exclusion rule (MUST NOT)

`_rk-*`-prefixed sessions and the operator's own session are **never** spawn targets:

- `_rk-*` covers run-kit infrastructure — `_rk-ctl` (control anchor), `_rk-pin-*` (board pin-sessions), `_rk-operator` (the operator's own session). These are reserved constants in run-kit's `internal/tmux/tmux.go`.
- This **strengthens — does not replace —** the existing "the ambient session is never an implicit target" prohibition (which stays verbatim, along with the shell-escaped `-t '<session>:'` requirement and step 7's escaping rule).
- Coupling note to record in the change (plan Design Decisions → hydrate): fab formally adopts run-kit's `_rk-*` prefix as a **reserved-infrastructure naming convention**. Precedent exists — fab already consumes run-kit's `@rk_pane_agent_state` pane-option convention.

### 2. §6 step 2 — judgment over a rigid ladder

Within the remaining candidate sessions (after exclusions), the operator **decides from the evidence it already holds**, in strength order:

1. **Monitored agents for the target repo** — today's rungs (a)/(b), still the strongest signal, with the existing multi-session majority rule and most-recently-enrolled tie-break preserved.
2. **Pane-map repo affinity** — the session holding panes whose worktrees belong to the target repo, read from the same `fab pane map --all-sessions` snapshot the operator already fetches, with the existing majority/tie rules applied to pane-map evidence.
3. **Structural dominance** — exactly one plausible user-facing work session remains after exclusions.

Attached state and window count are **weaker signals used to support an announced choice, never a silent one**. In the reproduction above, tier 2/3 evidence (13 fab-kit worktree panes in `fabKit`; sole non-excluded session) decides `fabKit` without asking.

### 3. §6 step 2 — default-and-announce posture

On an evidence-backed inference the operator **acts**: it spawns into the chosen session, **announces the chosen session and its reason in the spawn output**, and **auto-sets it as the §8 "Spawn target session" setting** (mirroring what the cold-start answer already did) so later spawns stay consistent. The user can override at any time with "spawn into session X" (the existing §8 override phrase).

### 4. §6 step 2 — ask only when genuinely torn

The ask (attended) / §5 escalation (unattended) survives **only** for the genuinely-torn case: two or more plausible candidates with **no repo affinity separating them**. An evidence-backed decision is a *derivation*, not a *guess* — so autopilot/watch ticks get to decide too, instead of escalating on every cold start. The "never guess, never fall back to ambient" rule survives with "guess" now meaning **"decide without evidence"**.

### 5. §8 settings table — "Spawn target session" row

Update the row (currently `| Spawn target session | derived (§6 step 2 ladder) | "spawn into session {name}" |`, line ~850) to reflect the inference: the default is derived per §6 step 2's evidence-ordered inference, and the setting is now also **auto-set by the operator on each announced inference**, not only by a cold-start answer. Session-lifetime scoping (reset on compaction/`/clear`/restart) is unchanged.

### 6. Sibling sweep (code-quality.md obligation)

Grep repo-wide for restatements of the cold-start/ladder behavior and update every occurrence in the class. Verified sweep sites at intake time:

- `src/kit/skills/fab-operator.md:624` — "§6's target-repo + target-session … sequence" cross-reference in § Working a Change (likely wording-stable, verify).
- `src/kit/skills/fab-operator.md:703` — the autopilot working-list spawn step ("establish the change's target repo and target session") (likely wording-stable, verify).
- `src/kit/skills/fab-operator.md:516` — rung (d) itself and its §5-escalation sentence (the primary edit site).
- `docs/memory/runtime/operator.md:94` (spawn-derivation prose) and the z597 Design Decision block at lines ~467–476 — **memory updates belong to hydrate**, listed under Affected Memory.
- Verified **pointer-only, no restatement** (grep at intake): `docs/specs/skills.md` (zero target-session mentions), `docs/memory/runtime/agent-primitives.md:33` (points at "the operator's derivation ladder lives in `fab-operator.md` §6" — the pointer stays valid), `src/kit/skills/_cli-agents.md`. Re-verify with a fresh grep before finishing apply; user-facing string literals count.

### Non-goals

- **No run-kit changes.** Exposing a session-role / user-facing-session query from rk (`@rk_ses_role`, or `rk mux sessions --json` with attached/user-facing facts) is conceptually the right long-term home but cross-repo — file it as a run-kit backlog idea as a follow-up, outside this change.
- **No persistent config setting** (`operator.spawn_session`) — rejected in discussion as a staleness trap.
- **No Go code changes, no migration** (no user-data restructuring).

## Affected Memory

- `runtime/operator`: (modify) §6 spawn-derivation prose (the 4-rung ladder restatement) and the z597 "Spawn Target Session Is Derived Live Per Spawn" Design Decision block — update to the evidence-ordered inference model with the `_rk-*`/own-session hard exclusion, default-and-announce posture, §8 auto-set, and torn-only ask/escalation.

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (§6 step 2 + nearby prose, §8 settings row, internal cross-references at ~624/~703); `docs/memory/runtime/operator.md` at hydrate. Scale: one skill file plus one memory file.
- **Behavioral surface**: operator spawn flow only — cold-start attended spawns stop asking when evidence decides; unattended (watch/autopilot) spawns stop escalating except when genuinely torn. All other spawn steps (1, 3–8), the `-t` requirement, escaping, `-P -F` enrollment feed, and dependency handling are untouched.
- **Cross-repo coupling**: adopts run-kit's `_rk-*` reserved-prefix convention (documented, precedented — `@rk_pane_agent_state`).
- **Tests/CI**: none expected (skill-prose-only; no `.go` files touched).

## Open Questions

- None blocking. (Promptless dispatch — no decisions required user input; see Assumptions.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Hard MUST NOT exclusion: `_rk-*`-prefixed sessions and the operator's own session are never spawn targets; strengthens (not replaces) the ambient-never-implicit-target rule | Discussed — converged decision 1; prefix constants verified in run-kit's `internal/tmux/tmux.go` | S:95 R:85 A:90 D:95 |
| 2 | Certain | Rigid rungs replaced by evidence-ordered judgment: (i) monitored agents for target repo (existing majority/tie rules kept), (ii) pane-map repo affinity from the already-fetched `fab pane map --all-sessions` snapshot, (iii) structural dominance; attached state/window count support an announced choice only | Discussed — converged decision 2, with the live `fabKit` reproduction as evidence | S:90 R:80 A:85 D:90 |
| 3 | Certain | Default-and-announce: act on evidence-backed inference, announce session + reason in spawn output, auto-set the §8 "Spawn target session" setting; user overrides via "spawn into session X" | Discussed — converged decision 3; wrong target is highly reversible (`move-window`, `-P -F` confirms landing) | S:90 R:90 A:85 D:90 |
| 4 | Certain | Ask (attended) / §5-escalate (unattended) only when genuinely torn — ≥2 plausible candidates with no repo affinity separating them; "never guess" survives with guess = decide **without evidence**, so autopilot ticks may decide on evidence | Discussed — converged decision 4 | S:90 R:80 A:85 D:85 |
| 5 | Certain | Rejected alternatives stand: persistent `operator.spawn_session` config (ephemeral-identity staleness trap), run-kit session-role exposure (right home, cross-repo — deferred to an rk backlog idea as follow-up), status quo (asks exactly when obvious) | Discussed — all three explicitly rejected with reasons | S:90 R:85 A:90 D:90 |
| 6 | Certain | Skill-prose-only scope: canonical `src/kit/skills/fab-operator.md` (never `.claude/skills/` copies); no Go changes, no migration | Constitution V + code-quality anti-pattern; description states no user-data restructuring | S:85 R:75 A:90 D:85 |
| 7 | Certain | The change records the coupling note: fab formally adopts `_rk-*` as run-kit's reserved-infrastructure prefix (precedent: `@rk_pane_agent_state`) | Discussed — explicit constraint in the converged design | S:90 R:75 A:80 D:85 |
| 8 | Confident | Sweep class = fab-operator.md internal restatements (lines ~516/~624/~703, §8 row ~850) + `runtime/operator` memory at hydrate; `docs/specs/skills.md`, `agent-primitives.md`, and `_cli-agents.md` verified pointer-only by intake-time grep — re-verify before finishing apply | code-quality.md sibling-sweep obligation; grep evidence gathered at intake, apply re-confirms | S:75 R:80 A:80 D:80 |
| 9 | Confident | Exclusion scope is exactly `_rk-*` + the operator's own session — not a blanket `_*` hidden-session rule | Discussion scoped it to run-kit's reserved constants; a broader convention was not discussed and rk owns the substrate | S:70 R:80 A:75 D:70 |
| 10 | Confident | §8 auto-set fires on every announced inference (any evidence tier), mirroring the cold-start answer's session-lifetime persistence; §8 scoping/reset semantics unchanged | Decision 3 says "auto-sets it … so later spawns stay consistent"; direct extension of existing rung-(d) behavior | S:75 R:85 A:80 D:75 |
| 11 | Confident | Presentation of §6 step 2 (recast lettered rungs as evidence tiers vs. keep a lettered structure) is decided at apply; the requirement is semantic — exclusions, evidence order, posture — not typographic | Prose-structure choice, highly reversible, no behavioral ambiguity | S:50 R:90 A:70 D:50 |
| 12 | Confident | `change_type` = feat — operator behavior change (who gets asked, when spawns proceed), not a docs change, despite touching only markdown | Skill prose IS the operator's implementation in this repo; matches z597 precedent | S:70 R:90 A:85 D:80 |

12 assumptions (7 certain, 5 confident, 0 tentative, 0 unresolved).
