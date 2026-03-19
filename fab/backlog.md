## Open

- [ ] [ngaw] 2026-02-23: Quality gate - how to decide which PR has had deep thought vs just surface level?
- [ ] [v34t] 2026-02-23: A timeline or user journey mermaid diagram showing which commands are typed in the main repo vs the worktree
- [ ] [lc3m] 2026-03-01: Switch fab/.kit/LICENSE from PolyForm Internal Use to MIT. Reasoning: fab-kit is a process (prompts, markdown, shell scripts) — not protectable IP in the copyright sense. The US Copyright Office ruled prompts convey unprotectable ideas. PolyForm Internal Use blocks community contributions and ecosystem growth while providing only illusory control. MIT gives attribution (copyright notice preserved), maximum adoption, zero friction, and lets the community keep the tool evolving — the real moat is speed and opinionation, not licensing restrictions.
- [ ] [q0lw] 2026-03-11: If fab binary or wt or idea binary not found, stop. Add to preamble?
- [ ] [t13m] 2026-03-18: Configurable agent spawn command in config.yaml — centralize the agent binary/flags so all spawn scripts and operator read from one place. Default: `claude --dangerously-skip-permissions --effort max -n "$(basename "$(pwd)")"`. Currently hardcoded in 4+ locations.
- [ ] [9tqo] 2026-03-18: The idea section on _cli-fab.md is factually incorrect. Move it to _cli_external - the whole Backlog section, and correct it. idea is a standalone binary that is shipped with fab-kit
- [ ] [02eh] 2026-03-18: When operator5 works in branch mode for matching a change, it asks for confirmation. Don't do so, just proceed
- [ ] [tm9h] 2026-03-19: When starting to creat PRs - create drafts. fab-kit should always create draft PRs. The devs generally need to check the implementation before marking it ok for code review
- [ ] [heht] 2026-03-19: Ensure randomness in worktree names is actually getting following - inrease the name universe even further
