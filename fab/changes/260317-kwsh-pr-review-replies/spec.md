# Spec: PR Review Reply Comments

**Change**: 260317-kwsh-pr-review-replies
**Created**: 2026-03-17
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Thread resolution via GraphQL — would introduce a new API pattern (GraphQL) not used anywhere in the kit. Replies communicate disposition; reviewers can resolve threads themselves.
- Changing existing triage logic — the actionable/informational classification in Step 4 is unchanged. Disposition is an additional layer on top of it.
- Replying to informational comments — praise, summaries, and general questions get no reply, same as today.

## Skill: Disposition Taxonomy

### Requirement: Three-Disposition Classification

The skill SHALL classify each non-informational review comment into exactly one of three disposition intents: `fix`, `defer`, or `skip`. The intent SHALL be assigned during triage (Step 4) alongside the existing actionable/informational classification. Replies posted in Step 5.5 confirm outcomes using past-tense (`Fixed —`, `Deferred —`, `Skipped —`).

#### Scenario: Comment with code fix to apply

- **GIVEN** a review comment identifying a specific code issue
- **WHEN** the skill triages the comment and determines a fix will be applied
- **THEN** the comment SHALL be assigned disposition intent `fix`
- **AND** after the fix is applied and pushed, the reply SHALL include a brief description of the change and the commit SHA

#### Scenario: Comment valid but out of scope

- **GIVEN** a review comment raising a valid concern
- **WHEN** the concern is outside the scope of this PR's changes
- **THEN** the comment SHALL be assigned disposition intent `defer`
- **AND** the reply SHALL include a reason explaining why it was deferred

#### Scenario: Comment is nitpick, stale, or not applicable

- **GIVEN** a review comment that is a nitpick, references stale code, or does not apply to the current code path
- **WHEN** the skill triages the comment
- **THEN** the comment SHALL be assigned disposition intent `skip`
- **AND** the reply SHALL include a reason (e.g., "nitpick", "stale reference", "not applicable to this code path")

#### Scenario: Informational comment

- **GIVEN** a review comment that is purely informational (praise, summary, question without code implication)
- **WHEN** the skill triages the comment
- **THEN** no disposition SHALL be assigned
- **AND** no reply SHALL be posted

### Requirement: Disposition Reply Format

Each disposition intent SHALL produce a reply comment with a specific text format. Replies use past-tense to confirm the outcome.

| Intent (triage) | Reply (outcome) |
|-----------------|-----------------|
| `fix` | `Fixed — {description}. ({sha})` |
| `defer` | `Deferred — {reason}.` |
| `skip` | `Skipped — {reason}.` |

The `{sha}` in `fix` replies SHALL be the short SHA (7 characters) of the commit that contains the fix. The `{description}` SHALL be a concise summary of what was changed (e.g., "added null check for empty input"). The `{reason}` in `defer` and `skip` replies SHALL be free-text determined by the agent per comment.

#### Scenario: Fixed reply format

- **GIVEN** a comment with disposition intent `fix` and description "added null check" and SHA `a3f2b1c`
- **WHEN** the reply is composed
- **THEN** the reply body SHALL be `Fixed — added null check. (a3f2b1c)`

#### Scenario: Deferred reply format

- **GIVEN** a comment with disposition intent `defer` and reason "requires broader refactor outside this PR's scope"
- **WHEN** the reply is composed
- **THEN** the reply body SHALL be `Deferred — requires broader refactor outside this PR's scope.`

#### Scenario: Skipped reply format

- **GIVEN** a comment with disposition intent `skip` and reason "nitpick"
- **WHEN** the reply is composed
- **THEN** the reply body SHALL be `Skipped — nitpick.`

## Skill: Extended Comment Fetching

### Requirement: Capture Comment IDs

Step 3 (Fetch Comments) SHALL expand the `--jq` projection in both Path A and Path B to include `id` and `node_id` fields alongside existing fields.

