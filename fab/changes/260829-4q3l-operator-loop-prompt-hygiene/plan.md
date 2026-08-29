# Plan: Operator Loop-Prompt Hygiene

**Change**: 260829-4q3l-operator-loop-prompt-hygiene
**Intake**: `intake.md`

## Requirements

### Operator: Loop Prompt

#### R1: Loop prompt is a hard rule with a copyable literal
`fab-operator.md` §4 MUST own a `### Loop Prompt` subsection containing the exact `/loop 3m "operator tick"` / `/loop 90s "operator tick"` literals, a MUST (bare text `operator tick`) / MUST NOT (`/fab-operator` or any slash command) pair with the one-line macro-expansion reason, and a clause extending the rule to the self-paced (dynamic) `/loop` mode's wakeup prompt. §9 `Uses /loop?` and §2 Init step 4 MUST point at it, never restate it.

- **GIVEN** an operator session about to start or re-establish its loop
- **WHEN** it reads §4
- **THEN** it copies a literal from § Loop Prompt and never composes a slash-command prompt

#### R2: Init ready line carries the loop literal
§2 Init step 5 MUST print the loop line the agent uses — `Operator ready. Loop active ({interval}) — /loop {interval} "operator tick"` when the loop started, otherwise `Operator ready. Loop idle — start with /loop 3m "operator tick" on first enrollment` — and the §4 loop-start sentence MUST repeat the literal so a mid-session start is copied.

- **GIVEN** Init completes with no tracked items
- **WHEN** the ready line prints
- **THEN** the literal the agent will later use is already on screen

#### R3: Single cadence owner
§4 Adaptive cadence MUST be the only cadence owner: the "Autopilot composition" bullet and Tick Behavior step 7's "(autopilot uses §6's cadence)" are rewritten so autopilot rides the single `3m`/`90s` loop; no "default 2m" autopilot cadence remains anywhere in kit skills, memory, or specs.

- **GIVEN** an autopilot queue is driving
- **WHEN** the tick applies loop lifecycle
- **THEN** it uses §4 Adaptive cadence with no separate autopilot interval

### Operator: Context Survival

#### R4: "Survive compaction" replaces "Self-manage context"
The §1 row MUST state the agent cannot `/clear` itself and point at a new §4 `### Post-Compaction Reload` subsection owning the procedure: trigger (tick fires, Tick Behavior not in context), one-shot `/fab-operator` reload + §2 Init re-run, triggering tick treated as consumed, then bare `operator tick` firings; durable-state fact retained; `/clear` described as a user action landing on the same procedure. Every other `/clear` mention in `fab-operator.md` MUST be reworded per the intake's sweep table so `/clear` never appears as an agent action.

- **GIVEN** a long session auto-compacted and a tick arrives
- **WHEN** the agent finds no Tick Behavior in context
- **THEN** it runs `/fab-operator` once, re-inits, and resumes lean ticks — never puts `/fab-operator` in the loop prompt

### Helpers: _cli-external

#### R5: `/loop` section owns only tool-level facts
`_cli-external.md` § /loop Constraints MUST keep the one-loop rule (with the re-establish-not-add clause) and a pointer to `fab-operator.md` §4 for start/stop, cadence, autopilot composition, and the loop-prompt rule; the Start/Stop/Autopilot-override bullets MUST be removed; Usage MUST note the self-paced mode with a pointer to § Loop Prompt.

- **GIVEN** a reader of `_cli-external.md`
- **WHEN** they look for operator loop policy
- **THEN** they find a pointer, not a contradicting restatement

### Docs: Sibling sweep

