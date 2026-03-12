# fab-operator2

## Summary

Multi-agent coordination layer with proactive monitoring. Runs in a dedicated tmux pane, observes all running fab agents via `fab pane-map`, and interacts with them via `fab resolve --pane` + raw `tmux send-keys`. Translates natural-language user instructions into cross-agent actions: broadcasting commands, sequencing rebases, spawning new worktrees, and merging PRs.

**Key difference from operator1**: After every action that dispatches work to another agent, operator2 automatically enrolls the target in a monitoring loop. Instead of fire-and-forget (send and go idle), operator2 uses `/loop` to periodically check agent progress, detect state transitions, and report them to the user.

Not a lifecycle enforcer — individual agents already know their pipeline. The operator handles coordination that requires cross-agent awareness.

---

## Primitives

The operator relies on these primitives:

| Primitive | Source | Purpose |
|-----------|--------|---------|
| `fab pane-map` | fab CLI | Primary observation: pane-to-change mapping with stage and agent state (all panes) |
| `fab resolve --pane` | fab CLI | Resolve a change to its tmux pane ID for use with raw `tmux send-keys` |
| `wt list` + `fab change list` | fab CLI / wt CLI | Fallback observation when outside tmux |
| `/loop` | fab skill | Periodic heartbeat for monitoring ticks |

### `fab resolve --pane` (change-to-pane primitive)

```
tmux send-keys -t "$(fab resolve <change> --pane)" "<text>" Enter
```

Resolves `<change>` to a tmux pane ID via standard change resolution (4-char ID, substring, full name), then the operator composes with raw `tmux send-keys`. This is the composable primitive — the operator constructs the full command.

**Safety**: `fab resolve --pane` errors if no matching pane exists. The operator SHOULD verify the pane exists and the agent is idle before sending.

---

## Discovery

On startup (and refreshable on demand), the operator builds a **pane map**: a mapping from tmux pane to worktree path to change name to agent state.

### Procedure

1. Run `fab pane-map` — returns the full pane, tab, worktree, change, stage, and agent-state table in one call

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

## Monitoring Behavior

### Monitored Set

The operator maintains a monitored set in conversation context (not a persistent file). Each entry tracks: change ID, pane ID, last-known stage, last-known agent state, enrolled-at timestamp, and last-transition-at timestamp.

### Enrollment Triggers

- Sending a command to an agent via `resolve --pane` + `tmux send-keys`
- User explicitly requests monitoring ("tell me when X finishes")
- Automatic action dispatched toward an agent (sequenced rebase)

Read-only actions (status check, pane map refresh) do NOT trigger enrollment.

### Tick Processing

On each `/loop` tick, the operator:

1. Re-queries `fab pane-map`
2. Compares current state against last-known state for every monitored change
3. Reports transitions:

| Transition | Example Report |
|------------|----------------|
| Stage advance | "r3m7: spec -> tasks" |
| Pipeline completion | "r3m7: reached hydrate -- pipeline complete" |
| Review failure | "r3m7: review failed, reworking (back at apply)" |
| Pane death | "r3m7: pane %3 is gone -- agent exited" |
| Stuck detection | "r3m7: idle at apply for 15m -- may be stuck" |

### Removal Triggers

A change is removed from the monitored set when:

1. It reaches a terminal stage (`hydrate`, `ship`, `review-pr`)
2. Its pane dies (no longer in pane map)
3. The user explicitly stops monitoring it

### Loop Lifecycle

- Started when the first change is enrolled
- Stopped when the monitored set becomes empty
- Only one loop active at a time
- Default interval: 5m (configurable via user instruction)

### Stuck Detection

Agents idle at a non-terminal stage for longer than the stuck threshold (default 15m) are flagged. This is **advisory only** — the operator reports but does NOT auto-nudge. Nudging requires explicit UC6 invocation or user instruction.

---

## Use Cases

### 1. Broadcast a command to all agents

User says: "run /fab-continue in all idle agents"

Operator:
1. Refreshes pane map via `fab pane-map`
2. Filters for agents where the Agent column shows idle
3. For each: `tmux send-keys -t "$(fab resolve <change> --pane)" "/fab-continue" Enter`
4. **Enrolls all recipients in monitoring**

