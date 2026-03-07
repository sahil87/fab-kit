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
│     ├─ Read: intake.md (PR title), spec.md, tasks.md, .status.yaml
│     ├─ Bash: gh repo view --json (for blob URLs)
│     └─ Bash: gh pr create --title --body
│
├─ Step 4a: Record PR URL
│  └─ Bash: fab status add-pr <change> <url>
│
├─ Step 4b: Finish Ship Stage
│  └─ Bash: fab status finish <change> ship git-pr
│
├─ Step 4c: Commit Status Update
│  ├─ Bash: git add .status.yaml .history.jsonl
│  ├─ Bash: git commit
│  └─ Bash: git push
│
└─ Step 4d: Write PR Sentinel
   └─ Bash: echo "$PR_URL" > .pr-done
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Intake, spec, tasks, .status.yaml (for PR body generation) |
| Bash | All git operations, gh CLI, fab status commands |

### Sub-agents

None.
