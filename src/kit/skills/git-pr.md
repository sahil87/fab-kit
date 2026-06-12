---
name: git-pr
description: "Autonomously commit, push, and create a draft GitHub PR — no prompts, no questions."
allowed-tools: Bash(git:*), Bash(gh:*)
---

# /git-pr

> Branch naming conventions are defined in `_preamble.md` § Naming Conventions.

Autonomously ship local changes to a GitHub PR. No questions, no prompts — just execute.

---

## Behavior

### Step 0: Resolve Change Context

Resolve the active change **once** and derive four variables used throughout this skill. Later steps reference these variables and MUST NOT re-run `fab change resolve` — reuse this single resolution to avoid inconsistency.

1. Run `fab change resolve 2>/dev/null`:
   - **Succeeds** → `{has_fab} = true`, `{name}` = resolved change name.
   - **Fails** → `{has_fab} = false`; every step gated on `{has_fab}` is skipped silently.
2. `{has_intake}` — whether `fab/changes/{name}/intake.md` exists *(only when `{has_fab}`)*.
3. `{change_type}` — the `change_type` value from `fab/changes/{name}/.status.yaml` *(only when `{has_fab}`; may be null)*.

### Step 0a: Start Ship Stage

If `{has_fab}` and `progress.ship` is not `done`, attempt to start the `ship` stage:

```bash
fab status start {name} ship git-pr 2>/dev/null || true
```

This is best-effort — failures are silently ignored. If the stage is already `active`, the call is a no-op. If `{has_fab}` is false, skip entirely.

### Step 0b: Resolve PR Type

Determine the PR type before gathering state. The type controls the PR title prefix and body template.

**Valid types**: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`

**Resolution chain** (evaluated in order, first match wins):

1. **Explicit argument**: If the user invoked `/git-pr {type}` where `{type}` is one of the 7 valid types (case-insensitive), normalize to lowercase and use it. If the argument is not a valid type, ignore it and fall through to step 2.

2. **Read from `.status.yaml`**: If `{has_fab}` (Step 0) and `{change_type}` is non-null and one of the 7 valid types (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`), use `{change_type}`. Fall through if `{has_fab}` is false, `{change_type}` is null, or `{change_type}` is not a valid type.

3. **Infer from fab change intake**: If `{has_fab}` AND `{has_intake}` (Step 0), read the intake content at `fab/changes/{name}/intake.md` and pattern-match (case-insensitive). Keyword lists are evaluated in order — first match wins:
   - Contains any of: "fix", "bug", "broken", "regression" → type = `fix`
   - Contains any of: "refactor", "restructure", "consolidate", "split", "rename" → type = `refactor`
   - Otherwise → type = `feat`

4. **Infer from diff**: Collect changed file paths by running each command and taking the first non-empty result: (a) `git diff --name-only HEAD`, (b) `git diff --name-only --cached`, (c) `git diff --name-only @{u}..HEAD` (only if upstream exists). This covers uncommitted, staged, and committed-but-unpushed changes.

   If **no files** are returned (empty diff — clean working tree and no unpushed commits), default to `chore`.

   Otherwise, analyze the changed file paths:
   - All files in `.github/` or CI config files → type = `ci`
   - All files in `docs/` or non-code `.md` files → type = `docs`
   - All files in test directories or test files → type = `test`
   - Otherwise → type = `chore`

Store the resolved `type` (always lowercase) and the resolution source (`explicit`, `status`, `intake`, `diff`) for use in Step 3c.

This step MUST NOT ask questions or present options. If resolution is ambiguous, default to `chore`.

### Step 1: Gather State

Run these commands to understand the current situation:

```bash
git branch --show-current
git status --porcelain
git log --oneline -5
git log --oneline @{u}..HEAD 2>/dev/null || echo "NO_UPSTREAM"
gh pr view --json number,state,url 2>/dev/null || echo "NO_PR"
default_branch=$(git symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
[ -n "$default_branch" ] || default_branch=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name 2>/dev/null)
```

