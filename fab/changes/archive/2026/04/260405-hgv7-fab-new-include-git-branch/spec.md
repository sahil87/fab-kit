# Spec: fab-new Include Git Branch

**Change**: 260405-hgv7-fab-new-include-git-branch
**Created**: 2026-04-05
**Affected memory**: `docs/memory/fab-workflow/planning-skills.md`

## Non-Goals

- Modifying the standalone `/git-branch` skill — it remains unchanged and idempotent
- Removing the `git-branch` dispatch from `/fab-proceed` — that is a follow-on decision (see open question in intake)
- Changing the branch naming convention — branch name continues to equal the change folder name per `_naming.md`

## fab-new: Git Branch Step

### Requirement: Step 11 Inline Git Branch Creation

After activating the change (Step 10), `/fab-new` SHALL create or check out the matching git branch inline, following the same branching logic as the standalone `/git-branch` skill.

The step MUST run after activation — the change folder name (which becomes the branch name) must exist before any git operations.

#### Scenario: On main, fresh start
- **GIVEN** the change has been activated and current branch is `main` or `master`
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it runs `git checkout -b "{name}"` and reports `Branch: {name} (created)`

#### Scenario: On local-only branch (no upstream)
- **GIVEN** the current branch has no remote tracking (not yet pushed)
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it runs `git branch -m "{name}"` and reports `Branch: {name} (renamed from {old})`

#### Scenario: On pushed branch (has upstream)
- **GIVEN** the current branch has a remote tracking ref
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it runs `git checkout -b "{name}"` and reports `Branch: {name} (created, leaving {old} intact)`

#### Scenario: Target branch already exists but not current
- **GIVEN** `git rev-parse --verify "{name}"` succeeds and current branch differs
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it runs `git checkout "{name}"` and reports `Branch: {name} (checked out)`

#### Scenario: Already on target branch
- **GIVEN** current branch already equals the change folder name
- **WHEN** `/fab-new` reaches Step 11
- **THEN** no git operation is performed and it reports `Branch: {name} (already active)`

### Requirement: Non-Fatal Git Step

The git branch step SHALL be non-fatal. If the repo check (`git rev-parse --is-inside-work-tree`) fails, the step MUST be skipped with a warning. If a git operation fails (e.g., uncommitted conflicts), the error MUST be reported but the change SHALL remain activated.

#### Scenario: Not in a git repo
- **GIVEN** the working directory is not inside a git repository
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it warns `Not in a git repository — skipping branch creation` and continues
- **AND** the change remains activated with intake at `ready`

#### Scenario: Git operation fails
- **GIVEN** the branch creation/checkout fails (e.g., uncommitted changes blocking checkout)
- **WHEN** `/fab-new` reaches Step 11
- **THEN** it reports the git error message
- **AND** the change remains activated
- **AND** the output includes `Run /git-branch to create the branch manually`

### Requirement: Output and Frontmatter Updates

`/fab-new`'s output block MUST include a `Branch:` line after `Activated:`.

The skill frontmatter MUST be updated:
- `description`: `"Start a new change — creates the intake, activates it, and creates the git branch."`

<!-- clarified: The original requirement to add `allowed-tools: Bash(git:*)` has been dropped.
     fab-new.md has no `allowed-tools` frontmatter today, which means all Bash calls are unrestricted.
     Adding `Bash(git:*)` alone would restrict ALL Bash calls to git-only, breaking Steps 3–10
     (which run `fab change new`, `fab score`, `fab status advance`, etc.).
     The `allowed-tools` field is only used on single-purpose skills (git-branch, git-pr).
     fab-new already implicitly allows Bash(git:*) through its unrestricted Bash permission.
     Only the description update is needed. -->

The `SPEC-fab-new.md` at `docs/specs/skills/SPEC-fab-new.md` MUST be updated to reflect:
- Step 11 added to the flow (git branch creation, post activation)
- `Bash(git:*)` calls added to the Tools used table
- Updated description in the summary

#### Scenario: Successful full run
- **GIVEN** a fresh `/fab-new` invocation on `main` completes all steps
- **WHEN** the output is rendered
- **THEN** it includes both `Activated: {name}` and `Branch: {name} (created)` lines

