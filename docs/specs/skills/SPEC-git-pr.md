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
│     ├─ Read: intake.md (PR title + Summary + Changes), spec.md, plan.md OR tasks.md, .status.yaml
│     ├─ Read: config.yaml (linear_workspace for issue links)
│     ├─ Bash: gh repo view --json (for blob URLs)
│     ├─ Compute true-impact (gated on {has_fab}):
│     │  ├─ Bash: git merge-base origin/main HEAD (with origin/master fallback)
│     │  └─ Bash: fab impact "$BASE" HEAD
│     │     (subcommand reads true_impact_exclude + test_paths from
│     │      config.yaml, emits YAML with added/deleted/net + optional
│     │      excluding + optional tests; impact rendering omitted when
│     │      fab impact fails, excluding is absent in the YAML, no fab
│     │      context, no merge-base, or the total pass yields +0/−0)
│     ├─ Assemble body: ## Meta (table + **Pipeline** + optional **Issues** + optional True-impact block),
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
| Bash | All git operations, gh CLI, fab status commands. Step 3c additionally runs `fab impact "$BASE" HEAD` once to compute the true-impact line-count passes rendered as the True-impact block in the body's `## Meta` block — the subcommand internally reads `true_impact_exclude` and `test_paths` and runs the raw, `excluding`, and `tests` `git diff --shortstat` passes. |

### Sub-agents

None.

### True-impact rendering

The PR body's `## Meta` block renders a true-impact breakdown derived from the `fab impact` YAML (the raw, `excluding`, and `tests` passes). The `total` shown is the **scaffolding-excluded** number (`excluding.*` when `true_impact_exclude` is non-empty, else the raw pass) — the raw-with-`fab/`/`docs/`-included count is never displayed in the PR body.

- **When the `tests` sub-block is present** (project has a non-empty `test_paths`): render a three-row block:
  ```
  True impact:
    impl:  +140 / −38  (net +102)
    tests: +400 / −0   (net +400)
    total: +540 / −38  (net +502)   ← excludes fab/, docs/
  ```
  - `total` = scaffolding-excluded pass (fallback raw when `true_impact_exclude` empty, in which case the `← excludes …` annotation is omitted).
  - `tests` = the `tests` pass (test lines within the excluded universe).
  - `impl` = the render-time residual, computed PER COMPONENT as `max(0, total.X − tests.X)` for each of `added`/`deleted`/`net` independently. It is NOT read from any stored field. A per-component clamp emits a one-line stderr warning (best-effort posture) and never renders a negative component.
  - The `← excludes …` annotation reflects the actual `true_impact_exclude` config values verbatim — never hardcoded. The Unicode minus `−` (U+2212) is used.
- **When the `tests` sub-block is absent** (empty/absent `test_paths`): fall back to the prior single inline `**Impact**: …code (excluding …) · …total` line.
- The rendering is omitted entirely when `fab impact` fails, the `total` pass yields `+0/−0`, there is no merge-base, or `{has_fab}` is false.
