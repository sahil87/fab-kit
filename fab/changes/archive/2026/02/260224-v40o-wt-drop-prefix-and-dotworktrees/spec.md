# Spec: Drop wt/ Branch Prefix and Switch to .worktrees Directory

**Change**: 260224-v40o-wt-drop-prefix-and-dotworktrees
**Created**: 2026-02-24
**Affected memory**: `docs/memory/fab-workflow/distribution.md`

## Non-Goals

- Migration tooling for existing worktrees or branches — existing `<repo>-worktrees/` directories and `wt/*` branches continue to work; users clean up organically
- Making the branch prefix or worktree directory convention configurable — this change removes the prefix entirely and adopts the GitLens convention as the single default

## wt Package: Branch Naming

### Requirement: Exploratory Worktrees Use Unprefixed Branch Names

`wt_create_exploratory_worktree()` in `fab/.kit/packages/wt/bin/wt-create` SHALL create branches using the worktree name directly, without any prefix. The branch name MUST equal the worktree directory name.

#### Scenario: Create exploratory worktree with default name
- **GIVEN** a git repository with no existing worktree named `swift-fox`
- **WHEN** the user runs `wt-create` and accepts the suggested name `swift-fox`
- **THEN** a worktree is created at `<repo>.worktrees/swift-fox/`
- **AND** the branch is named `swift-fox` (not `wt/swift-fox`)

#### Scenario: Create exploratory worktree with user-overridden name
- **GIVEN** a git repository with no existing worktree named `my-feature`
- **WHEN** the user runs `wt-create` and overrides the suggested name with `my-feature`
- **THEN** a worktree is created at `<repo>.worktrees/my-feature/`
- **AND** the branch is named `my-feature` (not `wt/my-feature`)

#### Scenario: Non-interactive exploratory worktree
- **GIVEN** a git repository
- **WHEN** the user runs `wt-create --non-interactive`
- **THEN** a worktree is created with a random name (e.g., `calm-owl`)
- **AND** the branch is named `calm-owl` (not `wt/calm-owl`)

### Requirement: Branch-Based Worktrees Are Unchanged

`wt_create_branch_worktree()` SHALL continue to use the user-specified branch name as-is. This behavior is already correct and MUST NOT be modified.

#### Scenario: Create worktree for existing branch
- **GIVEN** a git repository with a remote branch `feature/auth`
- **WHEN** the user runs `wt-create feature/auth`
- **THEN** a worktree is created at `<repo>.worktrees/auth/`
- **AND** the branch checked out is `feature/auth` (unchanged behavior)

## wt Package: Worktree Directory Convention

### Requirement: Worktrees Directory Uses Dot-Suffix Convention

`wt_get_repo_context()` in `fab/.kit/packages/wt/lib/wt-common.sh` SHALL set `WT_WORKTREES_DIR` to `<parent>/<repo-name>.worktrees` instead of `<parent>/<repo-name>-worktrees`.

#### Scenario: Worktree directory path derivation
- **GIVEN** a git repository at `/home/user/code/my-project`
- **WHEN** `wt_get_repo_context()` runs
- **THEN** `WT_WORKTREES_DIR` is set to `/home/user/code/my-project.worktrees`
- **AND** the value `/home/user/code/my-project-worktrees` is NOT used

#### Scenario: Worktree directory created on first use
- **GIVEN** a git repository at `/home/user/code/my-project` with no `.worktrees` sibling
- **WHEN** `wt-create` runs and calls `wt_ensure_worktrees_dir()`
- **THEN** the directory `/home/user/code/my-project.worktrees/` is created

### Requirement: Help Text Reflects New Convention

The `wt_show_help()` function in `wt-create` SHALL display the `.worktrees` directory convention in all path references. This includes both the dynamic path computation (`${repo_name}-worktrees/` → `${repo_name}.worktrees/`) and the static branch description text.
<!-- clarified: help text covers both dynamic path computation and static text -->

#### Scenario: Help text shows correct worktrees path
- **GIVEN** a git repository named `my-project`
- **WHEN** the user runs `wt-create help`
- **THEN** the help output shows worktrees path as `<parent>/my-project.worktrees/`
- **AND** the branch description says `If omitted, creates <random-name> branch` (no `wt/` prefix mentioned)

## git-branch Skill: Remove wt/* Special-Casing

### Requirement: No Branch-Pattern-Based Default Selection

