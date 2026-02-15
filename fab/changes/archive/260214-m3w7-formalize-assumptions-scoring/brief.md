# Brief: Formalize Assumptions Tables & Fix Scoring Pipeline

**Change**: 260214-m3w7-formalize-assumptions-scoring
**Created**: 2026-02-14
**Status**: Draft

## Origin

> Formalize SRAD assumptions tables across the pipeline. All four grades (Certain, Confident, Tentative, Unresolved) in every Assumptions table with required Scores column. calc-score.sh reads only spec.md (not brief.md) to eliminate double-counting. Fix AWK column index bug (cols[4] should be cols[6] for Scores). Remove implicit Certain carry-forward. Parse Unresolved from table instead of hardcoding to 0. Formalize table structure in both brief.md and spec.md templates. Brief assumptions serve as state transfer — spec uses them as starting point (confirm/upgrade/override). Unresolved Rationale must include status context (e.g. "Asked — user undecided", "Deferred — exceeded question budget"). Brief must be self-contained and maximally informative as the sole bridge between fab-new and fab-continue (independent agent contexts).

**Design context**: This change emerged from a detailed design discussion analyzing: (1) whether SRAD analysis is needed at the brief stage if scoring only happens at spec, (2) how assumptions flow between pipeline stages, (3) the realization that pipeline stages may execute in separate agent contexts with no shared memory — making artifacts the sole continuity mechanism.

## Why

The Assumptions table is the **state transfer mechanism** between pipeline stages that may be executed by different AI agents with zero shared context. Currently, the system has several problems:

1. **Incomplete state transfer**: Only Confident and Tentative grades are recorded. Certain decisions (what the agent considered deterministic) and Unresolved decisions (what was asked or deferred) are invisible to downstream agents. A fresh spec-stage agent has no idea what the brief-stage agent already resolved or what's still open.

2. **Double-counting**: `calc-score.sh` reads Assumptions tables from both `brief.md` and `spec.md`. Since the spec builds its own independent assumptions list (not inherited from brief), the same decision space gets counted twice, inflating grade counts and distorting the confidence score.

3. **Broken dimension parsing**: The AWK parser in `calc-score.sh` extracts `cols[4]` for the Scores column, but in the 5-column table (`| # | Grade | Decision | Rationale | Scores |`), `cols[4]` is actually the Decision column. The Scores column is `cols[6]`. This means fuzzy dimension score aggregation has been silently broken — the `grep -oP 'S:\K[0-9]+'` pattern fails against Decision text, and all rows fall back to non-fuzzy mode.

4. **Hardcoded Unresolved count**: `calc-score.sh` hardcodes `unresolved=0` because the current rules exclude Unresolved grades from tables. This means the formula's `if unresolved > 0: score = 0.0` branch is dead code — it can never trigger from parsed data.

5. **Optional Scores column**: The Scores column (`S:nn R:nn A:nn D:nn`) is optional, requiring `has_scores` detection logic in the AWK parser. Making it required simplifies parsing and ensures every assumption has a transparent rationale for its grade.

6. **No formalized table structure in templates**: Neither `brief.md` nor `spec.md` templates include the `## Assumptions` table structure. Agents must infer the format from prose in `_context.md`, leading to inconsistent output.

## What Changes

### Rules changes (`_context.md`)

- **All four SRAD grades** (Certain, Confident, Tentative, Unresolved) SHALL be recorded in every Assumptions table — not just Confident and Tentative
- **Scores column required**: The `S:nn R:nn A:nn D:nn` column is mandatory for every row, not optional
- **Unresolved status context**: Unresolved rows MUST include status context in the Rationale column explaining what happened (e.g., "Asked — user undecided", "Deferred — exceeded question budget", "Asked — needs team input")
- **Confidence Grades table**: Update Output Visibility for Certain ("Noted in Assumptions summary" instead of "None — not worth mentioning") and Unresolved ("Asked as question AND noted in Assumptions summary")
- **Summary line format**: Change from `{N} assumptions ({C} confident, {T} tentative)` to `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved)`

### Template changes

- **`brief.md` template**: Add formalized `## Assumptions` section with 5-column header (`| # | Grade | Decision | Rationale | Scores |`) and HTML comment explaining its purpose as state transfer to the spec-stage agent
- **`spec.md` template**: Add formalized `## Assumptions` section with 5-column header and HTML comment explaining it is the sole scoring source for `calc-score.sh`

### Script changes (`calc-score.sh`)

- **Remove `brief.md` from parsing**: Delete `brief_file` variable and `parse_assumptions "$brief_file"` call. Only parse `spec.md`
- **Remove implicit Certain carry-forward**: Delete the entire block (lines 179-194) that reads previous Certain count from `.status.yaml` and computes implicit carry-forward. With spec-only parsing, `total_certain = table_certain` directly
- **Fix AWK column index**: Change `scores = cols[4]` to `scores = cols[6]` to correctly extract the Scores column from the 5-column table
- **Remove `has_scores` detection**: Always parse Scores from `cols[6]` since the column is now required
- **Parse Unresolved grade**: Add `unresolved)` case to the grade counting switch. Remove hardcoded `unresolved=0`
- The scoring formula itself (`5.0 - 0.3*confident - 1.0*tentative`, with `unresolved > 0 → 0.0`) does NOT change — it just gets real Unresolved data instead of hardcoded 0

### Skill changes

