# Spec: Non-Interactive Branch Rename for /git-branch

**Change**: 260226-3g6f-git-branch-non-interactive-rename
**Created**: 2026-02-26
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Changing the branch naming convention (branch name = change folder name) — out of scope
- Adding `--rename` / `--create` flags for manual override — may be added later if needed
- Modifying `wt-create` to set upstream tracking — not relevant, `wt-create` correctly creates local-only branches

## `/git-branch`: Non-Interactive Branch Action

### Requirement: Deterministic branch action on non-main branches

When the user is on a non-main, non-target branch, `/git-branch` SHALL select the branch action deterministically based on upstream tracking status, without presenting an interactive menu.

- If the current branch has **no upstream tracking** (`git config branch.{current}.remote` returns empty), the skill SHALL rename the current branch to the target branch name via `git branch -m "{branch_name}"`.
- If the current branch **has upstream tracking**, the skill SHALL create a new branch from the current HEAD via `git checkout -b "{branch_name}"`.

#### Scenario: Worktree branch with no upstream (most common path)

- **GIVEN** the user is in a worktree on branch `brave-bear` with no upstream tracking
- **AND** the active change is `260226-r3m7-some-feature`
- **WHEN** the user runs `/git-branch`
- **THEN** the branch `brave-bear` is renamed to `260226-r3m7-some-feature`
- **AND** the report shows `Branch: 260226-r3m7-some-feature (renamed from brave-bear)`

#### Scenario: Branch with upstream tracking preserved

- **GIVEN** the user is on branch `experiment-xyz` which has been pushed (upstream tracking exists)
- **AND** the active change is `260226-r3m7-some-feature`
- **WHEN** the user runs `/git-branch`
- **THEN** a new branch `260226-r3m7-some-feature` is created from current HEAD
- **AND** the branch `experiment-xyz` is left intact
- **AND** the report shows `Branch: 260226-r3m7-some-feature (created, leaving experiment-xyz intact)`

### Requirement: Upstream tracking detection

The skill SHALL detect upstream tracking by checking `git config "branch.{current_branch}.remote"`. An empty or error result indicates no upstream; any non-empty result indicates the branch has been pushed.

#### Scenario: Fresh worktree branch has no upstream

- **GIVEN** a worktree was created via `wt-create` (which uses `git worktree add -b`)
- **WHEN** `/git-branch` checks `git config branch.brave-bear.remote`
- **THEN** the result is empty (no upstream configured)
- **AND** the branch qualifies for rename

#### Scenario: Pushed branch has upstream

- **GIVEN** a branch was pushed via `git push -u origin feature-branch`
- **WHEN** `/git-branch` checks `git config branch.feature-branch.remote`
- **THEN** the result is `origin`
- **AND** the branch qualifies for create-new (not rename)

### Requirement: Standalone fallback uses same upstream logic

When `/git-branch` enters standalone fallback mode (explicit argument doesn't match any change), the same upstream-tracking heuristic SHALL apply for the branch action decision.

#### Scenario: Standalone branch rename on local-only branch

- **GIVEN** the user is on branch `brave-bear` with no upstream
- **WHEN** the user runs `/git-branch my-custom-branch` and no change matches
- **THEN** standalone fallback activates
- **AND** `brave-bear` is renamed to `my-custom-branch`
- **AND** the report shows `Branch: my-custom-branch (renamed from brave-bear)`

### Requirement: Existing branch actions preserved

The pre-existing branch actions for "already on target," "target exists," and "on main/master" SHALL NOT change. Only the "on other branch" case is modified.

#### Scenario: Already on target branch (unchanged)

- **GIVEN** the user is already on branch `260226-r3m7-some-feature`
- **AND** that is the target branch name
- **WHEN** the user runs `/git-branch`
- **THEN** no git operation occurs
- **AND** the report shows `Branch: 260226-r3m7-some-feature (already active)`

#### Scenario: Target branch exists elsewhere (unchanged)

- **GIVEN** the user is on `main`
- **AND** branch `260226-r3m7-some-feature` exists locally
- **WHEN** the user runs `/git-branch`
- **THEN** `git checkout 260226-r3m7-some-feature` is executed
- **AND** the report shows `Branch: 260226-r3m7-some-feature (checked out)`

#### Scenario: On main, target doesn't exist (unchanged)

- **GIVEN** the user is on `main`
- **AND** branch `260226-r3m7-some-feature` does not exist
- **WHEN** the user runs `/git-branch`
- **THEN** `git checkout -b 260226-r3m7-some-feature` is executed
- **AND** the report shows `Branch: 260226-r3m7-some-feature (created)`

## Deprecated Requirements

### Interactive 3-option menu for non-main branches

**Reason**: Replaced by deterministic upstream-tracking logic. The "Adopt this branch" default contradicted user intent — invoking `/git-branch` implies wanting a branch named after the change. The "Skip" option is unnecessary — users who don't want a branch operation simply don't run the command.

**Migration**: No migration needed. The new behavior is strictly better — the most common path (worktree rename) becomes automatic, and the less common path (pushed branch) creates a new branch without losing the existing one.

## Design Decisions

1. **Rename over create for local branches**: When a branch has no upstream, renaming is preferred over creating a new branch because it preserves all commits and avoids leaving an orphaned branch behind. The worktree continues tracking the renamed branch automatically.
   - *Why*: `git branch -m` is the minimal-disruption operation — it changes only the name, preserving the full commit history and worktree association.
   - *Rejected*: Always create new branch — leaves orphaned disposable branches (e.g., `brave-bear`) that clutter `git branch` output.

2. **Upstream tracking as the safety guard**: `git config branch.{name}.remote` is the canonical git mechanism for detecting whether a branch has been pushed. It is set by `git push -u` and absent for locally-created branches.
   - *Why*: Simple, reliable, and directly tests the property we care about — "has this branch been shared with a remote?"
   - *Rejected*: Checking reflog for push entries (complex, unreliable), checking if branch name matches a known pattern (fragile, pattern-dependent), always prompting (defeats the non-interactive goal).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Branch name = change folder name (no prefix) | Confirmed from intake #1 — naming spec and config both enforce this | S:95 R:90 A:95 D:95 |
| 2 | Certain | `git branch -m` works correctly in worktrees | Confirmed from intake #2 — git worktree tracks branch by name; rename updates the tracking | S:90 R:85 A:90 D:95 |
| 3 | Confident | `git config branch.{name}.remote` as upstream check | Confirmed from intake #3 — standard git mechanism, `wt-create` never sets upstream, `git push -u` always does | S:85 R:80 A:80 D:75 |
| 4 | Certain | Remove "Adopt" entirely | Confirmed from intake #4 — user-confirmed during discussion, contradicts invocation intent | S:90 R:75 A:85 D:90 |
| 5 | Certain | Same logic for standalone fallback | Confirmed from intake #5 — upstream heuristic applies regardless of how branch name was derived | S:80 R:80 A:85 D:90 |
| 6 | Certain | Report format includes old branch name on rename | New — user needs to see what was renamed for confirmation and traceability | S:85 R:90 A:90 D:95 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