If `{has_fab}` (Step 0), read issues via `fab status get-issues {name}` and capture the output (one ID per line, may be empty).

Determine:
- **branch** — current branch name. An **empty** value means a detached HEAD (`git symbolic-ref -q HEAD` exits 1) — handled by the Step 2 guard before any commit or push
- **has_uncommitted** — whether `git status --porcelain` has output
- **has_unpushed** — whether there are commits ahead of upstream (or no upstream at all)
- **pr_state** — the `state` field from `gh pr view` (`OPEN`, `CLOSED`, or `MERGED`), or `none` when no PR exists. Step 3 branches on this explicitly — a CLOSED or MERGED PR is NOT treated as "the branch already has a PR"
- **number** / **url** — the `number` and `url` fields from `gh pr view` (unset when no PR exists). Interpolated by Step 3's MERGED STOP and the "already shipped" output
- **default_branch** — the resolved default branch from the commands above (symbolic-ref first, `gh repo view` fallback). If both fail (empty), fall back to treating literal `main`/`master` as the default
- **issues** — the issue IDs from `fab status get-issues` (space-joined), or empty if none

### Step 1b: Branch Mismatch Nudge

If `branch` from Step 1 is **empty** (detached HEAD), skip this step silently — Step 2's detached-HEAD guard STOPs anyway, and a nudge comparing an empty branch name would be noise.

If `{has_fab}` (Step 0), compare the current branch against `{name}`.

A match is: (1) exact string equality between current branch and change name, or (2) the change name appears as a substring of the current branch.

If there is **no match** and the current branch is **not** the default branch (`{default_branch}`, falling back to literal `main`/`master` — Step 1), show a non-blocking nudge before proceeding:

```
Note: branch '{current_branch}' doesn't match active change '{change_name}'.
Run /git-branch to switch, or continue if this is intentional.
```

Then proceed to Step 2 normally. If `{has_fab}` is false, skip this step silently.

### Step 2: Branch Guard

**Detached-HEAD guard** (checked first): if `branch` from Step 1 is **empty** (detached HEAD — `git symbolic-ref -q HEAD` exits 1), STOP immediately, before any commit or push:

```
Cannot ship from a detached HEAD — check out a branch first (run /git-branch).
```