## Project Governance: Canonical Source Clarification

### Requirement: Constitution Documents src/kit Canonicity

`fab/project/constitution.md` Additional Constraints MUST include a rule making explicit that `src/kit/` is the canonical source for all kit content and `.claude/skills/` is a gitignored deployment artifact.

The rule SHALL be placed immediately after the existing constraint about `src/kit/skills/*.md` spec updates, so the two `src/kit` rules are co-located.

#### Scenario: Developer reads constitution before editing a skill
- **GIVEN** a developer opens `fab/project/constitution.md` to understand editing conventions
- **WHEN** they read the Additional Constraints section
- **THEN** they see an explicit rule that `src/kit/` is canonical and `.claude/skills/` must never be edited directly
- **AND** the rule is adjacent to the existing `src/kit/skills/*.md` → `SPEC-*.md` update rule

### Requirement: Rule Text

The constraint text SHALL be:

```
- `src/kit/` is the canonical source for all kit content (skills, templates, migrations).
  `.claude/skills/` contains deployed copies produced by `fab sync` and is gitignored —
  never edit files there directly
```

#### Scenario: Rule is already partially covered by context.md
- **GIVEN** `fab/project/context.md` already mentions `src/kit/skills/*.md` as canonical
- **WHEN** the constitution rule is added
- **THEN** the constitution rule is more complete (covers all kit content, not just skills) and normative (MUST vs informational note)
- **AND** no change to `context.md` is needed

## Clarifications

### Session 2026-04-05 (auto)

| # | Issue | Resolution |
|---|-------|------------|
| A | `allowed-tools: Bash(git:*)` frontmatter change would restrict all Bash calls in fab-new to git-only, breaking Steps 3–10 | Dropped — fab-new has no `allowed-tools` today (unrestricted). Only description update is needed. |
| B | SPEC-fab-new.md update requirement was implicit (mandated by constitution) but not stated in spec | Made explicit: requirement added to "Output and Frontmatter Updates" section |

## Design Decisions

1. **Inline vs delegated**: git-branch logic runs inline in `/fab-new` rather than dispatching the `/git-branch` skill as a subagent.
   - *Why*: `/fab-new` already runs in the main context; spawning a subagent for 5 git commands is heavyweight. The logic is simple and self-contained.
   - *Rejected*: Dispatching `/git-branch` subagent — adds latency and complexity with no benefit for a simple, well-understood operation.

2. **Non-fatal step**: git branch creation is advisory, not blocking.
   - *Why*: The primary value of `/fab-new` is the change folder + intake + activation. A branch is a convenience. Failing the whole command because git has uncommitted changes would be surprising.
   - *Rejected*: Fatal on git failure — unnecessarily blocks users who may have legitimate reasons for the branch to fail (e.g., mid-rebase state).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Step 11 goes after activation (Step 10) | Folder name must exist before branch creation; confirmed from intake #1 | S:95 R:90 A:95 D:95 |
| 2 | Certain | Branch name = change folder name, no prefix | `_naming.md` is explicit; confirmed from intake #2 | S:95 R:90 A:95 D:95 |
| 3 | Certain | Step is non-fatal | User confirmed in conversation; confirmed from intake #3 | S:90 R:85 A:90 D:95 |
| 4 | Certain | `src/kit/` canonical; `.claude/skills/` gitignored | User stated explicitly twice; confirmed from intake #4 | S:100 R:95 A:100 D:100 |
| 5 | Certain | Constitution note goes after existing `src/kit/skills` bullet | Co-location; confirmed from intake #5 | S:90 R:95 A:95 D:90 |
| 6 | Certain | Git logic is inline, not a subagent dispatch | Simple 5-command operation; subagent dispatch would be heavyweight | S:90 R:85 A:90 D:90 |
| 7 | Confident | `fab-proceed` git-branch dispatch remains for now; follow-up to decide redundancy | Open question in intake — not in scope of this change | S:70 R:75 A:70 D:65 |
| 8 | Certain | `SPEC-fab-new.md` update is in scope | Constitution mandates skill spec updates; file exists at `docs/specs/skills/SPEC-fab-new.md`; clarified — requirement now explicit in spec | S:95 R:90 A:95 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
