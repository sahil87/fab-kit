# git-pr

## Summary

Autonomously commits, pushes, and creates a GitHub PR. No prompts, no questions. Resolves PR type from status/intake/diff. Generates PR body from fab artifacts when available. Records PR URL in `.status.yaml`.

## Flow

```
/git-pr invoked (user or sub-agent)
│
├─ Step 0a: Start Ship Stage
│  └─ Bash: fab status start <change> ship git-pr
│
├─ Step 0b: Resolve PR Type
│  ├─ Read: fab/changes/{name}/.status.yaml (change_type)
│  ├─ Read: fab/changes/{name}/intake.md (keyword match)
│  └─ Bash: git diff --name-only (fallback)
│
├─ Step 1: Gather State
│  ├─ Bash: git branch --show-current
│  ├─ Bash: git status --porcelain
│  ├─ Bash: git log --oneline -5
│  ├─ Bash: git log --oneline @{u}..HEAD
│  ├─ Bash: gh pr view --json
│  └─ Bash: fab status get-issues <change>
│
├─ Step 1b: Branch Mismatch Nudge
│  └─ Bash: fab change resolve
│
├─ Step 2: Branch Guard (STOP if main/master)
│
├─ Step 3: Execute Pipeline
│  ├─ 3a. Commit (if uncommitted)
│  │  ├─ Bash: git add -A
│  │  └─ Bash: git commit -m "<message>"
│  ├─ 3b. Push (if unpushed)
│  │  └─ Bash: git push [-u origin <branch>]
│  └─ 3c. Create PR (if no PR exists)
│     ├─ Read: intake.md (PR title + Summary + Changes), plan.md OR tasks.md, .status.yaml
│     ├─ Read: config.yaml (linear_workspace for issue links)
│     ├─ Bash: gh repo view --json (for blob URLs)
│     ├─ Compute true-impact (gated on {has_fab}):
│     │  ├─ Bash: git merge-base origin/main HEAD (with origin/master fallback)
│     │  └─ Bash: fab impact "$BASE" HEAD
│     │     (subcommand reads true_impact_exclude + test_paths from config.yaml,
│     │      emits YAML with added/deleted/net + optional excluding + optional tests;
│     │      total = excluding (else raw); tests = tests sub-block;
│     │      impl = max(0, total − tests) derived at render time, per component;
│     │      three-row impl/tests/total Impact form when tests present, single
│     │      total line otherwise; ← excludes annotation from actual
│     │      true_impact_exclude config values, never hardcoded;
│     │      Impact line omitted when fab impact fails, no fab context,
│     │      no merge-base, or total yields +0/−0)
│     ├─ Assemble body: ## Meta (table + **Pipeline** + optional **Issues** + optional **Impact**),
│     │                 ## Summary (from intake ## Why), ## Changes (from intake ## What Changes)
│     │                 (Meta block omitted entirely when {has_fab} is false)
│     └─ Bash: gh pr create --draft --title --body
│
├─ Step 4a: Record PR URL
│  └─ Bash: fab status add-pr <change> <url>
│
├─ Step 4b: Finish Ship Stage
│  └─ Bash: fab status finish <change> ship git-pr
│
└─ Step 4c: Commit Status Update
   ├─ Bash: git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl
   ├─ Bash: git commit
   └─ Bash: git push
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Intake, spec, plan, .status.yaml, config.yaml (for PR body generation including Change section) |
| Bash | All git operations, gh CLI, fab status commands. Step 3c additionally runs `fab impact "$BASE" HEAD` once to compute the true-impact line counts rendered as the `**Impact**` line(s) in the body's `## Meta` block — the subcommand internally reads `true_impact_exclude` and `test_paths` and runs up to three `git diff --shortstat` passes (raw, excluding, tests). When a `tests` sub-block is present the Impact renders as a three-row impl / tests / total breakdown, where `impl = max(0, total − tests)` is the render-time residual (per component, never stored) and the `← excludes …` annotation reflects the actual `true_impact_exclude` config values; otherwise it collapses to the single `total` line. |

### Sub-agents

None.
