# Intake: Smart git-pr Category Taxonomy

**Change**: 260225-54vl-smart-git-pr-category-taxonomy
**Created**: 2026-02-25
**Status**: Draft

## Origin

> Backlog item [54vl]: "Make git-pr smarter. There is a category of changes that does not really require whole fab change folder to come in — these are mostly Chores or housekeeping tasks."

Initiated via `/fab-discuss` session exploring the backlog item. Conversational mode — extensive discussion covered conventional commits research, taxonomy consolidation, PR template design, type resolution chain, and broken link fixes. Multiple decisions settled through back-and-forth.

**Key decisions from conversation:**
- Consolidated conventional commits taxonomy from 10 types down to 7 (style→refactor, perf→feat/refactor, build→ci)
- Two-tier PR template: fab-linked (feat/fix/refactor) vs lightweight (docs/test/ci/chore)
- Type resolution chain: explicit argument → infer from intake → infer from diff
- Fix broken links by using GitHub blob URLs on branch instead of relative paths
- PR title always gets conventional-commits-style prefix
- Default to `chore` when inferring from diff with no clear pattern match
- No questions asked (git-pr's "no prompts" contract preserved)

## Why

`/git-pr` currently assumes every PR is backed by a fab change folder. This creates two problems:

1. **Friction for lightweight changes**: Chores, CI tweaks, doc fixes, and test additions don't warrant the full intake→spec→tasks→apply→review→hydrate pipeline. Today you either skip `/git-pr` entirely (losing its automation) or create unnecessary fab artifacts.

2. **Broken artifact links**: The current PR template uses relative paths (`fab/changes/{name}/intake.md`) for intake/spec links. GitHub resolves these against the default branch (main), where the change files don't exist. Every PR has dead links in its Context section.

3. **Missing reviewer signal**: Reviewers can't tell at a glance whether a PR is a carefully spec'd feature or a quick housekeeping fix. The PR body format is identical regardless.

Without this change, `/git-pr` remains useful only for pipeline-tracked changes, and even those have broken links.

## What Changes

### PR Category Taxonomy

Introduce 7 PR types derived from conventional commits, consolidated for practical use:

| Type | Description | Fab pipeline? |
|------|-------------|---------------|
| `feat` | New feature or capability | Yes |
| `fix` | Bug fix | Yes |
| `refactor` | Restructure without behavior change | Yes |
| `docs` | Documentation-only changes | No |
| `test` | Adding/fixing tests only | No |
| `ci` | CI/CD and build system changes | No |
| `chore` | Maintenance, cleanup, housekeeping | No |

Consolidation rationale:
- `style` → merged into `refactor` (formatting is restructuring)
- `perf` → merged into `feat` or `refactor` (perf changes either add capability or restructure internals)
- `build` → merged into `ci` (build config and CI config are the same mental space)

### Type Resolution Chain

When `/git-pr` runs, it determines the PR type via a three-step chain:

1. **Explicit argument**: `/git-pr chore` — user provides the type directly. Wins unconditionally.
2. **Infer from fab change**: No argument provided, but `changeman.sh resolve` succeeds and `intake.md` exists — pattern-match intake content:
   - Contains "fix", "bug", "broken", "regression" → `fix`
   - Contains "refactor", "restructure", "consolidate", "split", "rename" → `refactor`
   - Otherwise → `feat`
3. **Infer from diff**: No argument, no fab change — analyze changed files:
   - All changes in `.github/`, CI config files (`.yml`, `.yaml` in CI paths) → `ci`
   - All changes in `docs/`, non-code `*.md` files → `docs`
   - All changes in test files/directories → `test`
   - Otherwise → `chore`

### Two-Tier PR Template

**Tier 1 — Fab-linked** (feat/fix/refactor):

```markdown
## Summary
{1-3 sentences derived from intake's ## Why section}

## Changes
{bulleted list from intake's ## What Changes subsection headings}

## Context
| | |
|---|---|
| Type | feat |
| Change | `260225-54vl-smart-git-pr-category-taxonomy` |
| [Intake](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/intake.md) | [Spec](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/spec.md) |
```

**Tier 2 — Lightweight** (docs/test/ci/chore):

```markdown
## Summary
{auto-generated from commit messages or diff stat}

## Context
| | |
|---|---|
| Type | chore |

No design artifacts — housekeeping change.
```

### PR Title Prefix

All PR titles get a conventional-commits-style prefix: `{type}: {title}`. Examples:
- `feat: Smart change resolution and PR summary`
- `chore: Tidy gitignore entries`
- `ci: Add shellcheck to workflow`

### Fix Broken Artifact Links

Replace relative paths with GitHub blob URLs that resolve against the PR's branch:

```
# Before (broken — resolves against main)
[Intake](fab/changes/{name}/intake.md)

# After (works — resolves against feature branch)
[Intake](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/intake.md)
```

Derive `{owner}/{repo}` from `gh repo view --json nameWithOwner -q '.nameWithOwner'`. Derive `{branch}` from `git branch --show-current`.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update git-pr documentation with new type resolution, template tiers, and link fix

## Impact

- **`fab/.kit/skills/git-pr.md`** — primary change: new Step 0 (type resolution), modified Step 3c (two-tier template, blob URLs, title prefix)
- **Existing PRs** — no retroactive changes; only new PRs get the updated format
- **`/fab-ff` and `/fab-fff`** — no impact (they invoke `/git-pr` without arguments, which will infer from fab state as before)
- **Taxonomy spec** — the finalized 7-type list should be persisted in `docs/specs/` for reference

## Open Questions

None — all design decisions were settled in the preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 7 PR types: feat, fix, refactor, docs, test, ci, chore | Discussed — user explicitly consolidated from conventional commits, approved each merge | S:95 R:85 A:90 D:95 |
| 2 | Certain | Three-step type resolution: explicit → intake → diff | Discussed — user confirmed "explicit wins, then infer" | S:90 R:80 A:85 D:90 |
| 3 | Certain | Two-tier PR template (fab-linked vs lightweight) | Discussed — user approved both templates with reviewer signaling | S:90 R:80 A:85 D:90 |
| 4 | Certain | Fix links with blob URLs on branch | Discussed — user confirmed relative paths are broken, blob URLs are the fix | S:95 R:90 A:90 D:95 |
| 5 | Certain | PR title gets conventional-commits prefix | Discussed — user proposed this as the place for the type signal | S:90 R:85 A:85 D:95 |
| 6 | Certain | Default to chore when diff inference has no clear match | Discussed — user agreed lightweight changes without fab folder are overwhelmingly chores | S:85 R:85 A:80 D:85 |
| 7 | Confident | Intake pattern-matching for feat/fix/refactor inference | Reasonable heuristic; keywords like "fix", "bug" → fix; "refactor", "restructure" → refactor; else → feat. Not explicitly debated but follows from the resolution chain design | S:70 R:80 A:75 D:70 |
| 8 | Confident | Derive owner/repo from `gh repo view` | Standard gh CLI approach; assumes gh is authenticated (already a prerequisite for git-pr) | S:75 R:90 A:85 D:90 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
