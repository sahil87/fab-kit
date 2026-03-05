# Spec: Absorb Ship Command

**Change**: 260222-n811-absorb-ship-command
**Created**: 2026-02-22
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Merge support — out of scope per user direction; the pipeline never merges
- Repo detection — fab-kit skills always run from CWD, which is the project root
- Interactive prompts or target selection — the skill always goes commit → push → PR
- Draft vs. ready PR distinction — always creates a ready PR via `--fill`

## Skill: `/git-pr`

### Requirement: Skill File

The skill SHALL be defined as `fab/.kit/skills/git-pr.md` with YAML frontmatter containing `name: git-pr`, `description`, and `model_tier: fast`. The sync mechanism SHALL symlink it to `.claude/skills/git-pr/SKILL.md` per the existing convention.

The skill frontmatter SHALL include `allowed-tools` restricting tool access to `Bash(git:*)`, `Bash(gh:*)`.

#### Scenario: Skill Discovery

- **GIVEN** `fab/.kit/skills/git-pr.md` exists with valid frontmatter
- **WHEN** `fab-sync.sh` runs
- **THEN** `.claude/skills/git-pr/SKILL.md` is created as a symlink to `../../../fab/.kit/skills/git-pr.md`
- **AND** `/git-pr` appears in `fab-help.sh` output

### Requirement: Autonomous Execution

The skill SHALL execute the full commit → push → PR pipeline without any user interaction. It MUST NOT ask questions, present options, or wait for confirmation. Every decision SHALL be made autonomously.

#### Scenario: Clean Pipeline Run

- **GIVEN** the CWD is a git repo on a feature branch with uncommitted changes
- **WHEN** `/git-pr` is invoked
- **THEN** all changes are staged via `git add -A`
- **AND** a commit is created with a message derived from the diff and existing commit style
- **AND** the branch is pushed to origin (with `-u` if no upstream is set)
- **AND** a PR is created via `gh pr create --fill`
- **AND** the PR URL is reported to the user

#### Scenario: No Uncommitted Changes

- **GIVEN** the CWD has no uncommitted changes but has unpushed commits
- **WHEN** `/git-pr` is invoked
- **THEN** the commit step is skipped
- **AND** the branch is pushed and PR created normally

#### Scenario: PR Already Exists

- **GIVEN** a PR already exists for the current branch
- **WHEN** `/git-pr` is invoked
- **THEN** any uncommitted changes are committed and pushed
- **AND** the existing PR URL is reported (no new PR created)

#### Scenario: Already Fully Shipped

- **GIVEN** no uncommitted changes, no unpushed commits, and a PR already exists
- **WHEN** `/git-pr` is invoked
- **THEN** the skill reports "Already shipped" with the PR URL
- **AND** no git operations are performed

### Requirement: Branch Guard

The skill SHALL refuse to run on `main` or `master` branches. PRs are only meaningful from feature branches.

#### Scenario: Invoked on Main

- **GIVEN** the current branch is `main` or `master`
- **WHEN** `/git-pr` is invoked
- **THEN** the skill reports an error: "Cannot create PR from main/master branch"
- **AND** no git operations are performed

### Requirement: Commit Message Generation

The skill SHALL generate commit messages by analyzing the diff and matching the existing commit style. The commit message SHOULD be concise (subject line + optional body). The skill MUST NOT include "Co-Authored-By" lines.
<!-- assumed: Claude generates commit messages from diff context — fast-tier model (haiku) is sufficient for this level of reasoning -->

#### Scenario: Commit Message Style Matching

- **GIVEN** the repo has existing commits with a consistent style (e.g., `type: description`)
- **WHEN** the skill generates a commit message
- **THEN** the message follows the same style pattern
- **AND** the subject line summarizes the change scope from `git diff --stat`

### Requirement: Error Handling

The skill SHALL fail fast on errors. Each step (commit, push, PR creation) SHALL check the exit code and report the error immediately without retrying.

#### Scenario: Push Rejected

- **GIVEN** the push is rejected (e.g., diverged history)
- **WHEN** `git push` fails
- **THEN** the skill reports the git error output
- **AND** does not attempt PR creation

#### Scenario: gh CLI Missing

- **GIVEN** `gh` is not available on PATH
- **WHEN** `/git-pr` is invoked
- **THEN** the skill reports "gh CLI not found" and stops

