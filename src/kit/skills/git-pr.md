---
name: git-pr
description: "Autonomously commit, push, and create a GitHub PR — no prompts, no questions."
allowed-tools: Bash(git:*), Bash(gh:*)
---

# /git-pr

> Branch naming conventions are defined in `_preamble.md` § Naming Conventions.

Autonomously ship local changes to a GitHub PR. No questions, no prompts — just execute.

---

## Behavior

### Step 0a: Start Ship Stage

If an active change resolves (`fab change resolve 2>/dev/null`) and `progress.ship` is not `done`, attempt to start the `ship` stage:

```bash
fab status start <change> ship git-pr 2>/dev/null || true
```

This is best-effort — failures are silently ignored. If the stage is already `active`, the call is a no-op. If no active change, skip entirely.

### Step 0b: Resolve PR Type

Determine the PR type before gathering state. The type controls the PR title prefix and body template.

**Valid types**: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`

**Resolution chain** (evaluated in order, first match wins):

1. **Explicit argument**: If the user invoked `/git-pr {type}` where `{type}` is one of the 7 valid types (case-insensitive), normalize to lowercase and use it. If the argument is not a valid type, ignore it and fall through to step 2.

2. **Read from `.status.yaml`**: Run `fab change resolve 2>/dev/null`. If resolution succeeds, read `change_type` from `fab/changes/{name}/.status.yaml`. If non-null and one of the 7 valid types (`feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`), use it. Fall through if resolution fails, `change_type` is null, or `change_type` is not a valid type.

3. **Infer from fab change intake**: If `fab change resolve` succeeded (from step 2) and `fab/changes/{name}/intake.md` exists, read the intake content and pattern-match (case-insensitive). Keyword lists are evaluated in order — first match wins:
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
```

If an active change is resolved (via `fab change resolve`), read issues via `fab status get-issues <change>` and capture the output (one ID per line, may be empty).

Determine:
- **branch** — current branch name
- **has_uncommitted** — whether `git status --porcelain` has output
- **has_unpushed** — whether there are commits ahead of upstream (or no upstream at all)
- **has_pr** — whether a PR already exists
- **issues** — the issue IDs from `fab status get-issues` (space-joined), or empty if none

### Step 1b: Branch Mismatch Nudge

If there is an active change (resolve via `fab change resolve 2>/dev/null`), compare the current branch against the change name.

A match is: (1) exact string equality between current branch and change name, or (2) the change name appears as a substring of the current branch.

If there is **no match** and the current branch is **not** `main`/`master`, show a non-blocking nudge before proceeding:

```
Note: branch '{current_branch}' doesn't match active change '{change_name}'.
Run /git-branch to switch, or continue if this is intentional.
```

Then proceed to Step 2 normally. If resolution fails or there is no active change, skip this step silently.

### Step 2: Branch Guard

If the current branch is `main` or `master`, STOP immediately.

If there is an active change (from Step 1b), enhance the message:

```
Cannot create PR from main/master branch.
Tip: run /git-branch to switch to the change's branch first.
```

If there is no active change:

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
Before stopping, attempt to record the existing PR URL per Steps 4a–4c (silently, no errors). Then STOP.

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

2. **Derive PR title**: Compute `{pr_title}` where:
   - If `fab change resolve 2>/dev/null` succeeds AND `fab/changes/{name}/intake.md` exists: `{title}` = first `# ` heading from `fab/changes/{name}/intake.md`, stripping `Intake: ` prefix if present
   - Otherwise: `{title}` = commit message subject line from `git log -1 --format=%s`

   If `issues` (from Step 1) is non-empty: `{pr_title}` = `{type}: {issues} {title}` (e.g., `feat: DEV-123 DEV-456 Add OAuth support`), where `{issues}` is space-joined.
   If `issues` is empty: `{pr_title}` = `{type}: {title}`.

   The `{pr_title}` variable (already prefixed) is used as-is in step 4's `gh pr create --title`.

