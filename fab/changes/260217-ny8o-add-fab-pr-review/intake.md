# Intake: Add /fab-pr-review Skill

**Change**: 260217-ny8o-add-fab-pr-review
**Created**: 2026-02-17
**Status**: Draft

## Origin

> Conversational exploration of whether GitHub Copilot PR reviews could be incorporated into the Fab review loop. Analysis of recent PRs showed Copilot reviews arrive ~3.5 minutes after PR creation with substantive findings (stale docs, missing context, contradictions). Discussion evolved from Copilot-specific to source-agnostic — PR comments are PR comments regardless of author. User chose `fab-pr-review` as the name to signal it's a PR-level concern, not a pipeline stage. Positioned as a standalone, opt-in skill run after PR creation.

## Why

1. **Gap in feedback loop**: The Fab pipeline's review stage (sub-agent dispatch) catches spec mismatches and code quality issues pre-commit, but PR-level feedback from Copilot, human reviewers, and other bots arrives post-PR and currently has no structured path back into the rework machinery.
2. **Wasted signal**: Copilot reviews on this repo consistently produce actionable findings (5 comments on PR #113 alone — stale docs, missing context items, contradictions). Without a triage pathway, these either get manually addressed or ignored.
3. **Existing rework machinery is reusable**: The reset/rework options (fix code, revise tasks, revise spec) and stage reset mechanism already exist in `/fab-continue`. This skill only needs to fetch, triage, and bridge into that machinery.

## What Changes

### New Skill: `/fab-pr-review`

A standalone skill (not a pipeline stage) that:

1. **Resolves the PR** for the current change — uses the active change's git branch to find the associated PR via `gh pr list --head <branch>`, or accepts an explicit PR number argument
2. **Fetches all review comments** via `gh api`:
   - PR review bodies (`/repos/{owner}/{repo}/pulls/{number}/reviews`)
   - Inline review comments (`/repos/{owner}/{repo}/pulls/{number}/comments`)
   - General PR comments (`/repos/{owner}/{repo}/issues/{number}/comments`) — optional, may include non-review noise
3. **Triages comments** into the existing three-tier priority scheme:
   - **Must-fix**: Spec mismatches, functional defects, failing test observations, security concerns
   - **Should-fix**: Code quality issues, pattern inconsistencies, stale documentation references
   - **Nice-to-have**: Style suggestions, minor wording improvements, cosmetic feedback
4. **Presents findings** in the same structured format as the sub-agent review output
5. **Offers rework options** (same as `/fab-continue` review failure):
   - **Fix code** — uncheck affected tasks, re-run apply
   - **Revise tasks** — add/modify tasks, re-run apply
   - **Revise spec** — reset to spec stage, regenerate downstream
   - **Dismiss** — acknowledge comments without rework (for clean or nice-to-have-only results)
6. **Handles stage reset** — if rework is chosen and the change has already been hydrated, the skill resets the pipeline to the appropriate stage (e.g., `apply: active`) before rework begins. If the change is archived, it advises running `/fab-archive restore` first.

### Source-Agnostic Design

The skill does not filter or distinguish by comment author. All unresolved PR comments are triaged uniformly — whether from `copilot-pull-request-reviewer[bot]`, a human reviewer, or another bot. This means:
- It naturally handles Copilot's initial review
- Human reviewer comments that arrive later
- Any other review bot comments
- Can be run multiple times as new comments come in

### Not a Pipeline Stage

`/fab-pr-review` is explicitly outside the 6-stage pipeline (`intake → spec → tasks → apply → review → hydrate`). It:
- Does NOT appear in `.status.yaml` progress
- Does NOT have a stage entry in `config.yaml`
- Is user-invoked only (no auto-mode, no pipeline chaining)
- Is closer in nature to `/fab-archive` (standalone housekeeping) than to `/fab-continue` (pipeline progression)

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add `/fab-pr-review` as a standalone skill alongside `/fab-archive`. Document PR resolution, comment fetching, triage, and rework bridge behavior.

## Impact

- **New skill file**: `fab/.kit/skills/fab-pr-review.md`
- **Memory update**: `docs/memory/fab-workflow/execution-skills.md` — new section for `/fab-pr-review`
- **Specs update**: `docs/specs/skills.md` — add skill reference
- **README**: May need a mention in the pipeline overview (as an optional post-PR step)
- **No changes to existing pipeline logic** — reuses existing rework/reset machinery
- **External dependency**: Requires `gh` CLI (already a project dependency per constitution's single-binary rule)

## Open Questions

- Should resolved/outdated PR comments be filtered out, or should all comments be shown for completeness?
- Should the skill log its invocation and findings to `.history.jsonl` (like review does with `log-review`)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Uses `gh api` for PR comment fetching | Constitution mandates single-binary dependencies; `gh` is already approved | S:90 R:95 A:95 D:95 |
| 2 | Certain | Reuses existing must-fix/should-fix/nice-to-have triage scheme | Established pattern from sub-agent review (260216-gqpp-DEV-1040) | S:95 R:90 A:95 D:95 |
| 3 | Certain | Not a pipeline stage | Explicit user decision during conversation — PR review is optional, post-pipeline | S:95 R:85 A:90 D:95 |
| 4 | Confident | PR resolved via branch name matching | Standard `gh pr list --head <branch>` pattern; explicit PR number as fallback | S:70 R:90 A:85 D:80 |
| 5 | Confident | Offers same 3 rework options + dismiss | Natural extension of existing rework menu; dismiss needed for clean/nice-to-have-only results | S:75 R:85 A:80 D:75 |
| 6 | Tentative | Fetches all 3 comment types (reviews, inline, issue comments) | Issue comments may include non-review noise; inline comments are the most valuable signal | S:60 R:80 A:55 D:50 |
| 7 | Tentative | Handles archived changes by advising restore | Could auto-restore, but that's a side-effect-heavy action for a review skill | S:55 R:50 A:60 D:55 |

7 assumptions (3 certain, 2 confident, 2 tentative, 0 unresolved).