Path A new projection:
```
{id: .id, node_id: .node_id, path: .path, line: .line, body: .body, user: .user.login, in_reply_to_id: .in_reply_to_id}
```

Path B new projection:
```
{id: .id, node_id: .node_id, path: .path, line: .line, body: .body, user: .user.login}
```

The `id` field is the REST comment ID, used as `in_reply_to_id` when posting reply comments. The `node_id` field is captured for forward compatibility but not used in this change.

#### Scenario: Path A captures IDs

- **GIVEN** the skill is fetching comments via Path A (all reviewers)
- **WHEN** the `gh api` call returns comments
- **THEN** each comment object SHALL include `id` and `node_id` in addition to `path`, `line`, `body`, `user`, and `in_reply_to_id`

#### Scenario: Path B captures IDs

- **GIVEN** the skill is fetching comments via Path B (Copilot-specific review)
- **WHEN** the `gh api` call returns comments
- **THEN** each comment object SHALL include `id` and `node_id` in addition to `path`, `line`, `body`, and `user`

## Skill: Reply Posting

### Requirement: Post Replies After Push

A new Step 5.5 SHALL be added after Step 5 (Commit and Push) and before Step 6 (Update Review-PR Stage). This step posts reply comments to the PR for each comment that received a disposition.

For each comment with a disposition intent (`fix`, `defer`, or `skip`) that passes the deduplication check:

```bash
gh api repos/{owner}/{repo}/pulls/{number}/comments \
  -f body="{reply_text}" \
  -F in_reply_to={comment_id}
```

The field name SHALL be `in_reply_to` (not `in_reply_to_id`) per the GitHub REST API for creating reply comments.

#### Scenario: Post reply for fixed comment

- **GIVEN** a comment with `id=12345` and disposition intent `fix` with description "added null check" and SHA `a3f2b1c`
- **WHEN** Step 5.5 executes
- **THEN** the skill SHALL POST a reply with body `Fixed — added null check. (a3f2b1c)` and `in_reply_to=12345`

#### Scenario: Post reply for deferred comment

- **GIVEN** a comment with `id=12345` and disposition intent `defer` with reason "out of scope"
- **WHEN** Step 5.5 executes
- **THEN** the skill SHALL POST a reply with body `Deferred — out of scope.` and `in_reply_to=12345`

#### Scenario: Post reply for skipped comment

- **GIVEN** a comment with `id=12345` and disposition intent `skip` with reason "stale reference"
- **WHEN** Step 5.5 executes
- **THEN** the skill SHALL POST a reply with body `Skipped — stale reference.` and `in_reply_to=12345`

### Requirement: Reply When No Code Changes

When all comments are defer or skip (zero `fix` intents), the current skill stops at Step 5 with "No changes needed." Instead, the skill SHALL proceed to Step 5.5 to post replies. The communication loop MUST close even when no code was changed.

#### Scenario: All comments defer or skip

- **GIVEN** 3 review comments, all classified as `defer` or `skip`
- **WHEN** Step 5 finds no modifications (`git status --porcelain` is empty)
- **THEN** the skill SHALL skip the commit/push but proceed to Step 5.5
- **AND** reply to each comment with its disposition

### Requirement: Best-Effort Reply Posting

Reply posting SHALL be best-effort. If a reply POST fails for a specific comment, the skill SHALL log the error and continue to the next comment. A failed reply MUST NOT cause the skill to abort or mark the stage as failed. The code changes have already been pushed.

#### Scenario: Reply POST fails

- **GIVEN** 3 comments with dispositions
- **WHEN** the reply POST for comment #2 returns a non-2xx response
- **THEN** the skill SHALL log the error
- **AND** continue posting replies for comment #3
- **AND** the overall skill outcome SHALL still be success

### Requirement: Reply Summary Output

After all replies are posted, Step 5.5 SHALL print:

```
Replied to {N} comment(s): {F} fix, {D} defer, {S} skip
```

