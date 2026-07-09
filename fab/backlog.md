## Open

- [ ] [ngaw] 2026-02-23: Quality gate - how to decide which PR has had deep thought vs just surface level?
- [ ] [v34t] 2026-02-23: A timeline or user journey mermaid diagram showing which commands are typed in the main repo vs the worktree
- [ ] [q0lw] 2026-03-11: If fab binary or wt or idea binary not found, stop. Add to preamble?
- [ ] [ub2y] 2026-04-02: Make hooks work directly using the fab system command - remove fab/.kit folder dependency from everywhere
- [ ] [ioku] 2026-07-06: Divest agent active/idle state production: delete the .fab-runtime.yaml _agents pipeline (hooks/GC/PID-walker/flock), make fab pane send/map/capture read the @rk_agent_state tmux pane-option convention written by rk agent-setup. Full pickup detail: fab/plans/sahil/agent-state-divestment.md
- [ ] [xz4f] 2026-07-08: The current breakup of tiers isnt working. I need access to hydrate step also separately. Maybe it would be easier to name each stage directly again, along with default and operator tiers. Also - I need a migration where we reset back to default - i.e. remove the agent section completely, and replace it with the commented defaults (via a fab config ..... command). The tiers should be one of the sections that put up their defaults in fab/project/config.yaml. So in the migration we should remove it using the agent, and it should get added back via a "fab config ..." command mechanically
