# fab-operator1

## Summary

Multi-agent coordination layer that runs in a dedicated tmux pane. Observes all running fab agents via `fab pane-map`, and interacts with them via `fab resolve --pane` + raw `tmux send-keys`. Translates natural-language user instructions into cross-agent actions: broadcasting commands, sequencing rebases, spawning new worktrees, and merging PRs.

Not a lifecycle enforcer — individual agents already know their pipeline. The operator handles coordination that requires cross-agent awareness.

---

## Primitives

The operator relies on these primitives:

| Primitive | Source | Purpose |
|-----------|--------|---------|
| `fab pane-map` | fab CLI | Primary observation: pane-to-change mapping with stage and agent state (all panes) |
| `fab resolve --pane` | fab CLI | Resolve a change to its tmux pane ID for use with raw `tmux send-keys` |
| `wt list` + `fab change list` | fab CLI / wt CLI | Fallback observation when outside tmux |

### `fab resolve --pane` (change-to-pane primitive)

```
tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter
```

Resolves `<change>` to a tmux pane ID via standard change resolution (4-char ID, substring, full name), then the operator composes with raw `tmux send-keys`. This is the composable primitive — the operator constructs the full command.

**Safety**: `fab resolve --pane` errors if no matching pane exists. The operator SHOULD verify the pane exists and the agent is idle before sending.

---

## Discovery

On startup (and refreshable on demand), the operator builds a **pane map**: a mapping from tmux pane → worktree path → change name → agent state.

### Procedure

1. Run `fab pane-map` — returns the full pane → tab → worktree → change → stage → agent-state table in one call

`fab pane-map` handles all internal mechanics: session-scoped tmux pane discovery (`-s`), worktree detection, active change resolution (via `.fab-status.yaml` symlink), stage lookup, and agent idle state (via `.fab-runtime.yaml`). The operator consumes the output, not the underlying files.

### Pane Map Structure

```
Pane  Tab        Worktree                          Change                              Stage     Agent
%3    alpha      /repo.worktrees/alpha/            260306-r3m7-add-retry-logic         apply     active
%7    bravo      /repo.worktrees/bravo/            260306-k8ds-ship-wt-binary          review    idle (2m)
%12   main       /repo.worktrees/charlie/          260306-ab12-refactor-auth           hydrate   idle (8m)
```

The operator displays this map on startup and refreshes it before each action.

---

## Use Cases

### 1. Broadcast a command to all agents

User says: "run /fab-continue in all idle agents"

Operator:
1. Refreshes pane map via `fab pane-map`
2. Filters for agents where the Agent column shows idle
3. For each: `tmux send-keys -t "$(fab resolve <change> --pane)" "/fab-continue" Enter`

### 2. Sequenced rebase after completion

User says: "when r3m7 finishes, rebase k8ds on main"

Operator:
1. Polls `fab pane-map` to check the trigger change's stage
2. When r3m7 reaches hydrate (or a target stage), sends to k8ds:
   - `tmux send-keys -t "$(fab resolve k8ds --pane)" "git fetch origin main && git rebase origin/main" Enter`

### 3. Merge completed PRs

User says: "merge all PRs that are ready"

Operator:
1. Refreshes pane map, filters for changes at `ship` or `review-pr (pass)` stage
2. For each candidate: `fab status get-prs <change>` to retrieve PR URLs
3. Confirms before executing (destructive): "Will merge PRs for {change1}, {change2}. Proceed?"
4. On confirmation: runs `gh pr merge <url>` from the operator's own shell for each PR

### 4. Spawn a new worktree + agent for an idea

User says: "start working on the retry logic idea"

Operator:
1. Looks up the idea via `fab idea show "retry logic"`
2. Creates a worktree: `wt create --non-interactive --worktree-name <name>`
3. Opens a new tmux tab with a Claude session:
   `tmux new-window -n "fab-<id>" -c <worktree> "claude --dangerously-skip-permissions '/fab-new <description from backlog>'"`

This replaces the manual `batch-fab-switch-change` workflow with natural language.

### 5. Status dashboard

User says: "what's everyone doing?"

Operator:
1. Refreshes pane map via `fab pane-map` (includes tab, stage, and agent state)
2. Presents a human-readable summary

### 6. Unstick a stuck agent