Where `{N}` is the total replies posted, `{F}` is fix count, `{D}` defer count, `{S}` skip count.

#### Scenario: Reply summary

- **GIVEN** 5 comments triaged: 2 fix, 1 defer, 2 skip
- **WHEN** all replies are posted successfully
- **THEN** the output SHALL be `Replied to 5 comment(s): 2 fix, 1 defer, 2 skip`

## Skill: Reply Deduplication

### Requirement: Skip Already-Replied Comments

Before posting a reply to a comment, Step 5.5 SHALL check if any existing reply in the fetched comments (those with `in_reply_to_id` matching the target comment's `id`) starts with `Fixed —`, `Deferred —`, or `Skipped —`. If a disposition reply already exists, the skill SHALL skip replying to that comment.

#### Scenario: Re-run after previous reply

- **GIVEN** a previous run posted `Fixed — added null check. (a3f2b1c)` as a reply to comment #123
- **WHEN** the skill re-runs and fetches all comments
- **THEN** the fetched replies SHALL include a comment with `in_reply_to_id=123` and body starting with `Fixed —`
- **AND** the skill SHALL skip posting a reply to comment #123

#### Scenario: Re-run with no previous replies

- **GIVEN** no previous disposition replies exist
- **WHEN** the skill runs and triages comments
- **THEN** all comments with dispositions SHALL receive replies

#### Scenario: Mixed — some replied, some new

- **GIVEN** comment #123 has an existing `Fixed —` reply and comment #456 has no disposition reply
- **WHEN** the skill re-runs
- **THEN** the skill SHALL skip comment #123
- **AND** the skill SHALL post a reply to comment #456

## Skill: Triage Summary Update

### Requirement: Expanded Triage Summary

The triage summary line in Step 4 SHALL be expanded to include disposition counts:

Current: `{N} comments triaged: {A} actionable, {I} skipped`

New: `{N} comments triaged: {F} fix, {D} defer, {S} skip, {I} informational (no reply)`

#### Scenario: Mixed triage results

- **GIVEN** 8 review comments fetched
- **WHEN** triage classifies 3 as fix, 1 as defer, 2 as skip, 2 as informational
- **THEN** the output SHALL be `8 comments triaged: 3 fix, 1 defer, 2 skip, 2 informational (no reply)`

## Skill: Phase Sub-State

### Requirement: Add Replying Phase

The Phase Sub-State Tracking table SHALL add a `replying` phase, set before posting replies in Step 5.5:

| Phase | When set |
|-------|----------|
| `waiting` | After requesting Copilot review |
| `received` | Reviews detected or Copilot review arrived |
| `triaging` | Before classifying comments |
| `fixing` | Before applying fixes |
| `pushed` | After commit and push |
| `replying` | Before posting reply comments (Step 5.5) |

#### Scenario: Phase transitions with replies

- **GIVEN** the skill processes comments, pushes fixes, and posts replies
- **WHEN** Step 5.5 begins
- **THEN** the phase SHALL be set to `replying` before the first reply is posted

#### Scenario: Phase when no code changes but replies needed

- **GIVEN** all comments are defer or skip (no push)
- **WHEN** Step 5.5 begins
- **THEN** the phase SHALL be set to `replying` (skipping `pushed` since no push occurred)

## Skill: Disposition Reference

### Requirement: Reference Table in Skill File

A `## Disposition Reference` section SHALL be added at the bottom of `git-pr-review.md`, after the `## Rules` section. This mirrors the `## PR Type Reference` pattern in `git-pr.md`.

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

#### Scenario: Reference table exists

- **GIVEN** the updated skill file
- **WHEN** an agent reads `git-pr-review.md`
- **THEN** it SHALL find the `## Disposition Reference` section after `## Rules`
- **AND** the table SHALL list all three dispositions with their reply formats

## Design Decisions

1. **REST-only replies, no GraphQL thread resolution**: Post reply comments via REST API (`POST /pulls/{n}/comments` with `in_reply_to`). Do not resolve threads via GraphQL `resolveReviewThread` mutation.
   - *Why*: GraphQL is not used anywhere in the kit. Introducing it for a single cosmetic operation violates the "Pure Prompt Play" principle of minimal external dependencies and adds a new API pattern.
   - *Rejected*: GraphQL `resolveReviewThread` — would resolve threads automatically but introduces a new API surface for marginal benefit. Replies communicate disposition clearly; thread resolution is one click for the reviewer.

2. **Disposition intent assigned during triage, replies posted after push**: Disposition intents are determined in Step 4 (when the agent has full code context) but replies are batched and posted in Step 5.5 (after push succeeds).
   - *Why*: If push fails, no orphaned replies exist. The `fix` intent needs the commit SHA for its reply, which is only available after commit.
   - *Rejected*: Reply-per-comment during triage — would create orphaned replies if push fails, and `fix` replies wouldn't have a SHA.

3. **Deduplication via disposition prefix matching**: Check existing replies for `Fixed —`/`Deferred —`/`Skipped —` prefixes rather than tracking reply state externally.
   - *Why*: Stateless — no extra files, no status.yaml fields. Uses the same comment data already fetched in Step 3. Consistent with the skill's existing idempotency model.
   - *Rejected*: External state tracking (e.g., replied comment IDs in status.yaml) — adds state management complexity for something the API data already provides.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Three disposition intents: fix, defer, skip (replies use past-tense: Fixed/Deferred/Skipped) | Confirmed from intake #1 — user explicitly chose these three; refined to separate intent from outcome | S:95 R:85 A:90 D:95 |
| 2 | Certain | Reply format: `Fixed — {desc}. ({sha})`, `Deferred — {reason}.`, `Skipped — {reason}.` | Confirmed from intake #2 — agreed format with free-text reason | S:90 R:90 A:85 D:90 |
| 3 | Certain | Informational comments get no reply | Confirmed from intake #3 — "nothing you can do" for informational | S:95 R:90 A:90 D:95 |
| 4 | Certain | Replies posted after push, not per-comment during triage | Confirmed from intake #5 — batch-after-push avoids orphaned replies | S:85 R:85 A:90 D:90 |
| 5 | Certain | Use existing `gh api` REST for replies, no new tooling | Confirmed from intake #6 — no new dependencies | S:90 R:90 A:95 D:95 |
| 6 | Certain | No GraphQL thread resolution | User decision — GraphQL not used anywhere in kit, don't introduce new pattern | S:95 R:95 A:95 D:95 |
| 7 | Certain | Reply and resolve are best-effort (don't fail the skill) | Confirmed from intake #7 — code already pushed, reply failure is non-fatal | S:85 R:90 A:85 D:85 |
| 8 | Certain | Disposition Reference table mirrors PR Type Reference from /git-pr | Confirmed from intake #8 — user requested parallel structure | S:80 R:90 A:85 D:85 |
| 9 | Certain | Add `replying` phase to sub-state tracking after `pushed` | Confirmed from intake #10 — consistency with existing phase tracking | S:95 R:90 A:90 D:95 |
| 10 | Certain | Post replies even when no code changes (all defer/skip) | Confirmed from intake #11 — communication loop must close | S:95 R:85 A:90 D:95 |
| 11 | Certain | Deduplicate replies by checking existing reply prefixes | Confirmed from intake #12 — stateless deduplication using fetched data | S:95 R:85 A:90 D:90 |
| 12 | Confident | `in_reply_to` is the correct REST API field name for reply comments | GitHub REST API docs indicate `in_reply_to` for creating pull request review comment replies | S:75 R:85 A:80 D:85 |
| 13 | Certain | `node_id` captured for forward compat but unused in this change | Thread resolution dropped; node_id retained to avoid re-fetching if GraphQL is added later | S:90 R:85 A:85 D:90 |

13 assumptions (12 certain, 1 confident, 0 tentative, 0 unresolved).
