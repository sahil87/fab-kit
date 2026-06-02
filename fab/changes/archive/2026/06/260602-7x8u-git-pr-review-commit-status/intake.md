# Intake: git-pr-review Commit Status Updates

**Change**: 260602-7x8u-git-pr-review-commit-status
**Created**: 2026-06-02
**Status**: Draft

## Origin

> fix: git-pr-review leaves orphaned status writes in the worktree — after Step 6 calls
> `fab status finish <change> review-pr`, the resulting writes to `.status.yaml`
> (review-pr active→done, completed_at, last_updated) and `.history.jsonl` (appended
> review:passed event) are never committed, so they sit as uncommitted artifacts in the
> worktree. Fix: add a "Commit Status Updates" step to git-pr-review.md mirroring
> git-pr.md Step 4c — stage .status.yaml + .history.jsonl, commit ("Update review-pr
> status"), push; skip silently if no changes.

Conversational. Originated from a `/fab-discuss` session investigating why the last
pipeline command leaves behind uncommitted artifacts. Root cause was traced live:
`git-pr-review` Step 6 (`git-pr-review.md:182-190`) calls `fab status finish` — which
produces exactly the observed diff — and then the skill ends with no commit step. Its
sibling `git-pr.md` (Step 4c, lines 303-314) *does* commit its own bookkeeping; the two
stage-finishing skills are asymmetric. User considered and rejected "stop tracking
commands after review-pr" (would lose real completion state); chose to make
`git-pr-review` symmetric with `git-pr` instead.

## Why

1. **Problem**: The `review-pr` stage is the last in the 6-stage pipeline. When
   `git-pr-review` finishes, `fab status finish <change> review-pr` writes the stage to
   `done`, sets `stage_metrics.review-pr.completed_at`, bumps `last_updated`, and appends
   a `{"event":"review","result":"passed",...}` line to `.history.jsonl`. The skill then
   terminates without committing these writes. They remain as uncommitted changes in the
   worktree — orphaned artifacts that never reach the PR and dirty the tree.

2. **Consequence if unfixed**: Every completed pipeline run ends with a dirty worktree.
   This breaks the clean-tree expectation for the terminal stage, creates noise for the
   operator / `/fab-fff` flows, and risks the state write being accidentally discarded
   (e.g., a later `git checkout` / branch cleanup) — losing the authoritative record that
   the change finished its PR-review cycle.

3. **Why this approach over alternatives**: The state write itself is *correct and worth
   keeping* — it is what lets `/fab-status`, `/fab-archive`, and the history log know the
   change completed review-pr. Suppressing the write or untracking the files would trade a
   cosmetic dirty-tree problem for real loss of state. The right fix is to commit the
   write, exactly as `git-pr` already does for its own ship-status bookkeeping. Reusing the
   established, already-tested Step 4c pattern keeps the two terminal skills symmetric and
   parsimonious (no new mechanism invented).

   Rejected alternatives:
   - **Commit before finishing**: non-functional — `fab status finish` is what *produces*
     the diff, so the commit must come after it regardless.
   - **Fold the bookkeeping into the Step 5 code push**: tangles "PR fix" and "stage
     bookkeeping" into one commit, and breaks the no-reviews / no-fix paths where there is
     no code push to attach to.

## What Changes

### `src/kit/skills/git-pr-review.md` — add a commit step after Step 6

Add a new step (after Step 6 "Update Review-PR Stage") that commits the bookkeeping
writes produced by `fab status finish`. Mirror `git-pr.md` Step 4c:

1. **Gate**: Only run if an active change was resolved in Step 0 *and* Step 6's
   `fab status finish` (success / no-reviews path) ran. Skip silently otherwise (no active
   change, or the `fail` path).
2. Stage the status and history files:
   `git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl`
3. Check for staged changes: `git diff --cached --quiet`
4. If changes exist: commit (`git commit -m "Update review-pr status"`) and push
   (`git push`). On commit/push failure → report the error. (See Open Questions re:
   fail-fast vs. best-effort.)
5. If no changes (already committed / idempotent re-run): skip commit+push silently.
6. Print (if committed):
   `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)`

The step must preserve `git-pr-review`'s **idempotency** rule (re-running finds no new
modifications and exits cleanly) — the `git diff --cached --quiet` guard handles this.

### `docs/specs/skills/SPEC-git-pr-review.md` — document the new step

Per constitution ("Changes to skill files MUST update the corresponding
`docs/specs/skills/SPEC-*.md`"), add the new step to the spec's flow/step description and
any tool-usage or bookkeeping tables so the spec stays in sync with the skill.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) — documents git-pr / git-pr-review behavior;
  note that git-pr-review now commits its own review-pr status bookkeeping, mirroring
  git-pr's Step 4c. Implementation-level detail; light touch.

## Impact

- **Code areas**: `src/kit/skills/git-pr-review.md` (canonical source; deployed copy in
  `.claude/skills/` regenerated via `fab sync`), `docs/specs/skills/SPEC-git-pr-review.md`.
- **APIs/commands**: No new fab CLI surface. Reuses existing `git add` / `git commit` /
  `git push` and the already-called `fab status finish`.
- **Behavioral**: The terminal pipeline stage now leaves a clean worktree and an extra
  bookkeeping commit on the PR branch (after the PR is already open — acceptable, matches
  `git-pr`).
- **Dependencies**: none new.

## Open Questions

- On commit/push failure, should the step be **fail-fast** (mirror git-pr Step 4c's
  "report the error and STOP") or **best-effort** (mirror git-pr-review's Step 6 statusman
  calls, which are `2>/dev/null || true` and never abort)? git-pr is fail-fast on its
  commit; git-pr-review's existing ethos for status writes is best-effort. Leaning
  best-effort for the push specifically (a transient push failure should not abort a
  completed review cycle), but fail-fast is defensible for symmetry. (Tentative — resolved
  toward git-pr parity below; revisit at apply if push-flakiness is a concern.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix = add a "Commit Status Updates" step to `git-pr-review.md`, mirroring `git-pr.md` Step 4c | Discussed — user explicitly chose Option 1 over commit-before-finish and fold-into-push; the git-pr Step 4c pattern is the established, tested precedent | S:98 R:80 A:95 D:95 |
| 2 | Certain | Keep the `fab status finish` write; do NOT untrack/suppress it | Discussed — user agreed the write is correct state; only the commit was missing | S:95 R:75 A:95 D:95 |
| 3 | Certain | Also update `docs/specs/skills/SPEC-git-pr-review.md` | Constitution mandates spec update for any skill-file change | S:98 R:70 A:98 D:98 |
| 4 | Confident | Step is gated on active-change-resolved + Step 6 success path; skip silently otherwise | Direct parallel to git-pr Step 4c's "if Step 4a recorded a PR URL" gate; the fail path should not commit a half-finished state | S:80 R:75 A:85 D:80 |
| 5 | Confident | Commit message: `"Update review-pr status"` | Parallels git-pr's `"Update ship status and record PR URL"`; no PR URL recorded here so the shorter message fits | S:75 R:90 A:80 D:80 |
| 6 | Confident | `git diff --cached --quiet` guard preserves idempotency (re-run = no-op) | git-pr-review's Rules section mandates idempotency; same guard git-pr uses | S:85 R:80 A:90 D:85 |
| 7 | Tentative | Push failure handling leans best-effort (don't abort a completed cycle); commit mirrors git-pr | git-pr is fail-fast; git-pr-review's status writes are best-effort. Two valid readings — defer final wording to apply | S:55 R:75 A:60 D:50 |

7 assumptions (3 certain, 3 confident, 1 tentative, 0 unresolved).
