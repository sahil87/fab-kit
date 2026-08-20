# Plan: Autopilot Merge-Mode Naming Symmetry + `stacked-prs` Mode

**Change**: 260820-t6rq-autopilot-merge-modes-stacked-prs
**Intake**: `intake.md`

## Requirements

### CLI: `--mode` Flag on `fab operator autopilot start`

#### R1: `--mode` flag with validation and default
`fab operator autopilot start` SHALL accept a new optional `--mode <name>` flag validating against exactly `stack-then-review`, `merge-on-complete`, and `stacked-prs`. When omitted, the mode SHALL default to `stack-then-review`. An unknown value SHALL exit non-zero with a one-line error, matching the shared state-verb error posture (`_cli-fab.md` § shared state-verb mechanics).

- **GIVEN** no autopilot queue is active
- **WHEN** `fab operator autopilot start --queue ab12,cd34 --mode stacked-prs` runs
- **THEN** the autopilot block is written with `mode: stacked-prs`

- **GIVEN** no autopilot queue is active
- **WHEN** `fab operator autopilot start --queue ab12 --mode merge-everything` runs
- **THEN** the command exits non-zero with a one-line error naming the valid modes and writes no state

#### R2: Persisted `mode` field with lifecycle retention and back-compat read
`autopilotState` (`src/go/fab/cmd/fab/operator_state.go`) SHALL gain a `Mode string \`yaml:"mode"\`` field. `start` writes it; `pause`/`resume`/`advance` retain it; queue exhaustion (advance past the last entry) retains it alongside `queue`/`completed`; `stop` clears it with the whole block (`autopilot: null`). A state file whose `autopilot` block lacks `mode` SHALL read as `stack-then-review` (tolerant read; the owned-section re-marshal adds the field on the next mutation). The binary stores, validates, and prints the mode (via `fab operator state`) — no merge choreography enters the binary.

- **GIVEN** a queue started with `--mode merge-on-complete`
- **WHEN** `pause`, `resume`, then `advance` run and the queue exhausts
- **THEN** the retained block still carries `mode: merge-on-complete` with `current: null, state: null`, and only `stop` clears it

- **GIVEN** a pre-existing state file whose `autopilot` block has `queue`/`current`/`completed`/`state` but no `mode`
- **WHEN** any autopilot verb reads it
- **THEN** the mode is treated as `stack-then-review` and no error occurs

#### R3: Tests ship with the Go change
`src/go/fab/cmd/fab/operator_autopilot_test.go` SHALL gain cases covering: flag parsing, default fill when `--mode` is omitted, validation error on an unknown value, persistence through the verb lifecycle (start → pause → resume → advance → exhaustion → stop), and the absent-field back-compat read (Constitution VII / Go-changes-ship-tests).

- **GIVEN** the new test cases
- **WHEN** the `cmd/fab` operator tests run
- **THEN** all pass

#### R4: `_cli-fab.md` documents the new surface
`src/kit/skills/_cli-fab.md` § fab operator autopilot SHALL show the new `start` signature (`--queue <id,id,...> [--mode <stack-then-review|merge-on-complete|stacked-prs>]`) and state-shape semantics (default, validation, retention on exhaustion, cleared by `stop`, absent-field default), per the constitution's CLI ⇒ docs + tests rule.

- **GIVEN** the updated `_cli-fab.md`
- **WHEN** an agent reads § fab operator autopilot
- **THEN** the signature and mode semantics match the binary's behavior exactly

### Skill Prose: Symmetric Mode Names (`fab-operator.md`)

#### R5: Three flat noun mode names; pseudo-flag spelling removed
`src/kit/skills/fab-operator.md` SHALL name the three merge modes as flat nouns throughout — `stack-then-review` (default), `merge-on-complete`, `stacked-prs` — and SHALL NOT spell `merge-on-complete` as a `--merge-on-complete` pseudo-flag anywhere. Mode selection is `fab operator autopilot start --mode <name>` (persisted per R2) or natural language mapping onto it: the existing equivalents for `merge-on-complete` ("merge as you go", "merge on complete", "merge each when done") stay, and `stacked-prs` gains its own (e.g. "stacked PRs", "stack the PRs"). The confirmation prompt SHALL gain a third per-mode line following the existing pattern (e.g. "stacked-prs: Confirm upfront (creates stacked PRs — merge after review)."). Sections in scope: § Autopilot (mode definitions, per-change loop, confirmation prompts, the `start --queue` persistence sentence gains `--mode`), § Dependency Resolution (mode-conditional same-repo strategy per R6), the **Failures** line, **Interrupts**, § Queue Completion Summary, § Ordered Merge.

