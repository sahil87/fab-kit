# git-pr
Autonomously commits, pushes, and creates a draft GitHub PR — no prompts. Resolves PR type from status/intake/diff, generates the body from fab artifacts, records the PR URL in `.status.yaml`.
## Flow
```
/git-pr [<change>] [<type>]
├─ Resolve: Bash: fab change resolve → {has_fab}, {name} (explicit-arg failure STOPs); branch-matches-change guard → STOP on mismatch/detached
├─ Bash: fab status start {name} ship git-pr (if {has_fab}); PR type: status.yaml → intake keywords → git diff fallback
├─ Gather: branch/status/log, gh pr view → {pr_state}, default branch, fab status get-issues
├─ Guards: detached HEAD / on default branch / {pr_state} MERGED → STOP
├─ 3a Commit: expected-area guard for untracked files → git add -u + in-area untracked → commit
├─ 3a-bis (if {has_fab} + committed): Bash: fab memory-index → commit docs/memory drift (no --amend)
├─ 3b Push; 3c Create PR (no OPEN PR): Read intake → Bash: fab pr-meta → ## Meta block → gh pr create --draft (--fill fallback)
├─ 3d Retrofit ## Meta onto existing OPEN PR (idempotent prepend)
└─ 4a–4c: fab status add-pr + finish ship stage; commit + push .status.yaml/.history.jsonl
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read | Intake (PR title, Summary, Changes) |
| Bash | git, gh, fab status; `fab pr-meta` renders the entire `## Meta` block (self-contained; non-zero/empty → Meta omitted) |
### Sub-agents
None.