### Requirement: Progress Output

The skill SHALL output a pipeline-style progress indicator showing each step's status.

#### Scenario: Full Pipeline Output

- **GIVEN** all three steps need to execute
- **WHEN** `/git-pr` completes successfully
- **THEN** output resembles:
  ```
  /git-pr — shipping to PR

    ✓ commit — "fix: resolve auth callback race condition"
    ✓ push   — origin/260222-n811-absorb-ship-command
    ✓ pr     — https://github.com/user/repo/pull/42

  Shipped.
  ```

## Pipeline Integration

### Requirement: Replace Ship Command

The pipeline orchestrator (`fab/.kit/scripts/pipeline/run.sh`) SHALL replace the `/changes:ship pr` invocation with `/git-pr` in the tmux send-keys call. The ship delay, PR polling, and ship timeout mechanics SHALL remain unchanged.

#### Scenario: Pipeline Ships via /git-pr

- **GIVEN** the pipeline detects `hydrate:done` for a dispatched change
- **WHEN** the ship sequence begins
- **THEN** `run.sh` sends `/git-pr` (not `/changes:ship pr`) via `tmux send-keys`
- **AND** the existing `sleep 0.5` + Enter pattern is preserved
- **AND** PR detection via `gh pr view` continues to work as before

#### Scenario: Ship Log Message

- **GIVEN** the pipeline is about to send the ship command
- **WHEN** the log message is emitted
- **THEN** it reads `Sending /git-pr` (not `Sending /changes:ship pr`)

## Design Decisions

1. **Skill (markdown prompt) rather than shell script**
   - *Why*: The pipeline sends commands to an interactive Claude session via tmux. Claude needs a skill definition to interpret `/git-pr`. A shell script would require a different invocation mechanism.
   - *Rejected*: Shell script called directly by pipeline — would require changing the pipeline's tmux-based architecture, which the user explicitly wants preserved.

2. **`fast` model tier (haiku)**
   - *Why*: The skill's reasoning load is lightweight — commit message generation from diffs and git log style matching. No complex architectural decisions. `fast` tier minimizes token cost in the pipeline (many changes shipped per session).
   - *Rejected*: `capable` tier — overkill for deterministic git operations with a thin layer of commit-message reasoning.

3. **`gh pr create --fill` for PR creation**
   - *Why*: `--fill` auto-populates title and body from the commit history. This is sufficient for pipeline-generated PRs where the commit messages already describe the change. No need for AI-generated PR descriptions.
   - *Rejected*: AI-generated PR body from spec/tasks — adds latency and complexity for marginal improvement in automated PRs.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Name is `/git-pr` | User explicitly specified. Confirmed from intake #1 | S:100 R:90 A:100 D:100 |
| 2 | Certain | Pipeline still uses tmux send-keys | User explicitly specified. Confirmed from intake #2 | S:100 R:80 A:100 D:100 |
| 3 | Certain | No merge support | User scoped to "get to github PR". Confirmed from intake #3 | S:90 R:90 A:90 D:95 |
| 4 | Certain | Fully autonomous, no prompts | User specified "make all decisions". Confirmed from intake #4 | S:100 R:85 A:100 D:100 |
| 5 | Certain | Skill file at `fab/.kit/skills/git-pr.md` | Follows existing skill convention in kit-architecture memory | S:80 R:90 A:95 D:95 |
| 6 | Confident | `gh pr create --fill` for PR creation | Standard gh CLI; `--fill` suits autonomous operation. Confirmed from intake #5 | S:70 R:85 A:80 D:75 |
| 7 | Confident | Commit message from diff context, not template | AI generates from diff + git log style. Upgraded from intake #6 — spec-level analysis confirms haiku sufficient | S:65 R:85 A:80 D:75 |
| 8 | Confident | `fast` model tier (haiku) | Lightweight reasoning: commit messages + git commands. No complex decisions. Pipeline cost savings | S:60 R:85 A:75 D:70 |
| 9 | Confident | Branch guard: refuse on main/master | Pipeline always uses feature branches; guard prevents misuse in interactive context. No intake precedent — new at spec level | S:65 R:90 A:80 D:80 |
| 10 | Confident | `git add -A` for staging | Pipeline context: fab-ff generates all code, no need for selective staging. Risk: could stage unintended files, but in worktree isolation this is safe | S:70 R:80 A:75 D:75 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
