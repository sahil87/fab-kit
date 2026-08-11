# git-pr-review
Processes PR review comments from any reviewer, fully autonomous: detects reviews, requests a Copilot review and polls up to 10 minutes when none exist, triages (fix/defer/skip), applies fixes, commits, pushes, and posts outcome replies.
## Flow
```
/git-pr-review [<change>] [--tool <name>]
├─ Start: Bash: fab change resolve → {name}; branch-matches-change guard → STOP on mismatch/detached; fab status start review-pr
├─ Resolve PR (gh pr view, gh repo view); validate --tool (copilot only) or STOP
├─ Detect: [comments exist] → triage / [none] → request Copilot review, poll gh pr view 30s×20 synchronously → [timeout] Step 6 timeout (stage stays active)
├─ Fetch: Bash: gh api --paginate pulls/{n}/comments (reply comments skipped)
├─ Triage fix/defer/skip/informational → Read + Edit fixes → commit + push ([commit fails] reset + STOP; [push fails] keep commit, no replies)
├─ Post disposition replies (dedup existing, best-effort POSTs); Step 6: fab status finish / fail / timeout-left-active
└─ Step 6.5 (success only, idempotent): commit + best-effort push .status.yaml/.history.jsonl; yq phase tracking on .status.yaml
```
### Tools used
| Tool | Purpose |
|------|---------|
| Read/Edit | Source files for review fixes |
| Bash | gh API (REST only), git (incl. Step 6.5 status commit), fab status, yq phase tracking |
### Sub-agents
None.
