# Intake: PR Review Reply Comments

**Change**: 260317-kwsh-pr-review-replies
**Created**: 2026-03-17
**Status**: Draft

## Origin

> Add reply comments to /git-pr-review. Currently the skill fetches Copilot/human review comments, triages them, applies fixes, commits and pushes — but never replies to the comment threads. We want it to post reply comments on the PR after processing each review comment, making it fully autonomous end-to-end.

Conversational mode — extended discussion preceded this intake. Key decisions were made about disposition taxonomy, reply format, and REST-only approach (no GraphQL).

## Why

1. **Visibility gap**: When `/git-pr-review` fixes a review comment, the reviewer sees a new commit but has no thread-level confirmation of what was addressed, deferred, or intentionally skipped. They must manually diff-hunt to understand the response to each comment.
2. **Incomplete autonomy**: The skill is "fully autonomous" for fixing code but stops short of the final communication step — replying to the reviewer. A human would always reply before moving on; the skill should too.
3. **Stale threads**: Without replies or resolution, review threads remain open indefinitely, cluttering the PR conversation and making it hard to see what's actually unresolved.

## What Changes

### 1. Disposition Taxonomy

Add a three-state classification for how each review comment is handled. This mirrors the PR Type Reference pattern in `/git-pr` — a documented taxonomy with clear definitions.

Triage assigns **intent** (action verb); replies confirm **outcome** (past-tense).

| Intent (triage) | Description | Reply (outcome) |
|-----------------|-------------|-----------------|
| `fix` | Code will be changed to address the comment | `Fixed — {description}. ({sha})` |
| `defer` | Valid concern, out of scope for this PR | `Deferred — {reason}.` |
| `skip` | Nitpick, stale, or not applicable | `Skipped — {reason}.` |

The `{description}` in `fix` replies is a brief summary of what was changed (e.g., "added null check for empty input"). The `{reason}` in `defer` and `skip` replies explains why (e.g., "requires broader refactor outside this PR's scope", "stale reference, code already changed", "nitpick", "not applicable to this code path").

**Informational comments** (praise, summaries, "LGTM", general questions without code implications) receive **no reply** — they are filtered out before disposition applies, same as the current "informational" classification in Step 4.

### 2. Extended Comment Fetching (Step 3)

Expand the `--jq` projections in both Path A and Path B to capture additional fields needed for replies:

- `id` — REST comment ID, used as `in_reply_to` when posting reply comments
- `node_id` — captured for forward compatibility (not used in this change)

Current projection:
```
{path: .path, line: .line, body: .body, user: .user.login, in_reply_to_id: .in_reply_to_id}
```

New projection:
```
{id: .id, node_id: .node_id, path: .path, line: .line, body: .body, user: .user.login, in_reply_to_id: .in_reply_to_id}
```

### 3. Disposition Assignment in Triage (Step 4)

The current Step 4 classifies comments as **actionable** or **informational**. Extend this to assign one of three disposition intents to each non-informational comment:

1. **`fix`** — the comment identifies a specific code issue and a fix will be applied
2. **`defer`** — the comment raises a valid concern but it's out of scope for this PR
3. **`skip`** — the comment is a nitpick, stale, or not applicable

The agent determines disposition intent and reason text per comment based on the comment body, code context, and whether a fix will be applied. The triage summary line expands:

Current: `{N} comments triaged: {A} actionable, {I} skipped`
New: `{N} comments triaged: {F} fix, {D} defer, {S} skip, {I} informational (no reply)`

### 4. Reply Deduplication (Idempotency)

On re-run, the skill must not post duplicate replies. During comment fetching (Step 3), reply comments (`in_reply_to_id` non-null) are already captured. Before posting a reply to a comment, check if any existing reply in that thread starts with `Fixed —`, `Deferred —`, or `Skipped —` (the disposition prefixes). If a disposition reply already exists for that comment, skip replying to it.

This works with the existing idempotency model: the current skill already skips reply comments during triage (only processes originals). The deduplication check adds the same principle to the reply-posting step — don't reply twice.

### 5. Phase Sub-State: `replying`

Add a `replying` phase to the Phase Sub-State Tracking table, set before posting replies (after `pushed`). This keeps the sub-state tracking consistent:

`waiting` → `received` → `triaging` → `fixing` → `pushed` → `replying`

### 6. New Step 5.5: Post Replies (after push, or when no code changes)

After the successful commit and push in Step 5, add a new step that posts reply comments. This step runs **after push** to ensure all `fix` replies reference a real commit SHA.