The `/git-branch` skill SHALL NOT use branch name patterns (such as `wt/*`) to determine the default action when presenting options to the user. All non-main, non-target branches SHALL be treated identically.

#### Scenario: On a random-name branch (formerly wt/* pattern)
- **GIVEN** the user is on branch `swift-fox` (an exploratory worktree branch)
- **AND** the active change is `260224-v40o-some-change`
- **WHEN** `/git-branch` runs
- **THEN** the skill presents the standard three options (Create new / Adopt / Skip)
- **AND** the default is "Adopt this branch" (same as any other feature branch)

#### Scenario: On main branch
- **GIVEN** the user is on the `main` branch
- **WHEN** `/git-branch` runs
- **THEN** the skill auto-creates the change branch without prompting (unchanged behavior)

## Deprecated Requirements

### wt/ Branch Prefix for Exploratory Worktrees

**Reason**: The `wt/` prefix was only applied to exploratory worktrees, not branch-based ones. This inconsistency caused confusion when users overrode the suggested name — the prefix was applied to user-chosen names, which users didn't expect. No popular worktree tool prefixes branch names.

**Migration**: N/A — existing `wt/*` branches continue to work. The prefix is simply no longer applied to new worktrees.

### `wt/*` Default Override in git-branch Skill

**Reason**: The `wt/*` pattern match in `/git-branch` Step 5 set "Create new branch" as the default for branches matching `wt/*`. With the prefix removed, this pattern no longer applies. All non-main branches now use "Adopt this branch" as the default.

**Migration**: N/A — the skill now treats all non-main branches uniformly.

## Design Decisions

1. **Dot-suffix over hyphen-suffix for worktrees directory**: `.worktrees` instead of `-worktrees`
   - *Why*: GitLens (~30M installs) uses `.worktrees` as its convention. The dot-prefix reads as possessive ("project's worktrees"), sorts adjacent to the repo in file managers, and avoids confusion with a separate project named `project-worktrees`.
   - *Rejected*: `-worktrees` (current) — could be mistaken for a separate project; no industry backing.

2. **Remove prefix entirely, not make it configurable**: No `wt.branch_prefix` config option
   - *Why*: The prefix was an implementation artifact, not a user-facing feature. No popular tool prefixes branch names. Adding configurability for a pattern nobody wants increases code complexity for zero benefit.
   - *Rejected*: Configurable prefix via `config.yaml` — adds complexity for a feature with no demand.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop `wt/` prefix entirely | Confirmed from intake #1 — user explicitly decided, no alternative interpretations | S:90 R:85 A:90 D:95 |
| 2 | Certain | Use `.worktrees` suffix (GitLens convention) | Confirmed from intake #2 — user explicitly requested matching GitLens convention | S:95 R:80 A:90 D:95 |
| 3 | Certain | Keep single-name prompt driving both directory and branch | Confirmed from intake #3 — UX flow unchanged | S:90 R:90 A:85 D:95 |
| 4 | Confident | No migration for existing worktrees | Confirmed from intake #4 — worktrees are transient, no migration friction | S:70 R:85 A:75 D:80 |
| 5 | Certain | `wt-list` should not auto-detect legacy directories | Clarified — user confirmed no legacy detection; worktrees are transient, complexity not justified | S:90 R:90 A:85 D:90 |
<!-- clarified: no legacy detection — user confirmed, worktrees are transient -->
| 6 | Certain | Remove `wt/*` default override in git-branch skill | Pattern match is an artifact of the prefix; with prefix removed, the pattern has no semantic meaning | S:85 R:90 A:90 D:90 |
| 7 | Confident | Treat all non-main branches uniformly in git-branch | "Adopt this branch" is the sensible default for any feature branch — no reason to special-case by name pattern | S:75 R:85 A:80 D:85 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-25

1. **Legacy directory detection in `wt-list`** — User confirmed: no auto-detection of `<repo>-worktrees/` alongside `<repo>.worktrees/`. Worktrees are transient; detection logic not worth the complexity. (Assumption #5 upgraded Tentative → Certain)
2. **`--worktree-name` flag coverage** — Already covered implicitly by the "user-overridden name" scenario. No spec change needed.
3. **Help text dynamic path computation** — Clarified that `wt_show_help()` has two update sites: the dynamic `${repo_name}-worktrees/` path computation AND the static "creates wt/<random-name> branch" text. Requirement updated to call out both.