- **GIVEN** the updated skill file
- **WHEN** `grep -- '--merge-on-complete' src/kit/skills/fab-operator.md` runs
- **THEN** it returns no matches, while the noun `merge-on-complete` remains as a mode name

#### R6: `stacked-prs` mode contract
`fab-operator.md` SHALL define `stacked-prs` as stack-then-review merge timing (PRs created up front, merged only on explicit user request) with stacked topology for same-repo chains: the dependent's branch is created off the dependency's **branch** (not `origin/{default_branch}` + cherry-pick — the squashed `"operator: cherry-pick"` commit does not exist for same-repo deps in this mode; the base-ref seam is the §6 spawn sequence's worktree/branch creation step, e.g. the `wt create --checkout <dep-branch>` route), and the dependent's PR base is retargeted to the dependency's branch by the operator after creation via `gh pr edit <pr> --base <dep-branch>` (`/git-pr` itself is unchanged and mode-unaware). Cross-repo dependencies remain ordering-only barriers in every mode. Dependency-branch drift after a dependent PR exists is out of scope (same exposure as today's cherry-pick model).

- **GIVEN** a `stacked-prs` queue `ab12 → cd34` in one repo
- **WHEN** the operator spawns `cd34`
- **THEN** `cd34`'s worktree branch is created off `ab12`'s branch with no cherry-pick commit, and after `/git-pr` creates `cd34`'s PR the operator retargets its base to `ab12`'s branch

#### R7: `stacked-prs` merge-all choreography and failure rows
`fab-operator.md` § Ordered Merge SHALL extend for `stacked-prs`: (1) base-first merge order per repo, waiting on CI per PR, as today; (2) after each base PR merges, rely on GitHub's base auto-retarget onto the default branch when the merged branch is deleted, and verify/retarget explicitly (`gh pr edit --base`) when it was not; (3) after each squash merge, rebase the next chain branch onto the default branch to drop the already-merged dependency commits (`git fetch origin && git rebase --onto origin/{default_branch} <merged-dep-branch> <next-branch>` + force-push), with `{default_branch}` resolved per Dependency Resolution step 0; (4) a rebase conflict during stacked-prs merge-all SHALL escalate (never silently skip), while the mid-queue rebase-conflict-skip row stays `merge-on-complete`-only — the **Failures** line SHALL distinguish the two.

- **GIVEN** a completed stacked-prs queue whose base PR was squash-merged
- **WHEN** the operator runs "merge all"
- **THEN** the next PR's branch is rebased `--onto` the default branch before its merge, and a conflict in that rebase halts with an escalation rather than a skip

### Docs Sweep

#### R8: Memory sweep + index regeneration
`docs/memory/runtime/operator.md` SHALL be updated to the three-mode model (mode names, `--mode` flag + persisted `mode` field in the state-block sentence, stacked-prs topology + merge-all choreography, updated failure-matrix row wording, confirmation texts including the third line, and the `--merge-on-complete` opt-in paragraph rewritten to the noun mode). `docs/memory/pipeline/change-lifecycle.md`'s "`--merge-on-complete` rebases" mention (default-branch-resolution convention) SHALL be rephrased to the noun mode. Memory indexes SHALL be regenerated via `fab memory-index` after the writes.

- **GIVEN** the updated memory files
- **WHEN** `grep -rn -- '--merge-on-complete' docs/memory/` runs
- **THEN** it returns no matches, and `fab memory-index --check` reports no drift

#### R9: Repo-wide sweep of remaining present-truth mode claims
Before finishing apply, a grep-driven sweep of `merge-on-complete` / `stack-then-review` / autopilot-mode claims SHALL cover every remaining present-truth occurrence (per code-quality.md § Sibling Sweeps, the class includes aggregate specs `docs/specs/operator.md`/`skills.md`/`glossary.md` even where the intake found no current mode-name occurrences). Historical records — spec version-history rows (e.g. `docs/specs/operator.md` v8 row), `log.md`, `log.seed.md`, archived changes — stay untouched per the FKF present-truth rule.

- **GIVEN** the finished apply tree
- **WHEN** `grep -rn -- '--merge-on-complete' src/kit/ docs/memory/ docs/specs/` runs (excluding archives/logs)
- **THEN** no pseudo-flag spellings remain in present-truth prose

### Non-Goals