- **`fab-new.md`**: Update Step 5.8 to specify all four grades with required Scores column. Add guidance that the brief is the sole context for downstream stages — every section must be substantive, not placeholder text
- **`_generation.md`**: Update Spec Generation Procedure Step 6 — spec reads `brief.md`'s Assumptions table as starting point, confirms/upgrades/overrides each assumption, adds new assumptions discovered during spec generation, includes all four grades
- **`fab-ff.md`**: Update Step 1 — all four SRAD grades tracked in cumulative Assumptions summary

### Memory updates

- **`planning-skills.md`**: Update calc-score invocation note ("spec Assumptions table" instead of "brief + spec"), update SRAD/Assumptions references for all four grades, add changelog entry
- **`change-lifecycle.md`**: Update confidence field description (computed from spec.md only), add changelog entry

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update calc-score invocation, SRAD grade rules, assumptions summary format
- `fab-workflow/change-lifecycle`: (modify) Update confidence field description for spec-only scoring

## Impact

### Files modified

| File | Nature of change |
|------|-----------------|
| `fab/.kit/skills/_context.md` | Rules: all four grades, required Scores column, Unresolved status context, updated Confidence Grades table, updated summary format |
| `fab/.kit/templates/brief.md` | Add formalized `## Assumptions` table structure with HTML comment |
| `fab/.kit/templates/spec.md` | Add formalized `## Assumptions` table structure with HTML comment |
| `fab/.kit/scripts/lib/calc-score.sh` | Remove brief.md parsing, fix AWK cols[6], remove has_scores, parse Unresolved, remove carry-forward |
| `fab/.kit/skills/fab-new.md` | Step 5.8: all four grades, self-contained brief guidance |
| `fab/.kit/skills/_generation.md` | Spec Procedure Step 6: brief assumptions as starting point |
| `fab/.kit/skills/fab-ff.md` | Step 1: all four grades in cumulative summary |
| `docs/memory/fab-workflow/planning-skills.md` | Update scoring, assumptions, changelog |
| `docs/memory/fab-workflow/change-lifecycle.md` | Update confidence field, changelog |

### Dependencies

- `260212-f9m3-enhance-srad-fuzzy` introduced fuzzy 0-100 scoring and the optional Scores column. This change makes the Scores column mandatory and fixes the AWK bug that prevented fuzzy scores from being parsed. No conflict — the changes are complementary.

### Behavioral impact

- **Existing briefs**: Won't have all four grades or required Scores column. `calc-score.sh` no longer reads brief.md, so this is a non-issue for scoring. Existing briefs remain valid but won't conform to the new template.
- **Existing specs**: May have Assumptions tables with only Confident/Tentative grades and optional Scores. `calc-score.sh` should handle missing Scores gracefully (skip dimension aggregation for that row) during transition. However, new specs will always include all four grades with Scores.
- **Score values may change**: Removing brief.md from parsing may lower grade counts (especially Certain). Since the formula only penalizes Confident (0.3) and Tentative (1.0), and Certain has zero penalty, the practical impact on scores is minimal. The main change: Unresolved in spec will now correctly trigger `score = 0.0` instead of being invisible.

## Open Questions

None — all decision points resolved through the design discussion. The user provided explicit direction on every aspect: which grades to include, whether Scores is optional, what calc-score should read, how Unresolved should be annotated, and the rationale for brief-as-state-transfer.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | calc-score.sh reads only spec.md, not brief.md | User explicitly directed this to eliminate double-counting; spec is the authoritative decision record | S:95 R:80 A:95 D:95 |
| 2 | Certain | All four SRAD grades in every Assumptions table | User explicitly stated "even certain and unresolved assumptions, not just confident and tentative"; needed for state transfer between isolated agent contexts | S:95 R:85 A:95 D:95 |
| 3 | Certain | Scores column always required (not optional) | User explicitly stated "The score column should no longer be optional"; simplifies parsing, ensures transparent grade rationale | S:90 R:75 A:90 D:90 |
| 4 | Certain | Unresolved Rationale must include status context | User chose "Add a Status note" when asked; context like "Asked — user undecided" or "Deferred — exceeded budget" gives downstream agents actionable information | S:90 R:85 A:90 D:85 |
| 5 | Certain | Fix AWK column index from cols[4] to cols[6] | Confirmed bug: 5-column table split by `\|` produces cols[1]="", cols[2]="#", cols[3]="Grade", cols[4]="Decision", cols[5]="Rationale", cols[6]="Scores" | S:95 R:90 A:95 D:95 |
| 6 | Certain | Formalize table structure in both templates | User explicitly requested; machine parsing requires predictable structure; templates are the authoritative shape definition | S:90 R:85 A:85 D:80 |
| 7 | Certain | Brief must be self-contained as sole bridge between agents | User emphasized "these documents are what give us the continuity" — no shared context between pipeline stages | S:90 R:85 A:90 D:85 |
| 8 | Confident | Remove implicit Certain carry-forward from calc-score.sh | Follows logically from spec-only parsing — carry-forward was designed for brief+spec aggregation. With one file, total_certain = table_certain directly | S:85 R:70 A:85 D:80 |
| 9 | Confident | Spec generation reads brief assumptions as starting point (confirm/upgrade/override) | User explicitly stated this approach; brief assumptions were previously not formally inherited by spec | S:90 R:80 A:85 D:80 |
| 10 | Confident | Scoring formula unchanged (5.0 - 0.3*confident - 1.0*tentative; unresolved > 0 → 0.0) | User's changes are about what data feeds the formula, not the formula itself; existing penalties are validated by 260212-f9m3-enhance-srad-fuzzy research | S:70 R:75 A:80 D:75 |

10 assumptions (7 certain, 3 confident, 0 tentative, 0 unresolved).
