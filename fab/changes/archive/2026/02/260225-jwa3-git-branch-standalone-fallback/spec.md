# Spec: git-branch Standalone Fallback

**Change**: 260225-jwa3-git-branch-standalone-fallback
**Created**: 2026-02-25
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/change-lifecycle.md`

## Non-Goals

- Modifying `changeman.sh` or any shell scripts — the fallback lives entirely in the skill markdown
- Creating fab artifacts (change folder, intake, status) for standalone branches
- Supporting standalone fallback when no argument is provided (omitted argument always resolves via `fab/current`)

## git-branch: Standalone Branch Fallback

### Requirement: Fallback to literal branch name on resolution failure

When an explicit `<change-name>` argument is provided and `changeman.sh resolve` fails to match any existing change, `/git-branch` SHALL use the raw argument as a literal branch name instead of stopping with an error. Change resolution MUST take precedence — the fallback SHALL only activate when resolution fails.

#### Scenario: Argument matches a fab change

- **GIVEN** a change folder `260225-jwa3-git-branch-standalone-fallback` exists in `fab/changes/`
- **WHEN** the user runs `/git-branch jwa3`
- **THEN** `changeman.sh resolve` succeeds and the resolved change name is used
- **AND** behavior is identical to today (no fallback path)

#### Scenario: Argument matches no fab change

- **GIVEN** no change folder matches the argument `fix-typo`
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the skill prints `No matching change found — creating standalone branch 'fix-typo'`
- **AND** the raw argument `fix-typo` is used as the branch name
- **AND** the flow proceeds to Step 4 (derive branch name) with the literal value

#### Scenario: No argument provided

- **GIVEN** `fab/current` points to a valid change
- **WHEN** the user runs `/git-branch` with no argument
- **THEN** the skill resolves from `fab/current` as today
- **AND** no fallback path is attempted (resolution failure stops with changeman's error)

### Requirement: Standalone branches skip branch prefix

When the standalone fallback is active, the `git.branch_prefix` from `config.yaml` SHALL NOT be applied. The literal argument SHALL be used exactly as provided — no prefix, no slug transformation, no casing changes.

#### Scenario: Config has branch prefix, standalone fallback active

- **GIVEN** `git.branch_prefix` is `feat/` in config.yaml
- **AND** no change matches `hotfix-login`
- **WHEN** the user runs `/git-branch hotfix-login`
- **THEN** the branch name is `hotfix-login` (not `feat/hotfix-login`)

#### Scenario: Config has empty branch prefix, standalone fallback active

- **GIVEN** `git.branch_prefix` is `""` in config.yaml
- **AND** no change matches `fix-typo`
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the branch name is `fix-typo`

### Requirement: Existing standalone branch switches instead of failing

When the standalone fallback produces a branch name that already exists locally, the skill SHALL switch to it (`git checkout`) rather than attempting `git checkout -b` (which would fail).

#### Scenario: Standalone branch already exists

- **GIVEN** a local branch `fix-typo` already exists
- **AND** no change matches `fix-typo`
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the skill runs `git checkout fix-typo`
- **AND** reports `Branch: fix-typo (checked out)`

#### Scenario: Already on the standalone branch

- **GIVEN** the user is currently on branch `fix-typo`
- **AND** no change matches `fix-typo`
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the skill reports `Branch: fix-typo (already active)`
- **AND** no git operation is performed

### Requirement: Context-dependent action for standalone branches

The standalone fallback SHALL use the same Step 5 context-dependent action as change-resolved branches: auto-create when on `main`/`master`, prompt when on another branch, no-op when already on target.

#### Scenario: On main, creating standalone branch

- **GIVEN** the user is on `main`
- **AND** no change matches `fix-typo`
- **AND** no local branch `fix-typo` exists
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the skill runs `git checkout -b fix-typo`
- **AND** reports `Branch: fix-typo (created)`

#### Scenario: On another branch, creating standalone branch

- **GIVEN** the user is on branch `feature-x`
- **AND** no change matches `fix-typo`
- **WHEN** the user runs `/git-branch fix-typo`
- **THEN** the skill presents options: create new branch, adopt current branch, or skip

### Requirement: Clear feedback distinguishing standalone from change-based

When the standalone fallback activates, the skill SHALL print a message before proceeding that clearly indicates this is not a change-resolved branch. The message SHALL be: `No matching change found — using standalone branch '{name}'`.

#### Scenario: Feedback message displayed

- **GIVEN** no change matches `quick-patch`
- **WHEN** the user runs `/git-branch quick-patch`
- **THEN** the output includes `No matching change found — creating standalone branch 'quick-patch'` before the branch action output

## Design Decisions

### Fallback in Skill, Not in changeman.sh

**Decision**: The standalone fallback is implemented as control flow in the skill markdown, not as a new mode in `changeman.sh`.

**Why**: `changeman.sh resolve` has a clear contract: resolve to an existing change or fail. Adding a "pass-through" mode would blur that contract. The skill is the right place for this decision because it's the skill that knows the user's intent (create a branch) and can decide how to handle resolution failure.

**Rejected**: Adding a `--fallback-literal` flag to `changeman.sh` — couples a UI concern (standalone branch creation) to a data-layer utility (change resolution).

### No Prefix for Standalone Branches

**Decision**: `git.branch_prefix` is not applied to standalone branches.

**Why**: The prefix exists to namespace fab-managed branches (e.g., `feat/260225-jwa3-...`). Standalone branches are explicitly outside fab conventions — applying the prefix would be inconsistent and potentially confusing. "Literal means literal."

**Rejected**: Apply prefix for consistency — creates false impression that the branch is fab-managed.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change resolution takes precedence over literal fallback | Confirmed from intake #1 — user explicitly decided | S:95 R:90 A:95 D:95 |
| 2 | Certain | Literal branch name used as-is, no transformation | Confirmed from intake #2 — user explicitly said "literal, as it is" | S:95 R:85 A:90 D:95 |
| 3 | Certain | Standalone branches skip `git.branch_prefix` | Confirmed from intake #3 — follows from literal naming decision | S:90 R:85 A:85 D:90 |
| 4 | Certain | Existing branch → switch instead of fail | Confirmed from intake #4 — consistent with existing change-branch behavior | S:85 R:90 A:85 D:90 |
| 5 | Certain | No fab artifacts created for standalone branches | Confirmed from intake #5 — user confirmed "intentionally outside" | S:95 R:90 A:90 D:95 |
| 6 | Confident | Feedback message distinguishes standalone from change-based | Upgraded from intake #6 — now a spec requirement with defined message text | S:80 R:90 A:85 D:85 |
| 7 | Confident | Fallback only when explicit argument provided | Confirmed from intake #7 — omitted argument means "use active change" | S:75 R:85 A:85 D:90 |
| 8 | Certain | Affected memory includes change-lifecycle.md (documents /git-branch behavior) | Codebase signal — change-lifecycle.md has the canonical `/git-branch` section | S:90 R:85 A:90 D:90 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
