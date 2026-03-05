# Intake: Richer Git PR Output

**Change**: 260227-8q33-richer-git-pr-output
**Created**: 2026-02-27
**Status**: Draft

## Origin

> Make the git-pr output show more fields from status.yaml - richer output for the code reviewer. Also fix the table format — intake and spec links are side by side in the same row instead of stacked vertically as separate Field/Detail rows.

Initiated from backlog item `[8q33]`. User observed that the Tier 1 PR body template currently shows only three fields (Type, Change, and a combined Intake/Spec link row) in the Context table. A code reviewer seeing the PR has no visibility into the change's confidence score, checklist completion, stage metrics, or linked issues — all of which live in `.status.yaml` and would help a reviewer gauge the depth of thought behind the PR.

## Why

1. **Reviewer blind spot**: The current Tier 1 PR body shows `Type`, `Change`, and artifact links. A reviewer has no way to tell whether the change went through full pipeline stages, what the confidence score was, or whether the checklist passed — without manually navigating to the change folder.

2. **Table format bug**: The current template puts `[Intake]` and `[Spec]` as two cells in the same row (`| [Intake](url) | [Spec](url) |`), which breaks the `Field | Detail` column pattern used by the other rows. "Intake" isn't a "Field" label and "Spec" isn't its "Detail" — they're two separate artifact links that should each be their own row.

3. **If we don't fix it**: Reviewers continue making approval decisions with incomplete context. The malformed table row is a minor but visible quality issue in every Tier 1 PR.

## What Changes

### Fix the Context table structure

The current Tier 1 template row:
```
| [Intake]({intake_url}) | [Spec]({spec_url}) |
```

Should become two separate rows following the `Field | Detail` pattern:
```
| Intake | [{change_name}/intake.md]({intake_url}) |
| Spec | [{change_name}/spec.md]({spec_url}) |
```

If `spec.md` does not exist, omit the Spec row entirely (rather than emitting an empty cell).

### Add richer fields from `.status.yaml` to Tier 1 Context table

Read the following fields from `.status.yaml` (via `yq` or by reading the file directly) and add rows to the Context table:

1. **Confidence** — `confidence.score` (e.g., `3.5 / 5.0`). Gives reviewer a quick signal of how well-resolved the decisions are.

2. **Pipeline** — derive from `progress` map. Show which stages completed (e.g., `intake → spec → tasks → apply → review → hydrate`). Use a compact format: list stage names that have `done` status, joined with ` → `.

### Updated Tier 1 template

The full Context table becomes:

```
## Context
| Field | Detail |
|---|---|
| Type | {type} |
| Change | `{change_name}` |
| Confidence | {score} / 5.0 |
| Pipeline | {completed_stages joined with →} |
| Intake | [{change_name}/intake.md]({intake_url}) |
| Spec | [{change_name}/spec.md]({spec_url}) |
```

Rows with missing data are omitted (Spec if `spec.md` doesn't exist).

### No changes to Tier 2 template

The lightweight template remains as-is — it has no `.status.yaml` to read.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update git-pr documentation to reflect richer PR body template

## Impact

- **`fab/.kit/skills/git-pr.md`** — primary change: update Tier 1 PR body template in Step 3c
- No script changes — `.status.yaml` is already available via the resolved change directory
- No template file changes — the PR body is generated inline in the skill

## Open Questions

- None — scope is clear.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only Tier 1 template changes | Tier 2 has no `.status.yaml` to read — no changes needed | S:90 R:95 A:95 D:95 |
| 2 | Certain | Read fields directly from `.status.yaml` file | The file is already available after `changeman.sh resolve` — no new script needed | S:85 R:90 A:90 D:90 |
| 3 | Confident | Omit rows with empty data rather than showing empty cells | Cleaner output; follows the existing pattern of conditional spec row | S:70 R:90 A:80 D:75 |
| 4 | Confident | Show confidence as `{score} / 5.0` format | Consistent with how confidence is displayed elsewhere in fab (e.g., fab-status) | S:75 R:90 A:85 D:80 |
| 5 | Confident | Use `→` separator for pipeline stages | Compact, readable, shows progression | S:65 R:95 A:70 D:65 |
| 6 | Certain | Fix Intake/Spec to be separate rows with Field|Detail pattern | User explicitly requested this fix; current format is structurally wrong | S:95 R:95 A:95 D:95 |
| 7 | Confident | Use `[{change_name}/file.md](url)` link text for artifacts | More informative than bare `[Intake]` — reviewer can see the path | S:60 R:95 A:75 D:70 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
