# Operator

The operator (`/fab-operator`) is a long-running coordination layer that sits in its own tmux pane, observing and directing agents across other panes.

| Command | Purpose |
|---------|---------|
| `/fab-operator` | Multi-agent coordination — monitoring, auto-answering, tracked work queues, dependency-aware spawning |

## Version History

The current operator (v12) evolved through twelve iterations:

| Version | Key addition |
|---------|-------------|
| v1 | Observe and interact with agents across tmux panes |
| v2 | Proactive monitoring after every action |
| v3 | Auto-nudge for agents waiting on user input |
| v4 | `/loop`-driven monitoring, auto-nudge answer model, playbook catalog |
| v5 | Use case registry (Linear inbox, PR freshness), branch fallback, autopilot queues |
| v6 | Clean rewrite — principles-driven inference, persistent operator state on disk (a repo-rooted `.fab-operator.yaml` at the time; the state file is now server-keyed at `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`), generic watches, framed status output |
| v7 | Dependency-aware agent spawning (cherry-pick chains), branch map persistence, bounded retries, pre-send validation tiers |
| v8 | Pipeline-first routing, unified tick status frame, stack-then-review autopilot (with ordered merge), `»<wt>` tab naming, mandatory auto-enroll, `/fab-proceed` integration |
| v9 | Spawn-in-worktree principle — operator pane reserved for coordination state; all pipeline work runs in freshly spawned agent tabs, never in the operator pane itself |
| v10 | rk-mandatory skill (one startup gate; 27 fallback passages deleted), CLI reference split by command family (`_cli-fab-operator` / `_cli-fab-pane`), live cadence from `rk cron list --json` in the ready line and compact frame |
| v11 | Generic tracked items — one `tracked:` list with a probe loop and derived schedule replaces the monitored/watches/autopilot/notes sections, `track` verb family, one-table frame (2026-09-11, change 260911-4a8m) |
| v12 | fab owns the operator-tick entry end-to-end — the clock reconcile seeds it when the server has none, and `fab operator clock sync` heals and reads it for Init and the per-tick frame (2026-09-12, change 260912-bjrk) |
