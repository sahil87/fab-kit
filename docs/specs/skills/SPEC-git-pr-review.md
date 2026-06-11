# git-pr-review

## Summary

Processes PR review comments from any reviewer (human or bot). Fully autonomous — detects reviews, requests an automated Copilot review and polls up to 10 minutes for it to appear when no existing reviews are found, triages comments with disposition intent (fix/defer/skip), applies fixes, commits, pushes, and posts reply comments confirming outcomes.

## Arguments

- **`--tool <name>`** *(optional)* — Forces a specific review tool. Valid values: `copilot` only.

## Configuration

The `review_tools` block in `fab/project/config.yaml` controls whether Copilot is attempted:

```yaml
review_tools:
  copilot: true    # try GitHub Copilot (remote) — default when key is absent
```

Setting `copilot` to `false` skips Phase 2 entirely. When the `review_tools` key is absent, Copilot defaults to enabled.

## Flow

```
/git-pr-review [--tool <name>] invoked (user or sub-agent)
│
├─ Step 0: Start Review-PR Stage
│  ├─ Bash: fab change resolve 2>/dev/null → {name} (change folder
│  │        name — instantiates <change> below and the Step 6.5 paths)
│  └─ Bash: fab status start <change> review-pr git-pr-review
│
├─ Step 1: Resolve PR
│  ├─ Bash: gh pr view --json number,url
│  └─ Bash: gh repo view --json nameWithOwner
│
├─ Step 1.5: Parse --tool Flag
│  └─ Validate tool name (copilot only) or STOP on invalid
│
├─ Step 2: Detect Reviews and Route
│  ├─ Phase 1: Check existing reviews
│  │  ├─ Bash: gh api .../pulls/{n}/reviews
│  │  └─ Bash: gh api .../pulls/{n}/comments
│  │     └─ [if comments exist] → Step 3
│  │
│  └─ Phase 2: Copilot Review Request (no reviews found)
│     ├─ Read config: review_tools.copilot from fab/project/config.yaml
│     ├─ [copilot: false] "No automated reviewer available" → Step 6, outcome no-reviews (clean finish)
│     ├─ Bash: gh pr edit {n} --add-reviewer copilot-pull-request-reviewer
│     │  ├─ [success] Print "Copilot review requested. Waiting up to 10 minutes..."
│     │  │  └─ Poll: gh pr view --json reviews every 30s, up to 20 attempts
│     │  │     ├─ [review appears] → Step 3
│     │  │     └─ [20 attempts, no review] "...not yet available. Re-run /git-pr-review..." → Step 6, outcome timeout (stage left active — no finish, no fail)
│     │  └─ [failure] "No automated reviewer available..." → Step 6, outcome no-reviews (clean finish)
│
├─ Step 3: Fetch Comments (with id, node_id)
│  └─ Bash: gh api .../pulls/{n}/comments
│
├─ Step 4: Triage Comments (single classify-and-assign list —
│  │        260611-szxd f098; the Disposition Reference table is the
│  │        single reply-format source)
│  ├─ Classify + assign intent in one pass: fix, defer, skip, or informational
│  ├─ Read: source files at {path}
│  └─ Edit: source files (targeted fixes for "fix" comments)
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
├─ Step 6: Update Review-PR Stage (exit point for every terminal
│  │        path after Step 0 — Steps 1/2/4 route here with a
│  │        named outcome. Two direct-STOP exceptions: Step 1.5
│  │        invalid --tool; Step 5 commit/push failure after
│  │        git reset)
│  ├─ [success / no-reviews] Bash: fab status finish <change> review-pr
│  ├─ [failure] Bash: fab status fail <change> review-pr
│  └─ [timeout] stage left active — no finish, no fail
│               (re-run picks up the still-active stage)
│
└─ Step 6.5: Commit Status Updates (mirrors git-pr Step 4c)
   ├─ [gate] active change resolved AND Step 6 success/no-reviews path (skip on no-change / fail / timeout path)
   ├─ Bash: git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl
   ├─ Bash: git diff --cached --quiet  (idempotency guard — re-run is a silent no-op)
   ├─ [staged changes] Bash: git commit -m "Update review-pr status" && git push
   └─ [push fails] report error, do NOT STOP (best-effort push; local commit retained)

Phase tracking (via yq directly on .status.yaml):
  waiting → received → triaging → fixing → pushed → replying
```

### Copilot Review Request (Phase 2)

Phase 2 runs when Phase 1 finds no existing reviews with inline comments. It requests a Copilot review and polls for up to 10 minutes:

| Tool | Type | Detection | On Success | On Failure |
|------|------|-----------|------------|------------|
| Copilot | Remote | Attempt `gh pr edit --add-reviewer copilot-pull-request-reviewer` | Poll 30s/attempt up to 20× — proceed to Step 3 when review appears; on timeout: Step 6 `timeout` outcome (stage stays active — no finish, no fail) | Clean finish (no-reviews): "No automated reviewer available..." |

The `--tool copilot` flag forces the Copilot path regardless of config — the config check is skipped entirely when this flag is present. Without the flag, if `review_tools.copilot: false`, Phase 2 exits cleanly without attempting the request.

### Disposition taxonomy

Triage assigns **intent** (action verb); replies confirm **outcome** (past-tense).

| Intent (triage) | Reply (outcome) |
|-----------------|-----------------|
| `fix` | `Fixed — {description}. ({sha})` |
| `defer` | `Deferred — {reason}.` |
| `skip` | `Skipped — {reason}.` |

Informational comments receive no reply.

### Status-commit bookkeeping (Step 6.5)

Step 6's `fab status finish` writes the terminal `review-pr` stage to `done` (plus `completed_at`, `last_updated`) in `.status.yaml` and appends a `review:passed` event to `.history.jsonl`. Step 6.5 commits those writes so the terminal stage leaves a clean worktree, mirroring `git-pr.md` Step 4c (which commits its own ship bookkeeping).

| Aspect | Behavior |
|--------|----------|
| Gate | Runs only when an active change was resolved (Step 0) AND Step 6 took the success / no-reviews path. Skipped silently on the Step 6 `fail` path and when no active change resolved — the fail path must not commit a half-finished state. |
| Staged files | `fab/changes/{name}/.status.yaml`, `fab/changes/{name}/.history.jsonl` |
| Idempotency | `git diff --cached --quiet` guard — a re-run finds nothing staged and is a silent no-op (no commit, no push, no output) |
| Commit | `git commit -m "Update review-pr status"` when staged changes exist |
| Push | `git push` — **best-effort**: a transient push failure is reported but does NOT STOP the skill or fail the stage (unlike git-pr's fail-fast push). The local commit is retained and reconciled on a later run. |
| Output | `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)` (only when a commit was made) |

The best-effort push softens git-pr's fail-fast parity for the terminal stage, consistent with git-pr-review's best-effort status-write and reply ethos: a completed review cycle must not be aborted by a transient push failure.

### Tools used

| Tool | Purpose |
|------|---------|
| Read | Source files for applying fixes |
| Edit | Source files (targeted fixes from review comments) |
| Bash | gh API calls (REST only), git operations (including the Step 6.5 status commit + push), fab status commands, yq phase tracking |

### Sub-agents

None.

### Direct .status.yaml writes (via yq, not fab CLI)

| Field | When |
|-------|------|
| `stage_metrics.review-pr.phase` | At each phase transition (including `replying`) |
| `stage_metrics.review-pr.reviewer` | When reviews detected |
