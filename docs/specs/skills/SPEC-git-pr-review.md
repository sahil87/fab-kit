# git-pr-review

## Summary

Processes PR review comments from any reviewer (human or Copilot). Fully autonomous — detects reviews, requests Copilot if none exist, triages comments with disposition intent (fix/defer/skip), applies fixes, commits, pushes, and posts reply comments confirming outcomes.

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
├─ Step 3: Fetch Comments (with id, node_id)
│  ├─ Path A: Bash: gh api .../pulls/{n}/comments
│  └─ Path B: Bash: gh api .../reviews/{id}/comments
│
├─ Step 4: Triage Comments
│  ├─ Classify intent: fix, defer, skip, or informational
│  ├─ Read: source files at {path}
│  └─ Edit: source files (targeted fixes for "fixed" comments)
│
├─ Step 5: Commit and Push
│  ├─ Bash: git add {files}
│  ├─ Bash: git commit -m "fix: address review feedback"
│  ├─ Bash: git push
│  └─ [no modifications] → proceed to Step 5.5 (don't stop)
│
├─ Step 5.5: Post Replies
│  ├─ Deduplicate: skip comments with existing disposition replies
│  ├─ Bash: gh api .../pulls/{n}/comments -f body=... -F in_reply_to=...
│  └─ Best-effort: failed POSTs logged, not fatal
│
└─ Step 6: Update Review-PR Stage
   ├─ [pass] Bash: fab status finish <change> review-pr
   └─ [fail] Bash: fab status fail <change> review-pr

Phase tracking (via yq directly on .status.yaml):
  waiting → received → triaging → fixing → pushed → replying
```

### Disposition taxonomy

Triage assigns **intent** (action verb); replies confirm **outcome** (past-tense).

| Intent (triage) | Reply (outcome) |
|-----------------|-----------------|
| `fix` | `Fixed — {description}. ({sha})` |
| `defer` | `Deferred — {reason}.` |
| `skip` | `Skipped — {reason}.` |

Informational comments receive no reply.

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Source files for applying fixes |
| Edit | Source files (targeted fixes from review comments) |
| Bash | gh API calls (REST only), git operations, fab status commands, yq phase tracking |

### Sub-agents

None.

### Direct .status.yaml writes (via yq, not fab CLI)

| Field | When |
|-------|------|
| `stage_metrics.review-pr.phase` | At each phase transition (including `replying`) |
| `stage_metrics.review-pr.reviewer` | When reviews detected |
