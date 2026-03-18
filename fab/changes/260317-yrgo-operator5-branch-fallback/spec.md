# Spec: Operator5 — Use Case Registry, Branch Fallback, and Proactive Monitoring

**Change**: 260317-yrgo-operator5-branch-fallback
**Created**: 2026-03-18
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Modifying `fab/.kit/skills/fab-operator4.md` or `docs/specs/skills/SPEC-fab-operator4.md` — operator4 remains untouched
- Adding branch awareness to any `fab` CLI command — fab is orthogonal to git
- User-definable or plugin-style use cases — operator5 ships with a fixed set of 3 built-in use cases
- Per-use-case tick cadences — all use cases run on the same `/loop` interval

## Operator5 Skill: Structure

### Requirement: Operator5 SHALL be a standalone skill containing all of operator4 plus additions

The new skill file `fab/.kit/skills/fab-operator5.md` SHALL contain operator4's full content with the following modifications:
- Section 3 (Safety Model): new "Branch Fallback Resolution" subsection after "Pre-Send Validation"
- Section 4 (Monitoring System): replaced by "Use Case Registry" with `monitor-changes` as one use case
- Section 6 (Modes of Operation): renamed to "Playbooks" with identical content
- New sections for `linear-inbox`, `pr-freshness` use cases, and "Tab Preparation Procedure"

#### Scenario: Fresh skill file creation
- **GIVEN** operator4's skill file exists at `fab/.kit/skills/fab-operator4.md`
- **WHEN** the change is applied
- **THEN** `fab/.kit/skills/fab-operator5.md` is created with all operator4 content plus additions
- **AND** `fab/.kit/skills/fab-operator4.md` is NOT modified

#### Scenario: Spec file creation
- **GIVEN** operator4's spec exists at `docs/specs/skills/SPEC-fab-operator4.md`
- **WHEN** the change is applied
- **THEN** `docs/specs/skills/SPEC-fab-operator5.md` is created reflecting operator5's structure
- **AND** `docs/specs/skills/SPEC-fab-operator4.md` is NOT modified

## Use Case Registry

### Requirement: Operator5 SHALL manage a persistent use case registry via `.fab-operator.yaml`

The config file `.fab-operator.yaml` at the repo root SHALL define enabled/disabled use cases with optional per-use-case configuration. The operator reads this file on startup and on each tick.

```yaml
use_cases:
  monitor-changes:
    enabled: true
  linear-inbox:
    enabled: false
    config:
      assignee: "@me"
  pr-freshness:
    enabled: true
```

#### Scenario: Config file exists on startup
- **GIVEN** `.fab-operator.yaml` exists at repo root with `monitor-changes: enabled: true` and `linear-inbox: enabled: false`
- **WHEN** the operator starts
- **THEN** it reads the config and activates only `monitor-changes`
- **AND** reports `linear-inbox` as disabled in the status roster

#### Scenario: Config file missing on startup
- **GIVEN** `.fab-operator.yaml` does not exist
- **WHEN** the operator starts
- **THEN** it creates `.fab-operator.yaml` with all use cases set to their defaults (`monitor-changes: enabled`, others: disabled)

### Requirement: Use cases SHALL be toggleable via conversational commands

Users enable/disable use cases through natural language. The operator parses intent and writes to `.fab-operator.yaml`.

#### Scenario: Enable a disabled use case
- **GIVEN** `linear-inbox` is disabled in `.fab-operator.yaml`
- **WHEN** user says "Turn on Linear monitoring"
- **THEN** operator sets `linear-inbox.enabled: true` in `.fab-operator.yaml`
- **AND** reports: "Enabled: linear-inbox"

#### Scenario: Disable an enabled use case
- **GIVEN** `pr-freshness` is enabled
- **WHEN** user says "Stop watching for stale PRs"
- **THEN** operator sets `pr-freshness.enabled: false`
- **AND** reports: "Disabled: pr-freshness"

