# Intake: Add `/fab-operator1` Skill

**Change**: 260306-qkov-operator1-skill
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Extended `/fab-discuss` session exploring multi-agent coordination in tmux. The user described wanting an "orchestrator" that can observe agents running in tmux panes and interact with them as the user — sending commands, coordinating rebases, spawning worktrees. The name evolved from "orchestrator" to "connector" to "conductor" to "operator1" (experiment 1). A detailed spec was drafted at `docs/specs/skills/SPEC-fab-conductor.md` during the discussion (pre-rename).

Conversational — multiple rounds of design refinement before this intake.

## Why

1. **Tab-hopping doesn't scale** — the assembly-line workflow (`batch-fab-switch-change`, `batch-fab-new-backlog`) creates N tmux tabs with N agents. Today, coordinating them requires the user to manually hop between tabs: checking status, sending commands, sequencing rebases. With 5+ agents this becomes the bottleneck.

2. **Cross-agent actions have no home** — rebasing agent B after agent A finishes, broadcasting `/fab-continue` to all idle agents, merging completed PRs in order — these require awareness of multiple agents simultaneously. No existing skill or script handles this. The user does it manually.

3. **Natural language over scripts** — `batch-fab-switch-change --all` works but is rigid. "Start working on the retry logic idea" is more natural than remembering script flags and manually creating worktrees. The operator translates intent into multi-agent actions.

## What Changes

### New skill: `/fab-operator1`

A Claude Code skill that runs in a dedicated tmux pane. It loads the always-load layer (like `/fab-discuss`) but NOT change-specific artifacts — it only needs coordination primitives.

### Primitives consumed

The operator relies on:

| Primitive | Source | Purpose |
|-----------|--------|---------|
| `fab pane-map` | New CLI subcommand (separate change bh45) | Unified view: pane → worktree → change → stage → agent state |
| `fab status show --all` | Existing CLI | Change stage and progress |
| `fab runtime` | Existing CLI | Agent idle/active detection |
| `fab send-keys <change> "<text>"` | **New** CLI subcommand | Send text to a target agent's tmux pane |

### New subcommand: `fab send-keys`

```
fab send-keys <change> "<text>"
```

Resolves `<change>` to a tmux pane (via pane-map data), then runs `tmux send-keys -t <pane> "<text>" Enter`. Uses standard change resolution (4-char ID, substring, full name).

Safety requirements:
- Validate target pane exists before sending
- Return error if pane is gone (agent exited, tab closed)
- The operator checks idle state before calling `send-keys`; the subcommand itself does NOT enforce idle checks (separation of concerns — policy in the skill, mechanism in the CLI)

### Use cases

**1. Broadcast a command to all agents**
User: "run /fab-continue in all idle agents"
- Refreshes pane map, filters for idle agents, sends `/fab-continue` to each via `fab send-keys`

**2. Sequenced rebase after completion**
User: "when r3m7 finishes, rebase k8ds on main"
- Polls `fab status show --all`, waits for r3m7 to reach target stage, then sends rebase command to k8ds

**3. Merge completed PRs**
User: "merge all PRs that are ready"
- Checks for changes at `ship` or `review-pr (pass)` stage, runs `gh pr merge` directly from operator's shell

**4. Spawn new worktree + agent from idea**
User: "start working on the retry logic idea"
- Looks up idea via `idea show`, creates worktree via `wt-create --non-interactive`, opens tmux tab, sends `/fab-new <description>`

**5. Status dashboard**
User: "what's everyone doing?"
- Runs `fab pane-map`, presents human-readable summary

**6. Unstick a stuck agent**
User: "r3m7 looks stuck, nudge it"
- Confirms agent is idle at a non-terminal stage, reads current stage, sends `/fab-continue` or a targeted prompt
- Can diagnose via `tmux capture-pane -t <pane> -p` to read recent terminal output and reason about what went wrong

**7. Notification surface**
The operator pane is the single notification surface for cross-agent events. "Tell me when r3m7 finishes" or "notify me if anything fails" — the operator holds the instruction in conversation context and checks on the next user interaction. User-driven, not event-driven.

### Interaction model

- Long-running Claude session in its own tmux pane
- NOT a daemon or polling loop — waits for user input, acts, reports back
- User talks to it like any other Claude session

### Confirmation model

| Risk | Examples | Behavior |
|------|----------|----------|
| Read-only | Status check, pane map | No confirmation |
| Recoverable | Send `/fab-continue`, rebase | Announce before sending |
| Destructive | Merge PR, archive, delete worktree | Confirm before executing |

### Pre-send validation

Before sending keys to any pane, the operator MUST:
1. Verify target pane still exists
2. Check agent is idle via `fab runtime` — sending to a busy agent risks corrupting its work
3. If agent is not idle, warn user and ask for confirmation

### Guardrails (informed by agent-orchestrator patterns)

The operator needs bounded autonomy. Key guardrails:

