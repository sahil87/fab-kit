# Brief: Extract confidence scoring into standalone script

**Change**: 260213-w8p3-extract-fab-score
**Created**: 2026-02-13
**Status**: Draft

## Origin

> Extract confidence score calculation into standalone fab-score.sh script. Currently duplicated across fab-new, fab-continue, and fab-clarify. Move to a single bash script (fab/.kit/scripts/fab-score.sh) that reads brief.md + spec.md Assumptions tables, counts grades, applies formula, writes to .status.yaml. Remove scoring from fab-new (brief stage). Only recalculate at spec stage (via fab-continue) and after fab-clarify suggest mode. Drop the redundant Lifecycle table from _context.md. See plan at /home/parallels/.claude/plans/parallel-scribbling-swing.md for full details.

## Why

Confidence score calculation is duplicated across three skills (`/fab-new`, `/fab-continue`, `/fab-clarify`) with nearly identical 3-step inline procedures (scan artifacts, count grades, apply formula, write `.status.yaml`). This creates maintenance burden and inconsistency risk. Additionally, scoring at the brief stage is premature — real decisions happen at spec stage. Centralizing to a script eliminates duplication, makes the algorithm deterministic (no LLM interpretation needed), and simplifies the trigger model.

## What Changes

- New `fab/.kit/scripts/fab-score.sh` — standalone bash script that scans `## Assumptions` tables in `brief.md` + `spec.md` only (not `tasks.md`), counts SRAD grades, applies the confidence formula, writes the `confidence` block to `.status.yaml`
- Remove confidence computation from `/fab-new` (Step 7) — template defaults (score 5.0) persist until spec stage
- Remove inline recomputation from `/fab-continue` (Step 3b) — replace with `fab-score.sh` invocation at spec stage only (not tasks or execution stages)
- Remove inline recomputation from `/fab-clarify` (Step 7) — replace with `fab-score.sh` invocation in suggest mode (auto mode unchanged)
- Drop the Lifecycle table from `_context.md` Confidence Scoring section — the "Recomputes confidence?" row in the Skill-Specific Autonomy table already covers this
- Update `srad.md` Confidence Lifecycle section with a simplified 3-row table
- Update `planning-skills.md` and `change-lifecycle.md` references

## Affected Docs

- `fab-workflow/planning-skills`: (modify) Remove confidence scoring paragraph from `/fab-new`, update `/fab-continue` forward flow, update `/fab-fff` recomputation references
- `fab-workflow/change-lifecycle`: (modify) Update confidence field description in `.status.yaml` section

## Impact

- **Skill files**: `fab-new.md`, `fab-continue.md`, `fab-clarify.md`, `_context.md` — remove inline scoring logic, add script invocations
- **Design docs**: `srad.md` — simplify lifecycle table and autonomy table
- **Scripts**: New `fab-score.sh` alongside existing `fab-preflight.sh`, `fab-status.sh`
- **No behavioral change for `/fab-ff` or `/fab-fff`** — they already don't recompute; gate check in `/fab-fff` is unaffected (reads stored score)

## Design Rationale

**Why only `brief.md` + `spec.md`?** Analysis of 48 archived changes shows `## Assumptions` tables appear in brief (46%), spec (81%), and tasks (6%). The 3 tasks.md instances contain only Confident-grade assumptions about task grouping/ordering — not design decisions. `tasks.md` is a derivative artifact that breaks down the spec; it almost never introduces new decision points worth scoring. Scanning brief + spec captures effectively all meaningful signal.

## Open Questions

None — scope fully resolved during planning conversation.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Script interface: `$1` = change dir path, stdout = YAML, exit codes 0/1 | Follows existing `fab-preflight.sh` and `fab-status.sh` conventions |
| 2 | Confident | Carry-forward logic for implicit Certain counts preserved | User explicitly chose to keep `certain` in the schema; existing carry-forward mechanism needed to count decisions not listed in Assumptions tables |
| 3 | Confident | Auto mode in `/fab-clarify` does not invoke `fab-score.sh` | Auto mode is called internally by `/fab-ff` which doesn't recompute; keeping current non-recomputing behavior is consistent |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.
