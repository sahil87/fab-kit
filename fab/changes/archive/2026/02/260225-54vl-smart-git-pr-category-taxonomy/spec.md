# Spec: Smart git-pr Category Taxonomy

**Change**: 260225-54vl-smart-git-pr-category-taxonomy
**Created**: 2026-02-25
**Affected memory**: `fab-workflow/execution-skills.md`

## Executive Summary

Enhance `/git-pr` to support lightweight changes (chores, CI, test, docs) that don't require a full fab change folder. Introduce a 7-type PR category taxonomy (feat, fix, refactor, docs, test, ci, chore) with conventional-commits-style prefixes. Implement a two-tier PR template: fab-linked changes show intake/spec links with proper blob URLs; lightweight changes explicitly signal "no design artifacts." Type resolution uses a three-step chain: explicit argument → infer from fab intake → infer from diff. Also fix broken relative links in current PR template.

---

## Execution Skills: git-pr Enhancements

### Requirement: PR Category Taxonomy

The PR system SHALL support 7 distinct change categories derived from the conventional commits specification, consolidated to practical use within fab-kit:

| Category | Description | Fab Pipeline? | PR Template Tier |
|----------|-------------|---------------|------------------|
| `feat` | New feature or capability | Yes | 1 (fab-linked) |
| `fix` | Bug fix | Yes | 1 (fab-linked) |
| `refactor` | Restructure without behavior change | Yes | 1 (fab-linked) |
| `docs` | Documentation-only changes | No | 2 (lightweight) |
| `test` | Adding/fixing tests only | No | 2 (lightweight) |
| `ci` | CI/CD and build system changes | No | 2 (lightweight) |
| `chore` | Maintenance, cleanup, housekeeping | No | 2 (lightweight) |

Consolidation rationale: `style` is merged into `refactor` (formatting is code restructuring), `perf` is merged into `feat` or `refactor` (performance changes are either new capability or internal restructuring), `build` is merged into `ci` (build config and CI config are the same mental space).

#### Scenario: User ships a feature
- **GIVEN** a change with a fab change folder containing `intake.md` and `spec.md`
- **WHEN** the user runs `/git-pr` (no argument)
- **THEN** the PR type is inferred as `feat`, `fix`, or `refactor` based on intake content
- **AND** the PR title gets the prefix `feat: `, `fix: `, or `refactor: `
- **AND** the PR body includes links to intake and spec using proper blob URLs

#### Scenario: User ships a chore without a fab folder
- **GIVEN** a change with no fab folder (simple commit in the working directory)
- **WHEN** the user runs `/git-pr` (no argument)
- **THEN** the PR type is inferred as `chore` (default for lightweight changes)
- **AND** the PR title gets the prefix `chore: `
- **AND** the PR body explicitly states "No design artifacts — housekeeping change"

#### Scenario: User explicitly specifies the PR type
- **GIVEN** any state (fab folder or not)
- **WHEN** the user runs `/git-pr docs`
- **THEN** the PR type is set to `docs` (explicit argument wins)
- **AND** the PR title gets the prefix `docs: `
- **AND** the PR body uses the lightweight template

### Requirement: Type Resolution Chain

`/git-pr` SHALL resolve the PR type using a three-step chain:

1. **Explicit argument** (if provided): `/git-pr {type}` — the user-provided type wins unconditionally
2. **Infer from fab change intake** (if no argument, fab change exists):
   - Run `changeman.sh resolve` to find the active change
   - Read `fab/changes/{name}/intake.md` if it exists
   - Pattern-match the intake content:
     - If it contains "fix", "bug", "broken", or "regression" → type = `fix`
     - If it contains "refactor", "restructure", "consolidate", "split", or "rename" → type = `refactor`
     - Otherwise → type = `feat`
3. **Infer from diff** (if no argument, no fab change):
   - Analyze changed files in the current working directory
   - If all changes are in `.github/` or CI config files (`.yml`, `.yaml` in CI paths) → type = `ci`
   - If all changes are in `docs/` or non-code `*.md` files → type = `docs`
   - If all changes are in test files/directories → type = `test`
   - Otherwise → type = `chore` (default)

#### Scenario: Explicit type overrides everything
- **GIVEN** a change with a fab folder AND `/git-pr chore` is called
- **WHEN** the explicit argument is evaluated
- **THEN** the type is set to `chore` (not `feat`/`fix`/`refactor` from intake)

