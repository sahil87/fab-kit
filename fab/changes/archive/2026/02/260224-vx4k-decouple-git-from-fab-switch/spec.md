# Spec: Decouple Git from Fab Switch

**Change**: 260224-vx4k-decouple-git-from-fab-switch
**Created**: 2026-02-24
**Affected memory**: `docs/memory/fab-workflow/change-lifecycle.md` (modify), `docs/memory/fab-workflow/execution-skills.md` (modify)

## Non-Goals

- Pipeline isolation abstraction (worktree vs copy) — out of scope per user decision
- `created_by` detection changes — `detect_created_by()` in `changeman.sh` stays as-is
- Changes to `/git-pr` commit message generation — only branch-awareness behavior changes
- Changes to `create_worktree()` in `dispatch.sh` — its own git branching is independent

## Change Lifecycle: fab-switch Git Removal

### Requirement: fab-switch SHALL NOT perform git operations

`/fab-switch` SHALL only resolve the change name and write it to `fab/current`. It SHALL NOT create, checkout, or otherwise modify git branches. The `--branch <name>` and `--no-branch-change` arguments SHALL be removed.

#### Scenario: Normal switch on main branch
- **GIVEN** the user is on the `main` branch with `git.enabled: true`
- **WHEN** the user runs `/fab-switch mychange`
- **THEN** `fab/current` is written with the resolved change name
- **AND** the git branch remains on `main` (no checkout, no branch creation)

#### Scenario: Switch with no git repo
- **GIVEN** the working directory is not inside a git repository
- **WHEN** the user runs `/fab-switch mychange`
- **THEN** `fab/current` is written with the resolved change name
- **AND** no git commands are executed
- **AND** no error or warning about git is shown

#### Scenario: Former --branch flag is rejected
- **GIVEN** a user runs `/fab-switch mychange --branch feat/my-branch`
- **WHEN** the skill parses arguments
- **THEN** it reports an unknown argument error for `--branch`

#### Scenario: Former --no-branch-change flag is rejected
- **GIVEN** a user runs `/fab-switch mychange --no-branch-change`
- **WHEN** the skill parses arguments
- **THEN** it reports an unknown argument error for `--no-branch-change`
<!-- clarified: --no-branch-change hard error — user chose immediate cleanup over backward compat. dispatch.sh must be updated in the same change. -->

### Requirement: changeman.sh switch SHALL be git-free

`changeman.sh cmd_switch()` SHALL remove all git branch integration code (config reading for `git.enabled`/`git.branch_prefix`, `git rev-parse`, `git show-ref`, `git checkout`). The `Branch:` output line SHALL be removed from the summary format.

#### Scenario: changeman.sh switch output format
- **GIVEN** changeman.sh switch resolves a change successfully
- **WHEN** it outputs the summary
- **THEN** the output matches:
  ```
  fab/current → {name}

  Stage:  {display_stage} ({N}/6) — {state}
  Next:   {routing_stage} (via {default_command})
  ```
- **AND** no `Branch:` line is present

### Requirement: fab-switch output SHOULD suggest /git-branch

When `git.enabled` is `true` in `config.yaml`, `/fab-switch` output SHOULD include a hint line after the summary suggesting `/git-branch` for branch operations.
<!-- clarified: hint confirmed by user — aids discoverability for the new workflow -->

#### Scenario: Hint shown when git enabled
- **GIVEN** `config.yaml` has `git.enabled: true`
- **WHEN** `/fab-switch mychange` completes successfully
- **THEN** the output includes a line: `Tip: run /git-branch to create or switch to the matching branch`

#### Scenario: No hint when git disabled
- **GIVEN** `config.yaml` has `git.enabled: false`
- **WHEN** `/fab-switch mychange` completes successfully
- **THEN** no git-related hint is shown

## Git Branch: New Command

### Requirement: /git-branch SHALL create or checkout a matching branch

`/git-branch` SHALL be a new command (skill file `fab/.kit/skills/git-branch.md`) that creates or checks out a git branch named `{branch_prefix}{change-name}` for the active or specified change.

