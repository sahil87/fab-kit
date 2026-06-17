# Plan: Operator activates change pointer at spawn for existing changes

**Change**: 260617-5xnx-operator-spawn-activate-pointer
**Intake**: `intake.md`

## Requirements

### Operator Spawn: Pointer Activation

#### R1: Existence-guarded pointer activation in the §6 spawn sequence
The operator's §6 "Spawning an Agent" sequence SHALL set the newly-created worktree's own `.fab-status.yaml` by running `fab change switch <change>` in that worktree's directory, **guarded** so the switch runs only when the change folder already exists (verified via `fab resolve --folder <change>`). The step SHALL sit between worktree creation (step 2) and opening the agent tab, alongside dependency resolution.

- **GIVEN** the operator spawns an agent for a change whose folder/intake already exists (the Existing-change entry form)
- **WHEN** the spawn sequence runs after `wt create`
- **THEN** the operator runs `fab change switch <change>` in the new worktree's CWD, writing that worktree's own `.fab-status.yaml`
- **AND** the activation runs only after `fab resolve --folder <change>` succeeds (the existence guard)

#### R2: Fail-soft activation
A `fab change switch` failure at spawn SHALL be non-fatal — the operator logs one line and continues opening the agent tab. The transient `<change>` override on the embedded pipeline command still resolves the pipeline correctly, so the activation is an ergonomic enhancement, not a correctness prerequisite.

- **GIVEN** the existence-guarded activation step runs
- **WHEN** `fab change switch <change>` exits non-zero
- **THEN** the operator logs one line and proceeds to open the agent tab
- **AND** the spawn is not aborted by the failure

#### R3: No activation for not-yet-existing changes (raw-text / backlog forms)
For spawn paths where the change folder does not exist yet (the raw-text and backlog entry forms, before `/fab-new` runs inside the spawned agent), the operator MUST NOT attempt a switch at spawn. `/fab-new` creates **and activates** the change inside the spawned agent (fab-new Step 10), so activation is owned there.

- **GIVEN** the operator spawns an agent via the raw-text or backlog entry form
- **WHEN** the spawn sequence runs (the change folder does not yet exist)
- **THEN** the existence guard (`fab resolve --folder`) fails and no `fab change switch` is attempted
- **AND** the spawned `/fab-new` creates+activates the change itself once the agent runs

#### R4: No cross-tab collision rationale documented
The skill SHALL state that the activation carries zero cross-tab collision risk because each operator worktree is a dedicated, single-change checkout that owns its own per-worktree `.fab-status.yaml` — the switch writes only that worktree's pointer, never the operator's own checkout or any other worktree.

- **GIVEN** a reader of §6 "Spawning an Agent"
- **WHEN** they read the new activation step
- **THEN** the no-cross-tab-collision rationale (dedicated single-change worktree owns its own `.fab-status.yaml`) is stated

#### R5: Working-a-Change entry-form table notes
The §6 "Working a Change" entry-form table SHALL note that the operator also activates the pointer at spawn on the **Existing change** row (while keeping the statement that the transient `<change>` override — not the pointer — is what targets the pipeline, and keeping the `&&`-no-chaining prohibition UNCHANGED). The raw-text and backlog rows SHALL note that activation is owned by `/fab-new` inside the spawned agent (no operator switch at spawn).

- **GIVEN** a reader of the §6 "Working a Change" entry-form table
- **WHEN** they read the Existing-change row
- **THEN** it notes the operator activates the pointer at spawn AND retains the override-targets-the-pipeline statement AND retains the `&&`-no-chaining prohibition verbatim
- **AND** the raw-text and backlog rows note that `/fab-new` owns activation inside the spawned agent

### SPEC Mirror

