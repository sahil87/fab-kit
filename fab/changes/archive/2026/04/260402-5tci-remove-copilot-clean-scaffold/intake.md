# Intake: Remove Copilot Integration and Clean Stale Scaffold

**Change**: 260402-5tci-remove-copilot-clean-scaffold
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Remove Copilot integration and clean up stale scaffold entries. Four targeted changes: delete the Copilot code review scaffold file, strip Copilot-specific phases from git-pr-review skill, remove stale .gitignore entries from the scaffold fragment, and append cleanup steps to the existing 0.46.0-to-0.47.0 migration.

## Why

The Copilot code review integration assumes every GitHub account has Copilot reviewer enabled. Accounts without it experience a hard failure when `git-pr-review` tries to request a Copilot review (Phase 2) and then polls for 8 minutes (Phase 3) before timing out. This wastes time and produces confusing error messages. The scaffold also ships a `copilot-code-review.yml` workflow file to new projects that serves no purpose without Copilot.

Additionally, the `.gitignore` scaffold fragment includes two stale entries (`fab/changes/**/.pr-done` and `/.ralph`) that reference patterns no longer relevant to the current workflow.

Removing these simplifies the skill, eliminates a failure mode for non-Copilot accounts, and cleans dead references from the scaffold.

## What Changes

### 1. Delete Copilot Code Review Scaffold

Delete `src/kit/scaffold/.github/copilot-code-review.yml` entirely. This file currently contains a Copilot review configuration with path exclusions for `fab/changes/archive/**`. New projects scaffolded after this change will no longer receive this file.

### 2. Strip Copilot Phases from git-pr-review Skill

In `src/kit/skills/git-pr-review.md`:

- **Remove Phase 2** (request Copilot review fallback) — the entire block that POSTs to `requested_reviewers` with `copilot-pull-request-reviewer[bot]`.
- **Remove Phase 3** (poll for Copilot review completion) — the 30-second poll loop with 16 attempts / 8-minute timeout.
- **Remove Path B** from Step 3 (fetch Copilot-specific review comments) — only Path A (fetch all comments) remains.
- **Update routing logic**: After Phase 1, if no existing reviews with comments are found, print `No reviews found — nothing to do.` and STOP. Existing reviews from any source (human, Copilot, bots) still get processed normally via Phase 1 / Path A.
- **Remove Phase Sub-State Tracking `waiting`** entry — that phase no longer exists. The `received` phase description changes to simply "Reviews detected".
- **Remove the API login name discrepancy comment** (the HTML comment block about `Copilot` vs `copilot-pull-request-reviewer[bot]` login names).
- **Update commit message logic in Step 5**: Remove the Copilot-specific branch (`copilot-pull-request-reviewer[bot]` only). Simplify to: single reviewer → `fix: address review feedback from @{username}`, multiple reviewers → `fix: address PR review feedback`.
- **Update description frontmatter**: Change from "human or Copilot" to "human or bot" since the skill still handles any reviewer's comments via Path A.
- **Update Step 6**: Remove the "Copilot unavailable, Copilot timeout" references from the "On no reviews" case — simplify to "no reviews found".

### 3. Clean .gitignore Scaffold Fragment

In `src/kit/scaffold/fragment-.gitignore`:

- Remove the line `fab/changes/**/.pr-done`
- Remove the line `/.ralph`

The resulting file keeps the `.fab-*` pattern and the agent-specific folder ignores (`.agents`, `.claude`, `.cursor`, `.opencode`, `.codex`, `.gemini`).

### 4. Append Migration Steps to 0.46.0-to-0.47.0.md

In `src/kit/migrations/0.46.0-to-0.47.0.md`, append three new sections after the existing "4. Clean .gitignore" section:

**5. Remove Copilot code review config**: If `.github/copilot-code-review.yml` exists, delete it. Print status.

**6. Clean stale .gitignore entries**: If `.gitignore` exists, remove lines matching `/.ralph` and `fab/changes/**/.pr-done`. Print status.

**7. Delete .pr-done files**: Find and delete any `.pr-done` files under `fab/changes/`. Print status.

Update the Verification section to include checks for all three new steps.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update git-pr-review documentation to reflect removal of Copilot phases
- `fab-workflow/distribution`: (modify) Update scaffold file listing to remove copilot-code-review.yml
- `fab-workflow/migrations`: (modify) Document the new migration steps in 0.46.0-to-0.47.0

## Impact

- **Scaffold**: New projects will no longer receive `.github/copilot-code-review.yml`
- **git-pr-review skill**: Simpler flow — check for existing reviews, process them or stop. No Copilot request/poll cycle
- **Existing projects**: Migration removes the Copilot config file, stale .gitignore lines, and leftover .pr-done files
- **No breaking changes**: Accounts that had Copilot enabled will still have their Copilot reviews processed via Phase 1 / Path A if Copilot reviews are already present on the PR

## Open Questions

None — the scope and approach are fully specified.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete copilot-code-review.yml scaffold file | Discussed — user explicitly requested removal | S:95 R:90 A:95 D:95 |
| 2 | Certain | Strip Phase 2 and Phase 3 from git-pr-review | Discussed — user specified exact behavior: stop with message when no reviews found | S:95 R:85 A:90 D:95 |
| 3 | Certain | Keep Phase 1 / Path A processing for all reviewer types | Discussed — existing reviews from any source still processed normally | S:95 R:90 A:95 D:95 |
| 4 | Certain | Remove .pr-done and .ralph from .gitignore scaffold | Discussed — user explicitly listed both entries | S:95 R:95 A:90 D:95 |
| 5 | Certain | Append to existing 0.46.0-to-0.47.0.md migration | Discussed — user specified appending, not creating a new migration file | S:95 R:85 A:95 D:95 |
| 6 | Confident | Remove Path B (Copilot-specific comment fetch) from Step 3 | Implied by removal of Phase 2/3 — no Copilot review_id will be captured, so Path B is unreachable | S:80 R:85 A:90 D:85 |
| 7 | Confident | Update skill description from "human or Copilot" to "human or bot" | Skill still handles bot reviews via Path A, but Copilot is no longer special-cased | S:75 R:95 A:85 D:80 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