#### Scenario: Branch does not exist
- **GIVEN** `git.enabled: true` and `git.branch_prefix: ""`
- **AND** no branch named `260224-vx4k-decouple-git-from-fab-switch` exists
- **WHEN** the user runs `/git-branch`
- **THEN** a new branch `260224-vx4k-decouple-git-from-fab-switch` is created from HEAD and checked out
- **AND** output shows: `Branch: 260224-vx4k-decouple-git-from-fab-switch (created)`

#### Scenario: Branch already exists
- **GIVEN** `git.enabled: true` and a branch `260224-vx4k-decouple-git-from-fab-switch` exists
- **WHEN** the user runs `/git-branch`
- **THEN** the existing branch is checked out
- **AND** output shows: `Branch: 260224-vx4k-decouple-git-from-fab-switch (checked out)`

#### Scenario: Already on the correct branch
- **GIVEN** the current branch is already `260224-vx4k-decouple-git-from-fab-switch`
- **WHEN** the user runs `/git-branch`
- **THEN** no git operation is performed
- **AND** output shows: `Branch: 260224-vx4k-decouple-git-from-fab-switch (already active)`

#### Scenario: Branch prefix applied
- **GIVEN** `git.branch_prefix: "feat/"`
- **AND** active change is `260224-vx4k-decouple-git-from-fab-switch`
- **WHEN** the user runs `/git-branch`
- **THEN** the branch name is `feat/260224-vx4k-decouple-git-from-fab-switch`

#### Scenario: Explicit change name argument
- **GIVEN** no active change in `fab/current`
- **WHEN** the user runs `/git-branch mychange`
- **THEN** the change name is resolved via `changeman.sh resolve mychange`
- **AND** the matching branch is created or checked out

### Requirement: /git-branch SHALL respect git.enabled

If `git.enabled` is `false` in `config.yaml`, `/git-branch` SHALL report the status and stop without executing any git commands.

#### Scenario: Git disabled
- **GIVEN** `config.yaml` has `git.enabled: false`
- **WHEN** the user runs `/git-branch`
- **THEN** output shows: `Git integration is disabled (git.enabled: false in config.yaml)`
- **AND** no git commands are executed

### Requirement: /git-branch SHALL present context-dependent prompts

When the user is on a branch other than the expected change branch, `/git-branch` SHALL present context-appropriate options.

#### Scenario: On main/master branch
- **GIVEN** the current branch is `main` or `master`
- **WHEN** the user runs `/git-branch`
- **THEN** the change branch is auto-created without prompting (Certain grade per SRAD: high R, A, D — creating from main is always safe)

#### Scenario: On a feature branch (not matching the change)
- **GIVEN** the current branch is `feat/other-work` (not matching the active change)
- **WHEN** the user runs `/git-branch`
- **THEN** the user is prompted with options:
  1. **Create new branch** from current HEAD
  2. **Adopt this branch** (no git operation, just acknowledge)
  3. **Skip** (cancel)

#### Scenario: On a wt/* branch
- **GIVEN** the current branch matches `wt/*`
- **WHEN** the user runs `/git-branch`
- **THEN** the user is prompted with options:
  1. **Create new branch** (default)
  2. **Adopt this branch**
  3. **Skip**

### Requirement: /git-branch SHALL NOT modify fab state

`/git-branch` SHALL NOT write to `fab/current` or modify any `.status.yaml` file. It is purely a git operation.

#### Scenario: fab/current untouched
- **GIVEN** `fab/current` contains `some-other-change`
- **WHEN** the user runs `/git-branch 260224-vx4k-decouple-git-from-fab-switch`
- **THEN** `fab/current` still contains `some-other-change`

### Requirement: /git-branch SHALL handle errors gracefully

If git operations fail, `/git-branch` SHALL report the error without modifying fab state.

#### Scenario: Checkout fails due to uncommitted changes
- **GIVEN** the working tree has uncommitted changes that conflict with the target branch
- **WHEN** the user runs `/git-branch`
- **THEN** the git error is reported to the user
- **AND** no fab state is modified

