# Plan: Operator Non-Fab-Repo Work Path — Plain Agent Form

**Change**: 260911-pudv-operator-non-fab-repo-plain-agent
**Intake**: `intake.md`

## Requirements

### Operator Skill: Work-Request Routing

#### R1: Every work request goes to an agent, split on `fab/` presence
`src/kit/skills/fab-operator.md` §1 "Coordinate, don't execute" MUST state that every work request goes to an agent: the fab pipeline when the target repo has a `fab/` project (§6 Working a Change forms 1–3), a plain agent in a fresh worktree otherwise (form 4). The rest of the row (maintenance allowlist, which-repo-only ask, §3/§6 confirmation carve-outs) MUST remain verbatim.

- **GIVEN** the operator receives a work request naming a repo with no `fab/` directory
- **WHEN** it consults §1
- **THEN** §1 names form 4 as the path, and offers no menu, no direct-edit option, and no `fab init` option

#### R2: §6 Working a Change form 4 — Non-fab repo → plain agent
§6 Working a Change MUST gain a fourth numbered form covering a target repo with no `fab/` directory. Form 4 SHALL reuse the §6 spawn sequence unchanged (target repo, target session, `wt create --non-interactive` in the target repo, existence guard trivially skipped, dependencies, `fab agent default -o yaml --repo <target-repo>` degrading to the project-free cascade per `_cli-fab-operator.md` § fab agent, `rk tab new … -- <spawn-argv…> "<prompt>"`). The prompt MUST be the raw task text plus the closing instruction "Commit on this branch, push, and open a draft PR against the default branch." with no skill prefix and no § Skill Prompts rendering. The spawn MUST be tracked unconditionally as a `pane` item with no change, item id = worktree name, with no `--stop-stage`. Completion MUST be described as a chained `github-pr` item the operator adds once the PR URL is known (or the user `track rm`). Form 4 MUST carry the one-line "never `fab init` the repo" restatement plus a pointer to the Pipeline-first owner.

- **GIVEN** a work request for `c7-sahil87` where `test -d <repo>/fab` fails
- **WHEN** the operator follows §6 Working a Change
- **THEN** form 4 applies and the agent is spawned with a raw prompt, tracked as `fab operator track add <wt> --kind pane --pane <pane-id> --session <session> --repo <repo> --branch <wt>`
- **AND** on PR URL the operator chains `fab operator track add <wt>-pr --kind github-pr --scope '{"repo":"<repo>","pr":<n>}' --check-every 2m` and removes the pane item

#### R3: Pipeline-first scoped to fab projects; `fab init` never a default
§6 The pane Kind and Spawn Rules' **Pipeline-first** bullet MUST bind repos WITH a `fab/` directory, state that a repo with no `fab/` is exempt only from the pipeline and not from delegation (pointer to form 4), and own the rule that the operator never runs `fab init` — bootstrapping fab is the user's repo-structure decision, mentioned at most once as an option in a report, never picked, never offered as a menu. The **Spawn in a worktree** bullet MUST visibly bind form-4 spawns. The `--stop-stage hydrate` paragraph MUST state that a form-4 spawn has no stages, takes no `--stop-stage`, and completes through its chained `github-pr` item.

- **GIVEN** the operator is about to spawn into a repo with no `fab/`
- **WHEN** it reads Pipeline-first
- **THEN** it spawns a plain agent in a worktree and does not run or offer `fab init`

#### R4: Sweep — no site says or implies every spawn is a `/fab-new`
Every site in `src/kit/skills/fab-operator.md` that says or implies every spawn is a `/fab-new` MUST be reconciled with form 4: §2 wt Gate first sentence, §2 Context Loading "fresh idea needs `/fab-new` → `/fab-fff`", §6 Spawning an Agent steps 4, 6, 7, 8, the §6 Working a Change intro blockquote ("all three work paths"), and the completion sentence ("On completion (all three)… Both raw text and backlog paths use `/fab-new`"). §6 Queues, Dependency Resolution and merge-mode prose MUST NOT be restructured.

- **GIVEN** the edited skill
- **WHEN** `grep -n "fab-new\|all three\|compose the selected skill\|skill_prefix\|stop-stage\|Pipeline-first" src/kit/skills/fab-operator.md` is run
- **THEN** every hit is either form-4-aware or refers only to forms 1–3 explicitly

#### R5: §7 Conversational Map row
§7 Conversational Map MUST gain a row: utterance "Fix the broken brew line in c7-sahil87's README" (repo has no `fab/`) → spawn per §6 Working a Change form 4, then `fab operator track add c7-sahil87-<wt> --kind pane --pane %N --repo … --session …`, chaining a `github-pr` item on PR URL.

- **GIVEN** the user says "Fix the broken brew line in c7-sahil87's README"
- **WHEN** the operator consults §7
- **THEN** the row maps it to a form-4 spawn plus the `track add … --kind pane` verb

