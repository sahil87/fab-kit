---
name: git-rebase
description: "Fetch the latest main and rebase the current branch onto it."
allowed-tools: Bash(git:*), AskUserQuestion
---

# /git-rebase

Fetch the latest main branch and rebase the current branch onto it. Guards against running on main and prompts the user if uncommitted changes are present.

---

## Arguments

None.

---

## Behavior

### Step 1: Check Git Repo

Verify inside a git repository:

```bash
git rev-parse --is-inside-work-tree >/dev/null 2>&1
```

If not in a git repo:

```
Not inside a git repository.
```

STOP.

### Step 2: Branch Guard

Get the current branch:

```bash
git branch --show-current
```

If the current branch is `main` or `master`, STOP immediately:

```
Cannot rebase — already on {branch}. Switch to a feature branch first.
```

### Step 3: Check for Uncommitted Changes

```bash
git status --porcelain
```

If there is output (uncommitted changes exist), show the user what's pending and ask for confirmation:

```
You have uncommitted changes:

{git status --porcelain output}

Options:
1. Stash changes, rebase, then restore — runs git stash, rebases, then git stash pop
2. Abort — cancel the rebase so you can commit or handle changes first

How would you like to proceed?
```

Wait for user response. If the user chooses to abort, STOP. If the user chooses to stash, proceed with stash workflow (see Step 4).

### Step 4: Fetch and Rebase

Determine the upstream default branch name:

```bash
ref=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null) && echo "${ref#refs/remotes/origin/}"
```

If that fails (e.g., `origin/HEAD` not set), fall back to remote ref detection, then local:

```bash
git rev-parse --verify refs/remotes/origin/main >/dev/null 2>&1 && echo main || \
git rev-parse --verify refs/remotes/origin/master >/dev/null 2>&1 && echo master || \
git rev-parse --verify main >/dev/null 2>&1 && echo main || echo master
```

**If stashing** (user chose option 1 in Step 3):

```bash
git stash push -m "git-rebase: auto-stash before rebase"
```

Fetch the latest main:

```bash
git fetch origin {main_branch}
```

If fetch fails, report the error. If stashed, run `git stash pop` before stopping. STOP.

Rebase onto the fetched main:

```bash
git rebase origin/{main_branch}
```

If rebase fails (conflicts), report the conflict and advise the user:

```
Rebase conflict detected. Resolve conflicts, then:
  git rebase --continue

Or abort with:
  git rebase --abort
```

If stashed, note that stashed changes will need to be restored after conflict resolution (`git stash pop`). STOP.

**If stashed and rebase succeeded**, restore stashed changes:

```bash
git stash pop
```

If `git stash pop` produces conflicts, report them and STOP.

### Step 5: Report

```
Rebased {current_branch} onto origin/{main_branch}.
```

If stashed changes were restored:

```
Rebased {current_branch} onto origin/{main_branch}. Stashed changes restored.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Not in a git repo | Report and stop |
| On main/master | Report and stop |
| Uncommitted changes | Ask user to stash or abort |
| Fetch fails | Report error (pop stash if stashed), stop |
| Rebase conflicts | Report conflict, advise on resolution (note stash if applicable), stop |
| Stash pop conflicts | Report conflict, stop |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Idempotent? | Yes — rebasing an already up-to-date branch is a no-op |
| Modifies `.fab-status.yaml`? | No |
| Modifies `.status.yaml`? | No |
| Modifies git state? | Yes — fetches, rebases, may stash/unstash |
| Requires config/constitution? | No |