#### Scenario: Inference from intake
- **GIVEN** a fab change with intake.md containing "Consolidate git and fab commands into a single script"
- **WHEN** `/git-pr` runs with no argument
- **THEN** the type is inferred as `refactor`

#### Scenario: Inference from diff with no fab folder
- **GIVEN** uncommitted changes only in `.github/workflows/*.yml`
- **WHEN** `/git-pr` runs with no argument and no active fab change
- **THEN** the type is inferred as `ci`

### Requirement: Two-Tier PR Template

`/git-pr` SHALL generate PR bodies using one of two templates based on the PR type:

**Tier 1 — Fab-Linked** (types: feat, fix, refactor):

```markdown
## Summary
{1-3 sentences from intake's ## Why section}

## Changes
{bulleted list of subsection headings from intake's ## What Changes section}

## Context
| | |
|---|---|
| Type | {type} |
| Change | {change-name} |
| [Intake](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/intake.md) | [Spec](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/spec.md) |
```

**Tier 2 — Lightweight** (types: docs, test, ci, chore):

```markdown
## Summary
{auto-generated from commit message or diff --stat}

## Context
| | |
|---|---|
| Type | {type} |

No design artifacts — housekeeping change.
```

The summary in Tier 2 is generated from:
- The commit message (if available) — first 1-3 sentences
- Otherwise, a brief description derived from `git diff --stat` output

#### Scenario: PR for a feature change
- **GIVEN** intake.md with "## Why" containing "Developers need a way to categorize PRs to help reviewers understand the scope."
- **WHEN** the spec.md is present and type is `feat`
- **THEN** the PR body uses Tier 1 template
- **AND** the Context section includes working links to intake and spec

#### Scenario: PR for a CI fix
- **GIVEN** changes only in `.github/workflows/` and type is `ci`
- **WHEN** the PR is created
- **THEN** the PR body uses Tier 2 template
- **AND** the Context section states "No design artifacts — housekeeping change"

### Requirement: PR Title Prefix

All PR titles SHALL have a conventional-commits-style prefix: `{type}: {title}`.

The title itself is derived from:
- **Fab-linked changes** (feat/fix/refactor): The first `# ` heading from `intake.md` (stripping "Intake: " prefix if present)
- **Lightweight changes** (docs/test/ci/chore): The commit message subject line, or a brief auto-generated title from `gh pr create --fill`

#### Scenario: PR title for a feature
- **GIVEN** intake.md with `# Intake: Smart git-pr Category Taxonomy`
- **WHEN** the PR is created with type `feat`
- **THEN** the PR title is `feat: Smart git-pr Category Taxonomy`

#### Scenario: PR title for a chore
- **GIVEN** a commit message "Fix typo in README"
- **WHEN** `/git-pr chore` is called
- **THEN** the PR title is `chore: Fix typo in README`

### Requirement: Fix Broken Artifact Links

Replace relative paths in PR body links with GitHub blob URLs that resolve against the PR's feature branch, not the default branch.

**Before** (broken):
```
[Intake](fab/changes/{name}/intake.md)
[Spec](fab/changes/{name}/spec.md)
```

**After** (working):
```
[Intake](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/intake.md)
[Spec](https://github.com/{owner}/{repo}/blob/{branch}/fab/changes/{name}/spec.md)
```

Derivation:
- `{owner}/{repo}` from: `gh repo view --json nameWithOwner -q '.nameWithOwner'`
- `{branch}` from: `git branch --show-current`
- `{name}` from: `changeman.sh resolve` (or via `fab/current`)

