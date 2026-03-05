# Intake: Smart Change Resolution & PR Summary Generation

**Change**: 260224-1jkh-smart-resolve-and-pr-summary
**Created**: 2026-02-24
**Status**: Draft

## Origin

> Two workflow improvements discussed during `/fab-discuss` session:
> - **[7k6b]** from `fab/backlog.md`: "git-pr should generate better summary for the PR. If possible reference the intake.md of the change for which we are submitting the PR."
> - **[w5ne]** from `fab/backlog.md`: "How do current commands resolve the change? They don't read fab/current directly (or at least should not). They call a changeman argument and that does it for them. We need to update this fn with 'guessing' ability of the current change (if there is only one active change in the change folder). Then upgrade git-branch to use this."
>
> Conversational mode — both items were analyzed in detail before `/fab-new`. Key decisions made during discussion:
> - w5ne should be implemented first (changeman resolve change), then 7k6b builds on reliable resolution
> - The guess should emit a stderr note for visibility
> - Folders without `.status.yaml` should be filtered out during the guess
> - PR body should include intake summary + links to intake and spec files
> - Fall back to `gh pr create --fill` when no active change exists

## Why

**Problem 1 — Blind change resolution**: When `fab/current` is missing or empty (common in fresh worktrees or after `/fab-switch --blank`), `changeman.sh resolve` fails even when there's exactly one active change in `fab/changes/`. Every skill that depends on preflight — `/fab-continue`, `/fab-ff`, `/fab-fff`, `/git-branch`, `/git-pr` — hits this wall. The user must manually run `/fab-switch` first, which is pure friction when the answer is unambiguous.

**Problem 2 — Context-free PR descriptions**: `/git-pr` step 3c runs `gh pr create --fill`, which populates the PR from commit messages. For fab-managed changes, this discards the rich context in `intake.md` and `spec.md` — the "why", scope, design decisions, and assumptions. Reviewers land on a PR with commit-message-level context when a full intake document exists. The user explicitly wants to direct reviewers to the intake file.

**If we don't fix it**: Users keep hitting "No active change" errors in worktrees and manually running `/fab-switch`. PR reviewers continue seeing commit-message-only descriptions and miss the design context that would make reviews more effective.

## What Changes

### 1. Changeman Resolve — Single-Change Guessing

Modify `cmd_resolve()` in `fab/.kit/scripts/lib/changeman.sh` (lines 55–129). When the default mode (no override) finds `fab/current` missing or empty, add a fallback:

```bash
# After existing fab/current check fails (lines 113-128):
# NEW: Guess from single active change
local changes_dir="$FAB_ROOT/changes"
local candidates=()
for d in "$changes_dir"/*/; do
  [ -d "$d" ] || continue
  local base="$(basename "$d")"
  [ "$base" = "archive" ] && continue
  # Filter: must have valid .status.yaml
  [ -f "$d/.status.yaml" ] || continue
  candidates+=("$base")
done

if [ ${#candidates[@]} -eq 1 ]; then
  echo "(resolved from single active change)" >&2
  echo "${candidates[0]}"
elif [ ${#candidates[@]} -eq 0 ]; then
  echo "No active change." >&2
  return 1
else
  echo "No active change (multiple changes exist — use /fab-switch)." >&2
  return 1
fi
```

**Behavior summary**:
1. Read `fab/current` — if valid, return it (existing, unchanged)
2. If `fab/current` missing/empty:
   - List non-archive folders in `fab/changes/` that have a `.status.yaml`
   - Exactly 1 → return it + stderr note `(resolved from single active change)`
   - 0 → `"No active change."` (existing error, unchanged)
   - 2+ → `"No active change (multiple changes exist — use /fab-switch)."` (improved error)

**Propagation**: Since `preflight.sh`, `git-branch`, and `git-pr` all call `changeman.sh resolve`, they all get the guessing behavior automatically. No changes needed in those callers.

### 2. Git-PR — Intake-Aware PR Summary

Modify `.claude/skills/git-pr/SKILL.md` step 3c. Before calling `gh pr create`, check for an active change and read its intake:

**New logic for step 3c** (replacing the current `gh pr create --fill`):

1. Attempt to resolve the active change: `fab/.kit/scripts/lib/changeman.sh resolve 2>/dev/null`
2. **If resolution succeeds** and `fab/changes/{name}/intake.md` exists:
   - Read `intake.md` — extract the `## Why` and `## What Changes` sections
   - Derive PR title from the intake's H1 heading (strip "Intake: " prefix)
   - Generate PR body with:
     - A summary paragraph derived from the `## Why` section
     - A "Changes" section derived from `## What Changes` subsection headings
     - A "Context" section with relative links to the change artifacts:
       - `[Intake](fab/changes/{name}/intake.md)` — always included
       - `[Spec](fab/changes/{name}/spec.md)` — included if the file exists
   - Create PR: `gh pr create --title "<title>" --body "<generated body>"`
3. **If resolution fails** or `intake.md` doesn't exist:
   - Fall back to current behavior: `gh pr create --fill`

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the changeman resolve guessing behavior and the git-pr intake-aware summary generation

## Impact

- **`fab/.kit/scripts/lib/changeman.sh`** — `cmd_resolve()` function, default-mode branch (lines ~112-128)
- **`.claude/skills/git-pr/SKILL.md`** — Step 3c (PR creation logic)
- **All skills using preflight** — benefit from guessing automatically (no changes needed)
- **`/git-branch`** — benefits from guessing automatically (no changes needed)
- **Worktree workflows** — primary beneficiary of the resolve guessing

## Open Questions

None — all questions were resolved during the `/fab-discuss` session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Guess only when exactly one candidate exists | Discussed — user confirmed; multiple candidates should error with guidance | S:90 R:90 A:95 D:95 |
| 2 | Certain | Filter candidates by .status.yaml existence | Discussed — user confirmed; folders without status are corrupted | S:85 R:90 A:90 D:95 |
| 3 | Certain | Emit stderr note when guessing | Discussed — user confirmed; visible but non-blocking | S:90 R:95 A:90 D:95 |
| 4 | Certain | PR body includes intake summary + links to intake and spec | Discussed — user explicitly requested directing reviewers to intake | S:95 R:85 A:90 D:90 |
| 5 | Certain | Fall back to --fill when no active change | Discussed — user confirmed; non-fab PRs keep working | S:90 R:95 A:90 D:95 |
| 6 | Confident | PR title derived from intake H1 heading | Strong signal from intake template structure; easily changed if user prefers different source | S:70 R:90 A:80 D:75 |
| 7 | Confident | Include spec link only if spec.md exists | Early-stage changes may not have a spec yet; conditional inclusion is safe | S:75 R:90 A:85 D:80 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
