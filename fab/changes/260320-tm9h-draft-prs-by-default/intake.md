# Intake: Draft PRs by Default

**Change**: 260320-tm9h-draft-prs-by-default
**Created**: 2026-03-20
**Status**: Draft

## Origin

> [tm9h] 2026-03-19: When starting to create PRs - create drafts. fab-kit should always create draft PRs. The devs generally need to check the implementation before marking it ok for code review

One-shot from backlog. Related backlog item [m1ef] covers the same request.

## Why

The `/git-pr` skill currently creates ready-for-review PRs via `gh pr create`. This is premature — developers need to inspect the agent-generated implementation before exposing it to code reviewers. A ready PR signals "this is review-worthy" when the author hasn't verified it yet. Draft PRs give the developer a chance to sanity-check the diff, run manual verification, and mark it ready only when they're confident the implementation is correct. Without this, reviewers waste time on PRs that the author would have caught themselves.

## What Changes

### Add `--draft` flag to `gh pr create` in `/git-pr`

In `fab/.kit/skills/git-pr.md`, Step 3c item 4, the PR creation command changes from:

```
gh pr create --title "{pr_title}" --body "<body>"
```

to:

```
gh pr create --draft --title "{pr_title}" --body "<body>"
```

This is unconditional — all PRs created by fab-kit are drafts. No configuration toggle.

### Update spec file

Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file"), `docs/specs/skills/SPEC-git-pr.md` needs to reflect the draft behavior if it documents the `gh pr create` invocation.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document that `/git-pr` creates draft PRs by default

## Impact

- `fab/.kit/skills/git-pr.md` — primary change: add `--draft` flag
- `docs/specs/skills/SPEC-git-pr.md` — update spec to reflect draft behavior (if it exists)
- Related backlog: `[m1ef]` is a duplicate and should be marked done when this ships

## Open Questions

(none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `gh pr create --draft` flag | gh CLI natively supports `--draft`; this is the standard mechanism | S:90 R:95 A:95 D:95 |
| 2 | Certain | Change applies to `/git-pr` skill only | Only skill that invokes `gh pr create` | S:85 R:90 A:95 D:95 |
| 3 | Certain | No configuration toggle | User explicitly stated "always create draft PRs" — unconditional | S:90 R:85 A:90 D:95 |
| 4 | Confident | Mark [m1ef] done when this ships | [m1ef] is a duplicate backlog item covering the same request | S:75 R:90 A:70 D:80 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
