# Plan: Operator Spawn Target Session — Delegate to `rk tab new`'s Role-Aware Default

**Change**: 260913-1nxw-operator-spawn-session-rk-default
**Intake**: `intake.md`

## Requirements

### Operator skill: spawn sequence (`src/kit/skills/fab-operator.md` §6)

#### R1: Step 2 delegates instead of choosing
§6 "Spawning an Agent" step 2 MUST keep its slot and number and MUST be replaced by a delegation of ≤ 60 words: the target session is resolved by `rk tab new` at step 7 (omit `--session`; rk applies its role-aware default and reports `session` + `session_rung`); `--session =<name>` is passed only when the user set the §8 "Spawn target session" override. The a–e rung list, the candidate-set definition, the `fab pane map` pane-count rung, the zero-candidate clause, and the fab-side announce rule MUST be deleted. The sentence "Never trust a persisted `scope.session` alone; the ambient session is never an implicit target" MUST be kept verbatim.

- **GIVEN** the edited step 2
- **WHEN** it is read as the operator model
- **THEN** the operator makes no session decision of its own and passes `--session` only for the §8 override

#### R2: Step 7 raw form, parenthetical, and announce
The raw form MUST read `rk tab new [--session =<name>] --cwd <worktree> --name <wt> --ready --json -- <spawn-argv…> ["<prompt>"]`. The parenthetical MUST state: `--session =<name>` only for the §8 override, otherwise rk resolves the landing session (step 2); an unknown explicit session and rk's "nowhere to spawn" both error loudly and are surfaced, never retried against the ambient session; the `--json` report (the `result` object of run-kit's envelope) carries `session` and `session_rung` alongside `window_id`, `pane_id`, and `ready`; step 8 consumes `session`; a one-line announce echoes `session` + `session_rung` in the spawn output, never via §5. `<wt>` is the worktree name from step 3. The `ready:` sentence and the Window marks paragraph are unchanged.

- **GIVEN** an `_rk-operator` caller with one user session on the server
- **WHEN** step 7 runs without `--session`
- **THEN** rk lands the window in that session and the report says `session_rung: sole-user`, which the announce echoes

- **GIVEN** no user session on the server
- **WHEN** step 7 runs
- **THEN** rk fails "nowhere to spawn" and the operator surfaces the error like any other `rk tab new` failure

#### R3: Step 8 takes `<session>` from the report
Step 8's `track add … --session <session>` and `track update … "session":"<session>"` MUST take `<session>` from the `--json` report's `session` field and no other source. Command shapes are unchanged.

- **GIVEN** a spawn whose report says `session: fab_kit`
- **WHEN** step 8 tracks the item
- **THEN** `--session fab_kit` is passed, regardless of the §8 setting or any persisted `scope.session`

### Operator skill: gate and settings (`fab-operator.md` §2, §8, §9)

#### R4: rk Gate gains the default-session capability half
The §2 rk Gate probe MUST gain a third `&&`-chained half that fails on an rk predating `rk tab new`'s default-session resolution (recommended: `rk tab new --help 2>&1 | grep -q session_rung`); the "either half" explanation MUST become "any part" and name both `rk cron` and `rk tab new`'s default-session resolution; the STOP line text stays (an upgrade hint MAY be appended on the same line). The §9 Key Properties "Requires HexoKit?" row MUST name the third half. One clause MUST carry the rationale: without it a pre-3.19.59 rk silently reintroduces the z597 misplacement once `--session` is omitted.

- **GIVEN** rk 3.19.58 or older on PATH
- **WHEN** the operator starts
- **THEN** the gate STOPs with the HexoKit error instead of later misplacing a window

#### R5: §8 row is a pure user override onto `--session`
The §8 "Spawn target session" row MUST read, in shape: default "none — rk tab new's role-aware default (§6 step 2)"; override `"spawn into session {name}"` → passes `--session =<name>`. The footnote is unchanged.

- **GIVEN** the user said "spawn into session work"
- **WHEN** the next spawn runs
- **THEN** step 7 passes `--session =work` and the report's `session_rung` is `explicit`