- Dependency-branch drift after a dependent PR exists (a dep's review-pr rework moving its branch) — same exposure as today's cherry-pick model; conflicts surface at merge-all and escalate.
- A mid-queue mode-change verb — mode is fixed at `start` for the queue's lifetime; changing mode = `stop` + new `start`.
- Merge choreography in the Go binary — the binary stores/validates/prints the mode only.
- Teaching `/git-pr` a base parameter — the operator retargets post-creation.
- A `src/kit/migrations/` file — the operator state file is binary-owned, self-creating, and outside the migration policy's scope; the absent-field default is additive.

### Design Decisions

#### Operator-Side PR Retarget Over a `/git-pr` Base Parameter
**Decision**: In `stacked-prs`, the operator retargets each dependent PR's base after creation (`gh pr edit <pr> --base <dep-branch>`); `/git-pr` stays unchanged and mode-unaware.
**Why**: Mode is operator state — the worker agent running `/git-pr` inside the spawned pane has no mode awareness, and plumbing it through the spawn command adds a parameter to a shared skill for one consumer. Retarget-after-create is one `gh` call at a point where the operator already collects the PR URL.
**Rejected**: A `--base` parameter on `/git-pr` — couples a pipeline-generic skill to operator-only state and still needs the operator to compute the base.
*Introduced by*: 260820-t6rq-autopilot-merge-modes-stacked-prs

#### Mode Fixed at `start` for the Queue's Lifetime
**Decision**: `--mode` is set once at `fab operator autopilot start` and persisted; there is no mode-mutation verb. Changing mode means `stop` + a new `start`.
**Why**: A queue's dependency topology is built per-mode at spawn time (branch-off-dep vs cherry-pick) — switching mid-queue would leave half the queue in the wrong topology. The persisted-at-start design is also the minimal fix for the `/clear` durability gap.
**Rejected**: A `mode` mutation verb — could be added later without breaking this contract, but today it would imply a mid-queue topology switch the choreography cannot honor.
*Introduced by*: 260820-t6rq-autopilot-merge-modes-stacked-prs

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add `Mode string \`yaml:"mode"\`` to `autopilotState` in `src/go/fab/cmd/fab/operator_state.go`, with the absent-field default `stack-then-review` applied at read (e.g. a small accessor or normalization in `loadAutopilot`) <!-- R2 -->
- [x] T002 Add the `--mode` flag to `start` in `src/go/fab/cmd/fab/operator_autopilot.go`: default `stack-then-review`, validate against the three names (one-line non-zero error otherwise), write `Mode` in the start block; confirm `pause`/`resume`/`advance` retain it via the typed re-marshal and `stop` clears it <!-- R1 -->
- [x] T003 Extend `src/go/fab/cmd/fab/operator_autopilot_test.go`: flag parsing, default fill, unknown-value validation error, mode persistence through start → pause → resume → advance → exhaustion → stop, absent-field back-compat read; run the `cmd/fab` operator tests <!-- R3 -->
- [x] T004 [P] Update `src/kit/skills/_cli-fab.md` § fab operator autopilot: `start` signature with `[--mode <...>]`, default/validation/retention/clear semantics, absent-field read <!-- R4 -->
- [x] T005 Rewrite `src/kit/skills/fab-operator.md` autopilot prose: three flat mode names (kill every `--merge-on-complete` pseudo-flag spelling), `--mode` on the persist-the-queue sentence, third confirmation-prompt line, `stacked-prs` NL equivalents, mode definitions in § Autopilot <!-- R5 -->
- [x] T006 Add the `stacked-prs` mode contract to `src/kit/skills/fab-operator.md`: branch-off-dep-branch same-repo resolution (mode-conditional branch in § Dependency Resolution + the §6 spawn seam, `wt create --checkout <dep-branch>` route), post-creation PR retarget (`gh pr edit --base`), cross-repo unchanged <!-- R6 -->
- [x] T007 Extend `fab-operator.md` § Ordered Merge / § Queue Completion Summary with the stacked-prs merge-all choreography (auto-retarget rely-then-verify, post-squash `rebase --onto` + force-push) and split the **Failures** line's rebase-conflict rows (mid-queue skip = `merge-on-complete` only; stacked-prs merge-all = escalate) <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Update `docs/memory/runtime/operator.md` to the three-mode model (mode names, `--mode` + persisted `mode` field, stacked-prs topology + merge-all choreography, failure-matrix wording, confirmation texts) and rephrase `docs/memory/pipeline/change-lifecycle.md`'s "`--merge-on-complete` rebases" mention; regenerate indexes via `fab memory-index` <!-- R8 -->
- [x] T009 Grep-driven repo-wide sweep for `merge-on-complete` / `stack-then-review` / autopilot-mode claims across `src/kit/` and `docs/`; update any remaining present-truth occurrences, leaving historical records (spec version rows, `log.md`, `log.seed.md`, archives) untouched <!-- R9 -->

## Execution Order

- T001 blocks T002; T002 blocks T003.
- T004 is parallel to T005–T007 but must reflect T002's final surface.
- T005–T007 edit the same file — run sequentially in that order.
- T008–T009 run after all prose edits (the sweep verifies the finished tree).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab operator autopilot start` accepts `--mode` with the three valid values, defaults to `stack-then-review`, and rejects unknown values with a one-line non-zero error
- [x] A-002 R2: `autopilotState.Mode` persists through pause/resume/advance and exhaustion, is cleared by `stop`, and an absent field reads as `stack-then-review`
- [x] A-003 R5: `fab-operator.md` names the three modes as flat nouns with no `--merge-on-complete` pseudo-flag spelling; confirmation prompt has three per-mode lines; NL equivalents exist for all opt-in modes
- [x] A-004 R6: `fab-operator.md` defines the `stacked-prs` contract — branch off dep branch (no cherry-pick commit for same-repo deps), operator PR retarget post-creation, cross-repo barriers unchanged
- [x] A-005 R7: § Ordered Merge carries the stacked-prs merge-all choreography (retarget verify, post-squash `rebase --onto` + force-push) and the Failures line distinguishes the two rebase-conflict cases

### Behavioral Correctness

- [x] A-006 R2: `fab operator state` prints the `mode` field for an active queue; a queue started without `--mode` shows `mode: stack-then-review`
- [x] A-007 R4: `_cli-fab.md` § fab operator autopilot matches the binary's actual flag surface and state shape

### Scenario Coverage

- [x] A-008 R3: New test cases exist for flag parsing, default, validation, lifecycle persistence, and back-compat read, and the `cmd/fab` operator tests pass

### Edge Cases & Error Handling

- [x] A-009 R1: `--mode` with an unknown value writes no state (validation precedes the mutate)
- [x] A-010 R2: Queue exhaustion retains `mode` alongside `queue`/`completed` so the completion summary can still read it

### Documentation & Sweep

- [x] A-011 R8: `docs/memory/runtime/operator.md` and `docs/memory/pipeline/change-lifecycle.md` reflect the three-mode model; `grep -rn -- '--merge-on-complete' docs/memory/` is clean; indexes regenerated (`fab memory-index --check` passes)
- [x] A-012 R9: No present-truth `--merge-on-complete` pseudo-flag spelling remains in `src/kit/` or `docs/specs/`; historical records untouched

### Code Quality

- [x] A-013 Pattern consistency: the `--mode` flag/validation follows the existing state-verb style in `operator_autopilot.go` (cobra flag, one-line errors, `mutateOperatorState` closure)
- [x] A-014 Canonical source only: all skill edits land in `src/kit/skills/`, never `.claude/skills/`
- [x] A-015 Owner-or-pointer: no downstream file restates the mode semantics `_cli-fab.md`/`fab-operator.md` own — pointers only
- [x] A-016 No unnecessary duplication: the mode-validation list is a named set/constant, not repeated string literals

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The `--merge-on-complete` pseudo-flag prose removed by this change was deleted in the apply diff itself; no further discovered redundancy.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Absent-field default applied at read time (normalization in `loadAutopilot` or accessor), not a data migration | Matches the tolerant-read/typed-write posture `_cli-fab.md` documents; intake decision 7 | S:70 R:85 A:85 D:80 |
| 2 | Confident | Specs sweep (R9) expected near-no-op: grep found only historical/generic autopilot mentions in `docs/specs/` — the task verifies rather than rewrites | Direct grep evidence at plan time; sweep still runs to catch phrase-class variants | S:65 R:90 A:80 D:75 |
| 3 | Confident | `stacked-prs` confirmation line: "Confirm upfront (creates stacked PRs — merge after review)." | Intake supplied the example verbatim; mirrors the existing per-mode pattern | S:80 R:90 A:85 D:85 |
| 4 | Tentative | Branch-off-dep spawn seam documented as the `wt create --checkout <dep-branch>` route (probe-and-route already exists for existing branches), with branch-from-ref noted as the fallback wording | Intake deferred the exact seam to apply; the `--checkout` route reuses existing spawn choreography with the least new prose | S:55 R:75 A:55 D:50 |

4 assumptions (0 certain, 3 confident, 1 tentative).