3. **Generate PR body** using a single unified template with conditional field population based on artifact availability:

   **Resolve fab context** (attempt once, used for all conditional fields):
   - Run `fab change resolve 2>/dev/null`. If it succeeds, set `{has_fab} = true` and `{name}` = resolved change name
   - Check if `fab/changes/{name}/intake.md` exists → `{has_intake}`
   - Check if `fab/changes/{name}/spec.md` exists → `{has_spec}`
   - Check if `fab/changes/{name}/plan.md` exists → `{has_plan}`
   - Read `fab/changes/{name}/.status.yaml` for `id`, `name`, `confidence`, `plan`, `progress`, and `stage_metrics` fields
   - Read `fab/project/config.yaml` for the optional `linear_workspace` field under `project:`

   **Construct blob URLs** (only when `{has_fab}`):
   - `{owner_repo}` = `gh repo view --json nameWithOwner -q '.nameWithOwner'`
   - `{branch}` = `git branch --show-current`
   - If `{has_intake}`: Intake URL = `https://github.com/{owner_repo}/blob/{branch}/fab/changes/{name}/intake.md`
   - If `{has_spec}`: Spec URL = `https://github.com/{owner_repo}/blob/{branch}/fab/changes/{name}/spec.md`

   **Generate body sections**:

   ```
   ## Summary
   {if has_fab AND has_intake: 1-3 sentences derived from intake's ## Why section}
   {otherwise: 1-3 sentences auto-generated from commit messages or git diff --stat}

   ## Changes
   {if has_fab AND has_intake: bulleted list of subsection headings from intake's ## What Changes section}
   {otherwise: omit this section entirely}

   ## Change
   {if has_fab: render the Change table below}
   {otherwise: omit this section entirely}
   | ID | Name | Issue |
   |----|------|-------|
   | {id} | {status_name} | {issue_display} |

   ## Stats
   | Type | Confidence | Tasks | Acceptance | Review |
   |------|-----------|-------|-----------|--------|
   | {type} | {confidence} | {tasks} | {acceptance} | {review} |
   ```

   **Change column population** (only when `{has_fab}`):
   - **ID**: From `.status.yaml` `id` field (4-char change ID). Show `—` if unavailable
   - **Name**: From `.status.yaml` `name` field (full change folder name). Use a distinct variable (e.g., `{status_name}`) to avoid clobbering `{name}` (the resolved change folder used for path construction). Show `—` if unavailable
   - **Issue**: From `issues` resolved in Step 1 (`fab status get-issues`). If `linear_workspace` is configured in `fab/project/config.yaml`, render each issue as `[{ID}](https://linear.app/{linear_workspace}/issue/{ID})`. If `linear_workspace` is absent, render bare issue IDs. Multiple issues are comma-separated. Show `—` if no issues

   **Stats column population**:
   - **Type**: Always populated from the resolved PR type
   - **Confidence**: `{confidence.score} / 5.0` from `.status.yaml`. Show `—` if no fab change or confidence data absent
   - **Tasks**: Parse `plan.md` `## Tasks` section for checkbox counts (`- [x]` vs `- [ ]`), formatted as `{done}/{total}`. Append ` ✓` when `done == total` AND `total > 0`. Show `—` if `plan.md` doesn't exist or has no `## Tasks` heading
   - **Acceptance**: `{plan.acceptance_completed}/{plan.acceptance_count}` from `.status.yaml`. Append ` ✓` when `completed == count` AND `count > 0`. Show `—` if not available
   - **Review**: Derive from `.status.yaml` `progress.review` state and `stage_metrics.review.iterations`. Show `Pass ({N} iterations)` if review is `done`, `Fail ({N} iterations)` if review is `failed`, `—` if review not yet reached. If `iterations` is not populated, omit the parenthetical

   **Pipeline progress line** (only when `{has_fab}`):

   Below the Stats table, show a pipeline progress line. Stages with `done` status from `.status.yaml`'s `progress` map are listed in fixed order: intake, spec, apply, review, hydrate, ship, review-pr — joined with ` → `.

   - If `{has_intake}`: "intake" is a hyperlink → `[intake]({intake_url})`
   - If `{has_spec}`: "spec" is a hyperlink → `[spec]({spec_url})`
   - All other stage names are plain text

   If no fab change exists (`{has_fab}` is false), the pipeline line is omitted entirely.

