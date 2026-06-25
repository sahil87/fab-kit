---
name: git-pr
description: "Autonomously commit, push, and create a draft GitHub PR — no prompts, no questions."
allowed-tools: Bash(git:*), Bash(gh:*)
---

# /git-pr [<change>] [<type>]

> Branch naming conventions are defined in `_preamble.md` § Naming Conventions.

Autonomously ship local changes to a GitHub PR. No questions, no prompts — just execute.

**Arguments** (both optional, in any order — classified by value): an argument matching one of the 7 valid PR types (case-insensitive) is the `<type>` argument (Step 0b); any other argument is the `<change>` argument — an explicit change to target instead of the active one (Step 0). Callers SHOULD pass the change **folder name** (or a distinctive substring), not a bare 4-char id: an id that happens to spell a type word (`feat`, `docs`, `test`) would be classified as a type.

---

## Contents

- Behavior
- Rules
- PR Type Reference
- Key Properties

---

## Behavior

### Step 0: Resolve Change Context

Resolve the change **once** and derive four variables used throughout this skill. Later steps reference these variables and MUST NOT re-run `fab change resolve` — reuse this single resolution to avoid inconsistency.

1. Resolve the change (`fab change resolve` accepts a 4-char ID, folder substring, or full folder name — see `_cli-fab.md` § fab change):
   - **Explicit `<change>` argument provided** (per the Arguments classification above — any argument that is NOT one of the 7 valid PR types) → run `fab change resolve <change> 2>/dev/null` (transient override — `.fab-status.yaml` is untouched). **Succeeds** → `{has_fab} = true`, `{name}` = resolved change name. **Fails** → STOP with `Cannot resolve change '<change>'.` — a named target that doesn't resolve is a caller error; do NOT fall back to the active change.
   - **No `<change>` argument** → run `fab change resolve 2>/dev/null` (the active change). **Succeeds** → `{has_fab} = true`, `{name}` = resolved change name. **Fails** → `{has_fab} = false`; every step gated on `{has_fab}` is skipped silently.
