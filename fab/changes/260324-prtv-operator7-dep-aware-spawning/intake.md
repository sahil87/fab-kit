# Intake: Operator 7 — Dependency-Aware Agent Spawning

**Change**: 260324-prtv-operator7-dep-aware-spawning
**Created**: 2026-03-24
**Status**: Draft

## Origin

> Create operator7 — a copy of operator6 with dependency-aware agent spawning via cherry-pick

Extended conversational design session. User and agent discussed the dependency cherry-pick mechanism in detail, resolving cherry-pick vs merge-squash, range semantics (whole branch vs tip commit), origin/main as base, transitive dependency handling via leaf-only cherry-picking, and ancestor pruning for chains. User explicitly directed: "lets not touch operator6. Lets create an operator7 which is operator6 + the above change."

## Why

When the operator runs multiple changes concurrently (via autopilot or manual dispatch), some changes depend on others that haven't merged yet. Today, the operator spawns agents into worktrees branched from `origin/main` with no awareness of inter-change dependencies. The agent starts working without the code its change depends on, leading to:

1. **Build/test failures** — the agent's change references functions, types, or config that only exist in the dependency branch
2. **Spec-implementation divergence** — the agent implements against a stale baseline, producing code that conflicts with the dependency when both merge
3. **Manual intervention** — the user must notice the dependency gap, manually cherry-pick or rebase, and restart the agent

If unaddressed, the operator cannot reliably sequence dependent changes in autopilot mode. Every dependency chain requires manual babysitting, defeating the operator's "automate the routine" principle.

## What Changes

### New skill: `fab/.kit/skills/fab-operator7.md`

A copy of `fab-operator6.md` with two additions: **dependency-aware agent spawning** and **timestamp in idle messages**. All operator6 behavior (principles, startup, safety, loop, auto-nudge, coordination patterns, watches, configuration) carries forward unchanged.

### Pre-spawn dependency resolution (§6 addition)

A new step between worktree creation (`wt create`) and agent tab creation (`tmux new-window`) in §6 "Spawning an Agent":

**Step sequence becomes:**
1. Create worktree (`wt create ...`)
2. **Resolve dependencies** *(new)*
3. Open agent tab (`tmux new-window ...`)
4. Enroll in monitored set

**Resolve dependencies step:**

For each change ID in the monitored entry's `depends_on` list:

1. **Look up branch** — from `branch_map` (top-level in `.fab-operator.yaml`) or from the monitored entry's `branch` field if still active
2. **Prune redundant deps** — if dep A is an ancestor of dep B (both in depends_on), skip A. Check via `git merge-base --is-ancestor <A-branch> <B-branch>`. This prevents duplicate cherry-picks in chains where B's branch already carries A's content transitively
3. **Check if already present** — `git merge-base --is-ancestor <dep-branch> HEAD` in the worktree. Skip if already an ancestor
4. **Cherry-pick** — in the worktree directory:
   ```bash
   git cherry-pick --no-commit origin/main..<dep-branch> && \
   git commit -m "operator: cherry-pick <dep-change> dependency"
   ```
   This cherry-picks the full range of commits unique to the dependency branch since it diverged from main, stages them all without individual commits, then squashes into a single operator commit
5. **On conflict** — abort immediately, do not spawn:
   ```bash
   git cherry-pick --abort
   ```
   Log a warning: `"{change}: cherry-pick conflict with dependency {dep-change}. Escalating."`
   Escalate to user. Do not silently proceed without the dependency content

**Why `origin/main` as base**: Each dependency branch carries its full transitive dependency content. When operator spawned dep B, it cherry-picked dep A into B's worktree first. B's branch therefore contains A's commits. So `origin/main..<B-branch>` gives the complete transitive closure — no need to chase transitive deps manually. This is why only direct/leaf dependencies need cherry-picking.

### `.fab-operator.yaml` schema additions

**Monitored entry additions:**

```yaml
monitored:
  cd34:
    pane: "%5"
    stage: intake
    agent: active
    stop_stage: null
    spawned_by: null
    depends_on: [ab12]          # NEW — change IDs this depends on
    branch: feat/cd34-thing     # NEW — this change's branch name
    enrolled_at: "2026-03-24T10:00:00Z"
    last_transition: "2026-03-24T10:02:00Z"
```

- `depends_on` — list of change IDs that must be cherry-picked before spawning. Empty list or absent means no dependencies
- `branch` — this change's branch name, needed so other changes can cherry-pick from it

**New top-level section:**

```yaml
branch_map:
  ab12: 260324-ab12-fix-auth
  cd34: 260324-cd34-add-oauth
```

Persists branch names after changes leave the monitored set (merged, archived, pane died). Without this, a completed dependency's branch name is lost and downstream changes can't cherry-pick from it. Populated when changes are enrolled. Persists until operator session ends or user clears.

### Updated "Working a Change" flows (§6)

**From backlog ID or Linear issue** (structured flow):
1. Look up the idea or resolve the Linear issue
2. Create worktree
3. **Resolve dependencies** *(inserted)*
4. Spawn agent tab
5. Enroll in monitored set
6. On completion: merge PR, optionally archive