**When no code changes were made** (all comments are defer or skip, zero `fix` intents): the current skill stops at Step 5 with "No changes needed." Instead of stopping, the skill should proceed to this reply step — defer and skip replies don't reference a SHA, so they work without a push. The communication loop must still close even when no code was changed.

**For each comment with a disposition intent** (fix, defer, or skip) **that doesn't already have a disposition reply**:

**Post reply comment** via REST API:
```bash
gh api repos/{owner}/{repo}/pulls/{number}/comments \
  -f body="{reply_text}" \
  -F in_reply_to={comment_id}
```

**Error handling**: Reply posting is **best-effort**. If a reply POST fails, log the error and continue to the next comment — do not abort the entire step. The code changes have already been pushed; failing to comment should not be treated as a skill failure.

**Output**: After all replies, print:
```
Replied to {N} comment(s): {F} fix, {D} defer, {S} skip
```

### 7. Disposition Reference Table

Add a `## Disposition Reference` section at the bottom of the skill file, mirroring the `## PR Type Reference` pattern in `/git-pr`:

```markdown
## Disposition Reference

Triage assigns **intent** (action verb); replies confirm **outcome** (past-tense).

| Intent (triage) | Description | Reply (outcome) |
|-----------------|-------------|-----------------|
| `fix` | Code will be changed to address the comment | `Fixed — {description}. ({sha})` |
| `defer` | Valid concern, out of scope for this PR | `Deferred — {reason}.` |
| `skip` | Nitpick, stale, or not applicable | `Skipped — {reason}.` |

Informational comments (praise, summaries, questions without code implications) receive no reply.
```

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document disposition taxonomy and reply behavior in the `/git-pr-review` section

## Impact

- **Skill file**: `fab/.kit/skills/git-pr-review.md` — primary change target (new step, expanded triage, reference table)
- **Spec file**: `docs/specs/skills/SPEC-git-pr-review.md` — must be updated per constitution constraint
- **GitHub API surface**: Adds one new API interaction (POST reply comment via REST) — uses the existing `gh` CLI, no new dependencies
- **No breaking changes**: Existing behavior (fetch → triage → fix → commit → push) is unchanged; the new step is additive after push

## Clarifications

### Session 2026-03-17

| # | Action | Detail |
|---|--------|--------|
| 10 | Added | Phase sub-state `replying` — recommended approach accepted |
| 11 | Added | Reply even with no code changes — recommended approach accepted |
| 12 | Added | Reply deduplication on re-run — user raised idempotency concern |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Three disposition intents: fix, defer, skip (replies use past-tense: Fixed/Deferred/Skipped) | Discussed — user explicitly chose these three after rejecting alternatives; refined to separate intent from outcome | S:95 R:85 A:90 D:95 |
| 2 | Certain | Reply format: `Fixed — {desc}. ({sha})`, `Deferred — {reason}.`, `Skipped — {reason}.` | Discussed — user agreed on disposition + reason format with free-text reason per comment | S:90 R:90 A:85 D:90 |
| 3 | Certain | Informational comments get no reply | Discussed — user said "there's nothing like what you can do" for informational comments | S:95 R:90 A:90 D:95 |
| 4 | Certain | Replies posted after push, not per-comment during triage | Discussed — agreed batch-after-push is safer (avoids orphaned replies if push fails) | S:85 R:85 A:90 D:90 |
| 5 | Certain | REST-only replies, no GraphQL thread resolution | User decision — GraphQL not used anywhere in kit, don't introduce new pattern | S:95 R:95 A:95 D:95 |
| 6 | Certain | Reply posting is best-effort (don't fail the skill) | Code is already pushed; failing to comment shouldn't be treated as a skill failure | S:85 R:90 A:85 D:85 |
| 7 | Confident | Disposition Reference table mirrors PR Type Reference pattern from /git-pr | Discussed — user asked to check /git-pr for parallel structure | S:80 R:90 A:85 D:85 |
| 8 | Certain | Add `replying` phase to sub-state tracking after `pushed` | Clarified — user chose recommended approach for consistency with existing phase tracking | S:95 R:90 A:90 D:95 |
| 9 | Certain | Post replies even when no code changes (all defer/skip) | Clarified — user chose recommended approach; communication loop must close regardless | S:95 R:85 A:90 D:95 |
| 10 | Certain | Deduplicate replies on re-run by checking for existing disposition prefix replies | Clarified — user raised idempotency concern; check for `Fixed —`/`Deferred —`/`Skipped —` prefixes | S:95 R:85 A:90 D:90 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