2. `{has_intake}` — whether `fab/changes/{name}/intake.md` exists *(only when `{has_fab}`)*.
3. `{change_type}` — the `change_type` value from `fab/changes/{name}/.status.yaml` *(only when `{has_fab}`; may be null)*.
4. **Branch-matches-change guard** *(only when `{has_fab}`)* — run `git branch --show-current`. If the output is **empty** (detached HEAD), STOP immediately with Step 2's detached-HEAD message (`Cannot ship from a detached HEAD — check out a branch first (run /git-branch).`) — before Step 0a's status mutation (verify-before-mutate; Step 2's own guard still covers the `{has_fab} = false` path, where this guard never runs). Otherwise the current branch MUST match `{name}`: exact string equality, or `{name}` appearing as a substring of the branch. On mismatch, STOP **before any status mutation, commit, or push** (Step 0a has not run yet) — do NOT check out another branch autonomously:

   ```
   Branch '{current_branch}' does not match change '{name}'.
   Run /git-branch to switch to the change's branch, /fab-switch to change the active change,
   or pass the intended change explicitly: /git-pr <change>.
   ```

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

1. **Explicit argument**: If the invocation includes an argument that is one of the 7 valid types (case-insensitive), normalize to lowercase and use it (non-type arguments are the `<change>` argument, consumed by Step 0 — see Arguments above); else fall through to step 2.

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
[ -n "$default_branch" ] || default_branch=$(git rev-parse --verify -q refs/remotes/origin/main >/dev/null && echo main || echo master)
```

If `{has_fab}` (Step 0), read issues via `fab status get-issues {name}` and capture the output (one ID per line, may be empty).

Determine:
- **branch** — current branch name; **empty** = detached HEAD (handled by the Step 2 guard before any commit or push)
- **has_uncommitted** — `git status --porcelain` has output
- **has_unpushed** — commits ahead of upstream, or no upstream at all
- **pr_state** — `state` from `gh pr view` (`OPEN`, `CLOSED`, `MERGED`), or `none` when no PR exists. Step 3 branches on this explicitly — a CLOSED or MERGED PR is NOT treated as "the branch already has a PR"
- **number** / **url** — from `gh pr view` (unset when no PR exists); interpolated by Step 3's MERGED STOP and the "already shipped" output
- **default_branch** — resolved by the commands above (always non-empty), so every later `{default_branch}` interpolation is meaningful
- **issues** — issue IDs from `fab status get-issues` (space-joined), or empty

### Step 2: Branch Guard

**Detached-HEAD guard** (checked first): if `branch` from Step 1 is **empty** (detached HEAD — `git symbolic-ref -q HEAD` exits 1), STOP immediately, before any commit or push:

```
Cannot ship from a detached HEAD — check out a branch first (run /git-branch).
```

**Default-branch guard**: if the current branch is `{default_branch}` (or literal `main`/`master` — whichever literal Step 1's probe did not pick stays a safety net), STOP immediately.

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

#### 3a-bis. Refresh Memory Indexes (if {has_fab} AND 3a just committed)

Gated on BOTH conditions — skip the entire sub-step otherwise:

- `{has_fab}` (Step 0) is true, AND
- step 3a **just committed this invocation** — i.e. the `has_uncommitted` path in 3a ran and produced a commit. It is NOT reached on the "already shipped" / no-changes re-run paths (where 3a did not commit).

When both hold, regenerate the memory indexes and conditionally commit them in a **separate** follow-up commit:

```bash
fab memory-index
if ! git diff --quiet -- docs/memory; then
  git add docs/memory
  git commit -m "docs: refresh memory indexes"
fi
```

1. Run `fab memory-index` (byte-stable; writes only `docs/memory/` index + log files; a no-op when nothing drifted).
2. If `docs/memory/` changed (`git diff --quiet -- docs/memory` exits non-zero): `git add docs/memory`, then a **SEPARATE** follow-up commit `git commit -m "docs: refresh memory indexes"`. Do **NOT** use `--amend` — keep 3a's authored content commit intact; squash collapses the pair on merge anyway.
3. If `git diff --quiet -- docs/memory` exits 0 (nothing drifted): make **no** commit — the guard suppresses an empty follow-up commit (Constitution III idempotency).
4. If the regen OR the commit fails → report the error and STOP. The 3a content commit is already made and intact; a failed refresh leaves a benign stale-date index, recoverable by re-running `fab memory-index` — never a torn state.

Print (ONLY when a follow-up commit was actually made): `  ✓ commit — "docs: refresh memory indexes"`

> **Why here, why gated.** This is the first moment `git log` reports the change's real commit date (`fab memory-index` stamps "Last Updated" from committed dates), so the step lives in **ship** not hydrate — hydrate is entirely pre-commit, so no in-hydrate regen can see the change's own commit. There is no push here; 3b pushes both commits together. When `/git-pr` runs standalone (`{has_fab}` false) this sub-step is a **silent no-op**.

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

   **Render the `## Meta` block** (only when `{has_fab}`): delegate the entire Meta block (table + optional Impact + optional `**Issues**` + `**Pipeline:**`, in that element order) to `fab pr-meta`, which reads `.status.yaml`, parses `plan.md` checkboxes, reads `fab/project/config.yaml` (`true_impact_exclude`, `test_paths`, `project.linear_workspace`), computes the impact math, resolves git/`gh` context (branch, owner/repo, merge-base), and stamps the running binary version, itself. The Impact section renders as a single normalized table whose first-column header is `Impact` (self-labeling — no lead-in line), columns `Impact | +/− | Net`, with the locked `raw / true / impl / tests / excluded` taxonomy (`true` is always the post-exclude diff) plus a `<sub>` provenance caption co-locating the excludes note and the `generated by fab-kit vX.Y.Z` stamp. Pass the Step 0 `{name}`, the resolved `{type}` (from Step 0b), and the space-joined `{issues}` (from Step 1):

   ```bash
   META=$(fab pr-meta "{name}" --type {type} --issues "{issues}" 2>/dev/null) || META=""
   ```

   - If exit 0 and `META` is non-empty: the `## Meta` block is `$META` **verbatim** — do not reformat, re-wrap, or re-derive any of it.
   - If exit non-zero or `META` is empty (no fab context, change unresolved, or `.status.yaml` absent): omit the `## Meta` block entirely, exactly as the legacy `{has_fab} = false` path did.

   `fab pr-meta` degrades gracefully on its own: an unreachable `gh` falls back to plain-text Pipeline labels, and a missing/failed merge-base or a `+0/−0` `true` diff drops only the Impact block — none of these break the block or the PR.

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
   - If body generation failed for any reason → create with `gh pr create --draft --fill` instead (silent fallback; evaluated before the creation attempt, so a body failure never reaches the STOP below)
   - If PR creation itself fails → report the error and STOP
5. Get the PR URL: `gh pr view --json url -q '.url'`

Print: `  ✓ pr     — <PR URL>`

**If an OPEN PR already exists** (from Step 1), just print: `  ✓ pr     — <existing PR URL> (existing)`

### Step 4a: Record PR URL

After the PR URL is known (from step 3c or from the existing PR in step 1), attempt to record it in the resolved change's `.status.yaml` (`{name}` from Step 0 — the active change or the explicit override):

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
| Idempotent? | Yes — re-run after ship is a no-op. The "already shipped" path (no uncommitted changes, no unpushed commits, an OPEN PR exists) re-records the existing PR URL silently and stops; `fab status add-pr` is idempotent and the Step 4c status commit is guarded by `git diff --cached --quiet`. Sub-step 3a-bis is gated on 3a having just committed this invocation, so a re-run skips it; even if reached it is byte-stable with the `git diff --quiet -- docs/memory` guard suppressing an empty commit (see 3a-bis, 4c guards) |
| Advances stage? | Yes — ship (start/finish, best-effort) |
| Modifies `.fab-status.yaml`? | No |
| Modifies git state? | Yes — commit, push, PR creation |