**From existing change**:
Same insertion — check `depends_on` before dispatching.

### Updated autopilot dispatch (§6)

The autopilot sequence becomes:
1. **Spawn** — create worktree
2. **Resolve dependencies** *(inserted)*
3. **Gate** — check confidence score
4. **Dispatch** — send `/fab-fff`
5. **Monitor** — normal tick detection
6. **Merge** — merge PR
7. **Rebase next** — rebase next queued change onto latest `origin/main`
8. **Cleanup** — optionally delete worktree
9. **Report**

### Timestamp in idle messages (§4 addition)

The operator's between-tick idle message currently reads `Waiting for next tick.` with no time context. When the user glances at the operator pane, they can't tell how long ago that message was printed or when the next tick fires.

**Change**: append the current time and next-tick time to idle messages:

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

The time is HH:MM in the operator's local timezone. The next-tick time is computed from the current loop interval (default 3m). This lets the user gauge staleness at a glance without scrolling to the last tick frame.

### Bounded retries addition (§3)

Add to the bounded retries table:

| Cherry-pick conflict | 0 | Abort, log, escalate. Do not spawn. |

### New launcher: `fab/.kit/scripts/fab-operator7.sh`

Follows the same structure as `fab-operator6.sh`, references `/fab-operator7`.

### Update deployed copies

After skill source changes, run `fab-sync.sh` to update `.claude/skills/` deployed copies.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add operator7 section documenting the dep-aware spawning addition over operator6

## Impact

- **Skills**: `fab/.kit/skills/fab-operator7.md` (new)
- **Scripts**: `fab/.kit/scripts/fab-operator7.sh` (new)
- **Deployed copies**: `.claude/skills/fab-operator7.md` updated via sync
- **No config.yaml changes** — operator is not referenced in project config
- **No template changes** — operator doesn't use artifact templates
- **No changes to operator6** — operator6 is preserved as-is

## Open Questions

- How should `depends_on` be declared? Three candidate paths were discussed: (1) explicit conversational ("cd34 depends on ab12"), (2) autopilot queue with ordering ("run ab12 then cd34, cd34 depends on ab12"), (3) `--base` flag implies `depends_on`. All three may coexist — the skill should document all as valid input methods.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | operator7 is a separate skill file, operator6 untouched | Discussed — user explicitly directed "lets not touch operator6" | S:95 R:95 A:95 D:95 |
| 2 | Certain | Use cherry-pick, not merge --squash | Discussed — user rejected merge-squash: "these are mostly for unattended sessions" | S:95 R:85 A:90 D:95 |
| 3 | Certain | Cherry-pick the full branch range, not just the tip commit | Discussed — user: "cherry-picks that looks at the whole branch instead of just the top commit" | S:95 R:85 A:90 D:95 |
| 4 | Certain | Command: `git cherry-pick --no-commit origin/main..<dep-branch> && git commit` | Discussed — squash into single operator commit | S:95 R:85 A:90 D:90 |
| 5 | Certain | On cherry-pick conflict: abort, log, escalate — do not spawn | From user's original description, reinforced in discussion | S:95 R:95 A:95 D:95 |
| 6 | Certain | Pre-spawn step sits between `wt create` and `tmux new-window` | From user's original description | S:95 R:90 A:90 D:95 |
| 7 | Certain | All operator6 behavior carries forward unchanged | Discussed — operator7 = operator6 + this one addition | S:95 R:90 A:95 D:95 |
| 8 | Confident | `origin/main` as cherry-pick base | Discussed extensively — user engaged critically about chain implications, didn't reject. Works because dep branches carry transitive content | S:85 R:80 A:85 D:80 |
| 9 | Confident | Only cherry-pick leaf/direct deps; prune redundant via `git merge-base --is-ancestor` | Discussed — follows from origin/main base decision. User engaged with the reasoning | S:85 R:80 A:85 D:85 |
| 10 | Confident | Schema: `depends_on` and `branch` on monitored entries, `branch_map` top-level | Proposed, user said "ok" and proceeded to next action | S:80 R:85 A:85 D:85 |
| 11 | Confident | Cherry-pick conflict bounded retry: 0, immediate escalation | Proposed in bounded retries table, consistent with user's "do not silently proceed" requirement | S:85 R:90 A:90 D:90 |
| 12 | Confident | New launcher script `fab-operator7.sh` following operator6 pattern | Follows established pattern from operator5→6 transition | S:80 R:90 A:90 D:90 |
| 13 | Certain | `--base` flag on autopilot implies `depends_on` | Clarified — user confirmed | S:95 R:85 A:90 D:90 |
| 14 | Certain | `branch_map` entries persist until session end or explicit clear | Clarified — user confirmed | S:95 R:80 A:90 D:90 |
| 15 | Certain | Idle message includes current time and next-tick time (e.g., "Time: 08:26 · next tick: 08:29") | Discussed — user requested timestamp for at-a-glance staleness | S:90 R:95 A:90 D:90 |

15 assumptions (10 certain, 5 confident, 0 tentative, 0 unresolved).