#### Scenario: Not in a git repo
- **GIVEN** the working directory is not inside a git repository
- **WHEN** the user runs `/git-branch`
- **THEN** output shows an appropriate error message
- **AND** no crash or unhandled exception

## Git PR: Branch Awareness

### Requirement: /git-pr SHOULD suggest /git-branch when on main

When `/git-pr` detects the user is on `main`/`master` and there is an active change, it SHOULD suggest running `/git-branch` before shipping, rather than auto-invoking it.

#### Scenario: On main with active change
- **GIVEN** current branch is `main` and `fab/current` points to an active change
- **WHEN** the user runs `/git-pr`
- **THEN** the existing branch guard message is enhanced:
  ```
  Cannot create PR from main/master branch.
  Tip: run /git-branch to switch to the change's branch first.
  ```
- **AND** no git operations are performed

#### Scenario: On main without active change
- **GIVEN** current branch is `main` and `fab/current` does not exist
- **WHEN** the user runs `/git-pr`
- **THEN** the standard branch guard message is shown (no /git-branch suggestion)

### Requirement: /git-pr SHOULD nudge when branch name doesn't match the active change

When `/git-pr` detects the current branch name does not match (or closely resemble) the active change's expected branch name, it SHOULD show a non-blocking nudge before proceeding. The nudge is informational — it does NOT block the PR workflow.

The match check SHALL compare the current branch (stripped of `git.branch_prefix` if present) against the active change's folder name. A match is exact string equality or the change name appearing as a substring of the branch name.

#### Scenario: Branch name mismatches active change
- **GIVEN** `fab/current` points to `260224-vx4k-decouple-git-from-fab-switch`
- **AND** `git.branch_prefix` is `""`
- **AND** current branch is `temp-work` (does not contain the change name)
- **WHEN** the user runs `/git-pr`
- **THEN** a nudge is shown before proceeding:
  ```
  Note: branch 'temp-work' doesn't match active change '260224-vx4k-decouple-git-from-fab-switch'.
  Run /git-branch to switch, or continue if this is intentional.
  ```
- **AND** the PR workflow proceeds normally (commit, push, create PR)

#### Scenario: Branch name matches with prefix
- **GIVEN** `fab/current` points to `260224-vx4k-decouple-git-from-fab-switch`
- **AND** `git.branch_prefix` is `"feat/"`
- **AND** current branch is `feat/260224-vx4k-decouple-git-from-fab-switch`
- **WHEN** the user runs `/git-pr`
- **THEN** no nudge is shown

#### Scenario: No active change
- **GIVEN** `fab/current` does not exist
- **WHEN** the user runs `/git-pr`
- **THEN** no nudge is shown (nothing to compare against)

## Pipeline: dispatch.sh Cleanup

### Requirement: dispatch.sh SHALL remove --no-branch-change from fab-switch invocation

The `tmux send-keys` call in `dispatch.sh` SHALL use `/fab-switch $CHANGE_ID` instead of `/fab-switch $CHANGE_ID --no-branch-change`.

#### Scenario: dispatch.sh sends simplified fab-switch
- **GIVEN** dispatch.sh launches a Claude session in a worktree
- **WHEN** it sends the fab-switch command via tmux send-keys
- **THEN** the command is `/fab-switch $CHANGE_ID` (no flags)
- **AND** branch creation is handled by `create_worktree()` before Claude starts, not by fab-switch

## Deprecated Requirements

### Branch Integration in /fab-switch

**Reason**: Branch operations are now handled by the standalone `/git-branch` command.
**Migration**: Users who relied on automatic branch creation during `/fab-switch` should call `/git-branch` after switching. The `/fab-switch` output hint (when `git.enabled: true`) guides this transition.

### --branch and --no-branch-change Flags

