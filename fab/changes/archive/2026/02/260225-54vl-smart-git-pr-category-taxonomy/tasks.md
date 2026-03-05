# Tasks: Smart git-pr Category Taxonomy

**Change**: 260225-54vl-smart-git-pr-category-taxonomy
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 Add Step 0 (Type Resolution) to `fab/.kit/skills/git-pr.md` — insert a new section before Step 1 that resolves the PR type via the three-step chain: (1) check for explicit argument matching one of 7 valid types (feat, fix, refactor, docs, test, ci, chore), (2) if no argument and `changeman.sh resolve` succeeds with `intake.md` present, pattern-match intake content case-insensitively for fix/refactor/feat keywords, (3) if no argument and no fab change, analyze `git diff --name-only` output for ci/docs/test/chore patterns with chore as default. Store the resolved type for use in Steps 3c and title generation.

- [x] T002 Rewrite Step 3c (Create PR) in `fab/.kit/skills/git-pr.md` — replace the current intake-aware PR creation with the two-tier template system. Tier 1 (feat/fix/refactor): derive summary from intake's Why section, changes from What Changes subsection headings, context table with Type, Change name, and blob URL links to intake + spec. Tier 2 (docs/test/ci/chore): auto-generate summary from commit messages or diff stat, context table with Type only, explicit "No design artifacts — housekeeping change." note. Both tiers construct the PR title as `{type}: {title}` where title comes from intake H1 (Tier 1) or commit subject (Tier 2).

- [x] T003 Add blob URL construction to `fab/.kit/skills/git-pr.md` — within Step 3c's Tier 1 template, derive `{owner}/{repo}` from `gh repo view --json nameWithOwner -q '.nameWithOwner'` and `{branch}` from `git branch --show-current`. Construct links as `https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/intake.md` (and spec.md). Include the spec link only if `spec.md` exists in the change folder.

## Phase 2: Integration & Edge Cases

- [x] T004 Update Step 1b (Branch Mismatch Nudge) in `fab/.kit/skills/git-pr.md` — the nudge currently fires when changeman resolution succeeds and the branch doesn't match. For lightweight changes (no fab change), changeman resolution will fail silently and the nudge is skipped. Verify this behavior is documented clearly: "If resolution fails or there is no active change, skip this step silently" already handles it. No code change needed if the existing guard is correct; otherwise adjust the guard.

- [x] T005 Update Steps 4–4c in `fab/.kit/skills/git-pr.md` — verify that Steps 4 (Record Shipped), 4b (Commit Status), and 4c (Write Sentinel) gracefully skip when no fab change exists. The current spec says "If resolution fails... skip silently" which should handle lightweight changes. Confirm existing guards are sufficient and document the lightweight skip path explicitly in the skill text.

## Phase 3: Polish

- [x] T006 Add the PR type taxonomy reference table to `fab/.kit/skills/git-pr.md` — add a reference section (after the Rules section) listing the 7 valid types with descriptions and which template tier each uses. This serves as inline documentation for the skill and as the canonical type reference.

---

## Execution Order

- T001 blocks T002 (type resolution must exist before template can use it)
- T003 is part of T002's template but extracted for focus; execute after T002
- T004 and T005 are independent verification tasks, can run after T001-T003
- T006 is independent polish