4. **Append true-impact block** (only when `{has_fab}` is true): compute the true-impact line counts and append a two-line metadata block to the assembled PR body.

   1. Read `true_impact_exclude` from `fab/project/config.yaml` via `yq`:
      ```bash
      readarray -t EXCLUDES < <(yq '.true_impact_exclude[]' fab/project/config.yaml 2>/dev/null)
      ```
      If `yq` errors, the key is missing, the value is `null`, or `EXCLUDES` is empty → skip this sub-step entirely (no block, no extra git invocation).

   2. Compute the merge-base against the default branch:
      ```bash
      BASE=$(git merge-base origin/main HEAD)
      ```
      `/git-pr` does not compute a merge-base elsewhere, so this is the canonical site for it. If `origin/main` is not present, fall back to `origin/master`; if neither resolves, skip the block silently.

   3. Run two `git diff --shortstat` invocations against `$BASE`:
      ```bash
      # True-impact pass (with pathspec exclusions)
      EXCLUDE_ARGS=()
      for pat in "${EXCLUDES[@]}"; do EXCLUDE_ARGS+=( ":(exclude)$pat" ); done
      IMPACT_RAW=$(git diff --shortstat "$BASE...HEAD" -- . "${EXCLUDE_ARGS[@]}")

      # Total pass (no exclusions)
      TOTAL_RAW=$(git diff --shortstat "$BASE...HEAD")
      ```
      Both invocations use the **three-dot range** `$BASE...HEAD` for "changes on this branch" semantics.

   4. Parse each `--shortstat` line for `(\d+) insertions?\(\+\)` and `(\d+) deletions?\(-\)`. Missing clauses default to `0` (e.g., `2 files changed, 50 insertions(+)` with no deletions clause yields `D=0`).

   5. If the true-impact pass yields `+0 / −0` (i.e., every modified file lies inside an excluded path), omit the entire block — do not render `+0 / −0`.

   6. Otherwise, build `{COMMA_LIST}` by joining the `EXCLUDES` array with `, ` (literal comma + space) and append the rendered block (see format below) to the PR body **after** the pipeline progress line, separated by a single blank line. Do NOT wrap the block in a `## Impact` heading.

   **Rendered block format**:

   ```markdown

   **Impact (excluding {COMMA_LIST})**: +A / −D
   **git diff total**: +A_total / −D_total
   ```

   The leading blank line is the single-blank-line separator from the preceding pipeline progress line. `{COMMA_LIST}` reflects the actual config values verbatim (e.g., `fab/, docs/` for the default scaffold) — never hardcoded.

   **Fallbacks** — the impact block is omitted entirely (PR body byte-for-byte identical to today's output) when any of the following holds:
   - (a) `true_impact_exclude` is absent, `null`, or an empty list in `fab/project/config.yaml` (matches spec assumption #11).
   - (b) `{has_fab}` is false (no `fab/project/config.yaml` resolved — matches spec's graceful-degradation requirement).
   - (c) The true-impact pass yields zero changes (`+0 / −0`) outside the exclusions — matches spec assumption #12 to avoid a misleading zero-impact line.

   Print: `  ✓ impact — +A / −D (excluding {COMMA_LIST})` after the block is appended (skipped when the block is omitted).

5. Create PR: `gh pr create --draft --title "{pr_title}" --body "<body>"` (where `{pr_title}` is the already-prefixed title from step 2)
   - If PR creation fails → report the error and STOP
   - Fall back to `gh pr create --draft --fill` if body generation fails for any reason (silent fallback)
6. Get the PR URL: `gh pr view --json url -q '.url'`

Print: `  ✓ pr     — <PR URL>`

**If PR already exists** (from Step 1), just print: `  ✓ pr     — <existing PR URL> (existing)`

### Step 4a: Record PR URL

After the PR URL is known (from step 3c or from the existing PR in step 1), attempt to record it in the active change's `.status.yaml`:

1. Resolve the active change: `fab change resolve 2>/dev/null`
2. If resolution succeeds (exit 0), call: `fab status add-pr <name> <pr_url>`
3. If resolution fails (exit non-zero), skip silently — do not print any error or warning

This step MUST NOT block or fail the PR workflow. Any error is silently ignored.

### Step 4b: Finish Ship Stage

If an active change was resolved in Step 0a and `progress.ship` was started (not already `done`):

```bash
fab status finish <change> ship git-pr 2>/dev/null || true
```

This marks `ship` as `done` and auto-activates `review-pr`. Best-effort — failures silently ignored.

### Step 4c: Commit and Push Status Update

If Step 4a successfully recorded a PR URL (changeman resolved and statusman add-pr ran):

1. Stage the status and history files: `git add fab/changes/{name}/.status.yaml fab/changes/{name}/.history.jsonl`
2. Check for changes: `git diff --cached --quiet`
3. If changes exist: commit (`git commit -m "Update ship status and record PR URL"`) and push (`git push`). If commit or push fails → report the error and STOP.
4. If no changes (already committed): skip commit+push silently

Print (if committed): `  ✓ status — committed and pushed status updates (.status.yaml, .history.jsonl)`

If Step 4a was skipped (no active change, changeman not found), skip this step silently.

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
