# git-pr-review

## Summary

Processes PR review comments from any reviewer (human or Copilot). Fully autonomous — detects reviews, requests Copilot if none exist, triages comments, applies fixes, commits and pushes.

## Flow

```
/git-pr-review invoked (user or sub-agent)
│
├─ Step 0: Start Review-PR Stage
│  └─ Bash: fab status start <change> review-pr git-pr-review
│
├─ Step 1: Resolve PR
│  ├─ Bash: gh pr view --json number,url
│  └─ Bash: gh repo view --json nameWithOwner
│
├─ Step 2: Detect Reviews and Route
│  ├─ Phase 1: Check existing reviews
│  │  ├─ Bash: gh api .../pulls/{n}/reviews
│  │  └─ Bash: gh api .../pulls/{n}/comments
│  │     └─ [if comments exist] → Step 3 Path A
│  │
│  ├─ Phase 2: Request Copilot (fallback)
│  │  └─ Bash: gh api .../requested_reviewers -X POST
│  │     └─ [if fails] STOP
│  │
│  └─ Phase 3: Poll for Copilot review
│     └─ Bash: gh api .../reviews (poll every 30s, max 16x)
│        └─ [if review arrives] → Step 3 Path B
│        └─ [if timeout] STOP
│
├─ Step 3: Fetch Comments
│  ├─ Path A: Bash: gh api .../pulls/{n}/comments
│  └─ Path B: Bash: gh api .../reviews/{id}/comments
│
├─ Step 4: Triage Comments
│  ├─ Classify: actionable vs informational
│  ├─ Read: source files at {path}
│  └─ Edit: source files (targeted fixes)
│
├─ Step 5: Commit and Push
│  ├─ Bash: git add {files}
│  ├─ Bash: git commit -m "fix: address review feedback"
│  └─ Bash: git push
│
└─ Step 6: Update Review-PR Stage
   ├─ [pass] Bash: fab status finish <change> review-pr
   └─ [fail] Bash: fab status fail <change> review-pr

Phase tracking (via yq directly on .status.yaml):
  waiting → received → triaging → fixing → pushed
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Source files for applying fixes |
| Edit | Source files (targeted fixes from review comments) |
| Bash | gh API calls, git operations, fab status commands, yq phase tracking |

### Sub-agents

None.

### Direct .status.yaml writes (via yq, not fab CLI)

| Field | When |
|-------|------|
| `stage_metrics.review-pr.phase` | At each phase transition |
| `stage_metrics.review-pr.reviewer` | When reviews detected |
