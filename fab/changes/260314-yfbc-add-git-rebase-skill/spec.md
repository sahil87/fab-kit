# Spec: Add Git Rebase Skill

**Change**: 260314-yfbc-add-git-rebase-skill
**Created**: 2026-03-15
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Git Workflow: Rebase Skill

### Requirement: Skill File Structure

The skill file `fab/.kit/skills/git-rebase.md` SHALL follow the standard skill frontmatter format with `name`, `description`, and `allowed-tools` fields. The `allowed-tools` field SHALL be `Bash(git:*), AskUserQuestion`.

#### Scenario: Skill file exists with correct frontmatter
- **GIVEN** the fab kit is installed
- **WHEN** an agent reads `fab/.kit/skills/git-rebase.md`
- **THEN** the file contains YAML frontmatter with `name: git-rebase`
- **AND** `description` summarizes the rebase behavior
- **AND** `allowed-tools` includes `Bash(git:*)` and `AskUserQuestion`

### Requirement: Git Repository Guard

The skill SHALL verify it is running inside a git repository before any other operation. If not in a git repo, it SHALL report "Not inside a git repository." and stop.

#### Scenario: Not in a git repo
- **GIVEN** the current directory is not inside a git repository
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill reports "Not inside a git repository."
- **AND** no git commands are executed

### Requirement: Branch Guard

The skill SHALL refuse to run when the current branch is `main` or `master`. It SHALL report the branch name and advise switching to a feature branch.

#### Scenario: On main branch
- **GIVEN** the current branch is `main`
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill reports "Cannot rebase — already on main. Switch to a feature branch first."
- **AND** no fetch or rebase is attempted

#### Scenario: On master branch
- **GIVEN** the current branch is `master`
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill reports "Cannot rebase — already on master. Switch to a feature branch first."

#### Scenario: On a feature branch
- **GIVEN** the current branch is `260314-yfbc-add-git-rebase-skill`
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill proceeds past the branch guard

### Requirement: Uncommitted Changes Detection

The skill SHALL check for uncommitted changes via `git status --porcelain` before fetching or rebasing. If changes exist, it SHALL display the changes and ask the user to choose between stashing or aborting.

#### Scenario: Clean working tree
- **GIVEN** the working tree has no uncommitted changes
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill proceeds directly to fetch and rebase without prompting

#### Scenario: Uncommitted changes — user chooses stash
- **GIVEN** the working tree has uncommitted changes
- **WHEN** the user runs `/git-rebase`
- **THEN** the skill displays the pending changes via `git status --short`
- **AND** presents two options: stash-and-rebase or abort
- **WHEN** the user chooses stash
- **THEN** the skill runs `git stash push -m "git-rebase: auto-stash before rebase"`
- **AND** proceeds to fetch and rebase
- **AND** runs `git stash pop` after successful rebase

#### Scenario: Uncommitted changes — user chooses abort
- **GIVEN** the working tree has uncommitted changes
- **WHEN** the user runs `/git-rebase` and chooses abort
- **THEN** no git operations are performed
- **AND** the skill stops with guidance to commit or handle changes first

### Requirement: Main Branch Auto-Detection

The skill SHALL auto-detect whether the repository uses `main` or `master` as its primary branch via `git rev-parse --verify main`. If `main` exists, use `main`; otherwise fall back to `master`.

#### Scenario: Repository uses main
- **GIVEN** the repository has a `main` branch
- **WHEN** the skill detects the main branch
- **THEN** it uses `origin/main` as the rebase target

#### Scenario: Repository uses master
- **GIVEN** the repository has no `main` branch but has `master`
- **WHEN** the skill detects the main branch
- **THEN** it uses `origin/master` as the rebase target

### Requirement: Fetch and Rebase

The skill SHALL fetch the latest main branch from origin and rebase the current branch onto it. Fetch and rebase are separate operations with independent error handling.

#### Scenario: Successful rebase
- **GIVEN** the current branch is a feature branch with no uncommitted changes
- **WHEN** the skill runs fetch and rebase
- **THEN** `git fetch origin {main_branch}` is executed
- **AND** `git rebase origin/{main_branch}` is executed
- **AND** the skill reports "Rebased {branch} onto origin/{main_branch}."

#### Scenario: Fetch failure
- **GIVEN** the remote `origin` is unreachable
- **WHEN** the skill runs `git fetch origin {main_branch}`
- **THEN** the skill reports the fetch error
- **AND** if changes were stashed, runs `git stash pop` before stopping
- **AND** no rebase is attempted

#### Scenario: Rebase conflict
- **GIVEN** the current branch has commits that conflict with main
- **WHEN** `git rebase origin/{main_branch}` encounters conflicts
- **THEN** the skill reports "Rebase conflict detected."
- **AND** advises `git rebase --continue` or `git rebase --abort`
- **AND** if changes were stashed, notes that `git stash pop` is needed after conflict resolution

### Requirement: Stash Safety

When stashing is used, the skill SHALL ensure stashed changes are restored after a successful rebase. If `git stash pop` produces conflicts, the skill SHALL report them and stop.

#### Scenario: Stash pop succeeds
- **GIVEN** changes were stashed before rebase
- **WHEN** the rebase completes successfully
- **THEN** `git stash pop` restores the stashed changes
- **AND** the skill reports "Rebased {branch} onto origin/{main_branch}. Stashed changes restored."

#### Scenario: Stash pop conflicts
- **GIVEN** changes were stashed before rebase
- **WHEN** `git stash pop` produces merge conflicts
- **THEN** the skill reports the conflicts and stops

### Requirement: No Fab State Modification

The skill SHALL NOT modify `.fab-status.yaml`, `.status.yaml`, or any fab pipeline state. It is a pure git operation skill, consistent with `/git-branch`.

#### Scenario: Fab state unchanged after rebase
- **GIVEN** a successful rebase
- **WHEN** the operation completes
- **THEN** no files in `fab/` are modified by the skill itself

### Requirement: Skill Deployment

The skill SHALL be deployed via the existing `fab-sync.sh` mechanism. No changes to sync scripts are needed — `2-sync-workspace.sh` auto-discovers `fab/.kit/skills/*.md`.

#### Scenario: Sync deploys the skill
- **GIVEN** `fab/.kit/skills/git-rebase.md` exists
- **WHEN** `fab-sync.sh` runs
- **THEN** `.claude/skills/git-rebase/SKILL.md` is created as a deployed copy

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill follows existing git-skill pattern | Confirmed from intake #1 — matches git-branch, git-pr structure | S:90 R:90 A:95 D:95 |
| 2 | Certain | Branch guard rejects main/master only | Confirmed from intake #2 — explicit user requirement | S:95 R:85 A:90 D:95 |
| 3 | Certain | Uncommitted changes prompt via AskUserQuestion | Confirmed from intake #3 — stash-or-abort flow | S:95 R:80 A:90 D:90 |
| 4 | Confident | Auto-detect main vs master via git rev-parse | Confirmed from intake #4 — standard portability pattern | S:70 R:90 A:85 D:85 |
| 5 | Certain | No fab state modifications | Confirmed from intake #5 — pure git skill | S:85 R:95 A:95 D:95 |
| 6 | Certain | Allowed tools: Bash(git:*) and AskUserQuestion | Confirmed from intake #6 | S:90 R:90 A:90 D:95 |
| 7 | Certain | No arguments — operates on current branch only | Codebase signal — consistent with the implemented skill file | S:85 R:95 A:95 D:95 |
| 8 | Certain | Deployment via existing fab-sync — no sync changes needed | Auto-discovery of *.md in skills/ is already implemented | S:90 R:95 A:95 D:95 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
