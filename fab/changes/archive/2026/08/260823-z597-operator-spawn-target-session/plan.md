# Plan: Operator Spawn Target-Session Derivation

**Change**: 260823-z597-operator-spawn-target-session
**Intake**: `intake.md`

## Requirements

### Operator Skill: Target-Session Derivation

#### R1: §6 gains an "Establish target session" step with a fallback ladder and a never-ambient rule
`src/kit/skills/fab-operator.md` §6 Spawning an Agent SHALL gain a new step 2, **Establish target session**, directly after step 1 (Establish target repo), renumbering current steps 2–7 → 3–8. The step SHALL derive the session via a fallback ladder: (a) live derivation — the session holding existing monitored agents for the target repo, re-verified from the current tick snapshot / `fab pane map --all-sessions` (per §1 Re-derive-state; the persisted `session` field is context-not-identity and MUST NOT be trusted stale); (b) the session holding any monitored agent (the single-work-session case — applies when exactly one candidate session results; multiple candidates with none matching the target repo fall through); (c) the §8 "Spawn target session" setting when the user set one; (d) cold start — ask the user once and keep the answer as the §8 setting for the session. On an **unattended** spawn (watch/autopilot tick) with no signal at rungs (a)–(c), the operator SHALL escalate via the §5 notification path instead of asking or guessing. The step SHALL state the normative rule: the operator MUST pass `-t "<session>:"` on every `new-window`; **the ambient session is never an implicit target** — the exact mirror of the never-rely-on-CWD rule. All step-number references inside `fab-operator.md` SHALL be updated consistently (the renumber is not complete until every internal citation agrees).

- **GIVEN** the operator runs in its own dedicated tmux session and the work windows live in a different session
- **WHEN** any spawn path (worker tab, watch spawn, autopilot spawn) reaches the open-agent-tab step
- **THEN** the spawn command carries an explicit `-t "<session>:"` derived by the ladder, and the window lands in the derived work session, never the operator's ambient session

- **GIVEN** a cold start — no monitored agents anywhere, no §8 setting
- **WHEN** a user-initiated spawn needs a target session
- **THEN** the operator asks the user once and records the answer as the §8 session-scoped setting; an unattended (watch/autopilot) spawn in the same state escalates via §5 instead

#### R2: The open-agent-tab command is hardened with `-t` and `-P -F`, feeding enrollment
§6's open-agent-tab step SHALL become:

```sh
tmux new-window -t "<session>:" -P -F '#{session_name} #{pane_id}' -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"
```

