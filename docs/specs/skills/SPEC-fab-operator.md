# fab-operator
Standalone multi-agent coordination layer: runs in a dedicated tmux pane, observes all fab agents on its tmux server, routes commands via `tmux send-keys`, auto-answers routine agent questions, and drives autopilot queues. One operator per tmux server; coordinates across agents rather than advancing stages itself.
## Flow
```
Started via `fab operator`; runs a continuous /loop cycle
├─ Re-derive state each cycle: Bash: fab pane map --all-sessions (never trust cached values)
├─ Auto-answer routine agent questions; nudge stalled agents; route commands via tmux send-keys
└─ Drive autopilot queues (/fab-new → /fab-fff); spawn each task in a fresh worktree
```
### Tools used
Bash (`fab pane map`, `tmux send-keys`, `wt create`), Skill (`/loop`); helpers `_cli-agents`, `_cli-fab`, `_cli-external`.
### Sub-agents
None (spawns agent sessions, not sub-agents).
