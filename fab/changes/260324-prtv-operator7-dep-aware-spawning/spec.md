# Spec: Operator 7 — Dependency-Aware Agent Spawning

**Change**: 260324-prtv-operator7-dep-aware-spawning
**Created**: 2026-03-24
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying `fab-operator6.md` — operator6 is preserved as-is
- Automatic dependency inference from code analysis — dependencies are explicitly declared
- Transitive dependency resolution — leaf deps carry transitive content via `origin/main` base

## Operator Skill: File Structure

### Requirement: operator7 is operator6 plus targeted additions

`fab/.kit/skills/fab-operator7.md` SHALL be a copy of `fab-operator6.md` with the additions specified in this change. All operator6 sections (§1 Principles, §2 Startup, §3 Safety, §4 The Loop, §5 Auto-Nudge, §6 Coordination Patterns, §7 Watches, §8 Configuration, §9 Key Properties) SHALL carry forward unchanged except where this spec adds to them.

#### Scenario: Skill file created
- **GIVEN** `fab/.kit/skills/fab-operator6.md` exists
- **WHEN** operator7 is created
- **THEN** `fab/.kit/skills/fab-operator7.md` exists with all operator6 content plus the additions below
- **AND** `fab-operator6.md` is not modified

#### Scenario: Deployed copy synced
- **GIVEN** `fab-operator7.md` exists in `fab/.kit/skills/`
- **WHEN** `fab-sync.sh` runs
- **THEN** `.claude/skills/fab-operator7.md` is created as a deployed copy

## Operator Skill: `.fab-operator.yaml` Schema

### Requirement: Monitored entries support dependency tracking

Each monitored entry in `.fab-operator.yaml` SHALL support two new optional fields: `depends_on` and `branch`.

`depends_on` SHALL be a list of change IDs (4-char) that must be cherry-picked into the worktree before the agent is spawned. An empty list or absent field means no dependencies.

`branch` SHALL be the change's git branch name, populated when the change is enrolled in the monitored set. This field is needed so other changes can look up the branch to cherry-pick from.

#### Scenario: Monitored entry with dependencies
- **GIVEN** a change `cd34` depends on `ab12`
- **WHEN** `cd34` is enrolled in the monitored set
- **THEN** its entry includes `depends_on: [ab12]` and `branch: 260324-cd34-add-oauth`

#### Scenario: Monitored entry without dependencies
- **GIVEN** a change `ef56` has no dependencies
- **WHEN** `ef56` is enrolled in the monitored set
- **THEN** its entry has `depends_on: []` or the field is absent
- **AND** `branch` is still populated with the change's branch name

### Requirement: branch_map persists branch names after removal

`.fab-operator.yaml` SHALL include a top-level `branch_map` section that maps change IDs to branch names. Entries SHALL be added when changes are enrolled in the monitored set. Entries SHALL persist after changes leave the monitored set (merged, archived, pane died). Entries SHALL persist until the operator session ends or the user explicitly clears them.

#### Scenario: Completed dependency's branch is available
- **GIVEN** change `ab12` was monitored with branch `260324-ab12-fix-auth`
- **WHEN** `ab12` is removed from the monitored set (merged)
- **THEN** `branch_map.ab12` remains `260324-ab12-fix-auth`
- **AND** change `cd34` with `depends_on: [ab12]` can look up the branch

#### Scenario: branch_map populated on enrollment
- **GIVEN** change `cd34` is enrolled with branch `260324-cd34-add-oauth`
- **WHEN** enrollment completes
- **THEN** `branch_map.cd34` equals `260324-cd34-add-oauth`

## Operator Skill: Pre-Spawn Dependency Resolution

### Requirement: Dependencies resolved before agent spawn

When spawning an agent for a change with a non-empty `depends_on` list, the operator SHALL resolve dependencies after worktree creation (`wt create`) and before agent tab creation (`tmux new-window`).

The spawn sequence in §6 "Spawning an Agent" SHALL be:
1. Create worktree (`wt create ...`)
2. Resolve dependencies
3. Open agent tab (`tmux new-window ...`)
4. Enroll in monitored set

#### Scenario: Agent spawned with dependency
- **GIVEN** change `cd34` has `depends_on: [ab12]`
- **AND** `ab12`'s branch is in `branch_map` or the monitored set
- **WHEN** the operator spawns `cd34`
- **THEN** the worktree is created first
- **AND** `ab12`'s commits are cherry-picked into the worktree
- **AND** the agent tab is opened after cherry-pick completes

#### Scenario: Agent spawned without dependencies
- **GIVEN** change `ef56` has empty `depends_on`
- **WHEN** the operator spawns `ef56`
- **THEN** the dependency resolution step is a no-op
- **AND** the agent tab is opened immediately after worktree creation

