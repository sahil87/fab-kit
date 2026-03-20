# Spec: Draft PRs by Default

**Change**: 260320-tm9h-draft-prs-by-default
**Created**: 2026-03-20
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Ship: PR Creation Mode

### Requirement: Draft PR Creation

The `/git-pr` skill SHALL create all GitHub PRs as drafts by passing the `--draft` flag to `gh pr create`. This is unconditional — there is no configuration toggle or override mechanism.

#### Scenario: New PR Created via /git-pr

- **GIVEN** a branch with unpushed commits and no existing PR
- **WHEN** `/git-pr` executes Step 3c (Create PR)
- **THEN** the `gh pr create` invocation includes the `--draft` flag
- **AND** the resulting PR is in draft state on GitHub

#### Scenario: Existing PR Already Exists

- **GIVEN** a branch that already has an open PR (draft or ready)
- **WHEN** `/git-pr` runs
- **THEN** no new PR is created (existing behavior unchanged)
- **AND** the existing PR's draft/ready state is not modified

#### Scenario: Fallback PR Creation

- **GIVEN** PR body generation fails for any reason
- **WHEN** `/git-pr` falls back to `gh pr create --fill`
- **THEN** the fallback command also includes the `--draft` flag

### Requirement: Spec File Update

The `docs/specs/skills/SPEC-git-pr.md` flow diagram SHALL reflect that `gh pr create` uses `--draft`. Specifically, the Step 3c line showing `Bash: gh pr create --title --body` SHALL include `--draft`.

#### Scenario: Spec File Accuracy

- **GIVEN** the SPEC-git-pr.md flow diagram documents Step 3c
- **WHEN** this change is applied
- **THEN** the PR creation line reads `Bash: gh pr create --draft --title --body`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `gh pr create --draft` flag | Confirmed from intake #1 — gh CLI natively supports `--draft` | S:90 R:95 A:95 D:95 |
| 2 | Certain | Change applies to `/git-pr` skill only | Confirmed from intake #2 — only skill invoking `gh pr create` | S:85 R:90 A:95 D:95 |
| 3 | Certain | No configuration toggle | Confirmed from intake #3 — user stated "always create draft PRs" | S:90 R:85 A:90 D:95 |
| 4 | Confident | Mark [m1ef] done when this ships | Confirmed from intake #4 — duplicate backlog item | S:75 R:90 A:70 D:80 |
| 5 | Certain | Fallback `gh pr create --fill` also needs `--draft` | Skill has a silent fallback path — must be consistent | S:80 R:85 A:90 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