**Default-branch guard**: if the current branch is `{default_branch}` (or literal `main`/`master` when Step 1's resolution failed), STOP immediately.

If `{has_fab}` (Step 0), enhance the message:

```
Cannot create PR from the default branch ({default_branch}).
Tip: run /git-branch to switch to the change's branch first.
```

If `{has_fab}` is false:

```
Cannot create PR from the default branch ({default_branch}).
```

Do NOT run any git operations.

### Step 3: Execute Pipeline

Branch on `pr_state` (Step 1) before doing anything else:

**If the branch's PR is MERGED** (`pr_state` = `MERGED`): STOP immediately — do NOT commit or push:

```
PR #{number} for this branch is already merged — {url}
New work needs a new change/branch. Run /fab-new to start one (or /git-branch <name> for a standalone branch).
```

**If the branch's PR is CLOSED** (`pr_state` = `CLOSED`): proceed — a closed PR does not block creation. Step 3c creates a fresh PR (`gh pr create` works after a closed PR; shipping intent is explicit — `/git-pr` was just invoked).

If the MERGED STOP did not fire, run each step in order, skipping steps that aren't needed.

**If nothing to do** (no uncommitted changes, no unpushed commits, an **OPEN** PR exists):
```
/git-pr — already shipped

  ✓ pr — {existing PR URL}

Nothing to do.
```
Before stopping, attempt to record the existing PR URL per Steps 4a–4c (silently, no errors). Then STOP.

**Otherwise**, print the header and execute:

```
/git-pr — shipping to PR
```

#### 3a. Commit (if has_uncommitted)

1. **Expected-area guard for untracked files** (evaluated FIRST, before anything is staged — the STOP path must leave no staged index): list untracked files (`git status --porcelain` lines starting `??`) and derive the expected write areas — each `source_paths` entry from `fab/project/config.yaml`, plus `docs/` and `fab/` (when `config.yaml` is absent, `docs/` and `fab/` only). If any untracked file falls **outside** the expected areas → STOP with the list (nothing staged, nothing committed):

   ```
   Unexpected untracked file(s) outside the expected write areas ({areas}):
     {file list}
   Stage, remove, or .gitignore them, then re-run /git-pr.
   ```
2. Stage tracked changes: `git add -u` (NOT `git add -A` — an autonomous run must never sweep unrelated untracked files into a pushed commit)
3. Stage the untracked files **inside** the expected areas (enumerated by the guard in step 1): `git add <file> ...`
4. Read `git log --oneline -5` for commit message style
5. Read `git diff --stat HEAD` for change scope
6. Generate a concise commit message matching the repo's existing style
   - Subject line only (unless changes warrant a body)
   - Do NOT include "Co-Authored-By" lines
7. Commit: `git commit -m "<message>"`
8. If commit fails → report error and STOP

Print: `  ✓ commit — "<commit subject>"`

#### 3b. Push (if has_unpushed or just committed)

1. Check if upstream exists: `git rev-parse --abbrev-ref @{u} 2>/dev/null`
2. If no upstream: `git push -u origin $(git branch --show-current)`
3. If upstream exists: `git push`
4. If push fails → report the git error output and STOP

Print: `  ✓ push   — origin/<branch>`

#### 3c. Create PR (if no OPEN PR exists — `pr_state` is `none` or `CLOSED`)

1. Verify `gh` is available: `command -v gh`
   - If missing → print `gh CLI not found — cannot create PR` and STOP

2. **Derive PR title**: Compute `{pr_title}` where:
   - If `{has_fab}` AND `{has_intake}` (Step 0): `{title}` = first `# ` heading from `fab/changes/{name}/intake.md`, stripping `Intake: ` prefix if present
   - Otherwise: `{title}` = commit message subject line from `git log -1 --format=%s`

   If `issues` (from Step 1) is non-empty: `{pr_title}` = `{type}: {issues} {title}` (e.g., `feat: DEV-123 DEV-456 Add OAuth support`), where `{issues}` is space-joined.
   If `issues` is empty: `{pr_title}` = `{type}: {title}`.

   The `{pr_title}` variable (already prefixed) is used as-is in step 4's `gh pr create --title`.

3. **Generate PR body**: the `## Meta` block is rendered mechanically by `fab pr-meta`; `## Summary` and `## Changes` stay agent-generated (they require prose synthesis from the intake).

   **Fab context** comes from Step 0: `{has_fab}`, `{name}`, `{has_intake}` (controls Summary/Changes sourcing below). Do NOT re-run `fab change resolve` — reuse the Step 0 resolution.

   **Render the `## Meta` block** (only when `{has_fab}`): delegate the entire Meta block (table + `**Pipeline**` + optional `**Issues**` + optional `**Impact**`) to `fab pr-meta`, which reads `.status.yaml`, parses `plan.md` checkboxes, reads `fab/project/config.yaml` (`true_impact_exclude`, `test_paths`, `project.linear_workspace`), computes the impact math, and resolves git/`gh` context (branch, owner/repo, merge-base) itself. Pass the Step 0 `{name}`, the resolved `{type}` (from Step 0b), and the space-joined `{issues}` (from Step 1):

   ```bash
   META=$(fab pr-meta "{name}" --type {type} --issues "{issues}" 2>/dev/null) || META=""
   ```

   - If exit 0 and `META` is non-empty: the `## Meta` block is `$META` **verbatim** — do not reformat, re-wrap, or re-derive any of it.
   - If exit non-zero or `META` is empty (no fab context, change unresolved, or `.status.yaml` absent): omit the `## Meta` block entirely, exactly as the legacy `{has_fab} = false` path did.

   `fab pr-meta` degrades gracefully on its own: an unreachable `gh` falls back to plain-text Pipeline labels, and a missing/failed merge-base or `+0/−0` total drops only the `**Impact**` line — none of these break the block or the PR.

   **Assemble the body** in this exact order:

   ```
   {META}

   ## Summary

   {summary_text}

   ## Changes

   {changes_bullets}
   ```

   When `{has_fab}` is false (or `$META` is empty), the body becomes just `## Summary` + `## Changes` (or just `## Summary` if no intake exists).

   **Summary text**: 1–3 sentences. Source:
   - If `{has_fab}` AND `{has_intake}`: derive from intake's `## Why` section.
   - Otherwise: auto-generate from commit messages or `git diff --stat`.

   **Changes bullets**: Bulleted list. Source:
   - If `{has_fab}` AND `{has_intake}`: subsection headings from intake's `## What Changes` section.
   - Otherwise: omit the `## Changes` section entirely.

   Print after body assembly: `  ✓ body  — meta + summary + changes` (skip the "meta" token when `$META` was empty/omitted).

4. Create PR: `gh pr create --draft --title "{pr_title}" --body "<body>"` (where `{pr_title}` is the already-prefixed title from step 2; `<body>` is the assembled body from step 3 including the Meta block when `{has_fab}`)
   - If PR creation fails → report the error and STOP
   - Fall back to `gh pr create --draft --fill` if body generation fails for any reason (silent fallback)
5. Get the PR URL: `gh pr view --json url -q '.url'`

Print: `  ✓ pr     — <PR URL>`

**If an OPEN PR already exists** (from Step 1), just print: `  ✓ pr     — <existing PR URL> (existing)`

### Step 4a: Record PR URL

After the PR URL is known (from step 3c or from the existing PR in step 1), attempt to record it in the active change's `.status.yaml`:

1. If `{has_fab}` (Step 0), call: `fab status add-pr {name} <pr_url>`
2. If `{has_fab}` is false, skip silently — do not print any error or warning

This step MUST NOT block or fail the PR workflow. Any error is silently ignored.

### Step 4b: Finish Ship Stage

If `{has_fab}` (Step 0) and `progress.ship` was started in Step 0a (not already `done`):

```bash
fab status finish {name} ship git-pr 2>/dev/null || true
```

This marks `ship` as `done` and auto-activates `review-pr`. Best-effort — failures silently ignored.

### Step 4c: Commit and Push Status Update

If Step 4a successfully recorded a PR URL (`{has_fab}` is true and `fab status add-pr` ran):

1. Stage the status and history files: `git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl`
2. Check for changes: `git diff --cached --quiet`
3. If changes exist: commit (`git commit -m "Update ship status and record PR URL"`) and push (`git push`). If commit or push fails → report the error and STOP.
4. If no changes (already committed): skip commit+push silently

Print (if committed): `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)`

If Step 4a was skipped (`{has_fab}` is false — no active change), skip this step silently.

### Step 5: Report

Print:
```

Shipped.
```

---

## Rules

- Fully autonomous — never ask questions, never present options
- Fail fast — if any step fails, report the error and stop immediately
- Skip steps that are already done (no uncommitted → skip commit, OPEN PR exists → skip create)
- Always operate on CWD — no repo detection
- No merge support — stop at PR creation

---

## PR Type Reference

| Type | Description |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `refactor` | Restructure without behavior change |
| `docs` | Documentation-only changes |
| `test` | Adding/fixing tests only |
| `ci` | CI/CD and build system changes |
| `chore` | Maintenance, cleanup, housekeeping |

Derived from [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), consolidated: `style` → `refactor`, `perf` → `feat`/`refactor`, `build` → `ci`.

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Yes — re-run after ship is a no-op: the "already shipped" path (no uncommitted changes, no unpushed commits, an OPEN PR exists) re-records the existing PR URL silently and stops; `fab status add-pr` is idempotent and the Step 4c status commit is guarded by `git diff --cached --quiet` |
| Advances stage? | Yes — ship (start/finish, best-effort) |
| Modifies `.fab-status.yaml`? | No |
| Modifies git state? | Yes — commit, push, PR creation |