### 2. Sequenced rebase after completion

User says: "when r3m7 finishes, rebase k8ds on main"

Operator:
1. Enrolls r3m7 in monitoring (if not already)
2. Monitoring loop detects when r3m7 reaches hydrate or later
3. Sends to k8ds: `tmux send-keys -t "$(fab resolve k8ds --pane)" "git fetch origin main && git rebase origin/main" Enter`
4. Reports: "r3m7 finished. Sending rebase to k8ds."
5. Enrolls k8ds in monitoring

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

### 5. Status dashboard

User says: "what's everyone doing?"

Operator:
1. Refreshes pane map via `fab pane-map` (includes tab, stage, and agent state)
2. Presents a human-readable summary
3. If monitoring is active, includes the monitored set with enrolled-at timestamps

### 6. Unstick a stuck agent

User says: "r3m7 looks stuck, nudge it"

Operator:
1. Refreshes pane map, confirms r3m7 is idle
2. Reads the agent's current stage to craft an appropriate nudge
3. Sends: `tmux send-keys -t "$(fab resolve r3m7 --pane)" "/fab-continue" Enter`
4. **Enrolls the nudged agent in monitoring** to check recovery
5. If a second nudge is requested for the same agent, warns: "Already nudged {change} once. Manual investigation recommended." Sends only if user explicitly insists.

The operator can also diagnose — "what's r3m7's last output?" uses `tmux capture-pane -t <pane> -p` to read recent terminal output.

### 7. Notification surface

The operator pane serves as the **single notification surface** for cross-agent events. When the user says "tell me when r3m7 finishes," operator2 immediately enrolls the change in monitoring. The `/loop` handles notification automatically — no "check on next user interaction" caveat.

When the monitoring tick detects the change at a terminal stage: "r3m7 finished — now at {stage}."

### 8. Sequential pipeline execution (autopilot)

User says: "take all these changes to completion: bh45, qkov, ab12"

This is the operator's most complex use case. It operates as an autonomous pipeline operator, driving each change through the full lifecycle sequentially.

#### Monitoring via `/loop`

Autopilot uses its own `/loop` cadence (default 2m):

1. After confirming the queue and starting the first change, invoke `/loop 2m "check autopilot progress"`
2. Each tick: re-derive state via `fab pane-map` -> check current change -> advance/flag/skip as needed
3. On queue completion: report final summary and stop the loop
4. On "pause": stop the loop; on "resume": restart it

The user can still interject between ticks.

#### Per-change loop

```
For each change (in order):
  1. Spawn worktree         -> wt create --non-interactive --reuse --worktree-name <change> <branch>
  2. Open tmux tab          -> tmux new-window -n "fab-<id>" -c <worktree> \
                               "claude --dangerously-skip-permissions '/fab-switch <change>'"
  3. Check confidence       -> fab status confidence <change>
     - confidence >= gate   -> tmux send-keys -t "$(fab resolve <change> --pane)" "/fab-ff" Enter
     - confidence < gate    -> flag to user, skip or run /fab-fff
  4. Monitor (on each tick) -> fab pane-map
     - stage reaches hydrate/ship -> change succeeded
     - review fails after rework budget -> flag and skip
     - agent idle >15 min at non-terminal stage -> nudge once, then flag
     - pane dies -> flag and skip
  5. Merge                  -> gh pr merge from operator's shell
  6. Rebase remaining       -> tmux send-keys -t "$(fab resolve <next> --pane)" "git fetch origin main && git rebase origin/main" Enter
     - conflict -> flag to user, skip to next (never auto-resolve)
  7. Cleanup                -> wt delete <worktree-name> (optional, after merge)
  8. Progress               -> report one-line status
```

#### Ordering strategies

- **User-provided order**: Deterministic — the user controls dependencies and sequencing.
- **Confidence-based**: Sort by confidence score descending. Highest confidence changes merge first.
- **Hybrid**: User provides constraints, operator optimizes the rest by confidence.

