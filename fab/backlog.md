## Open

- [ ] [ngaw] 2026-02-23: Quality gate - how to decide which PR has had deep thought vs just surface level?
- [ ] [v34t] 2026-02-23: A timeline or user journey mermaid diagram showing which commands are typed in the main repo vs the worktree
- [ ] [q0lw] 2026-03-11: If fab binary or wt or idea binary not found, stop. Add to preamble?
- [x] [41gc] 2026-04-02: Next step in removing .kit - remove dependency on .kit/scripts folder
- [x] [uqy8] 2026-04-02: Should fab sync command be made a part of the shim binary? So the user always does fab sync before working on a freshly cloned repo
- [ ] [ub2y] 2026-04-02: Make hooks work directly using the fab system command - remove fab/.kit folder dependency from everywhere
- [ ] [1old] 2026-04-04: fab init should check for git repo before downloading release — currently downloads first, creates config, then fails at sync's git check, leaving stale artifacts
- [x] [hin2] 2026-04-22: Operator /fab-operator: rule 4 (numbered menu → 1) is ambiguous. Distinguish routine numbered prompts (tool permission, trivial defaults) from strategic ones (scope/PR split/pipeline shape) — only auto-answer the former. Strategic prompts (multi-option choices that affect queue shape, commit content, or checklist compliance) should escalate even when a stated default exists.
- [x] [i1l6] 2026-04-22: Operator /fab-operator: add configurable auto-default-after-N-minutes for idle escalations. Keep the pipeline moving is highest priority — if an agent has been idle on an escalated prompt past a threshold (default 30m), operator auto-picks the agent's stated default and logs the action. Configurable per-change (or globally via .fab-operator.yaml). Trades user oversight for throughput; essential for long/headless autopilot runs. Pairs with rule-4 ambiguity fix (hin2).
