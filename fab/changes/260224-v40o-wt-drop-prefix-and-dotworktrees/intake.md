# Intake: Drop wt/ Branch Prefix and Switch to .worktrees Directory

**Change**: 260224-v40o-wt-drop-prefix-and-dotworktrees
**Created**: 2026-02-24
**Status**: Draft

## Origin

> Remove the wt/ prefix from branches created by wt-create and switch the worktree home directory from `<repo>-worktrees` to `<repo>.worktrees` (matching GitLens convention).

Conversational `/fab-discuss` session. Researched how Conductor (conductor.build), GitLens, tree-me, wtp, git-worktree-runner, and newt handle worktree naming. Key finding: no popular tool prefixes branch names — the worktree directory structure itself provides the namespace. GitLens uses `<repo>.worktrees/` as the sibling directory convention (~30M installs, de facto standard).

User identified a design flaw: the `wt/` prefix is only applied to exploratory worktrees (random name), not branch-based worktrees. If a user overrides the suggested name with something meaningful, the prefix still gets applied, which is unwanted. The prefix is inconsistent by design and should be removed entirely.

## Why

1. **Inconsistency**: Exploratory worktrees get `wt/<name>` branches, but branch-based worktrees (`wt-create feat/auth`) use the branch as-is. The prefix only applies in one code path, making behavior unpredictable.

2. **User override conflict**: When a user overrides the suggested random name with a meaningful name (e.g., `my-feature`), the branch becomes `wt/my-feature` — the prefix is noise the user didn't ask for and doesn't want.

3. **Industry convention**: Conductor uses city names as worktree identifiers with no branch prefix. GitLens, tree-me, wtp, and git-worktree-runner all use branch names as-is. No popular tool adds a tool-specific prefix to branch names.

4. **Directory convention mismatch**: `<repo>-worktrees` could be mistaken for a separate project. `<repo>.worktrees` (GitLens convention) reads as possessive, sorts adjacent to the repo in file managers, and matches the most widely-used Git extension.

## What Changes

### 1. Remove `wt/` branch prefix

In `fab/.kit/packages/wt/bin/wt-create`, the `wt_create_exploratory_worktree()` function (line 60):

```bash
# Before
local branch="wt/$name"

# After
local branch="$name"
```

The branch name becomes identical to the worktree directory name. For a worktree named `swift-fox`:
- Directory: `fab-kit.worktrees/swift-fox/`
- Branch: `swift-fox` (was `wt/swift-fox`)

### 2. Switch worktree home to `.worktrees` convention

In `fab/.kit/packages/wt/lib/wt-common.sh`, the `wt_get_repo_context()` function (line 305):

```bash
# Before
WT_WORKTREES_DIR="$(dirname "$WT_REPO_ROOT")/${WT_REPO_NAME}-worktrees"

# After
WT_WORKTREES_DIR="$(dirname "$WT_REPO_ROOT")/${WT_REPO_NAME}.worktrees"
```

### 3. Remove `wt/*` special-casing in git-branch skill

The git-branch skill currently treats `wt/*` branches as a special case (defaulting to "create new branch"). With the prefix removed, this pattern match no longer applies and should be removed.

### 4. Update help text and docs

- `wt-create` help text (line 89): `If omitted, creates wt/<random-name> branch` → `If omitted, creates <random-name> branch`
- `wt-create` help text (line 73-80): update worktrees path display to use `.worktrees`
- `docs/specs/packages.md`: update directory convention description and examples

### 5. Update tests

- `src/packages/wt/tests/wt-create.bats`: update assertions that expect `wt/` prefix or `-worktrees` directory

### UX flow (unchanged)

The interactive prompt stays identical:
```
$ wt-create
Worktree name [swift-fox]: _
```

User accepts or overrides. The name drives both directory and branch — just without the prefix.

## Affected Memory

- `fab-workflow/distribution`: (modify) Update wt package documentation for new directory convention and branch naming

## Impact

- **wt package**: `wt-create`, `wt-common.sh` (core changes)
- **git-branch skill**: Remove `wt/*` pattern matching
- **Existing worktrees**: Users with existing `<repo>-worktrees/` directories and `wt/*` branches are unaffected (existing worktrees continue to work). New worktrees use the new convention. No migration needed.
- **Specs**: `docs/specs/packages.md` needs updates to reflect new conventions
- **Tests**: `src/packages/wt/tests/wt-create.bats` needs assertion updates

## Open Questions

- Should `wt-list` display a note if it detects legacy `-worktrees` directory alongside the new `.worktrees` directory?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop `wt/` prefix entirely, not make it configurable | Discussed — user confirmed prefix is unwanted in all cases because it conflicts with user-provided names | S:90 R:85 A:90 D:95 |
| 2 | Certain | Use `.worktrees` suffix (GitLens convention) | Discussed — user explicitly requested matching GitLens convention | S:95 R:80 A:90 D:95 |
| 3 | Certain | Keep single-name prompt driving both directory and branch | Discussed — user confirmed UX flow should not change | S:90 R:90 A:85 D:95 |
| 4 | Confident | No migration for existing worktrees | Existing worktrees are transient by nature — users create and destroy them frequently. Asking them to rename is unnecessary friction | S:70 R:85 A:75 D:80 |
| 5 | Tentative | `wt-list` should not auto-detect legacy directories | Legacy detection adds complexity for a transient concern — users will naturally move to new convention | S:50 R:90 A:60 D:60 |
<!-- assumed: no legacy detection — worktrees are transient, low blast radius if users have to manually clean up -->

5 assumptions (3 certain, 1 confident, 1 tentative, 0 unresolved).