### Agent primitives helper (`src/kit/skills/_cli-agents.md` § Spawn Composition)

#### R6: Raw form and pin bullet are mechanics-only
The code block MUST read `rk tab new [--session =<session>] --cwd <dir> --name <name> --ready --json -- <composed-argv…> ["<initial-prompt>"]`. The bullet "`--session`, `--cwd`, and `--name` pin where the tab lands — there is no ambient-session guesswork. Which session is the right target is the caller's policy, not this file's." MUST be replaced by: rk resolves the landing session (role-aware default — an `_rk-*` caller never gets the window beside itself; `--session =S` overrides) and reports it as `session`/`session_rung` in the `--json` result; `--cwd`/`--name` still pin directory and name. No policy prose about which session is right.

- **GIVEN** any caller reading the helper
- **WHEN** it opens a tab without `--session`
- **THEN** it knows rk chooses the session and where to read the choice

### Cross-cutting

#### R7: Net shorter, swept, portable
After apply `wc -w src/kit/skills/fab-operator.md` MUST be < 14594 and the step-2 item ≤ 60 words. The sweep grep `--session =<session>|rung a|rung \(a\)|sole candidate|first row|nowhere to spawn|selector|session_rung` over `src/kit/` MUST return no operator-skill hit except the intended new mentions of `session_rung`/"nowhere to spawn" (the `fab agent` addressing-form "selector" mentions in `_cli-agents.md` are unrelated and stay). Pointers at `fab-operator.md` § Working a Change ("target-repo + target-session"), § Queues ("target repo and target session", "steps 4–8"), and "The same eight steps run" stay. `cd src/go/fab-kit && go test ./cmd/fab/` MUST pass; `fab sync` MUST run clean.

- **GIVEN** the completed apply
- **WHEN** the counts, grep, test, and sync run
- **THEN** all four conditions hold

### Memory (hydrate)

#### R8: Memory rewritten to present truth
At hydrate — never at apply — `docs/memory/runtime/operator.md` MUST be rewritten in place at: the startup rk-gate paragraph (third probe half + explanation); § Repo- and session-targeted spawning item 1 (delegation text) and item 4 (optional `--session`; report carries `session`, `session_rung`, `window_id`, `pane_id`, `ready`; `track add --session` consumes `session`); § Settings row (as R5); the Design Decision "Spawn Target Session Is Selected Deterministically Per Spawn, Never Asked, Never Persisted-and-Trusted" (retitled to resolution by `rk tab new`; Decision/Why/Rejected per intake § H; `*Updated by*` extended with this change); "Spawn-Target Candidacy Delegates to `rk mux sessions`" (candidacy and selection in rk; drop the rungs (c)/(e) clause); "The Spawn Command Reports Its Own Landing" (`session`/`session_rung` consumed); "rk Is a Hard Dependency of the Operator Skill" (probe list + "the capabilities the clock and the spawn need"). `docs/memory/runtime/agent-primitives.md` § Spawn composition: raw form `[--session =<session>]` and the pin clause → "rk resolves the landing session". `log.md` is never hand-edited.

- **GIVEN** the hydrate stage
- **WHEN** the memory files are re-read
- **THEN** no memory sentence describes a fab-side session selector, and every probe restatement names three halves

### Non-Goals

- No Go changes, migration, run-kit changes, or `fab` command-signature changes; `_cli-fab-operator.md` untouched.
- No renumbering of §6 steps.
- No pre-check for the zero-user-session case and no worktree cleanup: rk's "nowhere to spawn" surfaces at step 7 like today's unknown-session / `gone` failures (intake assumption 14).
- No edit to `docs/specs/operator.md` or to deployed copies under `.agents/skills/` / `.claude/skills/`.

### Design Decisions