**Reason**: `--branch` is replaced by `/git-branch` with explicit invocation. `--no-branch-change` is unnecessary when `/fab-switch` performs no git operations.
**Migration**: Both flags produce hard errors immediately. `dispatch.sh` must be updated in the same change to remove `--no-branch-change` from its `tmux send-keys` call.

### --blank --branch Combination in /fab-switch

**Reason**: `--blank` deactivates the change; the branch checkout that `--branch` provided is now a separate concern. Users can deactivate with `--blank` and switch branches manually or via `/git-branch`.
**Migration**: Run `/fab-switch --blank` then `git checkout main` (or whichever branch).

## Design Decisions

1. **Command over shell script for /git-branch**: Both `/git-branch` and `/git-pr` remain commands (skills) rather than shell scripts.
   - *Why*: Commands are invocable mid-conversation in Claude chat sessions, preserving the interactive affordance. Shell scripts would require the agent to know to call them explicitly.
   - *Rejected*: Shell script in `changeman.sh branch` — loses conversational invocability.

2. **Suggest, don't auto-invoke**: `/git-pr` suggests `/git-branch` in its branch guard message rather than auto-invoking it.
   - *Why*: `/git-pr` is designed as fully autonomous ("no prompts, no questions"). Auto-invoking `/git-branch` introduces a dependency and potential interactive prompts into an autonomous flow.
   - *Rejected*: Auto-invoke — breaks the autonomous contract of `/git-pr`.

3. **Hard removal of --no-branch-change**: Both `--branch` and `--no-branch-change` produce hard errors. `dispatch.sh` is updated in the same change.
   - *Why*: Clean break avoids lingering dead code. Since dispatch.sh is updated atomically in this change, there's no version skew risk.
   - *Rejected*: Silent ignore — defers cleanup, adds dead-flag debt. Deprecation warning — half-measure that still requires a follow-up removal.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch writes only fab/current, no git ops | Confirmed from intake #1 — user explicitly stated | S:95 R:90 A:95 D:95 |
| 2 | Certain | /git-branch is a command (skill), not a shell script | Confirmed from intake #2 — user explicitly chose commands for conversational invocability | S:95 R:85 A:90 D:90 |
| 3 | Certain | /git-pr remains a command | Confirmed from intake #3 — user explicitly confirmed | S:95 R:85 A:90 D:90 |
| 4 | Certain | Skip pipeline isolation abstraction | Confirmed from intake #4 — user explicitly excluded | S:95 R:95 A:95 D:95 |
| 5 | Certain | Skip config-based created_by | Confirmed from intake #5 — user explicitly excluded | S:95 R:95 A:95 D:95 |
| 6 | Confident | Interactive branch prompts move to /git-branch | Confirmed from intake #6 — prompts are git decisions, belong in git-focused command | S:70 R:80 A:85 D:75 |
| 7 | Confident | /git-pr suggests /git-branch rather than auto-invoking | Confirmed from intake #7 — preserves git-pr's autonomous contract | S:60 R:75 A:80 D:70 |
| 8 | Confident | dispatch.sh removes --no-branch-change, no other changes | Confirmed from intake #8 — dispatch handles its own branching | S:75 R:85 A:80 D:80 |
| 9 | Certain | --no-branch-change hard error (not silent ignore) | Clarified — user chose hard error; dispatch.sh updated in same change | S:95 R:70 A:95 D:95 |
| 10 | Certain | fab-switch output includes /git-branch hint when git enabled | Clarified — user confirmed hint for discoverability | S:95 R:85 A:95 D:95 |
| 11 | Certain | /git-pr nudge on branch mismatch is non-blocking | User specified nudge behavior — git-pr proceeds after showing the note | S:90 R:90 A:85 D:90 |

11 assumptions (8 certain, 3 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-24

1. **Q**: How should fab-switch handle the now-unnecessary `--no-branch-change` flag?
   **A**: Hard error — forces immediate cleanup. dispatch.sh must be updated in the same change.

2. **Q**: Should fab-switch output include a `/git-branch` hint when `git.enabled` is true?
   **A**: Yes — show a tip line for discoverability.
