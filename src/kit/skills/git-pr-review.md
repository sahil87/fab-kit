---
name: git-pr-review
description: "Process PR review comments — triage and fix feedback from any reviewer (human or bot). When no reviews exist, requests a Copilot review and waits up to 10 minutes for it to appear."
allowed-tools: Bash(git:*), Bash(gh:*), Bash(command:*)
---

# /git-pr-review [<change>] [--tool <name>]

Process GitHub PR review comments on the current branch's PR. Handles feedback from any reviewer — human or bot. When no reviews exist, requests an automated Copilot review and polls for up to 10 minutes for it to appear. Fully autonomous — no questions, no prompts.

**`<change>` argument** *(optional)*: an explicit change to target instead of the active one — resolved transiently in Step 0 (`.fab-status.yaml` untouched). Arguments are classified by value: `--tool` and the value following it are consumed as the flag; any remaining positional argument is the change reference (a `--tool` value can never be misread as a change).

**`--tool` flag**: Names the review tool Step 2 Phase 2 requests when no reviews exist, overriding the `code-review.md` § Review Tools check (a forced tool is attempted even when that section disables it). Valid values: `copilot` — currently the only wired tool, and also the default.

---

## Contents

- Behavior
- Rules
- Disposition Reference

---

## Behavior

### Step 0: Start Review-PR Stage

Resolve the change first (`fab change resolve` accepts a 4-char ID, folder substring, or full folder name — see `_cli-fab.md` § fab change):

- **Explicit `<change>` argument provided** → run `fab change resolve <change> 2>/dev/null` (transient override — `.fab-status.yaml` is untouched). On failure, STOP with `Cannot resolve change '<change>'.` — a named target that doesn't resolve is a caller error; do NOT fall back to the active change.
- **No `<change>` argument** → run `fab change resolve 2>/dev/null` (the active change). On failure, proceed with no change context — every `fab status` step below is skipped silently.

On success, capture the output as `{name}` — the change folder name, used wherever later steps reference `<change>` in `fab status` commands and in the Step 6.5 `fab/changes/{name}/…` file paths.

**Branch-matches-change guard** *(only when a change resolved; runs BEFORE the `fab status start` below — no status mutation on the STOP path)*: run `git branch --show-current`. If the output is **empty** (detached HEAD), STOP with `Cannot process PR reviews from a detached HEAD — check out the change's branch first (run /git-branch).` Otherwise the current branch MUST match `{name}`: exact string equality, or `{name}` appearing as a substring of the branch. On mismatch, STOP — do NOT check out another branch autonomously:

```
Branch '{current_branch}' does not match change '{name}'.
Run /git-branch to switch to the change's branch, /fab-switch to change the active change,
or pass the intended change explicitly: /git-pr-review <change>.
```

Then attempt to start the `review-pr` stage:

```bash
fab status start <change> review-pr git-pr-review 2>/dev/null || true
```

This is best-effort — failures are silently ignored. The `start` command handles both `pending` and `failed` → `active`. If the stage is already `active` or `done`, the call is a no-op (exits non-zero, silently ignored).

### Step 1: Resolve PR

1. Verify `gh` is available: `command -v gh`
   - If missing → print `gh CLI not found.` and go to Step 6 with outcome **failure**
2. Get current branch: `git branch --show-current`
3. Look up PR with `gh pr view --json number,url`, capturing its exit code and any stderr output.
   - If the command fails with a "no pull requests found" error → print `No PR found on this branch.` and go to Step 6 with outcome **failure**.
   - If the command fails for any other reason → print the `gh` error output and go to Step 6 with outcome **failure**.
4. If the command succeeds, capture `{number}` and `{url}` from the response.
5. Get owner/repo: `gh repo view --json nameWithOwner -q '.nameWithOwner'`

### Step 1.5: Parse `--tool` Flag

If the invocation includes `--tool <name>`:

1. Validate `<name>` is one of: `copilot` (case-insensitive, normalize to lowercase)
2. If invalid → print `Invalid tool: {name}. Valid values: copilot.` and STOP
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

If comments exist → proceed directly to Step 3 (skip Phase 2 — no Copilot review is requested when existing reviews with comments are found).

If reviews exist but no inline comments → print `Reviews exist but no actionable inline comments to process. Check the PR directly for reviewer feedback.` and go to Step 6 with outcome **no-reviews**. This prevents re-requesting automated reviews when a human reviewer left only a body-level comment (e.g., a summary in the review dialog).

If no reviews at all → proceed to Phase 2.

**Phase 2 — Copilot Review Request**:

Request an automated Copilot review and wait for it to appear.

**Forced tool override**: If `--tool copilot` was provided (Step 1.5), skip the config check below and proceed directly to the Copilot request.