User says: "r3m7 looks stuck, nudge it" — or notices via status that an agent has been idle at a non-terminal stage for too long.

Operator:
1. Refreshes pane map, confirms r3m7 is idle
2. Reads the agent's current stage to craft an appropriate nudge
3. Sends: `tmux send-keys -t "$(fab resolve r3m7 --pane)" "/fab-continue" Enter` (or a more targeted prompt if the user describes what's wrong)

The operator can also diagnose — "what's r3m7's last output?" could use `tmux capture-pane -t <pane> -p` to read recent terminal output and reason about what went wrong.

### 7. Notification surface

The operator pane serves as the **single notification surface** for cross-agent events. When the user asks "tell me when r3m7 finishes" or "notify me if anything fails," the operator holds the instruction in its conversation context and checks on the next user interaction (or when explicitly asked to check).

This is user-driven, not event-driven — the operator does not poll in the background. But it is the designated place where cross-agent status converges, making it the natural surface for the user to check.

### 8. Sequential pipeline execution (autopilot)

User says: "take all these changes to completion: bh45, qkov, ab12" — or "run all my changes, highest confidence first."

This is the operator's most complex use case. It operates as an autonomous pipeline operator, driving each change through the full lifecycle sequentially.

#### Monitoring via `/loop`

The operator itself stays reactive — it acts in response to input. Autopilot monitoring uses `/loop` to provide a periodic heartbeat:

1. After confirming the queue and starting the first change, invoke `/loop 2m "check autopilot progress"`
2. Each tick: re-derive state via `fab pane-map` → check current change → advance/flag/skip as needed
3. On queue completion: report final summary and stop the loop
4. On "pause": stop the loop; on "resume": restart it

The user can still interject between ticks — interrupt commands work immediately on the next prompt.

#### Per-change loop

```
For each change (in order):
  1. Spawn worktree         → wt create --non-interactive --reuse --worktree-name <change> <branch>
  2. Open tmux tab          → tmux new-window -n "fab-<id>" -c <worktree> \
                               "claude --dangerously-skip-permissions '/fab-switch <change>'"
  3. Check confidence       → fab status confidence <change>
     - confidence >= gate   → tmux send-keys -t "$(fab resolve <change> --pane)" "/fab-ff" Enter
     - confidence < gate    → flag to user, skip or run /fab-fff
  4. Monitor (on each tick) → fab pane-map
     - stage reaches hydrate/ship → change succeeded
     - review fails after rework budget → flag and skip
     - agent idle >15 min at non-terminal stage → nudge once, then flag
     - pane dies → flag and skip
  5. Merge                  → gh pr merge from operator's shell
  6. Rebase remaining       → tmux send-keys -t "$(fab resolve <next> --pane)" "git fetch origin main && git rebase origin/main" Enter
     - conflict → flag to user, skip to next (never auto-resolve)
  7. Cleanup                → wt delete <worktree-name> (optional, after merge)
  8. Progress               → report one-line status
```

#### Ordering strategies

- **User-provided order**: "run these in this order: bh45, qkov, ab12." Deterministic — the user controls dependencies and sequencing.
- **Confidence-based**: Sort by confidence score descending (query each change via `fab status confidence <change>`). Highest confidence changes are most likely to complete autonomously. Get the easy wins merged first, then tackle the ambiguous ones.
- **Hybrid**: User provides constraints ("bh45 before qkov"), operator optimizes the rest by confidence.

#### Failure handling

| Failure | Operator action |
|---------|-----------------|
| Confidence below gate | Flag to user: "qkov confidence is 2.1, below feat gate (3.0). Run /fab-fff (needs questions) or skip?" |
| Review fails (rework needed) | Let the agent's rework loop run (up to rework budget). If exhausted, flag to user and skip to next change. |
| Rebase conflict after merge | Detect conflict (check exit code), flag to user: "ab12 has rebase conflicts after merging bh45. Resolve manually or skip?" |
| Agent crashes / pane dies | Detect via stale pane check, report to user, skip to next change. |

#### Resumability

If interrupted (operator pane closed, session restarted), the operator reconstructs state from `fab pane-map` — each change's stage tells it where in the sequence it was. Changes already merged are at `review-pr (pass)` or archived. The operator can resume from the first non-completed change.

#### Lifecycle knowledge required

This use case requires the operator to understand:
- Confidence gates and thresholds per change type (from `docs/specs/change-types.md`)
- Stage progression: intake → spec → tasks → apply → review → hydrate → ship
- The difference between `/fab-ff` (gated, no frontloaded questions) and `/fab-fff` (ungated, frontloaded questions)
- Rework budget (max 3 cycles from `fab/project/code-review.md`)
- PR merge mechanics via `gh`

---

## Interaction Model

The operator is a long-running Claude session in its own tmux pane. It waits for user input, acts, and reports back. The user talks to it like any other Claude session. The one exception is autopilot mode, where `/loop` provides periodic heartbeat prompts (see UC8).

```
You (in operator pane):"rebase all agents on main"
Operator:                Refreshes pane map. 3 agents found.
                          Sending to %3 (r3m7): git fetch && git rebase origin/main
                          Sending to %7 (k8ds): git fetch && git rebase origin/main
                          Sending to %12 (ab12): git fetch && git rebase origin/main
                          Done. 3 agents rebased.
```

### Confirmation model

Actions are categorized by risk:

| Risk | Examples | Confirmation |
|------|----------|--------------|
| **Read-only** | Status check, pane map refresh | None |
| **Recoverable** | Send `/fab-continue`, rebase | Announce before sending, no confirmation needed |
| **Destructive** | Merge PR, archive change, delete worktree | Confirm before executing |

The user can override with "just do it" or similar to skip confirmation for destructive actions.

### Pre-send validation

Before sending keys to any pane, the operator MUST:

1. Verify the target pane still exists (refresh pane map)
2. Verify the agent is idle (via the Agent column in the pane map) — sending commands to a busy agent risks corrupting its work
3. If the agent is not idle, warn the user and ask for confirmation

---

## What the Operator Is Not

- **Not a daemon** — it's a Claude session, not a background process. It acts when you talk to it.
- **Not a lifecycle enforcer** — agents manage their own pipeline. The operator doesn't validate stage transitions.
- **Not a scheduler** — it doesn't queue work or manage priorities. It executes what you tell it to.
- **Not a replacement for batch scripts** — `batch-fab-switch-change` and `batch-fab-new-backlog` remain for scripted, non-interactive use. The operator is the conversational equivalent.

---

## Guardrails

Informed by patterns from [agent-orchestrator](https://github.com/sahil-weaver/agent-orchestrator) — a production orchestration layer for parallel AI agents. The guardrails below prevent the operator from going off-track, especially during autopilot mode.

### Always re-derive state

The operator MUST re-query live state (`fab pane-map`) before every action. Never rely on stale values from conversation memory. State derivation over caching — if a pane died or a change advanced since the last check, the operator must know.

### Retry limits + escalation

For autopilot mode, the operator tracks attempt counts per change:

| Situation | Max retries | Escalation |
|-----------|-------------|------------|
| Review fails (rework needed) | 3 (per `code-review.md` rework budget) | Flag to user: "r3m7 exhausted rework budget. Manual intervention needed." Skip to next change. |
| Agent idle too long (>15 min at non-terminal stage) | 1 nudge (`/fab-continue`) | If still idle after nudge, flag to user: "r3m7 appears stuck at apply stage." |
| Rebase conflict | 0 (never auto-resolve) | Immediately flag to user. |
| Agent pane dies | 1 respawn attempt | If respawn fails, flag and skip. |
| Pane death (non-autopilot) | 0 | Report: "Pane for {change} is gone." No respawn outside autopilot. |
| Send to busy agent | 0 | Warn user, require explicit confirmation before sending. |

The operator MUST NOT retry indefinitely. Every automatic action has a bounded retry count. When the budget is exhausted, escalate to the user and move on.

### Time-based escalation

In addition to retry counts, the operator SHOULD track wall-clock time per change during autopilot:

- **Stage timeout**: If a change has been at the same stage for >30 minutes (configurable), flag to user regardless of retry state. An agent that's "active" but stuck in a loop won't trigger retry limits, but will trigger the time gate.
- **Total timeout**: If a change has been in the autopilot pipeline for >2 hours total, flag to user for review.

These are advisory — the operator reports and asks, it doesn't kill agents autonomously.

### Never send to busy agents

The pre-send idle check (already specced in Pre-send validation) is a hard guardrail, not a soft suggestion. Sending commands to an active agent can corrupt its context mid-reasoning. The only exception is if the user explicitly confirms after a warning.

### Context discipline

The operator's own context window is finite. During long autopilot runs:

- **Report concisely**: Status updates should be brief ("bh45: merged. Rebasing qkov. 3 of 5 complete.")
- **Don't load change artifacts**: The operator never reads intakes, specs, or tasks — it only reads coordination state. This keeps its context lean.
- **Delegate diagnosis**: When an agent needs investigation ("what went wrong with r3m7?"), the operator reads terminal output via `tmux capture-pane` but does NOT try to deeply understand the agent's codebase work. It reports what it sees and lets the user decide.

### Interruptibility

During autopilot, the user MUST be able to interrupt at any time:

- "Stop after the current change" — operator finishes the active change, then halts
- "Skip qkov" — operator removes it from the queue and moves to the next
- "Pause" — operator stops sending new commands but doesn't kill running agents
- "Resume" — operator picks up from where it paused

The operator acknowledges interrupts immediately, even if an action is in progress.

---

## Implementation Considerations

### Launcher

Start the operator via `fab/.kit/scripts/fab-operator1.sh` — creates a singleton tmux tab named `operator` and invokes `/fab-operator1` in a new Claude session.

### Skill or package?

This is a **skill** (`/fab-operator1`), not a package. It requires an AI agent to interpret natural language and reason about cross-agent coordination. The supporting `fab resolve --pane` flag is the CLI primitive for change-to-pane resolution.

### Context loading

The operator loads the always-load layer (like `/fab-discuss`) but does NOT load change-specific artifacts. It doesn't need intakes, specs, or tasks — it only needs the coordination primitives (pane map, status).

### Stale pane detection

Tmux panes can disappear (user closes tab, agent crashes). The operator SHOULD detect stale entries in the pane map and prune them. `tmux has-session -t <pane>` returns non-zero for dead panes.

### Agent busy detection

Sending keys to a busy agent is dangerous — the text lands in the agent's input buffer and may be processed at the wrong time. The operator MUST check the Agent column in the pane map before sending. If the agent is busy, the operator can either wait or ask the user.

### Worktree-to-change resolution

A worktree may not have an active change yet (freshly created). The operator should handle this gracefully — showing the pane as "no change" rather than erroring.

---

## Relationship to Existing Components

| Component | Relationship |
|-----------|-------------|
| `batch-fab-switch-change` | Operator is the conversational equivalent. Batch script remains for non-interactive use. |
| `batch-fab-new-backlog` | Operator's "spawn new worktree" use case overlaps. Batch script remains for bulk operations. |
| `fab pane-map` | Primary observation mechanism. Operator consumes its output for pane, tab, change, stage, and agent state. |
| `wt list` + `fab change list` | Fallback observation when outside tmux (pane-map unavailable). |
| `wt-create`, `wt-delete` | Operator delegates worktree lifecycle to existing wt commands. |
| Assembly line (spec) | Operator is the natural evolution — same parallel model, but with a coordinator instead of manual tab-hopping. |

---

## Resolved Design Decisions

1. **No persistent standing orders in v1.** Conversation context within a session is sufficient — the operator holds instructions like "tell me when r3m7 finishes" in memory for the duration of the session. Persistence (surviving session restarts) deferred until real friction emerges. Natural future home: `fab/project/operator1.yaml`.

2. **User-driven, not event-driven.** The operator does not poll or watch files in the background. It refreshes state before each action and when the user asks. The one exception is autopilot mode, where `/loop` provides periodic heartbeat prompts — the operator itself stays reactive, but `/loop` injects the periodic input. The operator pane is the single notification surface — the user checks it for cross-agent status. The "unstick agent" and "notify me when done" use cases work within this model via conversation context.

3. **Single-repo only, by design.** Fab-kit's model is one repo with worktrees as checkouts of the same repo. Multi-repo coordination is a fundamentally different tool, out of scope.

4. **`fab resolve --pane` targets fab-managed worktrees only.** The flag resolves a `<change>` argument via standard change resolution — this is meaningless for plain shell tabs. For edge cases where the operator skill needs to interact with non-fab panes, it can use raw `tmux send-keys` with a pane ID from the pane map directly.
