# git-pr-review

## Contents

- Summary
- Arguments
- Configuration
- Flow

## Summary

Processes PR review comments from any reviewer (human or bot). Fully autonomous — detects reviews, requests an automated Copilot review and polls up to 10 minutes for it to appear when no existing reviews are found, triages comments with disposition intent (fix/defer/skip), applies fixes, commits, pushes, and posts reply comments confirming outcomes.

**Prose optimization** (260620-skop): skill content trimmed to remove re-explanation of partial-owned concepts (the `fab change resolve` ID/substring/name forms) and to collapse the triplicated two-login and synchronous-poll restatements to a single statement each (the `copilot-pull-request-reviewer`-on-`reviews` predicate, the REST-not-GraphQL confirmation rule, the MUST-NOT-yield directive, and the n30u/u1m1 memory citation are all preserved), and a `## Contents` TOC added; no behavioral change (Flow / Tools / Sub-agents unchanged).

## Arguments

- **`<change>`** *(optional, 260612-w7dp)* — explicit change to target instead of the active one (any non-flag argument). Resolved transiently in Step 0 (`.fab-status.yaml` untouched); an explicit argument that fails to resolve STOPs (caller error), while argless failure proceeds with no change context. `/fab-fff` Step 5 passes the change folder name through (`/git-pr-review {name}` — folder names never collide with git-pr's type tokens, so both dispatches use the same form).
- **`--tool <name>`** *(optional)* — Names the review tool Step 2 Phase 2 requests, overriding the `code-review.md` § Review Tools check (a forced tool is attempted even when that section disables it). Valid values: `copilot` — currently the only wired tool, and also the default.

## Configuration

The `copilot` entry in `fab/project/code-review.md` § Review Tools controls whether Copilot is attempted. An absent section — or an absent `copilot` entry — means Copilot is enabled; it is disabled only when the section lists it as `false`:

```markdown
## Review Tools

- copilot: false    # skip the GitHub Copilot (remote) review request
```

Setting `copilot` to `false` skips Phase 2 entirely. When the § Review Tools section (or the `copilot` entry) is absent, Copilot defaults to enabled.

## Flow

```
/git-pr-review [<change>] [--tool <name>] invoked (user or sub-agent)
│
├─ Step 0: Start Review-PR Stage
│  ├─ Bash: fab change resolve [<change>] 2>/dev/null → {name} (change folder
│  │        name — instantiates <change> below and the Step 6.5 paths;
│  │        explicit-arg resolution failure → STOP, 260612-w7dp)
│  ├─ Branch-matches-change guard (260612-w7dp, when a change resolved;
│  │  runs BEFORE the status start — no mutation on the STOP path):
│  │  branch must equal {name} or contain it as a substring → mismatch
│  │  STOPs (guidance: /git-branch, /fab-switch, or /git-pr-review <change>);
│  │  empty branch (detached HEAD) STOPs with a check-out-first message
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
│  │     ├─ [if comments exist] → Step 3 (no Copilot review is
│  │     │  requested when existing reviews with comments are found)
│  │     └─ [reviews but no inline comments] "no actionable inline
│  │        comments" → Step 6, outcome no-reviews
│  │
│  └─ Phase 2: Copilot Review Request (no reviews found)
│     ├─ Read config: copilot entry from fab/project/code-review.md § Review Tools
│     ├─ [copilot: false] "No automated reviewer available" → Step 6, outcome no-reviews (clean finish)
│     ├─ Bash: gh pr edit {n} --add-reviewer copilot-pull-request-reviewer
│     │  ├─ [success] Print "Copilot review requested. Waiting up to 10 minutes..."
│     │  │  └─ Poll SYNCHRONOUSLY (no yield mid-poll): gh pr view --json reviews
│     │  │     every 30s, up to 20 attempts; predicate
│     │  │     author.login == "copilot-pull-request-reviewer" (the landed
│     │  │     review-author login, NOT the "Copilot" login under requested_reviewers)
│     │  │     ├─ [review appears] → Step 3
│     │  │     └─ [20 attempts, no review] "...not yet available. Re-run /git-pr-review..." (the suggested command names the explicit <change> when one was passed — an argless re-run resolves the active change, 260612-w7dp) → Step 6, outcome timeout (stage left active — no finish, no fail)
│     │  └─ [failure] "No automated reviewer available..." → Step 6, outcome no-reviews (clean finish)
│
├─ Step 3: Fetch Comments (jq projection: id, path, line, body,
│  │        user, in_reply_to_id — reply comments skipped)
│  └─ Bash: gh api --paginate .../pulls/{n}/comments
│
├─ Step 4: Triage Comments (single classify-and-assign list —
│  │        260611-szxd f098; the Disposition Reference table is the
│  │        single reply-format source)
│  ├─ Classify + assign intent in one pass: fix, defer, skip, or informational
│  ├─ Read: source files at {path}
│  └─ Edit: source files (targeted fixes for "fix" comments)
│
├─ Step 5: Commit and Push (failure handling split — 260612-g8st)
│  ├─ [no modifications] → unpushed-commit re-run gate:
│  │  git log @{u}..HEAD (no-upstream = unpushed) — push any commits
│  │  stranded by a prior failed push, then Step 5.5 (replies cite the
│  │  now-pushed SHA); only a clean gate prints "No changes needed"
│  ├─ Bash: git add {files}
│  ├─ Bash: git commit -m "fix: address review feedback"
│  ├─ Bash: git push
│  ├─ [commit fails] → git reset, STOP (no partial state)
│  └─ [push fails] → KEEP the commit, print recovery guidance
│     (git pull --rebase && git push, then re-run — naming the
│     explicit <change> when one was passed; argless resolves the
│     active change), STOP without posting replies — no "Fixed"
│     reply may cite an unpushed SHA
│     └─ never-hand-merge pointer (260715-xu0k): a conflict in a
│        generated docs/memory/**/index.md or log.md during the
│        pull --rebase recovery (or any branch↔main reconciliation)
│        is resolved by fixing topic files + re-running fab
│        memory-index (wholesale), NOT by hand-editing the generated
│        file (FKF §5)
│
├─ Step 5.5: Post Replies
│  ├─ Deduplicate: skip comments with existing disposition replies
│  ├─ Bash: gh api .../pulls/{n}/comments -f body=... -F in_reply_to=...
│  └─ Best-effort: failed POSTs logged, not fatal
│
├─ Step 6: Update Review-PR Stage (exit point for every terminal
│  │        path after Step 0 — Steps 1/2/4 route here with a
│  │        named outcome. Two direct-STOP exceptions: Step 1.5
│  │        invalid --tool; Step 5 commit failure (after git reset,
│  │        no partial state) or push failure (commit kept + recovery
│  │        guidance, no replies — the re-run's unpushed gate completes
│  │        the cycle))
│  ├─ [success / no-reviews] Bash: fab status finish <change> review-pr
│  ├─ [failure] Bash: fab status fail <change> review-pr
│  └─ [timeout] stage left active — no finish, no fail
│               (re-run picks up the still-active stage)
│
└─ Step 6.5: Commit Status Updates (mirrors git-pr Step 4c)
   ├─ [gate] change resolved in Step 0 (active or explicit) AND Step 6 success/no-reviews path (skip on no-change / fail / timeout path)
   ├─ Bash: git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl
   ├─ Bash: git diff --cached --quiet  (idempotency guard — re-run is a silent no-op)
   ├─ [staged changes] Bash: git commit -m "Update review-pr status" && git push
   └─ [push fails] report error, do NOT STOP (best-effort push; local commit retained)

Phase tracking (via yq directly on .status.yaml):
  received → triaging → fixing → pushed → replying
```

### Copilot Review Request (Phase 2)

Phase 2 runs when Phase 1 finds no existing reviews with inline comments. It requests a Copilot review and polls for up to 10 minutes:

| Tool | Type | Detection | On Success | On Failure |
|------|------|-----------|------------|------------|
| Copilot | Remote | Attempt `gh pr edit --add-reviewer copilot-pull-request-reviewer` | Poll 30s/attempt up to 20× — proceed to Step 3 when review appears; on timeout: Step 6 `timeout` outcome (stage stays active — no finish, no fail) | Clean finish (no-reviews): "No automated reviewer available..." |

The `--tool copilot` flag forces the Copilot path regardless of config — the § Review Tools check is skipped entirely when this flag is present. Without the flag, if `code-review.md` § Review Tools lists `copilot: false`, Phase 2 exits cleanly without attempting the request.

**Poll discipline + query semantics (260615-qg64)**:

- **Synchronous to completion — no yield mid-poll**: when `/git-pr-review` runs as a dispatched subagent (e.g. `/fab-fff` Step 5), the poll MUST run synchronously within the single invocation — the subagent MUST NOT yield, return, or hand back control while the poll is pending; it stays in the loop until a review appears or all 20 attempts (30s × 20 / 10-minute window) are exhausted. This is permanent: the subagent stalled/died mid-poll 4× in prior efforts, leaving `review-pr` stuck `active`; Copilot lands ~4.5–6.5 min (inside the window), so patience-to-completion is correct, not an early return. Cadence is unchanged (30s × 20).
- **Two distinct logins — do not conflate**: you add the reviewer with `gh pr edit --add-reviewer copilot-pull-request-reviewer`, but the entry that then surfaces under the PR's **requested reviewers** carries login `Copilot`; the **landed-review** object in the `reviews` array carries `author.login == "copilot-pull-request-reviewer"` (`[bot]`). The landed-review poll predicate MUST match `author.login == "copilot-pull-request-reviewer"` (the review-author login), NOT `Copilot` (the requested-reviewer login) — a predicate keyed on the requested-reviewer login never matches a landed review object and times out spuriously. (Apparent oddity, recorded as empirical reality: the value passed to `--add-reviewer` matches the landed-review author login, while `requested_reviewers` shows `Copilot`.) This matches the established, deliberately-set behavior (`docs/memory/pipeline/execution-skills.md`: n30u — `"Copilot"` in `requested_reviewers` vs `"copilot-pull-request-reviewer[bot]"` in `reviews`; u1m1 — Phase 2 `.author.login` filter set to `copilot-pull-request-reviewer` so incoming Copilot reviews are detected).
- **GraphQL omits bot reviewers**: confirming the **request** succeeded MUST use REST `requested_reviewers` (`gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers`), since GraphQL `reviewRequests` omits bot/app reviewers like Copilot.

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
| Gate | Runs only when a change was resolved in Step 0 (active or explicit) AND Step 6 took the success / no-reviews path. Skipped silently on the Step 6 `fail` and `timeout` paths and when no change resolved — neither failure path may commit a half-finished state. |
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
