---
name: git-pr
description: "Autonomously commit, push, and create a GitHub PR — no prompts, no questions."
allowed-tools: Bash(git:*), Bash(gh:*)
---

# /git-pr

Autonomously ship local changes to a GitHub PR. No questions, no prompts — just execute.

---

## Behavior

### Step 1: Gather State

Run these commands to understand the current situation:

```bash
git branch --show-current
git status --porcelain
git log --oneline -5
git log --oneline @{u}..HEAD 2>/dev/null || echo "NO_UPSTREAM"
gh pr view --json number,state,url 2>/dev/null || echo "NO_PR"
```

Determine:
- **branch** — current branch name
- **has_uncommitted** — whether `git status --porcelain` has output
- **has_unpushed** — whether there are commits ahead of upstream (or no upstream at all)
- **has_pr** — whether a PR already exists

### Step 2: Branch Guard

If the current branch is `main` or `master`, STOP immediately:

```
Cannot create PR from main/master branch.
```

Do NOT run any git operations.

### Step 3: Execute Pipeline

Run each step in order, skipping steps that aren't needed.

**If nothing to do** (no uncommitted changes, no unpushed commits, PR exists):
```
/git-pr — already shipped

  ✓ pr — {existing PR URL}

Nothing to do.
```
Before stopping, attempt to record the existing PR URL per Step 4 (silently, no errors). Then STOP.

**Otherwise**, print the header and execute:

```
/git-pr — shipping to PR
```

#### 3a. Commit (if has_uncommitted)

1. Stage all changes: `git add -A`
2. Read `git log --oneline -5` for commit message style
3. Read `git diff --stat HEAD` for change scope
4. Generate a concise commit message matching the repo's existing style
   - Subject line only (unless changes warrant a body)
   - Do NOT include "Co-Authored-By" lines
5. Commit: `git commit -m "<message>"`
6. If commit fails → report error and STOP

Print: `  ✓ commit — "<commit subject>"`

#### 3b. Push (if has_unpushed or just committed)

1. Check if upstream exists: `git rev-parse --abbrev-ref @{u} 2>/dev/null`
2. If no upstream: `git push -u origin $(git branch --show-current)`
3. If upstream exists: `git push`
4. If push fails → report the git error output and STOP

Print: `  ✓ push   — origin/<branch>`

#### 3c. Create PR (if no PR exists)

1. Verify `gh` is available: `command -v gh`
   - If missing → print `gh CLI not found — cannot create PR` and STOP
2. Create PR: `gh pr create --fill`
3. If PR creation fails → report the error and STOP
4. Get the PR URL: `gh pr view --json url -q '.url'`

Print: `  ✓ pr     — <PR URL>`

**If PR already exists** (from Step 1), just print: `  ✓ pr     — <existing PR URL> (existing)`

### Step 4: Record Shipped

After the PR URL is known (from step 3c or from the existing PR in step 1), attempt to record it in the active change's `.status.yaml`:

1. Resolve the active change: `fab/.kit/scripts/lib/changeman.sh resolve 2>/dev/null`
2. If resolution succeeds (exit 0), derive the status file path: `fab/changes/{name}/.status.yaml`
3. Call: `fab/.kit/scripts/lib/stageman.sh ship <status_file> <pr_url>`
4. If resolution fails (exit non-zero) or `changeman.sh` is not found, skip silently — do not print any error or warning

This step MUST NOT block or fail the PR workflow. Any error from changeman or stageman is silently ignored.

### Step 5: Report

Print:
```

Shipped.
```

---

## Rules

- Fully autonomous — never ask questions, never present options
- Fail fast — if any step fails, report the error and stop immediately
- Skip steps that are already done (no uncommitted → skip commit, PR exists → skip create)
- Always operate on CWD — no repo detection
- No merge support — stop at PR creation