#### Session Landing Policy Lives in `rk tab new`, Gated by a Capability Probe
**Decision**: The operator skill carries no target-session selector. Step 7 omits `--session` (except the §8 user override) and rk's role-aware default places the window, reporting `session` and `session_rung`; the §2 rk Gate probes `rk tab new --help` for `session_rung` so an rk without the feature stops the operator at startup instead of misplacing windows.
**Why**: rk owns session roles and the tab primitive, so it is the right home for "where does a spawned window land"; the fab-side a–e selector (iyb4) was the interim and would otherwise be a second copy of rk's `sole-user`/`cwd-root`/`most-attached` policy. A pre-3.19.59 rk given no `--session` from an `_rk-operator` caller lands the window beside the operator silently (the z597 bug), so the delegation is safe only behind a capability probe, and fab's rk convention is capability probes, never version strings.
**Rejected**: Keeping the fab-side selector alongside rk's (drift); renumbering §6 (touches every step cross-reference for no gain); no probe (silent z597 regression on lagging installs); parsing `rk --version` (against the kit's probe convention).
*Introduced by*: 260913-1nxw-operator-spawn-session-rk-default

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §6 step 2 as the ≤ 60-word delegation (keep slot, keep the z597 sentence verbatim; delete rungs, candidate set, zero-candidate clause, announce); rewrite step 7's raw form to `[--session =<name>]` and its parenthetical (override-only `--session`, loud errors incl. "nowhere to spawn", report keys `session`/`session_rung`, one-line announce in spawn output never §5); add the step-8 clause that `<session>` is the report's `session` field only <!-- R1, R2, R3 -->
- [x] T002 [P] Edit `src/kit/skills/fab-operator.md` §2 rk Gate: third probe half `rk tab new --help 2>&1 | grep -q session_rung`, "any part fails" explanation naming `rk cron` and `rk tab new`'s default-session resolution, rationale clause, STOP line kept (+ upgrade hint); §9 Key Properties "Requires HexoKit?" row names the third half; §8 "Spawn target session" row → rk default / override passes `--session =<name>` <!-- R4, R5 -->
- [x] T003 [P] Edit `src/kit/skills/_cli-agents.md` § Spawn Composition: code block `[--session =<session>]`; replace the pin bullet with the rk-resolves-the-landing-session mechanics bullet; mention `session`/`session_rung` in the `--ready` bullet's report keys <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Sweep + verify: run the R7 grep over `src/kit/` and fix any skill hit; confirm the § Working a Change / § Queues / "eight steps" pointers need no change; `wc -w` file < 14594 and step 2 ≤ 60; `cd src/go/fab-kit && go test ./cmd/fab/`; `fab sync`; record both word counts and list the R8 memory sites in `## Notes` (no `docs/memory/` edit at apply) <!-- R7, R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Step 2 is ≤ 60 words, keeps its number, delegates to `rk tab new`, names the §8 override as the only `--session` source, keeps the z597 sentence verbatim, and contains no rung list, candidate set, zero-candidate clause, or announce rule
- [x] A-002 R2: Step 7's raw form shows `[--session =<name>]`; the parenthetical covers override-only `--session`, loud errors for an unknown explicit session and "nowhere to spawn", the report keys `session`/`session_rung`, step 8's consumption of `session`, and the one-line announce in spawn output (never §5)
- [x] A-003 R3: Step 8 states `<session>` comes from the report's `session` field and no other source; both command shapes are unchanged
- [x] A-004 R4: The gate one-liner has three `&&`-chained halves, the explanation says "any part" and names `rk tab new`'s default-session resolution, a rationale clause names the silent misplacement, and the §9 "Requires HexoKit?" row names the third half
- [x] A-005 R5: The §8 row reads as rk's default plus a user override that passes `--session =<name>`; the footnote is unchanged
- [x] A-006 R6: `_cli-agents.md`'s code block shows `[--session =<session>]` and the pin bullet is replaced by the rk-resolves-the-landing-session bullet with no which-session policy prose
- [x] A-007 R8: No `docs/memory/` file was modified at apply and `## Notes` enumerates the hydrate sites

### Behavioral Correctness

- [x] A-008 R1: Reading §6 as the operator model, a spawn from `_rk-operator` with one user session lands there via rk with no operator-side decision and no question
- [x] A-009 R2: The `ready:` handling sentence and the Window marks paragraph in step 7 are byte-identical to before
- [x] A-010 R4: The STOP line still begins `Error: the operator requires HexoKit — brew install`