#### R6: SPEC §6 / Coordination Patterns description notes pointer activation
Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/SPEC-*.md` file"), `docs/specs/skills/SPEC-fab-operator.md` SHALL note the existence-guarded pointer activation in the spawn-sequence description (the §6 / "Coordination Patterns" Section Structure entry).

- **GIVEN** the SPEC mirror for fab-operator
- **WHEN** the §6 spawn-sequence description is read
- **THEN** it notes the existence-guarded pointer activation (spawn-time `fab change switch` in the new worktree, guarded on folder existence, fail-soft)

#### R7: New Resolved Design Decision #12
The SPEC SHALL add a new **Resolved Design Decision #12** (the list currently ends at #11) capturing: the gap (pointer-less completed worktrees), the chosen fix (activate at spawn when the folder exists), the no-collision rationale (dedicated single-change worktree owns its own `.fab-status.yaml`), the existence guard (raw-text/backlog forms defer to `/fab-new`'s own activation), and the rejected alternative (leave-as-is / rely on the `<change>` arg). It SHALL cross-reference Decision #5 (the `/git-branch`-removal / `&&`-no-chaining lineage).

- **GIVEN** the SPEC's "Resolved Design Decisions" list
- **WHEN** a reader reaches the end of the list
- **THEN** a new Decision #12 is present capturing gap, chosen fix, no-collision rationale, existence guard, and rejected alternative
- **AND** it cross-references Decision #5

### Non-Goals

- No Go/binary change — the fix composes existing verbs `fab resolve --folder` and `fab change switch`.
- No `_cli-fab.md` change — no new or changed command signatures.
- No migration — `.fab-status.yaml` is per-worktree and already created/removed by existing verbs.
- No edit to the deployed `.claude/skills/fab-operator.md` copy (gitignored, regenerated by `fab sync`).
- No `fab sync` run — deployed copy is regenerated separately.

### Design Decisions

1. **Place the activation between `wt create` (step 2) and opening the agent tab**: The pointer must exist before the agent runs so both pipeline resolution and human follow-up see it — *Why*: aligns with where dependency resolution already sits — *Rejected*: setting it after the agent tab opens (a race against the agent's first command).
2. **Use `fab resolve --folder <change>` as the existence guard**: the skill's existing pure-query verb for change resolution; non-zero exit cleanly signals "no such change" — *Why*: reuses an established verb rather than a new probe — *Rejected*: a bespoke folder-existence test (`test -d`), which duplicates resolution logic and ignores archived state.
3. **Fail-soft (log + continue) on switch failure**: the transient `<change>` override still resolves the pipeline; activation is ergonomic — *Why*: matches the skill's fail-soft pattern for window-rename steps — *Rejected*: aborting the spawn on switch failure (over-couples ergonomics to correctness).

## Tasks

### Phase 1: Skill Source (canonical)

- [x] T001 Add the existence-guarded pointer-activation step to the §6 "Spawning an Agent" sequence in `src/kit/skills/fab-operator.md` — insert between step 2 (Create worktree) and the agent-tab step, with the guarded `fab resolve --folder` / `fab change switch` shell snippet, run in the new worktree's CWD, fail-soft, including the no-cross-tab-collision rationale; renumber subsequent steps <!-- R1 R2 R3 R4 -->
- [x] T002 Update the §6 "Working a Change" entry-form table in `src/kit/skills/fab-operator.md`: Existing-change row note reflects spawn-time pointer activation (keep override-targets-pipeline statement + `&&`-no-chaining prohibition unchanged); annotate raw-text and backlog rows that `/fab-new` owns activation inside the spawned agent <!-- R5 -->

### Phase 2: SPEC Mirror

- [x] T003 Update the §6 / "Coordination Patterns" Section Structure entry (item 6) in `docs/specs/skills/SPEC-fab-operator.md` to note the existence-guarded pointer activation in the spawn sequence <!-- R6 -->
- [x] T004 Add Resolved Design Decision #12 to `docs/specs/skills/SPEC-fab-operator.md` capturing the gap, chosen fix, no-collision rationale, existence guard, and rejected alternative; cross-reference Decision #5 <!-- R7 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: The §6 "Spawning an Agent" sequence in `src/kit/skills/fab-operator.md` contains an existence-guarded `fab change switch <change>` step (guarded on `fab resolve --folder <change>`), placed between worktree creation and opening the agent tab, run in the new worktree's CWD
- [ ] A-002 R2: The activation step is documented as fail-soft (log one line and continue) with the rationale that the transient `<change>` override still resolves the pipeline
- [ ] A-003 R3: The skill states that the raw-text/backlog forms do NOT switch at spawn (folder absent → guard fails) and that `/fab-new` owns activation inside the spawned agent
- [ ] A-004 R4: The no-cross-tab-collision rationale (dedicated single-change worktree owns its own `.fab-status.yaml`) is present in the §6 spawn-sequence prose
- [ ] A-005 R5: The §6 "Working a Change" Existing-change row notes spawn-time pointer activation, retains the override-targets-the-pipeline statement, and retains the `&&`-no-chaining prohibition verbatim; raw-text and backlog rows note `/fab-new`-owned activation
- [ ] A-006 R6: The SPEC §6 / "Coordination Patterns" Section Structure entry notes the existence-guarded pointer activation in the spawn sequence
- [ ] A-007 R7: A Resolved Design Decision #12 is present in `docs/specs/skills/SPEC-fab-operator.md` capturing gap, chosen fix, no-collision rationale, existence guard, and rejected alternative, cross-referencing Decision #5

### Behavioral Correctness

- [ ] A-008 R5: The `&&`-chaining prohibition on the Existing-change row is byte-for-byte unchanged from before the edit (the fix is a separate spawn step, not a chained slash command)

### Edge Cases & Error Handling

- [ ] A-009 R2: The skill's prose makes clear a switch failure does not abort the spawn (the agent tab still opens)
- [ ] A-010 R3: The existence guard prevents any `fab change switch` attempt when the change folder does not exist yet

### Code Quality

- [ ] A-011 Pattern consistency: New prose matches the surrounding operator-skill style — numbered spawn-sequence steps, shell-snippet fencing, and table formatting; the SPEC Decision #12 matches the style/format of Decisions #1–#11
- [ ] A-012 No unnecessary duplication: The fix composes existing verbs (`fab resolve --folder`, `fab change switch`) rather than introducing new ones; no `_cli-fab.md` change

### Documentation Accuracy

- [ ] A-013 Documentation accuracy: Only `src/kit/skills/fab-operator.md` (canonical) is edited — the deployed `.claude/skills/fab-operator.md` is NOT hand-edited; no Go/binary, `_cli-fab.md`, or migration files are created

### Cross References

- [ ] A-014 Cross references: SPEC Decision #12 cross-references Decision #5; the skill and SPEC descriptions are internally consistent with each other and with the intake

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Place the activation step as a new step 3 (after Create worktree, before Resolve dependencies) and renumber the existing steps 3–6 to 4–7 | Intake says "between worktree creation and opening the agent tab, alongside dependency resolution"; making it the immediate post-`wt create` step keeps the pointer set before any dependency work and before the tab opens; renumbering is mechanical | S:80 R:85 A:85 D:75 |
| 2 | Confident | Render the guarded activation as a fenced `sh` snippet matching the intake's snippet verbatim plus surrounding prose bullets | The intake supplies the exact snippet; the skill already uses fenced `sh`/`bash` snippets for spawn-sequence shell (e.g. step 5's `tmux new-window`), so this matches house style | S:85 R:85 A:90 D:85 |
| 3 | Confident | Phrase the raw-text/backlog table note as a short clause appended to each row's note column rather than a new table column | Keeps the existing 3-column table shape; the intake only asks for a note, not structural change | S:80 R:85 A:85 D:80 |

3 assumptions (0 certain, 3 confident, 0 tentative).