### Requirement: Branch lookup from branch_map or monitored set

For each dependency change ID, the operator SHALL look up the branch name first from the monitored entry's `branch` field (if the dep is still active), then from `branch_map` (if the dep has left the monitored set).

#### Scenario: Dependency still active
- **GIVEN** dep `ab12` is in the monitored set with `branch: 260324-ab12-fix-auth`
- **WHEN** the operator looks up `ab12`'s branch
- **THEN** it uses `260324-ab12-fix-auth` from the monitored entry

#### Scenario: Dependency completed
- **GIVEN** dep `ab12` is not in the monitored set
- **AND** `branch_map.ab12` is `260324-ab12-fix-auth`
- **WHEN** the operator looks up `ab12`'s branch
- **THEN** it uses `260324-ab12-fix-auth` from `branch_map`

#### Scenario: Dependency branch not found
- **GIVEN** dep `ab12` is not in the monitored set and not in `branch_map`
- **WHEN** the operator looks up `ab12`'s branch
- **THEN** the operator logs a warning: `"{change}: dependency ab12 branch not found. Escalating."`
- **AND** escalates to the user
- **AND** does not spawn the agent

### Requirement: Redundant dependency pruning

Before cherry-picking, the operator SHALL prune redundant dependencies from the `depends_on` list. If dep A's branch is an ancestor of dep B's branch (both in `depends_on`), dep A SHALL be skipped because B's branch already carries A's content transitively.

The check SHALL use `git merge-base --is-ancestor <A-branch> <B-branch>`.

#### Scenario: Chain A → B → C with transitive deps
- **GIVEN** change `C` has `depends_on: [A, B]`
- **AND** B's branch contains A's content (B was spawned with A cherry-picked)
- **WHEN** the operator prunes redundant deps
- **THEN** A is skipped (A's branch is an ancestor of B's branch)
- **AND** only B is cherry-picked

#### Scenario: Independent deps
- **GIVEN** change `C` has `depends_on: [A, B]`
- **AND** A's branch is NOT an ancestor of B's branch (independent changes)
- **WHEN** the operator prunes redundant deps
- **THEN** both A and B are cherry-picked

### Requirement: Skip already-present dependencies

For each non-pruned dependency, the operator SHALL check if the dependency branch is already an ancestor of the worktree's HEAD via `git merge-base --is-ancestor <dep-branch> HEAD`. If so, the cherry-pick SHALL be skipped.

#### Scenario: Dependency already merged into main
- **GIVEN** dep `ab12`'s branch has been merged into `origin/main`
- **AND** the worktree was created from `origin/main`
- **WHEN** the operator checks if `ab12` is already present
- **THEN** `git merge-base --is-ancestor` returns true
- **AND** the cherry-pick is skipped

### Requirement: Cherry-pick uses full branch range from origin/main

Each cherry-pick SHALL use the command:

```bash
git cherry-pick --no-commit origin/main..<dep-branch> && \
git commit -m "operator: cherry-pick <dep-change> dependency"
```

This cherry-picks all commits unique to the dependency branch since it diverged from `origin/main`, stages them without individual commits, and squashes into a single operator commit.

The `origin/main` base is correct because each dependency branch carries its full transitive dependency content — when the operator spawned that dependency, it cherry-picked that dependency's own dependencies first.

#### Scenario: Dependency with multiple commits
- **GIVEN** dep `ab12`'s branch has 3 commits since `origin/main`
- **WHEN** the operator cherry-picks
- **THEN** all 3 commits are applied with `--no-commit`
- **AND** a single commit `"operator: cherry-pick ab12 dependency"` is created

#### Scenario: Dependency is a single commit
- **GIVEN** dep `ab12`'s branch has 1 commit since `origin/main`
- **WHEN** the operator cherry-picks
- **THEN** the single commit is applied with `--no-commit`
- **AND** a single commit `"operator: cherry-pick ab12 dependency"` is created

### Requirement: Cherry-pick conflict aborts spawn

On cherry-pick conflict, the operator SHALL:
1. Run `git cherry-pick --abort`
2. Log: `"{change}: cherry-pick conflict with dependency {dep-change}. Escalating."`
3. Escalate to the user
4. NOT spawn the agent

The operator SHALL NOT silently proceed without the dependency content. The operator SHALL NOT retry the cherry-pick (bounded retry: 0).

#### Scenario: Conflict during cherry-pick
- **GIVEN** dep `ab12`'s commits conflict with the worktree state
- **WHEN** the cherry-pick fails
- **THEN** `git cherry-pick --abort` is run
- **AND** the warning is logged
- **AND** the user is notified
- **AND** the agent tab is NOT opened
- **AND** the change is NOT enrolled in the monitored set

