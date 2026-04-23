# Operator

The operator (`/fab-operator`) is a long-running coordination layer that sits in its own tmux pane, observing and directing agents across other panes.

| Command | Purpose |
|---------|---------|
| `/fab-operator` | Multi-agent coordination — monitoring, auto-answering, autopilot queues, dependency-aware spawning |

## Version History

The current operator (v8) evolved through eight iterations:

| Version | Key addition |
|---------|-------------|
| v1 | Observe and interact with agents across tmux panes |
| v2 | Proactive monitoring after every action |
| v3 | Auto-nudge for agents waiting on user input |
| v4 | `/loop`-driven monitoring, auto-nudge answer model, playbook catalog |
| v5 | Use case registry (Linear inbox, PR freshness), branch fallback, autopilot queues |
| v6 | Clean rewrite — principles-driven inference, persistent state via `.fab-operator.yaml`, generic watches, framed status output |
| v7 | Dependency-aware agent spawning (cherry-pick chains), branch map persistence, bounded retries, pre-send validation tiers |
| v8 | Pipeline-first routing, unified tick status frame, stack-then-review autopilot (with ordered merge), `»<wt>` tab naming, mandatory auto-enroll, `/fab-proceed` integration |
| v9 | Spawn-in-worktree principle — operator pane reserved for coordination state; all pipeline work runs in freshly spawned agent tabs, never in the operator pane itself |