#### Scenario: Query active use cases
- **GIVEN** some use cases are enabled, some disabled
- **WHEN** user says "What's active?"
- **THEN** operator displays the status roster (same format as tick-start roster)

### Requirement: Each tick SHALL begin with a status roster

Every `/loop` tick displays all use cases with status indicators before running any checks.

```
── Operator tick (17:32) ──────────────
🟢 Monitor changes (3 agents: 2 active, 1 idle 4m)
🟢 PR freshness (2 PRs clean, 1 stale)
⚪ Linear inbox (disabled)
────────────────────────────────────────
```

- 🟢 = enabled and running (with one-line summary from that tick's check)
- ⚪ = disabled (shows "(disabled)")

#### Scenario: All use cases enabled
- **GIVEN** all three use cases are enabled
- **WHEN** a tick fires
- **THEN** all three show 🟢 with their respective summaries

#### Scenario: All use cases disabled
- **GIVEN** all use cases are disabled
- **WHEN** a tick fires
- **THEN** all three show ⚪ with "(disabled)"
- **AND** the operator reports "All use cases disabled. Loop will stop."

### Requirement: The loop SHALL run as long as any use case is enabled

Operator5's `/loop` lifecycle differs from operator4. The loop is the operator's heartbeat — it runs continuously while any use case is enabled.

#### Scenario: Last use case disabled
- **GIVEN** only `monitor-changes` is enabled and the loop is running
- **WHEN** user says "stop monitoring changes"
- **THEN** operator disables `monitor-changes` and stops the loop
- **AND** reports: "All use cases disabled. Loop stopped."

#### Scenario: First use case enabled
- **GIVEN** no use cases are enabled and no loop is running
- **WHEN** user says "turn on PR freshness"
- **THEN** operator enables `pr-freshness` and starts the loop

## Branch Fallback Resolution

### Requirement: On user-initiated resolution failure, the operator SHALL search branch names

When `fab resolve` returns non-zero during a user-initiated action (not monitoring ticks), the operator searches local and remote branch names as a fallback.

```bash
git for-each-ref --format='%(refname:short)' refs/heads/ refs/remotes/ | grep -iF "<query>"
```

#### Scenario: Single branch match — read-only query
- **GIVEN** `fab resolve ab12` fails (no local change folder)
- **AND** branch `260310-ab12-add-retry-logic` exists in `refs/heads/`
- **WHEN** user asks "what stage is ab12 at?"
- **THEN** operator reads `.status.yaml` via `git show 260310-ab12-add-retry-logic:fab/changes/260310-ab12-add-retry-logic/.status.yaml`
- **AND** reports the stage without creating a worktree

#### Scenario: Single branch match — action query
- **GIVEN** `fab resolve ab12` fails
- **AND** branch `260310-ab12-add-retry-logic` exists
- **WHEN** user says "send /fab-continue to ab12"
- **THEN** operator reports: "Can't find ab12 in any worktree. Found branch `260310-ab12-add-retry-logic`. Create a worktree for it?"
- **AND** on confirmation, runs `wt create --non-interactive --worktree-name <name> 260310-ab12-add-retry-logic`

#### Scenario: Multiple branch matches
- **GIVEN** `fab resolve retry` fails
- **AND** branches `260310-ab12-add-retry-logic` and `260315-cd34-fix-retry-timeout` both match
- **WHEN** user references "retry"
- **THEN** operator presents a disambiguation list

#### Scenario: No match anywhere
- **GIVEN** `fab resolve xyz` fails
- **AND** no branch names match "xyz"
- **WHEN** user references "xyz"
- **THEN** operator reports: "Not found locally or in any branch."

#### Scenario: Remote-only branch match
- **GIVEN** `fab resolve ab12` fails
- **AND** branch `origin/260310-ab12-add-retry-logic` exists in `refs/remotes/` but not in `refs/heads/`
- **WHEN** user asks about ab12
- **THEN** operator finds the remote branch and reports it (behavior same as local match)

### Requirement: Branch fallback SHALL NOT trigger during monitoring ticks

The branch fallback is user-initiated only. Monitoring ticks handle pane death via monitored set removal.

#### Scenario: Pane death during monitoring
- **GIVEN** a monitored change's pane dies
- **WHEN** the monitoring tick detects pane death
- **THEN** the operator removes it from the monitored set
- **AND** does NOT attempt branch fallback resolution

## Use Case: `monitor-changes`

### Requirement: `monitor-changes` SHALL be operator4's monitoring system reframed as a use case

All existing behavior from operator4 sections 4-5 (monitored set, monitoring tick with 6 steps, auto-nudge with question detection and answer model, stuck detection) is preserved identically, housed under the use case registry.

#### Scenario: Use case enabled
- **GIVEN** `monitor-changes` is enabled in `.fab-operator.yaml`
- **WHEN** a tick fires
- **THEN** the operator runs the full 6-step monitoring tick (stage advance, completion, review failure, pane death, auto-nudge, stuck detection)

#### Scenario: Use case disabled
- **GIVEN** `monitor-changes` is disabled
- **WHEN** a tick fires
- **THEN** monitoring is skipped entirely; shows ⚪ in roster

## Use Case: `linear-inbox`

### Requirement: `linear-inbox` SHALL detect new Linear issues assigned to the user

On each tick, query Linear for issues assigned to the configured user and compare against known changes.

#### Scenario: New issue detected
- **GIVEN** `linear-inbox` is enabled with `assignee: "@me"`
- **AND** Linear issue DEV-123 is assigned to the user with status "Todo"
- **AND** no active or archived change has DEV-123 in its `.status.yaml` `issues` array
- **WHEN** a tick fires
- **THEN** operator reports: "New issue: DEV-123 — {title}. Spawn an agent?"

#### Scenario: Issue already has a change
- **GIVEN** DEV-123 exists in Linear
- **AND** an active change has DEV-123 in its `.status.yaml` `issues` array
- **WHEN** a tick fires
- **THEN** DEV-123 is not reported as new (deduplication)

#### Scenario: User confirms spawn
- **GIVEN** operator reported DEV-123 as new
- **WHEN** user confirms
- **THEN** operator spawns worktree + agent with `/fab-new DEV-123`

## Use Case: `pr-freshness`

### Requirement: `pr-freshness` SHALL detect stale PRs and route rebase to agents

On each tick, query GitHub for open PRs and their merge state. Route rebase instructions to agents — the operator does NOT run `git rebase` itself.

```bash
gh pr list --author @me --state open --json number,headRefName,mergeStateStatus
```

#### Scenario: Stale PR with idle agent
- **GIVEN** PR #42 has `mergeStateStatus: "BEHIND"` and `headRefName: "260310-ab12-add-retry-logic"`
- **AND** pane-map shows an idle agent in a tab with that branch
- **WHEN** a tick fires
- **THEN** operator sends "rebase on main, resolve any conflicts, and push" to that pane

#### Scenario: Stale PR with busy agent
- **GIVEN** PR #42 is stale
- **AND** agent in matching tab is busy (not idle)
- **WHEN** a tick fires
- **THEN** operator skips and reports: "PR #42: stale but agent is busy"

#### Scenario: Stale PR with no agent
- **GIVEN** PR #42 is stale
- **AND** no agent tab matches the branch
- **WHEN** a tick fires
- **THEN** operator reports: "PR #42 (260310-ab12-add-retry-logic): stale but no agent running. Spawn one?"

#### Scenario: PR with conflicts
- **GIVEN** PR #42 has `mergeStateStatus: "DIRTY"`
- **AND** matching agent tab is idle
- **WHEN** a tick fires
- **THEN** operator sends rebase instruction to the agent
- **AND** flags: "PR #42: has conflicts — agent may need help"

#### Scenario: Clean PR
- **GIVEN** PR #42 has `mergeStateStatus: "CLEAN"`
- **WHEN** a tick fires
- **THEN** no action taken for PR #42

## Tab Preparation Procedure

### Requirement: Before dispatching work, the operator SHALL ensure the tab is ready

This is a shared procedure used by playbooks and use cases that send commands to agent tabs.

Steps:
1. Verify pane exists (refresh pane map)
2. Check agent is idle (warn if busy, require confirmation)
3. Check change is active in that tab (send `/fab-switch <change>` if not)
4. Check branch alignment (send `/git-branch` if branch doesn't match change folder)
5. Dispatch the command

#### Scenario: Tab needs switch and branch alignment
- **GIVEN** an agent tab exists but a different change is active there
- **AND** the git branch doesn't match the target change
- **WHEN** operator prepares to dispatch `/fab-fff` to that tab
- **THEN** operator sends `/fab-switch <change>` first
- **AND** then sends `/git-branch`
- **AND** then sends `/fab-fff`

#### Scenario: Tab is ready
- **GIVEN** the target change is active and branch matches
- **WHEN** operator prepares to dispatch
- **THEN** operator sends the command directly (no switch or branch alignment needed)

## Playbooks

### Requirement: Operator4's "Modes of Operation" SHALL be renamed to "Playbooks"

The 9 coordination patterns from operator4 Section 6 are carried over with the label "Playbooks" — documented on-demand coordination patterns, not modes or states. All playbooks that dispatch work to a tab use the Tab Preparation Procedure.

Playbooks: Broadcast, Sequenced rebase, Merge PRs, Spawn agent, Status dashboard, Unstick agent, Notification, Rebase all, Autopilot.

#### Scenario: User asks to broadcast
- **GIVEN** user says "send /fab-continue to all idle agents"
- **WHEN** operator follows the Broadcast playbook
- **THEN** it runs Tab Preparation Procedure for each target tab before sending

## Legacy Script Cleanup

### Requirement: Legacy operator launcher scripts SHALL be deleted

Delete `fab/.kit/scripts/fab-operator1.sh`, `fab-operator2.sh`, and `fab-operator3.sh`. These are obsolete leftovers from the inheritance chain. `fab-operator4.sh` remains.

#### Scenario: Scripts deleted
- **GIVEN** `fab/.kit/scripts/fab-operator{1,2,3}.sh` exist
- **WHEN** the change is applied
- **THEN** all three are deleted
- **AND** `fab/.kit/scripts/fab-operator4.sh` is NOT modified

## Design Decisions

1. **Standalone operator5 over modifying operator4**: Operator4 remains as the previous stable version. New capabilities go into operator5. This is consistent with the operator1→2→3→4 versioning pattern (though we're now cleaning up the old ones).
   - *Why*: User explicitly requested operator4 remain untouched.
   - *Rejected*: In-place modification of operator4 — would lose the known-good version.

2. **Branch name matching over branch content inspection**: The branch fallback scans `refs/heads/` and `refs/remotes/` ref names, not `git ls-tree` or `git show` for folder existence. Branch names follow the same `YYMMDD-{id}-{slug}` convention as change folders.
   - *Why*: Much cheaper (ref listing vs tree reads), and the naming convention makes content inspection unnecessary for identification.
   - *Rejected*: `git ls-tree <branch> fab/changes/` — slower, unnecessary when names encode identity.

3. **Fixed use case set over extensible plugin model**: Operator5 ships with 3 built-in use cases. New use cases require a new operator version.
   - *Why*: Keeps the skill file self-contained and fully specified. Plugin architecture adds complexity without clear need.
   - *Rejected*: User-definable use cases via config — would need a generic execution model.

4. **Playbooks over modes**: Renamed "Modes of Operation" to "Playbooks" to reflect that these are coordination recipes, not state the operator switches between.
   - *Why*: User observed these are "a guidebook/cheat sheet," not modes.
   - *Rejected*: "Interactive modes" — implies state machine behavior that doesn't exist.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Create operator5, do not modify operator4. No CLI changes | Confirmed from intake #1 — user explicitly rejected CLI-level approaches and requested operator5 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Branch name matching via `git for-each-ref` + grep | Confirmed from intake #2 — user confirmed name matching, not content inspection | S:90 R:95 A:90 D:90 |
| 3 | Certain | Same substring/ID matching as existing resolution | Confirmed from intake #3 — naming convention makes this natural | S:85 R:90 A:95 D:95 |
| 4 | Confident | Offer worktree creation when branch match found (action queries) | Confirmed from intake #4 — natural next step per `_naming.md` spawning rules | S:75 R:80 A:85 D:80 |
| 5 | Confident | Single-match auto-report, multi-match disambiguation | Confirmed from intake #5 — consistent with `fab resolve` patterns | S:70 R:85 A:85 D:80 |
| 6 | Certain | Branch fallback in Section 3 Safety Model, after Pre-Send Validation | Confirmed from intake #6 — user chose this placement | S:95 R:90 A:60 D:50 |
| 7 | Certain | Branch fallback is user-initiated only, not monitoring ticks | Confirmed from intake #7 — monitoring handles pane death differently | S:95 R:90 A:90 D:95 |
| 8 | Certain | Search both local and remote branches | Confirmed from intake #8 — user requested remote | S:95 R:85 A:90 D:95 |
| 9 | Certain | Read-only queries use `git show`, action queries offer worktree | Confirmed from intake #9 — proportional response | S:95 R:90 A:90 D:90 |
| 10 | Certain | Use case registry with persistent `.fab-operator.yaml` config | Confirmed from intake #10 — user designed this model | S:95 R:85 A:90 D:95 |
| 11 | Certain | Config at `.fab-operator.yaml` repo root | Confirmed from intake #11 — hidden file, persistent | S:90 R:90 A:85 D:85 |
| 12 | Certain | Tick-start roster with 🟢/⚪ indicators | Confirmed from intake #12 — user specified format | S:95 R:95 A:95 D:95 |
| 13 | Certain | Loop runs while any use case enabled | Confirmed from intake #13 — heartbeat model | S:85 R:85 A:90 D:90 |
| 14 | Certain | `monitor-changes` = operator4 monitoring reframed | Confirmed from intake #14 | S:90 R:90 A:95 D:95 |
| 15 | Certain | `linear-inbox` watches for assigned issues | Confirmed from intake #15 | S:90 R:80 A:80 D:85 |
| 16 | Certain | `pr-freshness` detects stale PRs via `gh` mergeStateStatus | Confirmed from intake #16 | S:90 R:85 A:90 D:90 |
| 17 | Certain | PR rebase routed to agents, not executed by operator | Confirmed from intake #17 — user explicit | S:95 R:85 A:95 D:95 |
| 18 | Confident | Fixed set of 3 use cases, not extensible | Carried from intake #18 — consistent with versioning pattern | S:60 R:85 A:80 D:70 |
| 19 | Confident | All use cases same tick cadence | Carried from intake #19 — simpler model | S:55 R:90 A:80 D:75 |
| 20 | Certain | Tab preparation: switch → branch → dispatch | Confirmed from intake #20 — user specified sequence | S:90 R:85 A:90 D:90 |
| 21 | Certain | "Modes of Operation" renamed to "Playbooks" | Confirmed from intake #21 — user agreed | S:95 R:95 A:90 D:90 |
| 22 | Certain | 3 use cases + 9 playbooks = total capabilities | Confirmed from intake #22 | S:90 R:90 A:95 D:95 |
| 23 | Confident | `.fab-operator.yaml` created with defaults when missing | Spec-level decision — reasonable UX, easily reversed | S:70 R:90 A:85 D:80 |
| 24 | Confident | Linear query uses MCP `list_issues` with status filter | Spec-level decision — MCP is the available interface for Linear | S:65 R:85 A:75 D:75 |

24 assumptions (19 certain, 5 confident, 0 tentative).
