# Plan: Operator principles — read plans, never execute, always spawn

**Change**: 260911-kp3d-operator-read-plans-never-execute
**Intake**: `intake.md`

## Requirements

### Runtime: Operator §1 principle rows

#### R1: The read-prohibition row is deleted outright
`src/kit/skills/fab-operator.md` §1 Principles MUST NOT contain the "Keep context lean" row (currently `| Keep context lean | Never read intake/spec/plan artifacts; retain only pane maps, snapshots, and operator state (§2, §4). |`). No replacement row and no replacement condition (no "on demand only", no allowlist of document kinds) SHALL be added anywhere in the skill.

- **GIVEN** the §1 Principles table after apply
- **WHEN** its rows are enumerated
- **THEN** no row titled "Keep context lean" exists and no row or sentence prohibits or conditions reading plan, roadmap, intake, spec, or task documents
- **AND** the remaining rows (Multi-repo aware, Automate the routine, Do not enforce lifecycle, Re-derive state, Survive compaction) are unchanged

#### R2: "Coordinate, don't execute" is the single work rule
The §1 row titled "Coordinate, don't execute" MUST be rewritten to carry all five semantic elements from the intake (§ What Changes 1): (1) a trigger clause — every task, bug report, or idea the user hands the operator **is** a work request, with no "ask when ambiguous"; (2) the entry point — §6 Working a Change, raw-text form → `/fab-new` spawn in a fresh worktree for a fresh report, existing-change form (a send to that item's agent) for a report naming a live tracked item; (3) reading code to reproduce or diagnose, or editing files, in the operator pane **is** executing and is prohibited; (4) the operator reads whatever plan, roadmap, intake, or task document the tracked work needs and keeps handing the next unit to an agent until the plan is done; (5) a closed maintenance allowlist — merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution (§6), `fab operator track` verbs (§4), pane sends/answers/nudges (§5) — and the only question about a work request is which repo (plus §6 step 2's spawn-target-session tie-break). The row title MUST stay "Coordinate, don't execute".

- **GIVEN** a user message to the running operator that describes a bug or a task
- **WHEN** the operator applies §1
- **THEN** the row directs it to §6 Working a Change (spawn or send) and forbids diagnosing or editing in its own pane
- **AND** the row contains no "ask when ambiguous" and no open-ended "such as …" list of direct actions

- **GIVEN** a user asks the operator to check a multi-phase plan or roadmap
- **WHEN** the operator applies §1
- **THEN** the row permits reading the document and directs it to hand the next unit of work to an agent

### Runtime: Operator skill prose that cited the deleted row

