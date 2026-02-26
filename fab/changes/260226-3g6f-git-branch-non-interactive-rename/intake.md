# Intake: Non-Interactive Branch Rename for /git-branch

**Change**: 260226-3g6f-git-branch-non-interactive-rename
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Make git-branch non-interactive by defaulting to rename for local-only branches. When on a non-main branch, check upstream tracking: if no upstream (local-only), rename the branch to the change name; if upstream exists (already pushed), create a new branch. Remove the interactive 3-option menu and the "Adopt" concept entirely. This makes the worktree workflow seamless — wt-create's random branch name gets renamed to the change name without prompting.

Preceded by a `/fab-discuss` session that analyzed the problem, evaluated alternatives, and converged on the rename-with-upstream-guard approach. Key decisions were made during discussion.

## Why

The current `/git-branch` behavior when on a non-main, non-target branch presents an interactive 3-option menu (Create new / Adopt / Skip) with "Adopt this branch" as the default. This is wrong for two reasons:

1. **Intent mismatch**: Users invoke `/git-branch` to get a branch named after their change. If they wanted to keep the current branch, they wouldn't run the command. Defaulting to "Adopt" (no-op) contradicts the reason the command was invoked.

2. **Worktree friction**: In the standard worktree flow (`wt-create` → `/fab-new` → `/git-branch`), `wt-create` puts you on a random branch like `brave-bear`. The user wants `260226-r3m7-some-feature`. The current default says "keep `brave-bear`" — which is never what's wanted. The random worktree branch name is disposable.

If we don't fix this, every `/git-branch` invocation in a worktree requires the user to manually select option 1 (Create new branch) instead of accepting the default. This is unnecessary friction in the most common workflow path.

## What Changes

### Remove the Interactive Menu from Step 4

Replace the 3-option interactive menu (lines 115-119 of `git-branch.md`) with deterministic logic:

**Current behavior** (on non-main, non-target branch):
```
Present options:
1. Create new branch
2. Adopt this branch (default)
3. Skip
```

**New behavior** (on non-main, non-target branch):
```bash
# Check if the current branch has upstream tracking
upstream=$(git config "branch.$(git branch --show-current).remote" 2>/dev/null || true)

if [ -z "$upstream" ]; then
    # Local-only branch — safe to rename
    git branch -m "{branch_name}"
    # Report: "renamed from {old_branch}"
else
    # Has upstream — create new branch to preserve remote tracking
    git checkout -b "{branch_name}"
    # Report: "created (leaving {old_branch} intact)"
fi
```

### Updated Step 4 Decision Table

The full Step 4 logic becomes:

| Current state | Action | Report |
|---|---|---|
| Already on `{branch_name}` | No-op | `(already active)` |
| Target branch exists as different local branch | `git checkout "{branch_name}"` | `(checked out)` |
| On `main`/`master` | `git checkout -b "{branch_name}"` | `(created)` |
| On other branch, **no upstream** | `git branch -m "{branch_name}"` | `(renamed from {old_branch})` |
| On other branch, **has upstream** | `git checkout -b "{branch_name}"` | `(created, leaving {old_branch} intact)` |

### Updated Report Format

Add the new `renamed` verb to the report output and the "leaving intact" qualifier:

```
Branch: {branch_name} (renamed from {old_branch})
Branch: {branch_name} (created, leaving {old_branch} intact)
```

### Remove "Adopt" Concept

Remove all references to "Adopt this branch" from:
- `git-branch.md` Step 4 options
- `git-branch.md` Step 5 report format (remove `adopted` verb)
- Error handling table (no "Adopt" path to document)

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update `/git-branch` behavior section — replace the 3-option table with new deterministic logic, remove "Adopt" references, add rename and upstream-guard behavior
- `fab-workflow/change-lifecycle`: (modify) Update the branch management options table and the `/git-branch` behavior description — replace Adopt/Create/Skip with the new rename/create deterministic logic

## Impact

- **`fab/.kit/skills/git-branch.md`** — primary change: rewrite Step 4, update Step 5 report format, update error handling table
- **`docs/specs/skills.md`** — update `/git-branch` behavior section to reflect new logic
- **`docs/memory/fab-workflow/execution-skills.md`** — update changelog and behavior description
- **`docs/memory/fab-workflow/change-lifecycle.md`** — update branch management options table
- No shell scripts affected — this is purely a skill file (prompt) change plus docs
- No `.status.yaml` schema changes
- No new dependencies

## Open Questions

(none — all questions resolved during prior discussion)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Branch name = change folder name (no prefix) | Config and naming spec confirm this convention — no prefix applied | S:95 R:90 A:95 D:95 |
| 2 | Certain | `git branch -m` works in worktrees | Verified — git branch rename operates on the worktree's tracked branch correctly | S:90 R:85 A:90 D:95 |
| 3 | Confident | Use `git config branch.{name}.remote` as the upstream check | Discussed — this is the standard git mechanism for detecting upstream tracking. `wt-create` never sets upstream, so worktree branches will always pass. Pushed branches will have this set via `git push -u`. | S:85 R:80 A:80 D:75 |
| 4 | Certain | Remove "Adopt" entirely rather than keeping it as a fallback | Discussed — user confirmed. Adopt contradicts invocation intent. Users who want to keep the current branch simply don't run `/git-branch`. | S:90 R:75 A:85 D:90 |
| 5 | Certain | Same logic applies to standalone fallback path | Discussed — when the user types an explicit branch name, the rename-vs-create decision should follow the same upstream-tracking heuristic | S:80 R:80 A:85 D:90 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
