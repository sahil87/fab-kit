---
name: git-pr-review
description: "Process PR review comments — triage and fix feedback from any reviewer (human or bot)."
allowed-tools: Bash(git:*), Bash(gh:*), Bash(command:*), Bash(codex:*), Bash(claude:*)
---

# /git-pr-review [--tool <name>]

Process GitHub PR review comments on the current branch's PR. Handles feedback from any reviewer — human or bot. When no reviews exist, requests an automated review via a cascading tool chain (Copilot → Codex → Claude). Fully autonomous — no questions, no prompts.

**`--tool` flag**: Forces a specific review tool, bypassing the cascade. Valid values: `copilot`, `codex`, `claude`. When provided, only that tool is attempted — no fallthrough.

---

## Behavior

### Step 0: Start Review-PR Stage

If an active change resolves (`fab change resolve 2>/dev/null`), attempt to start the `review-pr` stage:

```bash
fab status start <change> review-pr git-pr-review 2>/dev/null || true
```

This is best-effort — failures are silently ignored. The `start` command handles both `pending` and `failed` → `active`. If the stage is already `active` or `done`, the call is a no-op (exits non-zero, silently ignored).

### Step 1: Resolve PR

1. Verify `gh` is available: `command -v gh`
   - If missing → print `gh CLI not found.` and STOP
2. Get current branch: `git branch --show-current`
3. Look up PR with `gh pr view --json number,url`, capturing its exit code and any stderr output.
   - If the command fails with a "no pull requests found" error → print `No PR found on this branch.` and STOP.
   - If the command fails for any other reason → print the `gh` error output and STOP.
4. If the command succeeds, capture `{number}` and `{url}` from the response.
5. Get owner/repo: `gh repo view --json nameWithOwner -q '.nameWithOwner'`
6. Get the PR base branch: `gh pr view --json baseRefName -q '.baseRefName'` — capture as `{base_branch}` for use in context enrichment (Step 2a)

### Step 1.5: Parse `--tool` Flag

If the invocation includes `--tool <name>`:

1. Validate `<name>` is one of: `copilot`, `codex`, `claude` (case-insensitive, normalize to lowercase)
2. If invalid → print `Invalid tool: {name}. Valid values: copilot, codex, claude.` and STOP
3. Store the forced tool name for use in Step 2 Phase 2

### Step 2: Detect Reviews and Route

Check for existing reviews with comments, then route accordingly.

**Phase 1 — Check for existing reviews with comments**:

Fetch all reviews on the PR:

```bash
gh api repos/{owner}/{repo}/pulls/{number}/reviews --jq '[.[] | select(.state != "PENDING")] | length'
```

If the count is > 0, check for actual inline comments:

```bash
gh api repos/{owner}/{repo}/pulls/{number}/comments --jq 'length'
```

If comments exist → proceed directly to Step 3 (skip Phase 2 — the cascade does not run when existing reviews with comments are found).

If reviews exist but no inline comments → print `Reviews exist but no actionable inline comments to process. Check the PR directly for reviewer feedback.` and STOP. This prevents re-requesting automated reviews when a human reviewer left only a body-level comment (e.g., a summary in the review dialog).

If no reviews at all → proceed to Phase 2.

**Phase 2 — Review Request Cascade**:

Request an automated review via a cascading tool chain. Each tool is attempted in order; the cascade stops on the first success.

**Configuration**: Read `review_tools` from `fab/project/config.yaml`. Each tool may be set to `true` (try) or `false` (skip). When the `review_tools` key is absent, all tools default to `true`.

```yaml
review_tools:
  copilot: true
  codex: true
  claude: true
```

**Forced tool override**: If `--tool <name>` was provided (Step 1.5), skip the cascade and attempt only the specified tool, regardless of config settings.

**Cascade order**: Copilot → Codex → Claude

For each tool in order (or just the forced tool):

1. **Check config**: If the tool is set to `false` in `review_tools`, skip it
2. **Detect availability**: Check if the tool can run (see per-tool details below)
3. **Run the tool**: Execute the review request
4. **On success**: Print result and STOP (cascade complete)
5. **On failure/unavailability**: Fall through to the next tool

**Tool 1 — Copilot (remote)**:

- Attempt: `gh pr edit {number} --add-reviewer copilot`
- Detection: by attempting the command and checking exit code
- On success: print `Copilot review requested — re-run /git-pr-review later to process comments.` and STOP
- On failure (non-zero exit): fall through to Codex

**Tool 2 — Codex (local)**:

- Detection: `command -v codex`
- If not found: fall through to Claude
- If found: run the built-in Codex review command:
  ```bash
  codex review --base {base_branch}
  ```
  Where `{base_branch}` is the PR base branch resolved in Step 1.
