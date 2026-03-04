# Intake: Unified PR Template

**Change**: 260305-b0xs-unified-pr-template
**Created**: 2026-03-05
**Status**: Draft

## Origin

> [b0xs] 2026-03-04: Single type of template in git-pr instead of switching between two formats

Discussion session revealed the core problem: a `test`-type change that went through the full fab pipeline (intake → spec → tasks → apply → review → hydrate) still got the Tier 2 "lightweight" template with "No design artifacts — housekeeping change." because the PR type — not artifact availability — gates which template is used.

User decided on a single unified template that populates fab-linked fields when artifacts exist and shows dashes when they don't. Additionally, the template should be enriched with new metrics and restructured for scannability.

## Why

The current two-tier template system in `/git-pr` gates template richness on PR type (`feat`/`fix`/`refactor` → rich, everything else → bare). This means any change of type `docs`, `test`, `ci`, or `chore` that goes through the full fab pipeline loses all its design context in the PR — confidence scores, checklist completion, pipeline progress, and artifact links are all dropped. The PR reviewer gets no signal that this was a well-specified, well-reviewed change.

If not fixed, every non-feat/fix/refactor change continues to produce PRs that look unplanned regardless of actual pipeline depth, reducing reviewer confidence and hiding quality signals.

## What Changes

### Single Template Replaces Two Tiers

Remove the Tier 1 / Tier 2 branching logic in Step 3c of `git-pr.md`. Replace with a single template where fab-linked fields are conditionally populated based on artifact availability (does `changeman.sh resolve` succeed? does `intake.md` exist?) — not on the resolved PR type.

### Horizontal Stats Table

Replace the current vertical `| Field | Detail |` Context table with a horizontal stats row:

```markdown
## Stats
| Type | Confidence | Checklist | Tasks | Review |
|------|-----------|-----------|-------|--------|
| feat | 3.5 / 5.0 | 21/21 ✓ | 13/13 | Pass (2 iterations) |
```

Column population rules:
- **Type**: Always populated (from resolved type)
- **Confidence**: `confidence.score` from `.status.yaml`, formatted as `{score} / 5.0`. Show `—` if no fab change.
- **Checklist**: `checklist.completed`/`checklist.total` from `.status.yaml`. Append `✓` when `completed == total && total > 0`. Show `—` if not available.
- **Tasks**: Parse `tasks.md` for checkbox counts (`- [x]` vs `- [ ]`), formatted as `{done}/{total}`. Show `—` if `tasks.md` doesn't exist.
- **Review**: From `.status.yaml` `progress.review` state and `stage_metrics.review.iterations`. `Pass ({N} iterations)` if review is `done`, `Fail ({N} iterations)` if review is `failed`, `—` if review not yet reached. If `iterations` is not populated, omit the parenthetical.

### Pipeline Line with Artifact Links

Below the Stats table, show a pipeline progress line. Stages with `done` status (from `.status.yaml` `progress` map) are listed in fixed order, joined with ` → `.

The words "intake" and "spec" in this line are hyperlinks to the GitHub blob URLs (same URL construction as current Tier 1). Other stage names are plain text.

```markdown
[intake](https://github.com/{owner_repo}/blob/{branch}/fab/changes/{name}/intake.md) → [spec](https://github.com/{owner_repo}/blob/{branch}/fab/changes/{name}/spec.md) → tasks → apply → review → hydrate
```

- If `spec.md` doesn't exist, "spec" is plain text (not a link)
- If `intake.md` doesn't exist, "intake" is plain text (not a link)
- If no fab change exists, the pipeline line is omitted entirely

### PR Title Derivation Simplified

The title derivation in Step 3c step 2 ("Derive PR title") currently branches on type. Simplify to: use intake `# ` heading when intake exists, commit subject otherwise — regardless of type.

### PR Type Reference Table Cleanup

Remove the "Fab Pipeline?" and "Template Tier" columns from the PR Type Reference table at the bottom of `git-pr.md`. These columns encode the old two-tier assumption and become meaningless with the unified template.

### Graceful Degradation

When no fab artifacts exist (no active change, or `changeman.sh resolve` fails):
- Summary: auto-generated from commits/diff
- Changes section: omitted
- Stats table: only Type populated, all other columns show `—`
- Pipeline line: omitted
- No "housekeeping change" footer

### Memory Update

Update `docs/memory/fab-workflow/execution-skills.md` — the "Two-Tier PR Templates with Type Resolution" decision entry (line ~241) needs to be revised to reflect the unified template design. The type resolution chain itself is unchanged; only the template tier system is removed.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update the "Two-Tier PR Templates" decision to reflect unified template. Update the changelog entry for this change.

## Impact

- `fab/.kit/skills/git-pr.md` — primary change target (Step 3c template logic, Step 3c step 2 title derivation, PR Type Reference table)
- `.claude/skills/git-pr.md` — deployed copy (synced from kit source)

## Open Questions

- None — design was fully resolved in discussion session.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Single template with conditional field population | Discussed — user explicitly chose unified over two-tier | S:95 R:80 A:90 D:95 |
| 2 | Certain | Horizontal stats table format with Type/Confidence/Checklist/Tasks/Review columns | Discussed — user approved mockup | S:95 R:85 A:85 D:95 |
| 3 | Certain | Pipeline line below stats table with intake/spec as links | Discussed — user refined from separate links line to inline pipeline links | S:95 R:90 A:90 D:95 |
| 4 | Certain | Review column shows Pass/Fail with iteration count | Discussed — user approved "Review Verdict" addition | S:90 R:85 A:85 D:90 |
| 5 | Certain | Tasks column parsed from tasks.md checkboxes | Discussed — user approved | S:90 R:85 A:80 D:90 |
| 6 | Confident | Show `—` for unavailable fields rather than omitting columns | Strong convention from mockup; keeps table shape consistent | S:75 R:90 A:80 D:75 |
| 7 | Certain | Remove "Fab Pipeline?" and "Template Tier" columns from PR Type Reference table | Discussed — user agreed these become meaningless | S:90 R:90 A:90 D:90 |
| 8 | Certain | PR title uses intake heading when available, commit subject otherwise, regardless of type | Discussed — user approved simplification | S:90 R:85 A:85 D:90 |

8 assumptions (7 certain, 1 confident, 0 tentative, 0 unresolved).