and the enrollment step SHALL consume the printed `#{session_name}` / `#{pane_id}` values directly as `fab operator enroll --pane <pane-id> --session <name>` — closing the previously unspecified how-does-enrollment-learn-the-pane gap and confirming where the window actually landed (a dead/absent `-t` session errors loudly at spawn, surfaced per the operator's normal error handling).

- **GIVEN** a spawn whose `tmux new-window` succeeds
- **WHEN** the enrollment step runs
- **THEN** `--pane`/`--session` come from the spawn command's own printed output, not from a post-spawn pane-map scrape, and the printed session confirms placement

#### R3: `_cli-agents.md` § Spawn Composition carries a mechanics-only ambient-session caveat
`src/kit/skills/_cli-agents.md` § Spawn Composition SHALL add a short caveat to the raw form's bullet list: `new-window` without `-t` targets the **ambient** session (the session of the pane running the command); a caller whose session differs from where the window should land must pass `-t '<session>:'`. The caveat SHALL carry no operator policy — the fallback ladder stays in `fab-operator.md` per the file's Scope Boundary and the owner-or-pointer rule.

- **GIVEN** a non-operator consumer reading § Spawn Composition's raw form
- **WHEN** it spawns from a session other than the intended landing session
- **THEN** the caveat tells it the ambient-resolution fact and the `-t '<session>:'` mechanics, without restating any operator ladder

#### R4: §8 Settings gains a "Spawn target session" row
`fab-operator.md` §8 Settings SHALL gain a row for the spawn-session setting — session-scoped, natural-language overridable ("spawn into session {name}"), default derived per §6's ladder — matching the shape of the existing four rows (resets on `/clear`/restart; not an operator-state-file field).

- **GIVEN** the user says "spawn into session work"
- **WHEN** a later spawn reaches ladder rung (c)
- **THEN** the setting supplies the target session for the rest of the operator session

#### R5: Sibling-sweep consistency across `_cli-external.md` and verified non-targets
`src/kit/skills/_cli-external.md` SHALL be swept: the § tmux `new-window` table row (~line 178) and the spawn usage-note bullet (~line 185) MUST NOT contradict the `-t` rule (the row gains the optional target, the bullet keeps pointing at the owners), and the "`fab-operator.md` §6 step 5" citation (~line 118) MUST be updated for the renumber (step 5 → step 6). The sweep SHALL verify: `docs/specs/skills.md` still restates no spawn command; the dispatch-adapter docs (`docs/specs/harness-adapters.md`, `docs/memory/runtime/dispatch.md`) and the Go spawn sites are untouched (deliberate non-targets — the dispatch adapter's ambient/`$TMUX_PANE` resolution is correct by design). Memory files (`docs/memory/runtime/operator.md`, `agent-primitives.md`) are hydrate-stage, not apply-stage.

- **GIVEN** the renumber and the new `-t` rule have landed in `fab-operator.md`
- **WHEN** the repo is grepped for `new-window` and §6 step-number citations across `src/kit/skills/`
- **THEN** every skill-file occurrence is consistent with the new numbering and the `-t` rule, and the deliberate non-targets carry no edits

### Non-Goals

- Go-side changes — `fab pane open`'s new-window shape (`src/go/fab/internal/pane/create.go`), `fab batch new`/`switch` (`batch_new.go:141`, `batch_switch.go:141`) share the ambient assumption but are a backlog-idea follow-up; `fab operator`'s launcher (`operator.go:106`) is deliberately ambient.
- Any `.claude/skills/` edit (gitignored deployed copies).
- Memory-file edits at apply (they land at hydrate).

### Design Decisions

#### Target session is derived live per spawn, never persisted-and-trusted
**Decision**: A 4-rung ladder derives the session at spawn time — live fleet derivation for the target repo, any-monitored-agent session, the §8 setting, then ask-once (user-initiated) / escalate (unattended) — and the operator MUST pass `-t "<session>:"` on every `new-window`.
**Why**: The skill's own doctrine says `session` is context-not-identity (reassigned by `move-window`/rename, changeable mid-lifetime) and §1 Re-derive-state forbids trusting stale values; live derivation is the only option consistent with both, and it handles multi-repo naturally (each repo's work clusters where it already lives). The ambient session silently misplacing windows is the observed failure — hence never-ambient as an absolute.
**Rejected**: Deriving from the persisted monitored/`branch_map` `session` field (stale-prone, structurally absent for fresh requests — `branch_map` carries no session); a config-only default work session (single value fights the multi-repo model; the operator may run with no resolvable `fab/` project); ambient-as-implicit-default (the bug itself).
*Introduced by*: 260823-z597-operator-spawn-target-session

#### The spawn command prints its own landing via `-P -F`, feeding enrollment
**Decision**: `tmux new-window -P -F '#{session_name} #{pane_id}'` returns the landed session and pane; enrollment consumes those values directly.
**Why**: The skill never said how enrollment learns `--pane`/`--session` post-spawn (a latent gap), and the print-back verifies placement in the same call — converting the silent-misplacement class into a checked output. fab's own Go spawn path already uses `-P -F '#{pane_id}'` (`internal/pane/create.go`), so the idiom is established.
**Rejected**: Post-spawn `fab pane map` scraping (racy, and still unspecified); a compare-and-escalate branch on the printed session (unreachable once `-t` is passed — tmux errors loudly on a missing session, so verification-feed is sufficient).
*Introduced by*: 260823-z597-operator-spawn-target-session

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/kit/skills/fab-operator.md` §6 Spawning an Agent: insert new step 2 "Establish target session" (ladder rungs a–d, unattended-escalation note, the MUST-pass `-t "<session>:"` / never-ambient rule), renumber current steps 2–7 → 3–8, and update every internal step-number reference in the file — the §6 intro + "Working a Change" chain line (~564: add target-session to the sequence; ~566 "spawn step 3" → 4), Dependency Declaration (~556 "step 7" → 8), Autopilot (~640 "§6 step 6" → 7; ~643 "steps 1–2" → 1–3; ~644 "steps 3–7" → 4–8 and "Step 6's `<command>`" → "Step 7's"), and the open-tab step's own internal refs ("step 2" → 3 for the worktree name, "step 5" → 6 for the session command) <!-- R1 -->
- [x] T002 In `src/kit/skills/fab-operator.md` §6: replace the open-agent-tab command with the hardened `-t "<session>:" -P -F '#{session_name} #{pane_id}'` form and rewrite the enrollment step to consume the printed values as `fab operator enroll --pane/--session` <!-- R2 -->
- [x] T003 In `src/kit/skills/fab-operator.md` §8 Settings: add the "Spawn target session" row (default: derived per §6 ladder; override: "spawn into session {name}") <!-- R4 -->
- [x] T004 [P] In `src/kit/skills/_cli-agents.md` § Spawn Composition: add the mechanics-only ambient-session caveat bullet to the raw form's bullet list (`-t '<session>:'` when the caller's session differs; no policy) <!-- R3 -->
- [x] T005 In `src/kit/skills/_cli-external.md`: update the § tmux `new-window` row (~178) and spawn bullet (~185) to be `-t`-consistent, fix the "§6 step 5" citation (~118) → step 6; then run the repo-wide verification grep (`new-window` + §6 step-number citations across `src/kit/skills/`; confirm `docs/specs/skills.md` clean and `harness-adapters.md`/`runtime/dispatch.md`/Go files untouched) <!-- R5 -->

## Execution Order

- T001 blocks T002, T003, and T005 (they reference the renumbered steps); T004 is independent.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab-operator.md` §6 contains an "Establish target session" step 2 with the 4-rung ladder, the unattended-escalation note, and the MUST-pass `-t "<session>:"` / never-ambient rule; steps run 1–8 with no numbering gaps
- [x] A-002 R2: the §6 open-agent-tab command carries `-t "<session>:"` and `-P -F '#{session_name} #{pane_id}'`, and the enrollment step names the printed values as the source of `--pane`/`--session`
- [x] A-003 R3: `_cli-agents.md` § Spawn Composition's raw-form bullets include the ambient-session `-t` caveat, with no ladder/policy content
- [x] A-004 R4: §8 Settings has the "Spawn target session" row in the existing table shape
- [x] A-005 R5: `_cli-external.md` lines ~118/~178/~185 are consistent with the renumber and the `-t` rule

### Behavioral Correctness

- [x] A-006 R1: every internal step-number reference in `fab-operator.md` (Working a Change ~564/566, Dependency Declaration ~556, Autopilot ~640/643/644, the open-tab step's internal "step 2"/"step 5" refs) agrees with the new 1–8 numbering
- [x] A-007 R1: the ladder's policy text lives only in `fab-operator.md` — neither `_cli-agents.md` nor `_cli-external.md` restates any rung (owner-or-pointer)

### Scenario Coverage

- [x] A-008 R1: the cold-start rung (d) distinguishes user-initiated (ask once, persist as §8 setting) from unattended watch/autopilot spawns (escalate via §5, never guess, never ambient)

### Edge Cases & Error Handling

- [x] A-009 R2: the text notes that a missing/dead `-t` target errors loudly at spawn (tmux refuses), surfaced per normal operator error handling — no silent fallback to ambient
- [x] A-010 R5: deliberate non-targets are untouched: `docs/specs/harness-adapters.md`, `docs/memory/runtime/dispatch.md`, all `src/go/` files, `.claude/skills/`

### Code Quality

- [x] A-011 Pattern consistency: the new step, settings row, and caveat match the surrounding file's voice, table shapes, and section conventions
- [x] A-012 No unnecessary duplication: no owned rule is restated alongside a pointer (code-quality.md anti-pattern); the sweep covered the whole sibling class up front
- [x] A-013 Canonical source only: all edits under `src/kit/skills/`; zero diffs under `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change inserts a new step, hardens the spawn command in place, and renumbers references; it makes no existing prose, symbol, or section redundant

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Task granularity: the step insert + renumber + internal-reference sweep is one atomic work unit (T001) — the renumber is incomplete until every citation agrees, so splitting it would create a broken intermediate state; 5 tasks total | Natural edit units per file/section; keeps each task one focused session | S:70 R:85 A:85 D:70 |
| 2 | Confident | `_cli-external.md` § tmux row gains an optional `[-t "<session>:"]` rather than a mandatory one — the row is a generic tmux reference serving non-operator consumers whose ambient session is often correct | The MUST rule is operator policy (owned by fab-operator.md); the generic row only needs to stop contradicting it | S:65 R:85 A:80 D:70 |
| 3 | Confident | §8 row wording: Setting "Spawn target session", Default "derived (§6 ladder)", Override "spawn into session {name}" | Matches the existing table's three-column shape; exact words not discussed | S:60 R:90 A:85 D:75 |

3 assumptions (0 certain, 3 confident, 0 tentative).