- **Always re-derive state** — re-query `fab pane-map` / `fab status show --all` / `fab runtime` before every action. Never trust stale conversation memory.
- **Retry limits + escalation** — every automatic action has a bounded retry count. Stuck agent nudge: max 1. Rebase conflict: 0 retries (immediate escalation). When budget exhausted, flag to user.
- **Context discipline** — operator never loads change artifacts (intakes, specs, tasks). Reports concisely. Delegates diagnosis to the user.

### What the operator is not

- Not a daemon — it's a Claude session, not a background process
- Not a lifecycle enforcer — agents manage their own pipeline
- Not a scheduler — no queuing or priority management
- Not a replacement for batch scripts — those remain for non-interactive use

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the operator skill alongside existing execution skills
- `fab-workflow/kit-architecture`: (modify) Document `fab send-keys` subcommand
- `fab-workflow/distribution`: (modify) Note that the operator skill ships with the kit

## Impact

- **New skill file**: `fab/.kit/skills/fab-operator1.md` (source) → `.claude/skills/fab-operator1` (deployed)
- **New spec file**: `docs/specs/skills/SPEC-fab-operator1.md` (to be renamed from `SPEC-fab-conductor.md`)
- **Go binary**: New `send-keys` subcommand
- **Dependencies**: `fab pane-map` (change bh45) must land first — operator consumes its output
- **Existing skills**: No modifications — operator is purely additive
- **External deps**: tmux (required at runtime), `gh` (for PR merging use case)

## Open Questions

None — all four open questions resolved during discussion (see Assumptions #11–14).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Name is "operator1" (experiment 1 — more operators expected) | User renamed from "conductor" to "operator1" — this is the first in a series of operator experiments | S:95 R:95 A:95 D:95 |
| 2 | Certain | This is a skill (`/fab-operator1`), not a package | Discussed — requires AI agent for NL interpretation, not a shell script | S:90 R:85 A:95 D:95 |
| 3 | Certain | Operator is NOT a lifecycle enforcer | Discussed — agents self-govern; operator handles cross-agent coordination only | S:95 R:90 A:90 D:95 |
| 4 | Certain | Observation via `fab pane-map` + `fab status show --all` + `fab runtime` | Discussed — three primitives for unified observation | S:90 R:85 A:90 D:90 |
| 5 | Certain | Interaction via `fab send-keys <change> "<text>"` | Discussed — indirection through fab CLI for standard change resolution | S:90 R:80 A:90 D:90 |
| 6 | Certain | Pre-send idle check is mandatory | Discussed — sending to busy agent corrupts its work | S:90 R:70 A:90 D:95 |
| 7 | Certain | Not a daemon — user-driven Claude session | Discussed — waits for input, acts, reports back | S:95 R:85 A:90 D:95 |
| 8 | Confident | Confirmation model: read-only (none), recoverable (announce), destructive (confirm) | Discussed — three tiers. Exact boundary of "destructive" vs "recoverable" may shift | S:80 R:80 A:80 D:75 |
| 9 | Certain | Seven use cases: broadcast, sequenced rebase, merge PRs, spawn worktree, status dashboard, unstick agent, notification surface | Discussed — autopilot (sequential pipeline execution) deferred to separate change | S:90 R:85 A:85 D:90 |
| 10 | Confident | Depends on `fab pane-map` (change bh45) landing first | Discussed — operator consumes pane-map output. Could degrade gracefully without it but design assumes it | S:80 R:75 A:85 D:80 |
| 11 | Certain | No standing orders in v1 — session context suffices | Resolved — user agreed. Persistence deferred until real friction. Future home: `fab/project/operator1.yaml` | S:90 R:85 A:85 D:90 |
| 12 | Certain | User-driven, not event-driven — operator pane is single notification surface | Resolved — user agreed. Refreshes state before each action, holds "notify me" instructions in conversation context | S:90 R:80 A:85 D:90 |
| 13 | Certain | Single-repo only, by design | Resolved — user confirmed. Fab-kit's worktree model is same-repo checkouts | S:95 R:90 A:95 D:95 |
| 14 | Certain | `fab send-keys` targets fab-managed worktrees only | Resolved — user confirmed. Change resolution requires a change argument; raw `tmux send-keys` available for edge cases | S:90 R:85 A:90 D:90 |
| 15 | Certain | Always re-derive state before every action — never trust stale conversation memory | Informed by agent-orchestrator pattern: state derivation over caching | S:90 R:85 A:90 D:95 |
| 16 | Certain | Bounded retries + escalation — every auto action has a retry limit, then escalate to user | Informed by agent-orchestrator reaction system. Prevents infinite loops | S:90 R:75 A:85 D:90 |
| 17 | Certain | Context discipline — never load change artifacts, report concisely, delegate diagnosis | Operator context window is finite; keep it lean | S:90 R:85 A:90 D:90 |

17 assumptions (15 certain, 2 confident, 0 tentative, 0 unresolved).
