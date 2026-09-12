# Plan: Operator Spawn Forms as Two Orthogonal Axes — Task-less Bare-Agent Spawn

**Change**: 260912-6n6o-operator-spawn-orthogonal-axes
**Intake**: `intake.md`

## Requirements

### Operator Skill: Spawn Routing

#### R1: Two-axis routing replaces the enumerated form list
`src/kit/skills/fab-operator.md` §6 Working a Change MUST define spawn routing as two independent axes — **Axis A** (the target repo has a `fab/` directory, probed `test -d <target-repo>/fab` against the main-worktree root) and **Axis B** (a task was given: raw text, backlog ID, Linear issue, or existing change) — rendered as a 2×2 cell table plus one bare-agent paragraph. The numbered forms 1–4, the pipeline-first intro blockquote, and the "On completion (forms 1–3)" sentence SHALL be removed. The fab+task cell MUST preserve forms 1–3's behavior verbatim (skill `fab-fff <change>` / `fab-new <raw>` / `fab-new <id>` rendered per `_cli-agents.md` § Skill Prompts); the non-fab+task cell MUST preserve form 4's behavior verbatim (raw task + closing PR instruction, `track add <wt> … --branch <wt>`, no `--stop-stage`, chained `github-pr` completion).

- **GIVEN** the operator receives a work request or a spawn request
- **WHEN** it reaches §6 Working a Change
- **THEN** it selects exactly one cell by (Axis A, Axis B), and every other site in the skill names the cell, never a form number

