# Intake: git-branch Standalone Fallback

**Change**: 260225-jwa3-git-branch-standalone-fallback
**Created**: 2026-02-25
**Status**: Draft

## Origin

> Backlog item [jwa3]: "git-pr should be able to create a branch, given that I pass an argument to it, even though the argument doesn't match an existing change"

Corrected during discussion to target `/git-branch`, not `/git-pr`. The user described a specific workflow: making changes directly on `main`, then needing to move them onto a branch for PR submission. The current `/git-branch` skill requires its argument to resolve to an existing fab change via `changeman.sh resolve` — if nothing matches, it fails. The user wants a fallback that creates a standalone branch with the literal argument as the branch name, intentionally outside the fab pipeline.

Key decisions from conversation:
- Branch name is used **as-is** — no `{YYMMDD}-{XXXX}-` prefix, no slug transformation
- Change resolution takes precedence: try `changeman.sh resolve` first, fall back to literal only on failure
- Dirty working tree is the expected state (changes already made on main)
- Intentionally outside fab — no change folder, no intake, no spec artifacts created
- If the literal branch name already exists, switch to it instead of failing

## Why

When working on small fixes or exploratory changes, users often start editing directly on `main` without going through the full fab pipeline. Once the changes are ready, they need a branch to submit a PR. Currently, `/git-branch` only works with fab changes — there's no way to create an ad-hoc branch through the skill. The user must drop to raw `git checkout -b` manually, which breaks the workflow of staying within fab/skill commands.

Without this change, the `/git-branch` skill has a usability gap for the common "hack first, formalize later" workflow.

## What Changes

### Modified: `fab/.kit/skills/git-branch.md`

The skill's Step 3 (Resolve Change Name) gains a fallback path:

1. If `<change-name>` is provided, attempt `changeman.sh resolve "<change-name>"` as today
2. If resolution **succeeds**: proceed with the resolved change name (existing behavior, unchanged)
3. If resolution **fails**: use the raw `<change-name>` argument as a **literal branch name** and proceed to Step 4
   - Print a clear message: `"No matching change found — creating standalone branch '{name}'"`
   - Skip `branch_prefix` — the literal name is used exactly as provided
   - The rest of the flow (Step 5: context-dependent action) works the same: create if on main, prompt if on another branch, no-op if already on target

No changes to `changeman.sh` or any other scripts. The fallback is entirely within the skill's markdown behavior.

### Behavior when argument is omitted

No change. When no argument is given, the skill resolves from `fab/current` as today. The fallback only applies when an explicit argument is provided and fails to resolve.

### Branch prefix handling

The `git.branch_prefix` from config is **not applied** to standalone branches. The rationale: standalone branches are intentionally outside fab conventions, so applying fab's prefix would be inconsistent. The literal name means literal.

### Existing branch handling

If the literal branch name matches an existing local branch, the skill switches to it (`git checkout`) rather than failing on `git checkout -b`. This mirrors the existing behavior for change-resolved branches.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Document the standalone fallback behavior in the git-branch section

## Impact

- **`fab/.kit/skills/git-branch.md`** — sole file modified; adds ~10-15 lines to the skill
- No script changes, no template changes, no config schema changes
- Backwards compatible: existing `/git-branch` invocations with change names work identically

## Open Questions

None — all key decisions were resolved during pre-intake discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change resolution takes precedence over literal fallback | Discussed — user confirmed: try resolve first, fall back only on failure | S:95 R:90 A:95 D:95 |
| 2 | Certain | Literal branch name used as-is, no prefix or transformation | Discussed — user explicitly said "literal, as it is" | S:95 R:85 A:90 D:95 |
| 3 | Certain | Standalone branches skip `git.branch_prefix` | Follows from decision #2 — literal means no prefix applied | S:90 R:85 A:85 D:90 |
| 4 | Certain | Existing branch → switch instead of fail | Discussed — user agreed; consistent with existing change-branch behavior | S:85 R:90 A:85 D:90 |
| 5 | Certain | No fab artifacts created for standalone branches | Discussed — user confirmed "intentionally outside" the fab pipeline | S:95 R:90 A:90 D:95 |
| 6 | Confident | Feedback message distinguishes standalone from change-based | Not explicitly discussed but strongly implied by the need for clarity; easily changed | S:70 R:90 A:80 D:85 |
| 7 | Confident | Fallback only when explicit argument provided, not when resolving from fab/current | Logical consequence — omitted argument means "use active change", no fallback makes sense | S:75 R:85 A:85 D:90 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