## Operator Skill: Updated Coordination Flows

### Requirement: Working a Change includes dependency resolution

The "Working a Change" flows in §6 SHALL insert the dependency resolution step between worktree creation and agent tab creation.

**From backlog ID or Linear issue** (structured flow):
1. Look up the idea or resolve the Linear issue
2. Create worktree
3. Resolve dependencies
4. Spawn agent tab
5. Enroll in monitored set
6. On completion: merge PR, optionally archive

**From existing change**: same insertion.

#### Scenario: Structured flow with dependency
- **GIVEN** a backlog item `cd34` with `depends_on: [ab12]`
- **WHEN** the operator works the change
- **THEN** step 3 (resolve dependencies) cherry-picks `ab12` before step 4 (spawn)

### Requirement: Autopilot dispatch includes dependency resolution

The autopilot dispatch sequence SHALL insert the dependency resolution step after worktree creation and before the confidence gate:

1. Spawn — create worktree
2. Resolve dependencies
3. Gate — check confidence score
4. Dispatch — send `/fab-fff`
5. Monitor — normal tick detection
6. Merge — merge PR
7. Rebase next
8. Cleanup
9. Report

#### Scenario: Autopilot with chained changes
- **GIVEN** an autopilot queue `[ab12, cd34]` where `cd34` depends on `ab12`
- **WHEN** `ab12` completes and `cd34` is dispatched
- **THEN** `cd34`'s worktree is created
- **AND** `ab12`'s commits are cherry-picked (from `branch_map`)
- **AND** then confidence is gated
- **AND** then `/fab-fff` is dispatched

### Requirement: --base flag implies depends_on

When the autopilot `--base <prev-change>` flag is used, it SHALL imply `depends_on: [<prev-change-id>]` for the subsequent change in the queue. This is in addition to the explicit conversational and queue-based declaration paths.

#### Scenario: Autopilot with --base
- **GIVEN** an autopilot queue with `--base ab12` specified for `cd34`
- **WHEN** the operator enrolls `cd34`
- **THEN** `cd34.depends_on` includes `ab12`

## Operator Skill: Dependency Declaration

### Requirement: Three declaration paths

Dependencies SHALL be declarable through three conversational paths, all of which coexist:

1. **Explicit**: "cd34 depends on ab12" — operator sets `depends_on` directly
2. **Autopilot queue**: "run ab12 then cd34, cd34 depends on ab12" — operator populates `depends_on` during queue setup
3. **--base flag**: autopilot `--base <prev-change>` implies `depends_on`

#### Scenario: Explicit declaration
- **GIVEN** user says "cd34 depends on ab12"
- **WHEN** the operator processes the instruction
- **THEN** `cd34`'s monitored entry has `depends_on: [ab12]`

#### Scenario: Multiple dependencies
- **GIVEN** user says "ef56 depends on ab12 and cd34"
- **WHEN** the operator processes the instruction
- **THEN** `ef56`'s monitored entry has `depends_on: [ab12, cd34]`

## Operator Skill: Bounded Retries

### Requirement: Cherry-pick conflict in bounded retries table

§3 bounded retries table SHALL include an entry for cherry-pick conflicts:

| Situation | Max retries | Escalation |
|-----------|-------------|------------|
| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |

#### Scenario: Cherry-pick conflict retry behavior
- **GIVEN** a cherry-pick conflict occurs
- **WHEN** the operator evaluates retry policy
- **THEN** no retry is attempted
- **AND** the operator escalates immediately

## Operator Skill: Idle Message Timestamp

### Requirement: Idle messages include current and next-tick time

The operator's between-tick idle message SHALL include the current time and computed next-tick time in HH:MM format (local timezone):

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

The next-tick time SHALL be computed by adding the current loop interval (default 3m) to the current time.

#### Scenario: Standard 3-minute interval
- **GIVEN** current time is 08:26 and loop interval is 3m
- **WHEN** the operator prints its idle message
- **THEN** the message reads `Waiting for next tick. Time: 08:26 · next tick: 08:29`

#### Scenario: Custom interval
- **GIVEN** current time is 14:10 and loop interval is 5m (user override)
- **WHEN** the operator prints its idle message
- **THEN** the message reads `Waiting for next tick. Time: 14:10 · next tick: 14:15`

## Operator Skill: Launcher Script

### Requirement: New launcher script

`fab/.kit/scripts/fab-operator7.sh` SHALL follow the same structure as `fab-operator6.sh`, referencing `/fab-operator7` instead of `/fab-operator6`.