- On success: post output as PR review (see Step 2b), print output to terminal, and STOP
- On failure: fall through to Claude

**Tool 3 — Claude (local)**:

- Detection: `command -v claude`
- If not found: cascade exhausted — print `No review tools available. Install codex or claude, or enable Copilot on this repository.` and STOP
- If found: construct the enriched review prompt (see Step 2a) and run:
  ```bash
  claude -p "Review this PR diff for bugs, edge cases, and improvements: <enriched_prompt>"
  ```
  Where `{base_branch}` is the PR base branch resolved in Step 1.
- On success: post output as PR review (see Step 2b), print output to terminal, and STOP
- On failure: cascade exhausted — print `No review tools available.` and STOP

**All tools disabled**: If all tools are disabled in config (all set to `false`) and no `--tool` flag was provided, print `All review tools are disabled in config. Enable at least one tool in review_tools.` and STOP.

**All tools unavailable**: If all enabled tools fail detection or execution, print `No review tools available. Install codex or claude, or enable Copilot on this repository.` and STOP.

### Step 2a: Context Enrichment

When running a local tool (Codex or Claude), construct an enriched review prompt containing these labeled sections:

1. **Diff**: `git diff {base_branch}...HEAD` (where `{base_branch}` is the PR base branch resolved in Step 1)
2. **Changed files**: `git diff --name-only {base_branch}...HEAD`
3. **PR description**: `gh pr view --json body -q .body` (omit section if empty or command fails)
4. **Test results**: Run the project's test suite and capture output (best-effort — omit section if no test command is configured or tests fail to run)

Format the prompt as:

```
Review this PR diff for bugs, edge cases, and improvements.

## Changed Files
{file_list}

## PR Description
{pr_body}

## Diff
{diff}

## Test Results
{test_output}
```

Missing sections are omitted — the prompt remains valid with whatever context is available.

### Step 2b: Local Review Output Posting

When a local tool (Codex or Claude) produces review output:

1. Attempt to post the output as a PR review (not an issue comment — this ensures Phase 1 detects it on re-run):
   ```bash
   gh api repos/{owner}/{repo}/pulls/{number}/reviews -f body="{review_output}" -f event="COMMENT"
   ```
2. If posting succeeds: print `Review posted on PR.`
3. If posting fails: log the error and continue — the output is still printed to the terminal
4. Always print the review output to the terminal regardless of posting outcome

### Step 3: Fetch Comments

Fetch all review comments on the PR:

```bash
gh api --paginate repos/{owner}/{repo}/pulls/{number}/comments --jq '.[] | {id: .id, node_id: .node_id, path: .path, line: .line, body: .body, user: .user.login, in_reply_to_id: .in_reply_to_id}'
```

This captures comments from all submitted reviews regardless of reviewer. Track the set of unique `user` values for the commit message in Step 5. Skip reply comments (`in_reply_to_id` is non-null) — these are conversational follow-ups, not new review findings.

> Note: GitHub's REST API does not expose thread resolution state on individual
> comments. All non-reply comments are processed regardless of whether their
> thread has been marked resolved in the GitHub UI.

### Step 4: Triage Comments

For each fetched comment:

1. **Classify**: Determine if the comment is **fixable** (identifies a specific code issue with an implied or explicit fix — e.g., "This variable is unused", "Missing null check", "Should use `const` instead of `let`"), **deferrable** (valid concern but out of scope for this PR — e.g., "This whole module needs better error handling"), **skippable** (nitpick, stale reference, or not applicable — e.g., "I'd name this differently", references code already changed), or **informational** (summary, praise, general observation, question without a clear fix action — e.g., "Looks good overall", "Why was this approach chosen?")
2. **Skip** informational comments — no disposition, no reply
3. **Assign disposition intent** to each non-informational comment:
   - **`fix`** — the comment identifies a specific code issue and a fix will be applied
   - **`defer`** — the comment raises a valid concern but it's out of scope for this PR
   - **`skip`** — the comment is a nitpick, stale, or not applicable
4. **For `fix` comments**:
   - Read the file at `{path}`
   - Understand the issue described in `{body}`
   - If `{line}` is non-null, focus on that area of the file
   - If `{line}` is null, locate the issue from context in the body
   - Apply a targeted fix — do NOT make unrelated changes beyond what the comment addresses
   - Record a brief description of the change for the reply

Print: `{N} comments triaged: {F} fix, {D} defer, {S} skip, {I} informational (no reply)`

If all comments are informational → print `No actionable comments.` and STOP.

### Step 5: Commit and Push

After all `fix` comments are processed:

1. Check for modifications: `git status --porcelain`
2. If no modifications → print `No changes needed.` and proceed to Step 5.5 (do NOT stop here)
3. Stage only the specific modified files: `git add {file1} {file2} ...` (NOT `git add -A`)
4. Generate commit message based on reviewer source:
   - Comments from a single reviewer: `fix: address review feedback from @{username}`
   - Comments from multiple reviewers: `fix: address PR review feedback`
5. Commit: `git commit -m "<message>"`
6. Push: `git push`
7. If commit or push fails → run `git reset` to clear any staged changes, then print the error and STOP (no partial state)

Print: `Fixed {N} comment(s) across {M} file(s)`

### Step 5.5: Post Replies

After Step 5 (whether or not code was pushed), post reply comments for each comment that received a disposition. This step also runs when no code changes were made (all deferred/skipped) — the communication loop must close regardless.

**Deduplication**: Before posting replies, do a fresh fetch of all review comments to capture any existing disposition replies (Step 3 excludes replies, so those results cannot be used here):

```bash
gh api --paginate repos/{owner}/{repo}/pulls/{number}/comments --jq '.[] | select(.in_reply_to_id != null) | {id: .id, in_reply_to_id: .in_reply_to_id, body: .body}'
```

For each comment about to receive a reply, check if any fetched reply (where `in_reply_to_id` matches the target comment's `id`) starts with `Fixed —`, `Deferred —`, or `Skipped —`. If a disposition reply already exists, skip that comment.

**For each comment with a disposition** that passes deduplication:

1. Compose reply text based on disposition intent (replies use past-tense to confirm the outcome):
   - `fix` → `Fixed — {description}. ({sha})` where `{sha}` is the short (7-char) commit SHA and `{description}` is the brief change summary recorded during triage
   - `defer` → `Deferred — {reason}.`
   - `skip` → `Skipped — {reason}.`

2. Post reply via REST API:
   ```bash
   gh api repos/{owner}/{repo}/pulls/{number}/comments \
     -f body="{reply_text}" \
     -F in_reply_to={comment_id}
   ```

**Error handling**: Reply posting is best-effort. If a reply POST fails for a specific comment, log the error and continue to the next comment. A failed reply does not cause the skill to abort or mark the stage as failed.

Print: `Replied to {N} comment(s): {F} fix, {D} defer, {S} skip`

### Step 6: Update Review-PR Stage

If an active change was resolved in Step 0:

1. **On success** (comments processed and pushed, or no actionable comments): Call `fab status finish <change> review-pr git-pr-review 2>/dev/null || true`.
2. **On failure** (no PR found, processing error): Call `fab status fail <change> review-pr git-pr-review 2>/dev/null || true`.
3. **On no reviews** (no reviews found): Call `fab status finish <change> review-pr git-pr-review 2>/dev/null || true` — a successful no-op outcome.

All statusman calls are best-effort — failures silently ignored to avoid blocking the PR review workflow.

### Phase Sub-State Tracking

When an active change is resolved, update `stage_metrics.review-pr.phase` at key points during the workflow. Phase values track the skill's progress through its steps:

| Phase | When set |
|-------|----------|
| `received` | Reviews detected (Step 2, Phase 1 hit) |
| `triaging` | Before classifying comments (Step 4 start) |
| `fixing` | Before applying fixes (Step 4, `fix` comments found) |
| `pushed` | After commit and push (Step 5 success) |
| `replying` | Before posting reply comments (Step 5.5 start) |

Phase updates are written via `yq -i ".stage_metrics.\"review-pr\".phase = \"<phase>\"" <status_file>`. Best-effort — failures silently ignored.

The `reviewer` field is set when reviews are detected: `yq -i ".stage_metrics.\"review-pr\".reviewer = \"<login>\"" <status_file>`. Value is `@{username}` (first reviewer found).

---

## Rules

- Fully autonomous — never ask questions, never present options
- Fail fast — if any step fails, report the error and stop immediately
- No partial commits — if the commit or push fails, no changes are left staged
- Targeted fixes only — do not modify code beyond what each comment addresses
- Idempotent — re-running after fixes finds no new modifications and exits cleanly; re-running after replies skips already-replied comments
- Best-effort replies — failed reply POSTs do not abort the skill or mark the stage as failed

---

## Disposition Reference

Triage assigns an **intent** (action verb); replies confirm the **outcome** (past-tense).

| Intent (triage) | Description | Reply (outcome) |
|-----------------|-------------|-----------------|
| `fix` | Code will be changed to address the comment | `Fixed — {description}. ({sha})` |
| `defer` | Valid concern, out of scope for this PR | `Deferred — {reason}.` |
| `skip` | Nitpick, stale, or not applicable | `Skipped — {reason}.` |

Informational comments (praise, summaries, questions without code implications) receive no reply.