**Configuration**: Read the `copilot` entry from `fab/project/code-review.md` § Review Tools. Only the `copilot` entry is honored here. An absent § Review Tools section — or an absent `copilot` entry — means Copilot is enabled; it is disabled only when the section lists `- copilot: false`.

If the `copilot` entry is `false` (and `--tool copilot` was **not** provided): print `No automated reviewer available. Run /git-pr-review when reviews are added.` and go to Step 6 with outcome **no-reviews** (clean finish).

> **Two distinct logins — do NOT conflate** (getting these backwards is the #1 cause of a poll never seeing a review that has in fact landed): you add the reviewer via `gh pr edit --add-reviewer copilot-pull-request-reviewer`, but the entry that then surfaces under the PR's **requested reviewers** has login `Copilot`; once a review actually lands, the object in the `reviews` array carries `author.login == "copilot-pull-request-reviewer"` (commonly `copilot-pull-request-reviewer[bot]`). (Apparent oddity, recorded as empirical reality: the value you `--add-reviewer` with matches the landed-review author login, while `requested_reviewers` shows `Copilot`.)
>
> The landed-review poll MUST therefore match `author.login == "copilot-pull-request-reviewer"` (the review-author login on the `reviews` array), **not** `Copilot` (the `requested_reviewers` login) — a predicate keyed on the request-side login never matches a landed review object and would time out even though the review arrived. This matches the established, deliberately-set behavior (`docs/memory/pipeline/execution-skills.md`: n30u documents `"Copilot"` in `requested_reviewers` vs `"copilot-pull-request-reviewer[bot]"` in `reviews`; u1m1 set the Phase 2 `.author.login` filter to `copilot-pull-request-reviewer` so incoming Copilot reviews are detected). Confirming the **request** itself succeeded is separate: GraphQL `reviewRequests` **omits bot/app reviewers** like Copilot, so a request confirmation MUST use REST `requested_reviewers` (`gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers`), never GraphQL.

> **Synchronous-poll discipline — the subagent MUST NOT yield mid-poll.** When `/git-pr-review` runs as a dispatched subagent (e.g. from `/fab-fff` Step 5), the Copilot poll below MUST run **synchronously to completion within this single invocation**: the subagent MUST NOT yield, return, or hand back control while the poll is pending — it stays in the loop until a review appears or all 20 attempts (30s × 20 / 10-minute window) are exhausted, then proceeds to Step 3 or the timeout exit. This is a permanent, non-negotiable directive (prior efforts stalled mid-poll and left `review-pr` stuck `active`). Copilot reviews land ~4.5–6.5 min after the request — inside the window — so patience-to-completion is correct, never an early return.

**Copilot request and poll**:

1. Attempt: `gh pr edit {number} --add-reviewer copilot-pull-request-reviewer` (the value `--add-reviewer` takes — correct here; note this is the same string as the landed-review author login, even though the resulting `requested_reviewers` entry surfaces as login `Copilot` — see the two-login note above).
2. **On success** (exit 0):
   - Print: `Copilot review requested. Waiting up to 10 minutes...`
   - *(Optional request confirmation — GraphQL omits bot reviewers, so use REST if confirming:* `gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers` *should now list the Copilot reviewer surfacing under login `Copilot`.)*
   - Poll every 30 seconds, up to 20 attempts (run this loop **synchronously to completion** — do NOT yield or return between attempts, per the discipline note above). The predicate matches the **review-author** login `copilot-pull-request-reviewer` (the login on the landed `reviews` object — NOT the `Copilot` login that surfaces under `requested_reviewers`):
     ```bash
     gh pr view {number} --json reviews -q '.reviews | map(select(.author.login == "copilot-pull-request-reviewer")) | length'
     ```
   - When Copilot review count > 0: proceed to Step 3 (Fetch Comments)
   - If 20 attempts exhausted without a Copilot-authored review: print `Copilot review requested but not yet available. Re-run /git-pr-review to process when ready.` — when an explicit `<change>` was passed in Step 0, include it in the suggested command (`Re-run /git-pr-review <change> …`; an argless re-run would resolve the active change instead) — and go to Step 6 with outcome **timeout** (no error, no fail event — the requested review is still pending)
3. **On failure** (non-zero exit from `gh pr edit`):
   - Print: `No automated reviewer available. Run /git-pr-review when reviews are added.`
   - Go to Step 6 with outcome **no-reviews** (clean finish — no error, no fail event)

### Step 3: Fetch Comments

Fetch all review comments on the PR:

```bash
gh api --paginate repos/{owner}/{repo}/pulls/{number}/comments --jq '.[] | {id: .id, path: .path, line: .line, body: .body, user: .user.login, in_reply_to_id: .in_reply_to_id}'
```

This captures comments from all submitted reviews regardless of reviewer. Track the set of unique `user` values for the commit message in Step 5. Skip reply comments (`in_reply_to_id` is non-null) — these are conversational follow-ups, not new review findings.

> Note: GitHub's REST API does not expose thread resolution state on individual
> comments. All non-reply comments are processed regardless of whether their
> thread has been marked resolved in the GitHub UI.

### Step 4: Triage Comments

For each fetched comment:

1. **Classify and assign disposition intent** in one pass (reply formats: see the Disposition Reference table — the single source):
   - **`fix`** — identifies a specific code issue with an implied or explicit fix that will be applied (e.g., "This variable is unused", "Missing null check", "Should use `const` instead of `let`")
   - **`defer`** — valid concern but out of scope for this PR (e.g., "This whole module needs better error handling")
   - **`skip`** — nitpick, stale reference, or not applicable (e.g., "I'd name this differently", references code already changed)
   - **informational** — summary, praise, general observation, or question without a clear fix action (e.g., "Looks good overall", "Why was this approach chosen?") — no disposition, no reply
2. **For `fix` comments**:
   - Read the file at `{path}`
   - Understand the issue described in `{body}`
   - If `{line}` is non-null, focus on that area of the file
   - If `{line}` is null, locate the issue from context in the body
   - Apply a targeted fix — do NOT make unrelated changes beyond what the comment addresses
   - Record a brief description of the change for the reply

Print: `{N} comments triaged: {F} fix, {D} defer, {S} skip, {I} informational (no reply)`

If all comments are informational → print `No actionable comments.` and go to Step 6 with outcome **success**.

### Step 5: Commit and Push

After all `fix` comments are processed:

1. Check for modifications: `git status --porcelain`
2. If no modifications → **unpushed-commit re-run gate**: before declaring "No changes needed", check for unpushed commits:

   ```bash
   git log --oneline @{u}..HEAD 2>/dev/null || echo "NO_UPSTREAM"
   ```

   Treat a missing upstream (`NO_UPSTREAM`) as unpushed. This catches a prior run whose commit succeeded but whose push failed — without it, the re-run would post "Fixed" replies citing a SHA that never reached the remote, permanently stranding the fix.
   - **Unpushed commits exist** → push them (`git push`, or `git push -u origin $(git branch --show-current)` when no upstream). On push failure, apply step 7's push-failure handling. On success, print `Pushed {N} previously unpushed commit(s).` and proceed to Step 5.5 — replies cite the now-pushed SHA.
   - **No unpushed commits** → print `No changes needed.` and proceed to Step 5.5 (do NOT stop here)
3. Stage only the specific modified files: `git add {file1} {file2} ...` (NOT `git add -A`)
4. Generate commit message based on reviewer source:
   - Comments from a single reviewer: `fix: address review feedback from @{username}`
   - Comments from multiple reviewers: `fix: address PR review feedback`
5. Commit: `git commit -m "<message>"`
6. Push: `git push`
7. The two failure modes differ — handle them separately:
   - **Commit fails** → run `git reset` to clear any staged changes, then print the error and STOP (no partial state — true for a failed commit)
   - **Push fails** → **KEEP the commit** (`git reset` cannot undo it, and discarding it would lose the fixes). Print the push error plus recovery guidance, then STOP **without posting replies** — no `Fixed` reply may cite an unpushed SHA:

     ```
     Push failed — the fix commit {sha} is kept locally.
     Recover with: git pull --rebase && git push, then re-run /git-pr-review.
     (The re-run detects the unpushed commit, pushes it, and posts the replies.)
     ```

     When an explicit `<change>` was passed in Step 0, include it in the recovery's re-run command (`… then re-run /git-pr-review <change>.`) — an argless re-run would resolve the active change instead.

Print: `Fixed {N} comment(s) across {M} file(s)`

### Step 5.5: Post Replies

After Step 5 (whether or not code was pushed), post reply comments for each comment that received a disposition. This step also runs when no code changes were made (all deferred/skipped) — the communication loop must close regardless.

**Deduplication**: Before posting replies, do a fresh fetch of all review comments to capture any existing disposition replies (Step 3 excludes replies, so those results cannot be used here):

```bash
gh api --paginate repos/{owner}/{repo}/pulls/{number}/comments --jq '.[] | select(.in_reply_to_id != null) | {id: .id, in_reply_to_id: .in_reply_to_id, body: .body}'
```

For each comment about to receive a reply, check if any fetched reply (where `in_reply_to_id` matches the target comment's `id`) starts with `Fixed —`, `Deferred —`, or `Skipped —`. If a disposition reply already exists, skip that comment.

**For each comment with a disposition** that passes deduplication:

1. Compose the reply text per the **Disposition Reference** table below (past-tense outcome confirmations). For `fix` replies, `{sha}` is the short (7-char) commit SHA from Step 5 and `{description}` is the brief change summary recorded during triage.

2. Post reply via REST API:
   ```bash
   gh api repos/{owner}/{repo}/pulls/{number}/comments \
     -f body="{reply_text}" \
     -F in_reply_to={comment_id}
   ```

**Error handling**: Reply posting is best-effort. If a reply POST fails for a specific comment, log the error and continue to the next comment. A failed reply does not cause the skill to abort or mark the stage as failed.

Print: `Replied to {N} comment(s): {F} fix, {D} defer, {S} skip`

### Step 6: Update Review-PR Stage

Step 6 is the exit point for every terminal path after Step 0: Steps 1, 2, and 4 route here with a named outcome. The two direct-STOP exceptions that never reach Step 6 are **Step 1.5** (invalid `--tool`) and **Step 5** (commit failure after `git reset`; push failure keeping the commit with recovery guidance and no replies).

If a change was resolved in Step 0 (active or explicit), act on the outcome class:

1. **On success** (comments processed and pushed, or no actionable comments): Call `fab status finish <change> review-pr git-pr-review 2>/dev/null || true`.
2. **On failure** (gh missing, no PR found, processing error): Call `fab status fail <change> review-pr git-pr-review 2>/dev/null || true`.
3. **On no-reviews** (no reviews found, no inline comments to process, or no automated reviewer available): Call `fab status finish <change> review-pr git-pr-review 2>/dev/null || true` — a successful no-op outcome.
4. **On timeout** (Copilot review requested but not yet available after 10 minutes): **leave the review-pr stage `active` — no finish, no fail.** The requested review is still pending; finishing here would mark the stage `done` with the review unprocessed, and `start` cannot reactivate a done stage. The earlier re-run message stands (naming the explicit `<change>` when one was passed in Step 0) — the re-run picks up the still-`active` stage.

All `fab status` calls are best-effort — failures silently ignored to avoid blocking the PR review workflow.

### Step 6.5: Commit Status Updates

If a change was resolved in Step 0 (active or explicit) **and** Step 6 took its success / no-reviews path (i.e., `fab status finish` ran — not the `fail` or `timeout` path), commit the bookkeeping writes that `fab status finish` produced (`.status.yaml` review-pr active→done, `completed_at`, `last_updated`; appended `review:passed` event in `.history.jsonl`). This mirrors `git-pr.md` Step 4c.

1. Stage the status and history files: `git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl`
2. Check for staged changes: `git diff --cached --quiet`
3. If staged changes exist: commit (`git commit -m "Update review-pr status"`) then push (`git push`).
4. If no staged changes (already committed / idempotent re-run): skip commit+push silently.

Print (if committed): `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)`

**Skip this step silently** when no change was resolved in Step 0, or when Step 6 took the `fail` or `timeout` path — neither path ran `finish`, and the fail path MUST NOT commit a half-finished state.

**Failure handling** (best-effort push): If the commit fails, report the error. If `git push` fails (e.g., a transient network error), report the error but do **not** STOP the skill or mark the stage as failed — a completed review cycle must not be aborted by a transient push failure. The local commit is retained and a later re-run / push reconciles it. (This softens git-pr's fail-fast push specifically for the terminal stage, consistent with git-pr-review's best-effort status-write ethos.)

### Phase Sub-State Tracking

When a change is resolved (active or explicit), update `stage_metrics.review-pr.phase` at key points during the workflow. Phase values track the skill's progress through its steps:

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
- Fail fast — if any step fails, report the error and stop immediately, except where a step explicitly declares best-effort handling (it reports but does not abort)
- Targeted fixes only — do not modify code beyond what each comment addresses
- Idempotent — re-running after fixes finds no new modifications and exits cleanly; re-running after a failed push detects the kept commit via the Step 5 unpushed-commit gate and pushes it before replying; re-running after replies skips already-replied comments; re-running after the status commit finds nothing staged (`git diff --cached --quiet`) and is a silent no-op

---

## Disposition Reference

Triage assigns an **intent** (action verb); replies confirm the **outcome** (past-tense).

| Intent (triage) | Description | Reply (outcome) |
|-----------------|-------------|-----------------|
| `fix` | Code will be changed to address the comment | `Fixed — {description}. ({sha})` |
| `defer` | Valid concern, out of scope for this PR | `Deferred — {reason}.` |
| `skip` | Nitpick, stale, or not applicable | `Skipped — {reason}.` |

Informational comments (praise, summaries, questions without code implications) receive no reply.