#### Scenario: Launcher creates operator tab
- **GIVEN** `fab-operator7.sh` is executed
- **WHEN** tmux is active
- **THEN** a singleton tmux tab named `operator` is created
- **AND** it invokes `/fab-operator7`

## Design Decisions

1. **Cherry-pick over merge --squash**: Cherry-pick chosen for unattended sessions where merge machinery introduces risk. `--no-commit` with explicit `git commit` produces a clean squashed operator commit.
   - *Why*: Unattended sessions need predictable, non-interactive git operations
   - *Rejected*: `git merge --squash` — user explicitly rejected for unattended use

2. **origin/main as cherry-pick base**: Each dep branch carries its full transitive content because the operator cherry-picked that dep's own deps when it was spawned.
   - *Why*: Avoids needing to chase transitive dependency graphs; `origin/main..<dep-branch>` gives the complete closure
   - *Rejected*: Dynamic base computation — adds complexity without benefit since dep branches are self-contained

3. **Leaf-only cherry-picking with ancestor pruning**: Only direct deps are cherry-picked; redundant deps (ancestors of other deps in the list) are pruned via `git merge-base --is-ancestor`.
   - *Why*: Prevents duplicate cherry-picks that would cause conflicts in chains
   - *Rejected*: Cherry-pick all declared deps — causes conflicts when B contains A transitively

4. **branch_map for post-removal persistence**: A top-level map in `.fab-operator.yaml` preserves branch→name mappings after changes leave the monitored set.
   - *Why*: Without it, completed dependencies' branch names are lost and downstream changes can't cherry-pick
   - *Rejected*: Re-scan git branches — unreliable, expensive, naming ambiguity risk

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | operator7 is a separate skill file, operator6 untouched | Confirmed from intake #1 — user explicitly directed | S:95 R:95 A:95 D:95 |
| 2 | Certain | Use cherry-pick, not merge --squash | Confirmed from intake #2 — user rejected merge-squash | S:95 R:85 A:90 D:95 |
| 3 | Certain | Cherry-pick the full branch range, not just tip commit | Confirmed from intake #3 — user explicit | S:95 R:85 A:90 D:95 |
| 4 | Certain | Command: `git cherry-pick --no-commit origin/main..<dep-branch> && git commit` | Confirmed from intake #4 | S:95 R:85 A:90 D:90 |
| 5 | Certain | On conflict: abort, log, escalate — do not spawn | Confirmed from intake #5 | S:95 R:95 A:95 D:95 |
| 6 | Certain | Pre-spawn step between `wt create` and `tmux new-window` | Confirmed from intake #6 | S:95 R:90 A:90 D:95 |
| 7 | Certain | All operator6 behavior carries forward unchanged | Confirmed from intake #7 | S:95 R:90 A:95 D:95 |
| 8 | Certain | `origin/main` as cherry-pick base | Upgraded from intake Confident #8 — user engaged critically and confirmed | S:90 R:85 A:90 D:85 |
| 9 | Certain | Only cherry-pick leaf/direct deps; prune via ancestor check | Upgraded from intake Confident #9 — user confirmed after explanation | S:90 R:85 A:90 D:90 |
| 10 | Certain | Schema: `depends_on` and `branch` on monitored, `branch_map` top-level | Upgraded from intake Confident #10 — user confirmed | S:90 R:85 A:90 D:90 |
| 11 | Certain | Cherry-pick conflict bounded retry: 0, immediate escalation | Upgraded from intake Confident #11 — user confirmed | S:90 R:90 A:90 D:90 |
| 12 | Certain | New launcher `fab-operator7.sh` following operator6 pattern | Upgraded from intake Confident #12 — user confirmed | S:90 R:90 A:90 D:90 |
| 13 | Certain | `--base` flag on autopilot implies `depends_on` | Confirmed from intake #13 — user confirmed | S:95 R:85 A:90 D:90 |
| 14 | Certain | `branch_map` entries persist until session end or explicit clear | Confirmed from intake #14 — user confirmed | S:95 R:80 A:90 D:90 |
| 15 | Certain | Idle message includes current time and next-tick time | Confirmed from intake #15 — user requested | S:90 R:95 A:90 D:90 |
| 16 | Certain | Three dependency declaration paths coexist (explicit, queue, --base) | Spec-level decision — all three are natural input modes, no conflict between them | S:85 R:90 A:90 D:85 |
| 17 | Certain | Missing dependency branch → escalate, do not spawn | Spec-level decision — consistent with conflict escalation pattern | S:85 R:90 A:95 D:90 |

17 assumptions (17 certain, 0 confident, 0 tentative, 0 unresolved).