#### R2: Bare agent — the no-task cell
The no-task column MUST be one cell, identical for both repo types: the same eight-step spawn sequence; step 3 `wt create --non-interactive` with no branch argument (wt's random name, or `--name <name>` when the user gave one — that name is the worktree, the branch, the tab name, and the item id); step 4's guard trivially skips; step 7 passes **no prompt token** (argv after `--` ends at the session command — no skill prefix, no closing PR instruction); step 8 `fab operator track add <wt> --kind pane --pane <pane-id> --session <session> --repo <repo> --branch <wt>` with no `--stop-stage` and **no chained `github-pr` item**. Completion is pane death, agent exit, or the user's `track rm`; the operator reports and acks per §4 and never respawns. The paragraph MUST state that pipeline-first still holds in the fab-project cell because nothing is sent.

- **GIVEN** a user says "Start an agent in a new worktree" (no task) in a fab or non-fab repo
- **WHEN** the operator routes it
- **THEN** it spawns a bare agent per the above and tracks it, asking at most which repo
- **AND** never asks for a task and never refuses

#### R3: §1 ask rule extends to spawns
The §1 "Coordinate, don't execute" row MUST state that a request to start an agent with no task is a bare spawn that goes to an agent, that the operator never refuses to start an agent, that routing is §6 Working a Change's two axes, and that the only question about a work request **or a spawn** is which repo (plus the session tie-break) — never for a task. The rest of the row (executing prohibition, maintenance allowlist, confirmations still apply) is unchanged.

- **GIVEN** the §1 row
- **WHEN** read by the operator model
- **THEN** a task-less spawn request is classified as a bare spawn, not as a missing input

#### R4: Sweep — every form reference becomes a cell reference
Every remaining `form 4` / `form-4` / `forms 1–3` / `Working a Change form` reference in `src/kit/skills/fab-operator.md` (§2 Context Loading, §2 wt Gate, §6 pane Kind bullets ×3, §6 Spawning steps 4/6/7/8, §7 Linear step 4, §7 Conversational Map row) MUST be reworded to the cell name, with the bare-agent case added where the sentence enumerates spawn kinds (wt gate, spawn-in-worktree bullet, `--stop-stage` paragraph, step 4 guard, step 6 `skill_prefix`, step 7 prompt composition, step 8 tracking). Step 7's `rk tab new` line SHALL show the prompt as optional (`["<prompt>"]`); the window note for a change-less item is `"<id>"` (no stage segment). The §7 Conversational Map MUST gain a bare-spawn row. `src/kit/skills/_cli-external.md` § wt Operator Spawning Rules line about no-change-branch spawns MUST cover both the non-fab and the bare case and point at §6 Working a Change (no form number).

- **GIVEN** the apply is complete
- **WHEN** `grep -rn 'form 4\|form-4\|forms 1–3\|forms 1-3\|Working a Change form' src/kit/skills/` runs
- **THEN** it returns zero hits

#### R5: Worktree-name flag respelled in touched text
`wt create --help` reports `--worktree-name` as **deprecated, use `--name`** (verified: the alias still works and prints a deprecation line). Every sentence and code block this change rewrites — §6 Working a Change, `fab-operator.md` §6 step 3 dependency-resolution line 153, `_cli-external.md` § wt Operator Spawning Rules (the two code-block lines and the flag list on line 109) — SHOULD spell the flag `--name`. Untouched sites (`_preamble.md` § Naming, `_cli-fab.md` § fab batch, `docs/specs/naming.md`, the Go batch launcher) are out of scope (see Non-Goals).

- **GIVEN** the touched text
- **WHEN** it names the worktree-name flag
- **THEN** it reads `--name`

### Non-Goals

- Any Go change (including `batch_new.go`/`batch_switch.go` still passing the deprecated `--worktree-name` alias — works today, separate follow-up), any `fab operator track` or `rk tab new` change, auto-detecting "bare" in the binary.
- Respawn semantics (queue-driven only), the built-in pane completion predicate, `github-pr` chaining for plain agents, `fab init` behavior — all unchanged.
- `docs/specs/operator.md` (does not restate the forms; a v13 row is a human call) and the Constitution (no MUST rule added or changed).
- `docs/memory/runtime/operator.md` — hydrate's job, not apply's.

### Design Decisions

#### Spawn Routing Is Two Orthogonal Axes, Not an Enumerated Form List
**Decision**: The operator routes every spawn by (target repo has `fab/`) × (task given). The no-task column is a single cell — the bare agent — identical for both repo types; the operator never refuses to start an agent and never asks for a task.
**Why**: The four forms were the cartesian product of two yes/no questions with one cell missing; the model read the missing cell as "task is a required input" and refused a bare "start an agent" request. Naming the axes fills the cell and deletes the enumeration, so complexity goes down while the gap closes.
**Rejected**: A fifth "bare spawn" form (keeps the enumeration, adds a case); rewording the "asks only which repo" sentence alone (the input was never classified as a work request, so the sentence never fired); pushing bare spawns out of the operator to `wt create` + `fab agent` by hand (contradicts the delegate-everything direction, 260911-kp3d).
*Introduced by*: 260912-6n6o-operator-spawn-orthogonal-axes

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `src/kit/skills/fab-operator.md` §6 Working a Change as the two-axis cell table + bare-agent paragraph (remove forms 1–4, intro blockquote, "On completion" sentence; keep the `fab init` pointer); rewrite the §1 "Coordinate, don't execute" row's opening and ask sentences <!-- R1, R2, R3 -->
- [x] T002 Sweep `src/kit/skills/fab-operator.md`: §2 Context Loading, §2 wt Gate, §6 pane Kind bullets (pipeline-first + bare-spawn exemption sentence, spawn-in-worktree, `--stop-stage` paragraph), §6 Spawning steps 4/6/7 (prompt composition, optional `["<prompt>"]`, change-less window note)/8, step 3 line 153 flag respell, §7 Linear step 4, §7 Conversational Map c7 row + new bare-spawn row <!-- R4, R5 -->

### Phase 3: Integration & Edge Cases

- [x] T003 [P] Edit `src/kit/skills/_cli-external.md` § wt Operator Spawning Rules: no-change-branch sentence covers non-fab and bare spawns and points at §6 Working a Change; respell `--worktree-name` → `--name` in the two code-block lines and the line-109 flag list <!-- R4, R5 -->
- [x] T004 Verify: `grep -rn 'form 4\|form-4\|forms 1–3\|forms 1-3\|Working a Change form' src/kit/skills/` is empty; touched text has no `--worktree-name`; `go test ./src/go/fab-kit/cmd/fab/` (portability guard) passes; `fab sync` refreshes deployed copies without error <!-- R4, R5 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: §6 Working a Change contains the Axis A / Axis B definitions and a 2×2 table whose fab+task and non-fab+task cells carry forms 1–3's and form 4's behavior verbatim; no numbered form list remains
- [x] A-002 R2: A bare-agent paragraph defines the no-task cell for both repo types — no prompt token, no PR instruction, `track add <wt> … --branch <wt>` with no `--stop-stage`, no chained `github-pr` item, completion by pane death / agent exit / `track rm`, no respawn, pipeline-first-still-holds sentence
- [x] A-003 R3: The §1 row says a task-less spawn request is a bare spawn, the operator never refuses to start an agent, and the only question about a work request or a spawn is which repo — never a task
- [x] A-004 R4: Every listed sweep site in `fab-operator.md` and the `_cli-external.md` sentence name a cell (or "§6 Working a Change"), never a form number; the §7 map has a bare-spawn row
- [x] A-005 R5: Touched text spells the worktree-name flag `--name`

### Behavioral Correctness

- [x] A-006 R2: The fab-project + no-task cell explicitly states pipeline-first is not violated because nothing is sent
- [x] A-007 R4: Step 7's `rk tab new` line shows the trailing prompt as optional and the bare cell passes none; the change-less window note has no stage segment

### Removal Verification

- [x] A-008 R1: `grep -rn 'form 4\|form-4\|forms 1–3\|forms 1-3\|Working a Change form' src/kit/skills/` returns zero hits

### Scenario Coverage

- [x] A-009 R2: Reading §1 + §6 as the operator model, the utterance "Start an agent in a new worktree" in a non-fab repo resolves to the bare cell with at most a which-repo question (the §7 map row makes this explicit)

### Edge Cases & Error Handling

- [x] A-010 R2: Linear/Slack-driven (§7 step 4) and queue-driven spawns remain tasked (never bare) and reference the cells only

### Code Quality

- [x] A-011 Pattern consistency: Edits follow the skill's existing §-reference and owner-or-pointer style; cells are defined once in §6 Working a Change and pointed at elsewhere
- [x] A-012 No unnecessary duplication: No site restates the cell table or an owned rule alongside its pointer; the four-form enumeration is gone and any net growth in `fab-operator.md` is confined to the new bare-agent cell and the §7 map row
- [x] A-013 Canonical source only: Edits land in `src/kit/skills/`, never `.agents/skills/` or `.claude/skills/`
- [x] A-014 Deployed content cites no fab-kit-only paths (Constitution V; `go test ./src/go/fab-kit/cmd/fab/` passes)
- [x] A-015 Sibling sweep: `_cli-external.md` pointer sentence updated alongside the owner; memory mirror left for hydrate

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the four-form enumeration was removed in place as the planned removal (verified by A-008's zero-hit grep); the change makes no other existing code or prose redundant.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--worktree-name` is a **deprecated alias** of `--name` (not a phantom flag as the intake's row 11 said); respell only touched text, leave the Go batch launcher and untouched docs as a follow-up | Verified: `wt create --worktree-name … --help` prints "Flag --worktree-name has been deprecated, use --name instead" and exits 0 | S:90 R:95 A:95 D:90 |
| 2 | Certain | Also respell `fab-operator.md` line 153 (dependency-resolution `wt create … --checkout`) since it sits in the same skill and is a one-token fix | Same file, same class of drift; zero behavior impact | S:70 R:95 A:95 D:85 |
| 3 | Confident | Change-less window note is `"<id>"` (no ` · <stage>` segment); mark/unmark contract otherwise unchanged | Intake row 13; a change-less pane has no stage to render | S:50 R:90 A:85 D:80 |
| 4 | Confident | Accept ~+350 words net in `fab-operator.md` (14224 → 14572): the enumeration is gone, but the bare cell is new normative content and the `fab/` cell is now self-contained; restated owned rules were trimmed to pointers | Intake said prose SHOULD NOT grow; the growth is the new behavior itself, not duplication | S:60 R:95 A:85 D:75 |

4 assumptions (2 certain, 2 confident, 0 tentative).
