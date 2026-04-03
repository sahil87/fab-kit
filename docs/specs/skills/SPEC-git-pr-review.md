# git-pr-review

## Summary

Processes PR review comments from any reviewer (human or bot). Fully autonomous — detects reviews, requests automated reviews via a cascading tool chain (Copilot → Codex → Claude) when no existing reviews are found, triages comments with disposition intent (fix/defer/skip), applies fixes, commits, pushes, and posts reply comments confirming outcomes.

## Arguments

- **`--tool <name>`** *(optional)* — Forces a specific review tool, bypassing the cascade. Valid values: `copilot`, `codex`, `claude`.

## Configuration

The `review_tools` block in `fab/project/config.yaml` controls which tools are attempted in the cascade:

```yaml
review_tools:
  copilot: true    # try GitHub Copilot (remote)
  codex: true      # try OpenAI Codex CLI (local)
  claude: true     # try Claude CLI (local)
```

Setting a tool to `false` skips it. When the `review_tools` key is absent, all tools default to `true`.

## Flow

```
/git-pr-review [--tool <name>] invoked (user or sub-agent)
│
├─ Step 0: Start Review-PR Stage
│  └─ Bash: fab status start <change> review-pr git-pr-review
│
├─ Step 1: Resolve PR
│  ├─ Bash: gh pr view --json number,url
│  └─ Bash: gh repo view --json nameWithOwner
│
├─ Step 1.5: Parse --tool Flag
│  └─ Validate tool name (copilot, codex, claude) or STOP on invalid
│
├─ Step 2: Detect Reviews and Route
│  ├─ Phase 1: Check existing reviews
│  │  ├─ Bash: gh api .../pulls/{n}/reviews
│  │  └─ Bash: gh api .../pulls/{n}/comments
│  │     └─ [if comments exist] → Step 3
│  │
│  └─ Phase 2: Review Request Cascade (no reviews found)
│     ├─ Read config: review_tools from fab/project/config.yaml
│     ├─ If --tool flag: attempt only that tool
│     ├─ Tool 1 — Copilot (remote):
│     │  └─ Bash: gh pr edit {n} --add-reviewer copilot
│     │     └─ [success] "Copilot review requested" → STOP
│     │     └─ [fail] fall through
│     ├─ Tool 2 — Codex (local):
│     │  ├─ Bash: command -v codex
│     │  ├─ Construct enriched prompt (Step 2a)
│     │  └─ Bash: codex --quiet "<enriched_prompt>"
│     │     └─ [success] post as PR comment (Step 2b) → STOP
│     │     └─ [fail] fall through
│     └─ Tool 3 — Claude (local):
│        ├─ Bash: command -v claude
│        ├─ Construct enriched prompt (Step 2a)
│        └─ Bash: claude -p "<enriched_prompt>"
│           └─ [success] post as PR comment (Step 2b) → STOP
│           └─ [fail] "No review tools available" → STOP
│
├─ Step 2a: Context Enrichment (for local tools)
│  ├─ Bash: git diff main...HEAD (diff)
│  ├─ Bash: git diff --name-only main...HEAD (file list)
│  ├─ Bash: gh pr view --json body -q .body (PR description)
│  └─ Best-effort: test suite output
│
├─ Step 2b: Local Review Output Posting
│  └─ Bash: gh api .../issues/{n}/comments -f body="..." (best-effort)
│
├─ Step 3: Fetch Comments (with id, node_id)
│  └─ Bash: gh api .../pulls/{n}/comments
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

### Review Request Cascade

The cascade runs when Phase 1 finds no existing reviews with inline comments. It attempts review tools in a fixed order (Copilot → Codex → Claude), stopping on the first success:

| Tool | Type | Detection | On Success | On Failure |
|------|------|-----------|------------|------------|
| Copilot | Remote | Attempt `gh pr edit --add-reviewer copilot` | Print message, STOP (user re-invokes later) | Fall through |
| Codex | Local | `command -v codex` | Post as PR comment + print to terminal | Fall through |
| Claude | Local | `command -v claude` | Post as PR comment + print to terminal | Cascade exhausted |

The `--tool` flag bypasses the cascade and attempts only the specified tool. Config-disabled tools are skipped (unless forced via `--tool`).

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
| Bash | gh API calls (REST only), git operations, fab status commands, yq phase tracking, codex/claude CLI invocation |

### Sub-agents

None.

### Direct .status.yaml writes (via yq, not fab CLI)

| Field | When |
|-------|------|
| `stage_metrics.review-pr.phase` | At each phase transition (including `replying`) |
| `stage_metrics.review-pr.reviewer` | When reviews detected |