#### R6: Memory and specs reflect present truth
`docs/memory/runtime/operator.md` (§ `/loop` lifecycle paragraph, § Settings table's 2m row, context-loading `/clear` wording, a Design Decisions entry), `docs/memory/_shared/context-loading.md` (operator partial-exception `/clear` wording), and `docs/specs/skills.md` (§ /fab-operator continuity line + Flow line) MUST carry the loop-prompt rule, the compaction-reload procedure, and the single-cadence fact; no "default 2m" claim survives.

- **GIVEN** a repo-wide grep for `default \`2m\``, `Self-manage context`, and operator-context `/clear`
- **WHEN** run after apply (excluding `.claude/`, archive, findings, `log*.md`)
- **THEN** only user-action/durability `/clear` mentions remain and no 2m cadence claim exists

### Non-Goals
- Slimming the `_cli-fab.md` load — user decision.
- Shrinking the per-tick status frame / `--quiet` tick output — separate follow-up FULL change.
- Go-side operator heartbeat daemon — backlog `[2ne8]`.
- Any change to `/loop` itself (harness-owned) or to `src/go/`.

### Design Decisions

#### Loop Prompt Is a Bare Token, Never a Slash Command
**Decision**: The operator's `/loop` prompt (and the dynamic-mode wakeup prompt) is the literal bare text `operator tick`; `/fab-operator` or any slash command in that position is prohibited, and the literals are printed at Init and at the loop-start point so the agent copies rather than composes.
**Why**: Slash commands macro-expand their full source into the turn on every firing; `fab-operator.md` is ~21k tokens, so a `/fab-operator` loop prompt re-pays the whole skill each tick (~400k tokens/hour at 3m) and exhausts the context window in ~10 ticks — observed live 2026-08-29. The tick procedure is already in context; the prompt only needs to name it.
**Rejected**: Leaving the rule as a single prose mention (it was there and still drifted); a per-tick `--quiet` frame (orthogonal, deferred); a Go daemon heartbeat (removes the LLM tick entirely — much larger, backlog).
*Introduced by*: 260829-4q3l-operator-loop-prompt-hygiene

#### Compaction Recovery Is a One-Shot Reload, Not an Agent `/clear`
**Decision**: The agent never `/clear`s (it cannot). When a tick arrives and the Tick Behavior procedure is not in context, it runs `/fab-operator` exactly once (reload + §2 Init), treats that tick as consumed, and resumes bare `operator tick` firings. Autopilot has no separate cadence — it rides the single `3m`/`90s` loop.
**Why**: The old principle named a user-only mechanism and omitted the one that actually happens (harness auto-compaction), leaving the agent with no procedure for "told to tick, but no tick procedure in context". Level-triggered deltas re-emit on the next `tick-start --diff`, so the consumed tick loses nothing durable. A second autopilot cadence contradicted the one-loop invariant and had no real owner (circular pointer between `fab-operator.md` and `_cli-external.md`).
**Rejected**: Instructing a periodic `/clear` (impossible for the agent); keeping `/fab-operator` in the loop prompt to "stay loaded" (the failure mode itself); relocating a 2m autopilot cadence into §4 (a second cadence for one loop).
*Introduced by*: 260829-4q3l-operator-loop-prompt-hygiene

## Tasks

### Phase 2: Core Implementation

- [x] T001 Edit `src/kit/skills/fab-operator.md`: §1 row → "Survive compaction"; §2 Context Loading `/clear` wording; §2 Init steps 2/4/5 (ready-line literals, pointer to § Loop Prompt); §4 opening sentence + Adaptive-cadence "Autopilot composition" bullet; add `### Loop Prompt` and `### Post-Compaction Reload` after the cadence bullets; §4 Monitored Set "reload-restored"; Tick Behavior step 7; §6 rule 4; §8 Settings sentence; §9 `Uses /loop?` pointer <!-- R1 R2 R3 R4 -->
- [x] T002 [P] Edit `src/kit/skills/_cli-external.md` § /loop: Usage self-paced-mode note + Constraints rewrite (one-loop rule + pointer; drop Start/Stop/Autopilot-override) <!-- R5 -->
- [x] T003 [P] Sweep `docs/memory/runtime/operator.md` (§ `/loop` lifecycle paragraph, drop the "Autopilot tick interval 2m" Settings row, `/clear` wording at lines ~31/69/383/414, add the two Design Decisions), `docs/memory/_shared/context-loading.md` line ~200, `docs/specs/skills.md` lines ~1102/1111 <!-- R6 R3 -->

### Phase 4: Polish

- [x] T004 Run `fab sync`; grep repo-wide (excl. `.claude/`, `fab/changes/archive/`, `docs/specs/findings/`, `log*.md`) for `default \`2m\``, `default 2m`, `Self-manage context`, and `/clear` in operator context; fix stragglers; run `fab memory-index --check` <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §4 has `### Loop Prompt` with both literals, the MUST/MUST NOT pair with the macro-expansion reason, and the dynamic-mode clause; §9 and §2 Init step 4 point at it without restating
- [x] A-002 R2: §2 Init step 5 shows both ready-line forms carrying the `/loop … "operator tick"` literal; the §4 loop-start sentence repeats the literal
- [x] A-003 R3: no `2m` autopilot cadence remains in `fab-operator.md`; the Autopilot-composition bullet and Tick step 7 name §4 Adaptive cadence as the single owner
- [x] A-004 R4: §1 row reads "Survive compaction" and points at `### Post-Compaction Reload`, which contains trigger, one-shot reload, consumed-tick, durable-state, and `/clear`-is-a-user-action clauses
- [x] A-005 R5: `_cli-external.md` § /loop Constraints has exactly the one-loop bullet and the operator-policy pointer bullet; Usage carries the self-paced-mode pointer
- [x] A-006 R6: `docs/memory/runtime/operator.md`, `docs/memory/_shared/context-loading.md`, and `docs/specs/skills.md` carry the new rule/procedure and no 2m autopilot cadence

### Behavioral Correctness

- [x] A-007 R4: every remaining `/clear` in `fab-operator.md` describes a user action or a durability property — none instructs the agent to `/clear`

### Removal Verification

- [x] A-008 R5: the Start / Stop / Autopilot-override bullets are gone from `_cli-external.md`
- [x] A-009 R3: the "Autopilot tick interval | 2m" row is gone from `docs/memory/runtime/operator.md` § Settings

### Scenario Coverage

- [x] A-010 R1: repo-wide grep (excl. `.claude/`, archive, findings, `log*.md`) for `"operator tick"` finds only bare-token literals — no `/loop … "/fab-operator"` anywhere

### Code Quality

- [x] A-011 Owner-or-pointer: each rule (loop prompt, reload procedure, cadence) is stated once in its owning section and pointed at elsewhere — no second restatement in §9, Init, `_cli-external.md`, or memory beyond present-truth summary
- [x] A-012 Canonical source only: no edits under `.claude/skills/`; `fab sync` regenerates deployed copies
- [x] A-013 No unnecessary duplication: the two Design Decisions in memory are lifted from this plan's four-field entries, not re-authored

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the redundancies this change created (the `_cli-external.md` Start/Stop/Autopilot-override bullets, the memory "Autopilot tick interval 2m" Settings row, the §1 "Self-manage context" row) were planned removals already executed in the diff; no further redundant files/symbols/blocks were discovered.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The memory § Settings "Autopilot tick interval 2m" row is deleted rather than reworded | The skill's §8 Settings table already lacks that row; memory was stale independent of this change, and R3 leaves autopilot with no separate interval | S:85 R:95 A:90 D:85 |
| 2 | Confident | The Init "loop idle" ready-line text is `Operator ready. Loop idle — start with /loop 3m "operator tick" on first enrollment` | Intake Assumption 10 fixed the shape; exact wording is orchestrator's call | S:80 R:95 A:90 D:80 |
| 3 | Confident | Historical `docs/memory/runtime/operator.md` Design Decisions that mention `/clear` (zc9m, operator7 pipeline-first, `»` prefix) keep their `/clear` wording where it describes a user action or historical rationale | FKF present-truth style forbids rewriting past rationale; only present-truth paragraphs are swept | S:80 R:90 A:85 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
