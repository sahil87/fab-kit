# Plan: Operator Spawn Target-Session Selector — Deterministic, Never Asks

**Change**: 260913-iyb4-operator-spawn-session-selector
**Intake**: `intake.md`

## Requirements

### Operator skill: target-session selection (`src/kit/skills/fab-operator.md` §6 step 2)

#### R1: Ordered selector replaces the pane-count majority rule
§6 "Spawning an Agent" step 2 MUST select the target session by an ordered rung list evaluated over the unchanged candidate set (`rk mux sessions --json` `role: "user"` rows minus the operator's own session). The first rung that decides wins, and the list MUST NOT be able to come up empty for a non-empty candidate set: (a) the §8 "Spawn target session" setting when the **user** set it; (b) exactly one candidate; (c) the candidate whose `path` equals the target repo's main-worktree root (step 1's value); (d) the candidate holding the most panes whose `repo` (per `fab pane map --all-sessions --json`) is the target repo; (e) the highest `attached`, residual tie → the first candidate in `rk mux sessions` order. Step 2 MUST contain no ask, notify-when-torn, or auto-set clause. The sentence "Never trust a persisted `scope.session` alone; the ambient session is never an implicit target" MUST be retained.

- **GIVEN** one `role: "user"` session on the server and a target repo with nothing open in it
- **WHEN** the operator reaches step 2
- **THEN** rung (b) decides that session and no question is asked

- **GIVEN** two user sessions, zero panes in the target repo, and one session whose `path` is the target repo's root
- **WHEN** the operator reaches step 2
- **THEN** rung (c) decides that session

- **GIVEN** two user sessions with no `path` match and no target-repo panes
- **WHEN** the operator reaches step 2
- **THEN** rung (e) decides deterministically (highest `attached`, then first row) — never a tie declared torn

#### R2: Zero candidates is a loud error, not a question
When the candidate set is empty (no `role: "user"` session on the server), step 2 MUST surface a "nowhere to spawn" error and the spawn MUST NOT be attempted. Attended, the error lands in the spawn output; unattended (queue / Linear-Slack tick), it rides the §5 Notification Send path.

- **GIVEN** `rk mux sessions --json` returns no `role: "user"` row
- **WHEN** a spawn is requested
- **THEN** the operator reports the error and does not run `rk tab new`

#### R3: One-line announce
On every deterministic pick, step 2 MUST announce exactly one line naming the chosen session and the deciding rung (spelling is apply's call), and MUST NOT send a §5 notification for the pick.

- **GIVEN** rung (c) decided `fab_kit`
- **WHEN** the spawn proceeds
- **THEN** the spawn output carries one line such as `→ session fab_kit (rung c: path = /home/x/code/fab-kit)`

#### R4: §1 question licence removed
The §1 "Coordinate, don't execute" row MUST NOT contain the parenthetical `(plus the target-session tie-break in §6 Spawning an Agent step 2)`; the sentence reads "The only thing the operator asks about a work request **or a spawn** is **which repo** — never whether to spawn, …". Nothing else in the row changes.

- **GIVEN** the edited §1 table
- **WHEN** the row is read
- **THEN** "which repo" is the sole licensed question and no session tie-break is mentioned

#### R5: §8 row becomes a pure user override
The §8 Settings row for "Spawn target session" MUST state that the default is none (§6 step 2's selector decides) and that the setting is set only by the user (rung a); the override phrase `"spawn into session {name}"` and the session-scoped footnote stay unchanged. No new validation prose (an unknown session already errors loudly at spawn).

- **GIVEN** the edited §8 table
- **WHEN** the row is read
- **THEN** it carries no "inferred", "majority rule", or "auto-set" wording

#### R6: Net shorter
After apply, the step-2 list item MUST be ≤ 123 words and `wc -w src/kit/skills/fab-operator.md` MUST be < 14,604 (baselines measured at `c57f6584`).

- **GIVEN** the edited skill
- **WHEN** `wc -w` runs on the file and on the step-2 item
- **THEN** both counts are below their baselines

#### R7: Sibling sweep within the skill set
Every restatement of the old rule in `src/kit/skills/` MUST be updated or verified: the cross-reference pointers at § Working a Change ("§6's target-repo + target-session → …") and § Queues step 2 ("establish the change's target repo and target session") stay as pointers; a repo-wide grep for `majority rule`, `genuinely torn`, `tie-break`, `auto-set`, `structural dominance`, `is under the target repo` over `src/kit/` MUST return no operator-skill hits (`fab-new.md:49` "Tie-breaker (default-closed)" is the unrelated micro-change backstop and stays). Deployed content MUST cite no fab-kit-only paths (Constitution V; `go test ./src/go/fab-kit/cmd/fab/` stays green).

- **GIVEN** the completed apply
- **WHEN** the grep runs over `src/kit/`
- **THEN** the only hit is `fab-new.md:49`

### Memory (hydrate)

#### R8: Memory rewritten to present truth with the corrected rationale
At hydrate — never at apply — `docs/memory/runtime/operator.md` MUST be rewritten in place at: § Principles line ~21 (drop the tie-break parenthetical); § Repo- and session-targeted spawning item 1 (the five-rung selector, zero-candidate error, one-line announce; delete tie/ask/auto-set/"genuinely torn"; keep the `rk tab new --session =<session>` pin, "the ambient session is never an implicit target", and "an unknown session errors loudly at spawn"); § Settings row (as R5); and the Design Decision "Spawn Target Session Is Inferred Live Per Spawn by Majority Rule, Never Persisted-and-Trusted" (retitled to selection-not-inference, Decision/Why/Rejected corrected, the false "decides every observed case" clause replaced by the corrected record, trail extended with `260913-iyb4-operator-spawn-session-selector`). `docs/memory/runtime/agent-primitives.md` § Spawn composition pointer wording "majority rule" → "session selector". `fab status set-summary` mentions the run-kit `rk tab new` default-session follow-up. `docs/memory/runtime/log.md` is not edited by hand.

- **GIVEN** the hydrate stage
- **WHEN** the memory files are re-read
- **THEN** no sentence claims the pane-count majority "decides every observed case", and the DD trail records z597 → cx52 → 4a8m (regression) → iyb4

### Non-Goals

- No run-kit changes — the `rk tab new` default-session idea is recorded as a follow-up only (intake § G); not filed in run-kit's backlog by this change.
- No Go changes, no migration, no CLI reference changes, no persistent config key (`operator.spawn_session` stays rejected).
- No edits to `.agents/skills/` or `.claude/skills/` deployed copies.

### Design Decisions

#### Target Session Is Selected by an Ordered, Non-Empty Rung List — Never Asked
**Decision**: §6 step 2 selects the spawn target session by an ordered rung list — user override → sole candidate → session `path` equals the target repo root → most target-repo panes → highest `attached` (then row order) — over `rk mux sessions --json` `role: "user"` rows minus the operator's own session. Zero candidates is a loud "nowhere to spawn" error. The operator announces the pick in one line and never asks.
**Why**: The pane-count-only rule ties at zero on the most common spawn (a repo with nothing open yet), and its tie-break — the §8 setting — resets on every compaction, so the "one ask" recurred per compaction. `rk mux sessions` already supplies `path` and `attached`, which are non-empty exactly when pane counts are not; a sole candidate needs no evidence at all. A wrong landing is cheap to correct (`move-window`; `rk tab new --json` confirms the landing), so asking has negative value.
**Rejected**: Keeping the ask as a tie-break (recurs; fires when the answer is obvious — cx52's own finding); the pane-count majority alone (4a8m's collapse regressed cx52 — its memory rationale "decides every observed case" was false when written, cx52's reproduction being a zero-count case); adding tiers on top of the majority rule (prose growth, vetoed); a persistent `operator.spawn_session` key (staleness trap); run-kit default first (changes nothing while the skill pins `--session`; needs an rk release plus a fab version gate).
*Introduced by*: 260913-iyb4-operator-spawn-session-selector

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §6 step 2 (the "Establish target session" list item): unchanged candidate set with a shortened envelope note; rungs a–e as a compact lettered list; zero-candidates loud error (attended → spawn output, unattended → §5); one-line announce; keep the "Never trust a persisted `scope.session` alone; the ambient session is never an implicit target" sentence; delete the `rk mux panes` `cwd` source clause, the tie/ask/notify clause, and "auto-set §8"; ≤ 123 words <!-- R1, R2, R3, R6 -->
- [x] T002 [P] Edit `src/kit/skills/fab-operator.md` §1 "Coordinate, don't execute" row: delete `(plus the target-session tie-break in §6 Spawning an Agent step 2)`; edit the §8 Settings "Spawn target session" row to `none — §6 step 2's selector decides; set only by the user (rung a)` with the override phrase unchanged <!-- R4, R5 -->

### Phase 3: Integration & Edge Cases

- [x] T003 Sweep + verify in `src/kit/skills/`: grep `majority rule|genuinely torn|tie-break|auto-set|structural dominance|is under the target repo` (only `fab-new.md:49` may remain); confirm § Working a Change and § Queues step 2 pointers need no rewording; `wc -w` step-2 item ≤ 123 and file < 14,604; `go test ./src/go/fab-kit/cmd/fab/` passes; `fab sync` refreshes deployed copies without error; list the memory sites for hydrate in `## Notes` (no `docs/memory/` edit at apply) <!-- R6, R7, R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Step 2 lists rungs a–e in the decided order over `role: "user"` rows minus the operator's own session, states that the first deciding rung wins, and contains no ask, notify-when-torn, or auto-set clause
- [x] A-002 R2: Step 2 states that zero candidates is a "nowhere to spawn" error, the spawn is not attempted, and the unattended path is §5
- [x] A-003 R3: Step 2 requires a one-line announce naming session + deciding rung, with no §5 notification for a pick
- [x] A-004 R4: The §1 row's ask sentence names only "which repo" and the tie-break parenthetical is gone; the rest of the row is byte-identical
- [x] A-005 R5: The §8 row reads as a pure user override (no "inferred" / "majority rule" / "auto-set"); the override phrase and the session-scoped footnote are unchanged
- [x] A-006 R8: No `docs/memory/` file was modified at apply, and `## Notes` enumerates the hydrate sites (`operator.md` § Principles, § Repo- and session-targeted spawning item 1, § Settings row, the Design Decision block; `agent-primitives.md` § Spawn composition pointer)

### Behavioral Correctness

- [x] A-007 R1: Reading step 2 as the operator model, the screenshot scenario (one user session, target change only on origin, zero target-repo panes) resolves at rung (b) without a question
- [x] A-008 R1: Rung (d) joins on `fab pane map --all-sessions --json` `repo` (main-worktree root equality), not on `cwd` prefix — the `rk mux panes` `cwd` clause is gone
- [x] A-009 R1: The z597 invariant sentence ("Never trust a persisted `scope.session` alone; the ambient session is never an implicit target") is present verbatim

### Removal Verification

- [x] A-010 R7: `grep -n 'majority rule\|genuinely torn\|tie-break\|auto-set\|structural dominance\|is under the target repo' src/kit/skills/fab-operator.md` returns zero hits, and the same grep over `src/kit/` returns only `fab-new.md:49`

### Scenario Coverage

- [x] A-011 R1: The three R1 scenarios (sole session → b; path match with zero panes → c; no path/pane evidence → e with row-order tiebreak) are each derivable from the step-2 text without inference beyond the listed rungs

### Edge Cases & Error Handling

- [x] A-012 R2: An empty candidate set is the only failure path and is described as an error, never as a question, in both attended and unattended modes
- [x] A-013 R6: `wc -w` on the step-2 item ≤ 123 and on `src/kit/skills/fab-operator.md` < 14,604

### Code Quality

- [x] A-014 Pattern consistency: Edits follow the skill's existing §-reference, bold-lead, and owner-or-pointer style; the selector is defined once in step 2 and pointed at from §1, §8, § Working a Change, and § Queues
- [x] A-015 No unnecessary duplication: No site restates the rung list alongside its pointer; the memory restatements are left for hydrate
- [x] A-016 Canonical source only: Edits land in `src/kit/skills/`, never `.agents/skills/` or `.claude/skills/`
- [x] A-017 Deployed content cites no fab-kit-only paths (Constitution V; `go test ./src/go/fab-kit/cmd/fab/` passes)
- [x] A-018 Sibling sweep: Every skill-set restatement of the old rule was updated or verified as a pointer before apply finished (`fab/project/code-quality.md` § Sibling Sweeps)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **Hydrate checklist (R8)** — `docs/memory/runtime/operator.md`: § Principles "Coordinate, don't execute" (drop the tie-break parenthetical); § Repo- and session-targeted spawning item 1 → five-rung selector + zero-candidate error + one-line announce; § Settings row → pure user override; Design Decision "Spawn Target Session Is Inferred Live Per Spawn by Majority Rule, Never Persisted-and-Trusted" → retitle, correct Decision/Why/Rejected (replace the false "decides every observed case" clause), extend the trail with iyb4; the neighbouring "Spawn-Target Candidacy Delegates to `rk mux sessions`" DD may note that `path`/`attached` now decide rungs c/e. `docs/memory/runtime/agent-primitives.md` § Spawn composition: "the operator's majority rule lives in `fab-operator.md` §6" → "the operator's session selector lives in `fab-operator.md` §6". `fab status set-summary` names the run-kit `rk tab new` default-session follow-up. `log.md` untouched.

## Deletion Candidates

- None — the change rewrites three prose sites in place; the now-stale memory restatements (`docs/memory/runtime/operator.md`, `agent-primitives.md`) are planned hydrate rewrites (plan R8 / A-006), not deletion candidates

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Rungs are rendered as a compact lettered inline list inside the step-2 item, not a markdown table | A table's header/separator rows spend words the ≤123 budget cannot afford; the intake left presentation to apply | S:70 R:95 A:80 D:70 |
| 2 | Confident | Rung (c) "equals" is read as path equality after trailing-slash normalization; no symlink-resolution prose is added | Adding normalization prose grows the item; `rk mux sessions` prints the start path as tmux stores it, matching step 1's absolute root in every observed case | S:60 R:90 A:75 D:60 |
| 3 | Confident | The run-kit envelope aside is kept in the short form "(the `result` array in run-kit's envelope form)"; the "same holds for every `rk … --json` read" tail is dropped only if the word budget requires it | Intake assumption #18 — the aside is the skill's only general envelope note | S:55 R:95 A:70 D:55 |
| 4 | Confident | The announce line's example spelling is `→ session <name> (rung <x>: <evidence>)`; the contract is one line, session + rung | Intake assumption #13 left spelling to apply | S:70 R:90 A:80 D:70 |
| 5 | Certain | Memory edits are hydrate's; apply touches only `src/kit/skills/fab-operator.md` and lists the sites in `## Notes` | Pipeline convention; intake assumption #10 | S:90 R:95 A:90 D:90 |

5 assumptions (1 certain, 4 confident, 0 tentative).