### Removal Verification

- [x] A-011 R7: `grep -nE 'rung a|rung \(a\)|sole candidate|first row|fab pane map --all-sessions --json' src/kit/skills/fab-operator.md` returns zero hits in §6 step 2, and `grep -n -- '--session =<session>' src/kit/skills/` returns zero hits (second clause's intent met — the sole hit is `_cli-agents.md:87`'s bracketed `[--session =<session>]`, the optional form R6 itself mandates; no mandatory `--session =<session>` remains)

### Scenario Coverage

- [x] A-012 R2: The two R2 scenarios (sole-user landing with echoed rung; "nowhere to spawn" surfaced at step 7) are derivable from the step-7 text
- [x] A-013 R5: The §8 override scenario resolves to `--session =<name>` and rk's `explicit` rung

### Edge Cases & Error Handling

- [x] A-014 R4: An rk lacking `session_rung` in `rk tab new --help` fails the gate at startup; the skill carries no below-the-gate fallback
- [x] A-015 R7: `wc -w src/kit/skills/fab-operator.md` < 14594 and the step-2 item ≤ 60 words

### Code Quality

- [x] A-016 Pattern consistency: Edits follow the skill's §-reference and owner-or-pointer style; the probe follows the kit's `--help`-discriminant precedent; `_cli-agents.md` stays mechanics-only
- [x] A-017 No unnecessary duplication: No site restates rk's rung list; step 2, step 7, §8, §9, and `_cli-agents.md` each carry only what they own
- [x] A-018 Canonical source only: Edits land in `src/kit/skills/`, never `.agents/skills/` or `.claude/skills/`
- [x] A-019 Deployed content cites no fab-kit-only paths (Constitution V; `cd src/go/fab-kit && go test ./cmd/fab/` passes)
- [x] A-020 Sibling sweep: Every skill-set restatement of the gate probe and the raw form was updated before apply finished; memory restatements are listed for hydrate

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **Hydrate checklist (R8)** — `docs/memory/runtime/operator.md`: startup rk-gate paragraph (third half + explanation); § Repo- and session-targeted spawning items 1 and 4; § Settings row; DD "Spawn Target Session Is Selected Deterministically…" → resolution by `rk tab new` (trail extended); DD "Spawn-Target Candidacy Delegates to `rk mux sessions`" → candidacy and selection in rk; DD "The Spawn Command Reports Its Own Landing" → `session`/`session_rung` consumed; DD "rk Is a Hard Dependency of the Operator Skill" → probe list + Rejected wording. `docs/memory/runtime/agent-primitives.md` § Spawn composition: raw form and pin clause. `log.md` untouched; `fab status set-summary` names the delegation and the capability gate.

## Deletion Candidates

None — this change removes the fab-side session selector itself (step 2's a–e rungs, candidate set, zero-candidate clause, and announce rule were deleted in the diff); no surviving code or prose was made redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Probe spelled `rk tab new --help 2>&1 \| grep -q session_rung`, `&&`-chained third in the existing one-liner | Intake assumption 6 front-runner; `--help`-discriminant precedent in `_cli-fab-operator.md` | S:75 R:90 A:80 D:70 |
| 2 | Confident | STOP line gains "(or upgrade)" — `brew install (or upgrade) sahil87/tap/run-kit` — on the same line | Intake allows the hint if it fits; an installed-but-old rk is now a real STOP cause | S:65 R:95 A:80 D:70 |
| 3 | Confident | Announce rendered as `→ session <name> (<session_rung>)` in the spawn output | Intake assumption 15 left rendering to apply | S:70 R:90 A:80 D:70 |
| 4 | Confident | The run-kit envelope parenthetical is dropped from step 2; step 7 already names the envelope | Intake assumption 17; word budget | S:65 R:95 A:85 D:70 |
| 5 | Certain | Memory edits are hydrate's; apply touches only the two skill files and lists the sites in `## Notes` | Pipeline convention; intake assumption 12 | S:90 R:95 A:90 D:90 |

5 assumptions (1 certain, 4 confident, 0 tentative).