#### R3: §2 Context Loading re-anchored, no artifact-load guard
In `fab-operator.md` §2 Context Loading the rationale clause "which the operator never does (§1 Context discipline)" MUST be re-anchored to the work rule (the operator never authors or reviews artifacts — §1 Coordinate, don't execute), and the trailing sentence "Do not load change artifacts." MUST be deleted with no replacement guard sentence. The three-file always-load set and "Do not run `fab preflight`." MUST remain verbatim.

- **GIVEN** §2 Context Loading after apply
- **WHEN** read
- **THEN** it names `config.yaml`, `constitution.md`, `context.md` (optional) and "Do not run `fab preflight`" exactly as before
- **AND** it contains neither "Context discipline" nor "Do not load change artifacts"

#### R4: §6 and §9 point at or describe the §1 rule, never re-enumerate it
The §6 "Pipeline-first" bullet's trailing sentence "Orchestration maintenance (merge, archive, worktree deletion) remains direct." MUST become a pointer to the §1 maintenance allowlist (e.g. "Direct actions are the §1 maintenance allowlist."). The §9 Key Properties row `| Loads change artifacts? | No — orchestration context only |` MUST become a descriptive row stating that the operator reads change/plan artifacts as the work needs (any plan, roadmap, intake, or task document required to drive tracked work — §1) and that none are startup always-loads (§2). Neither site MAY carry a second enumeration of the allowlist (owner-or-pointer rule).

- **GIVEN** the skill file after apply
- **WHEN** grepped for "maintenance-level", "Keep context lean", "Context discipline", "ask when ambiguous", "Loads change artifacts", "Do not load change artifacts"
- **THEN** zero matches
- **AND** the allowlist is enumerated exactly once, in the §1 row

### Memory-docs: Sibling sweep of restatements

#### R5: `docs/memory/runtime/operator.md` mirrors the new rule
The memory file MUST: rewrite the "**Coordinate, don't execute.**" principle paragraph (~line 21) to mirror R2's five elements and drop "If the target is ambiguous, ask."; delete the "**Context discipline.**" principle paragraph (~line 25); re-anchor the Context Loading clause "(per its own §1 Context discipline principle)" (~line 31) to the work rule and delete its trailing "It does NOT load change-specific artifacts." sentence (three-file text unchanged); delete the "**No change artifacts**" Design Constraints bullet (~line 260). The zc9m Design Decision body (~lines 384–388) MUST stay as history with exactly one appended forward-looking clause, e.g. "(the §1 read-prohibition row was later removed — see *Operator Reads Plans, Never Executes (kp3d)* below; the three-file startup load stands on the re-pay cost alone)". The `description:` frontmatter MUST NOT change.

- **GIVEN** the memory file after apply
- **WHEN** grepped for "Context discipline", "never reads change artifacts", "No change artifacts", "If the target is ambiguous, ask", "does NOT load change-specific artifacts"
- **THEN** the only remaining match for "Context discipline" is inside the zc9m Design Decision's **Why** field (history), immediately followed by the appended clause
- **AND** the frontmatter `description:` is byte-identical to before

#### R6: New Design Decision recorded in memory
`docs/memory/runtime/operator.md` `## Design Decisions` MUST gain an entry `### Operator Reads Plans, Never Executes (kp3d)` in the file's four-field shape (**Decision** / **Why** / **Rejected** / *Introduced by*: 260911-kp3d-operator-read-plans-never-execute), with content per the intake § What Changes 4 (decision = row deleted outright + single work rule with trigger, diagnosis-is-execution, closed allowlist, only-asks-which-repo; why = the two 2026-09-11 incidents and the context-budget rationale being obsolete; rejected = adding a clause beside the rows, "on demand only"/document-kind allowlist, keeping "ask when ambiguous").

- **GIVEN** the `## Design Decisions` section after apply
- **WHEN** read
- **THEN** the kp3d entry exists with all four fields and the exact `*Introduced by*` change name

#### R7: `docs/specs/skills.md` `/fab-operator` section mirrors the new rule
In `docs/specs/skills.md`, the **Context** paragraph (~line 1101) MUST no longer say "never reads change artifacts, keeping a long-lived context window reserved for coordination state"; it SHALL say the operator runs no `fab preflight`, loads no change artifacts at startup, and reads plan/roadmap/intake/task documents as the tracked work needs (§1 work rule). The **Coordinates, never executes** bullet (~line 1104) MUST carry the trigger clause, diagnosis-is-execution, and the closed allowlist (merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges), replacing "Operational maintenance (merge PR, archive, delete worktree) is the one direct-execution exception."

- **GIVEN** the `/fab-operator` section of `docs/specs/skills.md` after apply
- **WHEN** grepped for "never reads change artifacts" and "the one direct-execution exception"
- **THEN** zero matches
- **AND** the bullet lists the six allowlist items

#### R8: Verify-only sites and history are left alone
`docs/specs/operator.md` v9 row and `docs/specs/glossary.md` `/fab-operator` row MUST be read and left unchanged unless they contradict R2 (they are expected to be consistent — both speak of *work*, not *reading*). `docs/memory/pipeline/log.md`, `log.seed.md`, and every `docs/specs/findings/*` file MUST NOT be edited. `.agents/skills/` and `.claude/skills/` MUST NOT be edited.

- **GIVEN** `git diff --name-only` after apply
- **WHEN** inspected
- **THEN** it contains only `src/kit/skills/fab-operator.md`, `docs/memory/runtime/operator.md`, `docs/specs/skills.md`, and files under `fab/changes/260911-kp3d-*/`
- **AND** no path under `.agents/`, `.claude/`, `docs/specs/findings/`, or `docs/memory/pipeline/`

### Non-Goals

- Time-based (10-minute) full-frame rule replacing the every-10th-tick constant in Go `tick-start` — deferred by the user to a later change.
- HTML fleet table written beside the operator state file for a run-kit quake-terminal tab — deferred by the user.
- Any Go, test, `_cli-fab-operator.md`, migration, or constitution change.
- Backlog `hf6x` (worker-reported CI failures routed to the authoring agent) — adjacent, not folded in.
- A new v12 row in `docs/specs/operator.md`'s version history — principle sharpening within v11, not a skill iteration (intake assumption 17).

### Design Decisions

#### Operator Reads Plans, Never Executes (kp3d)
**Decision**: Delete the §1 "Keep context lean" row (never read intake/spec/plan artifacts) outright — the operator reads any plan, roadmap, intake, or task document it needs to drive tracked work to completion, with no replacement condition. "Coordinate, don't execute" becomes the single work rule: every user task, bug report, or idea is a work request entering via §6 Working a Change (`/fab-new` spawn in a fresh worktree; a send for a live tracked item); reading code to diagnose or editing files in the operator pane counts as executing; direct actions are a closed maintenance allowlist (merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges); the only question about a work request is which repo.
**Why**: Two 2026-09-11 incidents. (1) Handed a bug, the operator reproduced and fixed it inline — the old row had no trigger clause, and "ask when ambiguous" + "maintenance-level actions such as…" read as outs; diagnosis was treated as not-execution. (2) Asked to "check the rk-mcp plan" (W0/W1 merged, W2–W4 unspawned), the operator refused to read the roadmap, citing the read-prohibition, and asked the user what the next phase should cover. The read rule existed for context budget (PR #550 era); the server-keyed state file + §4 Post-Compaction Reload already make the operator survive context loss, so the rule's cost (cannot sequence a roadmap) exceeded its benefit. User direction: remove conditions rather than add them; the one condition that stays is "never execute in the operator pane".
**Rejected**: Adding a "bug report = work request" clause beside the existing rows (adds conditions). Relaxing the read rule to "on demand only" or an allowlist of document kinds (new conditions; the rk-mcp roadmap was not a fab artifact). Keeping "ask when ambiguous" (the stall hatch the incident exercised).
*Introduced by*: 260911-kp3d-operator-read-plans-never-execute

### Deprecated Requirements

#### Operator never reads change artifacts
**Reason**: The prohibition blocked the operator from sequencing multi-phase plans (the rk-mcp incident); its context-budget rationale is obsolete now that durable state lives in the server-keyed state file and §4 Post-Compaction Reload restores the procedure losslessly.
**Migration**: N/A — no replacement condition. The §1 work rule states the positive permission.

## Tasks

### Phase 1: Setup

- [x] T001 Baseline grep — record `grep -n -i "keep context lean\|context discipline\|maintenance-level\|ask when ambiguous\|never read\|change artifacts\|direct-execution exception\|ambiguous, ask" src/kit/skills/fab-operator.md docs/memory/runtime/operator.md docs/specs/skills.md docs/specs/operator.md docs/specs/glossary.md` output as the sweep checklist for T002–T009; confirm `docs/memory/runtime/operator.md` frontmatter `description:` value to check byte-identity in T010 <!-- R8 -->

### Phase 2: Core Implementation

- [x] T002 In `src/kit/skills/fab-operator.md` §1 Principles: delete the `| Keep context lean | … |` row and rewrite the `| Coordinate, don't execute | … |` row with the five required elements (trigger clause, §6 Working a Change entry incl. raw-text `/fab-new` spawn and existing-change send, diagnosis/editing-in-pane is execution, read-plans permission + drive-to-completion, closed six-item allowlist with §6/§4/§5 cross-refs, only-asks-which-repo + §6 step 2 tie-break); keep the row title; no "ask when ambiguous", no "such as" <!-- R1 R2 -->
- [x] T003 In `src/kit/skills/fab-operator.md` §2 Context Loading: replace "(§1 Context discipline)" with a re-anchor to the work rule ("the operator never authors or reviews artifacts — §1 Coordinate, don't execute"), delete "Do not load change artifacts."; leave the three files and "Do not run `fab preflight`." verbatim <!-- R3 -->
- [x] T004 In `src/kit/skills/fab-operator.md`: §6 "Pipeline-first" bullet — replace "Orchestration maintenance (merge, archive, worktree deletion) remains direct." with a pointer to the §1 maintenance allowlist; §9 Key Properties — replace `| Loads change artifacts? | No — orchestration context only |` with the descriptive "Reads change/plan artifacts? | As the work needs — any plan, roadmap, intake, or task document required to drive tracked work (§1); none are startup always-loads (§2)" row; then grep the file for every T001 phrase and fix any remaining citation <!-- R4 -->
- [x] T005 [P] In `docs/memory/runtime/operator.md` #### Principles: rewrite the "**Coordinate, don't execute.**" paragraph to mirror R2's five elements (drop "If the target is ambiguous, ask."); delete the "**Context discipline.**" paragraph <!-- R5 -->
- [x] T006 [P] In `docs/memory/runtime/operator.md`: #### Context Loading — re-anchor "(per its own §1 Context discipline principle)" to the work rule and delete "It does NOT load change-specific artifacts."; #### Design Constraints — delete the "**No change artifacts**" bullet; zc9m Design Decision **Why** — append the single forward-looking clause pointing at the kp3d entry; do not touch the `description:` frontmatter <!-- R5 -->
- [x] T007 [P] In `docs/memory/runtime/operator.md` `## Design Decisions`: append `### Operator Reads Plans, Never Executes (kp3d)` with the four fields from this plan's § Design Decisions, `*Introduced by*: 260911-kp3d-operator-read-plans-never-execute` <!-- R6 -->
- [x] T008 [P] In `docs/specs/skills.md` `/fab-operator` section: rewrite the **Context** paragraph (no preflight; no change artifacts at startup; reads plan/roadmap/intake/task documents as the work needs — §1 work rule) and the **Coordinates, never executes** bullet (trigger clause, diagnosis-is-execution, six-item closed allowlist replacing "the one direct-execution exception") <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T009 Verify-only: read `docs/specs/operator.md` v9 row and `docs/specs/glossary.md` `/fab-operator` row against R2; leave unchanged if consistent (expected); record the verdict in the apply result summary <!-- R8 -->
- [x] T010 Final sweep: re-run the T001 grep across the five files — the only permitted hit is "Context discipline" inside the zc9m DD **Why** field (with the appended clause); confirm `git diff --name-only` shows only the three target files plus `fab/changes/260911-kp3d-*/`; confirm `docs/memory/runtime/operator.md` frontmatter `description:` is unchanged (`git diff docs/memory/runtime/operator.md | grep '^[-+]description:'` prints nothing); confirm the allowlist string appears exactly once in `fab-operator.md` <!-- R4 R5 R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/fab-operator.md` §1 has no "Keep context lean" row and no sentence anywhere in the file prohibiting or conditioning reads of plan/roadmap/intake/spec/task documents
- [x] A-002 R2: The §1 "Coordinate, don't execute" row carries all five elements — trigger clause, §6 Working a Change entry (raw-text `/fab-new` spawn + existing-change send), diagnosis/editing-in-pane is execution, read-plans permission with drive-to-completion, closed six-item allowlist — and the only-asks-which-repo clause
- [x] A-003 R3: §2 Context Loading names the same three files and "Do not run `fab preflight`" verbatim, cites "§1 Coordinate, don't execute" for why code-quality/code-review/indexes are skipped, and no longer contains "Do not load change artifacts"
- [x] A-004 R4: The §6 Pipeline-first bullet points at the §1 allowlist and the §9 row reads as a descriptive "Reads change/plan artifacts?" property
- [x] A-005 R5: `docs/memory/runtime/operator.md` principle paragraph mirrors the work rule; "Context discipline" paragraph, "No change artifacts" bullet, "If the target is ambiguous, ask.", and "It does NOT load change-specific artifacts." are gone; zc9m DD carries the appended clause
- [x] A-006 R6: `### Operator Reads Plans, Never Executes (kp3d)` exists in `## Design Decisions` with **Decision** / **Why** / **Rejected** / *Introduced by* fields and the full change name
- [x] A-007 R7: `docs/specs/skills.md` `/fab-operator` **Context** and **Coordinates, never executes** entries carry the new rule; "never reads change artifacts" and "the one direct-execution exception" are gone

### Behavioral Correctness

- [x] A-008 R2: The row contains neither "ask when ambiguous" nor "such as"; the row title is still "Coordinate, don't execute"
- [x] A-009 R2: The row's allowlist is exactly: merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges (six items, no "etc.")

### Removal Verification

- [x] A-010 R1: `grep -n -i "keep context lean\|context discipline\|maintenance-level\|ask when ambiguous" src/kit/skills/fab-operator.md` returns zero lines
- [x] A-011 R5: In `docs/memory/runtime/operator.md`, the only line matching "Context discipline" is inside the zc9m Design Decision **Why** field

### Scenario Coverage

- [x] A-012 R2: A reader given the §1 row and a message "there's a bug in X" can identify the required action as a §6 spawn/send without consulting any other section
- [x] A-013 R2: A reader given the §1 row and "check the rk-mcp plan" finds explicit permission to read the plan and an instruction to hand the next phase to an agent

### Edge Cases & Error Handling

- [x] A-014 R3: The `_preamble.md` §1 note that `/fab-operator` loads a reduced 3-file set remains accurate (the three-file startup load is unchanged)
- [x] A-015 R8: `git diff --name-only` shows only `src/kit/skills/fab-operator.md`, `docs/memory/runtime/operator.md`, `docs/specs/skills.md`, and `fab/changes/260911-kp3d-*/` files; `docs/specs/operator.md` and `docs/specs/glossary.md` are unchanged (or, if edited, the edit is justified by a real contradiction in the apply summary)
- [x] A-016 R5: `docs/memory/runtime/operator.md` frontmatter `description:` is byte-identical to `main` (no `fab docs-index` regeneration needed)

### Code Quality

- [x] A-017 Pattern consistency: The rewritten §1 row keeps the table's `| Principle | Rule |` shape with §-cross-references in parentheses like its siblings; the new memory DD matches the file's existing four-field entries
- [x] A-018 No unnecessary duplication: The maintenance allowlist is enumerated once (§1); §6, §9, and the memory/spec restatements point at it or mirror it as documented sweep sites — no third divergent wording inside the skill
- [x] A-019 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/` (code-quality.md anti-pattern "Editing deployed skills directly")
- [x] A-020 Owner-or-pointer: no skill-file site both states the allowlist and points at §1 (code-quality.md anti-pattern "Stating an owned rule AND pointing at its owner")
- [x] A-021 Deployed content cites no fab-kit-only paths: the rewritten skill text references only §-internal sections and kit helpers, never `docs/specs/*` or `docs/memory/*` (Constitution V; code-quality.md anti-pattern)
- [x] A-022 Sibling sweep done up front: every sweep site from intake § What Changes 3 is either edited or recorded as verify-only in the apply summary (code-quality.md § Sibling Sweeps)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Hydrate note: the intake already drafts the kp3d Design Decision verbatim; T007 lands it during apply, so hydrate should verify rather than re-author it.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The §1 row is written as a single table cell (one `|`-row), not split into multiple rows, even though it carries five elements | The table's existing rows are single cells with §-refs; splitting into rows would re-introduce the multi-condition shape the user rejected | S:80 R:95 A:90 D:85 |
| 2 | Certain | The allowlist in the §1 row carries §-cross-references (§6 for rebase/cherry-pick, §4 for track verbs, §5 for pane sends) matching sibling rows' style | Pattern consistency with the table; the intake's proposed wording already includes them | S:85 R:95 A:90 D:90 |
| 3 | Confident | The zc9m DD annotation is one parenthetical clause appended to the end of its **Why** field, not a new field or a body rewrite | Intake assumption 10 ("stays as history … one-clause forward-looking annotation"); the **Why** field is where the contradicted claim lives | S:70 R:95 A:85 D:75 |
| 4 | Confident | T005–T008 are marked `[P]` (different files or non-overlapping regions of `operator.md`) | T005/T006/T007 touch distinct regions of one file; a sequential worker executes them in order anyway, so the marker is informational | S:60 R:95 A:80 D:70 |
| 5 | Certain | A `### Deprecated Requirements` entry is recorded for the removed read-prohibition, with Migration N/A | The change removes an existing normative rule; the template reserves this subsection for exactly that, and review's Removal Verification category keys off it | S:80 R:95 A:90 D:90 |
| 6 | Certain | Change type stays `docs`; no test or Go acceptance items are generated | Intake assumption 11; no source under `source_paths` is touched | S:90 R:95 A:95 D:95 |

6 assumptions (4 certain, 2 confident, 0 tentative).
