# Plan: Operator Canonical Pane Capture

**Change**: 260819-zs1u-operator-canonical-pane-capture
**Intake**: `intake.md`

## Requirements

### Skills: Operator question-detection capture

#### R1: §5 step 1 uses the canonical fab capture
`src/kit/skills/fab-operator.md` §5 Question Detection step 1 SHALL name `fab pane capture --raw -l 20 [-L <server>] <pane>` as the capture command, replacing the raw `tmux capture-pane -t <pane> -p -S -20`, with a parenthetical noting `-L <server>` applies only on a non-default tmux socket (the §9 second-operator case).

- **GIVEN** an operator executing §5 Question Detection on a monitored pane
- **WHEN** it reads step 1
- **THEN** the instructed command is `fab pane capture --raw -l 20 [-L <server>] <pane>`
- **AND** no raw `tmux capture-pane` invocation remains anywhere in `fab-operator.md`

#### R2: The §5 re-capture guard names the same command
The re-capture guard in § Sending Auto-Answers (currently "then re-capture the terminal", referenced by the § Non-Blocking Strategic Handling decision-table rows) SHALL explicitly reuse the § Question Detection step 1 capture command, so the before/after comparison is transform-symmetric and no reader falls back to raw tmux for the guard.

- **GIVEN** an operator about to deliver an auto-answer
- **WHEN** it runs the re-capture guard
- **THEN** the guard text names the same `fab pane capture --raw -l 20` capture as step 1

#### R3: Memory mirrors reflect the new capture command
`docs/memory/runtime/operator.md` SHALL mirror the change as present truth: the § Auto-Nudge question-detection step 1 (line 177) names the new command, and the § Re-capture before send paragraph (line 211) references the same capture. Memory index regenerated only if a `description:` frontmatter change makes it drift.

- **GIVEN** the memory file documenting §5 behavior
- **WHEN** hydrate completes
- **THEN** `operator.md` question-detection step 1 reads `fab pane capture --raw -l 20 [-L <server>] <pane>` and no longer instructs `tmux capture-pane -t <pane> -p -S -20`

#### R4: No live raw-capture instruction survives the sweep
After the edits, no **live instruction** in `src/kit/skills/` or `docs/memory/` SHALL tell an operator/agent to run raw `tmux capture-pane` for pane peeking. By-design raw tmux (pane spawn via `tmux new-window`, the rk-absent `send-keys` fallback, pre-delivery judgment-round `send-keys`) and historical records (`log.seed.md`, `docs/specs/findings/*`, Design-Decision Rejected/Why prose) are exempt.

- **GIVEN** the completed edits
- **WHEN** `grep -rn "capture-pane" src/kit/skills/ docs/memory/` runs (excluding `_cli-fab.md`'s mechanics documentation)
- **THEN** every remaining hit is an exempt category above

### Non-Goals

- No Go/binary change — `fab pane capture`'s surface is already correct post-y4mu (cherry-pick `ef9a0159` on this branch).
- No `_cli-agents.md` § Peek edit — already rewritten by y4mu.
- No fab-absent fallback text — `fab operator` guarantees fab's presence; y4mu's accepted § Peek resolution was drop-the-aside.
- No edits to historical records (log.seed.md, findings docs).

### Design Decisions

#### Detection capture uses `--raw`, not enriched output
**Decision**: §5's capture is `fab pane capture --raw -l 20`, the bare captured text.
**Why**: The step-4 detection patterns scan raw terminal lines (last-2-lines boundary guard, last-non-empty-line `?` check, lines ending `:`); the enriched default header would inject non-terminal lines (`agent: waiting`, `stage: …`) into the scanned window and false-positive the `:`-prompt indicator. Agent state reaches the answer model separately via the tick's `fab pane map` read and the `waiting` primary signal.
**Rejected**: Default enriched capture (header pollutes the pattern window) and `--json` (the patterns operate on visible text, not fields; parsing JSON per pane per tick adds ceremony with no detection gain).
*Introduced by*: 260819-zs1u-operator-canonical-pane-capture

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §5 Question Detection step 1 (line 333): `tmux capture-pane -t <pane> -p -S -20` → `fab pane capture --raw -l 20 [-L <server>] <pane>` with the non-default-socket parenthetical <!-- R1 -->
- [x] T002 Rewrite the re-capture guard in `src/kit/skills/fab-operator.md` § Sending Auto-Answers (line 389) to name the same capture command as step 1 <!-- R2 -->
- [x] T003 [P] Update `docs/memory/runtime/operator.md` present-truth mirrors: question-detection step 1 (line 177) and § Re-capture before send (line 211); run `fab memory-index --check` and regenerate if drift (check exits 0 — only pre-existing size-cap warnings; frontmatter descriptions unchanged, no regen needed) <!-- R3 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Sweep: `grep -rn "capture-pane\|-S -20" src/kit/ docs/memory/` — verified: remaining hits are the `_cli-external.md:182` canon rule itself, log.seed/log historical entries, kit-architecture's description of `_cli-external.md`, pane-commands mechanics (internal tmux fetch), and DD Rejected/Why prose (agent-primitives:125, operator:446, dispatch:784) — all exempt, none a live raw-capture instruction <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §5 step 1 instructs `fab pane capture --raw -l 20 [-L <server>] <pane>`; zero raw `tmux capture-pane` occurrences remain in the file
- [x] A-002 R2: the § Sending Auto-Answers re-capture guard explicitly names the same `fab pane capture --raw -l 20` capture as step 1
- [x] A-003 R3: `docs/memory/runtime/operator.md` step 1 mirror and re-capture-guard mirror updated to the new command; `fab memory-index --check` clean (exit 0 — pre-existing size/narration warnings only, no drift)

### Behavioral Correctness

- [x] A-004 R1: the replacement command is valid against the current `fab pane capture` surface (`--raw`, `-l`, `-L/--server` all exist; `-l 20` = last-20-lines per y4mu, cherry-picked on this branch)

### Scenario Coverage

- [x] A-005 R4: sweep grep output confirms only exempt hits remain (spawn `new-window`, rk-absent `send-keys`, `_cli-fab.md`/pane-commands mechanics, log.seed.md, findings docs, DD Rejected/Why prose)

### Code Quality

- [x] A-006 Owner-or-pointer: the edited §5 step states the command it owns; no restatement of `_cli-agents.md` § Peek or `_cli-fab.md` § capture content is introduced alongside a pointer
- [x] A-007 Canonical source only: edits land in `src/kit/skills/` and `docs/memory/`, never `.claude/skills/`
- [x] A-008 Pattern consistency: the `[-L <server>]` conditional mirrors the existing §9 non-default-socket phrasing (`rk mux -L <label> send` precedent)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change swaps two capture-command references (skill + memory mirror) in place without making any existing code, symbol, or block redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | The memory re-capture-guard paragraph (operator.md:211) is also updated, though the intake's Affected Memory named only the step-1 mirror | The guard mirror restates §5 behavior — same sibling-sweep class; skipping it would leave the memory internally inconsistent with its own step 1 | S:70 R:90 A:85 D:80 |
| 2 | Confident | The skill text is not gated on the y4mu release reaching users' installed binaries | `-l`/`--raw`/`-L` have existed since tam1 — the command runs on older binaries with a looser window (extra lines), degrading gracefully; this branch carries y4mu, and both ship forward together | S:65 R:80 A:85 D:75 |

2 assumptions (0 certain, 2 confident, 0 tentative).