#### Failure handling

| Failure | Operator action |
|---------|-----------------|
| Confidence below gate | Flag to user: "qkov confidence is 2.1, below feat gate (3.0). Run /fab-fff or skip?" |
| Review fails (rework needed) | Let the agent's rework loop run (up to rework budget). If exhausted, flag and skip. |
| Rebase conflict after merge | Detect conflict, flag to user, skip to next change. |
| Agent crashes / pane dies | Detect via pane check, 1 respawn attempt, then flag and skip. |

#### Resumability

If interrupted (operator pane closed, session restarted), the operator reconstructs state from `fab pane-map`. Merged changes appear at `review-pr (pass)` or archived. The operator resumes from the first non-completed change.

---

## Interaction Model

The operator is a long-running Claude session in its own tmux pane. It waits for user input, acts, and reports back. Between user interactions, the `/loop` provides periodic monitoring ticks — the operator processes each tick, reports transitions, and returns to waiting.

```
You (in operator pane):"send /fab-continue to all idle agents"
Operator:                Refreshes pane map. 3 idle agents found.
                          Sending to %3 (r3m7): /fab-continue
                          Sending to %7 (k8ds): /fab-continue
                          Sending to %12 (ab12): /fab-continue
                          Done. 3 agents received /fab-continue.
                          Monitoring r3m7, k8ds, ab12. Loop started (5m).

[5 minutes later, /loop tick]
Operator:                r3m7: apply -> review
                          k8ds: still at apply (active)
                          ab12: idle at spec for 6m

[10 minutes later, /loop tick]
Operator:                r3m7: reached hydrate -- pipeline complete
                          k8ds: apply -> review
                          ab12: idle at spec for 11m
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

- **Not a daemon** — it's a Claude session, not a background process. `/loop` provides periodic input, but the operator itself is reactive.
- **Not a lifecycle enforcer** — agents manage their own pipeline. The operator doesn't validate stage transitions.
- **Not a scheduler** — it doesn't queue work or manage priorities. It executes what you tell it to.
- **Not a replacement for batch scripts** — `batch-fab-switch-change` and `batch-fab-new-backlog` remain for scripted, non-interactive use. The operator is the conversational equivalent.

---

## Guardrails

Informed by patterns from [agent-orchestrator](https://github.com/sahil-weaver/agent-orchestrator) — a production orchestration layer for parallel AI agents. The guardrails below prevent the operator from going off-track, especially during monitoring and autopilot mode.

### Always re-derive state

The operator MUST re-query live state (`fab pane-map`) before every action and on every monitoring tick. Never rely on stale values from conversation memory. State derivation over caching — if a pane died or a change advanced since the last check, the operator must know.

### Retry limits + escalation

The operator tracks attempt counts per change:

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

- **Stage timeout**: If a change has been at the same stage for >30 minutes (configurable), flag to user regardless of retry state.
- **Total timeout**: If a change has been in the autopilot pipeline for >2 hours total, flag to user for review.

These are advisory — the operator reports and asks, it doesn't kill agents autonomously.

### Never send to busy agents

The pre-send idle check is a hard guardrail, not a soft suggestion. Sending commands to an active agent can corrupt its context mid-reasoning. The only exception is if the user explicitly confirms after a warning.

### Advisory-only stuck detection

Stuck detection in the monitoring loop is **advisory only**. The operator reports stuck agents to the user but NEVER auto-nudges. Sending `/fab-continue` to an agent that might be blocked on a legitimate long operation violates the pre-send validation contract. Nudging requires explicit UC6 invocation or user instruction.

### Context discipline

The operator's own context window is finite. During long monitoring and autopilot runs:

- **Report concisely**: Status updates should be brief ("bh45: merged. Rebasing qkov. 3 of 5 complete.")
- **Don't load change artifacts**: The operator never reads intakes, specs, or tasks — it only reads coordination state. This keeps its context lean.
- **Delegate diagnosis**: When an agent needs investigation, the operator reads terminal output via `tmux capture-pane` but does NOT try to deeply understand the agent's codebase work.

### Interruptibility

During autopilot, the user MUST be able to interrupt at any time:

- "Stop after the current change" — operator finishes the active change, then halts
- "Skip qkov" — operator removes it from the queue and moves to the next
- "Pause" — operator stops sending new commands but doesn't kill running agents
- "Resume" — operator picks up from where it paused

The operator acknowledges interrupts immediately, even if an action is in progress.

---

## Configuration

| Setting | Default | Override |
|---------|---------|----------|
| Monitoring interval | 5m | "check every {N}m" or "monitor every {N}m" |
| Stuck threshold | 15m | "flag agents stuck for more than {N} minutes" |
| Autopilot tick interval | 2m | "autopilot check every {N}m" |

All settings are session-scoped and reset when the operator session restarts.

---

## Implementation Considerations

### Launcher

Start the operator via `fab/.kit/scripts/fab-operator2.sh` — creates a singleton tmux tab named `operator` and invokes `/fab-operator2` in a new Claude session.

### Skill or package?

This is a **skill** (`/fab-operator2`), not a package. It requires an AI agent to interpret natural language and reason about cross-agent coordination. The supporting `fab resolve --pane` flag is the CLI primitive for change-to-pane resolution.

### Context loading

The operator loads the always-load layer (like `/fab-discuss`) but does NOT load change-specific artifacts. It doesn't need intakes, specs, or tasks — it only needs the coordination primitives (pane map, status).

### Stale pane detection

Tmux panes can disappear (user closes tab, agent crashes). The operator SHOULD detect stale entries in the pane map and prune them. The monitoring loop naturally handles this — pane deaths are detected on each tick.

### Agent busy detection

Sending keys to a busy agent is dangerous — the text lands in the agent's input buffer and may be processed at the wrong time. The operator MUST check the Agent column in the pane map before sending. If the agent is busy, the operator can either wait or ask the user.

### Worktree-to-change resolution

A worktree may not have an active change yet (freshly created). The operator should handle this gracefully — showing the pane as "no change" rather than erroring.

---

## Relationship to Existing Components

| Component | Relationship |
|-----------|-------------|
| `fab-operator1` | Predecessor — same capabilities, but fire-and-forget. Operator2 adds proactive monitoring. |
| `batch-fab-switch-change` | Operator is the conversational equivalent. Batch script remains for non-interactive use. |
| `batch-fab-new-backlog` | Operator's "spawn new worktree" use case overlaps. Batch script remains for bulk operations. |
| `fab pane-map` | Primary observation mechanism. Operator consumes its output for pane, tab, change, stage, and agent state. |
| `wt list` + `fab change list` | Fallback observation when outside tmux (pane-map unavailable). |
| `/loop` | Provides periodic heartbeat for monitoring ticks. |
| `wt-create`, `wt-delete` | Operator delegates worktree lifecycle to existing wt commands. |
| Assembly line (spec) | Operator is the natural evolution — same parallel model, but with a coordinator and proactive monitoring. |

---

## Resolved Design Decisions

1. **Monitoring via conversation-held state, not persistent file.** The monitored set is held in conversation context, not in a YAML file. This matches operator1's pattern for standing orders (UC7) and avoids introducing a new persistent state file. If the operator session ends, state is lost, but `fab pane-map` allows full reconstruction on restart.

2. **Advisory stuck detection, no auto-nudge.** Stuck detection reports to the user but does not automatically send `/fab-continue`. Auto-nudging could corrupt an agent that's blocked on a legitimate long operation — this violates the pre-send validation contract.

3. **Shared `operator` tab name.** Both operator1 and operator2 launchers use tab name `operator` (not `operator1`/`operator2`). Since only one runs at a time, this prevents stale tabs and accidental concurrent operators.

4. **Single `/loop` for all monitoring.** One loop covers all monitored changes. Adding a new change to the monitored set does not create a second loop — it extends the existing one.

5. **User-driven, not event-driven (with `/loop` bridge).** The operator does not poll or watch files in the background. `/loop` provides periodic input that the operator processes reactively. Between ticks, the operator waits for user input. This preserves the Claude session model while enabling proactive monitoring.