#### Scenario: PR from feature branch
- **GIVEN** current branch is `260225-54vl-smart-git-pr-category-taxonomy`
- **WHEN** the PR is created
- **THEN** the Context section links use the full GitHub URL with the branch name
- **AND** clicking the link navigates to the file on the feature branch (not main, where it doesn't exist)

---

## Design Decisions

1. **7-Type Taxonomy Over 10 (Conventional Commits)**
   - *Why*: Consolidation reduces cognitive load. `style` (formatting) is a refactor concern, `perf` is a feat/refactor concern, `build` overlaps with `ci`. The 7 categories cleanly separate pipeline-heavy (feat/fix/refactor) from lightweight (docs/test/ci/chore).
   - *Rejected*: Keeping all 10 types — adds complexity without proportional benefit. For sddr and most fab-kit projects, 7 is sufficient and clearer.

2. **Three-Step Type Resolution Chain**
   - *Why*: Preserves `/git-pr`'s "no questions" contract while supporting both fab-tracked and lightweight changes. Explicit argument allows override; fab inference leverages existing intake; diff inference provides a safe fallback for ad-hoc changes.
   - *Rejected*: Fixed tier detection (e.g., always lightweight if no fab folder) — too rigid. User might want `/git-pr feat "my-feature"` to create a PR without a full fab folder.

3. **Two-Tier Template**
   - *Why*: Signals to reviewers what to expect. Fab-linked PRs (feat/fix/refactor) have design artifacts and deserve extra scrutiny; lightweight PRs don't. Explicit "no artifacts" message prevents reviewer confusion about missing links.
   - *Rejected*: Single template with optional links — links are broken (current state), and reviewers wouldn't know if they're expected or missing.

4. **Blob URLs Over Relative Paths**
   - *Why*: Relative paths resolve against the default branch (main), where change files don't exist. Blob URLs resolve against the feature branch. Durable and clear.
   - *Rejected*: Commit SHAs in URLs — more stable against force-pushes, but less readable and harder to construct. Blob URLs on branch are good enough and conventional.

5. **Lightweight Default is `chore`**
   - *Why*: When diff inference can't clearly identify the type (e.g., multiple file types changed), defaulting to `chore` is safe and honest. Chores are the majority of non-fab changes.
   - *Rejected*: Prompting the user or failing — breaks `/git-pr`'s "no questions" contract. Defaulting to `feat` would be misleading.

---

## Non-Goals

- Retroactively fixing links in existing PRs — only new PRs get the updated format
- Automatic PR type detection from branch names — too error-prone; we infer from content instead
- Integration with git commit message parsing beyond simple keyword matching — regex matching on intake content is sufficient for the inference heuristic
- Auto-commit of `.status.yaml` changes when type is inferred differently than assumed in intake — the PR ships regardless; type is just a signal

---

## Implementation Notes

- All changes are in `fab/.kit/skills/git-pr.md` (primary skill file)
- The three-step chain is evaluated in the order: explicit → fab-inferred → diff-inferred
- Pattern matching for fab inference (step 2) uses case-insensitive keyword search on intake.md content
- Diff inference (step 3) checks file paths; use `git diff --name-only` to get the list
- Blob URL construction uses standard GitHub URL format: `https://github.com/{owner}/{repo}/blob/{branch}/{path}`
- The PR title prefix is always present; no conditional logic for whether to include it

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | 7 PR types (feat, fix, refactor, docs, test, ci, chore) | Consolidation from conventional commits. All merges discussed and approved: style→refactor, perf→feat/refactor, build→ci. | S:95 R:85 A:90 D:95 |
| 2 | Certain | Three-step type resolution: explicit → intake → diff | Discussed. User confirmed "explicit wins, then infer." Chain preserves /git-pr's autonomy contract. | S:90 R:80 A:85 D:90 |
| 3 | Certain | Two-tier PR template (fab-linked vs lightweight) | Discussed. User approved both templates for reviewer signaling. Tier 1 shows intake/spec; Tier 2 says "no artifacts". | S:90 R:80 A:85 D:90 |
| 4 | Certain | Fix broken links with blob URLs on branch | Discussed. User confirmed relative paths are broken; blob URLs are the fix. Resolves against feature branch, not main. | S:95 R:90 A:90 D:95 |
| 5 | Certain | PR title gets conventional-commits prefix | Discussed. User proposed this as the place for the type signal. All titles: {type}: {title}. | S:90 R:85 A:85 D:95 |
| 6 | Certain | Default to chore when diff inference has no clear match | Discussed. User agreed lightweight changes without fab folder are overwhelmingly chores. Safe default. | S:85 R:85 A:80 D:85 |
| 7 | Confident | Pattern matching for feat/fix/refactor inference | Reasonable heuristic: "fix"/"bug"→fix, "refactor"/"restructure"→refactor, else→feat. Not explicitly debated but follows naturally from the design. Case-insensitive keyword search. | S:70 R:80 A:75 D:70 |
| 8 | Confident | Derive owner/repo via `gh repo view`, branch via `git branch --show-current` | Standard gh CLI approaches. Assumes gh is authenticated (prerequisite for /git-pr). Both commands reliable and idempotent. | S:75 R:90 A:85 D:90 |
| 9 | Confident | Blob URL construction: `https://github.com/{owner}/{repo}/blob/{branch}/{path}` | Standard GitHub URL format. Works for all files. Branch name may contain special characters but GitHub URL-encodes them automatically. | S:80 R:85 A:85 D:90 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