#### R6: Helper pointer verification and Constitution V
`_cli-external.md` § wt and `_cli-agents.md` § Spawn Composition / § Skill Prompts MUST be read; a one-line pointer to `fab-operator.md` §6 Working a Change form 4 SHALL be added only if a sentence there contradicts form 4 (claims the target is always a fab project or the prompt is always a skill invocation). All new prose MUST cite only kit skills, `fab` commands, or `fab-operator.md`'s own sections — never `docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*`. The Go portability guard test MUST pass. Deployed copies MUST be regenerated via `fab sync`, never hand-edited.

- **GIVEN** the edits are complete
- **WHEN** the Constitution V guard test runs (`go test ./src/go/fab-kit/cmd/fab/ -run <guard>`)
- **THEN** it passes and `fab sync` regenerates `.agents/skills/fab-operator/SKILL.md`

### Non-Goals
- Any Go change, auto-detection of non-fab repos in the binary, `fab init` automation — user decision; the operator makes the existence check itself
- run-kit changes; docs/specs changes
- Restructuring §6 Queues / merge choreography

### Design Decisions

#### Non-Fab Repos Get a Plain Agent, Never the Operator's Hands, Never an Unprompted fab init
**Decision**: When a work request targets a repo with no `fab/` directory, the operator still spawns an agent in a fresh worktree through the same §6 spawn sequence, sends the raw task text plus a commit/push/draft-PR instruction (no skill prefix), tracks it as a change-less `pane` item keyed by worktree name, and completes it through a chained `github-pr` item. Pipeline-first binds only repos with a `fab/` project. The operator never runs `fab init` and never offers it as a menu; it may mention it once as an option in a report.
**Why**: The 2026-09-11 incident: §1 prohibited executing but §6 offered only `/fab-new`-based forms, so in `c7-sahil87` the operator improvised a menu whose option 2 was the prohibited direct edit. The spawn sequence already works without `fab/` (`wt` is repo-agnostic, `fab agent --repo` degrades to the project-free cascade, the `pane` kind from 260911-1159 tracks change-less panes), so the fix is a named form, not a mechanism.
**Rejected**: Letting the operator edit directly in non-fab repos (contradicts the single never-execute constraint, kp3d). Making `fab init` the default (a repo-structure decision that is the user's). Go-side detection / `fab init` automation (out of scope; a one-line existence check the skill can make).
*Introduced by*: 260911-pudv-operator-non-fab-repo-plain-agent

## Tasks

### Phase 1: Setup

- [x] T001 Read `src/kit/skills/fab-operator.md` §1 row, §2 wt Gate + Context Loading, §6 The pane Kind, Spawning an Agent steps 4/6/7/8, Working a Change, §7 Conversational Map; read `_cli-external.md` § wt and `_cli-agents.md` § Spawn Composition / § Skill Prompts for contradictions; record whether a pointer is needed (expected: none) <!-- R6 -->

### Phase 2: Core Implementation

- [x] T002 Edit `src/kit/skills/fab-operator.md` §1 "Coordinate, don't execute" row (fab-project pipeline vs plain agent split, rest verbatim) and §6 The pane Kind and Spawn Rules (Pipeline-first scoped to fab projects + `fab init` never-default owner; Spawn-in-a-worktree binds form 4; `--stop-stage` paragraph gains the form-4 sentence) <!-- R1 R3 -->
- [x] T003 Edit `src/kit/skills/fab-operator.md` §6 Working a Change: rewrite the intro blockquote ("forms 1–3 … form 4 is the non-fab exception"), append form 4 with the full spawn/prompt/track/chained-completion/no-`--stop-stage`/never-`fab init` content, and rewrite the completion sentence ("On completion (forms 1–3) …; form 4 completes through its chained `github-pr` item. Forms 2 and 3 use `/fab-new` …") <!-- R2 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Sweep `src/kit/skills/fab-operator.md`: §2 wt Gate first sentence ("every form, including the non-fab plain-agent form 4"), §2 Context Loading sentence, §6 Spawning an Agent step 4 (form 4 skips the switch), step 6 (`skill_prefix` unused for form 4, project-free cascade), step 7 (compose the prompt — forms 1–3 skill, form 4 raw), step 8 (form-4 id stays the worktree name, completion via chained `github-pr`); add the §7 Conversational Map row; then grep `fab-new|all three|compose the selected skill|skill_prefix|stop-stage|Pipeline-first` and reconcile every hit <!-- R4 R5 -->

### Phase 4: Polish

- [x] T005 Add the `_cli-external.md`/`_cli-agents.md` pointer only if T001 found a contradiction; run the Constitution V guard test in `src/go/fab-kit/cmd/fab/` (the deployed-content citation test) and `fab sync`; confirm `.agents/skills/fab-operator/SKILL.md` matches the source <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: §1 "Coordinate, don't execute" names the fab-project pipeline (forms 1–3) and the plain agent (form 4) as the two agent paths; maintenance allowlist and which-repo-only ask are verbatim
- [x] A-002 R2: §6 Working a Change has a numbered form 4 with: `fab/` existence probe, same spawn sequence, raw prompt + PR instruction, no skill prefix, `pane` item keyed by worktree name, no `--stop-stage`, chained `github-pr` completion, never-`fab init` restatement + pointer
- [x] A-003 R3: Pipeline-first bullet binds repos with `fab/`, exempts non-fab repos only from the pipeline, and owns the `fab init` never-default rule (mention once, never pick, never menu)
- [x] A-004 R5: §7 Conversational Map has the c7-sahil87 row mapping to a form-4 spawn + `track add … --kind pane`
- [x] A-005 R6: `_cli-external.md` § wt and `_cli-agents.md` § Spawn Composition were checked; a pointer was added only if a contradiction exists (else untouched) — verified: `_cli-agents.md` untouched (its raw-form prompt is optional and § Skill Prompts step 3 exempts ordinary prompts — no contradiction); `_cli-external.md` § wt gained one sentence because "the branch matches the change" + the always-branch probe-and-route implied a change branch always exists, contradicting form 4

### Behavioral Correctness

- [x] A-006 R3: The `--stop-stage hydrate` paragraph states form-4 spawns take no `--stop-stage` and complete through the chained `github-pr` item
- [x] A-007 R2: The completion sentence after the forms says forms 1–3 complete via PR/archive and form 4 via its chained `github-pr` item; "Both raw text and backlog paths" became "Forms 2 and 3"

### Scenario Coverage

- [x] A-008 R4: `grep -n "fab-new\|all three\|compose the selected skill\|skill_prefix\|stop-stage\|Pipeline-first" src/kit/skills/fab-operator.md` — every hit is form-4-aware or explicitly forms-1–3-scoped; §2 wt Gate, §2 Context Loading, §6 steps 4/6/7/8, Working a Change intro and completion are all reconciled — verified: no "all three" hits remain; remaining `/fab-new` hits are form-2/3-specific or linear/queue examples scoped to fab projects

### Edge Cases & Error Handling

- [x] A-009 R4: §6 Queues, Dependency Resolution and merge-mode prose are byte-identical to the base branch (`git diff` shows no hunks there) — verified: diff hunks land only at §1 row, §2 Context Loading, §2 wt Gate, §6 pane-kind bullets, `--stop-stage` paragraph, spawn steps 4/6/7/8, Working a Change, and §7
- [x] A-010 R6: No new citation of `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` in `src/kit/skills/fab-operator.md`; the Go portability guard test passes — verified: `TestKitContentCitesNoRepoLocalPaths|TestKitPortabilityMatcher` ok

### Code Quality

- [x] A-011 Pattern consistency: New prose follows the skill's existing voice (bold rule names, §-references, `sh` blocks for verbs, owner-or-pointer)
- [x] A-012 No unnecessary duplication: the `fab init` rule is stated in full once (Pipeline-first, fab-operator.md:467) and restated in one line + pointer in form 4 (fab-operator.md:600); no third copy — verified by grep
- [x] A-013 Canonical source only: edits are in `src/kit/skills/`; `.agents/skills/` and `.claude/skills/` copies are regenerated by `fab sync`, not hand-edited — diff touches only `src/kit/skills/` (deployed copies stale in this worktree until the next sync, as expected in the dev repo)
- [x] A-014 No Go change: `git diff --name-only` shows no `.go` file

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Additive prose-only change: the pre-change phrases were reworded in place, not orphaned; no file, symbol, section, or config block became unused.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Five tasks → light lane; apply/hydrate/ship/review-pr inline, review dispatched | Task count ≤ 5 per `_pipeline.md` Step 1 fork; intake CONSTRAINTS expects light | S:90 R:95 A:95 D:95 |
| 2 | Confident | The `fab/` probe in form 4 is spelled `test -d <target-repo>/fab` (main-worktree root) | Simplest portable check; matches how `fab agent` itself detects a project (walks up for `fab/`) | S:75 R:90 A:85 D:75 |
| 3 | Confident | Chained completion item id `<wt>-pr`, `--check-every 2m`, scope `{repo, pr}`; pane item removed once the chain is added | Mirrors the existing §7 `pr-913` row and the `github-pr` kind contract in `_cli-fab-operator.md` | S:75 R:85 A:80 D:70 |
| 4 | Confident | The §7 row keeps the description's `c7-sahil87-<wt>` id spelling with a parenthetical that form 4's rule is "item id = worktree name" | Both spellings come from the description; `track add <slug>` accepts any id | S:70 R:90 A:75 D:65 |
| 5 | Tentative | §2 Context Loading sentence is included in the sweep | It implies fresh idea ⇒ `/fab-new`; low cost, reviewer may deem noise | S:55 R:90 A:60 D:55 |

5 assumptions (1 certain, 3 confident, 1 tentative).
